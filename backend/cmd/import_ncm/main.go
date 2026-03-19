package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
)

type ncmRow struct {
	Code        string
	EX          string
	Description string
	IPIRate     *float64
	Level       int
}

type columnInfo struct {
	Name       string
	DataType   string
	MaxLength  int
	IsNullable bool
}

func main() {
	var (
		filePath = flag.String("file", "data/ncm/ncm2026.csv", "caminho do arquivo CSV")
		dbURL    = flag.String("database", defaultDatabaseURL(), "URL de conexão com o PostgreSQL")
		truncate = flag.Bool("truncate", false, "apaga os registros atuais antes de importar")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	if *dbURL == "" {
		log.Fatal("database URL não informada; use --database ou configure DATABASE_URL")
	}

	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("erro ao validar conexão com banco: %v", err)
	}

	rows, err := readNCMCSV(*filePath)
	if err != nil {
		log.Fatalf("erro ao ler CSV: %v", err)
	}

	if len(rows) == 0 {
		log.Fatal("nenhum registro válido encontrado no CSV")
	}

	rows = dedupeRows(rows)
	log.Printf("registros após deduplicação: %d", len(rows))

	columns, err := loadTableColumns(ctx, pool, "ncm_catalog")
	if err != nil {
		log.Fatalf("erro ao ler colunas de ncm_catalog: %v", err)
	}

	log.Printf("colunas detectadas em ncm_catalog: %s", strings.Join(sortedColumnNames(columns), ", "))

	required := []string{"code", "description"}
	for _, col := range required {
		if _, ok := columns[col]; !ok {
			log.Fatalf("a tabela ncm_catalog não possui a coluna obrigatória %q", col)
		}
	}

	if err := importRows(ctx, pool, rows, columns, *truncate); err != nil {
		log.Fatalf("erro ao importar NCM: %v", err)
	}

	log.Printf("importação concluída com sucesso: %d registros processados", len(rows))
}

func readNCMCSV(filePath string) ([]ncmRow, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("abrir arquivo %q: %w", filePath, err)
	}

	content = normalizeEncoding(content)

	reader := csv.NewReader(bytes.NewReader(content))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("ler cabeçalho: %w", err)
	}
	if len(header) < 4 {
		return nil, fmt.Errorf("cabeçalho inválido: esperado ao menos 4 colunas, recebido %d", len(header))
	}

	var items []ncmRow
	line := 1

	for {
		line++
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", line, err)
		}

		if isEmptyRecord(record) {
			continue
		}

		for len(record) < 4 {
			record = append(record, "")
		}

		rawCode := cleanField(record[0])
		ex := cleanField(record[1])
		description := normalizeDescription(record[2])
		rawRate := cleanField(record[3])

		if rawCode == "" || description == "" {
			continue
		}

		code := normalizeNCMCode(rawCode)
		level := detectLevel(code)
		if level == 0 {
			log.Printf("linha %d ignorada: código NCM inválido %q", line, rawCode)
			continue
		}

		ipiRate, err := parseIPIRate(rawRate)
		if err != nil {
			return nil, fmt.Errorf("linha %d: alíquota inválida %q: %w", line, rawRate, err)
		}

		items = append(items, ncmRow{
			Code:        code,
			EX:          normalizeEXCode(ex),
			Description: description,
			IPIRate:     ipiRate,
			Level:       level,
		})
	}

	return items, nil
}

func dedupeRows(rows []ncmRow) []ncmRow {
	seen := make(map[string]ncmRow, len(rows))

	for _, row := range rows {
		key := row.Code + "|" + defaultEXCode(row.EX)
		seen[key] = row
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]ncmRow, 0, len(keys))
	for _, k := range keys {
		out = append(out, seen[k])
	}

	return out
}

func normalizeEncoding(content []byte) []byte {
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	if utf8.Valid(content) {
		return content
	}

	decoded, err := charmap.Windows1252.NewDecoder().Bytes(content)
	if err == nil && utf8.Valid(decoded) {
		log.Println("arquivo convertido de Windows-1252/ANSI para UTF-8")
		return decoded
	}

	decoded, err = charmap.ISO8859_1.NewDecoder().Bytes(content)
	if err == nil && utf8.Valid(decoded) {
		log.Println("arquivo convertido de ISO-8859-1 para UTF-8")
		return decoded
	}

	return content
}

func loadTableColumns(ctx context.Context, pool *pgxpool.Pool, tableName string) (map[string]columnInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			column_name,
			data_type,
			COALESCE(character_maximum_length, 0) AS character_maximum_length,
			(is_nullable = 'YES') AS is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
	`, tableName)
	if err != nil {
		return nil, fmt.Errorf("consultar information_schema.columns: %w", err)
	}
	defer rows.Close()

	cols := make(map[string]columnInfo)
	for rows.Next() {
		var c columnInfo
		if err := rows.Scan(&c.Name, &c.DataType, &c.MaxLength, &c.IsNullable); err != nil {
			return nil, fmt.Errorf("scan column info: %w", err)
		}
		cols[c.Name] = c
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar colunas: %w", err)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("tabela %q não encontrada ou sem colunas", tableName)
	}

	return cols, nil
}

func importRows(ctx context.Context, pool *pgxpool.Pool, rows []ncmRow, columns map[string]columnInfo, truncate bool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	if truncate {
		log.Println("apagando registros antigos de ncm_catalog...")
		if _, err := tx.Exec(ctx, `DELETE FROM ncm_catalog`); err != nil {
			return fmt.Errorf("limpar tabela ncm_catalog: %w", err)
		}
	}

	insertColumns := []string{"code", "description"}
	if hasColumn(columns, "ex") {
		insertColumns = append(insertColumns, "ex")
	}
	if hasColumn(columns, "ex_code") {
		insertColumns = append(insertColumns, "ex_code")
	}
	if hasColumn(columns, "ipi_rate") {
		insertColumns = append(insertColumns, "ipi_rate")
	}
	if hasColumn(columns, "level") {
		insertColumns = append(insertColumns, "level")
	}
	if hasColumn(columns, "level_type") {
		insertColumns = append(insertColumns, "level_type")
	}
	if hasColumn(columns, "parent_code") {
		insertColumns = append(insertColumns, "parent_code")
	}
	if hasColumn(columns, "chapter_code") {
		insertColumns = append(insertColumns, "chapter_code")
	}
	if hasColumn(columns, "heading_code") {
		insertColumns = append(insertColumns, "heading_code")
	}
	if hasColumn(columns, "item_code") {
		insertColumns = append(insertColumns, "item_code")
	}
	if hasColumn(columns, "full_description") {
		insertColumns = append(insertColumns, "full_description")
	}
	if hasColumn(columns, "is_active") {
		insertColumns = append(insertColumns, "is_active")
	}

	placeholders := make([]string, 0, len(insertColumns))
	for i := range insertColumns {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	stmt := fmt.Sprintf(
		`INSERT INTO ncm_catalog (%s) VALUES (%s)`,
		strings.Join(insertColumns, ", "),
		strings.Join(placeholders, ", "),
	)

	for _, row := range rows {
		args := make([]any, 0, len(insertColumns))

		for _, col := range insertColumns {
			switch col {
			case "code":
				args = append(args, fitString(columns[col], row.Code))
			case "description":
				args = append(args, fitString(columns[col], row.Description))
			case "ex":
				args = append(args, nilIfEmpty(fitString(columns[col], row.EX)))
			case "ex_code":
				args = append(args, fitString(columns[col], defaultEXCode(row.EX)))
			case "ipi_rate":
				args = append(args, row.IPIRate)
			case "level":
				args = append(args, row.Level)
			case "level_type":
				args = append(args, fitString(columns[col], levelType(row.Code)))
			case "parent_code":
				args = append(args, nilIfEmpty(fitString(columns[col], parentCodeForColumn(row.Code, columns[col]))))
			case "chapter_code":
				args = append(args, nilIfEmpty(fitString(columns[col], chapterCode(row.Code))))
			case "heading_code":
				args = append(args, nilIfEmpty(fitString(columns[col], headingCode(row.Code))))
			case "item_code":
				args = append(args, nilIfEmpty(fitString(columns[col], itemCodeForColumn(row.Code, columns[col]))))
			case "full_description":
				args = append(args, fitString(columns[col], row.Description))
			case "is_active":
				args = append(args, true)
			default:
				return fmt.Errorf("coluna não suportada no importador: %s", col)
			}
		}

		if _, err := tx.Exec(ctx, stmt, args...); err != nil {
			return fmt.Errorf("inserir código %s: %w", row.Code, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar transação: %w", err)
	}

	return nil
}

func hasColumn(columns map[string]columnInfo, name string) bool {
	_, ok := columns[name]
	return ok
}

func fitString(col columnInfo, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if col.MaxLength > 0 && len(value) > col.MaxLength {
		return value[:col.MaxLength]
	}
	return value
}

func parseIPIRate(value string) (*float64, error) {
	value = strings.TrimSpace(strings.ToUpper(value))
	value = strings.TrimSuffix(value, ";")

	if value == "" {
		return nil, nil
	}

	if value == "NT" {
		v := 0.0
		return &v, nil
	}

	value = strings.ReplaceAll(value, "%", "")
	value = strings.ReplaceAll(value, ",", ".")

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func normalizeNCMCode(code string) string {
	return onlyDigits(code)
}

func normalizeEXCode(ex string) string {
	ex = strings.TrimSpace(ex)
	if ex == "" {
		return ""
	}
	return onlyDigits(ex)
}

func defaultEXCode(ex string) string {
	ex = strings.TrimSpace(ex)
	if ex == "" {
		return "00"
	}
	return ex
}

func detectLevel(code string) int {
	switch len(code) {
	case 2:
		return 1
	case 4:
		return 2
	case 5, 6:
		return 3
	case 7, 8:
		return 4
	default:
		return 0
	}
}

func levelType(code string) string {
	switch detectLevel(code) {
	case 1:
		return "chapter"
	case 2:
		return "heading"
	case 3:
		return "subheading"
	case 4:
		return "item"
	default:
		return "unknown"
	}
}

func parentCodeForColumn(code string, col columnInfo) string {
	value := parentCode(code)
	if col.MaxLength > 0 && len(value) > col.MaxLength {
		return value[:col.MaxLength]
	}
	return value
}

func itemCodeForColumn(code string, col columnInfo) string {
	value := itemCode(code)
	if col.MaxLength > 0 && len(value) > col.MaxLength {
		return value[:col.MaxLength]
	}
	return value
}

func parentCode(code string) string {
	switch len(code) {
	case 2:
		return ""
	case 4:
		return code[:2]
	case 5, 6:
		return code[:4]
	case 7, 8:
		if len(code) >= 6 {
			return code[:6]
		}
	}
	return ""
}

func chapterCode(code string) string {
	if len(code) < 2 {
		return ""
	}
	return code[:2]
}

func headingCode(code string) string {
	if len(code) < 4 {
		return ""
	}
	return code[:4]
}

func itemCode(code string) string {
	if len(code) >= 8 {
		return code[:8]
	}
	return code
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeDescription(s string) string {
	s = cleanField(s)
	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimPrefix(s, "-- ")
	s = strings.TrimPrefix(s, "--- ")
	return strings.TrimSpace(s)
}

func cleanField(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ";")
	s = strings.TrimSpace(s)
	return s
}

func isEmptyRecord(record []string) bool {
	for _, field := range record {
		if cleanField(field) != "" {
			return false
		}
	}
	return true
}

func nilIfEmpty(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func sortedColumnNames(m map[string]columnInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func defaultDatabaseURL() string {
	if env := strings.TrimSpace(os.Getenv("DATABASE_URL")); env != "" {
		return env
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword("postgres", "postgres"),
		Host:   "127.0.0.1:5432",
		Path:   "fiscal_platform",
	}

	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	return u.String()
}