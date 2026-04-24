package adminusers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUserAlreadyInOrg      = errors.New("user already belongs to organization")
	ErrUserMembershipMissing = errors.New("user membership not found")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type OrganizationUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UserRecord struct {
	ID    string
	Name  string
	Email string
}

func (r *Repository) ListByOrganization(ctx context.Context, organizationID string) ([]OrganizationUser, error) {
	query := `
		SELECT
			u.id::text,
			COALESCE(u.name, ''),
			COALESCE(u.email, ''),
			COALESCE(ou.role, '')
		FROM organization_users ou
		INNER JOIN users u ON u.id = ou.user_id
		WHERE ou.organization_id = $1
		ORDER BY u.name ASC, u.email ASC
	`

	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list organization users: %w", err)
	}
	defer rows.Close()

	items := make([]OrganizationUser, 0)
	for rows.Next() {
		var item OrganizationUser
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Role); err != nil {
			return nil, fmt.Errorf("scan organization user: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organization users: %w", err)
	}

	return items, nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*UserRecord, error) {
	query := `
		SELECT id::text, COALESCE(name, ''), COALESCE(email, '')
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var item UserRecord
	err := r.db.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(email))).Scan(
		&item.ID,
		&item.Name,
		&item.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &item, nil
}

func (r *Repository) CreateUser(ctx context.Context, name, email, passwordHash string) (string, error) {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`

	var id string
	err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(name),
		strings.ToLower(strings.TrimSpace(email)),
		passwordHash,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return "", ErrEmailAlreadyExists
		}
		return "", fmt.Errorf("create user: %w", err)
	}

	return id, nil
}

func (r *Repository) AddUserToOrganization(ctx context.Context, userID, organizationID, role string) error {
	query := `
		INSERT INTO organization_users (user_id, organization_id, role)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, userID, organizationID, role)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyInOrg
		}
		return fmt.Errorf("add user to organization: %w", err)
	}

	return nil
}

func (r *Repository) UpdateRole(ctx context.Context, userID, organizationID, role string) error {
	query := `
		UPDATE organization_users
		SET role = $3
		WHERE user_id = $1
		  AND organization_id = $2
	`

	tag, err := r.db.Exec(ctx, query, userID, organizationID, role)
	if err != nil {
		return fmt.Errorf("update organization user role: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUserMembershipMissing
	}

	return nil
}

func (r *Repository) RemoveFromOrganization(ctx context.Context, userID, organizationID string) error {
	query := `
		DELETE FROM organization_users
		WHERE user_id = $1
		  AND organization_id = $2
	`

	tag, err := r.db.Exec(ctx, query, userID, organizationID)
	if err != nil {
		return fmt.Errorf("remove user from organization: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUserMembershipMissing
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505"
}
