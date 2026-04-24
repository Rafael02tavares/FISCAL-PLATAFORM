CREATE TABLE IF NOT EXISTS icms_state_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uf VARCHAR(2) NOT NULL,
    internal_rate NUMERIC(5,2) NOT NULL,
    fcp_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    valid_from DATE NOT NULL,
    valid_to DATE,
    source_reference TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_icms_state_rates_uf_validity
    ON icms_state_rates (uf, valid_from DESC, valid_to);

CREATE UNIQUE INDEX IF NOT EXISTS idx_icms_state_rates_uf_valid_from
    ON icms_state_rates (uf, valid_from);

INSERT INTO icms_state_rates (uf, internal_rate, fcp_rate, valid_from, valid_to, source_reference, source_url, notes)
VALUES
    ('AC', 19.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('AL', 19.00, 0, DATE '2026-01-01', DATE '2026-03-31', 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota anterior mantida ate 31/03/2026, antes da vigencia da Lei Estadual 9.776/2025.'),
    ('AL', 20.50, 0, DATE '2026-04-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota a partir de 01/04/2026 conforme Lei Estadual 9.776/2025.'),
    ('AM', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('AP', 18.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('BA', 20.50, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('CE', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('DF', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('ES', 17.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('GO', 19.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('MA', 23.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('MG', 18.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('MS', 17.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('MT', 17.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('PA', 19.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('PB', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('PE', 20.50, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('PI', 22.50, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('PR', 19.50, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('RJ', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('RN', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('RO', 19.50, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('RR', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('RS', 17.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('SC', 17.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('SE', 19.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('SP', 18.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.'),
    ('TO', 20.00, 0, DATE '2026-01-01', NULL, 'Conta Azul - Tabela ICMS 2026', 'https://contaazul.com/blog/tabela-de-aliquota-interestadual/', 'Aliquota interna informada no guia operacional de 2026.')
ON CONFLICT (uf, valid_from) DO UPDATE
SET
    internal_rate = EXCLUDED.internal_rate,
    fcp_rate = EXCLUDED.fcp_rate,
    valid_to = EXCLUDED.valid_to,
    source_reference = EXCLUDED.source_reference,
    source_url = EXCLUDED.source_url,
    notes = EXCLUDED.notes,
    updated_at = NOW();
