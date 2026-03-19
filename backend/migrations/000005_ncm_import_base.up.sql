CREATE TABLE import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    version_label TEXT,
    file_name TEXT,
    checksum TEXT,
    imported_at TIMESTAMP NOT NULL DEFAULT NOW(),
    total_rows INTEGER NOT NULL DEFAULT 0,
    success_rows INTEGER NOT NULL DEFAULT 0,
    failed_rows INTEGER NOT NULL DEFAULT 0,
    notes TEXT
);

CREATE TABLE ncm_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    import_batch_id UUID REFERENCES import_batches(id) ON DELETE SET NULL,

    code VARCHAR(8) NOT NULL,
    description TEXT NOT NULL,
    full_description TEXT,

    chapter_code VARCHAR(2),
    heading_code VARCHAR(4),
    item_code VARCHAR(6),

    parent_code VARCHAR(8),
    level_type TEXT, -- chapter, heading, item, subitem, ncm

    ex_code VARCHAR(3),

    start_date DATE,
    end_date DATE,

    legal_source TEXT,
    legal_reference TEXT,
    official_notes TEXT,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE ncm_fiscal_enrichment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    ncm_id UUID NOT NULL REFERENCES ncm_catalog(id) ON DELETE CASCADE,

    cest TEXT,
    cclas_trib TEXT,

    default_pis_cst TEXT,
    default_cofins_cst TEXT,
    default_icms_cst TEXT,
    default_csosn TEXT,

    default_pis_revenue_code TEXT,
    default_cofins_revenue_code TEXT,

    default_ibs_rate NUMERIC(8,4),
    default_cbs_rate NUMERIC(8,4),

    default_cbenef TEXT,
    default_icms_base_reduction NUMERIC(8,4),
    default_fcp_rate NUMERIC(8,4),
    default_icms_st_rate NUMERIC(8,4),

    legal_basis_summary TEXT,
    admin_notes TEXT,

    review_status TEXT NOT NULL DEFAULT 'pending', -- pending, reviewed, approved, archived

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_ncm_catalog_code_ex_active
    ON ncm_catalog (code, COALESCE(ex_code, ''), is_active);

CREATE INDEX idx_ncm_catalog_code
    ON ncm_catalog (code);

CREATE INDEX idx_ncm_catalog_parent_code
    ON ncm_catalog (parent_code);

CREATE INDEX idx_ncm_catalog_chapter_code
    ON ncm_catalog (chapter_code);

CREATE INDEX idx_ncm_catalog_heading_code
    ON ncm_catalog (heading_code);

CREATE INDEX idx_ncm_catalog_start_date
    ON ncm_catalog (start_date);

CREATE INDEX idx_ncm_catalog_end_date
    ON ncm_catalog (end_date);

CREATE UNIQUE INDEX idx_ncm_fiscal_enrichment_ncm_id
    ON ncm_fiscal_enrichment (ncm_id);

CREATE INDEX idx_import_batches_source_name
    ON import_batches (source_name);

CREATE INDEX idx_import_batches_imported_at
    ON import_batches (imported_at);