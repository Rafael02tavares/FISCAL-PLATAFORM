package cest

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type CEST struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	NCMCode     string `json:"ncm_code"`
	Segment     string `json:"segment"`
	Description string `json:"description"`
	LegalSource string `json:"legal_source"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	IsActive    bool   `json:"is_active"`
}

func (r *Repository) List(ctx context.Context, q string, ncmCode string, limit int) ([]CEST, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(ncm_code, ''),
			COALESCE(segment, ''),
			COALESCE(description, ''),
			COALESCE(legal_source, ''),
			COALESCE(start_date::text, ''),
			COALESCE(end_date::text, ''),
			is_active
		FROM cest_catalog
		WHERE is_active = TRUE
		  AND ($1 = '' OR code ILIKE $1 OR ncm_code ILIKE $1 OR segment ILIKE $1 OR description ILIKE $1)
		  AND ($2 = '' OR ncm_code = $2)
		ORDER BY segment NULLS LAST, code, ncm_code
		LIMIT $3
	`, "%"+strings.TrimSpace(q)+"%", strings.TrimSpace(ncmCode), limit)
	if err != nil {
		return nil, fmt.Errorf("list cest catalog: %w", err)
	}
	defer rows.Close()

	var items []CEST
	for rows.Next() {
		var item CEST
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.NCMCode,
			&item.Segment,
			&item.Description,
			&item.LegalSource,
			&item.StartDate,
			&item.EndDate,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan cest: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cest rows: %w", err)
	}

	return items, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (*CEST, error) {
	var item CEST
	err := r.db.QueryRow(ctx, `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(ncm_code, ''),
			COALESCE(segment, ''),
			COALESCE(description, ''),
			COALESCE(legal_source, ''),
			COALESCE(start_date::text, ''),
			COALESCE(end_date::text, ''),
			is_active
		FROM cest_catalog
		WHERE code = $1 AND is_active = TRUE
		ORDER BY updated_at DESC
		LIMIT 1
	`, strings.TrimSpace(code)).Scan(
		&item.ID,
		&item.Code,
		&item.NCMCode,
		&item.Segment,
		&item.Description,
		&item.LegalSource,
		&item.StartDate,
		&item.EndDate,
		&item.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("find cest by code: %w", err)
	}

	return &item, nil
}
