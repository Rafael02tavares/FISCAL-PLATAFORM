import { apiFetch } from "./api";

export async function listImportBatches(sourceName = "", limit = 20) {
  const params = new URLSearchParams();
  if (sourceName) {
    params.set("source_name", sourceName);
  }
  params.set("limit", String(limit));

  return apiFetch(`/admin/imports/batches?${params.toString()}`, {
    method: "GET",
  });
}

export async function uploadNCMCatalog(file: File, sourceName: string, versionLabel: string) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("source_name", sourceName);
  formData.append("version_label", versionLabel);

  return apiFetch("/admin/imports/ncm", {
    method: "POST",
    body: formData,
  });
}

export async function uploadCFOPCatalog(file: File, sourceName: string, versionLabel: string) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("source_name", sourceName);
  formData.append("version_label", versionLabel);

  return apiFetch("/admin/imports/cfop", {
    method: "POST",
    body: formData,
  });
}
