package importers

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NCMImporter struct {
	db *pgxpool.Pool
}

func NewNCMImporter(db *pgxpool.Pool) *NCMImporter {
	return &NCMImporter{db: db}
}

type NCMImportRow struct {
	Code           string
	ExCode         string
	Description    string
	Aliquota       string
	ChapterCode    string
	HeadingCode    string
	ItemCode       string
	ParentCode     string
	LevelType      string
	LegalSource    string
	LegalReference string
	OfficialNotes  string
}

func (i *NCMImporter) ImportCSV(
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

	if len(rows) < 2 {
		return fmt.Errorf("csv must contain header and at least one row")
	}

	header := rows[0]
	indexes := mapHeaderIndexesPT(header)

	// fallback defensivo para o formato real do seu arquivo:
	// NCM ;EX;DESCRIÇÃO ;ALÍQUOTA (%);
	if _, ok := indexes["ncm"]; !ok && len(header) > 0 {
		indexes["ncm"] = 0
	}
	if _, ok := indexes["ex"]; !ok && len(header) > 1 {
		indexes["ex"] = 1
	}
	if _, ok := indexes["descricao"]; !ok && len(header) > 2 {
		indexes["descricao"] = 2
	}
	if _, ok := indexes["aliquota(%)"]; !ok && len(header) > 3 {
		indexes["aliquota(%)"] = 3
	}

	batchID, err := i.createImportBatch(ctx, sourceName, "csv", versionLabel, filePath, len(rows)-1)
	if err != nil {
		return err
	}

	successRows := 0
	failedRows := 0

	for lineNumber, record := range rows[1:] {
		row := parseNCMRowPT(record, indexes)

		if strings.TrimSpace(row.Code) == "" || strings.TrimSpace(row.Description) == "" {
			failedRows++
			continue
		}

		if err := i.upsertNCM(ctx, batchID, row); err != nil {
			failedRows++
			fmt.Printf("line %d failed: %v\n", lineNumber+2, err)
			continue
		}

		successRows++
	}

	if successRows == 0 {
		_ = i.finishImportBatch(ctx, batchID, successRows, failedRows)
		return fmt.Errorf("import finished with zero successful rows; check csv encoding/header mapping")
	}

	if err := i.finishImportBatch(ctx, batchID, successRows, failedRows); err != nil {
		return err
	}

	return nil
}

func mapHeaderIndexesPT(header []string) map[string]int {
	out := make(map[string]int)
	for idx, name := range header {
		key := normalizeHeader(name)
		if key != "" {
			out[key] = idx
		}
	}
	return out
}

func normalizeHeader(v string) string {
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
		"(", "",
		")", "",
		"%", "",
		"��", "",
		"�", "",
	)
	v = replacer.Replace(v)

	// mapeamento tolerante para cabeçalhos quebrados
	switch {
	case strings.HasPrefix(v, "ncm"):
		return "ncm"
	case strings.HasPrefix(v, "ex"):
		return "ex"
	case strings.HasPrefix(v, "descr"):
		return "descricao"
	case strings.HasPrefix(v, "aliq"):
		return "aliquota(%)"
	}

	return v
}

func parseNCMRowPT(record []string, indexes map[string]int) NCMImportRow {
	get := func(name string) string {
		idx, ok := indexes[name]
		if !ok || idx >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[idx])
	}

	rawCode := get("ncm")
	normCode := normalizeNCMCode(rawCode)

	row := NCMImportRow{
		Code:           normCode,
		ExCode:         get("ex"),
		Description:    cleanText(get("descricao")),
		Aliquota:       cleanText(get("aliquota(%)")),
		LegalSource:    "TIPI",
		LegalReference: rawCode,
	}

	row.ChapterCode = chapterCodeFromNCM(normCode)
	row.HeadingCode = headingCodeFromNCM(normCode)
	row.ItemCode = itemCodeFromNCM(normCode)
	row.LevelType = detectLevelType(rawCode, normCode)
	row.ParentCode = parentCode(rawCode, normCode)

	if row.Aliquota != "" {
		row.OfficialNotes = "Alíquota/TIPI: " + row.Aliquota
	}

	return row
}

func cleanText(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"`)
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\t", " ")
	v = strings.Join(strings.Fields(v), " ")
	return v
}

func normalizeNCMCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func chapterCodeFromNCM(code string) string {
	if len(code) >= 2 {
		return code[:2]
	}
	return ""
}

func headingCodeFromNCM(code string) string {
	if len(code) >= 4 {
		return code[:4]
	}
	return ""
}

func itemCodeFromNCM(code string) string {
	if len(code) >= 6 {
		return code[:6]
	}
	return ""
}

func detectLevelType(rawCode, normalized string) string {
	switch len(normalized) {
	case 2:
		return "chapter"
	case 4:
		return "heading"
	case 5, 6:
		return "item"
	case 7:
		return "subitem"
	case 8:
		return "ncm"
	default:
		if strings.Count(rawCode, ".") >= 2 {
			return "ncm"
		}
		return "other"
	}
}

func parentCode(rawCode, normalized string) string {
	switch len(normalized) {
	case 4:
		return normalized[:2]
	case 5, 6:
		return normalized[:4]
	case 7:
		return normalized[:6]
	case 8:
		return normalized[:6]
	default:
		return ""
	}
}

func (i *NCMImporter) createImportBatch(
	ctx context.Context,
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
	err := i.db.QueryRow(
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

func (i *NCMImporter) finishImportBatch(
	ctx context.Context,
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

	_, err := i.db.Exec(ctx, query, batchID, successRows, failedRows)
	if err != nil {
		return fmt.Errorf("finish import batch: %w", err)
	}

	return nil
}

func (i *NCMImporter) upsertNCM(
	ctx context.Context,
	batchID string,
	row NCMImportRow,
) error {
	disableQuery := `
		UPDATE ncm_catalog
		SET is_active = FALSE,
		    updated_at = NOW()
		WHERE code = $1
		  AND COALESCE(ex_code, '') = COALESCE($2, '')
		  AND is_active = TRUE
	`
	_, err := i.db.Exec(ctx, disableQuery, row.Code, emptyToNil(row.ExCode))
	if err != nil {
		return fmt.Errorf("disable previous ncm version: %w", err)
	}

	insertQuery := `
		INSERT INTO ncm_catalog (
			import_batch_id,
			code,
			description,
			full_description,
			chapter_code,
			heading_code,
			item_code,
			parent_code,
			level_type,
			ex_code,
			legal_source,
			legal_reference,
			official_notes,
			is_active,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			NULLIF($9, ''),
			$10,
			$11,
			$12,
			TRUE,
			NOW()
		)
	`

	_, err = i.db.Exec(
		ctx,
		insertQuery,
		batchID,
		row.Code,
		row.Description,
		row.ChapterCode,
		row.HeadingCode,
		row.ItemCode,
		row.ParentCode,
		row.LevelType,
		row.ExCode,
		row.LegalSource,
		row.LegalReference,
		row.OfficialNotes,
	)
	if err != nil {
		return fmt.Errorf("insert ncm row: %w", err)
	}

	return nil
}

func emptyToNil(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
