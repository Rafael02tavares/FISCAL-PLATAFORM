import { apiFetch } from "./api";

export async function listInvoices() {
  return apiFetch("/invoices", { method: "GET" });
}

export async function getInvoice(id: string) {
  return apiFetch(`/invoices/${id}`, { method: "GET" });
}