DROP INDEX IF EXISTS idx_products_product_code;

ALTER TABLE tax_suggestions_log
DROP COLUMN IF EXISTS suggested_selective_tax_rate,
DROP COLUMN IF EXISTS suggested_selective_tax_code,
DROP COLUMN IF EXISTS suggested_ncm_ex;

ALTER TABLE product_tax_profiles
DROP COLUMN IF EXISTS selective_tax_rate,
DROP COLUMN IF EXISTS selective_tax_code,
DROP COLUMN IF EXISTS ncm_ex;

ALTER TABLE products
DROP COLUMN IF EXISTS product_code;
