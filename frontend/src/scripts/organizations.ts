import {
  clearSession,
  getOrganizationId,
  getToken,
  saveOrganizationId,
  saveOrganizationCRT,
  saveOrganizationRole,
  saveOrganizationTaxRegime,
} from "../lib/auth";
import { createOrganization, listOrganizations } from "../lib/organizations";

type Organization = {
  id: string;
  name?: string;
  cnpj?: string;
  role?: string;
  tax_regime?: string;
  crt?: string;
  home_uf?: string;
};

function initOrganizationsPage() {
  const sessionBox = document.getElementById("organization-session");
  const actionsBox = document.getElementById("organization-actions");
  const organizationsList = document.getElementById("organizations-list");
  const organizationForm = document.getElementById("organization-form");
  const organizationMessage = document.getElementById("organization-message");

  function renderAction(title: string, subtitle: string, badge: string, tone = "warning"): string {
    return `
      <div class="list-item">
        <div class="list-item__meta">
          <span class="list-item__title">${title}</span>
          <span class="list-item__subtitle">${subtitle}</span>
        </div>
        <span class="badge badge--${tone}">${badge}</span>
      </div>
    `;
  }

  function formatRole(role?: string): string {
    switch (String(role || "").toLowerCase()) {
      case "owner":
        return "Owner";
      case "admin":
        return "Administrador";
      case "analyst":
        return "Analista";
      case "viewer":
        return "Consulta";
      default:
        return role || "Membro";
    }
  }

  function showMessage(text: string, tone = "success"): void {
    if (!organizationMessage) return;
    organizationMessage.textContent = text;
    organizationMessage.className = `organization-message is-visible ${
      tone === "error" ? "is-error" : "is-success"
    }`;
  }

  function setSessionState(text: string): void {
    if (sessionBox) {
      sessionBox.innerHTML = text;
    }
  }

  function renderOrganizations(items: Organization[], activeId: string | null): void {
    if (!organizationsList) return;

    if (!items.length) {
      organizationsList.innerHTML = `
        <div class="dashboard-empty">
          <strong>Nenhuma organizacao cadastrada.</strong>
          <p>Cadastre a primeira empresa para habilitar a dashboard com dados reais.</p>
        </div>
      `;
      return;
    }

    const cards = items
      .map((item) => {
        const active = item.id === activeId;
        return `
          <article class="organization-card ${active ? "organization-card--active" : ""}">
            <div class="organization-card__header">
              <div>
                <strong>${item.name || "-"}</strong>
                <div class="organization-card__meta">${item.cnpj || "Sem CNPJ"} - ${formatRole(item.role)}</div>
              </div>
              <span class="badge badge--${active ? "success" : "warning"}">
                ${active ? "Ativa" : "Disponivel"}
              </span>
            </div>

            <div class="organization-card__details">
              <span>Regime: ${item.tax_regime || "-"}</span>
              <span>CRT: ${item.crt || "-"}</span>
              <span>UF: ${item.home_uf || "-"}</span>
            </div>

            <div class="organization-card__actions">
              <button type="button" class="secondary-button organization-select" data-id="${item.id}">
                ${active ? "Selecionada" : "Usar nesta sessao"}
              </button>
            </div>
          </article>
        `;
      })
      .join("");

    organizationsList.innerHTML = `<div class="organization-cards">${cards}</div>`;

    document.querySelectorAll(".organization-select").forEach((button) => {
      button.addEventListener("click", () => {
        const id = button.getAttribute("data-id");
        if (!id) return;

        const organization = items.find((item) => item.id === id);
        saveOrganizationId(id);
        saveOrganizationRole(organization?.role || "viewer");
        saveOrganizationTaxRegime(organization?.tax_regime || "");
        saveOrganizationCRT(organization?.crt || "");
        showMessage("Organizacao ativa atualizada. Redirecionando para a dashboard...", "success");
        window.setTimeout(() => {
          window.location.href = "/";
        }, 600);
      });
    });
  }

  async function loadOrganizationsScreen() {
    const token = getToken();
    const activeOrganizationId = getOrganizationId();

    if (!token) {
      setSessionState(`
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Sessao nao autenticada.</strong>
          <p>Faca login antes de consultar ou criar organizacoes.</p>
          <p><a href="/login">Ir para login</a></p>
        </div>
      `);

      if (actionsBox) {
        actionsBox.innerHTML = renderAction(
          "Autenticar usuario",
          "As rotas de organizacao sao protegidas e exigem token de acesso.",
          "Bloqueado",
          "danger"
        );
      }

      if (organizationsList) {
        organizationsList.innerHTML = "";
      }
      return;
    }

    setSessionState(`
      <div class="session-summary">
        <strong>Token ativo encontrado.</strong>
        <span>Organizacao ativa: ${activeOrganizationId || "nenhuma definida"}</span>
        <button type="button" class="secondary-button" id="logout-button">Encerrar sessao</button>
      </div>
    `);

    document.getElementById("logout-button")?.addEventListener("click", () => {
      clearSession();
      window.location.href = "/login";
    });

    try {
      const response = await listOrganizations();
      const organizations: Organization[] = response.organizations || [];

      renderOrganizations(organizations, activeOrganizationId);

      const actions: string[] = [];

      if (!organizations.length) {
        actions.push(
          renderAction(
            "Cadastrar primeira organizacao",
            "Sem organizacao nao ha contexto para invoices, tax suggestion e dashboard real.",
            "Proximo passo",
            "warning"
          )
        );
      }

      if (organizations.length && !activeOrganizationId) {
        actions.push(
          renderAction(
            "Selecionar organizacao ativa",
            "Escolha uma empresa da lista para salvar organization_id na sessao.",
            "Acao necessaria",
            "warning"
          )
        );
      }

      if (organizations.length && activeOrganizationId) {
        actions.push(
          renderAction(
            "Sessao pronta",
            "A dashboard ja pode consultar organizations e invoices com dados reais.",
            "Conectado",
            "success"
          )
        );
      }

      if (actionsBox) {
        actionsBox.innerHTML = actions.join("");
      }
    } catch (error) {
      setSessionState(`
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar organizacoes.</strong>
          <p>${String(error)}</p>
        </div>
      `);

      if (actionsBox) {
        actionsBox.innerHTML = renderAction(
          "Revisar autenticacao",
          "A API respondeu com erro ao buscar organizacoes. Verifique token, banco e backend.",
          "Erro",
          "danger"
        );
      }
    }
  }

  organizationForm?.addEventListener("submit", async (event) => {
    event.preventDefault();

    try {
      showMessage("Salvando organizacao...", "success");

      const payload = {
        name: (document.getElementById("org-name") as HTMLInputElement | null)?.value?.trim() || "",
        cnpj: (document.getElementById("org-cnpj") as HTMLInputElement | null)?.value?.trim() || "",
        tax_regime:
          (document.getElementById("org-tax-regime") as HTMLSelectElement | null)?.value || "simples_nacional",
        crt: (document.getElementById("org-crt") as HTMLSelectElement | null)?.value || "1",
        state_registration:
          (document.getElementById("org-state-registration") as HTMLInputElement | null)?.value?.trim() || "",
        home_uf:
          ((document.getElementById("org-home-uf") as HTMLInputElement | null)?.value?.trim() || "").toUpperCase(),
      };

      const response = await createOrganization(payload);
      const organization = response.organization;

      if (organization?.id) {
        saveOrganizationId(organization.id);
        saveOrganizationRole(organization.role || "owner");
        saveOrganizationTaxRegime(organization.tax_regime || payload.tax_regime || "");
        saveOrganizationCRT(organization.crt || payload.crt || "");
      }

      showMessage("Organizacao criada com sucesso. Atualizando lista...", "success");

      if (organizationForm instanceof HTMLFormElement) {
        organizationForm.reset();
      }

      await loadOrganizationsScreen();
    } catch (error) {
      showMessage(`Erro ao salvar organizacao: ${String(error)}`, "error");
    }
  });

  loadOrganizationsScreen();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initOrganizationsPage, { once: true });
} else {
  initOrganizationsPage();
}
