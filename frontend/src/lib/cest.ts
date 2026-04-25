import { apiFetch } from "./api";

export type CESTItem = {
  id: string;
  code: string;
  ncm_code: string;
  segment: string;
  description: string;
  legal_source: string;
  start_date: string;
  end_date: string;
  is_active: boolean;
};

export async function listCEST(params?: { q?: string; ncm?: string; limit?: number }) {
  const searchParams = new URLSearchParams();
  if (params?.q) searchParams.set("q", params.q);
  if (params?.ncm) searchParams.set("ncm", params.ncm);
  searchParams.set("limit", String(params?.limit || 120));

  return apiFetch<{ items: CESTItem[] }>(`/cest?${searchParams.toString()}`, {
    method: "GET",
  });
}

export async function findCESTByCode(code: string) {
  const params = new URLSearchParams({ code });
  return apiFetch<{ item: CESTItem }>(`/cest/find?${params.toString()}`, {
    method: "GET",
  });
}
