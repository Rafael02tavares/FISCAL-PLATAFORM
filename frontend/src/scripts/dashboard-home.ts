import { listInvoices } from "../lib/invoices";
import { listOrganizations } from "../lib/organizations";

const API_URL = (import.meta.env.PUBLIC_API_URL || "http://localhost:8081").replace(/\/+$/, "");

const statAPIStatus = document.getElementById("stat-api-status");
const statAPITrend = document.getElementById("stat-api-trend");
const statOrganizations = document.getElementById("stat-organizations");
const statOrganizationsTrend = document.getElementById("stat-organizations-trend");
const statActiveOrganization = document.getElementById("stat-active-organization");
const statActiveOrganizationTrend = document.getElementById("stat-active-organization-trend");
const statInvoices = document.getElementById("stat-invoices");
const statInvoicesTrend = document.getElementById("stat-invoices-trend");
const sessionState = document.getElementById("session-state");
const nextActions = document.getElementById("next-actions");
const invoicesPanel = document.getElementById("invoices-panel");

function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function setText(node: Element | null, value: string): void {
  if (node) node.textContent = value;
}

function renderList(target: Element | null, items: string[]): void {
  if (!target) return;
  target.innerHTML = items.join("");
}

function itemRow(title: string, subtitle: string, badge: string, tone = "warning"): string {
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

async function loadHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${API_URL}/health`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const payload = await response.json();
    setText(statAPIStatus, "Online");
    setText(statAPITrend, `Ambiente ${payload.app_env || "desconhecido"}`);
    return true;
  } catch (error) {
    setText(statAPIStatus, "Offline");
    setText(statAPITrend, `Falha ao consultar /health: ${String(error)}`);
    return false;
  }
}

function renderInvoices(invoices: any[]): void {
  if (!invoicesPanel) return;

  if (!invoices.length) {
    invoicesPanel.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma nota importada ainda.</strong>
        <p>Quando o upload de XML for utilizado, as notas mais recentes aparecerao aqui.</p>
      </div>
    `;
    return;
  }

  const rows = invoices
    .slice(0, 8)
    .map(
      (invoice) => `
        <tr>
          <td>${invoice.number || "-"}</td>
          <td>${invoice.series || "-"}</td>
          <td>${invoice.emitter_name || "-"}</td>
          <td>${invoice.issued_at || "-"}</td>
          <td>${invoice.total_amount || "-"}</td>
          <td>${invoice.status || "-"}</td>
        </tr>
      `
    )
    .join("");

  invoicesPanel.innerHTML = `
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Numero</th>
            <th>Serie</th>
            <th>Emitente</th>
            <th>Data</th>
            <th>Total</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    </div>
  `;
}

async function loadDashboard(): Promise<void> {
  const hasAPI = await loadHealth();
  const token = readStorage("token");
  const organizationId = readStorage("organization_id");

  renderList(sessionState, [
    itemRow(
      "Token de autenticacao",
      token ? "Sessao encontrada no navegador" : "Nenhum token salvo em localStorage",
      token ? "Ativo" : "Ausente",
      token ? "success" : "warning"
    ),
    itemRow(
      "Organizacao ativa",
      organizationId ? `ID selecionado: ${organizationId}` : "Nenhuma organizacao ativa foi definida",
      organizationId ? "Definida" : "Pendente",
      organizationId ? "success" : "warning"
    ),
  ]);

  if (!hasAPI) {
    renderList(nextActions, [
      itemRow(
        "Backend indisponivel",
        "Verifique se a API Go esta rodando antes de testar dados reais na dashboard.",
        "Bloqueado",
        "danger"
      ),
    ]);
    renderInvoices([]);
    return;
  }

  if (!token) {
    setText(statOrganizations, "--");
    setText(statOrganizationsTrend, "Login necessario");
    setText(statActiveOrganization, "--");
    setText(statActiveOrganizationTrend, "Sem sessao autenticada");
    setText(statInvoices, "--");
    setText(statInvoicesTrend, "Login necessario");

    renderList(nextActions, [
      itemRow(
        "Autenticar usuario",
        "As rotas de organizacoes e notas exigem token; sem isso a dashboard so consegue mostrar o status da API.",
        "Proximo passo",
        "warning"
      ),
    ]);
    renderInvoices([]);
    return;
  }

  try {
    const organizationsResponse = await listOrganizations();
    const organizations = organizationsResponse.organizations || [];
    const activeOrganization = organizations.find((item: any) => item.id === organizationId) || null;

    setText(statOrganizations, String(organizations.length));
    setText(
      statOrganizationsTrend,
      organizations.length ? "Organizacoes carregadas com sucesso" : "Nenhuma organizacao cadastrada"
    );
    setText(statActiveOrganization, activeOrganization?.name || "--");
    setText(
      statActiveOrganizationTrend,
      activeOrganization ? activeOrganization.cnpj || "Organizacao selecionada" : "Defina uma organizacao ativa"
    );

    if (!organizationId) {
      setText(statInvoices, "--");
      setText(statInvoicesTrend, "Selecione uma organizacao");

      renderList(nextActions, [
        itemRow(
          "Selecionar organizacao",
          "O token ja existe, mas ainda falta definir organization_id para consultar invoices.",
          "Acao necessaria",
          "warning"
        ),
      ]);
      renderInvoices([]);
      return;
    }

    const invoicesResponse = await listInvoices();
    const invoices = invoicesResponse.invoices || [];

    setText(statInvoices, String(invoices.length));
    setText(
      statInvoicesTrend,
      invoices.length ? "Notas retornadas pela API" : "Sem notas para a organizacao atual"
    );

    const actions: string[] = [];

    if (!organizations.length) {
      actions.push(
        itemRow(
          "Cadastrar organizacao",
          "A sessao esta autenticada, mas o usuario ainda nao possui organizacoes disponiveis.",
          "Cadastro pendente",
          "warning"
        )
      );
    }

    if (!invoices.length) {
      actions.push(
        itemRow(
          "Importar XML",
          "A organizacao esta valida, mas ainda nao ha notas importadas para alimentar o painel.",
          "Proxima acao",
          "warning"
        )
      );
    }

    if (organizations.length && invoices.length) {
      actions.push(
        itemRow(
          "Ambiente operacional",
          "A dashboard ja esta lendo organizacoes e notas reais da API.",
          "Conectado",
          "success"
        )
      );
    }

    renderList(nextActions, actions);
    renderInvoices(invoices);
  } catch (error) {
    setText(statOrganizations, "Erro");
    setText(statOrganizationsTrend, "Falha ao consultar organizacoes");
    setText(statActiveOrganization, "Erro");
    setText(statActiveOrganizationTrend, "Revise token e organization_id");
    setText(statInvoices, "Erro");
    setText(statInvoicesTrend, "Falha ao consultar invoices");

    renderList(nextActions, [
      itemRow(
        "Falha nas rotas protegidas",
        `A API respondeu, mas as chamadas autenticadas falharam: ${String(error)}`,
        "Erro",
        "danger"
      ),
    ]);

    if (invoicesPanel) {
      invoicesPanel.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Nao foi possivel carregar as notas.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

loadDashboard();
