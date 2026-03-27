BEGIN;

DROP TABLE IF EXISTS invoice_item_tax_decisions;

DROP TABLE IF EXISTS tax_engine_audit_steps;

DROP TABLE IF EXISTS tax_engine_runs;

DROP TABLE IF EXISTS product_classification_memory;

DROP TABLE IF EXISTS tax_rule_actions;

DROP TABLE IF EXISTS tax_rule_conditions;

DROP TABLE IF EXISTS tax_rules;

COMMIT;