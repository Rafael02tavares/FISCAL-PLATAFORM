    BEGIN;

CREATE TABLE IF NOT EXISTS tax_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    tax_type TEXT NOT NULL,
    jurisdiction_type TEXT NOT NULL,
    uf CHAR(2),
    priority INTEGER NOT NULL DEFAULT 0,
    specificity_hint INTEGER NOT NULL DEFAULT 0,
    valid_from DATE NOT NULL,
    valid_to DATE,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    legal_basis_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_rules_tax_type CHECK (
        tax_type IN ('ICMS', 'ICMS_ST', 'FCP', 'DIFAL', 'PIS', 'COFINS', 'IPI')
    ),
    CONSTRAINT chk_tax_rules_jurisdiction_type CHECK (
        jurisdiction_type IN ('FEDERAL', 'STATE')
    ),
    CONSTRAINT chk_tax_rules_status CHECK (
        status IN ('ACTIVE', 'INACTIVE', 'DRAFT')
    ),
    CONSTRAINT chk_tax_rules_uf_len CHECK (
        uf IS NULL OR LENGTH(TRIM(uf)) = 2
    ),
    CONSTRAINT chk_tax_rules_valid_range CHECK (
        valid_to IS NULL OR valid_to >= valid_from
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_tax_rules_code_tax_type_valid_from
    ON tax_rules (code, tax_type, valid_from);

CREATE INDEX IF NOT EXISTS idx_tax_rules_lookup
    ON tax_rules (tax_type, jurisdiction_type, status, valid_from, valid_to);

CREATE INDEX IF NOT EXISTS idx_tax_rules_uf
    ON tax_rules (uf);

CREATE TABLE IF NOT EXISTS tax_rule_conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES tax_rules(id) ON DELETE CASCADE,
    field_name TEXT NOT NULL,
    operator TEXT NOT NULL,
    value_text TEXT,
    value_number NUMERIC(18,6),
    value_bool BOOLEAN,
    value_list TEXT[],
    value_min NUMERIC(18,6),
    value_max NUMERIC(18,6),
    weight INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_rule_conditions_operator CHECK (
        operator IN (
            'eq',
            'neq',
            'in',
            'not_in',
            'prefix',
            'not_prefix',
            'contains',
            'is_true',
            'is_false'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_tax_rule_conditions_rule_id
    ON tax_rule_conditions (rule_id);

CREATE INDEX IF NOT EXISTS idx_tax_rule_conditions_field_name
    ON tax_rule_conditions (field_name);

CREATE TABLE IF NOT EXISTS tax_rule_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES tax_rules(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL,
    target_field TEXT NOT NULL,
    value_text TEXT,
    value_number NUMERIC(18,6),
    value_bool BOOLEAN,
    value_json JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_rule_actions_action_type CHECK (
        action_type IN ('set', 'append', 'reason')
    )
);

CREATE INDEX IF NOT EXISTS idx_tax_rule_actions_rule_id
    ON tax_rule_actions (rule_id);

CREATE INDEX IF NOT EXISTS idx_tax_rule_actions_target_field
    ON tax_rule_actions (target_field);

CREATE TABLE IF NOT EXISTS product_classification_memory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    supplier_id TEXT,
    supplier_product_code TEXT,
    gtin TEXT,
    description_normalized TEXT,
    ncm TEXT NOT NULL,
    extipi TEXT,
    cest TEXT,
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'FALLBACK',
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_product_classification_memory_confidence CHECK (
        confidence >= 0 AND confidence <= 1
    ),
    CONSTRAINT chk_product_classification_memory_source CHECK (
        source IN (
            'XML',
            'GTIN_MEMORY',
            'SUPPLIER_MEMORY',
            'DESCRIPTION_MATCH',
            'MANUAL_RULE',
            'FALLBACK'
        )
    ),
    CONSTRAINT chk_product_classification_memory_ncm_len CHECK (
        LENGTH(TRIM(ncm)) = 8
    ),
    CONSTRAINT chk_product_classification_memory_any_key CHECK (
        COALESCE(NULLIF(TRIM(gtin), ''), NULLIF(TRIM(description_normalized), '')) IS NOT NULL
        OR (
            NULLIF(TRIM(supplier_id), '') IS NOT NULL
            AND NULLIF(TRIM(supplier_product_code), '') IS NOT NULL
        )
    )
);

ALTER TABLE product_classification_memory
    DROP CONSTRAINT IF EXISTS uq_product_classification_memory_lookup;

ALTER TABLE product_classification_memory
    ADD CONSTRAINT uq_product_classification_memory_lookup
    UNIQUE NULLS NOT DISTINCT (
        tenant_id,
        organization_id,
        supplier_id,
        supplier_product_code,
        gtin,
        description_normalized
    );

CREATE INDEX IF NOT EXISTS idx_product_classification_memory_gtin
    ON product_classification_memory (tenant_id, organization_id, gtin);

CREATE INDEX IF NOT EXISTS idx_product_classification_memory_supplier
    ON product_classification_memory (tenant_id, organization_id, supplier_id, supplier_product_code);

CREATE INDEX IF NOT EXISTS idx_product_classification_memory_description
    ON product_classification_memory (tenant_id, organization_id, description_normalized);

CREATE TABLE IF NOT EXISTS tax_engine_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    invoice_id TEXT,
    invoice_item_id TEXT,
    input_payload JSONB,
    normalized_payload JSONB,
    classification_json JSONB,
    evaluation_json JSONB,
    taxes_json JSONB,
    output_payload JSONB,
    status TEXT NOT NULL DEFAULT 'SUCCESS',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_engine_runs_status CHECK (
        status IN ('SUCCESS', 'SUGGESTED', 'MANUAL_REVIEW_REQUIRED', 'BLOCKED', 'FAILED')
    )
);

CREATE INDEX IF NOT EXISTS idx_tax_engine_runs_org_created_at
    ON tax_engine_runs (tenant_id, organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tax_engine_runs_invoice_id
    ON tax_engine_runs (invoice_id);

CREATE INDEX IF NOT EXISTS idx_tax_engine_runs_invoice_item_id
    ON tax_engine_runs (invoice_item_id);

CREATE TABLE IF NOT EXISTS tax_engine_audit_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES tax_engine_runs(id) ON DELETE CASCADE,
    step_order INTEGER NOT NULL,
    step_name TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    matched_rule_id TEXT,
    payload_json JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_engine_audit_steps_status CHECK (
        status IN ('SUCCESS', 'SKIPPED', 'WARNING', 'FAILED')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_tax_engine_audit_steps_run_order
    ON tax_engine_audit_steps (run_id, step_order);

CREATE INDEX IF NOT EXISTS idx_tax_engine_audit_steps_run_id
    ON tax_engine_audit_steps (run_id);

CREATE TABLE IF NOT EXISTS invoice_item_tax_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    invoice_id TEXT NOT NULL,
    invoice_item_id TEXT NOT NULL,

    classification_ncm TEXT,
    classification_extipi TEXT,
    classification_cest TEXT,
    classification_source TEXT,
    classification_confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    needs_review BOOLEAN NOT NULL DEFAULT FALSE,
    summary_status TEXT,

    icms_base_value NUMERIC(18,2),
    icms_rate NUMERIC(10,4),
    icms_amount NUMERIC(18,2),
    icms_cst TEXT,
    icms_rule_id TEXT,

    icms_st_base_value NUMERIC(18,2),
    icms_st_rate NUMERIC(10,4),
    icms_st_amount NUMERIC(18,2),
    icms_st_cst TEXT,
    icms_st_rule_id TEXT,

    fcp_base_value NUMERIC(18,2),
    fcp_rate NUMERIC(10,4),
    fcp_amount NUMERIC(18,2),
    fcp_rule_id TEXT,

    difal_base_value NUMERIC(18,2),
    difal_internal_rate NUMERIC(10,4),
    difal_interstate_rate NUMERIC(10,4),
    difal_amount_destination NUMERIC(18,2),
    difal_amount_origin NUMERIC(18,2),
    difal_rule_id TEXT,

    pis_base_value NUMERIC(18,2),
    pis_rate NUMERIC(10,4),
    pis_amount NUMERIC(18,2),
    pis_cst TEXT,
    pis_rule_id TEXT,

    cofins_base_value NUMERIC(18,2),
    cofins_rate NUMERIC(10,4),
    cofins_amount NUMERIC(18,2),
    cofins_cst TEXT,
    cofins_rule_id TEXT,

    ipi_base_value NUMERIC(18,2),
    ipi_rate NUMERIC(10,4),
    ipi_amount NUMERIC(18,2),
    ipi_cst TEXT,
    ipi_rule_id TEXT,

    warnings_json JSONB,
    explanations_json JSONB,
    audit_trail_json JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_invoice_item_tax_decisions_invoice_item UNIQUE (invoice_item_id),
    CONSTRAINT chk_invoice_item_tax_decisions_confidence CHECK (
        classification_confidence >= 0 AND classification_confidence <= 1
    )
);

CREATE INDEX IF NOT EXISTS idx_invoice_item_tax_decisions_invoice_id
    ON invoice_item_tax_decisions (invoice_id);

CREATE INDEX IF NOT EXISTS idx_invoice_item_tax_decisions_org
    ON invoice_item_tax_decisions (tenant_id, organization_id);

CREATE INDEX IF NOT EXISTS idx_invoice_item_tax_decisions_summary_status
    ON invoice_item_tax_decisions (summary_status);

COMMIT;