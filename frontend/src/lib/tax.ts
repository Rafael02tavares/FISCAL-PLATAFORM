import { apiFetch } from "./api";

export async function suggestTax(payload: {
  gtin: string
  description: string
  ncm_code?: string
  operation_code?: string
  emitter_uf?: string
  recipient_uf?: string
  tax_regime?: string
  target_crt?: string
  source_icms_cst?: string
  source_icms_csosn?: string
  source_icms_rate?: string
  source_pis_cst?: string
  source_pis_rate?: string
  source_cofins_cst?: string
  source_cofins_rate?: string
  source_cfop?: string
}) {
  return apiFetch("/tax/suggest", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
