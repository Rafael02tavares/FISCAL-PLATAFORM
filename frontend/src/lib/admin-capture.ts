import { apiFetch } from "./api";

export type CaptureCandidate = {
  invoice_id: string;
  invoice_item_id: string;
  invoice_number: string;
  invoice_series: string;
  issued_at: string;
  emitter_name: string;
  emitter_uf: string;
  recipient_uf: string;
  operation_nature: string;
  item_number: number;
  product_code: string;
  gtin: string;
  description: string;
  ncm: string;
  cest: string;
  cfop: string;
  icms_cst: string;
  csosn: string;
  icms_rate: string;
  pis_cst: string;
  pis_rate: string;
  cofins_cst: string;
  cofins_rate: string;
  icms_value: string;
  ipi_value: string;
  pis_value: string;
  cofins_value: string;
  has_observed_taxes: boolean;
};

type CaptureCandidateListResponse = {
  items: CaptureCandidate[];
};

type AcceptCaptureCandidateResponse = {
  message: string;
};

export async function listCaptureCandidates(limit = 120): Promise<CaptureCandidateListResponse> {
  const params = new URLSearchParams({ limit: String(limit) });

  return apiFetch<CaptureCandidateListResponse>(`/admin/capture-rules?${params.toString()}`, {
    method: "GET",
  });
}

export async function acceptCaptureCandidate(
  invoiceItemId: string
): Promise<AcceptCaptureCandidateResponse> {
  return apiFetch<AcceptCaptureCandidateResponse>("/admin/capture-rules/accept", {
    method: "POST",
    body: JSON.stringify({
      invoice_item_id: invoiceItemId,
    }),
  });
}
