-- 000009_tax_engine_hardening.down.sql

DROP INDEX IF EXISTS idx_tax_suggestion_legal_basis_legal_rule_id;
DROP INDEX IF EXISTS idx_tax_suggestion_legal_basis_suggestion_id;

DROP INDEX IF EXISTS idx_legal_rule_mappings_resolution;
DROP INDEX IF EXISTS idx_legal_rule_mappings_effective_dates;
DROP INDEX IF EXISTS idx_legal_rule_mappings_active_priority;
DROP INDEX IF EXISTS idx_legal_rule_mappings_ufs;
DROP INDEX IF EXISTS idx_legal_rule_mappings_cest;
DROP INDEX IF EXISTS idx_legal_rule_mappings_ncm_code;
DROP INDEX IF EXISTS idx_legal_rule_mappings_tax_regime;
DROP INDEX IF EXISTS idx_legal_rule_mappings_operation_code;
DROP INDEX IF EXISTS idx_legal_rule_mappings_tax_type;

ALTER TABLE tax_suggestion_legal_basis
DROP CONSTRAINT IF EXISTS fk_tax_suggestion_legal_basis_suggestion;

ALTER TABLE legal_rule_mappings
DROP CONSTRAINT IF EXISTS fk_legal_rule_mappings_operation_code;

ALTER TABLE tax_suggestions_log
DROP CONSTRAINT IF EXISTS fk_tax_suggestions_log_operation_code;

ALTER TABLE product_tax_profiles
DROP CONSTRAINT IF EXISTS fk_product_tax_profiles_operation_code;