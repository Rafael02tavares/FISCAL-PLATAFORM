CREATE TABLE IF NOT EXISTS pis_cofins_revenue_natures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_code VARCHAR(16) NOT NULL,
    code VARCHAR(32) NOT NULL,
    description TEXT NOT NULL,
    source_name TEXT NOT NULL DEFAULT 'SPED Contribuicoes',
    source_file TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pis_cofins_revenue_natures_table_code
    ON pis_cofins_revenue_natures(table_code, code);

CREATE INDEX IF NOT EXISTS idx_pis_cofins_revenue_natures_code
    ON pis_cofins_revenue_natures(code);

CREATE INDEX IF NOT EXISTS idx_pis_cofins_revenue_natures_description
    ON pis_cofins_revenue_natures USING gin(to_tsvector('portuguese', description));
