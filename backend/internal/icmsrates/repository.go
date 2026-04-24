package icmsrates

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type StateRate struct {
	ID              string `json:"id"`
	UF              string `json:"uf"`
	InternalRate    string `json:"internal_rate"`
	FCPRate         string `json:"fcp_rate"`
	ValidFrom       string `json:"valid_from"`
	ValidTo         string `json:"valid_to"`
	SourceReference string `json:"source_reference"`
	SourceURL       string `json:"source_url"`
	Notes           string `json:"notes"`
}

type UpsertStateRateParams struct {
	UF              string
	InternalRate    string
	FCPRate         string
	ValidFrom       string
	ValidTo         string
	SourceReference string
	SourceURL       string
	Notes           string
}

func (r *Repository) ListStateRates(ctx context.Context) ([]StateRate, error) {
	if ok, err := r.tableExists(ctx); err != nil {
		return nil, err
	} else if !ok {
		return []StateRate{}, nil
	}

	query := `
		SELECT
			id::text,
			uf,
			TRIM(TO_CHAR(internal_rate, 'FM999990.00')),
			TRIM(TO_CHAR(fcp_rate, 'FM999990.00')),
			TO_CHAR(valid_from, 'YYYY-MM-DD'),
			COALESCE(TO_CHAR(valid_to, 'YYYY-MM-DD'), ''),
			COALESCE(source_reference, ''),
			COALESCE(source_url, ''),
			COALESCE(notes, '')
		FROM icms_state_rates
		ORDER BY uf ASC, valid_from DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list icms state rates: %w", err)
	}
	defer rows.Close()

	items := make([]StateRate, 0, 32)
	for rows.Next() {
		var item StateRate
		if err := rows.Scan(
			&item.ID,
			&item.UF,
			&item.InternalRate,
			&item.FCPRate,
			&item.ValidFrom,
			&item.ValidTo,
			&item.SourceReference,
			&item.SourceURL,
			&item.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan icms state rate: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate icms state rates: %w", err)
	}

	return items, nil
}

func (r *Repository) UpsertStateRate(ctx context.Context, p UpsertStateRateParams) (string, error) {
	if ok, err := r.tableExists(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", errors.New("icms_state_rates table not found; run the latest migrations")
	}

	query := `
		INSERT INTO icms_state_rates (
			uf,
			internal_rate,
			fcp_rate,
			valid_from,
			valid_to,
			source_reference,
			source_url,
			notes,
			updated_at
		)
		VALUES (
			$1,
			$2::numeric,
			COALESCE(NULLIF($3, ''), '0')::numeric,
			$4::date,
			NULLIF($5, '')::date,
			$6,
			$7,
			$8,
			NOW()
		)
		ON CONFLICT (uf, valid_from)
		DO UPDATE SET
			internal_rate = EXCLUDED.internal_rate,
			fcp_rate = EXCLUDED.fcp_rate,
			valid_to = EXCLUDED.valid_to,
			source_reference = EXCLUDED.source_reference,
			source_url = EXCLUDED.source_url,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id::text
	`

	var id string
	if err := r.db.QueryRow(
		ctx,
		query,
		p.UF,
		p.InternalRate,
		p.FCPRate,
		p.ValidFrom,
		p.ValidTo,
		p.SourceReference,
		p.SourceURL,
		p.Notes,
	).Scan(&id); err != nil {
		return "", fmt.Errorf("upsert icms state rate: %w", err)
	}

	return id, nil
}

func (r *Repository) FindApplicableStateRate(ctx context.Context, uf string, referenceDate string) (*StateRate, error) {
	if ok, err := r.tableExists(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	query := `
		SELECT
			id::text,
			uf,
			TRIM(TO_CHAR(internal_rate, 'FM999990.00')),
			TRIM(TO_CHAR(fcp_rate, 'FM999990.00')),
			TO_CHAR(valid_from, 'YYYY-MM-DD'),
			COALESCE(TO_CHAR(valid_to, 'YYYY-MM-DD'), ''),
			COALESCE(source_reference, ''),
			COALESCE(source_url, ''),
			COALESCE(notes, '')
		FROM icms_state_rates
		WHERE uf = $1
		  AND valid_from <= $2::date
		  AND (valid_to IS NULL OR valid_to >= $2::date)
		ORDER BY valid_from DESC
		LIMIT 1
	`

	var item StateRate
	err := r.db.QueryRow(ctx, query, strings.TrimSpace(strings.ToUpper(uf)), referenceDate).Scan(
		&item.ID,
		&item.UF,
		&item.InternalRate,
		&item.FCPRate,
		&item.ValidFrom,
		&item.ValidTo,
		&item.SourceReference,
		&item.SourceURL,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find applicable icms state rate: %w", err)
	}

	return &item, nil
}

func (r *Repository) tableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'icms_state_rates'
		)
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query).Scan(&exists); err != nil {
		return false, fmt.Errorf("check icms_state_rates table: %w", err)
	}
	return exists, nil
}

func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
