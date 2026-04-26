CREATE TABLE IF NOT EXISTS ncm_tax_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ncm_pattern VARCHAR(8) NOT NULL,
    match_type VARCHAR(16) NOT NULL DEFAULT 'prefix',
    tax_type VARCHAR(32) NOT NULL,
    tax_group VARCHAR(64) NOT NULL DEFAULT '',
    uf VARCHAR(2),
    operation_code VARCHAR(80) NOT NULL DEFAULT '',
    tax_regime VARCHAR(80) NOT NULL DEFAULT '',
    target_crt VARCHAR(8) NOT NULL DEFAULT '',

    cest VARCHAR(16) NOT NULL DEFAULT '',
    cfop VARCHAR(8) NOT NULL DEFAULT '',
    icms_cst VARCHAR(8) NOT NULL DEFAULT '',
    csosn VARCHAR(8) NOT NULL DEFAULT '',
    pis_cst VARCHAR(8) NOT NULL DEFAULT '',
    cofins_cst VARCHAR(8) NOT NULL DEFAULT '',
    pis_revenue_code VARCHAR(32) NOT NULL DEFAULT '',
    cofins_revenue_code VARCHAR(32) NOT NULL DEFAULT '',
    cclas_trib VARCHAR(16) NOT NULL DEFAULT '',

    icms_rate NUMERIC(8,4),
    pis_rate NUMERIC(8,4),
    cofins_rate NUMERIC(8,4),
    ipi_rate NUMERIC(8,4),
    fcp_rate NUMERIC(8,4),
    icms_st_rate NUMERIC(8,4),
    ibs_rate NUMERIC(8,4),
    cbs_rate NUMERIC(8,4),
    selective_tax_rate NUMERIC(8,4),

    ipi_cst VARCHAR(8) NOT NULL DEFAULT '',
    ipi_cenq VARCHAR(16) NOT NULL DEFAULT '',
    selective_tax_code VARCHAR(16) NOT NULL DEFAULT '',

    confidence_score NUMERIC(5,4) NOT NULL DEFAULT 0.75,
    source_reference TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    valid_from DATE NOT NULL DEFAULT DATE '2026-01-01',
    valid_to DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_ncm_tax_profiles_match_type
        CHECK (match_type IN ('exact', 'prefix')),
    CONSTRAINT chk_ncm_tax_profiles_tax_type
        CHECK (tax_type IN ('ICMS', 'ICMS_ST', 'PIS_COFINS', 'IPI', 'IBS_CBS', 'SELECTIVE_TAX'))
);

CREATE INDEX IF NOT EXISTS idx_ncm_tax_profiles_lookup
    ON ncm_tax_profiles (is_active, tax_type, uf, ncm_pattern, match_type);

CREATE INDEX IF NOT EXISTS idx_ncm_tax_profiles_context
    ON ncm_tax_profiles (operation_code, tax_regime, target_crt, valid_from, valid_to);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ncm_tax_profiles_unique_context
    ON ncm_tax_profiles (
        ncm_pattern,
        match_type,
        tax_type,
        COALESCE(uf, ''),
        operation_code,
        tax_regime,
        target_crt,
        valid_from
    );

INSERT INTO ncm_tax_profiles (
    ncm_pattern, match_type, tax_type, tax_group, operation_code,
    pis_cst, cofins_cst, pis_rate, cofins_rate,
    confidence_score, source_reference, notes
)
VALUES
    ('2202', 'prefix', 'PIS_COFINS', 'bebidas_monofasicas', 'sale_consumer_final', '04', '04', 0, 0, 0.88, 'Lei 13.097/2015; Decreto 8.442/2015', 'Bebidas frias no varejo: tributacao concentrada na cadeia anterior; saida varejista tende a CST 04 com aliquota zero.'),
    ('2203', 'prefix', 'PIS_COFINS', 'bebidas_monofasicas', 'sale_consumer_final', '04', '04', 0, 0, 0.90, 'Lei 13.097/2015; Decreto 8.442/2015', 'Cervejas no varejo: regime monofasico de PIS/COFINS, com aliquota zero na revenda.'),
    ('2204', 'prefix', 'PIS_COFINS', 'bebidas_monofasicas', 'sale_consumer_final', '04', '04', 0, 0, 0.82, 'Lei 13.097/2015; Decreto 8.442/2015', 'Bebidas alcoolicas: perfil federal monofasico inicial para triagem.'),
    ('2205', 'prefix', 'PIS_COFINS', 'bebidas_monofasicas', 'sale_consumer_final', '04', '04', 0, 0, 0.82, 'Lei 13.097/2015; Decreto 8.442/2015', 'Bebidas alcoolicas: perfil federal monofasico inicial para triagem.'),
    ('2206', 'prefix', 'PIS_COFINS', 'bebidas_monofasicas', 'sale_consumer_final', '04', '04', 0, 0, 0.82, 'Lei 13.097/2015; Decreto 8.442/2015', 'Bebidas fermentadas: perfil federal monofasico inicial para triagem.'),
    ('2208', 'prefix', 'PIS_COFINS', 'bebidas_monofasicas', 'sale_consumer_final', '04', '04', 0, 0, 0.82, 'Lei 13.097/2015; Decreto 8.442/2015', 'Bebidas destiladas: perfil federal monofasico inicial para triagem.'),
    ('2402', 'prefix', 'PIS_COFINS', 'cigarros_monofasicos', 'sale_consumer_final', '04', '04', 0, 0, 0.82, 'Regime monofasico PIS/COFINS', 'Cigarros e similares com tributacao concentrada na cadeia anterior.'),
    ('2710', 'prefix', 'PIS_COFINS', 'combustiveis_monofasicos', 'sale_consumer_final', '04', '04', 0, 0, 0.80, 'Regime monofasico PIS/COFINS', 'Combustiveis com tratamento federal concentrado; validar excecoes por produto.'),
    ('2711', 'prefix', 'PIS_COFINS', 'combustiveis_monofasicos', 'sale_consumer_final', '04', '04', 0, 0, 0.80, 'Regime monofasico PIS/COFINS', 'GLP e gases com tratamento federal concentrado; validar excecoes por produto.')
ON CONFLICT DO NOTHING;

INSERT INTO ncm_tax_profiles (
    ncm_pattern, match_type, tax_type, tax_group, uf, operation_code,
    cfop, icms_cst, csosn, icms_rate, confidence_score,
    source_reference, notes
)
VALUES
    ('2202', 'prefix', 'ICMS_ST', 'bebidas_st', 'GO', 'sale_consumer_final', '5405', '60', '500', 0, 0.88, 'RCTE/GO - bebidas sujeitas a ST', 'Regra operacional inicial para bebidas em venda interna GO; CEST apoia a identificacao, mas a decisao vem desta relacao NCM/UF.'),
    ('2203', 'prefix', 'ICMS_ST', 'bebidas_st', 'GO', 'sale_consumer_final', '5405', '60', '500', 0, 0.90, 'RCTE/GO - bebidas sujeitas a ST', 'Regra operacional inicial para cervejas em venda interna GO.'),
    ('2202', 'prefix', 'ICMS_ST', 'bebidas_st', 'TO', 'sale_consumer_final', '5405', '60', '500', 0, 0.86, 'Perfil interno validado por captura de NF-e', 'Regra operacional inicial para bebidas em venda interna TO; revisar excecoes estaduais antes de publicar como regra legal definitiva.'),
    ('2203', 'prefix', 'ICMS_ST', 'bebidas_st', 'TO', 'sale_consumer_final', '5405', '60', '500', 0, 0.90, 'Perfil interno validado por captura de NF-e', 'Regra operacional inicial para cervejas em venda interna TO.')
ON CONFLICT DO NOTHING;

INSERT INTO ncm_tax_profiles (
    ncm_pattern, match_type, tax_type, tax_group, operation_code,
    cclas_trib, ibs_rate, cbs_rate, confidence_score,
    source_reference, notes
)
VALUES
    ('', 'prefix', 'IBS_CBS', 'regra_transitoria_2026', 'sale_consumer_final', '000001', 0.1000, 0.9000, 0.72, 'LC 214/2025 arts. 343, 344 e 346', 'Perfil transitorio padrao para operacoes tributadas integralmente em 2026, usado quando nao houver regra especifica de IBS/CBS.')
ON CONFLICT DO NOTHING;
