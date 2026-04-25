import { listCEST, type CESTItem } from "../lib/cest";

const queryInput = document.getElementById("cest-filter") as HTMLInputElement | null;
const ncmInput = document.getElementById("cest-ncm") as HTMLInputElement | null;
const searchButton = document.getElementById("cest-search") as HTMLButtonElement | null;
const resultBox = document.getElementById("cest-result");
const statsBox = document.getElementById("cest-stats");

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function renderStats(items: CESTItem[]) {
  if (!statsBox) return;

  const withNCM = items.filter((item) => item.ncm_code).length;
  const segments = new Set(items.map((item) => item.segment).filter(Boolean)).size;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Total</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Com NCM</span>
      <strong>${withNCM}</strong>
    </article>
    <article class="stat-card stat-card--amber">
      <span>Segmentos</span>
      <strong>${segments}</strong>
    </article>
  `;
}

function renderList(items: CESTItem[]) {
  if (!resultBox) return;

  if (!items.length) {
    resultBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhum CEST encontrado.</strong>
        <p>Importe a tabela CEST em Fiscal Lab ou ajuste os filtros de busca.</p>
      </div>
    `;
    return;
  }

  resultBox.innerHTML = `
    <div class="cest-list">
      ${items
        .map(
          (item) => `
            <article class="cest-row" data-filter-item>
              <div class="cest-row__top">
                <strong>${escapeHtml(item.code)}</strong>
                <span>${escapeHtml(item.ncm_code || "NCM aberto")}</span>
              </div>
              <p>${escapeHtml(item.description || "-")}</p>
              <div class="cest-row__meta">
                ${item.segment ? `<span>${escapeHtml(item.segment)}</span>` : ""}
                ${item.legal_source ? `<span>${escapeHtml(item.legal_source)}</span>` : ""}
              </div>
            </article>
          `
        )
        .join("")}
    </div>
  `;
}

async function loadList() {
  if (searchButton) {
    searchButton.disabled = true;
    searchButton.textContent = "Buscando...";
  }

  try {
    const response = await listCEST({
      q: queryInput?.value?.trim() || "",
      ncm: ncmInput?.value?.trim() || "",
      limit: 150,
    });

    const items = Array.isArray(response?.items) ? response.items : [];
    renderStats(items);
    renderList(items);
  } catch (error) {
    if (resultBox) {
      resultBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar catalogo CEST.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  } finally {
    if (searchButton) {
      searchButton.disabled = false;
      searchButton.textContent = "Buscar";
    }
  }
}

searchButton?.addEventListener("click", () => void loadList());

[queryInput, ncmInput].forEach((input) => {
  input?.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      void loadList();
    }
  });
});

void loadList();
