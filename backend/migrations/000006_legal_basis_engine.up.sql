CREATE TABLE legal_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tax_type TEXT,
    source_type TEXT,
    jurisdiction TEXT NOT NULL,
    uf TEXT,

    title TEXT NOT NULL,
    reference_code TEXT,
    description TEXT,
    official_url TEXT,

    effective_from DATE,
    effective_to DATE,

    notes TEXT,

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT now()
);


CREATE TABLE legal_rule_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    legal_source_id UUID REFERENCES legal_sources(id),

    tax_type TEXT NOT NULL,
    operation_code TEXT,
    tax_regime TEXT,

    ncm_code TEXT,
    cest TEXT,
    cclas_trib TEXT,

    cfop TEXT,
    pis_cst TEXT,
    cofins_cst TEXT,
    icms_cst TEXT,
    csosn TEXT,

    cbenef TEXT,

    emitter_uf TEXT,
    recipient_uf TEXT,

    value_type TEXT NOT NULL,
    value_content JSONB,

    priority INT DEFAULT 100,
    confidence_base NUMERIC,

    effective_from DATE,
    effective_to DATE,

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT now()
);


CREATE TABLE tax_suggestion_legal_basis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    suggestion_id UUID,
    legal_rule_id UUID REFERENCES legal_rule_mappings(id),

    justification TEXT,

    created_at TIMESTAMP DEFAULT now()
);