package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type catalogSchemaSupport struct {
	productCodeColumn bool
	enhancedProfile   bool
	regimeContext     bool
	uniqueGTINIndex   bool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Product struct {
	ID                    string
	ProductCode           string
	GTIN                  string
	NormalizedGTIN        string
	Description           string
	NormalizedDescription string
}

type ProductTaxProfile struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	OrganizationID  string `json:"organization_id"`
	SourceInvoiceID string `json:"source_invoice_id"`

	NCM               string `json:"ncm"`
	NCMDescription    string `json:"ncm_description"`
	NCMEx             string `json:"ncm_ex"`
	CEST              string `json:"cest"`
	CESTDescription   string `json:"cest_description"`
	CFOP              string `json:"cfop"`
	CClasTrib         string `json:"cclas_trib"`
	PISCST            string `json:"pis_cst"`
	COFINSCST         string `json:"cofins_cst"`
	PISRevenueCode    string `json:"pis_revenue_code"`
	COFINSRevenueCode string `json:"cofins_revenue_code"`
	ICMSCST           string `json:"icms_cst"`
	CSOSN             string `json:"csosn"`
	CBenef            string `json:"cbenef"`

	ICMSValue         string `json:"icms_value"`
	IPIValue          string `json:"ipi_value"`
	PISValue          string `json:"pis_value"`
	COFINSValue       string `json:"cofins_value"`
	PISRate           string `json:"pis_rate"`
	COFINSRate        string `json:"cofins_rate"`
	ICMSRate          string `json:"icms_rate"`
	ICMSBaseReduction string `json:"icms_base_reduction"`
	FCPRate           string `json:"fcp_rate"`
	ICMSSTRate        string `json:"icms_st_rate"`
	IBSRate           string `json:"ibs_rate"`
	CBSRate           string `json:"cbs_rate"`
	SelectiveTaxCode  string `json:"selective_tax_code"`
	SelectiveTaxRate  string `json:"selective_tax_rate"`

	OperationCode     string  `json:"operation_code"`
	EmitterUF         string  `json:"emitter_uf"`
	RecipientUF       string  `json:"recipient_uf"`
	OperationNature   string  `json:"operation_nature"`
	TargetTaxRegime   string  `json:"target_tax_regime"`
	ObservedTaxRegime string  `json:"observed_tax_regime"`
	TargetCRT         string  `json:"target_crt"`
	ObservedCRT       string  `json:"observed_crt"`
	ConfidenceScore   float64 `json:"confidence_score"`
	SourceType        string  `json:"source_type"`
}

type CatalogProductView struct {
	ID          string            `json:"id"`
	ProductCode string            `json:"product_code"`
	GTIN        string            `json:"gtin"`
	Description string            `json:"description"`
	Profile     ProductTaxProfile `json:"profile"`
}

func (r *Repository) FindProductByNormalizedGTIN(ctx context.Context, normalizedGTIN string) (*Product, error) {
	schema, err := r.getCatalogSchemaSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("check catalog schema: %w", err)
	}

	query := `
		SELECT id, '' AS product_code, COALESCE(gtin, ''), COALESCE(normalized_gtin, ''), description, normalized_description
		FROM products
		WHERE normalized_gtin = $1
		LIMIT 1
	`
	if schema.productCodeColumn {
		query = `
			SELECT id, COALESCE(product_code, ''), COALESCE(gtin, ''), COALESCE(normalized_gtin, ''), description, normalized_description
			FROM products
			WHERE normalized_gtin = $1
			LIMIT 1
		`
	}

	var p Product
	err = r.db.QueryRow(ctx, query, normalizedGTIN).Scan(
		&p.ID,
		&p.ProductCode,
		&p.GTIN,
		&p.NormalizedGTIN,
		&p.Description,
		&p.NormalizedDescription,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *Repository) FindProductByNormalizedDescription(ctx context.Context, normalizedDescription string) (*Product, error) {
	schema, err := r.getCatalogSchemaSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("check catalog schema: %w", err)
	}

	query := `
		SELECT id, '' AS product_code, COALESCE(gtin, ''), COALESCE(normalized_gtin, ''), description, normalized_description
		FROM products
		WHERE normalized_description = $1
		LIMIT 1
	`
	if schema.productCodeColumn {
		query = `
			SELECT id, COALESCE(product_code, ''), COALESCE(gtin, ''), COALESCE(normalized_gtin, ''), description, normalized_description
			FROM products
			WHERE normalized_description = $1
			LIMIT 1
		`
	}

	var p Product
	err = r.db.QueryRow(ctx, query, normalizedDescription).Scan(
		&p.ID,
		&p.ProductCode,
		&p.GTIN,
		&p.NormalizedGTIN,
		&p.Description,
		&p.NormalizedDescription,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *Repository) CreateProduct(ctx context.Context, productCode, gtin, normalizedGTIN, description, normalizedDescription string) (string, error) {
	schema, err := r.getCatalogSchemaSupport(ctx)
	if err != nil {
		return "", fmt.Errorf("check catalog schema: %w", err)
	}

	if schema.uniqueGTINIndex && strings.TrimSpace(normalizedGTIN) != "" {
		return r.upsertProductByGTIN(ctx, schema, productCode, gtin, normalizedGTIN, description, normalizedDescription)
	}

	query := `
		INSERT INTO products (gtin, normalized_gtin, description, normalized_description)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	args := []any{gtin, normalizedGTIN, description, normalizedDescription}
	if schema.productCodeColumn {
		query = `
			INSERT INTO products (product_code, gtin, normalized_gtin, description, normalized_description)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		args = []any{productCode, gtin, normalizedGTIN, description, normalizedDescription}
	}

	var productID string
	err = r.db.QueryRow(ctx, query, args...).Scan(&productID)
	if err != nil {
		return "", fmt.Errorf("create product: %w", err)
	}

	return productID, nil
}

func (r *Repository) upsertProductByGTIN(ctx context.Context, schema catalogSchemaSupport, productCode, gtin, normalizedGTIN, description, normalizedDescription string) (string, error) {
	query := `
		INSERT INTO products (gtin, normalized_gtin, description, normalized_description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (normalized_gtin) WHERE NULLIF(TRIM(normalized_gtin), '') IS NOT NULL
		DO UPDATE SET
			gtin = EXCLUDED.gtin,
			description = CASE
				WHEN NULLIF(TRIM(EXCLUDED.description), '') IS NULL THEN products.description
				ELSE EXCLUDED.description
			END,
			normalized_description = CASE
				WHEN NULLIF(TRIM(EXCLUDED.normalized_description), '') IS NULL THEN products.normalized_description
				ELSE EXCLUDED.normalized_description
			END,
			updated_at = NOW()
		RETURNING id
	`
	args := []any{gtin, normalizedGTIN, description, normalizedDescription}

	if schema.productCodeColumn {
		query = `
			INSERT INTO products (product_code, gtin, normalized_gtin, description, normalized_description)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (normalized_gtin) WHERE NULLIF(TRIM(normalized_gtin), '') IS NOT NULL
			DO UPDATE SET
				product_code = CASE
					WHEN NULLIF(TRIM(EXCLUDED.product_code), '') IS NULL THEN products.product_code
					ELSE EXCLUDED.product_code
				END,
				gtin = EXCLUDED.gtin,
				description = CASE
					WHEN NULLIF(TRIM(EXCLUDED.description), '') IS NULL THEN products.description
					ELSE EXCLUDED.description
				END,
				normalized_description = CASE
					WHEN NULLIF(TRIM(EXCLUDED.normalized_description), '') IS NULL THEN products.normalized_description
					ELSE EXCLUDED.normalized_description
				END,
				updated_at = NOW()
			RETURNING id
		`
		args = []any{productCode, gtin, normalizedGTIN, description, normalizedDescription}
	}

	var productID string
	if err := r.db.QueryRow(ctx, query, args...).Scan(&productID); err != nil {
		return "", fmt.Errorf("upsert product by gtin: %w", err)
	}

	return productID, nil
}

func (r *Repository) UpdateProduct(ctx context.Context, productID, productCode, gtin, normalizedGTIN, description, normalizedDescription string) error {
	schema, err := r.getCatalogSchemaSupport(ctx)
	if err != nil {
		return fmt.Errorf("check catalog schema: %w", err)
	}

	query := `
		UPDATE products
		SET gtin = $2,
			normalized_gtin = $3,
			description = $4,
			normalized_description = $5,
			updated_at = NOW()
		WHERE id = $1::uuid
	`
	args := []any{productID, gtin, normalizedGTIN, description, normalizedDescription}
	if schema.productCodeColumn {
		query = `
			UPDATE products
			SET product_code = $2,
				gtin = $3,
				normalized_gtin = $4,
				description = $5,
				normalized_description = $6,
				updated_at = NOW()
			WHERE id = $1::uuid
		`
		args = []any{productID, productCode, gtin, normalizedGTIN, description, normalizedDescription}
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}

	return nil
}

type CreateTaxProfileParams struct {
	ProductID       string
	OrganizationID  string
	SourceInvoiceID string

	NCM               string
	NCMEx             string
	CEST              string
	CFOP              string
	CClasTrib         string
	PISCST            string
	COFINSCST         string
	PISRevenueCode    string
	COFINSRevenueCode string
	ICMSCST           string
	CSOSN             string
	CBenef            string

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
	IBSRate           string
	CBSRate           string
	SelectiveTaxCode  string
	SelectiveTaxRate  string

	OperationCode     string
	EmitterUF         string
	RecipientUF       string
	OperationNature   string
	TargetTaxRegime   string
	ObservedTaxRegime string
	TargetCRT         string
	ObservedCRT       string
	ConfidenceScore   float64
	SourceType        string
}

func (r *Repository) CreateTaxProfile(ctx context.Context, p CreateTaxProfileParams) error {
	schema, err := r.getCatalogSchemaSupport(ctx)
	if err != nil {
		return fmt.Errorf("check catalog schema: %w", err)
	}

	query := `
		INSERT INTO product_tax_profiles (
			product_id,
			organization_id,
			source_invoice_id,
			ncm,
			cest,
			cfop,
			cclas_trib,
			pis_cst,
			cofins_cst,
			pis_revenue_code,
			cofins_revenue_code,
			icms_cst,
			csosn,
			cbenef,
			icms_value,
			ipi_value,
			pis_value,
			cofins_value,
			pis_rate,
			cofins_rate,
			icms_rate,
			icms_base_reduction,
			fcp_rate,
			icms_st_rate,
			ibs_rate,
			cbs_rate,
			operation_code,
			emitter_uf,
			recipient_uf,
			operation_nature,
			target_tax_regime,
			observed_tax_regime,
			target_crt,
			observed_crt,
			confidence_score,
			source_type
		)
		VALUES (
			$1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			NULLIF($15, '')::numeric,
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
			CASE
				WHEN NULLIF($27, '') IS NULL THEN NULL
				WHEN EXISTS (SELECT 1 FROM fiscal_operations fo WHERE fo.code = $27) THEN $27
				ELSE NULL
			END,
			$28, $29, $30, $31, $32, $33, $34, $35, $36
		)
	`

	args := []any{
		p.ProductID,
		p.OrganizationID,
		p.SourceInvoiceID,
		p.NCM,
		p.CEST,
		p.CFOP,
		p.CClasTrib,
		p.PISCST,
		p.COFINSCST,
		p.PISRevenueCode,
		p.COFINSRevenueCode,
		p.ICMSCST,
		p.CSOSN,
		p.CBenef,
		p.ICMSValue,
		p.IPIValue,
		p.PISValue,
		p.COFINSValue,
		p.PISRate,
		p.COFINSRate,
		p.ICMSRate,
		p.ICMSBaseReduction,
		p.FCPRate,
		p.ICMSSTRate,
		p.IBSRate,
		p.CBSRate,
		p.OperationCode,
		p.EmitterUF,
		p.RecipientUF,
		p.OperationNature,
		p.TargetTaxRegime,
		p.ObservedTaxRegime,
		p.TargetCRT,
		p.ObservedCRT,
		p.ConfidenceScore,
		p.SourceType,
	}

	if schema.enhancedProfile {
		query = `
			INSERT INTO product_tax_profiles (
				product_id,
				organization_id,
				source_invoice_id,
				ncm,
				ncm_ex,
				cest,
				cfop,
				cclas_trib,
				pis_cst,
				cofins_cst,
				pis_revenue_code,
				cofins_revenue_code,
				icms_cst,
				csosn,
				cbenef,
				icms_value,
				ipi_value,
				pis_value,
				cofins_value,
				pis_rate,
				cofins_rate,
				icms_rate,
				icms_base_reduction,
				fcp_rate,
				icms_st_rate,
				ibs_rate,
				cbs_rate,
				selective_tax_code,
				selective_tax_rate,
				operation_code,
				emitter_uf,
				recipient_uf,
				operation_nature,
				target_tax_regime,
				observed_tax_regime,
				target_crt,
				observed_crt,
				confidence_score,
				source_type
			)
			VALUES (
				$1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15,
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
				NULLIF($29, '')::numeric,
				CASE
					WHEN NULLIF($30, '') IS NULL THEN NULL
					WHEN EXISTS (SELECT 1 FROM fiscal_operations fo WHERE fo.code = $30) THEN $30
					ELSE NULL
				END,
				$31, $32, $33, $34, $35, $36, $37, $38, $39
			)
		`
		args = []any{
			p.ProductID,
			p.OrganizationID,
			p.SourceInvoiceID,
			p.NCM,
			p.NCMEx,
			p.CEST,
			p.CFOP,
			p.CClasTrib,
			p.PISCST,
			p.COFINSCST,
			p.PISRevenueCode,
			p.COFINSRevenueCode,
			p.ICMSCST,
			p.CSOSN,
			p.CBenef,
			p.ICMSValue,
			p.IPIValue,
			p.PISValue,
			p.COFINSValue,
			p.PISRate,
			p.COFINSRate,
			p.ICMSRate,
			p.ICMSBaseReduction,
			p.FCPRate,
			p.ICMSSTRate,
			p.IBSRate,
			p.CBSRate,
			p.SelectiveTaxCode,
			p.SelectiveTaxRate,
			p.OperationCode,
			p.EmitterUF,
			p.RecipientUF,
			p.OperationNature,
			p.TargetTaxRegime,
			p.ObservedTaxRegime,
			p.TargetCRT,
			p.ObservedCRT,
			p.ConfidenceScore,
			p.SourceType,
		}
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("create tax profile: %w", err)
	}

	return nil
}

func (r *Repository) ListCatalogProducts(ctx context.Context, organizationID, rawQuery string, limit, offset int) ([]CatalogProductView, error) {
	schema, err := r.getCatalogSchemaSupport(ctx)
	if err != nil {
		return nil, fmt.Errorf("check catalog schema: %w", err)
	}

	trimmedQuery := strings.TrimSpace(rawQuery)
	normalizedQuery := NormalizeDescription(trimmedQuery)

	query := `
		SELECT
			p.id,
			'' AS product_code,
			COALESCE(p.gtin, ''),
			COALESCE(p.description, ''),
			COALESCE(ptp.id::text, ''),
			COALESCE(ptp.organization_id::text, ''),
			COALESCE(ptp.source_invoice_id::text, ''),
			COALESCE(ptp.ncm, ''),
			COALESCE((
				SELECT COALESCE(NULLIF(nc.full_description, ''), nc.description)
				FROM ncm_catalog nc
				WHERE nc.code = ptp.ncm
				  AND nc.is_active = TRUE
				ORDER BY nc.created_at DESC
				LIMIT 1
			), ''),
			'' AS ncm_ex,
			COALESCE(ptp.cest, ''),
			COALESCE((
				SELECT CONCAT_WS(' - ', NULLIF(cc.segment, ''), NULLIF(cc.description, ''))
				FROM cest_catalog cc
				WHERE cc.code = ptp.cest
				  AND cc.is_active = TRUE
				  AND (COALESCE(ptp.ncm, '') = '' OR COALESCE(cc.ncm_code, '') = '' OR cc.ncm_code = ptp.ncm)
				ORDER BY
					CASE WHEN cc.ncm_code = ptp.ncm THEN 0 ELSE 1 END,
					cc.segment NULLS LAST
				LIMIT 1
			), ''),
			COALESCE(ptp.cfop, ''),
			COALESCE(ptp.cclas_trib, ''),
			COALESCE(ptp.pis_cst, ''),
			COALESCE(ptp.cofins_cst, ''),
			COALESCE(ptp.pis_revenue_code, ''),
			COALESCE(ptp.cofins_revenue_code, ''),
			COALESCE(ptp.icms_cst, ''),
			COALESCE(ptp.csosn, ''),
			COALESCE(ptp.cbenef, ''),
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
			COALESCE(ptp.ibs_rate::text, ''),
			COALESCE(ptp.cbs_rate::text, ''),
			'' AS selective_tax_code,
			'' AS selective_tax_rate,
			COALESCE(ptp.operation_code, ''),
			COALESCE(ptp.emitter_uf, ''),
			COALESCE(ptp.recipient_uf, ''),
			COALESCE(ptp.operation_nature, ''),
			'' AS target_tax_regime,
			'' AS observed_tax_regime,
			'' AS target_crt,
			'' AS observed_crt,
			COALESCE(ptp.confidence_score, 0),
			COALESCE(ptp.source_type, '')
		FROM products p
		LEFT JOIN LATERAL (
			SELECT *
			FROM product_tax_profiles ptp
			WHERE ptp.product_id = p.id
			  AND (ptp.organization_id = $1::uuid OR ptp.organization_id IS NULL)
			ORDER BY
				CASE WHEN ptp.organization_id = $1::uuid THEN 0 ELSE 1 END,
				CASE WHEN ptp.source_type = 'manual_entry' THEN 0 ELSE 1 END,
				ptp.created_at DESC
			LIMIT 1
		) ptp ON true
		WHERE (
			$2 = ''
			OR p.description ILIKE ('%' || $2 || '%')
			OR COALESCE(p.gtin, '') ILIKE ('%' || $2 || '%')
			OR COALESCE(ptp.ncm, '') ILIKE ('%' || $2 || '%')
			OR COALESCE(ptp.cest, '') ILIKE ('%' || $2 || '%')
			OR COALESCE(ptp.cfop, '') ILIKE ('%' || $2 || '%')
			OR ($3 <> '' AND COALESCE(p.normalized_description, '') LIKE ('%' || $3 || '%'))
		)
		ORDER BY
			CASE
				WHEN $3 <> '' AND COALESCE(p.normalized_description, '') = $3 THEN 0
				WHEN $3 <> '' AND COALESCE(p.normalized_description, '') LIKE ($3 || '%') THEN 1
				WHEN $3 <> '' AND COALESCE(p.normalized_description, '') LIKE ('%' || $3 || '%') THEN 2
				WHEN p.description ILIKE ('%' || $2 || '%') THEN 3
				ELSE 4
			END,
			CASE
				WHEN $2 = '' AND $3 = '' THEN GREATEST(COALESCE(ptp.created_at, p.created_at), COALESCE(p.updated_at, p.created_at))
			END DESC NULLS LAST,
			p.description ASC
		LIMIT $4 OFFSET $5
	`
	if schema.productCodeColumn && schema.enhancedProfile && schema.regimeContext {
		query = `
			SELECT
				p.id,
				COALESCE(p.product_code, ''),
				COALESCE(p.gtin, ''),
				COALESCE(p.description, ''),
				COALESCE(ptp.id::text, ''),
				COALESCE(ptp.organization_id::text, ''),
				COALESCE(ptp.source_invoice_id::text, ''),
				COALESCE(ptp.ncm, ''),
				COALESCE((
					SELECT COALESCE(NULLIF(nc.full_description, ''), nc.description)
					FROM ncm_catalog nc
					WHERE nc.code = ptp.ncm
					  AND nc.is_active = TRUE
					ORDER BY
						CASE WHEN COALESCE(nc.ex_code, '') = COALESCE(ptp.ncm_ex, '') THEN 0 ELSE 1 END,
						nc.created_at DESC
					LIMIT 1
				), ''),
				COALESCE(ptp.ncm_ex, ''),
				COALESCE(ptp.cest, ''),
				COALESCE((
					SELECT CONCAT_WS(' - ', NULLIF(cc.segment, ''), NULLIF(cc.description, ''))
					FROM cest_catalog cc
					WHERE cc.code = ptp.cest
					  AND cc.is_active = TRUE
					  AND (COALESCE(ptp.ncm, '') = '' OR COALESCE(cc.ncm_code, '') = '' OR cc.ncm_code = ptp.ncm)
					ORDER BY
						CASE WHEN cc.ncm_code = ptp.ncm THEN 0 ELSE 1 END,
						cc.segment NULLS LAST
					LIMIT 1
				), ''),
				COALESCE(ptp.cfop, ''),
				COALESCE(ptp.cclas_trib, ''),
				COALESCE(ptp.pis_cst, ''),
				COALESCE(ptp.cofins_cst, ''),
				COALESCE(ptp.pis_revenue_code, ''),
				COALESCE(ptp.cofins_revenue_code, ''),
				COALESCE(ptp.icms_cst, ''),
				COALESCE(ptp.csosn, ''),
				COALESCE(ptp.cbenef, ''),
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
				COALESCE(ptp.ibs_rate::text, ''),
				COALESCE(ptp.cbs_rate::text, ''),
				COALESCE(ptp.selective_tax_code, ''),
				COALESCE(ptp.selective_tax_rate::text, ''),
				COALESCE(ptp.operation_code, ''),
				COALESCE(ptp.emitter_uf, ''),
				COALESCE(ptp.recipient_uf, ''),
				COALESCE(ptp.operation_nature, ''),
				COALESCE(ptp.target_tax_regime, ''),
				COALESCE(ptp.observed_tax_regime, ''),
				COALESCE(ptp.target_crt, ''),
				COALESCE(ptp.observed_crt, ''),
				COALESCE(ptp.confidence_score, 0),
				COALESCE(ptp.source_type, '')
			FROM products p
			LEFT JOIN LATERAL (
				SELECT *
				FROM product_tax_profiles ptp
				WHERE ptp.product_id = p.id
				  AND (ptp.organization_id = $1::uuid OR ptp.organization_id IS NULL)
				ORDER BY
					CASE WHEN ptp.organization_id = $1::uuid THEN 0 ELSE 1 END,
					CASE WHEN ptp.source_type = 'manual_entry' THEN 0 ELSE 1 END,
					ptp.created_at DESC
				LIMIT 1
			) ptp ON true
			WHERE (
				$2 = ''
				OR p.description ILIKE ('%' || $2 || '%')
				OR COALESCE(p.gtin, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(p.product_code, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(ptp.ncm, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(ptp.cest, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(ptp.cfop, '') ILIKE ('%' || $2 || '%')
				OR ($3 <> '' AND COALESCE(p.normalized_description, '') LIKE ('%' || $3 || '%'))
			)
			ORDER BY
				CASE
					WHEN $3 <> '' AND COALESCE(p.normalized_description, '') = $3 THEN 0
					WHEN $3 <> '' AND COALESCE(p.normalized_description, '') LIKE ($3 || '%') THEN 1
					WHEN $3 <> '' AND COALESCE(p.normalized_description, '') LIKE ('%' || $3 || '%') THEN 2
					WHEN p.description ILIKE ('%' || $2 || '%') THEN 3
					WHEN COALESCE(p.product_code, '') ILIKE ('%' || $2 || '%') THEN 4
					ELSE 5
				END,
				CASE
					WHEN $2 = '' AND $3 = '' THEN GREATEST(COALESCE(ptp.created_at, p.created_at), COALESCE(p.updated_at, p.created_at))
				END DESC NULLS LAST,
				p.description ASC
			LIMIT $4 OFFSET $5
		`
	} else if schema.productCodeColumn && schema.enhancedProfile {
		query = `
			SELECT
				p.id,
				COALESCE(p.product_code, ''),
				COALESCE(p.gtin, ''),
				COALESCE(p.description, ''),
				COALESCE(ptp.id::text, ''),
				COALESCE(ptp.organization_id::text, ''),
				COALESCE(ptp.source_invoice_id::text, ''),
				COALESCE(ptp.ncm, ''),
				COALESCE((
					SELECT COALESCE(NULLIF(nc.full_description, ''), nc.description)
					FROM ncm_catalog nc
					WHERE nc.code = ptp.ncm
					  AND nc.is_active = TRUE
					ORDER BY
						CASE WHEN COALESCE(nc.ex_code, '') = COALESCE(ptp.ncm_ex, '') THEN 0 ELSE 1 END,
						nc.created_at DESC
					LIMIT 1
				), ''),
				COALESCE(ptp.ncm_ex, ''),
				COALESCE(ptp.cest, ''),
				COALESCE((
					SELECT CONCAT_WS(' - ', NULLIF(cc.segment, ''), NULLIF(cc.description, ''))
					FROM cest_catalog cc
					WHERE cc.code = ptp.cest
					  AND cc.is_active = TRUE
					  AND (COALESCE(ptp.ncm, '') = '' OR COALESCE(cc.ncm_code, '') = '' OR cc.ncm_code = ptp.ncm)
					ORDER BY
						CASE WHEN cc.ncm_code = ptp.ncm THEN 0 ELSE 1 END,
						cc.segment NULLS LAST
					LIMIT 1
				), ''),
				COALESCE(ptp.cfop, ''),
				COALESCE(ptp.cclas_trib, ''),
				COALESCE(ptp.pis_cst, ''),
				COALESCE(ptp.cofins_cst, ''),
				COALESCE(ptp.pis_revenue_code, ''),
				COALESCE(ptp.cofins_revenue_code, ''),
				COALESCE(ptp.icms_cst, ''),
				COALESCE(ptp.csosn, ''),
				COALESCE(ptp.cbenef, ''),
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
				COALESCE(ptp.ibs_rate::text, ''),
				COALESCE(ptp.cbs_rate::text, ''),
				COALESCE(ptp.selective_tax_code, ''),
				COALESCE(ptp.selective_tax_rate::text, ''),
				COALESCE(ptp.operation_code, ''),
				COALESCE(ptp.emitter_uf, ''),
				COALESCE(ptp.recipient_uf, ''),
				COALESCE(ptp.operation_nature, ''),
				'' AS target_tax_regime,
				'' AS observed_tax_regime,
				'' AS target_crt,
				'' AS observed_crt,
				COALESCE(ptp.confidence_score, 0),
				COALESCE(ptp.source_type, '')
			FROM products p
			LEFT JOIN LATERAL (
				SELECT *
				FROM product_tax_profiles ptp
				WHERE ptp.product_id = p.id
				  AND (ptp.organization_id = $1::uuid OR ptp.organization_id IS NULL)
				ORDER BY
					CASE WHEN ptp.organization_id = $1::uuid THEN 0 ELSE 1 END,
					CASE WHEN ptp.source_type = 'manual_entry' THEN 0 ELSE 1 END,
					ptp.created_at DESC
				LIMIT 1
			) ptp ON true
			WHERE (
				$2 = ''
				OR p.description ILIKE ('%' || $2 || '%')
				OR COALESCE(p.gtin, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(p.product_code, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(ptp.ncm, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(ptp.cest, '') ILIKE ('%' || $2 || '%')
				OR COALESCE(ptp.cfop, '') ILIKE ('%' || $2 || '%')
				OR ($3 <> '' AND COALESCE(p.normalized_description, '') LIKE ('%' || $3 || '%'))
			)
			ORDER BY
				CASE
					WHEN $3 <> '' AND COALESCE(p.normalized_description, '') = $3 THEN 0
					WHEN $3 <> '' AND COALESCE(p.normalized_description, '') LIKE ($3 || '%') THEN 1
					WHEN $3 <> '' AND COALESCE(p.normalized_description, '') LIKE ('%' || $3 || '%') THEN 2
					WHEN p.description ILIKE ('%' || $2 || '%') THEN 3
					WHEN COALESCE(p.product_code, '') ILIKE ('%' || $2 || '%') THEN 4
					ELSE 5
				END,
				CASE
					WHEN $2 = '' AND $3 = '' THEN GREATEST(COALESCE(ptp.created_at, p.created_at), COALESCE(p.updated_at, p.created_at))
				END DESC NULLS LAST,
				p.description ASC
			LIMIT $4 OFFSET $5
		`
	}

	rows, err := r.db.Query(ctx, query, organizationID, trimmedQuery, normalizedQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list catalog products: %w", err)
	}
	defer rows.Close()

	items := make([]CatalogProductView, 0)
	for rows.Next() {
		var item CatalogProductView
		if err := rows.Scan(
			&item.ID,
			&item.ProductCode,
			&item.GTIN,
			&item.Description,
			&item.Profile.ID,
			&item.Profile.OrganizationID,
			&item.Profile.SourceInvoiceID,
			&item.Profile.NCM,
			&item.Profile.NCMDescription,
			&item.Profile.NCMEx,
			&item.Profile.CEST,
			&item.Profile.CESTDescription,
			&item.Profile.CFOP,
			&item.Profile.CClasTrib,
			&item.Profile.PISCST,
			&item.Profile.COFINSCST,
			&item.Profile.PISRevenueCode,
			&item.Profile.COFINSRevenueCode,
			&item.Profile.ICMSCST,
			&item.Profile.CSOSN,
			&item.Profile.CBenef,
			&item.Profile.ICMSValue,
			&item.Profile.IPIValue,
			&item.Profile.PISValue,
			&item.Profile.COFINSValue,
			&item.Profile.PISRate,
			&item.Profile.COFINSRate,
			&item.Profile.ICMSRate,
			&item.Profile.ICMSBaseReduction,
			&item.Profile.FCPRate,
			&item.Profile.ICMSSTRate,
			&item.Profile.IBSRate,
			&item.Profile.CBSRate,
			&item.Profile.SelectiveTaxCode,
			&item.Profile.SelectiveTaxRate,
			&item.Profile.OperationCode,
			&item.Profile.EmitterUF,
			&item.Profile.RecipientUF,
			&item.Profile.OperationNature,
			&item.Profile.TargetTaxRegime,
			&item.Profile.ObservedTaxRegime,
			&item.Profile.TargetCRT,
			&item.Profile.ObservedCRT,
			&item.Profile.ConfidenceScore,
			&item.Profile.SourceType,
		); err != nil {
			return nil, fmt.Errorf("scan catalog product: %w", err)
		}

		item.Profile.ProductID = item.ID
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog products: %w", err)
	}

	return items, nil
}

func (r *Repository) getCatalogSchemaSupport(ctx context.Context) (catalogSchemaSupport, error) {
	query := `
		SELECT
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'products'
				  AND column_name = 'product_code'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'product_tax_profiles'
				  AND column_name = 'ncm_ex'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'product_tax_profiles'
				  AND column_name = 'selective_tax_code'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'product_tax_profiles'
				  AND column_name = 'selective_tax_rate'
			),
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'product_tax_profiles'
				  AND column_name = 'target_tax_regime'
			),
			to_regclass('public.idx_products_normalized_gtin_unique') IS NOT NULL
	`

	var productCodeColumn bool
	var ncmEx bool
	var selectiveCode bool
	var selectiveRate bool
	var targetTaxRegime bool
	var uniqueGTINIndex bool
	if err := r.db.QueryRow(ctx, query).Scan(&productCodeColumn, &ncmEx, &selectiveCode, &selectiveRate, &targetTaxRegime, &uniqueGTINIndex); err != nil {
		return catalogSchemaSupport{}, err
	}

	return catalogSchemaSupport{
		productCodeColumn: productCodeColumn,
		enhancedProfile:   ncmEx && selectiveCode && selectiveRate,
		regimeContext:     targetTaxRegime,
		uniqueGTINIndex:   uniqueGTINIndex,
	}, nil
}
