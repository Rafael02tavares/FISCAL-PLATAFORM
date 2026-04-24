import { getOrganizationId, getToken } from "../lib/auth";
import {
  createAdminUser,
  listAdminUsers,
  removeAdminUser,
  updateAdminUserRole,
  type AdminUser,
} from "../lib/admin-users";

const form = document.getElementById("admin-user-form") as HTMLFormElement | null;
const feedback = document.getElementById("admin-user-feedback");
const listBox = document.getElementById("admin-users-list");
const statsBox = document.getElementById("admin-users-stats");

function setFeedback(message: string, tone: "muted" | "success" | "error" = "muted") {
  if (!feedback) return;

  feedback.textContent = message;
  feedback.className =
    tone === "success"
      ? "feedback feedback--success"
      : tone === "error"
        ? "feedback feedback--error"
        : "dashboard-note";
}

function formatRole(role: string) {
  switch (role) {
    case "owner":
      return "Owner";
    case "admin":
      return "Admin";
    case "analyst":
      return "Analista";
    case "viewer":
      return "Leitor";
    default:
      return role || "Sem papel";
  }
}

function renderStats(items: AdminUser[]) {
  if (!statsBox) return;

  const owners = items.filter((item) => item.role === "owner").length;
  const admins = items.filter((item) => item.role === "admin").length;
  const analysts = items.filter((item) => item.role === "analyst").length;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Total</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--gold">
      <span>Owners</span>
      <strong>${owners}</strong>
    </article>
    <article class="stat-card stat-card--blue">
      <span>Admins</span>
      <strong>${admins}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Analistas</span>
      <strong>${analysts}</strong>
    </article>
  `;
}

function renderList(items: AdminUser[]) {
  if (!listBox) return;

  renderStats(items);

  if (!items.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhum usuario vinculado.</strong>
        <p>Cadastre o primeiro usuario da organizacao usando o formulario ao lado.</p>
      </div>
    `;
    return;
  }

  listBox.innerHTML = `
    <div class="user-list">
      ${items
        .map(
          (item) => `
            <article class="user-card" data-user-id="${item.id}">
              <div class="user-card__top">
                <div>
                  <strong>${item.name || "-"}</strong>
                  <p>${item.email || "-"}</p>
                </div>
                <span class="role-badge role-badge--${item.role || "viewer"}">${formatRole(item.role)}</span>
              </div>

              <div class="user-card__actions">
                <select class="role-select" data-role-user-id="${item.id}">
                  <option value="owner" ${item.role === "owner" ? "selected" : ""}>Owner</option>
                  <option value="admin" ${item.role === "admin" ? "selected" : ""}>Admin</option>
                  <option value="analyst" ${item.role === "analyst" ? "selected" : ""}>Analista</option>
                  <option value="viewer" ${item.role === "viewer" ? "selected" : ""}>Leitor</option>
                </select>
                <button class="secondary-button" type="button" data-save-role="${item.id}">Salvar papel</button>
                <button class="danger-button" type="button" data-remove-user="${item.id}">Remover</button>
              </div>
            </article>
          `
        )
        .join("")}
    </div>
  `;

  listBox.querySelectorAll<HTMLButtonElement>("[data-save-role]").forEach((button) => {
    button.addEventListener("click", async () => {
      const userId = button.dataset.saveRole || "";
      const select = listBox.querySelector<HTMLSelectElement>(`[data-role-user-id="${userId}"]`);
      if (!userId || !select) return;

      button.disabled = true;
      try {
        const response = await updateAdminUserRole({ user_id: userId, role: select.value });
        renderList(Array.isArray(response?.items) ? response.items : []);
        setFeedback(response?.message || "Papel atualizado com sucesso.", "success");
      } catch (error) {
        setFeedback(`Falha ao atualizar papel: ${String(error)}`, "error");
      } finally {
        button.disabled = false;
      }
    });
  });

  listBox.querySelectorAll<HTMLButtonElement>("[data-remove-user]").forEach((button) => {
    button.addEventListener("click", async () => {
      const userId = button.dataset.removeUser || "";
      if (!userId) return;

      const confirmed = window.confirm("Deseja remover este usuario da organizacao atual?");
      if (!confirmed) return;

      button.disabled = true;
      try {
        const response = await removeAdminUser(userId);
        renderList(Array.isArray(response?.items) ? response.items : []);
        setFeedback(response?.message || "Usuario removido com sucesso.", "success");
      } catch (error) {
        setFeedback(`Falha ao remover usuario: ${String(error)}`, "error");
      } finally {
        button.disabled = false;
      }
    });
  });
}

async function loadUsers() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  if (!getOrganizationId()) {
    setFeedback("Selecione uma organizacao antes de gerenciar usuarios.", "error");
    return;
  }

  try {
    const response = await listAdminUsers();
    renderList(Array.isArray(response?.items) ? response.items : []);
  } catch (error) {
    if (!listBox) return;
    listBox.innerHTML = `
      <div class="dashboard-empty dashboard-empty--error">
        <strong>Falha ao carregar usuarios.</strong>
        <p>${String(error)}</p>
      </div>
    `;
  }
}

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(form);
  const payload = {
    name: String(formData.get("name") || "").trim(),
    email: String(formData.get("email") || "").trim(),
    password: String(formData.get("password") || "").trim(),
    role: String(formData.get("role") || "viewer").trim(),
  };

  const submit = form.querySelector<HTMLButtonElement>('button[type="submit"]');
  if (submit) {
    submit.disabled = true;
    submit.textContent = "Salvando...";
  }

  setFeedback("Salvando usuario da organizacao...", "muted");

  try {
    const response = await createAdminUser(payload);
    renderList(Array.isArray(response?.items) ? response.items : []);
    setFeedback(response?.message || "Usuario salvo com sucesso.", "success");
    form.reset();
    const roleField = form.querySelector<HTMLSelectElement>('select[name="role"]');
    if (roleField) roleField.value = "viewer";
  } catch (error) {
    setFeedback(`Falha ao salvar usuario: ${String(error)}`, "error");
  } finally {
    if (submit) {
      submit.disabled = false;
      submit.textContent = "Salvar usuario";
    }
  }
});

void loadUsers();
