import { getOrganizationCRT, getOrganizationId, getOrganizationTaxRegime, getToken } from "../lib/auth";
import { listFiscalOperations } from "../lib/fiscaloperations";
import { getInvoice } from "../lib/invoices";
import { suggestTax } from "../lib/tax";

const wrapper = document.getElementById("invoice-detail");
const PRODUCT_PREFILL_KEY = "catalog_product_prefill";
const id =
  wrapper instanceof HTMLElement ? wrapper.dataset.invoiceId || "" : "";

function confidenceLabel(score: unknown) {
  const n = Number(score || 0);
  if (n >= 0.9) return "Alta";
  if (n >= 0.7) return "Media";
  return "Baixa";
}

function confidenceClass(score: unknown) {
  const n = Number(score || 0);
  if (n >= 0.9) return "confidence-high";
  if (n >= 0.7) return "confidence-medium";
  return "confidence-low";
}

function formatDate(value: string) {
  if (!value) return "-";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(parsed);
}

function formatMoney(value: unknown) {
  const numeric = Number(value || 0);
  if (Number.isNaN(numeric)) return String(value || "-");
  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  }).format(numeric);
}

function normalizeValue(value: unknown) {
  return String(value || "").trim().toUpperCase();
}

function displayValue(value: unknown) {
  const normalized = String(value || "").trim();
  return normalized || "-";
}

function saveCatalogPrefill(payload: Record<string, string>) {
  try {
    localStorage.setItem(PRODUCT_PREFILL_KEY, JSON.stringify(payload));
  } catch {
    // ignora indisponibilidade de storage local
  }
}

function buildCatalogPrefill(item: any, invoice: any) {
  return {
    product_code: String(item?.product_code || "").trim(),
    gtin: String(item?.gtin || "").trim(),
    description: String(item?.description || "").trim(),
    ncm: String(item?.ncm || "").trim(),
    cest: String(item?.cest || "").trim(),
    cfop: String(item?.cfop || "").trim(),
    icms_cst: String(item?.icms_cst || item?.cst || "").trim(),
    csosn: String(item?.csosn || "").trim(),
    pis_cst: String(item?.pis_cst || "").trim(),
    cofins_cst: String(item?.cofins_cst || "").trim(),
    icms_value: String(item?.icms_value || "").trim(),
    ipi_value: String(item?.ipi_value || "").trim(),
    pis_value: String(item?.pis_value || "").trim(),
    cofins_value: String(item?.cofins_value || "").trim(),
    pis_rate: String(item?.pis_rate || "").trim(),
    cofins_rate: String(item?.cofins_rate || "").trim(),
    icms_rate: String(item?.icms_rate || "").trim(),
    emitter_uf: String(invoice?.emitter_uf || "").trim().toUpperCase(),
    recipient_uf: String(invoice?.recipient_uf || "").trim().toUpperCase(),
    operation_nature: String(invoice?.operation_nature || "").trim(),
    target_tax_regime: String(getOrganizationTaxRegime() || "").trim(),
    target_crt: String(getOrganizationCRT() || "").trim(),
  };
}

type FiscalOperation = {
  code?: string;
  default_cfop?: string;
  is_default?: boolean;
};

async function resolveOperationCode(item: any) {
  const response = await listFiscalOperations();
  const operations = Array.isArray(response?.operations)
    ? (response.operations as FiscalOperation[])
    : [];

  const itemCFOP = normalizeValue(item?.cfop);
  const matchedByCFOP = operations.find(
    (operation) => normalizeValue(operation.default_cfop) === itemCFOP
  );

  if (matchedByCFOP?.code) {
    return matchedByCFOP.code;
  }

  const defaultOperation = operations.find((operation) => operation.is_default);
  if (defaultOperation?.code) {
    return defaultOperation.code;
  }

  return "";
}

function renderLegalBasis(legalBasis: any[]) {
  if (!legalBasis || !legalBasis.length) {
    return `
      <div class="compare-box">
        <h4>Base legal</h4>
        <p>Nenhuma base legal especifica encontrada para este contexto.</p>
      </div>
    `;
  }

  const rows = legalBasis
    .map(
      (item) => `
        <div class="legal-item">
          <div><strong>${item.tax_type || "-"}</strong></div>
          <div><strong>Fonte:</strong> ${item.title || "-"}</div>
          <div><strong>Referencia:</strong> ${item.reference_code || "-"}</div>
          <div><strong>Jurisdicao:</strong> ${item.jurisdiction || "-"}</div>
          <div><strong>UF:</strong> ${item.uf || "-"}</div>
          <div><strong>Motivo:</strong> ${item.applied_reason || "-"}</div>
          <div><strong>Peso:</strong> ${item.weight || "-"}</div>
        </div>
      `
    )
    .join("");

  return `
    <div class="compare-box legal-basis-box">
      <h4>Base legal aplicada</h4>
      <div class="legal-list">${rows}</div>
    </div>
  `;
}

function renderSuggestion(result: any, item: any) {
  const suggestion = result?.suggestion || {};
  const confidence = Number(result?.confidence_score || 0);
  const confidencePercent = `${Math.round(confidence * 100)}%`;
  const warnings = Array.isArray(result?.warnings) ? result.warnings : [];

  return `
    <div class="suggestion-card">
      <div class="suggestion-top">
        <div class="suggestion-heading">
          <span class="eyebrow">Analise comparativa</span>
          <strong>Sugestao fiscal do item</strong>
          <div class="muted-line">
            Baseada no historico aprendido pela plataforma e nas regras legais
          </div>
        </div>

        <div class="confidence-badge ${confidenceClass(result.confidence_score)}">
          Confianca ${confidenceLabel(result.confidence_score)} · ${confidencePercent}
        </div>
      </div>

      <div class="suggestion-summary">
        <div class="summary-chip summary-chip--neutral">
          <span class="summary-chip__label">Match</span>
          <strong>${displayValue(result.match_type)}</strong>
        </div>
        <div class="summary-chip summary-chip--blue">
          <span class="summary-chip__label">Operacao usada</span>
          <strong>${displayValue(result.selected_operation?.name || result.selected_operation?.code)}</strong>
        </div>
        <div class="summary-chip summary-chip--green">
          <span class="summary-chip__label">CFOP sugerido</span>
          <strong>${displayValue(suggestion.cfop || item.cfop)}</strong>
        </div>
        <div class="summary-chip summary-chip--amber">
          <span class="summary-chip__label">NCM sugerido</span>
          <strong>${displayValue(suggestion.ncm || item.ncm)}</strong>
        </div>
      </div>

      ${
        warnings.length
          ? `
            <div class="warning-stack">
              ${warnings
                .map(
                  (warning) => `
                    <div class="warning-chip">${displayValue(warning)}</div>
                  `
                )
                .join("")}
            </div>
          `
          : ""
      }

      <div class="suggestion-grid">
        <div class="compare-box compare-box--source">
          <h4>O que veio na nota</h4>
          <div class="compare-list">
            <div class="compare-row">
              <span>Descricao</span>
              <strong>${displayValue(item.description)}</strong>
            </div>
            <div class="compare-row">
              <span>GTIN</span>
              <strong>${displayValue(item.gtin)}</strong>
            </div>
            <div class="compare-row">
              <span>NCM</span>
              <strong>${displayValue(item.ncm)}</strong>
            </div>
            <div class="compare-row">
              <span>CFOP</span>
              <strong>${displayValue(item.cfop)}</strong>
            </div>
            <div class="compare-row">
              <span>ICMS CST</span>
              <strong>${displayValue(item.icms_cst || item.cst)}</strong>
            </div>
            <div class="compare-row">
              <span>Aliq. ICMS</span>
              <strong>${displayValue(item.icms_rate)}</strong>
            </div>
            <div class="compare-row">
              <span>PIS CST</span>
              <strong>${displayValue(item.pis_cst)}</strong>
            </div>
            <div class="compare-row">
              <span>COFINS CST</span>
              <strong>${displayValue(item.cofins_cst)}</strong>
            </div>
          </div>
        </div>

        <div class="compare-box compare-box--result">
          <h4>O que a plataforma recomenda</h4>
          <div class="compare-list">
            <div class="compare-row">
              <span>NCM sugerido</span>
              <strong>${displayValue(suggestion.ncm || item.ncm)}</strong>
            </div>
            <div class="compare-row">
              <span>CEST sugerido</span>
              <strong>${displayValue(suggestion.cest)}</strong>
            </div>
            <div class="compare-row">
              <span>cClasTrib</span>
              <strong>${displayValue(suggestion.cclas_trib)}</strong>
            </div>
            <div class="compare-row">
              <span>CFOP sugerido</span>
              <strong>${displayValue(suggestion.cfop || item.cfop)}</strong>
            </div>
            <div class="compare-row">
              <span>ICMS CST</span>
              <strong>${displayValue(suggestion.icms_cst)}</strong>
            </div>
            <div class="compare-row">
              <span>CSOSN</span>
              <strong>${displayValue(suggestion.csosn)}</strong>
            </div>
            <div class="compare-row">
              <span>PIS CST</span>
              <strong>${displayValue(suggestion.pis_cst)}</strong>
            </div>
            <div class="compare-row">
              <span>COFINS CST</span>
              <strong>${displayValue(suggestion.cofins_cst)}</strong>
            </div>
            <div class="compare-row">
              <span>Cod. rec. PIS</span>
              <strong>${displayValue(suggestion.pis_revenue_code)}</strong>
            </div>
            <div class="compare-row">
              <span>Cod. rec. COFINS</span>
              <strong>${displayValue(suggestion.cofins_revenue_code)}</strong>
            </div>
          </div>
        </div>
      </div>

      <div class="tax-breakdown">
        <div class="tax-breakdown__header">
          <h4>Resumo tributario sugerido</h4>
          <span>Valores e aliquotas destacados para leitura rapida</span>
        </div>

        <div class="tax-pill-grid">
          <div class="tax-pill">
            <span>ICMS</span>
            <strong>${formatMoney(suggestion.icms_value)}</strong>
          </div>
          <div class="tax-pill">
            <span>IPI</span>
            <strong>${formatMoney(suggestion.ipi_value)}</strong>
          </div>
          <div class="tax-pill">
            <span>PIS</span>
            <strong>${formatMoney(suggestion.pis_value)}</strong>
          </div>
          <div class="tax-pill">
            <span>COFINS</span>
            <strong>${formatMoney(suggestion.cofins_value)}</strong>
          </div>
          <div class="tax-pill tax-pill--soft">
            <span>IBS</span>
            <strong>${displayValue(suggestion.ibs_rate)}</strong>
          </div>
          <div class="tax-pill tax-pill--soft">
            <span>CBS</span>
            <strong>${displayValue(suggestion.cbs_rate)}</strong>
          </div>
          <div class="tax-pill tax-pill--soft">
            <span>Aliq. ICMS</span>
            <strong>${displayValue(suggestion.icms_rate)}</strong>
          </div>
          <div class="tax-pill tax-pill--soft">
            <span>Aliq. PIS</span>
            <strong>${displayValue(suggestion.pis_rate)}</strong>
          </div>
          <div class="tax-pill tax-pill--soft">
            <span>Aliq. COFINS</span>
            <strong>${displayValue(suggestion.cofins_rate)}</strong>
          </div>
        </div>
      </div>

      <div class="legal-section">
        ${renderLegalBasis(result.legal_basis || [])}
      </div>
    </div>
  `;
}

async function loadInvoice() {
  const token = getToken();
  const organizationId = getOrganizationId();

  if (!token) {
    window.location.href = "/login";
    return;
  }

  if (!organizationId) {
    window.location.href = "/organizations";
    return;
  }

  if (!wrapper) return;

  if (!id) {
    wrapper.innerHTML = `
      <p>Erro ao carregar a nota.</p>
      <p>O identificador da invoice nao foi encontrado na pagina.</p>
    `;
    return;
  }

  try {
    const data = await getInvoice(id);
    const invoice = data.invoice;
    const items = Array.isArray(invoice.items) ? invoice.items : [];

    if (!invoice) {
      wrapper.innerHTML = "<p>Nota nao encontrada.</p>";
      return;
    }

    const rows = items
      .map(
        (item, index) => `
          <tr>
            <td>${item.item_number || index + 1}</td>
            <td>
              <div><strong>${item.description || "-"}</strong></div>
              <div class="small-muted">GTIN: ${item.gtin || "-"}</div>
            </td>
            <td>${item.ncm || "-"}</td>
            <td>${item.cfop || "-"}</td>
            <td>
              <div class="action-stack">
                <button class="suggest-btn" data-index="${index}">
                  Sugerir
                </button>
                <button class="suggest-btn suggest-btn--secondary" data-prefill-index="${index}">
                  Preencher cadastro
                </button>
              </div>
            </td>
          </tr>
          <tr>
            <td colspan="5">
              <div id="suggestion-${index}" class="suggestion-slot" style="display:none;"></div>
            </td>
          </tr>
        `
      )
      .join("");

    wrapper.innerHTML = `
      <div class="invoice-meta-grid">
        <div><strong>Numero:</strong> ${invoice.number || "-"}</div>
        <div><strong>Serie:</strong> ${invoice.series || "-"}</div>
        <div><strong>Data:</strong> ${formatDate(invoice.issued_at)}</div>
        <div><strong>Status:</strong> ${invoice.status || "-"}</div>
        <div><strong>Emitente:</strong> ${invoice.emitter_name || "-"}</div>
        <div><strong>Destinatario:</strong> ${invoice.recipient_name || "-"}</div>
        <div><strong>Natureza:</strong> ${invoice.operation_nature || "-"}</div>
        <div><strong>Total:</strong> ${formatMoney(invoice.total_amount)}</div>
      </div>

      <h3 style="margin-top: 20px;" class="section-title">Itens da nota</h3>

      ${
        items.length
          ? `
            <div class="table-wrap">
              <table class="table">
                <thead>
                  <tr>
                    <th>#</th>
                    <th>Produto</th>
                    <th>NCM</th>
                    <th>CFOP</th>
                    <th>Acao</th>
                  </tr>
                </thead>
                <tbody>${rows}</tbody>
              </table>
            </div>
          `
          : "<p>Essa nota ainda nao possui itens carregados.</p>"
      }
    `;

    document.querySelectorAll(".suggest-btn").forEach((button) => {
      if (button.hasAttribute("data-prefill-index")) {
        return;
      }
      button.addEventListener("click", async () => {
        const idx = button.getAttribute("data-index");
        if (idx === null) return;

        const item = items[Number(idx)];
        const target = document.getElementById(`suggestion-${idx}`);
        if (!target || !item) return;

        const originalLabel = button.textContent || "Sugerir";
        button.setAttribute("disabled", "true");
        button.textContent = "Consultando...";
        target.style.display = "block";
        target.innerHTML = `<div class="loading-box">Consultando sugestao fiscal...</div>`;

        try {
          const operationCode = await resolveOperationCode(item);
          const result = await suggestTax({
            gtin: item.gtin || "",
            description: item.description || "",
            ncm_code: item.ncm || "",
            operation_code: operationCode,
            emitter_uf: invoice.emitter_uf || "",
            recipient_uf: invoice.recipient_uf || "",
            tax_regime: getOrganizationTaxRegime() || "",
            target_crt: getOrganizationCRT() || "",
            source_icms_cst: item.icms_cst || item.cst || "",
            source_icms_rate: item.icms_rate || "",
            source_pis_cst: item.pis_cst || "",
            source_pis_rate: item.pis_rate || "",
            source_cofins_cst: item.cofins_cst || "",
            source_cofins_rate: item.cofins_rate || "",
          });

          target.innerHTML = renderSuggestion(result, item);
        } catch (error) {
          target.innerHTML = `
            <div class="error-box">
              Erro ao consultar sugestao: ${String(error)}
            </div>
          `;
        } finally {
          button.removeAttribute("disabled");
          button.textContent = originalLabel;
        }
      });
    });

    document.querySelectorAll("[data-prefill-index]").forEach((button) => {
      button.addEventListener("click", () => {
        const idx = button.getAttribute("data-prefill-index");
        if (idx === null) return;

        const item = items[Number(idx)];
        if (!item) return;

        saveCatalogPrefill(buildCatalogPrefill(item, invoice));
        window.location.href = "/produtos?prefill=invoice";
      });
    });
  } catch (error) {
    wrapper.innerHTML = `
      <p>Erro ao carregar a nota.</p>
      <p>${String(error)}</p>
    `;
  }
}

loadInvoice();
