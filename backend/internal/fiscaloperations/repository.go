package fiscaloperations

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

type FiscalOperation struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Direction   string `json:"direction"`
	DefaultCFOP string `json:"default_cfop"`
	IsDefault   bool   `json:"is_default"`
	Active      bool   `json:"active"`
}

func (r *Repository) ListActive(ctx context.Context) ([]FiscalOperation, error) {
	query := `
		SELECT
			id,
			code,
			name,
			direction,
			default_cfop,
			is_default,
			active
		FROM fiscal_operations
		WHERE active = TRUE
		ORDER BY is_default DESC, name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list fiscal operations: %w", err)
	}
	defer rows.Close()

	var items []FiscalOperation

	for rows.Next() {
		var item FiscalOperation
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Name,
			&item.Direction,
			&item.DefaultCFOP,
			&item.IsDefault,
			&item.Active,
		); err != nil {
			return nil, fmt.Errorf("scan fiscal operation: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fiscal operations: %w", err)
	}

	return items, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (*FiscalOperation, error) {
	query := `
		SELECT
			id,
			code,
			name,
			direction,
			default_cfop,
			is_default,
			active
		FROM fiscal_operations
		WHERE code = $1
		LIMIT 1
	`

	var item FiscalOperation
	err := r.db.QueryRow(ctx, query, code).Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.Direction,
		&item.DefaultCFOP,
		&item.IsDefault,
		&item.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("find fiscal operation by code: %w", err)
	}

	return &item, nil
}

func (r *Repository) FindDefault(ctx context.Context) (*FiscalOperation, error) {
	query := `
		SELECT
			id,
			code,
			name,
			direction,
			default_cfop,
			is_default,
			active
		FROM fiscal_operations
		WHERE is_default = TRUE
		LIMIT 1
	`

	var item FiscalOperation
	err := r.db.QueryRow(ctx, query).Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.Direction,
		&item.DefaultCFOP,
		&item.IsDefault,
		&item.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("find default fiscal operation: %w", err)
	}

	return &item, nil
}