package adminimports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type ImportBatch struct {
	ID           string `json:"id"`
	SourceName   string `json:"source_name"`
	SourceType   string `json:"source_type"`
	VersionLabel string `json:"version_label"`
	FileName     string `json:"file_name"`
	ImportedAt   string `json:"imported_at"`
	TotalRows    int    `json:"total_rows"`
	SuccessRows  int    `json:"success_rows"`
	FailedRows   int    `json:"failed_rows"`
	Notes        string `json:"notes"`
}

func (r *Repository) ListImportBatches(ctx context.Context, sourceName string, limit int) ([]ImportBatch, error) {
	query := `
		SELECT
			id,
			COALESCE(source_name, ''),
			COALESCE(source_type, ''),
			COALESCE(version_label, ''),
			COALESCE(file_name, ''),
			COALESCE(imported_at::text, ''),
			total_rows,
			success_rows,
			failed_rows,
			COALESCE(notes, '')
		FROM import_batches
		WHERE ($1 = '' OR source_name = $1)
		ORDER BY imported_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, sourceName, limit)
	if err != nil {
		return nil, fmt.Errorf("list import batches: %w", err)
	}
	defer rows.Close()

	var items []ImportBatch
	for rows.Next() {
		var item ImportBatch
		if err := rows.Scan(
			&item.ID,
			&item.SourceName,
			&item.SourceType,
			&item.VersionLabel,
			&item.FileName,
			&item.ImportedAt,
			&item.TotalRows,
			&item.SuccessRows,
			&item.FailedRows,
			&item.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan import batch: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import batches: %w", err)
	}

	return items, nil
}
