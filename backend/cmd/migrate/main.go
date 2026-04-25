package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	databaseURL := flag.String("database", strings.TrimSpace(os.Getenv("DATABASE_URL")), "PostgreSQL connection string")
	migrationsDir := flag.String("dir", "migrations", "directory containing .up.sql files")
	baseline := flag.Bool("baseline", false, "mark current migration files as applied without executing them")
	flag.Parse()

	if strings.TrimSpace(*databaseURL) == "" {
		log.Fatal("DATABASE_URL is required; set it in .env or pass --database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := ensureSchemaMigrations(ctx, db); err != nil {
		log.Fatalf("prepare schema_migrations: %v", err)
	}

	files, err := migrationFiles(*migrationsDir)
	if err != nil {
		log.Fatalf("list migrations: %v", err)
	}

	if *baseline {
		if err := baselineMigrations(ctx, db, files); err != nil {
			log.Fatalf("baseline migrations: %v", err)
		}
		return
	}

	applied := 0
	for _, file := range files {
		version := filepath.Base(file)
		alreadyApplied, err := migrationApplied(ctx, db, version)
		if err != nil {
			log.Fatalf("check migration %s: %v", version, err)
		}
		if alreadyApplied {
			fmt.Printf("Pulando %s (ja aplicada)\n", version)
			continue
		}

		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read migration %s: %v", version, err)
		}

		fmt.Printf("Aplicando %s...\n", version)
		if err := applyMigration(ctx, db, version, string(sqlBytes)); err != nil {
			log.Fatalf("apply migration %s: %v", version, err)
		}
		applied++
	}

	fmt.Printf("Migrations concluidas. Novas aplicadas: %d\n", applied)
}

func ensureSchemaMigrations(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func migrationApplied(ctx context.Context, db *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
	return exists, err
}

func baselineMigrations(ctx context.Context, db *pgxpool.Pool, files []string) error {
	marked := 0
	for _, file := range files {
		version := filepath.Base(file)
		tag, err := db.Exec(ctx, `
			INSERT INTO schema_migrations (version)
			VALUES ($1)
			ON CONFLICT (version) DO NOTHING
		`, version)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			marked++
		}
	}

	fmt.Printf("Baseline concluido. Migrations marcadas: %d\n", marked)
	return nil
}

func applyMigration(ctx context.Context, db *pgxpool.Pool, version string, sql string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
