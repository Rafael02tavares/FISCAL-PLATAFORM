package importers

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CFOPImporter struct {
	db *pgxpool.Pool
}

func NewCFOPImporter(db *pgxpool.Pool) *CFOPImporter {
	return &CFOPImporter{db: db}
}

type CFOPImportRow struct {
	Code             string
	Description      string
	OperationType    string
	IndNFe           bool
	IndCommunication bool
	IndTransport     bool
	IndDevolution    bool
}

func (i *CFOPImporter) ImportCSV(
	ctx context.Context,
	filePath string,
	sourceName string,
	versionLabel string,
) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read csv file bytes: %w", err)
	}

	content := string(contentBytes)
	content = strings.ReplaceAll(content, "\u201c", `"`)
	content = strings.ReplaceAll(content, "\u201d", `"`)
	content = strings.ReplaceAll(content, "\u201e", `"`)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}

	headerIndex, indexes, err := findCFOPHeader(rows)
	if err != nil {
		return err
	}

	totalRows := 0
	for _, record := range rows[headerIndex+1:] {
		if rowHasContent(record) {
			totalRows++
		}
	}

	if totalRows == 0 {
		return fmt.Errorf("arquivo nao possui linhas de CFOP apos o cabecalho")
	}

	batchID, err := createImportBatch(ctx, i.db, sourceName, "csv", versionLabel, filePath, totalRows)
	if err != nil {
		return err
	}

	successRows := 0
	failedRows := 0

	for lineNumber, record := range rows[headerIndex+1:] {
		if !rowHasContent(record) {
			continue
		}

		row := parseCFOPRow(record, indexes)
		if row.Code == "" || row.Description == "" {
			failedRows++
			continue
		}

		if err := i.upsertCFOP(ctx, row); err != nil {
			failedRows++
			fmt.Printf("line %d failed: %v\n", headerIndex+lineNumber+2, err)
			continue
		}

		successRows++
	}

	if err := finishImportBatch(ctx, i.db, batchID, successRows, failedRows); err != nil {
		return err
	}

	if successRows == 0 {
		return fmt.Errorf("import finished with zero successful rows; check csv encoding/header mapping")
	}

	return nil
}

func findCFOPHeader(rows [][]string) (int, map[string]int, error) {
	for idx, row := range rows {
		indexes := mapCFOPHeaderIndexes(row)
		if _, ok := indexes["cfop"]; !ok {
			continue
		}
		if _, ok := indexes["description"]; !ok {
			continue
		}
		return idx, indexes, nil
	}

	return -1, nil, fmt.Errorf("cabecalho de CFOP nao encontrado no arquivo")
}

func mapCFOPHeaderIndexes(header []string) map[string]int {
	out := make(map[string]int)
	for idx, name := range header {
		key := normalizeCFOPHeader(name)
		if key != "" {
			out[key] = idx
		}
	}
	return out
}

func normalizeCFOPHeader(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "\u00a0", "")

	replacer := strings.NewReplacer(
		"ç", "c",
		"ã", "a",
		"á", "a",
		"à", "a",
		"â", "a",
		"é", "e",
		"ê", "e",
		"í", "i",
		"ó", "o",
		"ô", "o",
		"õ", "o",
		"ú", "u",
		"ü", "u",
		"_", "",
		"-", "",
		"(", "",
		")", "",
	)
	v = replacer.Replace(v)

	switch {
	case v == "cfop" || strings.HasPrefix(v, "codigofiscal"):
		return "cfop"
	case strings.Contains(v, "descricaocfop") || strings.Contains(v, "descricao"):
		return "description"
	case strings.Contains(v, "grupocfop"):
		return "group"
	case strings.Contains(v, "iniciovigencia"):
		return "start_date"
	case strings.Contains(v, "indnfe"):
		return "ind_nfe"
	case strings.Contains(v, "indcomunication"), strings.Contains(v, "indcomunicacao"):
		return "ind_communication"
	case strings.Contains(v, "indtransport"), strings.Contains(v, "indtransporte"):
		return "ind_transport"
	case strings.Contains(v, "inddevolution"), strings.Contains(v, "inddevolucao"):
		return "ind_devolution"
	}

	return v
}

func parseCFOPRow(record []string, indexes map[string]int) CFOPImportRow {
	get := func(name string) string {
		idx, ok := indexes[name]
		if !ok || idx >= len(record) {
			return ""
		}
		return cleanText(record[idx])
	}

	code := normalizeCFOPCode(get("cfop"))
	description := get("description")
	group := get("group")

	return CFOPImportRow{
		Code:             code,
		Description:      description,
		OperationType:    detectCFOPOperationType(code, group),
		IndNFe:           parseBoolIndicator(get("ind_nfe"), true),
		IndCommunication: parseBoolIndicator(get("ind_communication"), strings.Contains(strings.ToLower(description), "comunica")),
		IndTransport:     parseBoolIndicator(get("ind_transport"), strings.Contains(strings.ToLower(description), "transporte")),
		IndDevolution:    parseBoolIndicator(get("ind_devolution"), strings.Contains(strings.ToLower(description), "devolu")),
	}
}

func normalizeCFOPCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func detectCFOPOperationType(code string, group string) string {
	group = strings.ToLower(strings.TrimSpace(group))

	switch {
	case strings.Contains(group, "entrada"), strings.HasPrefix(code, "1"), strings.HasPrefix(code, "2"), strings.HasPrefix(code, "3"):
		return "entrada"
	case strings.Contains(group, "saida"), strings.Contains(group, "prestacoes"), strings.HasPrefix(code, "5"), strings.HasPrefix(code, "6"), strings.HasPrefix(code, "7"):
		return "saida"
	default:
		return ""
	}
}

func parseBoolIndicator(value string, fallback bool) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "1", "true", "sim", "s", "x":
		return true
	case "0", "false", "nao", "não", "n":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}

func rowHasContent(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func (i *CFOPImporter) upsertCFOP(ctx context.Context, row CFOPImportRow) error {
	query := `
		INSERT INTO cfop_catalog (
			code,
			description,
			ind_nfe,
			ind_comunication,
			ind_transport,
			ind_devolution,
			operation_type
		)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))
		ON CONFLICT (code) DO UPDATE
		SET description = EXCLUDED.description,
		    ind_nfe = EXCLUDED.ind_nfe,
		    ind_comunication = EXCLUDED.ind_comunication,
		    ind_transport = EXCLUDED.ind_transport,
		    ind_devolution = EXCLUDED.ind_devolution,
		    operation_type = EXCLUDED.operation_type
	`

	_, err := i.db.Exec(
		ctx,
		query,
		row.Code,
		row.Description,
		row.IndNFe,
		row.IndCommunication,
		row.IndTransport,
		row.IndDevolution,
		row.OperationType,
	)
	if err != nil {
		return fmt.Errorf("insert cfop row: %w", err)
	}

	return nil
}

func createImportBatch(
	ctx context.Context,
	db *pgxpool.Pool,
	sourceName string,
	sourceType string,
	versionLabel string,
	fileName string,
	totalRows int,
) (string, error) {
	query := `
		INSERT INTO import_batches (
			source_name,
			source_type,
			version_label,
			file_name,
			total_rows
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var batchID string
	err := db.QueryRow(
		ctx,
		query,
		sourceName,
		sourceType,
		versionLabel,
		fileName,
		totalRows,
	).Scan(&batchID)
	if err != nil {
		return "", fmt.Errorf("create import batch: %w", err)
	}

	return batchID, nil
}

func finishImportBatch(
	ctx context.Context,
	db *pgxpool.Pool,
	batchID string,
	successRows int,
	failedRows int,
) error {
	query := `
		UPDATE import_batches
		SET success_rows = $2,
		    failed_rows = $3
		WHERE id = $1
	`

	_, err := db.Exec(ctx, query, batchID, successRows, failedRows)
	if err != nil {
		return fmt.Errorf("finish import batch: %w", err)
	}

	return nil
}

func buildImportTempPattern(prefix string, fileName string) string {
	base := strings.TrimSpace(fileName)
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		default:
			return -1
		}
	}, base)

	if base == "" {
		base = prefix + "-upload"
	}

	if len(base) > 40 {
		base = base[:40]
	}

	return prefix + "-" + base + "-" + strconv.FormatInt(int64(os.Getpid()), 10) + "-*.csv"
}
