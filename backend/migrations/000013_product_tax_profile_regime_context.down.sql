DROP INDEX IF EXISTS idx_product_tax_profiles_observed_tax_regime;
DROP INDEX IF EXISTS idx_product_tax_profiles_target_tax_regime;

ALTER TABLE product_tax_profiles
DROP COLUMN IF EXISTS observed_crt,
DROP COLUMN IF EXISTS target_crt,
DROP COLUMN IF EXISTS observed_tax_regime,
DROP COLUMN IF EXISTS target_tax_regime;
