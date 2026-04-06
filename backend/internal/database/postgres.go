package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns       = 10
	defaultMinConns       = 2
	defaultMaxConnLifetime = 30 * time.Minute
	defaultMaxConnIdleTime = 5 * time.Minute
	defaultConnectTimeout  = 10 * time.Second
)

func NewPostgres(databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = defaultMaxConns
	cfg.MinConns = defaultMinConns
	cfg.MaxConnLifetime = defaultMaxConnLifetime
	cfg.MaxConnIdleTime = defaultMaxConnIdleTime

	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()

	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}