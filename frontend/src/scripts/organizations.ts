import {
  clearSession,
  getOrganizationId,
  getToken,
  saveOrganizationHomeUF,
  saveOrganizationId,
  saveOrganizationCRT,
  saveOrganizationRole,
  saveOrganizationTaxRegime,
} from "../lib/auth";
import { createOrganization, listOrganizations, updateOrganization } from "../lib/organizations";

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
  const organizationContextForm = document.getElementById("organization-context-form") as HTMLFormElement | null;
  const organizationMessage = document.getElementById("organization-message");
  const contextMessage = document.getElementById("organization-context-message");
  const contextPanel = document.getElementById("organization-context-panel");

  let loadedOrganizations: Organization[] = [];

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

  function showContextMessage(text: string, tone = "success"): void {
    if (!contextMessage) return;
    contextMessage.textContent = text;
    contextMessage.className = `organization-message is-visible ${
      tone === "error" ? "is-error" : "is-success"
    }`;
  }

  function escapeHTML(value: unknown): string {
    return String(value || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function fiscalPayloadFromForm(form: HTMLFormElement) {
    const data = new FormData(form);
    return {
      name: String(data.get("name") || "").trim(),
      cnpj: String(data.get("cnpj") || "").trim(),
      tax_regime: String(data.get("tax_regime") || "").trim(),
      crt: String(data.get("crt") || "").trim(),
      state_registration: String(data.get("state_registration") || "").trim(),
      home_uf: String(data.get("home_uf") || "").trim().toUpperCase(),
    };
  }

  function persistOrganizationContext(organization: Organization): void {
    if (!organization?.id) return;
    saveOrganizationId(organization.id);
    saveOrganizationRole(organization.role || "viewer");
    saveOrganizationTaxRegime(organization.tax_regime || "");
    saveOrganizationCRT(organization.crt || "");
    saveOrganizationHomeUF(organization.home_uf || "");
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
        if (organization) {
          persistOrganizationContext(organization);
        } else {
          saveOrganizationId(id);
        }
        showMessage("Organizacao ativa atualizada. Redirecionando para a dashboard...", "success");
        window.setTimeout(() => {
          window.location.href = "/";
        }, 600);
      });
    });
  }

  function renderFiscalContextPanel(active: Organization | null): void {
    if (!contextPanel || !organizationContextForm) return;

    if (!active) {
      contextPanel.innerHTML = `
        <div class="dashboard-empty">
          <strong>Nenhuma organizacao ativa.</strong>
          <p>Selecione ou cadastre uma organizacao para definir UF e regime usados pelo motor tributario.</p>
        </div>
      `;
      organizationContextForm.style.display = "none";
      return;
    }

    organizationContextForm.style.display = "grid";
    contextPanel.innerHTML = `
      <div class="fiscal-context-summary">
        <article>
          <span>UF base</span>
          <strong>${escapeHTML(active.home_uf || "-")}</strong>
          <p>Usada como origem e destino padrao nas sugestoes internas.</p>
        </article>
        <article>
          <span>Regime</span>
          <strong>${escapeHTML(active.tax_regime || "-")}</strong>
          <p>Define CST/CSOSN e padrao de PIS/COFINS.</p>
        </article>
        <article>
          <span>CRT</span>
          <strong>${escapeHTML(active.crt || "-")}</strong>
          <p>CRT 1 usa CSOSN; CRT 3 usa CST ICMS.</p>
        </article>
      </div>
    `;

    (document.getElementById("context-org-name") as HTMLInputElement | null)!.value = active.name || "";
    (document.getElementById("context-org-cnpj") as HTMLInputElement | null)!.value = active.cnpj || "";
    (document.getElementById("context-org-tax-regime") as HTMLSelectElement | null)!.value =
      active.tax_regime || "simples_nacional";
    (document.getElementById("context-org-crt") as HTMLSelectElement | null)!.value = active.crt || "1";
    (document.getElementById("context-org-state-registration") as HTMLInputElement | null)!.value =
      active.state_registration || "";
    (document.getElementById("context-org-home-uf") as HTMLInputElement | null)!.value =
      (active.home_uf || "").toUpperCase();
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
      loadedOrganizations = organizations;
      const activeOrganization = organizations.find((item) => item.id === activeOrganizationId) || organizations[0] || null;

      renderOrganizations(organizations, activeOrganizationId);
      renderFiscalContextPanel(activeOrganization);

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
        persistOrganizationContext({
          ...organization,
          role: organization.role || "owner",
          tax_regime: organization.tax_regime || payload.tax_regime,
          crt: organization.crt || payload.crt,
          home_uf: organization.home_uf || payload.home_uf,
        });
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

  organizationContextForm?.addEventListener("submit", async (event) => {
    event.preventDefault();

    const activeOrganizationId = getOrganizationId();
    if (!activeOrganizationId) {
      showContextMessage("Selecione uma organizacao ativa antes de editar o contexto fiscal.", "error");
      return;
    }

    try {
      showContextMessage("Atualizando contexto fiscal da organizacao...", "success");
      const payload = fiscalPayloadFromForm(organizationContextForm);
      const response = await updateOrganization(activeOrganizationId, payload);
      const organization = response.organization as Organization;
      const existing = loadedOrganizations.find((item) => item.id === activeOrganizationId);
      persistOrganizationContext({
        ...organization,
        role: existing?.role || organization.role || "owner",
      });
      showContextMessage("Contexto fiscal atualizado. As proximas sugestoes usarao essa UF e regime.", "success");
      await loadOrganizationsScreen();
    } catch (error) {
      showContextMessage(`Erro ao atualizar contexto fiscal: ${String(error)}`, "error");
    }
  });

  loadOrganizationsScreen();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initOrganizationsPage, { once: true });
} else {
  initOrganizationsPage();
}
