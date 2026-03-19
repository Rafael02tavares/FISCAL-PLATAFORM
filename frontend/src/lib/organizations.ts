import { apiFetch } from "./api";

export async function listOrganizations() {
  return apiFetch("/organizations", {
    method: "GET",
  });
}

export async function createOrganization(payload: {
  name: string;
  cnpj: string;
  tax_regime: string;
  crt: string;
  state_registration: string;
  home_uf: string;
}) {
  return apiFetch("/organizations", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}