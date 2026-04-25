package importers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CBenefImporter struct {
	db *pgxpool.Pool
}

func NewCBenefImporter(db *pgxpool.Pool) *CBenefImporter {
	return &CBenefImporter{db: db}
}

type CBenefImportRow struct {
	UF             string
	Code           string
	AppliesSimples bool
	ApplicableCSTs []string
	LegalDevice    string
	Description    string
	Notes          string
	SourceName     string
	VersionLabel   string
}

func (i *CBenefImporter) ImportCSV(ctx context.Context, filePath string, sourceName string, versionLabel string, defaultUF string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open cbenef file: %w", err)
	}
	defer file.Close()

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read cbenef file bytes: %w", err)
	}

	content := normalizeCBenefContent(string(contentBytes))
	rows, err := readCBenefCSVRows(content)
	if err == nil {
		headerIndex, indexes, headerErr := findCBenefHeader(rows)
		if headerErr == nil {
			return i.importCBenefCSVRows(ctx, rows, headerIndex, indexes, sourceName, versionLabel, defaultUF, filePath)
		}
	}

	parsedRows, err := parseCBenefText(content, sourceName, versionLabel, defaultUF)
	if err != nil {
		return err
	}

	return i.importCBenefRows(ctx, parsedRows, sourceName, "text", versionLabel, filePath)
}

func readCBenefCSVRows(content string) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	return reader.ReadAll()
}

func (i *CBenefImporter) importCBenefCSVRows(ctx context.Context, rows [][]string, headerIndex int, indexes map[string]int, sourceName string, versionLabel string, defaultUF string, filePath string) error {
	parsedRows := make([]CBenefImportRow, 0, len(rows))
	for _, record := range rows[headerIndex+1:] {
		if !rowHasContent(record) {
			continue
		}
		row := parseCBenefCSVRow(record, indexes, sourceName, versionLabel, defaultUF)
		if row.Code != "" || row.Description != "" {
			parsedRows = append(parsedRows, row)
		}
	}

	return i.importCBenefRows(ctx, parsedRows, sourceName, "csv", versionLabel, filePath)
}

func (i *CBenefImporter) importCBenefRows(ctx context.Context, rows []CBenefImportRow, sourceName string, sourceType string, versionLabel string, filePath string) error {
	totalRows := 0
	for _, row := range rows {
		if row.Code != "" && row.Description != "" {
			totalRows++
		}
	}
	if totalRows == 0 {
		return fmt.Errorf("arquivo nao possui linhas validas de cBenef")
	}

	batchID, err := createImportBatch(ctx, i.db, sourceName, sourceType, versionLabel, filePath, totalRows)
	if err != nil {
		return err
	}

	successRows := 0
	failedRows := 0
	for lineNumber, row := range rows {
		if row.Code == "" || row.Description == "" || row.UF == "" {
			failedRows++
			continue
		}

		if err := i.upsertCBenef(ctx, row); err != nil {
			failedRows++
			fmt.Printf("line %d failed: %v\n", lineNumber+1, err)
			continue
		}
		successRows++
	}

	if err := finishImportBatch(ctx, i.db, batchID, successRows, failedRows); err != nil {
		return err
	}
	if successRows == 0 {
		return fmt.Errorf("import finished with zero successful cBenef rows; confira cabecalho, UF e codigos")
	}

	return nil
}

func findCBenefHeader(rows [][]string) (int, map[string]int, error) {
	for idx, row := range rows {
		indexes := mapCBenefHeaderIndexes(row)
		if _, ok := indexes["code"]; !ok {
			continue
		}
		if _, ok := indexes["description"]; !ok {
			continue
		}
		return idx, indexes, nil
	}

	return -1, nil, fmt.Errorf("cabecalho de cBenef nao encontrado")
}

func mapCBenefHeaderIndexes(header []string) map[string]int {
	out := make(map[string]int)
	for idx, name := range header {
		key := normalizeCBenefHeader(name)
		if key != "" {
			out[key] = idx
		}
	}
	return out
}

func normalizeCBenefHeader(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, "\u00a0", "")
	v = strings.NewReplacer(
		" ", "",
		"_", "",
		"-", "",
		"(", "",
		")", "",
		"/", "",
		"ó", "o",
		"ô", "o",
		"õ", "o",
		"ç", "c",
		"ã", "a",
		"á", "a",
		"à", "a",
		"é", "e",
		"ê", "e",
		"í", "i",
		"ú", "u",
		"Ã³", "o",
		"Ã§", "c",
		"Ã£", "a",
		"Ã¡", "a",
		"Ã©", "e",
	).Replace(v)

	switch {
	case v == "codigo" || v == "codigocbenef" || v == "cbenef":
		return "code"
	case v == "uf":
		return "uf"
	case strings.Contains(v, "simplesnacional"):
		return "simples"
	case strings.HasPrefix(v, "cst"):
		return "cst_" + normalizeNumericCode(strings.TrimPrefix(v, "cst"))
	case strings.Contains(v, "dispositivolegal") || strings.Contains(v, "dispositivo"):
		return "legal_device"
	case strings.Contains(v, "objetodescricao") || strings.Contains(v, "descricaonosdocumentos") || v == "descricao" || strings.Contains(v, "descricaoparanfae"):
		return "description"
	case strings.Contains(v, "observacao"):
		return "notes"
	default:
		return v
	}
}

func parseCBenefCSVRow(record []string, indexes map[string]int, sourceName string, versionLabel string, defaultUF string) CBenefImportRow {
	get := func(name string) string {
		idx, ok := indexes[name]
		if !ok || idx >= len(record) {
			return ""
		}
		return cleanText(record[idx])
	}

	code := normalizeCBenefCode(get("code"))
	description := firstNonEmpty(get("description"), get("legal_device"))
	uf := normalizeCBenefUF(firstNonEmpty(get("uf"), ufFromCBenefCode(code), defaultUF))
	csts := extractCBenefCSTs(indexes, record)

	return CBenefImportRow{
		UF:             uf,
		Code:           code,
		AppliesSimples: parseBoolIndicator(get("simples"), false),
		ApplicableCSTs: csts,
		LegalDevice:    get("legal_device"),
		Description:    description,
		Notes:          get("notes"),
		SourceName:     sourceName,
		VersionLabel:   versionLabel,
	}
}

func extractCBenefCSTs(indexes map[string]int, record []string) []string {
	out := make([]string, 0)
	for key, idx := range indexes {
		if !strings.HasPrefix(key, "cst_") || idx >= len(record) {
			continue
		}
		if !parseBoolIndicator(record[idx], false) {
			continue
		}
		code := strings.TrimPrefix(key, "cst_")
		if code != "" {
			out = append(out, code)
		}
	}
	return out
}

var cbenefCodePattern = regexp.MustCompile(`\b(?:[A-Z]{2}\d{6}|SEM\s+CBENEF|NULO|N[ÃA]O\s+PREENCHER|NAO\s+PREENCHER)\b`)

func normalizeCBenefContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.ReplaceAll(content, "\u00a0", " ")
	return html.UnescapeString(content)
}

func parseCBenefText(content string, sourceName string, versionLabel string, defaultUF string) ([]CBenefImportRow, error) {
	content = normalizeCBenefContent(content)
	content = strings.NewReplacer(
		"</td>", " | ",
		"</th>", " | ",
		"</tr>", "\n",
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
	).Replace(content)
	content = htmlTagPattern.ReplaceAllString(content, " ")

	rows := make([]CBenefImportRow, 0)
	for _, rawLine := range strings.Split(content, "\n") {
		line := cleanText(rawLine)
		if line == "" {
			continue
		}

		code := cbenefCodePattern.FindString(strings.ToUpper(line))
		if code == "" {
			continue
		}

		row := parseCBenefTextLine(line, code, sourceName, versionLabel, defaultUF)
		if row.Code != "" && row.Description != "" {
			rows = append(rows, row)
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("nao foi possivel identificar codigos cBenef no conteudo informado")
	}

	return rows, nil
}

func parseCBenefTextLine(line string, code string, sourceName string, versionLabel string, defaultUF string) CBenefImportRow {
	parts := splitCESTTableLine(line)
	code = normalizeCBenefCode(code)
	codeIndex := -1
	for idx, part := range parts {
		if normalizeCBenefCode(part) == code {
			codeIndex = idx
			break
		}
	}

	csts := make([]string, 0)
	legalDevice := ""
	description := line
	notes := ""

	if codeIndex >= 0 {
		for idx := codeIndex + 1; idx < len(parts); idx++ {
			part := parts[idx]
			if strings.EqualFold(part, "SIM") {
				cst := cbenefCSTFromPosition(idx - codeIndex)
				if cst != "" {
					csts = append(csts, cst)
				}
				continue
			}

			normalized := strings.ToLower(part)
			switch {
			case legalDevice == "" && (strings.Contains(normalized, "rcte") || strings.Contains(normalized, "art") || strings.Contains(normalized, "anexo")):
				legalDevice = part
			case description == line && part != code:
				description = part
			case notes == "":
				notes = part
			}
		}
	}

	return CBenefImportRow{
		UF:             normalizeCBenefUF(firstNonEmpty(ufFromCBenefCode(code), defaultUF)),
		Code:           code,
		AppliesSimples: false,
		ApplicableCSTs: csts,
		LegalDevice:    legalDevice,
		Description:    firstNonEmpty(description, line),
		Notes:          notes,
		SourceName:     sourceName,
		VersionLabel:   versionLabel,
	}
}

func cbenefCSTFromPosition(position int) string {
	csts := []string{"00", "10", "20", "30", "40", "41", "50", "51", "60", "70", "90"}
	if position <= 0 || position > len(csts) {
		return ""
	}
	return csts[position-1]
}

func normalizeCBenefCode(value string) string {
	value = cleanText(value)
	value = strings.ToUpper(value)
	value = strings.ReplaceAll(value, "NÃO", "NAO")
	switch value {
	case "", "NAO PREENCHER":
		return "NAO PREENCHER"
	case "NULO":
		return "NULO"
	default:
		return strings.Join(strings.Fields(value), " ")
	}
}

func normalizeCBenefUF(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) >= 2 {
		value = value[:2]
	}
	if len(value) != 2 {
		return ""
	}
	return value
}

func ufFromCBenefCode(code string) string {
	code = strings.TrimSpace(strings.ToUpper(code))
	if len(code) >= 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' {
		return code[:2]
	}
	return ""
}

func (i *CBenefImporter) upsertCBenef(ctx context.Context, row CBenefImportRow) error {
	cstsJSON, err := json.Marshal(row.ApplicableCSTs)
	if err != nil {
		return fmt.Errorf("marshal cbenef csts: %w", err)
	}

	_, err = i.db.Exec(ctx, `
		INSERT INTO cbenef_catalog (
			uf,
			code,
			applies_simples,
			applicable_csts,
			legal_device,
			description,
			notes,
			source_name,
			version_label,
			is_active,
			updated_at
		)
		VALUES ($1, $2, $3, $4::jsonb, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), TRUE, NOW())
		ON CONFLICT (uf, code) DO UPDATE
		SET applies_simples = EXCLUDED.applies_simples,
		    applicable_csts = EXCLUDED.applicable_csts,
		    legal_device = EXCLUDED.legal_device,
		    description = EXCLUDED.description,
		    notes = EXCLUDED.notes,
		    source_name = EXCLUDED.source_name,
		    version_label = EXCLUDED.version_label,
		    is_active = TRUE,
		    updated_at = NOW()
	`, row.UF, row.Code, row.AppliesSimples, string(cstsJSON), row.LegalDevice, row.Description, row.Notes, row.SourceName, row.VersionLabel)
	if err != nil {
		return fmt.Errorf("insert cbenef row: %w", err)
	}
	return nil
}
