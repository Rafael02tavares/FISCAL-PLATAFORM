CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gtin TEXT,
    normalized_gtin TEXT,
    description TEXT NOT NULL,
    normalized_description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE product_tax_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,

    source_invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    source_invoice_item_id UUID REFERENCES invoice_items(id) ON DELETE SET NULL,

    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,

    ncm TEXT,
    cest TEXT,
    cfop TEXT,

    icms_value NUMERIC(14,2),
    ipi_value NUMERIC(14,2),
    pis_value NUMERIC(14,2),
    cofins_value NUMERIC(14,2),

    emitter_uf TEXT,
    recipient_uf TEXT,
    operation_nature TEXT,

    confidence_score NUMERIC(5,2) DEFAULT 0,
    source_type TEXT NOT NULL DEFAULT 'invoice_import',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE tax_suggestions_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,

    gtin TEXT,
    description TEXT,

    suggested_ncm TEXT,
    suggested_cest TEXT,
    suggested_cfop TEXT,

    suggested_icms_value NUMERIC(14,2),
    suggested_ipi_value NUMERIC(14,2),
    suggested_pis_value NUMERIC(14,2),
    suggested_cofins_value NUMERIC(14,2),

    match_type TEXT,
    confidence_score NUMERIC(5,2),

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_normalized_gtin
    ON products(normalized_gtin);

CREATE INDEX idx_products_normalized_description
    ON products(normalized_description);

CREATE INDEX idx_product_tax_profiles_product_id
    ON product_tax_profiles(product_id);

CREATE INDEX idx_product_tax_profiles_org_id
    ON product_tax_profiles(organization_id);

CREATE INDEX idx_tax_suggestions_log_org_id
    ON tax_suggestions_log(organization_id);