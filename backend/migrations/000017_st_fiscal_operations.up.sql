INSERT INTO fiscal_operations (code, name, direction, default_cfop, is_default, active)
VALUES
('sale_st_internal', 'Venda interna de mercadoria sujeita a ST', 'saida', '5405', FALSE, TRUE),
('sale_st_interstate', 'Venda interestadual de mercadoria sujeita a ST', 'saida', '6404', FALSE, TRUE),
('sale_interstate', 'Venda interestadual', 'saida', '6102', FALSE, TRUE)
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    direction = EXCLUDED.direction,
    default_cfop = EXCLUDED.default_cfop,
    active = TRUE;
