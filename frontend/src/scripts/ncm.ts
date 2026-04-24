import { getNCMByCode, listNCM, searchNCM } from "../lib/ncm";

const queryInput = document.getElementById("ncm-query") as HTMLInputElement | null;
const searchButton = document.getElementById("ncm-search");
const listBox = document.getElementById("ncm-list");
const detailBox = document.getElementById("ncm-detail");

let items: any[] = [];

function normalizeCode(value: string) {
  return value.replace(/\D/g, "");
}

function isNotFoundError(error: unknown) {
  return String(error).toLowerCase().includes("ncm not found");
}

function renderDetail(item: any) {
  if (!detailBox || !item) return;

  detailBox.innerHTML = `
    <div class="detail-card">
      <strong>${item.code || "-"}</strong>
      <p>${item.full_description || item.description || "-"}</p>
      <div class="detail-grid">
        <span><strong>Capitulo:</strong> ${item.chapter_code || "-"}</span>
        <span><strong>Posicao:</strong> ${item.heading_code || "-"}</span>
        <span><strong>Item:</strong> ${item.item_code || "-"}</span>
        <span><strong>EX:</strong> ${item.ex_code || "-"}</span>
        <span><strong>Vigencia inicio:</strong> ${item.start_date || "-"}</span>
        <span><strong>Vigencia fim:</strong> ${item.end_date || "-"}</span>
        <span><strong>Fonte legal:</strong> ${item.legal_source || "-"}</span>
        <span><strong>Referencia:</strong> ${item.legal_reference || "-"}</span>
      </div>
      <p class="detail-notes">${item.official_notes || "Sem notas oficiais cadastradas."}</p>
    </div>
  `;
}

function renderList(list: any[]) {
  if (!listBox) return;

  if (!list.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhum NCM encontrado.</strong>
        <p>Tente outro termo ou verifique se a base foi importada corretamente.</p>
      </div>
    `;
    return;
  }

  const rows = list
    .map(
      (item) => `
        <tr>
          <td><button class="link-button" data-code="${item.code}">${item.code}</button></td>
          <td>${item.description || "-"}</td>
          <td>${item.level_type || "-"}</td>
          <td>${item.ex_code || "-"}</td>
        </tr>
      `
    )
    .join("");

  listBox.innerHTML = `
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Codigo</th>
            <th>Descricao</th>
            <th>Nivel</th>
            <th>EX</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    </div>
  `;

  listBox.querySelectorAll(".link-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const code = button.getAttribute("data-code");
      if (!code) return;
      await loadDetail(code);
    });
  });
}

async function loadDetail(code: string) {
  try {
    const response = await getNCMByCode(code);
    renderDetail(response.item);
  } catch (error) {
    if (detailBox) {
      detailBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar o detalhe do NCM.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

async function runSearch() {
  const q = (queryInput?.value || "").trim();
  const normalizedCode = normalizeCode(q);

  try {
    if (!q) {
      const response = await listNCM(100);
      items = Array.isArray(response?.items) ? response.items : [];
    } else {
      try {
        if (/^\d{4,8}$/.test(normalizedCode)) {
          const exact = await getNCMByCode(normalizedCode);
          items = exact.item ? [exact.item] : [];

          if (exact.item) {
            renderDetail(exact.item);
          }
        } else {
          const response = await searchNCM(q, 100);
          items = Array.isArray(response?.items) ? response.items : [];
        }
      } catch (error) {
        if (!isNotFoundError(error) || !normalizedCode) {
          throw error;
        }

        const fallback = await searchNCM(normalizedCode, 100);
        items = Array.isArray(fallback?.items) ? fallback.items : [];

        if (detailBox) {
          detailBox.innerHTML = `
            <div class="dashboard-empty">
              <strong>Nenhum NCM exato encontrado.</strong>
              <p>Mostrando resultados proximos para ${normalizedCode}.</p>
            </div>
          `;
        }
      }
    }

    renderList(items);
  } catch (error) {
    if (listBox) {
      listBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao consultar catalogo NCM.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

searchButton?.addEventListener("click", runSearch);
queryInput?.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    runSearch();
  }
});

runSearch();
