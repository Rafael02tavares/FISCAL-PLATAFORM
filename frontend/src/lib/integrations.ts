import { apiFetch } from "./api";

export type IntegrationSetting = {
  id: string;
  organization_id: string;
  provider: string;
  enabled: boolean;
  base_url: string;
  model_name: string;
  has_token: boolean;
  token_preview: string;
  notes: string;
  updated_at: string;
};

export type OpenAITestResult = {
  ok: boolean;
  status_code: number;
  message: string;
  model: string;
  output: string;
  classification?: Record<string, unknown>;
};

export type CosmosTestResult = {
  ok: boolean;
  status_code: number;
  message: string;
  gtin: string;
  description: string;
  ncm: string;
  raw?: Record<string, unknown>;
};

export type CosmosProductCandidate = {
  description: string;
  gtin: string;
  ncm: string;
  ncm_description: string;
  cest: string;
  brand: string;
  thumbnail: string;
  source: string;
  raw?: Record<string, unknown>;
};

export type CosmosSearchResult = {
  ok: boolean;
  status_code: number;
  message: string;
  query: string;
  items: CosmosProductCandidate[];
};

export async function getCosmosIntegration() {
  return apiFetch<IntegrationSetting>("/admin/integrations/cosmos", {
    method: "GET",
  });
}

export async function saveCosmosIntegration(payload: {
  enabled: boolean;
  base_url: string;
  api_token: string;
  notes: string;
}) {
  return apiFetch<{ message: string; item: IntegrationSetting }>("/admin/integrations/cosmos", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function testCosmosIntegration(payload: {
  gtin: string;
  api_token?: string;
}) {
  return apiFetch<CosmosTestResult>("/admin/integrations/cosmos/test", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function searchCosmosProducts(payload: {
  query: string;
  api_token?: string;
  limit?: number;
}) {
  return apiFetch<CosmosSearchResult>("/admin/integrations/cosmos/search", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function getOpenAIIntegration() {
  return apiFetch<IntegrationSetting>("/admin/integrations/openai", {
    method: "GET",
  });
}

export async function saveOpenAIIntegration(payload: {
  enabled: boolean;
  base_url: string;
  model_name: string;
  api_token: string;
  notes: string;
}) {
  return apiFetch<{ message: string; item: IntegrationSetting }>("/admin/integrations/openai", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function testOpenAIIntegration(payload: {
  api_token?: string;
  model_name?: string;
  description?: string;
  gtin?: string;
  ncm?: string;
  cest?: string;
  uf?: string;
  tax_regime?: string;
  operation?: string;
}) {
  return apiFetch<OpenAITestResult>("/admin/integrations/openai/test", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
