import { apiFetch } from "./api";

export type CFOPItem = {
  id: string;
  code: string;
  description: string;
  operation_type: string;
  ind_nfe: boolean;
  ind_communication: boolean;
  ind_transport: boolean;
  ind_devolution: boolean;
  created_at: string;
};

export async function listCFOP(params?: {
  q?: string;
  operationType?: string;
  limit?: number;
}) {
  const searchParams = new URLSearchParams();

  if (params?.q) {
    searchParams.set("q", params.q);
  }
  if (params?.operationType) {
    searchParams.set("operation_type", params.operationType);
  }
  searchParams.set("limit", String(params?.limit || 120));

  return apiFetch<{ items: CFOPItem[] }>(`/cfop?${searchParams.toString()}`, {
    method: "GET",
  });
}

export async function findCFOPByCode(code: string) {
  const params = new URLSearchParams({ code });
  return apiFetch<{ item: CFOPItem }>(`/cfop/find?${params.toString()}`, {
    method: "GET",
  });
}
