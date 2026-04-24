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
	repo         *Repository
	ncmImporter  *importers.NCMImporter
	cfopImporter *importers.CFOPImporter
}

func NewService(repo *Repository, db *pgxpool.Pool) *Service {
	return &Service{
		repo:         repo,
		ncmImporter:  importers.NewNCMImporter(db),
		cfopImporter: importers.NewCFOPImporter(db),
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
