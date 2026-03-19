package ncm

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

type NCM struct {
	ID              string `json:"id"`
	Code            string `json:"code"`
	Description     string `json:"description"`
	FullDescription string `json:"full_description"`
	ChapterCode     string `json:"chapter_code"`
	HeadingCode     string `json:"heading_code"`
	ItemCode        string `json:"item_code"`
	ParentCode      string `json:"parent_code"`
	LevelType       string `json:"level_type"`
	ExCode          string `json:"ex_code"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	LegalSource     string `json:"legal_source"`
	LegalReference  string `json:"legal_reference"`
	OfficialNotes   string `json:"official_notes"`
	IsActive        bool   `json:"is_active"`
}

func (r *Repository) List(ctx context.Context, limit int) ([]NCM, error) {
	query := `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(full_description, ''),
			COALESCE(chapter_code, ''),
			COALESCE(heading_code, ''),
			COALESCE(item_code, ''),
			COALESCE(parent_code, ''),
			COALESCE(level_type, ''),
			COALESCE(ex_code, ''),
			COALESCE(start_date::text, ''),
			COALESCE(end_date::text, ''),
			COALESCE(legal_source, ''),
			COALESCE(legal_reference, ''),
			COALESCE(official_notes, ''),
			is_active
		FROM ncm_catalog
		WHERE is_active = TRUE
		ORDER BY code
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list ncm catalog: %w", err)
	}
	defer rows.Close()

	var items []NCM

	for rows.Next() {
		var item NCM
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Description,
			&item.FullDescription,
			&item.ChapterCode,
			&item.HeadingCode,
			&item.ItemCode,
			&item.ParentCode,
			&item.LevelType,
			&item.ExCode,
			&item.StartDate,
			&item.EndDate,
			&item.LegalSource,
			&item.LegalReference,
			&item.OfficialNotes,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan ncm: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ncm rows: %w", err)
	}

	return items, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (*NCM, error) {
	query := `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(full_description, ''),
			COALESCE(chapter_code, ''),
			COALESCE(heading_code, ''),
			COALESCE(item_code, ''),
			COALESCE(parent_code, ''),
			COALESCE(level_type, ''),
			COALESCE(ex_code, ''),
			COALESCE(start_date::text, ''),
			COALESCE(end_date::text, ''),
			COALESCE(legal_source, ''),
			COALESCE(legal_reference, ''),
			COALESCE(official_notes, ''),
			is_active
		FROM ncm_catalog
		WHERE code = $1
		  AND is_active = TRUE
		ORDER BY start_date DESC NULLS LAST, created_at DESC
		LIMIT 1
	`

	var item NCM
	err := r.db.QueryRow(ctx, query, code).Scan(
		&item.ID,
		&item.Code,
		&item.Description,
		&item.FullDescription,
		&item.ChapterCode,
		&item.HeadingCode,
		&item.ItemCode,
		&item.ParentCode,
		&item.LevelType,
		&item.ExCode,
		&item.StartDate,
		&item.EndDate,
		&item.LegalSource,
		&item.LegalReference,
		&item.OfficialNotes,
		&item.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("find ncm by code: %w", err)
	}

	return &item, nil
}

func (r *Repository) Search(ctx context.Context, q string, limit int) ([]NCM, error) {
	query := `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(full_description, ''),
			COALESCE(chapter_code, ''),
			COALESCE(heading_code, ''),
			COALESCE(item_code, ''),
			COALESCE(parent_code, ''),
			COALESCE(level_type, ''),
			COALESCE(ex_code, ''),
			COALESCE(start_date::text, ''),
			COALESCE(end_date::text, ''),
			COALESCE(legal_source, ''),
			COALESCE(legal_reference, ''),
			COALESCE(official_notes, ''),
			is_active
		FROM ncm_catalog
		WHERE is_active = TRUE
		  AND (
			code ILIKE $1
			OR description ILIKE $1
			OR full_description ILIKE $1
		  )
		ORDER BY code
		LIMIT $2
	`

	like := "%" + strings.TrimSpace(q) + "%"

	rows, err := r.db.Query(ctx, query, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search ncm catalog: %w", err)
	}
	defer rows.Close()

	var items []NCM

	for rows.Next() {
		var item NCM
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Description,
			&item.FullDescription,
			&item.ChapterCode,
			&item.HeadingCode,
			&item.ItemCode,
			&item.ParentCode,
			&item.LevelType,
			&item.ExCode,
			&item.StartDate,
			&item.EndDate,
			&item.LegalSource,
			&item.LegalReference,
			&item.OfficialNotes,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan searched ncm: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate searched ncm rows: %w", err)
	}

	return items, nil
}