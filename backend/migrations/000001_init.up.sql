CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    cnpj TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE organization_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'owner',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, organization_id)
);

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    access_key TEXT,
    number TEXT,
    series TEXT,
    issued_at TIMESTAMP,

    emitter_name TEXT,
    emitter_cnpj TEXT,
    emitter_uf TEXT,

    recipient_name TEXT,
    recipient_cnpj TEXT,
    recipient_uf TEXT,

    operation_nature TEXT,
    total_amount NUMERIC(14,2),

    xml_raw TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'processed',

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,

    item_number INTEGER NOT NULL,
    product_code TEXT,
    gtin TEXT,
    gtin_tributable TEXT,
    description TEXT NOT NULL,

    ncm TEXT,
    cest TEXT,
    cfop TEXT,

    cst TEXT,
    csosn TEXT,

    unit TEXT,
    quantity NUMERIC(14,4),
    unit_value NUMERIC(14,4),
    total_value NUMERIC(14,2),

    icms_value NUMERIC(14,2),
    ipi_value NUMERIC(14,2),
    pis_value NUMERIC(14,2),
    cofins_value NUMERIC(14,2),

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organization_users_user_id
    ON organization_users(user_id);

CREATE INDEX idx_organization_users_organization_id
    ON organization_users(organization_id);

CREATE INDEX idx_invoices_organization_id
    ON invoices(organization_id);

CREATE INDEX idx_invoice_items_invoice_id
    ON invoice_items(invoice_id);

CREATE INDEX idx_invoice_items_gtin
    ON invoice_items(gtin);

CREATE INDEX idx_invoice_items_ncm
    ON invoice_items(ncm);