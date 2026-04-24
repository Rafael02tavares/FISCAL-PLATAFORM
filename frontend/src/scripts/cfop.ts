import { findCFOPByCode, listCFOP, type CFOPItem } from "../lib/cfop";

const queryInput = document.getElementById("cfop-filter") as HTMLInputElement | null;
const typeInput = document.getElementById("cfop-type") as HTMLSelectElement | null;
const searchButton = document.getElementById("cfop-search") as HTMLButtonElement | null;
const resultBox = document.getElementById("cfop-result");
const detailBox = document.getElementById("cfop-detail");
const statsBox = document.getElementById("cfop-stats");

let items: CFOPItem[] = [];
let selectedCode = "";

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function toneLabel(value: boolean, yes: string, no = "Nao") {
  return value ? yes : no;
}

function formatOperationType(value: string) {
  if (value === "entrada") return "Entrada";
  if (value === "saida") return "Saida";
  return "Nao definido";
}

function renderStats(list: CFOPItem[]) {
  if (!statsBox) return;

  const entrada = list.filter((item) => item.operation_type === "entrada").length;
  const saida = list.filter((item) => item.operation_type === "saida").length;
  const devolucao = list.filter((item) => item.ind_devolution).length;
  const transporte = list.filter((item) => item.ind_transport).length;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Total</span>
      <strong>${list.length}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Entradas</span>
      <strong>${entrada}</strong>
    </article>
    <article class="stat-card stat-card--amber">
      <span>Saidas</span>
      <strong>${saida}</strong>
    </article>
    <article class="stat-card stat-card--rose">
      <span>Devolucao</span>
      <strong>${devolucao}</strong>
    </article>
    <article class="stat-card stat-card--slate">
      <span>Transporte</span>
      <strong>${transporte}</strong>
    </article>
  `;
}

function renderDetail(item?: CFOPItem | null) {
  if (!detailBox) return;

  if (!item) {
    detailBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Selecione um CFOP.</strong>
        <p>Clique em um codigo da lista para ver a descricao e os sinais operacionais.</p>
      </div>
    `;
    return;
  }

  detailBox.innerHTML = `
    <article class="cfop-detail-card">
      <div class="cfop-detail-card__hero">
        <span class="cfop-code-badge">${escapeHtml(item.code)}</span>
        <span class="cfop-type-pill cfop-type-pill--${item.operation_type || "neutral"}">${formatOperationType(item.operation_type)}</span>
      </div>

      <div>
        <h3>${escapeHtml(item.description || "-")}</h3>
        <p class="cfop-detail-card__description">
          Esse CFOP esta disponivel no catalogo importado e pode ser usado como referencia na classificacao das operacoes fiscais.
        </p>
      </div>

      <div class="cfop-signal-grid">
        <div><span>NF-e</span><strong>${toneLabel(item.ind_nfe, "Permitido")}</strong></div>
        <div><span>Comunicacao</span><strong>${toneLabel(item.ind_communication, "Sim")}</strong></div>
        <div><span>Transporte</span><strong>${toneLabel(item.ind_transport, "Sim")}</strong></div>
        <div><span>Devolucao</span><strong>${toneLabel(item.ind_devolution, "Sim")}</strong></div>
      </div>

      <div class="cfop-hint-list">
        <span class="hint-chip">${item.ind_devolution ? "Fluxo de devolucao" : "Fluxo regular"}</span>
        <span class="hint-chip">${item.ind_transport ? "Ligado a transporte" : "Nao focado em transporte"}</span>
        <span class="hint-chip">${item.ind_communication ? "Servico de comunicacao" : "Nao comunicacao"}</span>
      </div>
    </article>
  `;
}

function renderList(list: CFOPItem[]) {
  if (!resultBox) return;

  if (!list.length) {
    resultBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhum CFOP encontrado.</strong>
        <p>Ajuste os filtros ou importe uma tabela de CFOP em Administracao.</p>
      </div>
    `;
    renderDetail(null);
    return;
  }

  const rows = list
    .map((item) => {
      const selectedClass = item.code === selectedCode ? "cfop-row--selected" : "";
      return `
        <button class="cfop-row ${selectedClass}" data-cfop-code="${escapeHtml(item.code)}" type="button">
          <div class="cfop-row__main">
            <div class="cfop-row__top">
              <strong>${escapeHtml(item.code)}</strong>
              <span class="cfop-type-pill cfop-type-pill--${item.operation_type || "neutral"}">${formatOperationType(item.operation_type)}</span>
            </div>
            <p>${escapeHtml(item.description || "-")}</p>
          </div>
          <div class="cfop-row__flags">
            ${item.ind_devolution ? '<span class="mini-flag mini-flag--rose">Devolucao</span>' : ""}
            ${item.ind_transport ? '<span class="mini-flag mini-flag--slate">Transporte</span>' : ""}
            ${item.ind_communication ? '<span class="mini-flag mini-flag--amber">Comunicacao</span>' : ""}
          </div>
        </button>
      `;
    })
    .join("");

  resultBox.innerHTML = `<div class="cfop-list">${rows}</div>`;

  resultBox.querySelectorAll<HTMLButtonElement>("[data-cfop-code]").forEach((button) => {
    button.addEventListener("click", async () => {
      const code = button.dataset.cfopCode || "";
      if (!code) return;
      selectedCode = code;
      renderList(list);
      await loadDetail(code);
    });
  });
}

async function loadDetail(code: string) {
  try {
    const response = await findCFOPByCode(code);
    renderDetail(response?.item || null);
  } catch (error) {
    if (!detailBox) return;
    detailBox.innerHTML = `
      <div class="dashboard-empty dashboard-empty--error">
        <strong>Falha ao carregar detalhe do CFOP.</strong>
        <p>${String(error)}</p>
      </div>
    `;
  }
}

async function loadList() {
  if (searchButton) {
    searchButton.disabled = true;
    searchButton.textContent = "Buscando...";
  }

  try {
    const response = await listCFOP({
      q: queryInput?.value?.trim() || "",
      operationType: typeInput?.value || "",
      limit: 150,
    });

    items = Array.isArray(response?.items) ? response.items : [];
    renderStats(items);

    if (!selectedCode && items[0]?.code) {
      selectedCode = items[0].code;
    }

    const selected = items.find((item) => item.code === selectedCode) || items[0] || null;
    selectedCode = selected?.code || "";

    renderList(items);
    renderDetail(selected);

    if (selected?.code) {
      await loadDetail(selected.code);
    }
  } catch (error) {
    if (resultBox) {
      resultBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar catalogo de CFOP.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
    renderDetail(null);
  } finally {
    if (searchButton) {
      searchButton.disabled = false;
      searchButton.textContent = "Buscar";
    }
  }
}

searchButton?.addEventListener("click", () => {
  selectedCode = "";
  void loadList();
});

queryInput?.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    selectedCode = "";
    void loadList();
  }
});

typeInput?.addEventListener("change", () => {
  selectedCode = "";
  void loadList();
});

void loadList();
