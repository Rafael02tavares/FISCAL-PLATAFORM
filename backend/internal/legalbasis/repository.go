package legalbasis

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type LegalSource struct {
	ID            string `json:"id"`
	TaxType       string `json:"tax_type"`
	SourceType    string `json:"source_type"`
	Jurisdiction  string `json:"jurisdiction"`
	UF            string `json:"uf"`
	Title         string `json:"title"`
	ReferenceCode string `json:"reference_code"`
	Description   string `json:"description"`
	OfficialURL   string `json:"official_url"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
	IsActive      bool   `json:"is_active"`
	Notes         string `json:"notes"`
}

type LegalRuleMapping struct {
	ID             string `json:"id"`
	LegalSourceID  string `json:"legal_source_id"`
	TaxType        string `json:"tax_type"`
	OperationCode  string `json:"operation_code"`
	TaxRegime      string `json:"tax_regime"`
	NCMCode        string `json:"ncm_code"`
	CEST           string `json:"cest"`
	CClasTrib      string `json:"cclas_trib"`
	CFOP           string `json:"cfop"`
	PISCST         string `json:"pis_cst"`
	COFINSCST      string `json:"cofins_cst"`
	ICMSCST        string `json:"icms_cst"`
	CSOSN          string `json:"csosn"`
	CBenef         string `json:"cbenef"`
	EmitterUF      string `json:"emitter_uf"`
	RecipientUF    string `json:"recipient_uf"`
	ValueType      string `json:"value_type"`
	ValueContent   string `json:"value_content"`
	Priority       int    `json:"priority"`
	ConfidenceBase string `json:"confidence_base"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to"`
	IsActive       bool   `json:"is_active"`
}

type CreateLegalSourceParams struct {
	TaxType       string
	SourceType    string
	Jurisdiction  string
	UF            string
	Title         string
	ReferenceCode string
	Description   string
	OfficialURL   string
	EffectiveFrom string
	EffectiveTo   string
	Notes         string
}

func (r *Repository) CreateLegalSource(ctx context.Context, p CreateLegalSourceParams) (string, error) {
	query := `
		INSERT INTO legal_sources (
			tax_type,
			source_type,
			jurisdiction,
			uf,
			title,
			reference_code,
			description,
			official_url,
			effective_from,
			effective_to,
			notes
		)
		VALUES (
			$1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8,
			NULLIF($9, '')::date,
			NULLIF($10, '')::date,
			$11
		)
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(
		ctx,
		query,
		p.TaxType,
		p.SourceType,
		p.Jurisdiction,
		p.UF,
		p.Title,
		p.ReferenceCode,
		p.Description,
		p.OfficialURL,
		p.EffectiveFrom,
		p.EffectiveTo,
		p.Notes,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create legal source: %w", err)
	}

	return id, nil
}

func (r *Repository) ListLegalSources(ctx context.Context, limit int) ([]LegalSource, error) {
	query := `
		SELECT
			id,
			COALESCE(tax_type, ''),
			COALESCE(source_type, ''),
			COALESCE(jurisdiction, ''),
			COALESCE(uf, ''),
			COALESCE(title, ''),
			COALESCE(reference_code, ''),
			COALESCE(description, ''),
			COALESCE(official_url, ''),
			COALESCE(effective_from::text, ''),
			COALESCE(effective_to::text, ''),
			is_active,
			COALESCE(notes, '')
		FROM legal_sources
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list legal sources: %w", err)
	}
	defer rows.Close()

	items := make([]LegalSource, 0)

	for rows.Next() {
		var item LegalSource
		if err := rows.Scan(
			&item.ID,
			&item.TaxType,
			&item.SourceType,
			&item.Jurisdiction,
			&item.UF,
			&item.Title,
			&item.ReferenceCode,
			&item.Description,
			&item.OfficialURL,
			&item.EffectiveFrom,
			&item.EffectiveTo,
			&item.IsActive,
			&item.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan legal source: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legal sources: %w", err)
	}

	return items, nil
}

type CreateLegalRuleMappingParams struct {
	LegalSourceID  string
	TaxType        string
	OperationCode  string
	TaxRegime      string
	NCMCode        string
	CEST           string
	CClasTrib      string
	CFOP           string
	PISCST         string
	COFINSCST      string
	ICMSCST        string
	CSOSN          string
	CBenef         string
	EmitterUF      string
	RecipientUF    string
	ValueType      string
	ValueContent   string
	Priority       int
	ConfidenceBase string
	EffectiveFrom  string
	EffectiveTo    string
}

func (r *Repository) CreateLegalRuleMapping(ctx context.Context, p CreateLegalRuleMappingParams) (string, error) {
	query := `
		INSERT INTO legal_rule_mappings (
			legal_source_id,
			tax_type,
			operation_code,
			tax_regime,
			ncm_code,
			cest,
			cclas_trib,
			cfop,
			pis_cst,
			cofins_cst,
			icms_cst,
			csosn,
			cbenef,
			emitter_uf,
			recipient_uf,
			value_type,
			value_content,
			priority,
			confidence_base,
			effective_from,
			effective_to
		)
		VALUES (
			$1::uuid,
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
			NULLIF($14, ''),
			NULLIF($15, ''),
			$16,
			$17::jsonb,
			$18,
			NULLIF($19, '')::numeric,
			NULLIF($20, '')::date,
			NULLIF($21, '')::date
		)
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(
		ctx,
		query,
		p.LegalSourceID,
		p.TaxType,
		p.OperationCode,
		p.TaxRegime,
		p.NCMCode,
		p.CEST,
		p.CClasTrib,
		p.CFOP,
		p.PISCST,
		p.COFINSCST,
		p.ICMSCST,
		p.CSOSN,
		p.CBenef,
		p.EmitterUF,
		p.RecipientUF,
		p.ValueType,
		p.ValueContent,
		p.Priority,
		p.ConfidenceBase,
		p.EffectiveFrom,
		p.EffectiveTo,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create legal rule mapping: %w", err)
	}

	return id, nil
}

func (r *Repository) ListLegalRuleMappings(ctx context.Context, limit int) ([]LegalRuleMapping, error) {
	query := `
		SELECT
			id,
			legal_source_id::text,
			COALESCE(tax_type, ''),
			COALESCE(operation_code, ''),
			COALESCE(tax_regime, ''),
			COALESCE(ncm_code, ''),
			COALESCE(cest, ''),
			COALESCE(cclas_trib, ''),
			COALESCE(cfop, ''),
			COALESCE(pis_cst, ''),
			COALESCE(cofins_cst, ''),
			COALESCE(icms_cst, ''),
			COALESCE(csosn, ''),
			COALESCE(cbenef, ''),
			COALESCE(emitter_uf, ''),
			COALESCE(recipient_uf, ''),
			COALESCE(value_type, ''),
			COALESCE(value_content::text, '{}'),
			priority,
			COALESCE(confidence_base::text, ''),
			COALESCE(effective_from::text, ''),
			COALESCE(effective_to::text, ''),
			is_active
		FROM legal_rule_mappings
		ORDER BY priority ASC, created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list legal rule mappings: %w", err)
	}
	defer rows.Close()

	items := make([]LegalRuleMapping, 0)

	for rows.Next() {
		var item LegalRuleMapping
		if err := rows.Scan(
			&item.ID,
			&item.LegalSourceID,
			&item.TaxType,
			&item.OperationCode,
			&item.TaxRegime,
			&item.NCMCode,
			&item.CEST,
			&item.CClasTrib,
			&item.CFOP,
			&item.PISCST,
			&item.COFINSCST,
			&item.ICMSCST,
			&item.CSOSN,
			&item.CBenef,
			&item.EmitterUF,
			&item.RecipientUF,
			&item.ValueType,
			&item.ValueContent,
			&item.Priority,
			&item.ConfidenceBase,
			&item.EffectiveFrom,
			&item.EffectiveTo,
			&item.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan legal rule mapping: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legal rule mappings: %w", err)
	}

	return items, nil
}

type ApplicableLegalRule struct {
	LegalSourceID  string
	TaxType        string
	Title          string
	ReferenceCode  string
	Jurisdiction   string
	UF             string
	ValueType      string
	ValueContent   string
	Priority       int
	ConfidenceBase string
}

type FindApplicableRulesParams struct {
	OperationCode string
	TaxRegime     string
	NCMCode       string
	EmitterUF     string
	RecipientUF   string
}

func (r *Repository) FindApplicableRules(ctx context.Context, p FindApplicableRulesParams) ([]ApplicableLegalRule, error) {
	query := `
		SELECT
			ls.id::text,
			COALESCE(ls.tax_type, ''),
			COALESCE(ls.title, ''),
			COALESCE(ls.reference_code, ''),
			COALESCE(ls.jurisdiction, ''),
			COALESCE(ls.uf, ''),
			COALESCE(lrm.value_type, ''),
			COALESCE(lrm.value_content::text, '{}'),
			lrm.priority,
			COALESCE(lrm.confidence_base::text, '')
		FROM legal_rule_mappings lrm
		INNER JOIN legal_sources ls ON ls.id = lrm.legal_source_id
		WHERE lrm.is_active = TRUE
		  AND ls.is_active = TRUE
		  AND (lrm.operation_code = $1 OR lrm.operation_code IS NULL OR lrm.operation_code = '')
		  AND (lrm.tax_regime = $2 OR lrm.tax_regime IS NULL OR lrm.tax_regime = '')
		  AND (lrm.ncm_code = $3 OR lrm.ncm_code IS NULL OR lrm.ncm_code = '')
		  AND (lrm.emitter_uf = $4 OR lrm.emitter_uf IS NULL OR lrm.emitter_uf = '')
		  AND (lrm.recipient_uf = $5 OR lrm.recipient_uf IS NULL OR lrm.recipient_uf = '')
		  AND (lrm.effective_from IS NULL OR lrm.effective_from <= CURRENT_DATE)
		  AND (lrm.effective_to IS NULL OR lrm.effective_to >= CURRENT_DATE)
		ORDER BY lrm.priority ASC, lrm.created_at DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		p.OperationCode,
		p.TaxRegime,
		p.NCMCode,
		p.EmitterUF,
		p.RecipientUF,
	)
	if err != nil {
		return nil, fmt.Errorf("find applicable legal rules: %w", err)
	}
	defer rows.Close()

	items := make([]ApplicableLegalRule, 0)

	for rows.Next() {
		var item ApplicableLegalRule
		if err := rows.Scan(
			&item.LegalSourceID,
			&item.TaxType,
			&item.Title,
			&item.ReferenceCode,
			&item.Jurisdiction,
			&item.UF,
			&item.ValueType,
			&item.ValueContent,
			&item.Priority,
			&item.ConfidenceBase,
		); err != nil {
			return nil, fmt.Errorf("scan applicable legal rule: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applicable legal rules: %w", err)
	}

	return items, nil
}

teste