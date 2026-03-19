CREATE TABLE cfop_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    code VARCHAR(4) NOT NULL,
    description TEXT NOT NULL,

    ind_nfe BOOLEAN DEFAULT TRUE,
    ind_comunication BOOLEAN DEFAULT FALSE,
    ind_transport BOOLEAN DEFAULT FALSE,
    ind_devolution BOOLEAN DEFAULT FALSE,

    operation_type VARCHAR(20),

    created_at TIMESTAMP NOT NULL DEFAULT now(),

    UNIQUE(code)
);