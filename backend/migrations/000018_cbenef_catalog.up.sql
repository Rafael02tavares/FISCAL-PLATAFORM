CREATE TABLE IF NOT EXISTS cbenef_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uf CHAR(2) NOT NULL,
    code VARCHAR(24) NOT NULL,
    applies_simples BOOLEAN NOT NULL DEFAULT FALSE,
    applicable_csts JSONB NOT NULL DEFAULT '[]'::jsonb,
    legal_device TEXT,
    description TEXT NOT NULL,
    notes TEXT,
    source_name VARCHAR(160),
    version_label VARCHAR(80),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cbenef_catalog_uf_code_unique
    ON cbenef_catalog(uf, code);

CREATE INDEX IF NOT EXISTS idx_cbenef_catalog_uf_active
    ON cbenef_catalog(uf, is_active);

CREATE INDEX IF NOT EXISTS idx_cbenef_catalog_code
    ON cbenef_catalog(code);

CREATE INDEX IF NOT EXISTS idx_cbenef_catalog_csts
    ON cbenef_catalog USING gin(applicable_csts);

CREATE INDEX IF NOT EXISTS idx_cbenef_catalog_description
    ON cbenef_catalog USING gin(to_tsvector('portuguese', description));
