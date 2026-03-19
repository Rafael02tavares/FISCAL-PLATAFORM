package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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
	VALUES ($1,$2,$3)
	`

	_, err := r.db.Exec(ctx, query, name, email, passwordHash)
	return err
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (string, string, string, error) {
	query := `
	SELECT id, email, password_hash
	FROM users
	WHERE email=$1
	`

	var id string
	var mail string
	var hash string

	err := r.db.QueryRow(ctx, query, email).Scan(&id, &mail, &hash)

	return id, mail, hash, err
}
