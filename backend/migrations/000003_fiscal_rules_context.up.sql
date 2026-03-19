ALTER TABLE organizations
ADD COLUMN tax_regime TEXT,
ADD COLUMN crt TEXT,
ADD COLUMN state_registration TEXT,
ADD COLUMN home_uf TEXT;

CREATE TABLE fiscal_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    direction TEXT NOT NULL,
    default_cfop TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO fiscal_operations (code, name, direction, default_cfop, is_default)
VALUES
('sale_consumer_final', 'Venda a consumidor final', 'outbound', '5102', TRUE),
('sale_taxpayer', 'Venda para contribuinte', 'outbound', '5102', FALSE),
('purchase_resale', 'Compra para revenda', 'inbound', '1102', FALSE),
('purchase_consumption', 'Compra para consumo', 'inbound', '1556', FALSE),
('transfer_out', 'Transferência de saída', 'outbound', '5152', FALSE),
('transfer_in', 'Transferência de entrada', 'inbound', '1152', FALSE),
('return_sale', 'Devolução de venda', 'inbound', '1202', FALSE),
('return_purchase', 'Devolução de compra', 'outbound', '5202', FALSE),
('bonus_out', 'Bonificação de saída', 'outbound', '5910', FALSE),
('remittance', 'Remessa', 'outbound', '5949', FALSE),
('return_remittance', 'Retorno de remessa', 'inbound', '1949', FALSE);

ALTER TABLE product_tax_profiles
ADD COLUMN operation_code TEXT,
ADD COLUMN cclas_trib TEXT,

ADD COLUMN pis_cst TEXT,
ADD COLUMN cofins_cst TEXT,
ADD COLUMN pis_rate NUMERIC(8,4),
ADD COLUMN cofins_rate NUMERIC(8,4),
ADD COLUMN pis_revenue_code TEXT,
ADD COLUMN cofins_revenue_code TEXT,

ADD COLUMN icms_cst TEXT,
ADD COLUMN csosn TEXT,
ADD COLUMN icms_rate NUMERIC(8,4),
ADD COLUMN icms_base_reduction NUMERIC(8,4),
ADD COLUMN cbenef TEXT,
ADD COLUMN fcp_rate NUMERIC(8,4),
ADD COLUMN icms_st_rate NUMERIC(8,4),

ADD COLUMN ibs_rate NUMERIC(8,4),
ADD COLUMN cbs_rate NUMERIC(8,4);

ALTER TABLE tax_suggestions_log
ADD COLUMN operation_code TEXT,
ADD COLUMN cclas_trib TEXT,

ADD COLUMN suggested_pis_cst TEXT,
ADD COLUMN suggested_cofins_cst TEXT,
ADD COLUMN suggested_pis_rate NUMERIC(8,4),
ADD COLUMN suggested_cofins_rate NUMERIC(8,4),
ADD COLUMN suggested_pis_revenue_code TEXT,
ADD COLUMN suggested_cofins_revenue_code TEXT,

ADD COLUMN suggested_icms_cst TEXT,
ADD COLUMN suggested_csosn TEXT,
ADD COLUMN suggested_icms_rate NUMERIC(8,4),
ADD COLUMN suggested_icms_base_reduction NUMERIC(8,4),
ADD COLUMN suggested_cbenef TEXT,
ADD COLUMN suggested_fcp_rate NUMERIC(8,4),
ADD COLUMN suggested_icms_st_rate NUMERIC(8,4),

ADD COLUMN suggested_ibs_rate NUMERIC(8,4),
ADD COLUMN suggested_cbs_rate NUMERIC(8,4);

CREATE INDEX idx_fiscal_operations_code
    ON fiscal_operations(code);

CREATE INDEX idx_fiscal_operations_default
    ON fiscal_operations(is_default);

CREATE INDEX idx_product_tax_profiles_operation_code
    ON product_tax_profiles(operation_code);

CREATE INDEX idx_product_tax_profiles_pis_cst
    ON product_tax_profiles(pis_cst);

CREATE INDEX idx_product_tax_profiles_cofins_cst
    ON product_tax_profiles(cofins_cst);

CREATE INDEX idx_product_tax_profiles_icms_cst
    ON product_tax_profiles(icms_cst);

CREATE INDEX idx_tax_suggestions_log_operation_code
    ON tax_suggestions_log(operation_code);