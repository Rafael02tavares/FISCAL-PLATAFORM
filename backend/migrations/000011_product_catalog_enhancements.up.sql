ALTER TABLE products
ADD COLUMN IF NOT EXISTS product_code TEXT;

ALTER TABLE product_tax_profiles
ADD COLUMN IF NOT EXISTS ncm_ex TEXT,
ADD COLUMN IF NOT EXISTS selective_tax_code TEXT,
ADD COLUMN IF NOT EXISTS selective_tax_rate NUMERIC(8,4);

ALTER TABLE tax_suggestions_log
ADD COLUMN IF NOT EXISTS suggested_ncm_ex TEXT,
ADD COLUMN IF NOT EXISTS suggested_selective_tax_code TEXT,
ADD COLUMN IF NOT EXISTS suggested_selective_tax_rate NUMERIC(8,4);

CREATE INDEX IF NOT EXISTS idx_products_product_code
    ON products(product_code);
