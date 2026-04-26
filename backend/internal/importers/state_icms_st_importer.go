package importers

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StateICMSSTImporter struct {
	db *pgxpool.Pool
}

func NewStateICMSSTImporter(db *pgxpool.Pool) *StateICMSSTImporter {
	return &StateICMSSTImporter{db: db}
}

type StateICMSSTImportRow struct {
	UF              string
	CEST            string
	Description     string
	Segment         string
	InternalRate    string
	FCPRate         string
	MVARate         string
	SourceReference string
	SourceURL       string
	ValidFrom       string
}

type xlsxCell struct {
	Col   string
	Index int
	Value string
}

func (i *StateICMSSTImporter) ImportCONFAZXLSX(ctx context.Context, filePath string, sourceName string, versionLabel string, uf string, sourceURL string) error {
	sourceName = firstNonEmpty(strings.TrimSpace(sourceName), "CONFAZ Portal Nacional ST")
	uf = strings.ToUpper(strings.TrimSpace(uf))
	if uf == "" {
		uf = "GO"
	}

	rows, err := parseCONFAZStateSTWorkbook(filePath, uf, sourceName, strings.TrimSpace(sourceURL))
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("planilha nao possui regras ST com Op. Interna = S")
	}

	batchID, err := createImportBatch(ctx, i.db, sourceName, "xlsx_state_icms_st", versionLabel, filePath, len(rows))
	if err != nil {
		return err
	}

	successRows := 0
	failedRows := 0
	for lineNumber, row := range rows {
		if row.CEST == "" {
			failedRows++
			continue
		}
		if err := i.upsertStateICMSSTRule(ctx, row); err != nil {
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
		return fmt.Errorf("importacao finalizada sem linhas gravadas")
	}

	return nil
}

func parseCONFAZStateSTWorkbook(filePath string, uf string, sourceName string, sourceURL string) ([]StateICMSSTImportRow, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer reader.Close()

	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}

	sharedStrings, err := readXLSXSharedStrings(files)
	if err != nil {
		return nil, err
	}

	worksheetNames := make([]string, 0)
	for name := range files {
		if strings.HasPrefix(name, "xl/worksheets/sheet") && strings.HasSuffix(name, ".xml") {
			worksheetNames = append(worksheetNames, name)
		}
	}
	sort.Strings(worksheetNames)

	out := make([]StateICMSSTImportRow, 0, 256)
	for _, name := range worksheetNames {
		content, err := readZipFile(files[name])
		if err != nil {
			return nil, err
		}
		out = append(out, parseCONFAZStateSTSheet(content, sharedStrings, uf, sourceName, sourceURL)...)
	}

	return out, nil
}

func parseCONFAZStateSTSheet(content []byte, sharedStrings []string, uf string, sourceName string, sourceURL string) []StateICMSSTImportRow {
	rows := extractXLSXRows(content, sharedStrings)
	out := make([]StateICMSSTImportRow, 0)
	header := map[string]int{}
	segment := ""
	validFrom := ""

	for _, row := range rows {
		values := rowValues(row)
		line := cleanText(strings.Join(values, " "))
		if line == "" {
			continue
		}
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "segmento:") {
			segment = cleanText(line)
		}
		if strings.Contains(lowerLine, "produção de efeitos") || strings.Contains(lowerLine, "producao de efeitos") {
			validFrom = parseBrazilianDate(line)
		}

		mapped := mapCONFAZHeader(row)
		if _, ok := mapped["cest"]; ok {
			if _, ok := mapped["description"]; ok {
				header = mapped
				continue
			}
		}
		if len(header) == 0 {
			continue
		}

		get := func(key string) string {
			idx, ok := header[key]
			if !ok || idx >= len(row) {
				return ""
			}
			return cleanText(row[idx].Value)
		}

		if !isPositiveSTOperation(get("internal_operation")) {
			continue
		}

		cest := normalizeNumericCode(get("cest"))
		description := get("description")
		if cest == "" || description == "" {
			continue
		}

		out = append(out, StateICMSSTImportRow{
			UF:              uf,
			CEST:            cest,
			Description:     description,
			Segment:         segment,
			InternalRate:    parsePercentRate(firstNonEmpty(get("internal_rate"), get("internal_rate_spec"))),
			FCPRate:         parseFCPRate(get("internal_rate_spec")),
			MVARate:         parsePercentRate(firstNonEmpty(get("mva_rate"), get("mva_rate_2"))),
			SourceReference: sourceName,
			SourceURL:       sourceURL,
			ValidFrom:       firstNonEmpty(validFrom, "2023-07-10"),
		})
	}

	return out
}

func mapCONFAZHeader(row []xlsxCell) map[string]int {
	out := map[string]int{}
	for idx, cell := range row {
		key := normalizeCONFAZHeader(cell.Value)
		if key != "" {
			out[key] = idx
		}
	}
	return out
}

func normalizeCONFAZHeader(value string) string {
	value = strings.ToLower(cleanText(value))
	value = strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u",
		"ç", "c",
		".", "", " ", "", "-", "", "_", "",
	).Replace(value)

	switch {
	case value == "cest":
		return "cest"
	case value == "descricao" || strings.Contains(value, "descricao"):
		return "description"
	case strings.Contains(value, "opinterna"):
		return "internal_operation"
	case strings.Contains(value, "especificacaomvast1") || strings.Contains(value, "especificacaomvast"):
		return "mva_spec"
	case value == "mvast" || value == "mvast1":
		return "mva_rate"
	case value == "mvast2":
		return "mva_rate_2"
	case strings.Contains(value, "especificacaoaliqinterna"):
		return "internal_rate_spec"
	case strings.Contains(value, "aliqinterna"):
		return "internal_rate"
	default:
		return ""
	}
}

func (i *StateICMSSTImporter) upsertStateICMSSTRule(ctx context.Context, row StateICMSSTImportRow) error {
	query := `
		INSERT INTO state_icms_rules (
			uf,
			ncm_pattern,
			match_type,
			cest,
			operation_code,
			rule_kind,
			cfop,
			icms_cst,
			csosn,
			icms_rate,
			fcp_rate,
			icms_st_rate,
			confidence_score,
			source_reference,
			source_url,
			notes,
			valid_from
		)
		VALUES (
			$1,
			'',
			'prefix',
			$2,
			'sale_consumer_final',
			'ST',
			'5405',
			'60',
			'500',
			0,
			COALESCE(NULLIF($3, ''), '0')::numeric,
			NULLIF($4, '')::numeric,
			0.9100,
			$5,
			$6,
			$7,
			NULLIF($8, '')::date
		)
		ON CONFLICT (
			uf,
			ncm_pattern,
			match_type,
			cest,
			operation_code,
			tax_regime,
			target_crt,
			rule_kind,
			valid_from
		) DO UPDATE
		SET
			cfop = EXCLUDED.cfop,
			icms_cst = EXCLUDED.icms_cst,
			csosn = EXCLUDED.csosn,
			icms_rate = EXCLUDED.icms_rate,
			fcp_rate = EXCLUDED.fcp_rate,
			icms_st_rate = EXCLUDED.icms_st_rate,
			confidence_score = EXCLUDED.confidence_score,
			source_reference = EXCLUDED.source_reference,
			source_url = EXCLUDED.source_url,
			notes = EXCLUDED.notes,
			updated_at = NOW()
	`

	notes := cleanText(strings.Join([]string{
		row.Segment,
		row.Description,
		"MVA-ST: " + firstNonEmpty(row.MVARate, "-"),
	}, " | "))

	_, err := i.db.Exec(
		ctx,
		query,
		row.UF,
		row.CEST,
		row.FCPRate,
		row.InternalRate,
		row.SourceReference,
		row.SourceURL,
		notes,
		row.ValidFrom,
	)
	if err != nil {
		return fmt.Errorf("upsert state icms st rule: %w", err)
	}
	return nil
}

func readXLSXSharedStrings(files map[string]*zip.File) ([]string, error) {
	file := files["xl/sharedStrings.xml"]
	if file == nil {
		return []string{}, nil
	}
	content, err := readZipFile(file)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	values := make([]string, 0)
	inText := false
	var current strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse shared strings: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				current.Reset()
			}
			if t.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
			if t.Name.Local == "si" {
				values = append(values, html.UnescapeString(current.String()))
			}
		case xml.CharData:
			if inText {
				current.Write([]byte(t))
			}
		}
	}
	return values, nil
}

func extractXLSXRows(content []byte, sharedStrings []string) [][]xlsxCell {
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	rows := make([][]xlsxCell, 0)
	var current []xlsxCell
	var currentCell xlsxCell
	cellType := ""
	inValue := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				current = []xlsxCell{}
			case "c":
				currentCell = xlsxCell{}
				cellType = ""
				for _, attr := range t.Attr {
					switch attr.Name.Local {
					case "r":
						currentCell.Col = cellColumn(attr.Value)
						currentCell.Index = columnIndex(currentCell.Col)
					case "t":
						cellType = attr.Value
					}
				}
			case "v", "t":
				inValue = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "c":
				current = append(current, currentCell)
			case "row":
				sort.Slice(current, func(i, j int) bool { return current[i].Index < current[j].Index })
				rows = append(rows, current)
			case "v", "t":
				inValue = false
			}
		case xml.CharData:
			if inValue {
				raw := string([]byte(t))
				if cellType == "s" {
					if idx, err := strconv.Atoi(raw); err == nil && idx >= 0 && idx < len(sharedStrings) {
						currentCell.Value += sharedStrings[idx]
					}
				} else {
					currentCell.Value += raw
				}
			}
		}
	}
	return rows
}

func rowValues(row []xlsxCell) []string {
	out := make([]string, 0, len(row))
	for _, cell := range row {
		out = append(out, cell.Value)
	}
	return out
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func cellColumn(ref string) string {
	var b strings.Builder
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func columnIndex(col string) int {
	total := 0
	for _, r := range col {
		total = total*26 + int(r-'A'+1)
	}
	return total
}

func isPositiveSTOperation(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	return value == "S" || value == "SIM"
}

func parsePercentRate(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	if value == "" || value == "-" {
		return ""
	}
	value = strings.TrimSuffix(value, "%")
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		if parsed > 0 && parsed < 1 {
			parsed = parsed * 100
		}
		return fmt.Sprintf("%.4f", parsed)
	}
	return ""
}

var fcpPattern = regexp.MustCompile(`(?i)(\d+(?:[,.]\d+)?)\s*%\s*\(?\s*FCP`)

func parseFCPRate(value string) string {
	matches := fcpPattern.FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return parsePercentRate(matches[1])
}

var brDatePattern = regexp.MustCompile(`\b(\d{2})/(\d{2})/(\d{4})\b`)

func parseBrazilianDate(value string) string {
	matches := brDatePattern.FindStringSubmatch(value)
	if len(matches) != 4 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s", matches[3], matches[2], matches[1])
}

func importersPattern(prefix string, fileName string) string {
	return buildImportTempPattern(prefix, filepath.Base(fileName))
}
