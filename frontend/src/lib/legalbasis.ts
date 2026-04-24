import { apiFetch } from "./api";

export type LegalSource = {
  id: string;
  tax_type: string;
  source_type: string;
  jurisdiction: string;
  uf: string;
  title: string;
  reference_code: string;
  description: string;
  official_url: string;
  effective_from: string;
  effective_to: string;
  is_active: boolean;
  notes: string;
};

export type LegalRule = {
  id: string;
  legal_source_id: string;
  tax_type: string;
  operation_code: string;
  tax_regime: string;
  ncm_code: string;
  cest: string;
  cclas_trib: string;
  cfop: string;
  pis_cst: string;
  cofins_cst: string;
  icms_cst: string;
  csosn: string;
  cbenef: string;
  emitter_uf: string;
  recipient_uf: string;
  value_type: string;
  value_content: string;
  priority: number;
  confidence_base: string;
  effective_from: string;
  effective_to: string;
  is_active: boolean;
};

export type CreateLegalRulePayload = {
  legal_source_id: string;
  tax_type: string;
  operation_code: string;
  tax_regime: string;
  ncm_code: string;
  cest: string;
  cclas_trib: string;
  cfop: string;
  pis_cst: string;
  cofins_cst: string;
  icms_cst: string;
  csosn: string;
  cbenef: string;
  emitter_uf: string;
  recipient_uf: string;
  value_type: string;
  value_content: string;
  priority: number;
  confidence_base: string;
  effective_from: string;
  effective_to: string;
};

export async function listLegalSources(limit = 100) {
  return apiFetch<{ items: LegalSource[] }>(`/legal-sources?limit=${limit}`, {
    method: "GET",
  });
}

export async function listLegalRules(limit = 100) {
  return apiFetch<{ items: LegalRule[] }>(`/legal-rules?limit=${limit}`, {
    method: "GET",
  });
}

export async function createLegalRule(payload: CreateLegalRulePayload) {
  return apiFetch<{ id: string }>("/legal-rules", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
