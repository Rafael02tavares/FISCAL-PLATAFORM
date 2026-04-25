package tax

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/catalog"
)

type Repository struct {
	db *pgxpool.Pool
}

type suggestionSchemaSupport struct {
	enhancedSuggestionLog bool
	enhancedLegalBasis    bool
	profileRegimeContext  bool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type TaxMatch struct {
	ProductID       string
	OrganizationID  string
	MatchType       string
	ConfidenceScore float64

	NCM       string
	NCMEx     string
	CEST      string
	CClasTrib string
	CFOP      string
	CSOSN     string

	PISCST            string
	COFINSCST         string
	PISRevenueCode    string
	COFINSRevenueCode string

	ICMSCST           string
	ICMSValue         string
	IPIValue          string
	PISValue          string
	COFINSValue       string
	PISRate           string
	COFINSRate        string
	ICMSRate          string
	ICMSBaseReduction string
	FCPRate           string
	ICMSSTRate        string
	CBenef            string

	IBSRate           string
	CBSRate           string
	SelectiveTaxCode  string
	SelectiveTaxRate  string
	OperationCode     string
	EmitterUF         string
	RecipientUF       string
	SourceType        string
	TargetTaxRegime   string
	ObservedTaxRegime string
	TargetCRT         string
	ObservedCRT       string
}

type CFOPMatch struct {
	Code            string
	Description     string
	OperationType   string
	ConfidenceScore float64
}

type CESTMatch struct {
	Code            string
	NCMCode         string
	Segment         string
	Description     string
	LegalSource     string
	ConfidenceScore float64
}

func normalizeOptionalNumeric(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	normalized := strings.NewReplacer("%", "", "R$", "", " ", "", ",", ".").Replace(trimmed)
	if normalized == "" {
		return ""
	}

	if _, err := strconv.ParseFloat(normalized, 64); err != nil {
		return ""
	}

	return normalized
}

func (r *Repository) FindBestMatch(
	ctx context.Context,
	organizationID string,
	gtin string,
	description string,
	ncmCode string,
	taxRegime string,
	targetCRT string,
	operationCode string,
	emitterUF string,
	recipientUF string,
) (*TaxMatch, error) {
	normalizedGTIN := catalog.NormalizeGTIN(gtin)
	normalizedDescription := catalog.NormalizeDescription(description)
	normalizedNCMCode := normalizeNCMCode(ncmCode)
	organizationID = strings.TrimSpace(organizationID)
	taxRegime = strings.TrimSpace(taxRegime)
	targetCRT = strings.TrimSpace(targetCRT)
	operationCode = strings.TrimSpace(operationCode)
	emitterUF = strings.ToUpper(strings.TrimSpace(emitterUF))
	recipientUF = strings.ToUpper(strings.TrimSpace(recipientUF))

	if normalizedGTIN != "" {
		match, err := r.findByGTIN(
			ctx,
			normalizedGTIN,
			organizationID,
			taxRegime,
			targetCRT,
			operationCode,
			emitterUF,
			recipientUF,
		)
		if err == nil && match != nil {
			r.enrichWithNCMCode(ctx, match, normalizedNCMCode)
			match.MatchType = "gtin"
			if match.ConfidenceScore == 0 {
				match.ConfidenceScore = 0.95
			}
			return match, nil
		}
	}

	if normalizedDescription != "" {
		match, err := r.findByDescription(
			ctx,
			normalizedDescription,
			organizationID,
			taxRegime,
			targetCRT,
			operationCode,
			emitterUF,
			recipientUF,
		)
		if err == nil && match != nil {
			r.enrichWithNCMCode(ctx, match, normalizedNCMCode)
			match.MatchType = "description"
			if match.ConfidenceScore == 0 {
				match.ConfidenceScore = 0.75
			}
			return match, nil
		}
	}

	if normalizedNCMCode != "" {
		profileMatch, profileErr := r.findByNCMProfile(
			ctx,
			normalizedNCMCode,
			organizationID,
			taxRegime,
			targetCRT,
			operationCode,
			emitterUF,
			recipientUF,
		)
		if profileErr == nil && profileMatch != nil {
			profileMatch.MatchType = "ncm_profile"
			if profileMatch.ConfidenceScore == 0 {
				profileMatch.ConfidenceScore = 0.72
			}
			return profileMatch, nil
		}

		catalogMatch, catalogErr := r.findByNCMCatalog(ctx, normalizedNCMCode)
		if catalogErr == nil && catalogMatch != nil {
			return catalogMatch, nil
		}
	}

	return nil, fmt.Errorf("no matching product found")
}

func (r *Repository) enrichWithNCMCode(ctx context.Context, match *TaxMatch, ncmCode string) {
	if match == nil || ncmCode == "" || strings.TrimSpace(match.NCM) != "" {
		return
	}

	if item, err := r.findByNCMCatalog(ctx, ncmCode); err == nil && item != nil {
		match.NCM = item.NCM
		if match.CClasTrib == "" {
			match.CClasTrib = item.CClasTrib
		}
		if match.CEST == "" {
			match.CEST = item.CEST
		}
	}
}

func (r *Repository) findByGTIN(
	ctx context.Context,
	normalizedGTIN string,
	organizationID string,
	taxRegime string,
	targetCRT string,
	operationCode string,
	emitterUF string,
	recipientUF string,
) (*TaxMatch, error) {
	schema, err := r.getSuggestionSchemaSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("check suggestion schema: %w", err)
	}

	query := `
		SELECT
			p.id,
			COALESCE(ptp.organization_id::text, ''),
			COALESCE(ptp.confidence_score, 0),

			COALESCE(ptp.ncm, ''),
			COALESCE(ptp.ncm_ex, ''),
			COALESCE(ptp.cest, ''),
			COALESCE(ptp.cclas_trib, ''),
			COALESCE(ptp.cfop, ''),
			COALESCE(ptp.csosn, ''),

			COALESCE(ptp.pis_cst, ''),
			COALESCE(ptp.cofins_cst, ''),
			COALESCE(ptp.pis_revenue_code, ''),
			COALESCE(ptp.cofins_revenue_code, ''),

			COALESCE(ptp.icms_cst, ''),
			COALESCE(ptp.icms_value::text, ''),
			COALESCE(ptp.ipi_value::text, ''),
			COALESCE(ptp.pis_value::text, ''),
			COALESCE(ptp.cofins_value::text, ''),
			COALESCE(ptp.pis_rate::text, ''),
			COALESCE(ptp.cofins_rate::text, ''),
			COALESCE(ptp.icms_rate::text, ''),
			COALESCE(ptp.icms_base_reduction::text, ''),
			COALESCE(ptp.fcp_rate::text, ''),
			COALESCE(ptp.icms_st_rate::text, ''),
			COALESCE(ptp.cbenef, ''),

			COALESCE(ptp.ibs_rate::text, ''),
			COALESCE(ptp.cbs_rate::text, ''),
			COALESCE(ptp.selective_tax_code, ''),
			COALESCE(ptp.selective_tax_rate::text, ''),
			COALESCE(ptp.operation_code, ''),
			COALESCE(ptp.emitter_uf, ''),
			COALESCE(ptp.recipient_uf, ''),
			COALESCE(ptp.source_type, ''),
			'' AS target_tax_regime,
			'' AS observed_tax_regime,
			'' AS target_crt,
			'' AS observed_crt
		FROM products p
		INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
		WHERE p.normalized_gtin = $1
		  AND (ptp.organization_id = $2::uuid OR ptp.organization_id IS NULL)
		  AND (COALESCE(NULLIF(ptp.operation_code, ''), '') = '' OR LOWER(ptp.operation_code) = LOWER($3))
		  AND (COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' OR UPPER(ptp.emitter_uf) = UPPER($4))
		  AND (COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' OR UPPER(ptp.recipient_uf) = UPPER($5))
		ORDER BY
			CASE WHEN ptp.organization_id = $2::uuid THEN 0 ELSE 1 END,
			CASE
				WHEN LOWER(COALESCE(ptp.operation_code, '')) = LOWER($3) AND $3 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.operation_code, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN UPPER(COALESCE(ptp.emitter_uf, '')) = UPPER($4) AND $4 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN UPPER(COALESCE(ptp.recipient_uf, '')) = UPPER($5) AND $5 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' THEN 1
				ELSE 2
			END,
			ptp.confidence_score DESC,
			ptp.created_at DESC
		LIMIT 1
	`
	args := []any{normalizedGTIN, organizationID, operationCode, emitterUF, recipientUF}
	if schema.profileRegimeContext {
		query = `
			SELECT
				p.id,
				COALESCE(ptp.organization_id::text, ''),
				COALESCE(ptp.confidence_score, 0),

				COALESCE(ptp.ncm, ''),
				COALESCE(ptp.ncm_ex, ''),
				COALESCE(ptp.cest, ''),
				COALESCE(ptp.cclas_trib, ''),
				COALESCE(ptp.cfop, ''),
				COALESCE(ptp.csosn, ''),

				COALESCE(ptp.pis_cst, ''),
				COALESCE(ptp.cofins_cst, ''),
				COALESCE(ptp.pis_revenue_code, ''),
				COALESCE(ptp.cofins_revenue_code, ''),

				COALESCE(ptp.icms_cst, ''),
				COALESCE(ptp.icms_value::text, ''),
				COALESCE(ptp.ipi_value::text, ''),
				COALESCE(ptp.pis_value::text, ''),
				COALESCE(ptp.cofins_value::text, ''),
				COALESCE(ptp.pis_rate::text, ''),
				COALESCE(ptp.cofins_rate::text, ''),
				COALESCE(ptp.icms_rate::text, ''),
				COALESCE(ptp.icms_base_reduction::text, ''),
				COALESCE(ptp.fcp_rate::text, ''),
				COALESCE(ptp.icms_st_rate::text, ''),
				COALESCE(ptp.cbenef, ''),

				COALESCE(ptp.ibs_rate::text, ''),
				COALESCE(ptp.cbs_rate::text, ''),
				COALESCE(ptp.selective_tax_code, ''),
				COALESCE(ptp.selective_tax_rate::text, ''),
				COALESCE(ptp.operation_code, ''),
				COALESCE(ptp.emitter_uf, ''),
				COALESCE(ptp.recipient_uf, ''),
				COALESCE(ptp.source_type, ''),
				COALESCE(ptp.target_tax_regime, ''),
				COALESCE(ptp.observed_tax_regime, ''),
				COALESCE(ptp.target_crt, ''),
				COALESCE(ptp.observed_crt, '')
			FROM products p
			INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
			WHERE p.normalized_gtin = $1
			  AND (ptp.organization_id = $2::uuid OR ptp.organization_id IS NULL)
			  AND (
				COALESCE(NULLIF(ptp.operation_code, ''), '') = ''
				OR LOWER(ptp.operation_code) = LOWER($3)
			  )
			  AND (
				COALESCE(NULLIF(ptp.emitter_uf, ''), '') = ''
				OR UPPER(ptp.emitter_uf) = UPPER($4)
			  )
			  AND (
				COALESCE(NULLIF(ptp.recipient_uf, ''), '') = ''
				OR UPPER(ptp.recipient_uf) = UPPER($5)
			  )
			  AND (
				COALESCE(NULLIF(ptp.target_tax_regime, ''), '') = ''
				OR LOWER(ptp.target_tax_regime) = LOWER($6)
			  )
			  AND (
				COALESCE(NULLIF(ptp.target_crt, ''), '') = ''
				OR ptp.target_crt = $7
			  )
			ORDER BY
				CASE
					WHEN ptp.organization_id = $2::uuid THEN 0
					ELSE 1
				END,
				CASE
					WHEN LOWER(COALESCE(ptp.operation_code, '')) = LOWER($3) AND $3 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.operation_code, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN UPPER(COALESCE(ptp.emitter_uf, '')) = UPPER($4) AND $4 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN UPPER(COALESCE(ptp.recipient_uf, '')) = UPPER($5) AND $5 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN LOWER(COALESCE(ptp.target_tax_regime, '')) = LOWER($6) AND $6 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.target_tax_regime, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN COALESCE(ptp.target_crt, '') = $7 AND $7 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.target_crt, ''), '') = '' THEN 1
					ELSE 2
				END,
				ptp.confidence_score DESC,
				ptp.created_at DESC
			LIMIT 1
		`
		args = []any{normalizedGTIN, organizationID, operationCode, emitterUF, recipientUF, taxRegime, targetCRT}
	}

	var item TaxMatch

	err = r.db.QueryRow(ctx, query, args...).Scan(
		&item.ProductID,
		&item.OrganizationID,
		&item.ConfidenceScore,

		&item.NCM,
		&item.NCMEx,
		&item.CEST,
		&item.CClasTrib,
		&item.CFOP,
		&item.CSOSN,

		&item.PISCST,
		&item.COFINSCST,
		&item.PISRevenueCode,
		&item.COFINSRevenueCode,

		&item.ICMSCST,
		&item.ICMSValue,
		&item.IPIValue,
		&item.PISValue,
		&item.COFINSValue,
		&item.PISRate,
		&item.COFINSRate,
		&item.ICMSRate,
		&item.ICMSBaseReduction,
		&item.FCPRate,
		&item.ICMSSTRate,
		&item.CBenef,

		&item.IBSRate,
		&item.CBSRate,
		&item.SelectiveTaxCode,
		&item.SelectiveTaxRate,
		&item.OperationCode,
		&item.EmitterUF,
		&item.RecipientUF,
		&item.SourceType,
		&item.TargetTaxRegime,
		&item.ObservedTaxRegime,
		&item.TargetCRT,
		&item.ObservedCRT,
	)
	if err != nil {
		return nil, fmt.Errorf("find tax profile by gtin: %w", err)
	}

	return &item, nil
}

func (r *Repository) findByDescription(
	ctx context.Context,
	normalizedDescription string,
	organizationID string,
	taxRegime string,
	targetCRT string,
	operationCode string,
	emitterUF string,
	recipientUF string,
) (*TaxMatch, error) {
	schema, err := r.getSuggestionSchemaSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("check suggestion schema: %w", err)
	}

	query := `
		SELECT
			p.id,
			COALESCE(ptp.organization_id::text, ''),
			COALESCE(ptp.confidence_score, 0),

			COALESCE(ptp.ncm, ''),
			COALESCE(ptp.ncm_ex, ''),
			COALESCE(ptp.cest, ''),
			COALESCE(ptp.cclas_trib, ''),
			COALESCE(ptp.cfop, ''),
			COALESCE(ptp.csosn, ''),

			COALESCE(ptp.pis_cst, ''),
			COALESCE(ptp.cofins_cst, ''),
			COALESCE(ptp.pis_revenue_code, ''),
			COALESCE(ptp.cofins_revenue_code, ''),

			COALESCE(ptp.icms_cst, ''),
			COALESCE(ptp.icms_value::text, ''),
			COALESCE(ptp.ipi_value::text, ''),
			COALESCE(ptp.pis_value::text, ''),
			COALESCE(ptp.cofins_value::text, ''),
			COALESCE(ptp.pis_rate::text, ''),
			COALESCE(ptp.cofins_rate::text, ''),
			COALESCE(ptp.icms_rate::text, ''),
			COALESCE(ptp.icms_base_reduction::text, ''),
			COALESCE(ptp.fcp_rate::text, ''),
			COALESCE(ptp.icms_st_rate::text, ''),
			COALESCE(ptp.cbenef, ''),

			COALESCE(ptp.ibs_rate::text, ''),
			COALESCE(ptp.cbs_rate::text, ''),
			COALESCE(ptp.selective_tax_code, ''),
			COALESCE(ptp.selective_tax_rate::text, ''),
			COALESCE(ptp.operation_code, ''),
			COALESCE(ptp.emitter_uf, ''),
			COALESCE(ptp.recipient_uf, ''),
			COALESCE(ptp.source_type, ''),
			'' AS target_tax_regime,
			'' AS observed_tax_regime,
			'' AS target_crt,
			'' AS observed_crt
		FROM products p
		INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
		WHERE (
			p.normalized_description = $1
			OR p.normalized_description LIKE ('%' || $1 || '%')
			OR $1 LIKE ('%' || p.normalized_description || '%')
		)
		  AND (ptp.organization_id = $2::uuid OR ptp.organization_id IS NULL)
		  AND (COALESCE(NULLIF(ptp.operation_code, ''), '') = '' OR LOWER(ptp.operation_code) = LOWER($3))
		  AND (COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' OR UPPER(ptp.emitter_uf) = UPPER($4))
		  AND (COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' OR UPPER(ptp.recipient_uf) = UPPER($5))
		ORDER BY
			CASE WHEN ptp.organization_id = $2::uuid THEN 0 ELSE 1 END,
			CASE
				WHEN p.normalized_description = $1 THEN 0
				WHEN p.normalized_description LIKE ($1 || '%') THEN 1
				WHEN p.normalized_description LIKE ('%' || $1 || '%') THEN 2
				WHEN $1 LIKE ('%' || p.normalized_description || '%') THEN 3
				ELSE 4
			END,
			CASE
				WHEN LOWER(COALESCE(ptp.operation_code, '')) = LOWER($3) AND $3 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.operation_code, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN UPPER(COALESCE(ptp.emitter_uf, '')) = UPPER($4) AND $4 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN UPPER(COALESCE(ptp.recipient_uf, '')) = UPPER($5) AND $5 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' THEN 1
				ELSE 2
			END,
			ptp.confidence_score DESC,
			ptp.created_at DESC
		LIMIT 1
	`
	args := []any{normalizedDescription, organizationID, operationCode, emitterUF, recipientUF}
	if schema.profileRegimeContext {
		query = `
			SELECT
				p.id,
				COALESCE(ptp.organization_id::text, ''),
				COALESCE(ptp.confidence_score, 0),

				COALESCE(ptp.ncm, ''),
				COALESCE(ptp.ncm_ex, ''),
				COALESCE(ptp.cest, ''),
				COALESCE(ptp.cclas_trib, ''),
				COALESCE(ptp.cfop, ''),
				COALESCE(ptp.csosn, ''),

				COALESCE(ptp.pis_cst, ''),
				COALESCE(ptp.cofins_cst, ''),
				COALESCE(ptp.pis_revenue_code, ''),
				COALESCE(ptp.cofins_revenue_code, ''),

				COALESCE(ptp.icms_cst, ''),
				COALESCE(ptp.icms_value::text, ''),
				COALESCE(ptp.ipi_value::text, ''),
				COALESCE(ptp.pis_value::text, ''),
				COALESCE(ptp.cofins_value::text, ''),
				COALESCE(ptp.pis_rate::text, ''),
				COALESCE(ptp.cofins_rate::text, ''),
				COALESCE(ptp.icms_rate::text, ''),
				COALESCE(ptp.icms_base_reduction::text, ''),
				COALESCE(ptp.fcp_rate::text, ''),
				COALESCE(ptp.icms_st_rate::text, ''),
				COALESCE(ptp.cbenef, ''),

				COALESCE(ptp.ibs_rate::text, ''),
				COALESCE(ptp.cbs_rate::text, ''),
				COALESCE(ptp.selective_tax_code, ''),
				COALESCE(ptp.selective_tax_rate::text, ''),
				COALESCE(ptp.operation_code, ''),
				COALESCE(ptp.emitter_uf, ''),
				COALESCE(ptp.recipient_uf, ''),
				COALESCE(ptp.source_type, ''),
				COALESCE(ptp.target_tax_regime, ''),
				COALESCE(ptp.observed_tax_regime, ''),
				COALESCE(ptp.target_crt, ''),
				COALESCE(ptp.observed_crt, '')
			FROM products p
			INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
			WHERE (
				p.normalized_description = $1
				OR p.normalized_description LIKE ('%' || $1 || '%')
				OR $1 LIKE ('%' || p.normalized_description || '%')
			)
			  AND (ptp.organization_id = $2::uuid OR ptp.organization_id IS NULL)
			  AND (
				COALESCE(NULLIF(ptp.operation_code, ''), '') = ''
				OR LOWER(ptp.operation_code) = LOWER($3)
			  )
			  AND (
				COALESCE(NULLIF(ptp.emitter_uf, ''), '') = ''
				OR UPPER(ptp.emitter_uf) = UPPER($4)
			  )
			  AND (
				COALESCE(NULLIF(ptp.recipient_uf, ''), '') = ''
				OR UPPER(ptp.recipient_uf) = UPPER($5)
			  )
			  AND (
				COALESCE(NULLIF(ptp.target_tax_regime, ''), '') = ''
				OR LOWER(ptp.target_tax_regime) = LOWER($6)
			  )
			  AND (
				COALESCE(NULLIF(ptp.target_crt, ''), '') = ''
				OR ptp.target_crt = $7
			  )
			ORDER BY
				CASE
					WHEN ptp.organization_id = $2::uuid THEN 0
					ELSE 1
				END,
				CASE
					WHEN p.normalized_description = $1 THEN 0
					WHEN p.normalized_description LIKE ($1 || '%') THEN 1
					WHEN p.normalized_description LIKE ('%' || $1 || '%') THEN 2
					WHEN $1 LIKE ('%' || p.normalized_description || '%') THEN 3
					ELSE 4
				END,
				CASE
					WHEN LOWER(COALESCE(ptp.operation_code, '')) = LOWER($3) AND $3 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.operation_code, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN UPPER(COALESCE(ptp.emitter_uf, '')) = UPPER($4) AND $4 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN UPPER(COALESCE(ptp.recipient_uf, '')) = UPPER($5) AND $5 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN LOWER(COALESCE(ptp.target_tax_regime, '')) = LOWER($6) AND $6 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.target_tax_regime, ''), '') = '' THEN 1
					ELSE 2
				END,
				CASE
					WHEN COALESCE(ptp.target_crt, '') = $7 AND $7 <> '' THEN 0
					WHEN COALESCE(NULLIF(ptp.target_crt, ''), '') = '' THEN 1
					ELSE 2
				END,
				ptp.confidence_score DESC,
				ptp.created_at DESC
			LIMIT 1
		`
		args = []any{normalizedDescription, organizationID, operationCode, emitterUF, recipientUF, taxRegime, targetCRT}
	}

	var item TaxMatch

	err = r.db.QueryRow(ctx, query, args...).Scan(
		&item.ProductID,
		&item.OrganizationID,
		&item.ConfidenceScore,

		&item.NCM,
		&item.NCMEx,
		&item.CEST,
		&item.CClasTrib,
		&item.CFOP,
		&item.CSOSN,

		&item.PISCST,
		&item.COFINSCST,
		&item.PISRevenueCode,
		&item.COFINSRevenueCode,

		&item.ICMSCST,
		&item.ICMSValue,
		&item.IPIValue,
		&item.PISValue,
		&item.COFINSValue,
		&item.PISRate,
		&item.COFINSRate,
		&item.ICMSRate,
		&item.ICMSBaseReduction,
		&item.FCPRate,
		&item.ICMSSTRate,
		&item.CBenef,

		&item.IBSRate,
		&item.CBSRate,
		&item.SelectiveTaxCode,
		&item.SelectiveTaxRate,
		&item.OperationCode,
		&item.EmitterUF,
		&item.RecipientUF,
		&item.SourceType,
		&item.TargetTaxRegime,
		&item.ObservedTaxRegime,
		&item.TargetCRT,
		&item.ObservedCRT,
	)
	if err != nil {
		return nil, fmt.Errorf("find tax profile by description: %w", err)
	}

	return &item, nil
}

func (r *Repository) findByNCMProfile(
	ctx context.Context,
	normalizedNCMCode string,
	organizationID string,
	taxRegime string,
	targetCRT string,
	operationCode string,
	emitterUF string,
	recipientUF string,
) (*TaxMatch, error) {
	schema, err := r.getSuggestionSchemaSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("check suggestion schema: %w", err)
	}
	if !schema.profileRegimeContext {
		return nil, fmt.Errorf("ncm profile lookup requires enhanced profile schema")
	}

	query := `
		SELECT
			p.id,
			COALESCE(ptp.organization_id::text, ''),
			COALESCE(ptp.confidence_score, 0),

			COALESCE(ptp.ncm, ''),
			COALESCE(ptp.ncm_ex, ''),
			COALESCE(ptp.cest, ''),
			COALESCE(ptp.cclas_trib, ''),
			COALESCE(ptp.cfop, ''),
			COALESCE(ptp.csosn, ''),

			COALESCE(ptp.pis_cst, ''),
			COALESCE(ptp.cofins_cst, ''),
			COALESCE(ptp.pis_revenue_code, ''),
			COALESCE(ptp.cofins_revenue_code, ''),

			COALESCE(ptp.icms_cst, ''),
			COALESCE(ptp.icms_value::text, ''),
			COALESCE(ptp.ipi_value::text, ''),
			COALESCE(ptp.pis_value::text, ''),
			COALESCE(ptp.cofins_value::text, ''),
			COALESCE(ptp.pis_rate::text, ''),
			COALESCE(ptp.cofins_rate::text, ''),
			COALESCE(ptp.icms_rate::text, ''),
			COALESCE(ptp.icms_base_reduction::text, ''),
			COALESCE(ptp.fcp_rate::text, ''),
			COALESCE(ptp.icms_st_rate::text, ''),
			COALESCE(ptp.cbenef, ''),

			COALESCE(ptp.ibs_rate::text, ''),
			COALESCE(ptp.cbs_rate::text, ''),
			COALESCE(ptp.selective_tax_code, ''),
			COALESCE(ptp.selective_tax_rate::text, ''),
			COALESCE(ptp.operation_code, ''),
			COALESCE(ptp.emitter_uf, ''),
			COALESCE(ptp.recipient_uf, ''),
			COALESCE(ptp.source_type, ''),
			COALESCE(ptp.target_tax_regime, ''),
			COALESCE(ptp.observed_tax_regime, ''),
			COALESCE(ptp.target_crt, ''),
			COALESCE(ptp.observed_crt, '')
		FROM products p
		INNER JOIN product_tax_profiles ptp ON ptp.product_id = p.id
		WHERE ptp.ncm = $1
		  AND (ptp.organization_id = $2::uuid OR ptp.organization_id IS NULL)
		  AND (
			COALESCE(NULLIF(ptp.operation_code, ''), '') = ''
			OR LOWER(ptp.operation_code) = LOWER($3)
		  )
		  AND (
			COALESCE(NULLIF(ptp.emitter_uf, ''), '') = ''
			OR UPPER(ptp.emitter_uf) = UPPER($4)
		  )
		  AND (
			COALESCE(NULLIF(ptp.recipient_uf, ''), '') = ''
			OR UPPER(ptp.recipient_uf) = UPPER($5)
		  )
		  AND (
			COALESCE(NULLIF(ptp.target_tax_regime, ''), '') = ''
			OR LOWER(ptp.target_tax_regime) = LOWER($6)
		  )
		  AND (
			COALESCE(NULLIF(ptp.target_crt, ''), '') = ''
			OR ptp.target_crt = $7
		  )
		ORDER BY
			CASE WHEN ptp.organization_id = $2::uuid THEN 0 ELSE 1 END,
			CASE WHEN ptp.source_type = 'manual_entry' THEN 0 ELSE 1 END,
			CASE
				WHEN LOWER(COALESCE(ptp.operation_code, '')) = LOWER($3) AND $3 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.operation_code, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN UPPER(COALESCE(ptp.emitter_uf, '')) = UPPER($4) AND $4 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.emitter_uf, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN UPPER(COALESCE(ptp.recipient_uf, '')) = UPPER($5) AND $5 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.recipient_uf, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN LOWER(COALESCE(ptp.target_tax_regime, '')) = LOWER($6) AND $6 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.target_tax_regime, ''), '') = '' THEN 1
				ELSE 2
			END,
			CASE
				WHEN COALESCE(ptp.target_crt, '') = $7 AND $7 <> '' THEN 0
				WHEN COALESCE(NULLIF(ptp.target_crt, ''), '') = '' THEN 1
				ELSE 2
			END,
			ptp.confidence_score DESC,
			ptp.created_at DESC
		LIMIT 1
	`

	var item TaxMatch
	err = r.db.QueryRow(ctx, query, normalizedNCMCode, organizationID, operationCode, emitterUF, recipientUF, taxRegime, targetCRT).Scan(
		&item.ProductID,
		&item.OrganizationID,
		&item.ConfidenceScore,

		&item.NCM,
		&item.NCMEx,
		&item.CEST,
		&item.CClasTrib,
		&item.CFOP,
		&item.CSOSN,

		&item.PISCST,
		&item.COFINSCST,
		&item.PISRevenueCode,
		&item.COFINSRevenueCode,

		&item.ICMSCST,
		&item.ICMSValue,
		&item.IPIValue,
		&item.PISValue,
		&item.COFINSValue,
		&item.PISRate,
		&item.COFINSRate,
		&item.ICMSRate,
		&item.ICMSBaseReduction,
		&item.FCPRate,
		&item.ICMSSTRate,
		&item.CBenef,

		&item.IBSRate,
		&item.CBSRate,
		&item.SelectiveTaxCode,
		&item.SelectiveTaxRate,
		&item.OperationCode,
		&item.EmitterUF,
		&item.RecipientUF,
		&item.SourceType,
		&item.TargetTaxRegime,
		&item.ObservedTaxRegime,
		&item.TargetCRT,
		&item.ObservedCRT,
	)
	if err != nil {
		return nil, fmt.Errorf("find tax profile by ncm: %w", err)
	}

	return &item, nil
}

func (r *Repository) findByNCMCatalog(ctx context.Context, normalizedNCMCode string) (*TaxMatch, error) {
	query := `
		SELECT
			COALESCE(code, '')
		FROM ncm_catalog
		WHERE code = $1
		  AND is_active = TRUE
		ORDER BY start_date DESC NULLS LAST, created_at DESC
		LIMIT 1
	`

	var item TaxMatch
	err := r.db.QueryRow(ctx, query, normalizedNCMCode).Scan(&item.NCM)
	if err != nil {
		return nil, fmt.Errorf("find ncm catalog by code: %w", err)
	}

	item.MatchType = "ncm_catalog"
	item.ConfidenceScore = 0.65

	return &item, nil
}

func (r *Repository) FindSuggestedCEST(ctx context.Context, ncmCode string, currentCEST string) (*CESTMatch, error) {
	currentCEST = normalizeCESTCode(currentCEST)
	ncmCode = normalizeNCMCode(ncmCode)

	if currentCEST != "" {
		item, err := r.findCESTByCode(ctx, currentCEST)
		if err == nil && item != nil {
			item.ConfidenceScore = 0.96
			return item, nil
		}
	}

	if ncmCode == "" {
		return nil, fmt.Errorf("ncm code is required to suggest cest")
	}

	query := `
		SELECT
			COALESCE(code, ''),
			COALESCE(ncm_code, ''),
			COALESCE(segment, ''),
			COALESCE(description, ''),
			COALESCE(legal_source, '')
		FROM cest_catalog
		WHERE is_active = TRUE
		  AND (
			ncm_code = $1
			OR ncm_code LIKE ($1 || '%')
			OR $1 LIKE (ncm_code || '%')
		  )
		ORDER BY
			CASE
				WHEN ncm_code = $1 THEN 0
				WHEN ncm_code LIKE ($1 || '%') THEN 1
				WHEN $1 LIKE (ncm_code || '%') THEN 2
				ELSE 3
			END,
			code
		LIMIT 1
	`

	var item CESTMatch
	if err := r.db.QueryRow(ctx, query, ncmCode).Scan(
		&item.Code,
		&item.NCMCode,
		&item.Segment,
		&item.Description,
		&item.LegalSource,
	); err != nil {
		return nil, fmt.Errorf("find cest by ncm: %w", err)
	}

	item.ConfidenceScore = 0.68
	return &item, nil
}

func (r *Repository) findCESTByCode(ctx context.Context, code string) (*CESTMatch, error) {
	query := `
		SELECT
			COALESCE(code, ''),
			COALESCE(ncm_code, ''),
			COALESCE(segment, ''),
			COALESCE(description, ''),
			COALESCE(legal_source, '')
		FROM cest_catalog
		WHERE code = $1
		  AND is_active = TRUE
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var item CESTMatch
	if err := r.db.QueryRow(ctx, query, code).Scan(
		&item.Code,
		&item.NCMCode,
		&item.Segment,
		&item.Description,
		&item.LegalSource,
	); err != nil {
		return nil, fmt.Errorf("find cest by code: %w", err)
	}

	return &item, nil
}

func normalizeNCMCode(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func normalizeCESTCode(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func normalizeCFOPCode(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func (r *Repository) FindSuggestedCFOP(
	ctx context.Context,
	defaultCFOP string,
	operationName string,
	operationDirection string,
	emitterUF string,
	recipientUF string,
) (*CFOPMatch, error) {
	defaultCFOP = normalizeCFOPCode(defaultCFOP)
	operationType := normalizeOperationDirection(operationDirection)
	prefixes := preferredCFOPPrefixes(operationType, emitterUF, recipientUF)

	if defaultCFOP != "" {
		item, err := r.findCFOPByCode(ctx, defaultCFOP)
		if err == nil && item != nil {
			item.ConfidenceScore = 0.98
			return item, nil
		}
	}

	item, err := r.findCFOPByHeuristics(ctx, operationName, operationType, prefixes)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (r *Repository) findCFOPByCode(ctx context.Context, code string) (*CFOPMatch, error) {
	query := `
		SELECT
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(operation_type, '')
		FROM cfop_catalog
		WHERE code = $1
		LIMIT 1
	`

	var item CFOPMatch
	if err := r.db.QueryRow(ctx, query, code).Scan(
		&item.Code,
		&item.Description,
		&item.OperationType,
	); err != nil {
		return nil, fmt.Errorf("find cfop by code: %w", err)
	}

	return &item, nil
}

func (r *Repository) findCFOPByHeuristics(ctx context.Context, operationName string, operationType string, prefixes []string) (*CFOPMatch, error) {
	query := `
		SELECT
			COALESCE(code, ''),
			COALESCE(description, ''),
			COALESCE(operation_type, ''),
			CASE
				WHEN $1 <> '' AND code LIKE ($1 || '%') THEN 40
				WHEN $2 <> '' AND code LIKE ($2 || '%') THEN 35
				WHEN $3 <> '' AND code LIKE ($3 || '%') THEN 30
				ELSE 0
			END
			+ CASE
				WHEN $4 <> '' AND description ILIKE ('%' || $4 || '%') THEN 25
				ELSE 0
			END
			+ CASE
				WHEN $5 <> '' AND operation_type = $5 THEN 15
				ELSE 0
			END
			+ CASE
				WHEN $6 AND ind_devolution THEN 12
				ELSE 0
			END
			+ CASE
				WHEN $7 AND ind_transport THEN 12
				ELSE 0
			END
			+ CASE
				WHEN $8 AND ind_comunication THEN 12
				ELSE 0
			END AS ranking
		FROM cfop_catalog
		WHERE ($5 = '' OR operation_type = $5)
		   OR ($1 <> '' AND code LIKE ($1 || '%'))
		   OR ($2 <> '' AND code LIKE ($2 || '%'))
		   OR ($3 <> '' AND code LIKE ($3 || '%'))
		ORDER BY ranking DESC, code ASC
		LIMIT 1
	`

	keyword := primaryOperationKeyword(operationName)
	var prefix1, prefix2, prefix3 string
	if len(prefixes) > 0 {
		prefix1 = prefixes[0]
	}
	if len(prefixes) > 1 {
		prefix2 = prefixes[1]
	}
	if len(prefixes) > 2 {
		prefix3 = prefixes[2]
	}

	nameNormalized := strings.ToLower(strings.TrimSpace(operationName))

	var item CFOPMatch
	var ranking int
	if err := r.db.QueryRow(
		ctx,
		query,
		prefix1,
		prefix2,
		prefix3,
		keyword,
		operationType,
		strings.Contains(nameNormalized, "devol"),
		strings.Contains(nameNormalized, "transport"),
		strings.Contains(nameNormalized, "comunic"),
	).Scan(
		&item.Code,
		&item.Description,
		&item.OperationType,
		&ranking,
	); err != nil {
		return nil, fmt.Errorf("find cfop by heuristics: %w", err)
	}

	item.ConfidenceScore = 0.55
	if ranking >= 55 {
		item.ConfidenceScore = 0.84
	} else if ranking >= 40 {
		item.ConfidenceScore = 0.72
	}

	return &item, nil
}

func normalizeOperationDirection(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(value, "entr"), strings.Contains(value, "inbound"):
		return "entrada"
	case strings.Contains(value, "exit"), strings.Contains(value, "sa"), strings.Contains(value, "outbound"):
		return "saida"
	default:
		return ""
	}
}

func preferredCFOPPrefixes(operationType string, emitterUF string, recipientUF string) []string {
	sameUF := strings.EqualFold(strings.TrimSpace(emitterUF), strings.TrimSpace(recipientUF))
	switch operationType {
	case "entrada":
		if sameUF {
			return []string{"1", "2", "3"}
		}
		return []string{"2", "1", "3"}
	case "saida":
		if sameUF {
			return []string{"5", "6", "7"}
		}
		return []string{"6", "5", "7"}
	default:
		return []string{}
	}
}

func primaryOperationKeyword(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(value, "devol"):
		return "devol"
	case strings.Contains(value, "transport"):
		return "transport"
	case strings.Contains(value, "comunic"):
		return "comunica"
	case strings.Contains(value, "transfer"):
		return "transfer"
	case strings.Contains(value, "industrial"):
		return "industrial"
	case strings.Contains(value, "comercial"):
		return "comercial"
	case strings.Contains(value, "venda"):
		return "venda"
	case strings.Contains(value, "compra"):
		return "compra"
	default:
		return ""
	}
}

type CreateSuggestionLogParams struct {
	OrganizationID string

	GTIN          string
	Description   string
	OperationCode string
	CClasTrib     string

	SuggestedNCM   string
	SuggestedNCMEx string
	SuggestedCEST  string
	SuggestedCFOP  string

	SuggestedPISCST        string
	SuggestedCOFINSCST     string
	SuggestedPISRevCode    string
	SuggestedCOFINSRevCode string

	SuggestedICMSCST           string
	SuggestedCSOSN             string
	SuggestedCBenef            string
	SuggestedICMS              string
	SuggestedIPI               string
	SuggestedPIS               string
	SuggestedCOFINS            string
	SuggestedPISRate           string
	SuggestedCOFINSRate        string
	SuggestedICMSRate          string
	SuggestedICMSBaseReduction string
	SuggestedFCPRate           string
	SuggestedICMSSTRate        string

	SuggestedIBSRate          string
	SuggestedCBSRate          string
	SuggestedSelectiveTaxCode string
	SuggestedSelectiveTaxRate string

	MatchType       string
	ConfidenceScore float64
}

func (r *Repository) CreateSuggestionLog(ctx context.Context, p CreateSuggestionLogParams) (string, error) {
	schema, err := r.getSuggestionSchemaSupport(ctx)
	if err != nil {
		return "", fmt.Errorf("check suggestion schema: %w", err)
	}

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

			suggested_icms_cst,
			suggested_csosn,
			suggested_cbenef,
			suggested_icms_value,
			suggested_ipi_value,
			suggested_pis_value,
			suggested_cofins_value,
			suggested_pis_rate,
			suggested_cofins_rate,
			suggested_icms_rate,
			suggested_icms_base_reduction,
			suggested_fcp_rate,
			suggested_icms_st_rate,

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
			$13,
			$14,
			$15,
			NULLIF($16, '')::numeric,
			NULLIF($17, '')::numeric,
			NULLIF($18, '')::numeric,
			NULLIF($19, '')::numeric,
			NULLIF($20, '')::numeric,
			NULLIF($21, '')::numeric,
			NULLIF($22, '')::numeric,
			NULLIF($23, '')::numeric,
			NULLIF($24, '')::numeric,
			NULLIF($25, '')::numeric,
			NULLIF($26, '')::numeric,
			NULLIF($27, '')::numeric,

			$28,
			$29
		)
		RETURNING id
	`

	args := []any{
		strings.TrimSpace(p.OrganizationID),
		strings.TrimSpace(p.GTIN),
		strings.TrimSpace(p.Description),
		strings.TrimSpace(p.OperationCode),
		strings.TrimSpace(p.CClasTrib),
		strings.TrimSpace(p.SuggestedNCM),
		strings.TrimSpace(p.SuggestedCEST),
		strings.TrimSpace(p.SuggestedCFOP),
		strings.TrimSpace(p.SuggestedPISCST),
		strings.TrimSpace(p.SuggestedCOFINSCST),
		strings.TrimSpace(p.SuggestedPISRevCode),
		strings.TrimSpace(p.SuggestedCOFINSRevCode),
		strings.TrimSpace(p.SuggestedICMSCST),
		strings.TrimSpace(p.SuggestedCSOSN),
		strings.TrimSpace(p.SuggestedCBenef),
		normalizeOptionalNumeric(p.SuggestedICMS),
		normalizeOptionalNumeric(p.SuggestedIPI),
		normalizeOptionalNumeric(p.SuggestedPIS),
		normalizeOptionalNumeric(p.SuggestedCOFINS),
		normalizeOptionalNumeric(p.SuggestedPISRate),
		normalizeOptionalNumeric(p.SuggestedCOFINSRate),
		normalizeOptionalNumeric(p.SuggestedICMSRate),
		normalizeOptionalNumeric(p.SuggestedICMSBaseReduction),
		normalizeOptionalNumeric(p.SuggestedFCPRate),
		normalizeOptionalNumeric(p.SuggestedICMSSTRate),
		normalizeOptionalNumeric(p.SuggestedIBSRate),
		normalizeOptionalNumeric(p.SuggestedCBSRate),
		strings.TrimSpace(p.MatchType),
		p.ConfidenceScore,
	}

	if schema.enhancedSuggestionLog {
		query = `
			INSERT INTO tax_suggestions_log (
				organization_id,
				gtin,
				description,
				operation_code,
				cclas_trib,

				suggested_ncm,
				suggested_ncm_ex,
				suggested_cest,
				suggested_cfop,

				suggested_pis_cst,
				suggested_cofins_cst,
				suggested_pis_revenue_code,
				suggested_cofins_revenue_code,

				suggested_icms_cst,
				suggested_csosn,
				suggested_cbenef,
				suggested_icms_value,
				suggested_ipi_value,
				suggested_pis_value,
				suggested_cofins_value,
				suggested_pis_rate,
				suggested_cofins_rate,
				suggested_icms_rate,
				suggested_icms_base_reduction,
				suggested_fcp_rate,
				suggested_icms_st_rate,

				suggested_ibs_rate,
				suggested_cbs_rate,
				suggested_selective_tax_code,
				suggested_selective_tax_rate,

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
				$13,
				$14,
				$15,
				$16,
				NULLIF($17, '')::numeric,
				NULLIF($18, '')::numeric,
				NULLIF($19, '')::numeric,
				NULLIF($20, '')::numeric,
				NULLIF($21, '')::numeric,
				NULLIF($22, '')::numeric,
				NULLIF($23, '')::numeric,
				NULLIF($24, '')::numeric,
				NULLIF($25, '')::numeric,
				NULLIF($26, '')::numeric,
				NULLIF($27, '')::numeric,
				NULLIF($28, '')::numeric,
				NULLIF($29, '')::numeric,
				$30,
				NULLIF($31, '')::numeric,

				$32,
				$33
			)
			RETURNING id
		`

		args = []any{
			strings.TrimSpace(p.OrganizationID),
			strings.TrimSpace(p.GTIN),
			strings.TrimSpace(p.Description),
			strings.TrimSpace(p.OperationCode),
			strings.TrimSpace(p.CClasTrib),
			strings.TrimSpace(p.SuggestedNCM),
			strings.TrimSpace(p.SuggestedNCMEx),
			strings.TrimSpace(p.SuggestedCEST),
			strings.TrimSpace(p.SuggestedCFOP),
			strings.TrimSpace(p.SuggestedPISCST),
			strings.TrimSpace(p.SuggestedCOFINSCST),
			strings.TrimSpace(p.SuggestedPISRevCode),
			strings.TrimSpace(p.SuggestedCOFINSRevCode),
			strings.TrimSpace(p.SuggestedICMSCST),
			strings.TrimSpace(p.SuggestedCSOSN),
			strings.TrimSpace(p.SuggestedCBenef),
			normalizeOptionalNumeric(p.SuggestedICMS),
			normalizeOptionalNumeric(p.SuggestedIPI),
			normalizeOptionalNumeric(p.SuggestedPIS),
			normalizeOptionalNumeric(p.SuggestedCOFINS),
			normalizeOptionalNumeric(p.SuggestedPISRate),
			normalizeOptionalNumeric(p.SuggestedCOFINSRate),
			normalizeOptionalNumeric(p.SuggestedICMSRate),
			normalizeOptionalNumeric(p.SuggestedICMSBaseReduction),
			normalizeOptionalNumeric(p.SuggestedFCPRate),
			normalizeOptionalNumeric(p.SuggestedICMSSTRate),
			normalizeOptionalNumeric(p.SuggestedIBSRate),
			normalizeOptionalNumeric(p.SuggestedCBSRate),
			strings.TrimSpace(p.SuggestedSelectiveTaxCode),
			normalizeOptionalNumeric(p.SuggestedSelectiveTaxRate),
			strings.TrimSpace(p.MatchType),
			p.ConfidenceScore,
		}
	}

	var id string

	err = r.db.QueryRow(
		ctx,
		query,
		args...,
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
	schema, err := r.getSuggestionSchemaSupport(ctx)
	if err != nil {
		return fmt.Errorf("check suggestion schema: %w", err)
	}

	if !schema.enhancedLegalBasis {
		return nil
	}

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

	_, err = r.db.Exec(
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

func (r *Repository) getSuggestionSchemaSupport(ctx context.Context) (suggestionSchemaSupport, error) {
	query := `
		SELECT
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestions_log'
				  AND column_name = 'suggested_ncm_ex'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestions_log'
				  AND column_name = 'suggested_selective_tax_code'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestions_log'
				  AND column_name = 'suggested_selective_tax_rate'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestion_legal_basis'
				  AND column_name = 'suggestion_log_id'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestion_legal_basis'
				  AND column_name = 'legal_source_id'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestion_legal_basis'
				  AND column_name = 'tax_type'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestion_legal_basis'
				  AND column_name = 'applied_reason'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'tax_suggestion_legal_basis'
				  AND column_name = 'weight'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'product_tax_profiles'
				  AND column_name = 'target_tax_regime'
			)
	`

	var ncmEx bool
	var selectiveCode bool
	var selectiveRate bool
	var suggestionLogID bool
	var legalSourceID bool
	var taxType bool
	var appliedReason bool
	var weight bool
	var profileRegimeContext bool
	if err := r.db.QueryRow(ctx, query).Scan(
		&ncmEx,
		&selectiveCode,
		&selectiveRate,
		&suggestionLogID,
		&legalSourceID,
		&taxType,
		&appliedReason,
		&weight,
		&profileRegimeContext,
	); err != nil {
		return suggestionSchemaSupport{}, err
	}

	return suggestionSchemaSupport{
		enhancedSuggestionLog: ncmEx && selectiveCode && selectiveRate,
		enhancedLegalBasis:    suggestionLogID && legalSourceID && taxType && appliedReason && weight,
		profileRegimeContext:  profileRegimeContext,
	}, nil
}
