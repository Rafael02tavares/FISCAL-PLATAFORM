BEGIN;

-- =========================================================
-- ICMS - operações internas GO, regra genérica
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'ICMS-GO-INTERNAL-GENERIC-001',
        'ICMS interno GO genérico',
        'ICMS',
        'STATE',
        'GO',
        100,
        50,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['ICMS_GO_GENERIC_INTERNAL']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'ICMS-GO-INTERNAL-GENERIC-001'
      AND tax_type = 'ICMS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_conditions (
    rule_id,
    field_name,
    operator,
    value_text,
    weight
)
SELECT rr.id, c.field_name, c.operator, c.value_text, c.weight
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('issuer_uf', 'eq', 'GO', 10),
        ('recipient_uf', 'eq', 'GO', 10),
        ('operation_scope', 'eq', 'INTERNAL', 15)
) AS c(field_name, operator, value_text, weight)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_conditions x
    WHERE x.rule_id = rr.id
      AND x.field_name = c.field_name
      AND x.operator = c.operator
      AND COALESCE(x.value_text, '') = COALESCE(c.value_text, '')
);

WITH resolved_rule AS (
    SELECT id
    FROM tax_rules
    WHERE code = 'ICMS-GO-INTERNAL-GENERIC-001'
      AND tax_type = 'ICMS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '00', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 17.00::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, TRUE),
        ('reason', 'reason', 'Aplicado ICMS interno genérico para operação interna em GO.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

-- =========================================================
-- ICMS - operações interestaduais saindo de GO, regra genérica
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'ICMS-GO-INTERSTATE-GENERIC-001',
        'ICMS interestadual GO genérico',
        'ICMS',
        'STATE',
        'GO',
        90,
        45,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['ICMS_GO_GENERIC_INTERSTATE']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'ICMS-GO-INTERSTATE-GENERIC-001'
      AND tax_type = 'ICMS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_conditions (
    rule_id,
    field_name,
    operator,
    value_text,
    weight
)
SELECT rr.id, c.field_name, c.operator, c.value_text, c.weight
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('issuer_uf', 'eq', 'GO', 10),
        ('operation_scope', 'eq', 'INTERSTATE', 15)
) AS c(field_name, operator, value_text, weight)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_conditions x
    WHERE x.rule_id = rr.id
      AND x.field_name = c.field_name
      AND x.operator = c.operator
      AND COALESCE(x.value_text, '') = COALESCE(c.value_text, '')
);

WITH resolved_rule AS (
    SELECT id
    FROM tax_rules
    WHERE code = 'ICMS-GO-INTERSTATE-GENERIC-001'
      AND tax_type = 'ICMS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '00', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 12.00::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, TRUE),
        ('reason', 'reason', 'Aplicado ICMS interestadual genérico para operações originadas em GO.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

-- =========================================================
-- PIS - regime não cumulativo genérico
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'PIS-NONCUMULATIVE-GENERIC-001',
        'PIS não cumulativo genérico',
        'PIS',
        'FEDERAL',
        NULL,
        100,
        40,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['PIS_GENERIC_NONCUMULATIVE']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'PIS-NONCUMULATIVE-GENERIC-001'
      AND tax_type = 'PIS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_conditions (
    rule_id,
    field_name,
    operator,
    value_text,
    weight
)
SELECT rr.id, c.field_name, c.operator, c.value_text, c.weight
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('crt', 'neq', '1', 10)
) AS c(field_name, operator, value_text, weight)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_conditions x
    WHERE x.rule_id = rr.id
      AND x.field_name = c.field_name
      AND x.operator = c.operator
      AND COALESCE(x.value_text, '') = COALESCE(c.value_text, '')
);

WITH resolved_rule AS (
    SELECT id
    FROM tax_rules
    WHERE code = 'PIS-NONCUMULATIVE-GENERIC-001'
      AND tax_type = 'PIS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '01', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 1.65::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, TRUE),
        ('reason', 'reason', 'Aplicado PIS genérico não cumulativo.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

-- =========================================================
-- PIS - Simples Nacional, não aplicável no cálculo padrão da engine
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'PIS-SIMPLES-GENERIC-001',
        'PIS Simples Nacional genérico',
        'PIS',
        'FEDERAL',
        NULL,
        110,
        60,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['PIS_GENERIC_SIMPLES']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'PIS-SIMPLES-GENERIC-001'
      AND tax_type = 'PIS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_conditions (
    rule_id,
    field_name,
    operator,
    value_text,
    weight
)
SELECT rr.id, c.field_name, c.operator, c.value_text, c.weight
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('crt', 'eq', '1', 20)
) AS c(field_name, operator, value_text, weight)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_conditions x
    WHERE x.rule_id = rr.id
      AND x.field_name = c.field_name
      AND x.operator = c.operator
      AND COALESCE(x.value_text, '') = COALESCE(c.value_text, '')
);

WITH resolved_rule AS (
    SELECT id
    FROM tax_rules
    WHERE code = 'PIS-SIMPLES-GENERIC-001'
      AND tax_type = 'PIS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '49', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 0.00::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, FALSE),
        ('reason', 'reason', 'PIS marcado como não aplicável no cálculo padrão para Simples Nacional na V1.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

-- =========================================================
-- COFINS - regime não cumulativo genérico
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'COFINS-NONCUMULATIVE-GENERIC-001',
        'COFINS não cumulativo genérico',
        'COFINS',
        'FEDERAL',
        NULL,
        100,
        40,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['COFINS_GENERIC_NONCUMULATIVE']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'COFINS-NONCUMULATIVE-GENERIC-001'
      AND tax_type = 'COFINS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_conditions (
    rule_id,
    field_name,
    operator,
    value_text,
    weight
)
SELECT rr.id, c.field_name, c.operator, c.value_text, c.weight
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('crt', 'neq', '1', 10)
) AS c(field_name, operator, value_text, weight)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_conditions x
    WHERE x.rule_id = rr.id
      AND x.field_name = c.field_name
      AND x.operator = c.operator
      AND COALESCE(x.value_text, '') = COALESCE(c.value_text, '')
);

WITH resolved_rule AS (
    SELECT id
    FROM tax_rules
    WHERE code = 'COFINS-NONCUMULATIVE-GENERIC-001'
      AND tax_type = 'COFINS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '01', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 7.60::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, TRUE),
        ('reason', 'reason', 'Aplicada COFINS genérica não cumulativa.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

-- =========================================================
-- COFINS - Simples Nacional, não aplicável no cálculo padrão da engine
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'COFINS-SIMPLES-GENERIC-001',
        'COFINS Simples Nacional genérico',
        'COFINS',
        'FEDERAL',
        NULL,
        110,
        60,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['COFINS_GENERIC_SIMPLES']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'COFINS-SIMPLES-GENERIC-001'
      AND tax_type = 'COFINS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_conditions (
    rule_id,
    field_name,
    operator,
    value_text,
    weight
)
SELECT rr.id, c.field_name, c.operator, c.value_text, c.weight
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('crt', 'eq', '1', 20)
) AS c(field_name, operator, value_text, weight)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_conditions x
    WHERE x.rule_id = rr.id
      AND x.field_name = c.field_name
      AND x.operator = c.operator
      AND COALESCE(x.value_text, '') = COALESCE(c.value_text, '')
);

WITH resolved_rule AS (
    SELECT id
    FROM tax_rules
    WHERE code = 'COFINS-SIMPLES-GENERIC-001'
      AND tax_type = 'COFINS'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '49', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 0.00::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, FALSE),
        ('reason', 'reason', 'COFINS marcada como não aplicável no cálculo padrão para Simples Nacional na V1.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

-- =========================================================
-- IPI - genérico não aplicado por padrão
-- =========================================================

WITH inserted_rule AS (
    INSERT INTO tax_rules (
        code,
        name,
        tax_type,
        jurisdiction_type,
        uf,
        priority,
        specificity_hint,
        valid_from,
        valid_to,
        status,
        legal_basis_ids
    )
    VALUES (
        'IPI-GENERIC-NOT-APPLIED-001',
        'IPI genérico não aplicado',
        'IPI',
        'FEDERAL',
        NULL,
        50,
        20,
        DATE '2024-01-01',
        NULL,
        'ACTIVE',
        ARRAY['IPI_GENERIC_NOT_APPLIED']
    )
    ON CONFLICT (code, tax_type, valid_from)
    DO NOTHING
    RETURNING id
),
resolved_rule AS (
    SELECT id FROM inserted_rule
    UNION
    SELECT id
    FROM tax_rules
    WHERE code = 'IPI-GENERIC-NOT-APPLIED-001'
      AND tax_type = 'IPI'
      AND valid_from = DATE '2024-01-01'
    LIMIT 1
)
INSERT INTO tax_rule_actions (
    rule_id,
    action_type,
    target_field,
    value_text,
    value_number,
    value_bool
)
SELECT rr.id, a.action_type, a.target_field, a.value_text, a.value_number, a.value_bool
FROM resolved_rule rr
CROSS JOIN (
    VALUES
        ('set', 'cst', '99', NULL::numeric, NULL::boolean),
        ('set', 'rate', NULL::text, 0.00::numeric, NULL::boolean),
        ('set', 'applies', NULL::text, NULL::numeric, FALSE),
        ('reason', 'reason', 'IPI não aplicado por padrão na seed inicial da engine.', NULL::numeric, NULL::boolean)
) AS a(action_type, target_field, value_text, value_number, value_bool)
WHERE NOT EXISTS (
    SELECT 1
    FROM tax_rule_actions x
    WHERE x.rule_id = rr.id
      AND x.action_type = a.action_type
      AND x.target_field = a.target_field
      AND COALESCE(x.value_text, '') = COALESCE(a.value_text, '')
      AND COALESCE(x.value_number, -999999) = COALESCE(a.value_number, -999999)
      AND COALESCE(x.value_bool, FALSE) = COALESCE(a.value_bool, FALSE)
);

COMMIT;