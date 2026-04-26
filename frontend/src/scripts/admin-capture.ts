import { getOrganizationId, getToken } from "../lib/auth";
import {
  acceptCaptureCandidate,
  acceptProductReviews,
  listCaptureCandidates,
  listProductReviews,
  type CaptureCandidate,
  type ProductReview,
} from "../lib/admin-capture";

const statsBox = document.getElementById("capture-stats");
const listBox = document.getElementById("capture-rules-list");
const feedback = document.getElementById("capture-feedback");
const productReviewFeedback = document.getElementById("product-review-feedback");
const productReviewList = document.getElementById("product-review-list");
const reviewProductsButton = document.getElementById("review-products-button") as HTMLButtonElement | null;
const acceptReadyProductsButton = document.getElementById("accept-ready-products-button") as HTMLButtonElement | null;

let cachedItems: CaptureCandidate[] = [];
let productReviews: ProductReview[] = [];

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

function setProductReviewFeedback(message: string, tone: "muted" | "success" | "error" = "muted") {
  if (!productReviewFeedback) return;

  productReviewFeedback.textContent = message;
  productReviewFeedback.className =
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

function displayValue(value: unknown, fallback = "-") {
  const normalized = String(value || "").trim();
  return escapeHTML(normalized || fallback);
}

function confidencePercent(value: number) {
  return `${Math.round((Number(value) || 0) * 100)}%`;
}

function captureCell(label: string, value: unknown) {
  return `
    <span class="capture-tax-cell">
      <small>${escapeHTML(label)}</small>
      <strong>${displayValue(value)}</strong>
    </span>
  `;
}

function captureTaxRow(title: string, tone: string, cells: string[]) {
  return `
    <tr class="capture-tax-row capture-tax-row--${tone}">
      <th scope="row">${escapeHTML(title)}</th>
      <td>${cells.join("")}</td>
    </tr>
  `;
}

function renderCaptureTaxTable(item: CaptureCandidate) {
  return `
    <table class="capture-tax-table">
      <tbody>
        ${captureTaxRow("Classificacao", "teal", [
          captureCell("NCM", item.ncm),
          captureCell("CEST", item.cest),
          captureCell("CFOP", item.cfop),
          captureCell("Natureza", item.operation_nature),
        ])}
        ${captureTaxRow("ICMS observado", "amber", [
          captureCell("CST", item.icms_cst),
          captureCell("CSOSN", item.csosn),
          captureCell("Aliquota", formatRate(item.icms_rate)),
          captureCell("Valor", item.icms_value),
        ])}
        ${captureTaxRow("PIS e COFINS", "sky", [
          captureCell("PIS CST", item.pis_cst),
          captureCell("Aliq. PIS", formatRate(item.pis_rate)),
          captureCell("COFINS CST", item.cofins_cst),
          captureCell("Aliq. COFINS", formatRate(item.cofins_rate)),
        ])}
      </tbody>
    </table>
  `;
}

function renderReviewTaxTable(suggestion: Record<string, any>) {
  return `
    <table class="capture-tax-table capture-tax-table--compact">
      <tbody>
        ${captureTaxRow("Identidade", "teal", [
          captureCell("NCM", suggestion.ncm),
          captureCell("CEST", suggestion.cest),
          captureCell("cClasTrib", suggestion.cclas_trib),
          captureCell("CFOP", suggestion.cfop),
        ])}
        ${captureTaxRow("Tributos", "sky", [
          captureCell("ICMS", suggestion.csosn || suggestion.icms_cst),
          captureCell("PIS", suggestion.pis_cst),
          captureCell("COFINS", suggestion.cofins_cst),
          captureCell("IBS/CBS", `${suggestion.ibs_rate || "-"} / ${suggestion.cbs_rate || "-"}`),
        ])}
      </tbody>
    </table>
  `;
}

function reviewChip(label: string, value: unknown) {
  return `
    <span class="review-chip">
      <small>${escapeHTML(label)}</small>
      <strong>${displayValue(value)}</strong>
    </span>
  `;
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

              ${renderCaptureTaxTable(item)}
            </article>
          `
        )
        .join("")}
    </div>
  `;
}

function renderProductReviews(items: ProductReview[]) {
  if (!productReviewList) return;

  const ready = items.filter((item) => item.can_accept && item.confidence_score >= 0.7);
  if (acceptReadyProductsButton) {
    acceptReadyProductsButton.disabled = ready.length === 0;
    acceptReadyProductsButton.textContent = ready.length
      ? `Aceitar ${ready.length} confiaveis em lote`
      : "Aceitar confiaveis em lote";
  }

  if (!items.length) {
    productReviewList.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhum produto cadastrado para revisar.</strong>
        <p>Importe notas ou cadastre produtos para alimentar essa fila automatica.</p>
      </div>
    `;
    return;
  }

  productReviewList.innerHTML = `
    <div class="capture-list">
      ${items
        .map((item) => {
          const suggestion = item.suggestion || {};
          return `
            <article class="product-review-card product-review-card--${escapeHTML(item.status || "review")}">
              <div class="product-review-card__head">
                <div>
                  <span class="capture-card__eyebrow">${displayValue(item.product_code || item.gtin, "Produto cadastrado")}</span>
                  <h4>${displayValue(item.description, "Produto sem descricao")}</h4>
                  <p>GTIN ${displayValue(item.gtin)} · Status ${displayValue(item.status)}</p>
                </div>
                <div class="capture-actions">
                  <span class="confidence-pill">${confidencePercent(item.confidence_score)}</span>
                  <button class="primary-button" type="button" data-accept-product-review="${escapeHTML(item.product_id)}" ${item.can_accept ? "" : "disabled"}>
                    Aceitar
                  </button>
                </div>
              </div>
              ${renderReviewTaxTable(suggestion)}
              ${
                Array.isArray(item.warnings) && item.warnings.length
                  ? `<p>${escapeHTML(item.warnings.slice(0, 2).join(" "))}</p>`
                  : ""
              }
            </article>
          `;
        })
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

async function handleReviewProducts() {
  if (!reviewProductsButton) return;
  reviewProductsButton.disabled = true;
  reviewProductsButton.textContent = "Revisando...";
  setProductReviewFeedback("Rodando motor tributario sobre os produtos cadastrados...", "muted");

  try {
    const response = await listProductReviews(150);
    productReviews = Array.isArray(response?.items) ? response.items : [];
    renderProductReviews(productReviews);
    setProductReviewFeedback(`Revisao concluida: ${productReviews.length} produtos analisados.`, "success");
  } catch (error) {
    setProductReviewFeedback(`Falha na revisao automatica: ${String(error)}`, "error");
  } finally {
    reviewProductsButton.disabled = false;
    reviewProductsButton.textContent = "Revisar produtos";
  }
}

async function handleAcceptProductReviews(productIds: string[], acceptAll = false) {
  if (acceptReadyProductsButton) {
    acceptReadyProductsButton.disabled = true;
  }
  setProductReviewFeedback("Aceitando revisoes e gravando perfis fiscais revisados...", "muted");

  try {
    const response = await acceptProductReviews({
      product_ids: productIds,
      accept_all: acceptAll,
      min_confidence: 0.7,
    });
    setProductReviewFeedback(`${response.accepted || 0} produto(s) aceitos. ${response.failures?.length ? `${response.failures.length} falharam.` : ""}`, "success");
    await handleReviewProducts();
  } catch (error) {
    setProductReviewFeedback(`Falha ao aceitar revisoes: ${String(error)}`, "error");
  } finally {
    renderProductReviews(productReviews);
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

productReviewList?.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof HTMLElement)) return;

  const button = target.closest<HTMLButtonElement>("[data-accept-product-review]");
  if (!button) return;

  const productId = button.dataset.acceptProductReview || "";
  if (!productId) return;

  void handleAcceptProductReviews([productId], false);
});

reviewProductsButton?.addEventListener("click", () => {
  void handleReviewProducts();
});

acceptReadyProductsButton?.addEventListener("click", () => {
  void handleAcceptProductReviews([], true);
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
