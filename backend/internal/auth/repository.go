package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, name, email, passwordHash string) error {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		strings.TrimSpace(name),
		strings.ToLower(strings.TrimSpace(email)),
		passwordHash,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailAlreadyExists
		}

		return err
	}

	return nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (string, string, string, error) {
	query := `
		SELECT id, email, password_hash
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var id string
	var userEmail string
	var hash string

	err := r.db.QueryRow(
		ctx,
		query,
		strings.ToLower(strings.TrimSpace(email)),
	).Scan(&id, &userEmail, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", ErrUserNotFound
		}

		return "", "", "", err
	}

	return id, userEmail, hash, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505"
}