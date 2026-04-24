import { apiFetch } from "./api";

export async function listInvoices() {
  return apiFetch("/invoices", { method: "GET" });
}

export async function getInvoice(id: string) {
  return apiFetch(`/invoices/${id}`, { method: "GET" });
}

export async function uploadInvoice(file: File) {
  return uploadInvoices([file]);
}

export async function uploadInvoices(files: File[]) {
  const formData = new FormData();
  for (const file of files) {
    formData.append("files", file);
  }

  return apiFetch("/invoices/upload", {
    method: "POST",
    body: formData,
  });
}
