import { apiFetch } from "./api";

export type DIFALRule = {
  id: string;
  code: string;
  name: string;
  uf: string;
  priority: number;
  specificity_hint: number;
  status: string;
  valid_from: string;
  valid_to: string;
  legal_basis_ids: string[];
  issuer_uf: string;
  recipient_uf: string;
  operation_scope: string;
  operation_type: string;
  final_consumer_mode: string;
  recipient_contributor: string;
  crt: string;
  cfop_prefix: string;
  ncm_prefix: string;
  internal_rate: string;
  interstate_rate: string;
  fcp_rate: string;
  applies: boolean;
  reason: string;
};

export type ICMSStateRate = {
  id: string;
  uf: string;
  internal_rate: string;
  fcp_rate: string;
  valid_from: string;
  valid_to: string;
  source_reference: string;
  source_url: string;
  notes: string;
};

export type ICMSRateReference = {
  mode: string;
  issuer_uf: string;
  recipient_uf: string;
  internal_rate: string;
  interstate_rate: string;
  fcp_rate: string;
  difference_rate: string;
  valid_from: string;
  valid_to: string;
  source_reference: string;
  source_url: string;
  notes: string;
  resolved_at: string;
};

export type CreateDIFALRulePayload = {
  code: string;
  name: string;
  uf: string;
  priority: number;
  status: string;
  valid_from: string;
  valid_to: string;
  legal_basis_ids: string[];
  issuer_uf: string;
  recipient_uf: string;
  operation_scope: string;
  operation_type: string;
  final_consumer_mode: string;
  recipient_contributor: string;
  crt: string;
  cfop_prefix: string;
  ncm_prefix: string;
  internal_rate: string;
  interstate_rate: string;
  fcp_rate: string;
  applies: boolean;
  reason: string;
};

export type UpsertICMSStateRatePayload = {
  uf: string;
  internal_rate: string;
  fcp_rate: string;
  valid_from: string;
  valid_to: string;
  source_reference: string;
  source_url: string;
  notes: string;
};

export async function listDIFALRules(limit = 120) {
  const params = new URLSearchParams({ limit: String(limit) });

  return apiFetch<{ items: DIFALRule[] }>(`/admin/icms-partilha?${params.toString()}`, {
    method: "GET",
  });
}

export async function createDIFALRule(payload: CreateDIFALRulePayload) {
  return apiFetch<{ id: string; message: string }>("/admin/icms-partilha", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function listICMSStateRates() {
  return apiFetch<{ items: ICMSStateRate[] }>("/admin/icms-rates", {
    method: "GET",
  });
}

export async function upsertICMSStateRate(payload: UpsertICMSStateRatePayload) {
  return apiFetch<{ id: string; message: string }>("/admin/icms-rates", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function resolveICMSRateReference(issuerUF: string, recipientUF: string) {
  const params = new URLSearchParams({
    issuer_uf: issuerUF,
    recipient_uf: recipientUF,
  });

  return apiFetch<{ item: ICMSRateReference | null }>(`/admin/icms-rates/reference?${params.toString()}`, {
    method: "GET",
  });
}
