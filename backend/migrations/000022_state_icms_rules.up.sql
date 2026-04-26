CREATE TABLE IF NOT EXISTS state_icms_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uf VARCHAR(2) NOT NULL,
    ncm_pattern VARCHAR(8) NOT NULL DEFAULT '',
    match_type VARCHAR(16) NOT NULL DEFAULT 'prefix',
    cest VARCHAR(16) NOT NULL DEFAULT '',
    operation_code VARCHAR(80) NOT NULL DEFAULT '',
    tax_regime VARCHAR(80) NOT NULL DEFAULT '',
    target_crt VARCHAR(8) NOT NULL DEFAULT '',
    rule_kind VARCHAR(32) NOT NULL DEFAULT 'NORMAL',
    cfop VARCHAR(8) NOT NULL DEFAULT '',
    icms_cst VARCHAR(8) NOT NULL DEFAULT '',
    csosn VARCHAR(8) NOT NULL DEFAULT '',
    icms_rate NUMERIC(8,4),
    fcp_rate NUMERIC(8,4),
    icms_st_rate NUMERIC(8,4),
    icms_base_reduction NUMERIC(8,4),
    cbenef VARCHAR(32) NOT NULL DEFAULT '',
    confidence_score NUMERIC(5,4) NOT NULL DEFAULT 0.7000,
    source_reference TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    valid_from DATE NOT NULL DEFAULT DATE '2026-01-01',
    valid_to DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT state_icms_rules_match_type_chk CHECK (match_type IN ('exact', 'prefix')),
    CONSTRAINT state_icms_rules_kind_chk CHECK (rule_kind IN ('NORMAL', 'ST', 'REDUCTION', 'EXEMPT', 'NON_TAXED', 'DEFERRED'))
);

CREATE INDEX IF NOT EXISTS idx_state_icms_rules_lookup
    ON state_icms_rules (uf, ncm_pattern, match_type, cest, operation_code, tax_regime, target_crt, is_active);

CREATE INDEX IF NOT EXISTS idx_state_icms_rules_validity
    ON state_icms_rules (uf, valid_from DESC, valid_to);

CREATE UNIQUE INDEX IF NOT EXISTS idx_state_icms_rules_unique_active
    ON state_icms_rules (
        uf,
        ncm_pattern,
        match_type,
        cest,
        operation_code,
        tax_regime,
        target_crt,
        rule_kind,
        valid_from
    );

INSERT INTO state_icms_rules (
    uf,
    ncm_pattern,
    match_type,
    operation_code,
    rule_kind,
    cfop,
    icms_cst,
    csosn,
    icms_rate,
    fcp_rate,
    confidence_score,
    source_reference,
    source_url,
    notes,
    valid_from,
    valid_to
)
SELECT
    uf,
    '',
    'prefix',
    'sale_consumer_final',
    'NORMAL',
    '5102',
    '00',
    '102',
    internal_rate,
    fcp_rate,
    0.6200,
    source_reference,
    source_url,
    'Regra estadual generica para venda interna ao consumidor final. Deve ser superada por regra especifica de NCM, CEST, beneficio ou ST.',
    valid_from,
    valid_to
FROM icms_state_rates
WHERE valid_to IS NULL
ON CONFLICT (
    uf,
    ncm_pattern,
    match_type,
    cest,
    operation_code,
    tax_regime,
    target_crt,
    rule_kind,
    valid_from
) DO UPDATE
SET
    cfop = EXCLUDED.cfop,
    icms_cst = EXCLUDED.icms_cst,
    csosn = EXCLUDED.csosn,
    icms_rate = EXCLUDED.icms_rate,
    fcp_rate = EXCLUDED.fcp_rate,
    confidence_score = EXCLUDED.confidence_score,
    source_reference = EXCLUDED.source_reference,
    source_url = EXCLUDED.source_url,
    notes = EXCLUDED.notes,
    updated_at = NOW();

INSERT INTO state_icms_rules (
    uf,
    ncm_pattern,
    match_type,
    operation_code,
    rule_kind,
    cfop,
    icms_cst,
    csosn,
    icms_rate,
    fcp_rate,
    icms_st_rate,
    confidence_score,
    source_reference,
    source_url,
    notes
)
VALUES
    ('GO', '2202', 'prefix', 'sale_consumer_final', 'ST', '5405', '60', '500', 0.0000, 0.0000, 0.0000, 0.9300, 'RCTE/GO art. 34 Ap. II inc. I An. VIII', 'https://www.informanet.com.br/Prodinfo/boletim/2023/go/icms_go_03_2023.html', 'Bebidas frias com evidencia de ST em venda interna de varejo em GO. CEST ajuda, mas a regra e estadual.'),
    ('GO', '2203', 'prefix', 'sale_consumer_final', 'ST', '5405', '60', '500', 0.0000, 0.0000, 0.0000, 0.9300, 'RCTE/GO art. 34 Ap. II inc. I An. VIII', 'https://www.informanet.com.br/Prodinfo/boletim/2023/go/icms_go_03_2023.html', 'Cervejas com evidencia de ST em venda interna de varejo em GO.'),
    ('TO', '2202', 'prefix', 'sale_consumer_final', 'ST', '5405', '60', '500', 0.0000, 0.0000, 0.0000, 0.9000, 'Regra interna ICMS TO - bebidas ST', '', 'Bebidas frias com evidencia de ST em venda interna de varejo em TO. Confirmar excecoes estaduais antes de publicar regra permanente.'),
    ('TO', '2203', 'prefix', 'sale_consumer_final', 'ST', '5405', '60', '500', 0.0000, 0.0000, 0.0000, 0.9000, 'Regra interna ICMS TO - bebidas ST', '', 'Cervejas com evidencia de ST em venda interna de varejo em TO. Confirmar excecoes estaduais antes de publicar regra permanente.')
ON CONFLICT (
    uf,
    ncm_pattern,
    match_type,
    cest,
    operation_code,
    tax_regime,
    target_crt,
    rule_kind,
    valid_from
) DO UPDATE
SET
    cfop = EXCLUDED.cfop,
    icms_cst = EXCLUDED.icms_cst,
    csosn = EXCLUDED.csosn,
    icms_rate = EXCLUDED.icms_rate,
    fcp_rate = EXCLUDED.fcp_rate,
    icms_st_rate = EXCLUDED.icms_st_rate,
    confidence_score = EXCLUDED.confidence_score,
    source_reference = EXCLUDED.source_reference,
    source_url = EXCLUDED.source_url,
    notes = EXCLUDED.notes,
    updated_at = NOW();
