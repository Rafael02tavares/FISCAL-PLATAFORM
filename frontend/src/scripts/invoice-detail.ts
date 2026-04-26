import { getOrganizationCRT, getOrganizationId, getOrganizationTaxRegime, getToken } from "../lib/auth";
import { getCBenefAdvice } from "../lib/cbenef-rules";
import { listFiscalOperations } from "../lib/fiscaloperations";
import { getInvoice } from "../lib/invoices";
import { getCSOSNInfo, getICMSCSTInfo, getICMSOriginInfo, getICMSRegimeAdvice, getPISCOFINSCSTInfo, getPISCOFINSRegimeAdvice } from "../lib/fiscal-codes";
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

function fiscalCell(label: string, value: unknown) {
  return `
    <span class="fiscal-cell">
      <small>${label}</small>
      <strong>${displayValue(value)}</strong>
    </span>
  `;
}

function fiscalRow(title: string, tone: string, cells: string[]) {
  return `
    <tr class="fiscal-tax-row fiscal-tax-row--${tone}">
      <th scope="row">${title}</th>
      <td>${cells.join("")}</td>
    </tr>
  `;
}

function renderFiscalTaxTable(suggestion: any, item: any) {
  return `
    <table class="fiscal-tax-table">
      <tbody>
        ${fiscalRow("Nota importada", "neutral", [
          fiscalCell("Descricao", item.description),
          fiscalCell("GTIN", item.gtin),
          fiscalCell("NCM", item.ncm),
          fiscalCell("CFOP", item.cfop),
        ])}
        ${fiscalRow("Identidade fiscal", "teal", [
          fiscalCell("NCM", suggestion.ncm || item.ncm),
          fiscalCell("CEST", suggestion.cest || item.cest),
          fiscalCell("cClasTrib", suggestion.cclas_trib),
          fiscalCell("CNPJ/Origem", item.icms_origin || item.origin || "-"),
        ])}
        ${fiscalRow("Operacao e ICMS", "sky", [
          fiscalCell("CFOP", suggestion.cfop || item.cfop),
          fiscalCell("ICMS CST", suggestion.icms_cst || item.icms_cst || item.cst),
          fiscalCell("CSOSN", suggestion.csosn || item.csosn),
          fiscalCell("Aliq. ICMS", suggestion.icms_rate || item.icms_rate),
        ])}
        ${fiscalRow("PIS e COFINS", "rose", [
          fiscalCell("PIS CST", suggestion.pis_cst || item.pis_cst),
          fiscalCell("COFINS CST", suggestion.cofins_cst || item.cofins_cst),
          fiscalCell("Aliq. PIS", suggestion.pis_rate || item.pis_rate),
          fiscalCell("Aliq. COFINS", suggestion.cofins_rate || item.cofins_rate),
        ])}
        ${fiscalRow("IPI", "amber", [
          fiscalCell("IPI CST", suggestion.ipi_cst),
          fiscalCell("CEnq", suggestion.ipi_cenq),
          fiscalCell("Aliq. IPI", suggestion.ipi_rate),
          fiscalCell("Valor IPI", formatMoney(suggestion.ipi_value)),
        ])}
        ${fiscalRow("Reforma", "teal", [
          fiscalCell("IBS", suggestion.ibs_rate),
          fiscalCell("CBS", suggestion.cbs_rate),
          fiscalCell("Imp. seletivo", suggestion.selective_tax_code),
          fiscalCell("Aliq. seletivo", suggestion.selective_tax_rate),
        ])}
      </tbody>
    </table>
  `;
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
  const inferredCode = inferOperationCodeFromCFOP(itemCFOP);
  const inferredOperation = operations.find(
    (operation) => normalizeValue(operation.code) === inferredCode
  );

  if (inferredOperation?.code) {
    return inferredOperation.code;
  }

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

function inferOperationCodeFromCFOP(cfop: unknown) {
  switch (String(cfop || "").replace(/\D/g, "")) {
    case "5403":
    case "5405":
      return "sale_st_internal";
    case "6403":
    case "6404":
      return "sale_st_interstate";
    case "6101":
    case "6102":
      return "sale_interstate";
    case "5101":
    case "5102":
      return "sale_consumer_final";
    default:
      return "";
  }
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

  const retailDefaults = legalBasis.filter((item) => item?.tax_type === "RETAIL_DEFAULT");
  const legalItems = legalBasis.filter((item) => item?.tax_type !== "RETAIL_DEFAULT");

  const rows = legalBasis
    .map(
      (item) => `
        <div class="legal-item ${item.tax_type === "RETAIL_DEFAULT" ? "legal-item--retail-default" : ""}">
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
      ${
        retailDefaults.length
          ? `
            <div class="retail-default-note">
              <strong>Perfil padrao varejista aplicado</strong>
              <p>A plataforma usou um default operacional para responder a consulta. Cadastre uma regra legal especifica para publicar essa decisao fiscal com seguranca.</p>
            </div>
          `
          : ""
      }
      ${
        retailDefaults.length && !legalItems.length
          ? `<p class="legal-basis-muted">Ainda nao ha regra legal especifica vinculada a este item.</p>`
          : ""
      }
      <div class="legal-list">${rows}</div>
    </div>
  `;
}

function onlyDigits(value: unknown) {
  return String(value || "").replace(/\D/g, "");
}

function isGenericSaleCFOP(value: unknown) {
  return ["5101", "5102", "6101", "6102"].includes(onlyDigits(value));
}

function isSubstitutionTaxCFOP(value: unknown) {
  return ["5403", "5405", "6403", "6404"].includes(onlyDigits(value));
}

function renderFiscalAnalysis(result: any, item: any) {
  const suggestion = result?.suggestion || {};
  const sourceCFOP = onlyDigits(item?.cfop);
  const suggestedCFOP = onlyDigits(suggestion?.cfop || item?.cfop);
  const sourceICMSCST = onlyDigits(item?.icms_cst || item?.cst);
  const suggestedICMSCST = onlyDigits(suggestion?.icms_cst);
  const sourcePISCST = onlyDigits(item?.pis_cst);
  const sourceCOFINSCST = onlyDigits(item?.cofins_cst);
  const suggestedPISCST = onlyDigits(suggestion?.pis_cst);
  const suggestedCOFINSCST = onlyDigits(suggestion?.cofins_cst);
  const suggestedCEST = onlyDigits(suggestion?.cest || item?.cest);
  const sourceICMSInfo = getICMSCSTInfo(sourceICMSCST);
  const suggestedICMSInfo = getICMSCSTInfo(suggestedICMSCST);
  const sourceICMSOrigin = getICMSOriginInfo(item?.icms_cst || item?.cst);
  const suggestedICMSOrigin = getICMSOriginInfo(suggestion?.icms_cst);
  const sourceCSOSNInfo = getCSOSNInfo(item?.csosn);
  const suggestedCSOSNInfo = getCSOSNInfo(suggestion?.csosn);
  const sourcePISInfo = getPISCOFINSCSTInfo(sourcePISCST);
  const suggestedPISInfo = getPISCOFINSCSTInfo(suggestedPISCST);
  const sourceCOFINSInfo = getPISCOFINSCSTInfo(sourceCOFINSCST);
  const suggestedCOFINSInfo = getPISCOFINSCSTInfo(suggestedCOFINSCST);
  const regimeAdvice = getPISCOFINSRegimeAdvice(
    getOrganizationTaxRegime(),
    suggestedPISCST || sourcePISCST,
    suggestedCOFINSCST || sourceCOFINSCST
  );
  const icmsRegimeAdvice = getICMSRegimeAdvice(
    getOrganizationTaxRegime(),
    getOrganizationCRT(),
    suggestedICMSCST || sourceICMSCST,
    suggestion?.csosn || item?.csosn
  );
  const cbenefAdvice = getCBenefAdvice(
    item?.emitter_uf || item?.recipient_uf,
    suggestedICMSCST || sourceICMSCST,
    suggestion?.csosn || item?.csosn,
    suggestion?.cbenef || item?.cbenef
  );
  const hasSTEvidence =
    sourceICMSCST === "60" ||
    isSubstitutionTaxCFOP(sourceCFOP) ||
    suggestedCEST !== "";
  const cfopPreserved = sourceCFOP !== "" && suggestedCFOP === sourceCFOP;
  const wrongGenericCFOP = hasSTEvidence && isGenericSaleCFOP(suggestedCFOP);
  const icmsPreserved = sourceICMSCST !== "" && suggestedICMSCST === sourceICMSCST;
  const pisCofinsPreserved =
    sourcePISCST !== "" &&
    sourceCOFINSCST !== "" &&
    suggestedPISCST === sourcePISCST &&
    suggestedCOFINSCST === sourceCOFINSCST;

  const status = wrongGenericCFOP
    ? "danger"
    : hasSTEvidence && cfopPreserved && (icmsPreserved || sourceICMSCST === "60")
      ? "success"
      : "warning";

  const title = wrongGenericCFOP
    ? "Divergencia critica na substituicao tributaria"
    : status === "success"
      ? "Leitura fiscal coerente com mercadoria ST"
      : "Analise fiscal exige revisao";

  const summary = wrongGenericCFOP
    ? "A nota indica substituicao tributaria, mas a sugestao retornou CFOP de venda comum. Para esse cenario, preserve o CFOP ST da nota ou aplique regra estadual especifica."
    : status === "success"
      ? "A nota e a recomendacao apontam para mercadoria sujeita a substituicao tributaria, preservando a operacao fiscal observada."
      : "Existem sinais fiscais importantes, mas ainda faltam campos para publicar a regra com seguranca.";

  const findings = [
    hasSTEvidence
      ? `ST identificada por ${[
          sourceICMSCST === "60" ? "CST ICMS 60" : "",
          isSubstitutionTaxCFOP(sourceCFOP) ? `CFOP ${sourceCFOP}` : "",
          suggestedCEST ? `CEST ${suggestedCEST}` : "",
        ].filter(Boolean).join(", ")}.`
      : "Nao ha evidencia suficiente de substituicao tributaria neste item.",
    cfopPreserved
      ? `CFOP recomendado preserva a operacao observada: ${suggestedCFOP}.`
      : `CFOP observado ${displayValue(sourceCFOP)} e sugestao ${displayValue(suggestedCFOP)} precisam ser revisados.`,
    icmsPreserved
      ? `CST ICMS recomendado preserva a leitura da nota: ${suggestedICMSCST} - ${suggestedICMSInfo?.label || "classificacao identificada"}${suggestedICMSOrigin ? `; origem ${suggestedICMSOrigin}` : ""}.`
      : sourceICMSCST === "60"
        ? `${sourceICMSInfo?.label || "CST ICMS 60"}${sourceICMSOrigin ? `; origem ${sourceICMSOrigin}` : ""}: ${sourceICMSInfo?.note || "indica ICMS cobrado anteriormente por substituicao tributaria."}`
        : "CST ICMS ainda nao esta fechado para a recomendacao.",
    `${icmsRegimeAdvice.title}: ${icmsRegimeAdvice.detail}`,
    `${cbenefAdvice.title}: ${cbenefAdvice.detail}`,
    sourceCSOSNInfo || suggestedCSOSNInfo
      ? `CSOSN ${displayValue(suggestedCSOSNInfo?.code || sourceCSOSNInfo?.code)} - ${suggestedCSOSNInfo?.label || sourceCSOSNInfo?.label}.`
      : "CSOSN nao aplicavel ou nao informado para este item.",
    pisCofinsPreserved
      ? `PIS/COFINS preservados como CST ${suggestedPISCST}/${suggestedCOFINSCST} - ${suggestedPISInfo?.label || suggestedCOFINSInfo?.label || "classificacao identificada"}.`
      : `PIS/COFINS devem ser revisados conforme regime tributario da empresa. Nota: PIS ${displayValue(sourcePISInfo?.label || sourcePISCST)}, COFINS ${displayValue(sourceCOFINSInfo?.label || sourceCOFINSCST)}.`,
    `${regimeAdvice.title}: ${regimeAdvice.detail}`,
  ];

  const nextActions = wrongGenericCFOP
    ? [
        "Corrigir regra do motor para CFOP 5405 em venda interna de mercadoria ST.",
        "Manter ICMS CST 60 quando a nota e o CEST indicarem substituicao tributaria.",
        "Cadastrar regra por NCM/CEST/UF para evitar retorno ao CFOP 5102.",
      ]
    : [
        "Validar se o CEST e aplicavel ao produto no estado de destino.",
        "Completar base de ICMS ST com MVA, FCP ST e fundamento estadual quando existir.",
        "Publicar a regra apos conferir regime tributario e operacao real.",
      ];

  return `
    <div class="fiscal-analysis fiscal-analysis--${status}">
      <div class="fiscal-analysis__head">
        <div>
          <span class="eyebrow">Analise fiscal</span>
          <h4>${title}</h4>
          <p>${summary}</p>
        </div>
        <strong>${status === "success" ? "Coerente" : status === "danger" ? "Corrigir" : "Revisar"}</strong>
      </div>

      <div class="fiscal-analysis__grid">
        <div>
          <h5>Leitura tecnica</h5>
          <ul>
            ${findings.map((finding) => `<li>${finding}</li>`).join("")}
          </ul>
        </div>
        <div>
          <h5>Proximas acoes</h5>
          <ul>
            ${nextActions.map((action) => `<li>${action}</li>`).join("")}
          </ul>
        </div>
      </div>
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

      ${renderFiscalAnalysis(result, item)}

      ${renderFiscalTaxTable(suggestion, item)}

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
            source_icms_csosn: item.csosn || "",
            source_icms_rate: item.icms_rate || "",
            source_pis_cst: item.pis_cst || "",
            source_pis_rate: item.pis_rate || "",
            source_cofins_cst: item.cofins_cst || "",
            source_cofins_rate: item.cofins_rate || "",
            source_cfop: item.cfop || "",
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
