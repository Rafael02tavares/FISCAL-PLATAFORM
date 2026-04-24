package cfop

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

type CFOP struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Description      string `json:"description"`
	OperationType    string `json:"operation_type"`
	IndNFe           bool   `json:"ind_nfe"`
	IndCommunication bool   `json:"ind_communication"`
	IndTransport     bool   `json:"ind_transport"`
	IndDevolution    bool   `json:"ind_devolution"`
	CreatedAt        string `json:"created_at"`
}

func (r *Repository) List(ctx context.Context, q string, operationType string, limit int) ([]CFOP, error) {
	query := `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(operation_type, ''),
			COALESCE(ind_nfe, FALSE),
			COALESCE(ind_comunication, FALSE),
			COALESCE(ind_transport, FALSE),
			COALESCE(ind_devolution, FALSE),
			COALESCE(created_at::text, '')
		FROM cfop_catalog
		WHERE ($1 = '' OR code ILIKE $1 OR description ILIKE $1)
		  AND ($2 = '' OR operation_type = $2)
		ORDER BY code
		LIMIT $3
	`

	like := "%" + strings.TrimSpace(q) + "%"

	rows, err := r.db.Query(ctx, query, like, strings.TrimSpace(operationType), limit)
	if err != nil {
		return nil, fmt.Errorf("list cfop catalog: %w", err)
	}
	defer rows.Close()

	var items []CFOP
	for rows.Next() {
		var item CFOP
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Description,
			&item.OperationType,
			&item.IndNFe,
			&item.IndCommunication,
			&item.IndTransport,
			&item.IndDevolution,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cfop: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cfop rows: %w", err)
	}

	return items, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (*CFOP, error) {
	query := `
		SELECT
			id,
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(operation_type, ''),
			COALESCE(ind_nfe, FALSE),
			COALESCE(ind_comunication, FALSE),
			COALESCE(ind_transport, FALSE),
			COALESCE(ind_devolution, FALSE),
			COALESCE(created_at::text, '')
		FROM cfop_catalog
		WHERE code = $1
		LIMIT 1
	`

	var item CFOP
	err := r.db.QueryRow(ctx, query, code).Scan(
		&item.ID,
		&item.Code,
		&item.Description,
		&item.OperationType,
		&item.IndNFe,
		&item.IndCommunication,
		&item.IndTransport,
		&item.IndDevolution,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find cfop by code: %w", err)
	}

	return &item, nil
}
