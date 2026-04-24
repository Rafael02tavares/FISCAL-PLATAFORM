import { getOrganizationId, getToken } from "../lib/auth";
import {
  acceptCaptureCandidate,
  listCaptureCandidates,
  type CaptureCandidate,
} from "../lib/admin-capture";

const statsBox = document.getElementById("capture-stats");
const listBox = document.getElementById("capture-rules-list");
const feedback = document.getElementById("capture-feedback");

let cachedItems: CaptureCandidate[] = [];

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

function formatDate(value: string) {
  if (!value) return "-";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(date);
}

function formatRate(value: string) {
  if (!value) return "-";
  const numeric = Number(value);
  if (Number.isNaN(numeric)) return value;
  return `${numeric.toLocaleString("pt-BR", { maximumFractionDigits: 4 })}%`;
}

function escapeHTML(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function renderStats(items: CaptureCandidate[]) {
  if (!statsBox) return;

  const withICMS = items.filter((item) => item.icms_cst || item.csosn).length;
  const withPisCofins = items.filter((item) => item.pis_cst || item.cofins_cst).length;
  const withCatalogAnchor = items.filter((item) => item.ncm || item.cfop || item.cest).length;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Regras capturadas</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Com ICMS observado</span>
      <strong>${withICMS}</strong>
    </article>
    <article class="stat-card stat-card--amber">
      <span>Com PIS/COFINS observado</span>
      <strong>${withPisCofins}</strong>
    </article>
    <article class="stat-card stat-card--sky">
      <span>Com ancora fiscal</span>
      <strong>${withCatalogAnchor}</strong>
    </article>
  `;
}

function renderItems(items: CaptureCandidate[]) {
  if (!listBox) return;

  renderStats(items);

  if (!items.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma regra capturada para triagem.</strong>
        <p>Importe uma nota fiscal para que o sistema extraia CST, CFOP, NCM e demais pistas tributarias.</p>
      </div>
    `;
    return;
  }

  listBox.innerHTML = `
    <div class="capture-list">
      ${items
        .map(
          (item) => `
            <article class="capture-card">
              <div class="capture-card__head">
                <div>
                  <span class="capture-card__eyebrow">NF ${escapeHTML(item.invoice_number || "-")}${item.invoice_series ? ` · serie ${escapeHTML(item.invoice_series)}` : ""}</span>
                  <h3>${escapeHTML(item.description || "Item sem descricao")}</h3>
                  <p>${escapeHTML(item.emitter_name || "Emitente nao informado")} · ${escapeHTML(formatDate(item.issued_at))}</p>
                </div>
                <button class="primary-button" type="button" data-accept-capture="${escapeHTML(item.invoice_item_id)}">
                  Aceitar regra
                </button>
              </div>

              <div class="capture-card__meta">
                <span>Item ${item.item_number || 0}</span>
                <span>UF ${escapeHTML(item.emitter_uf || "-")} → ${escapeHTML(item.recipient_uf || "-")}</span>
                <span>Codigo ${escapeHTML(item.product_code || "-")}</span>
                <span>GTIN ${escapeHTML(item.gtin || "-")}</span>
              </div>

              <div class="capture-card__grid">
                <section class="capture-panel capture-panel--teal">
                  <h4>Classificacao</h4>
                  <div class="capture-kpis">
                    <p><strong>NCM</strong><span>${escapeHTML(item.ncm || "-")}</span></p>
                    <p><strong>CEST</strong><span>${escapeHTML(item.cest || "-")}</span></p>
                    <p><strong>CFOP</strong><span>${escapeHTML(item.cfop || "-")}</span></p>
                    <p><strong>Natureza</strong><span>${escapeHTML(item.operation_nature || "-")}</span></p>
                  </div>
                </section>

                <section class="capture-panel capture-panel--amber">
                  <h4>ICMS observado</h4>
                  <div class="capture-kpis">
                    <p><strong>CST</strong><span>${escapeHTML(item.icms_cst || "-")}</span></p>
                    <p><strong>CSOSN</strong><span>${escapeHTML(item.csosn || "-")}</span></p>
                    <p><strong>Aliquota</strong><span>${escapeHTML(formatRate(item.icms_rate))}</span></p>
                    <p><strong>Valor</strong><span>${escapeHTML(item.icms_value || "-")}</span></p>
                  </div>
                </section>

                <section class="capture-panel capture-panel--sky">
                  <h4>PIS e COFINS observados</h4>
                  <div class="capture-kpis">
                    <p><strong>PIS CST</strong><span>${escapeHTML(item.pis_cst || "-")}</span></p>
                    <p><strong>PIS Aliquota</strong><span>${escapeHTML(formatRate(item.pis_rate))}</span></p>
                    <p><strong>COFINS CST</strong><span>${escapeHTML(item.cofins_cst || "-")}</span></p>
                    <p><strong>COFINS Aliquota</strong><span>${escapeHTML(formatRate(item.cofins_rate))}</span></p>
                  </div>
                </section>
              </div>
            </article>
          `
        )
        .join("")}
    </div>
  `;
}

async function loadItems() {
  const response = await listCaptureCandidates(150);
  cachedItems = Array.isArray(response?.items) ? response.items : [];
  renderItems(cachedItems);
}

async function handleAccept(invoiceItemId: string, button: HTMLButtonElement) {
  const previousLabel = button.textContent || "Aceitar regra";
  button.disabled = true;
  button.textContent = "Integrando...";

  setFeedback("Integrando regra capturada ao motor tributario...", "muted");

  try {
    await acceptCaptureCandidate(invoiceItemId);
    cachedItems = cachedItems.filter((item) => item.invoice_item_id !== invoiceItemId);
    renderItems(cachedItems);
    setFeedback("Regra capturada aceita e integrada ao motor tributario.", "success");
  } catch (error) {
    setFeedback(`Falha ao aceitar regra capturada: ${String(error)}`, "error");
    button.disabled = false;
    button.textContent = previousLabel;
  }
}

listBox?.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof HTMLElement)) return;

  const button = target.closest<HTMLButtonElement>("[data-accept-capture]");
  if (!button) return;

  const invoiceItemId = button.dataset.acceptCapture || "";
  if (!invoiceItemId) return;

  void handleAccept(invoiceItemId, button);
});

async function bootstrap() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  if (!getOrganizationId()) {
    setFeedback("Selecione uma organizacao antes de triar regras capturadas.", "error");
    return;
  }

  try {
    await loadItems();
    setFeedback("As regras capturadas por importacao de notas aparecem aqui para aceite.", "muted");
  } catch (error) {
    setFeedback(`Falha ao carregar triagem de regras: ${String(error)}`, "error");
    if (listBox) {
      listBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar a triagem de regras.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

void bootstrap();
