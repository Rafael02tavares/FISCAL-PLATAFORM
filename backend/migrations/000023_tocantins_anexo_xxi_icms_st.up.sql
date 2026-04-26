WITH inserted_source AS (
    INSERT INTO legal_sources (
        tax_type,
        source_type,
        jurisdiction,
        uf,
        title,
        reference_code,
        description,
        official_url,
        effective_from,
        notes
    )
    SELECT
        'icms',
        'state_regulation',
        'STATE',
        'TO',
        'RICMS/TO - Anexo XXI',
        'Decreto TO 2.912/2006 - Anexo XXI',
        'Anexo XXI do Regulamento do ICMS do Tocantins com mercadorias sujeitas a substituicao tributaria, CEST, NCM e MVA.',
        'https://dtri.sefaz.to.gov.br/legislacao/ntributaria/decretos/AnexoDec/Dec2912.06/AnexoXXI.htm',
        DATE '2026-01-12',
        'Fonte oficial SEFAZ-TO. Documento consultado para fortalecer regras estaduais de ICMS-ST por NCM/CEST.'
    WHERE NOT EXISTS (
        SELECT 1
        FROM legal_sources
        WHERE reference_code = 'Decreto TO 2.912/2006 - Anexo XXI'
           OR official_url = 'https://dtri.sefaz.to.gov.br/legislacao/ntributaria/decretos/AnexoDec/Dec2912.06/AnexoXXI.htm'
    )
    RETURNING id
),
source_row AS (
    SELECT id FROM inserted_source
    UNION ALL
    SELECT id
    FROM legal_sources
    WHERE reference_code = 'Decreto TO 2.912/2006 - Anexo XXI'
       OR official_url = 'https://dtri.sefaz.to.gov.br/legislacao/ntributaria/decretos/AnexoDec/Dec2912.06/AnexoXXI.htm'
    LIMIT 1
),
rules AS (
    SELECT
        '22030000'::text AS ncm_code,
        '0302103'::text AS cest,
        'Cervejas em venda interna de varejo no Tocantins com evidencia de ICMS-ST.'::text AS note
    UNION ALL
    SELECT
        '22029100',
        '0302201',
        'Bebidas frias nao alcoolicas em venda interna de varejo no Tocantins com evidencia de ICMS-ST.'
)
INSERT INTO legal_rule_mappings (
    legal_source_id,
    tax_type,
    operation_code,
    tax_regime,
    ncm_code,
    cest,
    cfop,
    icms_cst,
    csosn,
    emitter_uf,
    recipient_uf,
    value_type,
    value_content,
    priority,
    confidence_base,
    effective_from
)
SELECT
    source_row.id,
    'icms',
    'sale_consumer_final',
    '',
    rules.ncm_code,
    rules.cest,
    '5405',
    '60',
    '500',
    'TO',
    'TO',
    'cst_rule',
    jsonb_build_object(
        'cfop', '5405',
        'icms_cst', '60',
        'csosn', '500',
        'icms_rate', '0.0000',
        'icms_st', true,
        'uf', 'TO',
        'rule_source', 'RICMS/TO Anexo XXI',
        'notes', rules.note
    ),
    30,
    0.9400,
    DATE '2026-01-12'
FROM source_row
CROSS JOIN rules
WHERE NOT EXISTS (
    SELECT 1
    FROM legal_rule_mappings existing
    WHERE existing.legal_source_id = source_row.id
      AND existing.tax_type = 'icms'
      AND existing.operation_code = 'sale_consumer_final'
      AND existing.ncm_code = rules.ncm_code
      AND existing.cest = rules.cest
      AND existing.recipient_uf = 'TO'
      AND existing.cfop = '5405'
);

INSERT INTO state_icms_rules (
    uf,
    ncm_pattern,
    match_type,
    cest,
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
    ('TO', '22030000', 'exact', '0302103', 'sale_consumer_final', 'ST', '5405', '60', '500', 0.0000, 0.0000, 0.0000, 0.9500, 'Decreto TO 2.912/2006 - Anexo XXI', 'https://dtri.sefaz.to.gov.br/legislacao/ntributaria/decretos/AnexoDec/Dec2912.06/AnexoXXI.htm', 'Cervejas NCM 22030000 / CEST 0302103 com regra estadual de ICMS-ST para venda interna TO.'),
    ('TO', '22029100', 'exact', '0302201', 'sale_consumer_final', 'ST', '5405', '60', '500', 0.0000, 0.0000, 0.0000, 0.9500, 'Decreto TO 2.912/2006 - Anexo XXI', 'https://dtri.sefaz.to.gov.br/legislacao/ntributaria/decretos/AnexoDec/Dec2912.06/AnexoXXI.htm', 'Bebidas frias NCM 22029100 / CEST 0302201 com regra estadual de ICMS-ST para venda interna TO.')
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

UPDATE state_icms_rules
SET
    source_reference = 'Decreto TO 2.912/2006 - Anexo XXI',
    source_url = 'https://dtri.sefaz.to.gov.br/legislacao/ntributaria/decretos/AnexoDec/Dec2912.06/AnexoXXI.htm',
    notes = CASE
        WHEN ncm_pattern = '2202' THEN 'Bebidas frias com evidencia de ST em venda interna de varejo em TO. Confirmar excecoes estaduais no Anexo XXI.'
        WHEN ncm_pattern = '2203' THEN 'Cervejas com evidencia de ST em venda interna de varejo em TO. Confirmar excecoes estaduais no Anexo XXI.'
        ELSE notes
    END,
    updated_at = NOW()
WHERE uf = 'TO'
  AND rule_kind = 'ST'
  AND ncm_pattern IN ('2202', '2203');
