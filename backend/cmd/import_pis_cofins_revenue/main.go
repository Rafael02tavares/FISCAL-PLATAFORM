package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
)

type revenueNatureRow struct {
	TableCode   string
	Code        string
	Description string
	SourceFile  string
}

func main() {
	var (
		dirPath = flag.String("dir", `C:\siitServer\tabelas\SPED\Contribuicoes\Genericas`, "diretorio com tabelas 4.3.x do SPED")
		dbURL   = flag.String("database", defaultDatabaseURL(), "URL de conexao com o PostgreSQL")
		dryRun  = flag.Bool("dry-run", false, "apenas le e contabiliza os arquivos, sem gravar no banco")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if strings.TrimSpace(*dbURL) == "" {
		log.Fatal("database URL nao informada; use --database ou configure DATABASE_URL")
	}

	rows, err := readRevenueNatureFiles(*dirPath)
	if err != nil {
		log.Fatal(err)
	}
	if len(rows) == 0 {
		log.Fatal("nenhum codigo de natureza de receita encontrado")
	}

	if *dryRun {
		printSummary(rows)
		return
	}

	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("erro ao validar conexao com banco: %v", err)
	}

	imported, err := importRows(ctx, pool, rows)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("codigos de natureza da receita PIS/COFINS importados: %d", imported)
}

func printSummary(rows []revenueNatureRow) {
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.TableCode]++
	}

	tableCodes := make([]string, 0, len(counts))
	for tableCode := range counts {
		tableCodes = append(tableCodes, tableCode)
	}
	sort.Strings(tableCodes)

	total := 0
	for _, tableCode := range tableCodes {
		total += counts[tableCode]
		log.Printf("%s: %d codigos", tableCode, counts[tableCode])
	}
	log.Printf("total: %d codigos", total)
}

func readRevenueNatureFiles(dirPath string) ([]revenueNatureRow, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "tabela_4_3_*.txt"))
	if err != nil {
		return nil, fmt.Errorf("localizar arquivos: %w", err)
	}
	sort.Strings(files)

	rows := make([]revenueNatureRow, 0)
	for _, file := range files {
		tableCode := tableCodeFromFile(file)
		if tableCode == "" {
			continue
		}

		fileRows, err := readRevenueNatureFile(file, tableCode)
		if err != nil {
			return nil, err
		}
		rows = append(rows, fileRows...)
	}

	return rows, nil
}

func readRevenueNatureFile(filePath string, tableCode string) ([]revenueNatureRow, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("abrir arquivo %q: %w", filePath, err)
	}

	content := normalizeEncoding(raw)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	rows := make([]revenueNatureRow, 0)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			log.Printf("%s linha %d ignorada: formato invalido", filepath.Base(filePath), lineNumber)
			continue
		}

		code := strings.TrimSpace(parts[0])
		description := strings.TrimSpace(parts[1])
		if code == "" || description == "" {
			continue
		}

		rows = append(rows, revenueNatureRow{
			TableCode:   tableCode,
			Code:        code,
			Description: description,
			SourceFile:  filepath.Base(filePath),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ler arquivo %q: %w", filePath, err)
	}

	return rows, nil
}

func importRows(ctx context.Context, pool *pgxpool.Pool, rows []revenueNatureRow) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("iniciar transacao: %w", err)
	}
	defer tx.Rollback(ctx)

	imported := 0
	for _, row := range rows {
		_, err := tx.Exec(ctx, `
			INSERT INTO pis_cofins_revenue_natures (
				table_code,
				code,
				description,
				source_name,
				source_file,
				is_active,
				updated_at
			)
			VALUES ($1, $2, $3, 'SPED Contribuicoes', $4, TRUE, NOW())
			ON CONFLICT (table_code, code)
			DO UPDATE SET
				description = EXCLUDED.description,
				source_file = EXCLUDED.source_file,
				is_active = TRUE,
				updated_at = NOW()
		`, row.TableCode, row.Code, row.Description, row.SourceFile)
		if err != nil {
			return imported, fmt.Errorf("importar %s codigo %s: %w", row.TableCode, row.Code, err)
		}
		imported++
	}

	if err := tx.Commit(ctx); err != nil {
		return imported, fmt.Errorf("confirmar transacao: %w", err)
	}

	return imported, nil
}

func tableCodeFromFile(filePath string) string {
	name := filepath.Base(filePath)
	re := regexp.MustCompile(`tabela_4_3_(\d+)\.txt$`)
	match := re.FindStringSubmatch(name)
	if len(match) != 2 {
		return ""
	}
	return "4.3." + match[1]
}

func normalizeEncoding(content []byte) []byte {
	if utf8.Valid(content) {
		return content
	}
	decoded, err := charmap.Windows1252.NewDecoder().Bytes(content)
	if err != nil {
		return content
	}
	return decoded
}

func defaultDatabaseURL() string {
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}
	return "postgres://postgres:postgres@127.0.0.1:5432/fiscal_platform?sslmode=disable"
}
