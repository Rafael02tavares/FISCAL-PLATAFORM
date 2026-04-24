import { apiFetch } from "./api";

export async function listNCM(limit = 100) {
  return apiFetch(`/ncm?limit=${limit}`, {
    method: "GET",
  });
}

export async function searchNCM(query: string, limit = 50) {
  return apiFetch(`/ncm/search?q=${encodeURIComponent(query)}&limit=${limit}`, {
    method: "GET",
  });
}

export async function getNCMByCode(code: string) {
  return apiFetch(`/ncm/find?code=${encodeURIComponent(code)}`, {
    method: "GET",
  });
}
