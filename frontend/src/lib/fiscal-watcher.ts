import { apiFetch } from "./api";

export type FiscalWatcherSource = {
  id: string;
  code: string;
  name: string;
  authority: string;
  source_type: string;
  url: string;
  cadence_hours: number;
  active: boolean;
  last_checked_at?: string;
  last_status: string;
  updated_at: string;
};

export type FiscalWatcherEvent = {
  id: string;
  source_id: string;
  source_code: string;
  source_name: string;
  authority: string;
  status: string;
  severity: string;
  detection_mode: string;
  title: string;
  summary: string;
  detected_at: string;
  payload?: Record<string, unknown>;
};

export async function listFiscalWatcherSources() {
  return apiFetch<{ items: FiscalWatcherSource[] }>("/admin/fiscal-watcher/sources", {
    method: "GET",
  });
}

export async function listFiscalWatcherEvents(status = "", limit = 20) {
  const params = new URLSearchParams();
  if (status.trim()) params.set("status", status.trim());
  params.set("limit", String(limit));

  return apiFetch<{ items: FiscalWatcherEvent[] }>(`/admin/fiscal-watcher/events?${params.toString()}`, {
    method: "GET",
  });
}

export async function runFiscalWatcherCheck(sourceCode = "") {
  return apiFetch<{ message: string; items: FiscalWatcherEvent[] }>("/admin/fiscal-watcher/check", {
    method: "POST",
    body: JSON.stringify({ source_code: sourceCode }),
  });
}
