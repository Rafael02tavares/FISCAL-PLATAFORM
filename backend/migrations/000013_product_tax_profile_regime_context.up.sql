ALTER TABLE product_tax_profiles
ADD COLUMN IF NOT EXISTS target_tax_regime TEXT,
ADD COLUMN IF NOT EXISTS observed_tax_regime TEXT,
ADD COLUMN IF NOT EXISTS target_crt TEXT,
ADD COLUMN IF NOT EXISTS observed_crt TEXT;

CREATE INDEX IF NOT EXISTS idx_product_tax_profiles_target_tax_regime
    ON product_tax_profiles (target_tax_regime);

CREATE INDEX IF NOT EXISTS idx_product_tax_profiles_observed_tax_regime
    ON product_tax_profiles (observed_tax_regime);
