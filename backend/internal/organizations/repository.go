package organizations

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

type Organization struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	CNPJ              string `json:"cnpj"`
	Role              string `json:"role,omitempty"`
	TaxRegime         string `json:"tax_regime,omitempty"`
	CRT               string `json:"crt,omitempty"`
	StateRegistration string `json:"state_registration,omitempty"`
	HomeUF            string `json:"home_uf,omitempty"`
}

func (r *Repository) CreateOrganization(
	ctx context.Context,
	name, cnpj, taxRegime, crt, stateRegistration, homeUF string,
) (string, error) {
	query := `
		INSERT INTO organizations (
			name,
			cnpj,
			tax_regime,
			crt,
			state_registration,
			home_uf
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var organizationID string
	err := r.db.QueryRow(
		ctx,
		query,
		name,
		cnpj,
		taxRegime,
		crt,
		stateRegistration,
		homeUF,
	).Scan(&organizationID)
	if err != nil {
		return "", fmt.Errorf("create organization: %w", err)
	}

	return organizationID, nil
}

func (r *Repository) AddUserToOrganization(ctx context.Context, userID, organizationID, role string) error {
	query := `
		INSERT INTO organization_users (user_id, organization_id, role)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, userID, organizationID, role)
	if err != nil {
		return fmt.Errorf("add user to organization: %w", err)
	}

	return nil
}

func (r *Repository) ListOrganizationsByUser(ctx context.Context, userID string) ([]Organization, error) {
	query := `
		SELECT
			o.id,
			COALESCE(o.name, ''),
			COALESCE(o.cnpj, ''),
			COALESCE(ou.role, ''),
			COALESCE(o.tax_regime, ''),
			COALESCE(o.crt, ''),
			COALESCE(o.state_registration, ''),
			COALESCE(o.home_uf, '')
		FROM organizations o
		INNER JOIN organization_users ou ON ou.organization_id = o.id
		WHERE ou.user_id = $1
		ORDER BY o.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list organizations by user: %w", err)
	}
	defer rows.Close()

	var organizations []Organization

	for rows.Next() {
		var org Organization
		if err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.CNPJ,
			&org.Role,
			&org.TaxRegime,
			&org.CRT,
			&org.StateRegistration,
			&org.HomeUF,
		); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		organizations = append(organizations, org)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organizations: %w", err)
	}

	return organizations, nil
}

func (r *Repository) UserBelongsToOrganization(ctx context.Context, userID, organizationID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM organization_users
			WHERE user_id = $1
			  AND organization_id = $2
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID, organizationID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user organization membership: %w", err)
	}

	return exists, nil
}