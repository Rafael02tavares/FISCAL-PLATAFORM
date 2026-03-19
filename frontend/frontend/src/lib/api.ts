const API_URL = "http://localhost:8081";

export async function apiFetch(path: string, options: RequestInit = {}) {
  const token = localStorage.getItem("token");
  const organizationId = localStorage.getItem("organization_id");

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  if (organizationId) {
    headers["X-Organization-ID"] = organizationId;
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...(options.headers || {}),
    },
  });

  const contentType = response.headers.get("content-type") || "";
  const isJSON = contentType.includes("application/json");

  if (!response.ok) {
    const errorBody = isJSON ? await response.json() : await response.text();
    throw new Error(typeof errorBody === "string" ? errorBody : JSON.stringify(errorBody));
  }

  return isJSON ? response.json() : response.text();
}