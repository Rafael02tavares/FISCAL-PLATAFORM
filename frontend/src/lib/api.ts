const API_URL = (import.meta.env.PUBLIC_API_URL || "http://localhost:8081").replace(/\/+$/, "");
const LOGIN_PATH = "/login";

type APIErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

export class APIRequestError extends Error {
  status: number;
  code?: string;
  details?: unknown;

  constructor(message: string, status: number, code?: string, details?: unknown) {
    super(message);
    this.name = "APIRequestError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function getStoredValue(key: string): string | null {
  if (!isBrowser()) {
    return null;
  }

  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function clearStoredSession(): void {
  if (!isBrowser()) {
    return;
  }

  try {
    localStorage.removeItem("token");
    localStorage.removeItem("organization_id");
    localStorage.removeItem("organization_role");
  } catch {
    // evita quebrar a aplicacao em caso de storage indisponivel
  }
}

function redirectToLogin(): void {
  if (!isBrowser()) {
    return;
  }

  if (window.location.pathname === LOGIN_PATH) {
    return;
  }

  window.location.href = LOGIN_PATH;
}

function buildHeaders(options: RequestInit): Headers {
  const headers = new Headers(options.headers || {});
  const token = getStoredValue("token");
  const organizationId = getStoredValue("organization_id");
  const isFormData = options.body instanceof FormData;

  if (!isFormData && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  if (organizationId && !headers.has("X-Organization-ID")) {
    headers.set("X-Organization-ID", organizationId);
  }

  return headers;
}

async function parseResponseBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") || "";

  if (contentType.includes("application/json")) {
    return response.json();
  }

  return response.text();
}

function extractErrorMessage(payload: unknown, fallback: string): { message: string; code?: string } {
  if (payload && typeof payload === "object") {
    const data = payload as APIErrorPayload;

    if (data.error?.message) {
      return {
        message: data.error.message,
        code: data.error.code,
      };
    }
  }

  if (typeof payload === "string" && payload.trim() !== "") {
    return { message: payload };
  }

  return { message: fallback };
}

export async function apiFetch<T = unknown>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: buildHeaders(options),
  });

  const payload = await parseResponseBody(response);

  if (!response.ok) {
    const { message, code } = extractErrorMessage(
      payload,
      `Erro na requisicao (${response.status})`
    );

    if (response.status === 401) {
      clearStoredSession();
      redirectToLogin();
    }

    throw new APIRequestError(message, response.status, code, payload);
  }

  return payload as T;
}
