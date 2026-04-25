CREATE TABLE IF NOT EXISTS cest_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(7) NOT NULL,
    ncm_code VARCHAR(80),
    segment VARCHAR(160),
    description TEXT NOT NULL,
    legal_source VARCHAR(160),
    start_date DATE,
    end_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cest_catalog_code_ncm_unique
    ON cest_catalog(code, COALESCE(ncm_code, ''));

CREATE INDEX IF NOT EXISTS idx_cest_catalog_code
    ON cest_catalog(code);

CREATE INDEX IF NOT EXISTS idx_cest_catalog_ncm_active
    ON cest_catalog(ncm_code, is_active);

CREATE INDEX IF NOT EXISTS idx_cest_catalog_description
    ON cest_catalog USING gin(to_tsvector('portuguese', description));
