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

type StateICMSRule struct {
	ID                string `json:"id"`
	UF                string `json:"uf"`
	NCMPattern        string `json:"ncm_pattern"`
	MatchType         string `json:"match_type"`
	CEST              string `json:"cest"`
	OperationCode     string `json:"operation_code"`
	TaxRegime         string `json:"tax_regime"`
	TargetCRT         string `json:"target_crt"`
	RuleKind          string `json:"rule_kind"`
	CFOP              string `json:"cfop"`
	ICMSCST           string `json:"icms_cst"`
	CSOSN             string `json:"csosn"`
	ICMSRate          string `json:"icms_rate"`
	FCPRate           string `json:"fcp_rate"`
	ICMSSTRate        string `json:"icms_st_rate"`
	ICMSBaseReduction string `json:"icms_base_reduction"`
	CBenef            string `json:"cbenef"`
	ConfidenceScore   string `json:"confidence_score"`
	SourceReference   string `json:"source_reference"`
	SourceURL         string `json:"source_url"`
	Notes             string `json:"notes"`
	ValidFrom         string `json:"valid_from"`
	ValidTo           string `json:"valid_to"`
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

func (r *Repository) ListStateICMSRules(ctx context.Context, uf string, limit int) ([]StateICMSRule, error) {
	if limit <= 0 || limit > 500 {
		limit = 300
	}

	if ok, err := r.stateRulesTableExists(ctx); err != nil {
		return nil, err
	} else if !ok {
		return []StateICMSRule{}, nil
	}

	uf = strings.ToUpper(strings.TrimSpace(uf))
	query := `
		SELECT
			id::text,
			COALESCE(uf, ''),
			COALESCE(ncm_pattern, ''),
			COALESCE(match_type, ''),
			COALESCE(cest, ''),
			COALESCE(operation_code, ''),
			COALESCE(tax_regime, ''),
			COALESCE(target_crt, ''),
			COALESCE(rule_kind, ''),
			COALESCE(cfop, ''),
			COALESCE(icms_cst, ''),
			COALESCE(csosn, ''),
			COALESCE(icms_rate::text, ''),
			COALESCE(fcp_rate::text, ''),
			COALESCE(icms_st_rate::text, ''),
			COALESCE(icms_base_reduction::text, ''),
			COALESCE(cbenef, ''),
			COALESCE(confidence_score::text, ''),
			COALESCE(source_reference, ''),
			COALESCE(source_url, ''),
			COALESCE(notes, ''),
			TO_CHAR(valid_from, 'YYYY-MM-DD'),
			COALESCE(TO_CHAR(valid_to, 'YYYY-MM-DD'), '')
		FROM state_icms_rules
		WHERE is_active = TRUE
		  AND ($1 = '' OR UPPER(uf) = UPPER($1))
		ORDER BY
			uf ASC,
			CASE WHEN ncm_pattern = '' THEN 1 ELSE 0 END,
			CASE WHEN match_type = 'exact' THEN 0 ELSE 1 END,
			LENGTH(ncm_pattern) DESC,
			confidence_score DESC,
			created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, uf, limit)
	if err != nil {
		return nil, fmt.Errorf("list state icms rules: %w", err)
	}
	defer rows.Close()

	items := make([]StateICMSRule, 0, limit)
	for rows.Next() {
		var item StateICMSRule
		if err := rows.Scan(
			&item.ID,
			&item.UF,
			&item.NCMPattern,
			&item.MatchType,
			&item.CEST,
			&item.OperationCode,
			&item.TaxRegime,
			&item.TargetCRT,
			&item.RuleKind,
			&item.CFOP,
			&item.ICMSCST,
			&item.CSOSN,
			&item.ICMSRate,
			&item.FCPRate,
			&item.ICMSSTRate,
			&item.ICMSBaseReduction,
			&item.CBenef,
			&item.ConfidenceScore,
			&item.SourceReference,
			&item.SourceURL,
			&item.Notes,
			&item.ValidFrom,
			&item.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan state icms rule: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate state icms rules: %w", err)
	}

	return items, nil
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

func (r *Repository) stateRulesTableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'state_icms_rules'
		)
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query).Scan(&exists); err != nil {
		return false, fmt.Errorf("check state_icms_rules table: %w", err)
	}
	return exists, nil
}

func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
