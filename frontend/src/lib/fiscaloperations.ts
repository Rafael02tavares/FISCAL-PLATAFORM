import { apiFetch } from "./api";

export async function listFiscalOperations() {
  return apiFetch("/fiscal-operations", {
    method: "GET",
  });
}
