package importers

import (
	"context"
	"encoding/csv"
	"fmt"
	"html"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CESTImporter struct {
	db *pgxpool.Pool
}

func NewCESTImporter(db *pgxpool.Pool) *CESTImporter {
	return &CESTImporter{db: db}
}

type CESTImportRow struct {
	Code        string
	NCMCode     string
	Segment     string
	Description string
	LegalSource string
}

func (i *CESTImporter) ImportCSV(ctx context.Context, filePath string, sourceName string, versionLabel string) error {
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
	content = normalizeCESTContent(content)

	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err == nil {
		headerIndex, indexes, headerErr := findCESTHeader(rows)
		if headerErr == nil {
			return i.importCESTCSVRows(ctx, rows, headerIndex, indexes, sourceName, versionLabel, filePath)
		}
	}

	parsedRows, err := parseCESTOfficialText(content, sourceName)
	if err != nil {
		return err
	}

	return i.importCESTRows(ctx, parsedRows, sourceName, "text", versionLabel, filePath)
}

func (i *CESTImporter) importCESTCSVRows(ctx context.Context, rows [][]string, headerIndex int, indexes map[string]int, sourceName string, versionLabel string, filePath string) error {
	parsedRows := make([]CESTImportRow, 0, len(rows))
	for _, record := range rows[headerIndex+1:] {
		if !rowHasContent(record) {
			continue
		}
		parsedRows = append(parsedRows, parseCESTRow(record, indexes, sourceName))
	}

	return i.importCESTRows(ctx, parsedRows, sourceName, "csv", versionLabel, filePath)
}

func (i *CESTImporter) importCESTRows(ctx context.Context, rows []CESTImportRow, sourceName string, sourceType string, versionLabel string, filePath string) error {
	totalRows := 0
	for _, row := range rows {
		if row.Code != "" || row.Description != "" {
			totalRows++
		}
	}
	if totalRows == 0 {
		return fmt.Errorf("arquivo nao possui linhas de CEST apos o cabecalho")
	}

	batchID, err := createImportBatch(ctx, i.db, sourceName, sourceType, versionLabel, filePath, totalRows)
	if err != nil {
		return err
	}

	successRows := 0
	failedRows := 0
	for lineNumber, row := range rows {
		if row.Code == "" || row.Description == "" {
			failedRows++
			continue
		}

		expandedRows := expandCESTNCMReferences(row)
		rowFailed := false
		for _, expandedRow := range expandedRows {
			if err := i.upsertCEST(ctx, expandedRow); err != nil {
				rowFailed = true
				fmt.Printf("line %d failed: %v\n", lineNumber+1, err)
				continue
			}
		}
		if rowFailed {
			failedRows++
			if len(expandedRows) == 0 {
				continue
			}
		} else {
			successRows++
		}
	}

	if err := finishImportBatch(ctx, i.db, batchID, successRows, failedRows); err != nil {
		return err
	}
	if successRows == 0 {
		return fmt.Errorf("import finished with zero successful rows; check csv encoding/header mapping")
	}

	return nil
}

func findCESTHeader(rows [][]string) (int, map[string]int, error) {
	for idx, row := range rows {
		indexes := mapCESTHeaderIndexes(row)
		if _, ok := indexes["cest"]; !ok {
			continue
		}
		if _, ok := indexes["description"]; !ok {
			continue
		}
		return idx, indexes, nil
	}

	return -1, nil, fmt.Errorf("cabecalho de CEST nao encontrado no arquivo")
}

func mapCESTHeaderIndexes(header []string) map[string]int {
	out := make(map[string]int)
	for idx, name := range header {
		key := normalizeCESTHeader(name)
		if key != "" {
			out[key] = idx
		}
	}
	return out
}

func normalizeCESTHeader(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "\u00a0", "")
	v = strings.NewReplacer(
		"Ã§", "c",
		"Ã£", "a",
		"Ã¡", "a",
		"Ã©", "e",
		"Ãª", "e",
		"Ã­", "i",
		"Ã³", "o",
		"Ãµ", "o",
		"_", "",
		"-", "",
		"(", "",
		")", "",
	).Replace(v)

	switch {
	case v == "cest" || strings.Contains(v, "codigocest"):
		return "cest"
	case v == "ncm" || strings.Contains(v, "ncmsh"):
		return "ncm"
	case strings.Contains(v, "segmento"):
		return "segment"
	case strings.Contains(v, "descricao") || strings.Contains(v, "descricaoproduto") || strings.Contains(v, "item"):
		return "description"
	case strings.Contains(v, "fundamento") || strings.Contains(v, "fonte"):
		return "legal_source"
	default:
		return v
	}
}

func parseCESTRow(record []string, indexes map[string]int, sourceName string) CESTImportRow {
	get := func(name string) string {
		idx, ok := indexes[name]
		if !ok || idx >= len(record) {
			return ""
		}
		return cleanText(record[idx])
	}

	return CESTImportRow{
		Code:        normalizeNumericCode(get("cest")),
		NCMCode:     normalizeNCMReference(get("ncm")),
		Segment:     get("segment"),
		Description: get("description"),
		LegalSource: firstNonEmpty(get("legal_source"), sourceName),
	}
}

var cestCodePattern = regexp.MustCompile(`\b\d{2}\.?\d{3}\.?\d{2}\b`)
var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
var repeatedSpacePattern = regexp.MustCompile(`[ \t]+`)

func normalizeCESTContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.ReplaceAll(content, "\u00a0", " ")
	return html.UnescapeString(content)
}

func parseCESTOfficialText(content string, sourceName string) ([]CESTImportRow, error) {
	content = normalizeCESTContent(content)
	content = strings.NewReplacer(
		"</td>", " | ",
		"</th>", " | ",
		"</tr>", "\n",
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
	).Replace(content)
	content = htmlTagPattern.ReplaceAllString(content, " ")

	rows := make([]CESTImportRow, 0)
	currentSegment := ""

	for _, rawLine := range strings.Split(content, "\n") {
		line := cleanText(rawLine)
		line = repeatedSpacePattern.ReplaceAllString(line, " ")
		if line == "" {
			continue
		}

		normalizedLine := strings.ToLower(line)
		if strings.Contains(normalizedLine, "anexo") && !cestCodePattern.MatchString(line) {
			currentSegment = line
			continue
		}

		cest := cestCodePattern.FindString(line)
		if cest == "" {
			continue
		}

		row := parseCESTOfficialLine(line, cest, currentSegment, sourceName)
		if row.Code != "" && row.Description != "" {
			rows = append(rows, row)
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("nao foi possivel identificar linhas CEST no conteudo informado; cole uma tabela com colunas CEST, NCM/SH e descricao")
	}

	return rows, nil
}

func parseCESTOfficialLine(line string, cest string, segment string, sourceName string) CESTImportRow {
	parts := splitCESTTableLine(line)
	cestIndex := -1
	for idx, part := range parts {
		if normalizeNumericCode(part) == normalizeNumericCode(cest) {
			cestIndex = idx
			break
		}
	}

	ncm := ""
	description := ""
	if cestIndex >= 0 {
		if cestIndex+1 < len(parts) {
			ncm = parts[cestIndex+1]
		}
		if cestIndex+2 < len(parts) {
			description = strings.Join(parts[cestIndex+2:], " ")
		}
	}

	if description == "" {
		afterCEST := strings.TrimSpace(line[strings.Index(line, cest)+len(cest):])
		fields := strings.Fields(afterCEST)
		ncmParts := make([]string, 0)
		descriptionParts := make([]string, 0)
		for _, field := range fields {
			cleaned := strings.Trim(field, ",;")
			if descriptionParts == nil {
				descriptionParts = make([]string, 0)
			}
			if len(descriptionParts) == 0 && looksLikeNCMToken(cleaned) {
				ncmParts = append(ncmParts, cleaned)
				continue
			}
			descriptionParts = append(descriptionParts, field)
		}
		ncm = strings.Join(ncmParts, " ")
		description = strings.Join(descriptionParts, " ")
	}

	return CESTImportRow{
		Code:        normalizeNumericCode(cest),
		NCMCode:     normalizeNCMReference(ncm),
		Segment:     cleanText(segment),
		Description: cleanText(description),
		LegalSource: firstNonEmpty(sourceName, "CONFAZ CEST"),
	}
}

func splitCESTTableLine(line string) []string {
	rawParts := strings.Split(line, "|")
	if len(rawParts) == 1 {
		rawParts = strings.Split(line, "\t")
	}

	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = cleanText(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func looksLikeNCMToken(value string) bool {
	digits := normalizeNumericCode(value)
	return len(digits) >= 2 && len(digits) <= 16
}

func normalizeNCMReference(v string) string {
	refs := splitNCMReferences(v)
	return strings.Join(refs, " ")
}

func splitNCMReferences(v string) []string {
	v = strings.NewReplacer(",", " ", ";", " ", "/", " ").Replace(v)
	refs := make([]string, 0)
	seen := make(map[string]bool)
	for _, field := range strings.Fields(v) {
		code := normalizeNumericCode(field)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		refs = append(refs, code)
	}
	return refs
}

func expandCESTNCMReferences(row CESTImportRow) []CESTImportRow {
	refs := splitNCMReferences(row.NCMCode)
	if len(refs) == 0 {
		row.NCMCode = ""
		return []CESTImportRow{row}
	}

	rows := make([]CESTImportRow, 0, len(refs))
	for _, ref := range refs {
		expanded := row
		expanded.NCMCode = ref
		rows = append(rows, expanded)
	}
	return rows
}

func normalizeNumericCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (i *CESTImporter) upsertCEST(ctx context.Context, row CESTImportRow) error {
	_, err := i.db.Exec(ctx, `
		INSERT INTO cest_catalog (
			code,
			ncm_code,
			segment,
			description,
			legal_source,
			is_active,
			updated_at
		)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, NULLIF($5, ''), TRUE, NOW())
		ON CONFLICT (code, (COALESCE(ncm_code, ''))) DO UPDATE
		SET segment = EXCLUDED.segment,
		    description = EXCLUDED.description,
		    legal_source = EXCLUDED.legal_source,
		    is_active = TRUE,
		    updated_at = NOW()
	`, row.Code, row.NCMCode, row.Segment, row.Description, row.LegalSource)
	if err != nil {
		return fmt.Errorf("insert cest row: %w", err)
	}
	return nil
}
