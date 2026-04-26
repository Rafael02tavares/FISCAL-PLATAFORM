package adminimports

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rafa/fiscal-platform/backend/internal/importers"
)

type Service struct {
	repo           *Repository
	ncmImporter    *importers.NCMImporter
	cfopImporter   *importers.CFOPImporter
	cestImporter   *importers.CESTImporter
	cbenefImporter *importers.CBenefImporter
	stImporter     *importers.StateICMSSTImporter
}

func NewService(repo *Repository, db *pgxpool.Pool) *Service {
	return &Service{
		repo:           repo,
		ncmImporter:    importers.NewNCMImporter(db),
		cfopImporter:   importers.NewCFOPImporter(db),
		cestImporter:   importers.NewCESTImporter(db),
		cbenefImporter: importers.NewCBenefImporter(db),
		stImporter:     importers.NewStateICMSSTImporter(db),
	}
}

func (s *Service) ListImportBatches(ctx context.Context, sourceName string, limit int) ([]ImportBatch, error) {
	if limit <= 0 {
		limit = 20
	}

	return s.repo.ListImportBatches(ctx, strings.TrimSpace(sourceName), limit)
}

type ImportNCMParams struct {
	File         multipart.File
	FileName     string
	SourceName   string
	VersionLabel string
}

func (s *Service) ImportNCMCSV(ctx context.Context, params ImportNCMParams) error {
	if params.File == nil {
		return fmt.Errorf("arquivo CSV obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "NCM CSV"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("ncm", params.FileName))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, params.File); err != nil {
		tmpFile.Close()
		return fmt.Errorf("copy uploaded file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.ncmImporter.ImportCSV(ctx, tmpPath, sourceName, strings.TrimSpace(params.VersionLabel))
}

type ImportCFOPParams struct {
	File         multipart.File
	FileName     string
	SourceName   string
	VersionLabel string
}

func (s *Service) ImportCFOPCSV(ctx context.Context, params ImportCFOPParams) error {
	if params.File == nil {
		return fmt.Errorf("arquivo CSV obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "CFOP CSV"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("cfop", params.FileName))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, params.File); err != nil {
		tmpFile.Close()
		return fmt.Errorf("copy uploaded file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.cfopImporter.ImportCSV(ctx, tmpPath, sourceName, strings.TrimSpace(params.VersionLabel))
}

type ImportCESTParams struct {
	File         multipart.File
	FileName     string
	SourceName   string
	VersionLabel string
}

type ImportCESTTextParams struct {
	Content      string
	SourceName   string
	VersionLabel string
}

type ImportCBenefParams struct {
	File         multipart.File
	FileName     string
	SourceName   string
	VersionLabel string
	UF           string
}

type ImportCBenefTextParams struct {
	Content      string
	SourceName   string
	VersionLabel string
	UF           string
}

type ImportStateICMSSTParams struct {
	File         multipart.File
	FileName     string
	SourceName   string
	VersionLabel string
	UF           string
	SourceURL    string
}

func (s *Service) ImportCESTCSV(ctx context.Context, params ImportCESTParams) error {
	if params.File == nil {
		return fmt.Errorf("arquivo CSV obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "CEST CSV"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("cest", params.FileName))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, params.File); err != nil {
		tmpFile.Close()
		return fmt.Errorf("copy uploaded file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.cestImporter.ImportCSV(ctx, tmpPath, sourceName, strings.TrimSpace(params.VersionLabel))
}

func (s *Service) ImportCESTText(ctx context.Context, params ImportCESTTextParams) error {
	content := strings.TrimSpace(params.Content)
	if content == "" {
		return fmt.Errorf("conteudo da tabela CEST obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "CEST CONFAZ"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("cest-text", "confaz-cest.txt"))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.cestImporter.ImportCSV(ctx, tmpPath, sourceName, strings.TrimSpace(params.VersionLabel))
}

func (s *Service) ImportCBenefCSV(ctx context.Context, params ImportCBenefParams) error {
	if params.File == nil {
		return fmt.Errorf("arquivo CSV obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "cBenef CSV"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("cbenef", params.FileName))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, params.File); err != nil {
		tmpFile.Close()
		return fmt.Errorf("copy uploaded file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.cbenefImporter.ImportCSV(ctx, tmpPath, sourceName, strings.TrimSpace(params.VersionLabel), strings.TrimSpace(params.UF))
}

func (s *Service) ImportCBenefText(ctx context.Context, params ImportCBenefTextParams) error {
	content := strings.TrimSpace(params.Content)
	if content == "" {
		return fmt.Errorf("conteudo da tabela cBenef obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "cBenef"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("cbenef-text", "cbenef.txt"))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.cbenefImporter.ImportCSV(ctx, tmpPath, sourceName, strings.TrimSpace(params.VersionLabel), strings.TrimSpace(params.UF))
}

func (s *Service) ImportStateICMSSTXLSX(ctx context.Context, params ImportStateICMSSTParams) error {
	if params.File == nil {
		return fmt.Errorf("arquivo XLSX obrigatorio")
	}

	sourceName := strings.TrimSpace(params.SourceName)
	if sourceName == "" {
		sourceName = "CONFAZ ST Estadual"
	}

	tmpFile, err := os.CreateTemp("", importersPattern("state-icms-st", params.FileName))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, params.File); err != nil {
		tmpFile.Close()
		return fmt.Errorf("copy uploaded file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return s.stImporter.ImportCONFAZXLSX(
		ctx,
		tmpPath,
		sourceName,
		strings.TrimSpace(params.VersionLabel),
		strings.TrimSpace(params.UF),
		strings.TrimSpace(params.SourceURL),
	)
}

func importersPattern(prefix string, fileName string) string {
	base := filepath.Base(strings.TrimSpace(fileName))
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
		base = "ncm-upload"
	}

	if len(base) > 40 {
		base = base[:40]
	}

	return prefix + "-" + base + "-" + strconv.FormatInt(int64(os.Getpid()), 10) + "-*.csv"
}
