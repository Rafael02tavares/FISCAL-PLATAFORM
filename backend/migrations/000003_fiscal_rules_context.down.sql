DROP INDEX IF EXISTS idx_tax_suggestions_log_operation_code;
DROP INDEX IF EXISTS idx_product_tax_profiles_icms_cst;
DROP INDEX IF EXISTS idx_product_tax_profiles_cofins_cst;
DROP INDEX IF EXISTS idx_product_tax_profiles_pis_cst;
DROP INDEX IF EXISTS idx_product_tax_profiles_operation_code;
DROP INDEX IF EXISTS idx_fiscal_operations_default;
DROP INDEX IF EXISTS idx_fiscal_operations_code;

ALTER TABLE tax_suggestions_log
DROP COLUMN IF EXISTS operation_code,
DROP COLUMN IF EXISTS cclas_trib,

DROP COLUMN IF EXISTS suggested_pis_cst,
DROP COLUMN IF EXISTS suggested_cofins_cst,
DROP COLUMN IF EXISTS suggested_pis_rate,
DROP COLUMN IF EXISTS suggested_cofins_rate,
DROP COLUMN IF EXISTS suggested_pis_revenue_code,
DROP COLUMN IF EXISTS suggested_cofins_revenue_code,

DROP COLUMN IF EXISTS suggested_icms_cst,
DROP COLUMN IF EXISTS suggested_csosn,
DROP COLUMN IF EXISTS suggested_icms_rate,
DROP COLUMN IF EXISTS suggested_icms_base_reduction,
DROP COLUMN IF EXISTS suggested_cbenef,
DROP COLUMN IF EXISTS suggested_fcp_rate,
DROP COLUMN IF EXISTS suggested_icms_st_rate,

DROP COLUMN IF EXISTS suggested_ibs_rate,
DROP COLUMN IF EXISTS suggested_cbs_rate;

ALTER TABLE product_tax_profiles
DROP COLUMN IF EXISTS operation_code,
DROP COLUMN IF EXISTS cclas_trib,

DROP COLUMN IF EXISTS pis_cst,
DROP COLUMN IF EXISTS cofins_cst,
DROP COLUMN IF EXISTS pis_rate,
DROP COLUMN IF EXISTS cofins_rate,
DROP COLUMN IF EXISTS pis_revenue_code,
DROP COLUMN IF EXISTS cofins_revenue_code,

DROP COLUMN IF EXISTS icms_cst,
DROP COLUMN IF EXISTS csosn,
DROP COLUMN IF EXISTS icms_rate,
DROP COLUMN IF EXISTS icms_base_reduction,
DROP COLUMN IF EXISTS cbenef,
DROP COLUMN IF EXISTS fcp_rate,
DROP COLUMN IF EXISTS icms_st_rate,

DROP COLUMN IF EXISTS ibs_rate,
DROP COLUMN IF EXISTS cbs_rate;

DROP TABLE IF EXISTS fiscal_operations;

ALTER TABLE organizations
DROP COLUMN IF EXISTS tax_regime,
DROP COLUMN IF EXISTS crt,
DROP COLUMN IF EXISTS state_registration,
DROP COLUMN IF EXISTS home_uf;      