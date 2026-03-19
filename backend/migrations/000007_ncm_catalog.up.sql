CREATE TABLE ncm_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL,
    ex TEXT,
    description TEXT NOT NULL,
    ipi_rate NUMERIC,
    level INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_ncm_catalog_code_ex
    ON ncm_catalog (code, COALESCE(ex, ''));

CREATE INDEX idx_ncm_catalog_code
    ON ncm_catalog (code);

CREATE INDEX idx_ncm_catalog_level
    ON ncm_catalog (level);