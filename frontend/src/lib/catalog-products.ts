import { apiFetch } from "./api";

export type CatalogProductProfile = {
  id: string;
  product_id: string;
  organization_id: string;
  source_invoice_id?: string;
  ncm?: string;
  ncm_description?: string;
  ncm_ex?: string;
  cest?: string;
  cest_description?: string;
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

export type CatalogProductPage = {
  items: CatalogProductItem[];
  limit?: number;
  offset?: number;
  has_more?: boolean;
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

export async function listCatalogProducts(query = "", options: { limit?: number; offset?: number } = {}): Promise<CatalogProductPage> {
  const params = new URLSearchParams();
  if (query.trim()) {
    params.set("q", query.trim());
  }
  if (options.limit) {
    params.set("limit", String(options.limit));
  }
  if (options.offset) {
    params.set("offset", String(options.offset));
  }

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return apiFetch(`/catalog/products${suffix}`);
}

export async function saveCatalogProduct(payload: SaveCatalogProductPayload): Promise<{ items: CatalogProductItem[]; message?: string; product_id?: string }> {
  return apiFetch("/catalog/products", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
