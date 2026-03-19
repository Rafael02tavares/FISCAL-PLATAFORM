-- 000009_tax_engine_hardening.up.sql

-- 1. Limpeza preventiva de operation_code inválido antes de criar FKs
UPDATE product_tax_profiles ptp
SET operation_code = NULL
WHERE operation_code IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM fiscal_operations fo
    WHERE fo.code = ptp.operation_code
  );

UPDATE tax_suggestions_log tsl
SET operation_code = NULL
WHERE operation_code IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM fiscal_operations fo
    WHERE fo.code = tsl.operation_code
  );

UPDATE legal_rule_mappings lrm
SET operation_code = NULL
WHERE operation_code IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM fiscal_operations fo
    WHERE fo.code = lrm.operation_code
  );

-- 2. Foreign keys para operation_code
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_product_tax_profiles_operation_code'
    ) THEN
        ALTER TABLE product_tax_profiles
        ADD CONSTRAINT fk_product_tax_profiles_operation_code
        FOREIGN KEY (operation_code)
        REFERENCES fiscal_operations(code)
        ON UPDATE CASCADE
        ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_tax_suggestions_log_operation_code'
    ) THEN
        ALTER TABLE tax_suggestions_log
        ADD CONSTRAINT fk_tax_suggestions_log_operation_code
        FOREIGN KEY (operation_code)
        REFERENCES fiscal_operations(code)
        ON UPDATE CASCADE
        ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_legal_rule_mappings_operation_code'
    ) THEN
        ALTER TABLE legal_rule_mappings
        ADD CONSTRAINT fk_legal_rule_mappings_operation_code
        FOREIGN KEY (operation_code)
        REFERENCES fiscal_operations(code)
        ON UPDATE CASCADE
        ON DELETE SET NULL;
    END IF;
END $$;

-- 3. Vincular suggestion_id ao log de sugestões
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_tax_suggestion_legal_basis_suggestion'
    ) THEN
        ALTER TABLE tax_suggestion_legal_basis
        ADD CONSTRAINT fk_tax_suggestion_legal_basis_suggestion
        FOREIGN KEY (suggestion_id)
        REFERENCES tax_suggestions_log(id)
        ON DELETE CASCADE;
    END IF;
END $$;

-- 4. Índices do rule engine
CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_tax_type
    ON legal_rule_mappings (tax_type);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_operation_code
    ON legal_rule_mappings (operation_code);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_tax_regime
    ON legal_rule_mappings (tax_regime);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_ncm_code
    ON legal_rule_mappings (ncm_code);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_cest
    ON legal_rule_mappings (cest);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_ufs
    ON legal_rule_mappings (emitter_uf, recipient_uf);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_active_priority
    ON legal_rule_mappings (is_active, priority);

CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_effective_dates
    ON legal_rule_mappings (effective_from, effective_to);

-- 5. Índice composto principal para resolução de regras
CREATE INDEX IF NOT EXISTS idx_legal_rule_mappings_resolution
    ON legal_rule_mappings (
        is_active,
        tax_type,
        operation_code,
        tax_regime,
        ncm_code,
        emitter_uf,
        recipient_uf,
        priority
    );

-- 6. Índice para busca de base legal por sugestão
CREATE INDEX IF NOT EXISTS idx_tax_suggestion_legal_basis_suggestion_id
    ON tax_suggestion_legal_basis (suggestion_id);

CREATE INDEX IF NOT EXISTS idx_tax_suggestion_legal_basis_legal_rule_id
    ON tax_suggestion_legal_basis (legal_rule_id);