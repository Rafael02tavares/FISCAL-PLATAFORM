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

export async function uploadCESTCatalog(file: File | null, sourceName: string, versionLabel: string, content = "") {
  const formData = new FormData();
  if (file) {
    formData.append("file", file);
  }
  formData.append("source_name", sourceName);
  formData.append("version_label", versionLabel);
  if (content.trim()) {
    formData.append("content", content.trim());
  }

  return apiFetch("/admin/imports/cest", {
    method: "POST",
    body: formData,
  });
}

export async function uploadCBenefCatalog(file: File | null, sourceName: string, versionLabel: string, content = "", uf = "") {
  const formData = new FormData();
  if (file) {
    formData.append("file", file);
  }
  formData.append("source_name", sourceName);
  formData.append("version_label", versionLabel);
  if (uf.trim()) {
    formData.append("uf", uf.trim().toUpperCase());
  }
  if (content.trim()) {
    formData.append("content", content.trim());
  }

  return apiFetch("/admin/imports/cbenef", {
    method: "POST",
    body: formData,
  });
}

export async function uploadStateICMSSTCatalog(
  file: File | null,
  sourceName: string,
  versionLabel: string,
  content = "",
  uf = "",
  sourceUrl = ""
) {
  if (!file) {
    throw new Error("arquivo XLSX obrigatorio para importar ST estadual");
  }

  const formData = new FormData();
  formData.append("file", file);
  formData.append("source_name", sourceName);
  formData.append("version_label", versionLabel);
  if (uf.trim()) {
    formData.append("uf", uf.trim().toUpperCase());
  }
  if (sourceUrl.trim()) {
    formData.append("source_url", sourceUrl.trim());
  }

  return apiFetch("/admin/imports/state-icms-st", {
    method: "POST",
    body: formData,
  });
}
