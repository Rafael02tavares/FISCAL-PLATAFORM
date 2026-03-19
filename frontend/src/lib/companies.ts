import { apiFetch } from "./api";

export async function lookupCompanyByCNPJ(cnpj: string) {
  return apiFetch(`/companies/lookup?cnpj=${encodeURIComponent(cnpj)}`, {
    method: "GET",
  });
}