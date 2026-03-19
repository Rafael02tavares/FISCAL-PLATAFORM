package tax

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/catalog"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type TaxMatch struct {
	ProductID       string
	MatchType       string
	ConfidenceScore float64

	NCM       string
	CEST      string
	CClasTrib string

	PISCST            string
	COFINSCST         string
	PISRevenueCode    string
	COFINSRevenueCode string

	ICMSValue   string
	IPIValue    string
	PISValue    string
	COFINSValue string

	IBSRate string
	CBSRate string
}

func (r *Repository) FindBestMatch(ctx context.Context, gtin, description string) (*TaxMatch, error) {
	normalizedGTIN := catalog.NormalizeGTIN(gtin)
	normalizedDescription := catalog.NormalizeDescription(description)

	if normalizedGTIN != "" {
		match, err := r.findByGTIN(ctx, normalizedGTIN)
		if err == nil && match != nil {
			match.MatchType = "gtin"
			if match.ConfidenceScore == 0 {
				match.ConfidenceScore = 0.95
			}
			return match, nil
		}
	}

	if normalizedDescription != "" {
		match, err := r.findByDescription(ctx, normalizedDescription)
		if err == nil && match != nil {
			match.MatchType = "description"
			if match.ConfidenceScore == 0 {
				match.ConfidenceScore = 0.75
			}
			return match, nil
		}
	}

	return nil, fmt.Errorf("no matching product found")
}

func (r *Repository) findByGTIN(ctx context.Context, normalizedGTIN string) (*TaxMatch, error) {
	query := `
		SELECT
			p.id,
			COALESCE(ptp.confidence_score, 0),

			COALESCE(ptp.ncm, ''),
			COALESCE(ptp.cest, ''),
			COALESCE(ptp.cclas_trib, ''),

			COALESCE(ptp.pis_cst, ''),
			COALESCE(ptp.cofins_cst, ''),
			COALESCE(ptp.pis_revenue_code, ''),
			COALESCE(ptp.cofins_revenue_code, ''),

			COALESCE(ptp.icms_value::text, ''),
			COALESCE(ptp.ipi_value::text, ''),
			COALESCE(ptp.pis_value::text, ''),
			COALESCE(ptp.cofins_value::text, ''),

			COALESCE(ptp.ibs_rate::text, ''),
			COALESCE(ptp.cbs_rate::text, '')
		FROM products p
		INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
		WHERE p.normalized_gtin = $1
		ORDER BY ptp.confidence_score DESC, ptp.created_at DESC
		LIMIT 1
	`

	var item TaxMatch

	err := r.db.QueryRow(ctx, query, normalizedGTIN).Scan(
		&item.ProductID,
		&item.ConfidenceScore,

		&item.NCM,
		&item.CEST,
		&item.CClasTrib,

		&item.PISCST,
		&item.COFINSCST,
		&item.PISRevenueCode,
		&item.COFINSRevenueCode,

		&item.ICMSValue,
		&item.IPIValue,
		&item.PISValue,
		&item.COFINSValue,

		&item.IBSRate,
		&item.CBSRate,
	)
	if err != nil {
		return nil, fmt.Errorf("find tax profile by gtin: %w", err)
	}

	return &item, nil
}

func (r *Repository) findByDescription(ctx context.Context, normalizedDescription string) (*TaxMatch, error) {
	query := `
		SELECT
			p.id,
			COALESCE(ptp.confidence_score, 0),

			COALESCE(ptp.ncm, ''),
			COALESCE(ptp.cest, ''),
			COALESCE(ptp.cclas_trib, ''),

			COALESCE(ptp.pis_cst, ''),
			COALESCE(ptp.cofins_cst, ''),
			COALESCE(ptp.pis_revenue_code, ''),
			COALESCE(ptp.cofins_revenue_code, ''),

			COALESCE(ptp.icms_value::text, ''),
			COALESCE(ptp.ipi_value::text, ''),
			COALESCE(ptp.pis_value::text, ''),
			COALESCE(ptp.cofins_value::text, ''),

			COALESCE(ptp.ibs_rate::text, ''),
			COALESCE(ptp.cbs_rate::text, '')
		FROM products p
		INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
		WHERE p.normalized_description = $1
		ORDER BY ptp.confidence_score DESC, ptp.created_at DESC
		LIMIT 1
	`

	var item TaxMatch

	err := r.db.QueryRow(ctx, query, normalizedDescription).Scan(
		&item.ProductID,
		&item.ConfidenceScore,

		&item.NCM,
		&item.CEST,
		&item.CClasTrib,

		&item.PISCST,
		&item.COFINSCST,
		&item.PISRevenueCode,
		&item.COFINSRevenueCode,

		&item.ICMSValue,
		&item.IPIValue,
		&item.PISValue,
		&item.COFINSValue,

		&item.IBSRate,
		&item.CBSRate,
	)
	if err != nil {
		return nil, fmt.Errorf("find tax profile by description: %w", err)
	}

	return &item, nil
}

type CreateSuggestionLogParams struct {
	OrganizationID string

	GTIN          string
	Description   string
	OperationCode string
	CClasTrib     string

	SuggestedNCM  string
	SuggestedCEST string
	SuggestedCFOP string

	SuggestedPISCST        string
	SuggestedCOFINSCST     string
	SuggestedPISRevCode    string
	SuggestedCOFINSRevCode string

	SuggestedICMS   string
	SuggestedIPI    string
	SuggestedPIS    string
	SuggestedCOFINS string

	SuggestedIBSRate string
	SuggestedCBSRate string

	MatchType       string
	ConfidenceScore float64
}

func (r *Repository) CreateSuggestionLog(ctx context.Context, p CreateSuggestionLogParams) (string, error) {
	query := `
		INSERT INTO tax_suggestions_log (
			organization_id,
			gtin,
			description,
			operation_code,
			cclas_trib,

			suggested_ncm,
			suggested_cest,
			suggested_cfop,

			suggested_pis_cst,
			suggested_cofins_cst,
			suggested_pis_revenue_code,
			suggested_cofins_revenue_code,

			suggested_icms_value,
			suggested_ipi_value,
			suggested_pis_value,
			suggested_cofins_value,

			suggested_ibs_rate,
			suggested_cbs_rate,

			match_type,
			confidence_score
		)
		VALUES (
			NULLIF($1, '')::uuid,
			$2,
			$3,
			$4,
			$5,

			$6,
			$7,
			$8,

			$9,
			$10,
			$11,
			$12,

			NULLIF($13, '')::numeric,
			NULLIF($14, '')::numeric,
			NULLIF($15, '')::numeric,
			NULLIF($16, '')::numeric,

			NULLIF($17, '')::numeric,
			NULLIF($18, '')::numeric,

			$19,
			$20
		)
		RETURNING id
	`

	var id string

	err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(p.OrganizationID),
		p.GTIN,
		p.Description,
		p.OperationCode,
		p.CClasTrib,

		p.SuggestedNCM,
		p.SuggestedCEST,
		p.SuggestedCFOP,

		p.SuggestedPISCST,
		p.SuggestedCOFINSCST,
		p.SuggestedPISRevCode,
		p.SuggestedCOFINSRevCode,

		p.SuggestedICMS,
		p.SuggestedIPI,
		p.SuggestedPIS,
		p.SuggestedCOFINS,

		p.SuggestedIBSRate,
		p.SuggestedCBSRate,

		p.MatchType,
		p.ConfidenceScore,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create tax suggestion log: %w", err)
	}

	return id, nil
}
type CreateSuggestionLegalBasisParams struct {
	SuggestionLogID string
	LegalSourceID   string
	TaxType         string
	AppliedReason   string
	Weight          string
}

func (r *Repository) CreateSuggestionLegalBasis(ctx context.Context, p CreateSuggestionLegalBasisParams) error {
	query := `
		INSERT INTO tax_suggestion_legal_basis (
			suggestion_log_id,
			legal_source_id,
			tax_type,
			applied_reason,
			weight
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			NULLIF($5, '')::numeric
		)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		p.SuggestionLogID,
		p.LegalSourceID,
		p.TaxType,
		p.AppliedReason,
		p.Weight,
	)
	if err != nil {
		return fmt.Errorf("create suggestion legal basis: %w", err)
	}

	return nil
}
