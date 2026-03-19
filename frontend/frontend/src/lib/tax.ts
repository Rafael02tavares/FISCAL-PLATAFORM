import { apiFetch } from "./api";

export async function suggestTax(payload: {
  gtin: string
  description: string
  emitter_uf: string
  recipient_uf: string
}) {
  return apiFetch("/tax/suggest", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}