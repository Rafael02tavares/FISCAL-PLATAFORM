import { apiFetch } from "./api";

export type AdminUser = {
  id: string;
  name: string;
  email: string;
  role: string;
};

export async function listAdminUsers() {
  return apiFetch<{ items: AdminUser[] }>("/admin/users", {
    method: "GET",
  });
}

export async function createAdminUser(payload: {
  name: string;
  email: string;
  password: string;
  role: string;
}) {
  return apiFetch<{ message: string; items: AdminUser[] }>("/admin/users", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function updateAdminUserRole(payload: { user_id: string; role: string }) {
  return apiFetch<{ message: string; items: AdminUser[] }>("/admin/users/role", {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
}

export async function removeAdminUser(userId: string) {
  const params = new URLSearchParams({ user_id: userId });
  return apiFetch<{ message: string; items: AdminUser[] }>(`/admin/users?${params.toString()}`, {
    method: "DELETE",
  });
}
