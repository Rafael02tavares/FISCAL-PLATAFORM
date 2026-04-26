DELETE FROM state_icms_rules
WHERE uf = 'TO'
  AND source_reference = 'Decreto TO 2.912/2006 - Anexo XXI'
  AND ncm_pattern IN ('22030000', '22029100')
  AND cest IN ('0302103', '0302201');

DELETE FROM legal_rule_mappings
WHERE legal_source_id IN (
    SELECT id
    FROM legal_sources
    WHERE reference_code = 'Decreto TO 2.912/2006 - Anexo XXI'
)
AND tax_type = 'icms'
AND recipient_uf = 'TO'
AND ncm_code IN ('22030000', '22029100')
AND cest IN ('0302103', '0302201');

DELETE FROM legal_sources
WHERE reference_code = 'Decreto TO 2.912/2006 - Anexo XXI'
  AND NOT EXISTS (
    SELECT 1
    FROM legal_rule_mappings
    WHERE legal_rule_mappings.legal_source_id = legal_sources.id
  );
