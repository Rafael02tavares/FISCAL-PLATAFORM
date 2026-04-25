import { apiFetch } from "./api";

export type CatalogProductProfile = {
  id: string;
  product_id: string;
  organization_id: string;
  source_invoice_id?: string;
  ncm?: string;
  ncm_ex?: string;
  cest?: string;
  cfop?: string;
  cclas_trib?: string;
  pis_cst?: string;
  cofins_cst?: string;
  pis_revenue_code?: string;
  cofins_revenue_code?: string;
  icms_cst?: string;
  csosn?: string;
  cbenef?: string;
  icms_value?: string;
  ipi_value?: string;
  pis_value?: string;
  cofins_value?: string;
  pis_rate?: string;
  cofins_rate?: string;
  icms_rate?: string;
  icms_base_reduction?: string;
  fcp_rate?: string;
  icms_st_rate?: string;
  ibs_rate?: string;
  cbs_rate?: string;
  selective_tax_code?: string;
  selective_tax_rate?: string;
  operation_code?: string;
  emitter_uf?: string;
  recipient_uf?: string;
  operation_nature?: string;
  target_tax_regime?: string;
  observed_tax_regime?: string;
  target_crt?: string;
  observed_crt?: string;
  confidence_score?: number;
  source_type?: string;
};

export type CatalogProductItem = {
  id: string;
  product_code?: string;
  gtin?: string;
  description?: string;
  profile: CatalogProductProfile;
};

export type SaveCatalogProductPayload = {
  product_id?: string;
  product_code?: string;
  gtin?: string;
  description: string;
  ncm?: string;
  ncm_ex?: string;
  cest?: string;
  cfop?: string;
  cclas_trib?: string;
  pis_cst?: string;
  cofins_cst?: string;
  pis_revenue_code?: string;
  cofins_revenue_code?: string;
  icms_cst?: string;
  csosn?: string;
  cbenef?: string;
  icms_value?: string;
  ipi_value?: string;
  pis_value?: string;
  cofins_value?: string;
  pis_rate?: string;
  cofins_rate?: string;
  icms_rate?: string;
  icms_base_reduction?: string;
  fcp_rate?: string;
  icms_st_rate?: string;
  ibs_rate?: string;
  cbs_rate?: string;
  selective_tax_code?: string;
  selective_tax_rate?: string;
  operation_code?: string;
  emitter_uf?: string;
  recipient_uf?: string;
  operation_nature?: string;
  target_tax_regime?: string;
  observed_tax_regime?: string;
  target_crt?: string;
  observed_crt?: string;
};

export async function listCatalogProducts(query = ""): Promise<{ items: CatalogProductItem[] }> {
  const params = query.trim() ? `?q=${encodeURIComponent(query.trim())}` : "";
  return apiFetch(`/catalog/products${params}`);
}

export async function saveCatalogProduct(payload: SaveCatalogProductPayload): Promise<{ items: CatalogProductItem[]; message?: string; product_id?: string }> {
  return apiFetch("/catalog/products", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
