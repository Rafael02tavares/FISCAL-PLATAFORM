import { apiFetch } from "./api";

type LoginResponse = {
  token: string;
};

type RegisterResponse = {
  message: string;
};

type MeResponse = {
  user: {
    id: string;
  };
};

const TOKEN_KEY = "token";
const ORGANIZATION_ID_KEY = "organization_id";
const ORGANIZATION_ROLE_KEY = "organization_role";
const ORGANIZATION_TAX_REGIME_KEY = "organization_tax_regime";
const ORGANIZATION_CRT_KEY = "organization_crt";

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function safeSetItem(key: string, value: string): void {
  if (!isBrowser()) {
    return;
  }

  try {
    localStorage.setItem(key, value);
  } catch {
    // evita quebrar a aplicação em caso de storage indisponível
  }
}

function safeGetItem(key: string): string | null {
  if (!isBrowser()) {
    return null;
  }

  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeRemoveItem(key: string): void {
  if (!isBrowser()) {
    return;
  }

  try {
    localStorage.removeItem(key);
  } catch {
    // evita quebrar a aplicação em caso de storage indisponível
  }
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  clearSession();

  const response = await apiFetch<LoginResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });

  if (response.token) {
    saveToken(response.token);
  }

  return response;
}

export async function register(
  name: string,
  email: string,
  password: string
): Promise<RegisterResponse> {
  return apiFetch<RegisterResponse>("/auth/register", {
    method: "POST",
    body: JSON.stringify({ name, email, password }),
  });
}

export async function getCurrentUser(): Promise<MeResponse> {
  return apiFetch<MeResponse>("/auth/me", {
    method: "GET",
  });
}

export function saveToken(token: string): void {
  safeSetItem(TOKEN_KEY, token);
}

export function getToken(): string | null {
  return safeGetItem(TOKEN_KEY);
}

export function removeToken(): void {
  safeRemoveItem(TOKEN_KEY);
}

export function saveOrganizationId(organizationId: string): void {
  safeSetItem(ORGANIZATION_ID_KEY, organizationId);
}

export function getOrganizationId(): string | null {
  return safeGetItem(ORGANIZATION_ID_KEY);
}

export function removeOrganizationId(): void {
  safeRemoveItem(ORGANIZATION_ID_KEY);
}

export function saveOrganizationRole(role: string): void {
  safeSetItem(ORGANIZATION_ROLE_KEY, role);
}

export function getOrganizationRole(): string | null {
  return safeGetItem(ORGANIZATION_ROLE_KEY);
}

export function removeOrganizationRole(): void {
  safeRemoveItem(ORGANIZATION_ROLE_KEY);
}

export function saveOrganizationTaxRegime(taxRegime: string): void {
  safeSetItem(ORGANIZATION_TAX_REGIME_KEY, taxRegime);
}

export function getOrganizationTaxRegime(): string | null {
  return safeGetItem(ORGANIZATION_TAX_REGIME_KEY);
}

export function removeOrganizationTaxRegime(): void {
  safeRemoveItem(ORGANIZATION_TAX_REGIME_KEY);
}

export function saveOrganizationCRT(crt: string): void {
  safeSetItem(ORGANIZATION_CRT_KEY, crt);
}

export function getOrganizationCRT(): string | null {
  return safeGetItem(ORGANIZATION_CRT_KEY);
}

export function removeOrganizationCRT(): void {
  safeRemoveItem(ORGANIZATION_CRT_KEY);
}

export function hasAdminAreaAccess(): boolean {
  const role = String(getOrganizationRole() || "").trim().toLowerCase();
  return role === "owner" || role === "admin" || role === "analyst";
}

export function isAuthenticated(): boolean {
  const token = getToken();
  return typeof token === "string" && token.trim() !== "";
}

export function clearSession(): void {
  removeToken();
  removeOrganizationId();
  removeOrganizationRole();
  removeOrganizationTaxRegime();
  removeOrganizationCRT();
}

export function logout(): void {
  clearSession();
}
