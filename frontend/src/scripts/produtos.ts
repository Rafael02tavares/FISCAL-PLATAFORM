import { getOrganizationId, getToken } from "../lib/auth";
import { listCatalogProducts, saveCatalogProduct, type CatalogProductItem } from "../lib/catalog-products";
import { listOrganizations } from "../lib/organizations";
import { suggestTax } from "../lib/tax";

type OrganizationItem = {
  id: string;
  name?: string;
  tax_regime?: string;
  crt?: string;
  home_uf?: string;
};

const form = document.getElementById("catalog-product-form") as HTMLFormElement | null;
const messageBox = document.getElementById("catalog-form-message");
const result = document.getElementById("catalog-products-result");
const searchForm = document.getElementById("catalog-search-form") as HTMLFormElement | null;
const filterInput = document.getElementById("catalog-product-filter") as HTMLInputElement | null;
const statsBox = document.getElementById("catalog-stats");
const coverageBox = document.getElementById("catalog-coverage");
const governanceBox = document.getElementById("catalog-governance");
const summaryBox = document.getElementById("catalog-search-summary");
const paginationBox = document.getElementById("catalog-pagination");
const adminPanel = document.querySelector(".catalog-admin-panel") as HTMLDetailsElement | null;
const quickFilterButtons = Array.from(document.querySelectorAll("[data-product-filter]")) as HTMLButtonElement[];
const newProductButton = document.getElementById("new-product-button") as HTMLButtonElement | null;

let items: CatalogProductItem[] = [];
let organization: OrganizationItem | null = null;
let lastQuery = "";
let activeProductFilter = "all";
let lastSavedProductId = "";
let suggestionByProductId: Record<string, any> = {};
let suggestionLoadingProductId = "";
let suggestionErrorByProductId: Record<string, string> = {};
let suggestionExpandedByProductId: Record<string, boolean> = {};
const PRODUCT_PREFILL_KEY = "catalog_product_prefill";
const PAGE_SIZE = 24;
let currentPage = 1;
let catalogHasMore = false;

function normalizeText(value: unknown) {
  return String(value || "").trim();
}

function escapeHtml(value: unknown) {
  return normalizeText(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function hasValue(value: unknown) {
  return normalizeText(value) !== "";
}

function displayValue(value: unknown, fallback = "Nao informado") {
  return hasValue(value) ? escapeHtml(value) : fallback;
}

function getTaxRegimeLabel() {
  const taxRegime = normalizeText(organization?.tax_regime).toLowerCase();
  if (taxRegime.includes("simples")) return "Simples Nacional";
  if (taxRegime.includes("presum")) return "Lucro Presumido";
  if (taxRegime.includes("real")) return "Lucro Real";
  return normalizeText(organization?.tax_regime) || "Regime nao informado";
}

function getProductFilterLabel() {
  switch (activeProductFilter) {
    case "missing_ncm":
      return "sem NCM";
    case "missing_cest":
      return "sem CEST";
    case "missing_gtin":
      return "sem GTIN";
    case "missing_cst":
      return "sem CST";
    case "missing_reform":
      return "sem IBS/CBS";
    default:
      return "todos";
  }
}

function isSimplesNacional() {
  return getTaxRegimeLabel().toLowerCase().includes("simples");
}

function getOrganizationUF() {
  return normalizeText(organization?.home_uf).toUpperCase();
}

function getOrganizationCRT() {
  return normalizeText(organization?.crt);
}

function setMessage(text: string, tone: "muted" | "success" | "error" = "muted") {
  if (!messageBox) return;
  messageBox.textContent = text;
  messageBox.className =
    tone === "error"
      ? "message-box message-box--error"
      : tone === "success"
        ? "message-box message-box--success"
        : "dashboard-note";
}

function metricCard(label: string, value: number, tone = "") {
  const percentage = items.length ? Math.round((value / items.length) * 100) : 0;
  return `
    <article class="metric-card ${tone ? `metric-card--${tone}` : ""}">
      <div>
        <span>${label}</span>
        <strong>${value}</strong>
      </div>
      <div class="metric-card__bar">
        <i style="width:${percentage}%"></i>
      </div>
      <p>${items.length ? `${percentage}% da base` : "Aguardando cadastro"}</p>
    </article>
  `;
}

function countWhere(fn: (item: CatalogProductItem) => boolean) {
  return items.filter(fn).length;
}

function renderStats() {
  if (!statsBox) return;

  statsBox.innerHTML = [
    metricCard("Produtos", items.length),
    metricCard("Com GTIN", countWhere((item) => hasValue(item.gtin)), "sky"),
    metricCard("Com NCM", countWhere((item) => hasValue(item.profile?.ncm)), "teal"),
    metricCard("Com CEST", countWhere((item) => hasValue(item.profile?.cest)), "amber"),
    metricCard("Com CST", countWhere((item) => hasValue(item.profile?.icms_cst) || hasValue(item.profile?.csosn) || hasValue(item.profile?.pis_cst) || hasValue(item.profile?.cofins_cst)), "rose"),
  ].join("");
}

function coverageItem(label: string, count: number) {
  const percentage = items.length ? Math.round((count / items.length) * 100) : 0;
  return `
    <div class="coverage-card">
      <div class="product-card__top">
        <strong>${label}</strong>
        <span>${count}/${items.length || 0}</span>
      </div>
      <div class="coverage-card__bar">
        <div class="coverage-card__fill" style="width:${percentage}%"></div>
      </div>
      <p>${percentage}% de cobertura da base tributaria.</p>
    </div>
  `;
}

function renderCoverage() {
  if (!coverageBox) return;

  coverageBox.innerHTML = [
    coverageItem("Codigo de barras", countWhere((item) => hasValue(item.gtin))),
    coverageItem("Classificacao NCM", countWhere((item) => hasValue(item.profile?.ncm))),
    coverageItem("CEST informado", countWhere((item) => hasValue(item.profile?.cest))),
    coverageItem("CST base", countWhere((item) => hasValue(item.profile?.icms_cst) || hasValue(item.profile?.csosn) || hasValue(item.profile?.pis_cst) || hasValue(item.profile?.cofins_cst))),
  ].join("");
}

function renderGovernance() {
  if (!governanceBox) return;

  const cards = [
    {
      title: "Item como base",
      text: "Descricao, GTIN, NCM, CEST e CST formam a identidade fiscal fixa.",
    },
    {
      title: "Sugestao por contexto",
      text: isSimplesNacional()
        ? "A organizacao usa Simples Nacional; o motor prioriza CSOSN quando sugerir a saida."
        : "A organizacao usa regime normal; o motor prioriza CST e aliquotas por UF.",
    },
    {
      title: "Importacao como volume",
      text: "O XML cadastra itens rapidamente; a tela de produto limpa e consolida essa base.",
    },
  ];

  governanceBox.innerHTML = cards
    .map(
      (card) => `
        <article class="governance-card">
          <h4>${card.title}</h4>
          <p>${card.text}</p>
        </article>
      `,
    )
    .join("");
}

function renderSearchSummary(list: CatalogProductItem[]) {
  if (!summaryBox) return;

  const query = normalizeText(filterInput?.value);
  const highlighted = list[0];
  const profile = highlighted?.profile || {};
  const flowLabel = isSimplesNacional() ? "CSOSN" : "CFOP";
  const flowValue = isSimplesNacional() ? profile.csosn : profile.cfop;

  if (!items.length) {
    summaryBox.innerHTML = `
      <strong>Base fiscal vazia</strong>
      <span>Nenhum produto foi cadastrado ainda. Use o painel administrativo para iniciar a memoria tributaria.</span>
    `;
    return;
  }

  if (!query) {
    summaryBox.innerHTML = `
      <strong>${list.length} produto(s) nesta pagina fiscal</strong>
      <span>
        Pagina ${currentPage}. Filtro atual: ${escapeHtml(getProductFilterLabel())}. Contexto: ${escapeHtml(getTaxRegimeLabel())}.
        Pesquise por descricao, GTIN, NCM, CEST, ${flowLabel} ou codigo interno para localizar o perfil fiscal.
      </span>
    `;
    return;
  }

  if (!list.length) {
    summaryBox.innerHTML = `
      <strong>Nenhum produto encontrado para "${escapeHtml(query)}"</strong>
      <span>
        Filtro atual: ${escapeHtml(getProductFilterLabel())}. Tente buscar por um termo mais curto, GTIN, NCM, CEST, ${flowLabel} ou descricao parcial do item.
      </span>
    `;
    return;
  }

  summaryBox.innerHTML = `
    <strong>${list.length} resultado(s) para "${escapeHtml(query)}"</strong>
    <span>
      Pagina ${currentPage}. Filtro atual: ${escapeHtml(getProductFilterLabel())}. Primeiro destaque: ${displayValue(highlighted?.description)} | NCM ${displayValue(profile.ncm)} | ${flowLabel} ${displayValue(flowValue)}.
    </span>
  `;
}

function taxPill(label: string, value: unknown, tone = "teal") {
  return `
    <div class="tax-pill tax-pill--${tone}">
      <span>${label}</span>
      <strong>${displayValue(value)}</strong>
    </div>
  `;
}

function fieldItem(label: string, value: unknown, tone = "teal") {
  return `
    <span class="field-item field-item--${tone}">
      <small>${label}</small>
      <strong>${displayValue(value, "-")}</strong>
    </span>
  `;
}

function taxSection(title: string, subtitle: string, tone: string, content: string) {
  return `
    <section class="tax-section tax-section--${tone}">
      <div class="tax-section__title">
        <span>${title}</span>
        <p>${subtitle}</p>
      </div>
      <div class="tax-section__grid">
        ${content}
      </div>
    </section>
  `;
}

function fiscalInfoCard(title: string, codeLabel: string, code: unknown, description: unknown, tone = "teal") {
  return `
    <article class="fiscal-info-card fiscal-info-card--${tone}">
      <span>${title}</span>
      <strong>${codeLabel}: ${displayValue(code, "-")}</strong>
      <p>${displayValue(description, "Descricao ainda nao encontrada no catalogo importado.")}</p>
    </article>
  `;
}

function getProductCompletion(item: CatalogProductItem) {
  const profile = item.profile || {};
  const checks = [
    hasValue(item.description),
    hasValue(item.gtin),
    hasValue(profile.ncm),
    hasValue(profile.cest),
    hasValue(profile.icms_cst) || hasValue(profile.csosn) || hasValue(profile.pis_cst) || hasValue(profile.cofins_cst),
  ];
  const ready = checks.filter(Boolean).length;
  return Math.round((ready / checks.length) * 100);
}

function getProductHealth(item: CatalogProductItem) {
  const completion = getProductCompletion(item);

  if (completion < 60) {
    return {
      label: "Critico",
      tone: "rose",
      text: "faltam dados fixos para localizar o item com seguranca",
    };
  }

  if (completion < 100) {
    return {
      label: "Em revisao",
      tone: "amber",
      text: "identidade fiscal parcialmente preenchida",
    };
  }

  return {
    label: "Pronto",
    tone: "teal",
    text: "item pronto para sugestao por UF e regime",
  };
}

function buildSuggestionPayload(item: CatalogProductItem) {
  const profile = item.profile || {};
  const uf = getOrganizationUF();
  return {
    gtin: normalizeText(item.gtin),
    description: normalizeText(item.description),
    ncm_code: normalizeText(profile.ncm),
    operation_code: "sale_consumer_final",
    tax_regime: normalizeText(organization?.tax_regime) || normalizeText(profile.target_tax_regime),
    target_crt: getOrganizationCRT() || normalizeText(profile.target_crt),
    emitter_uf: uf || normalizeText(profile.emitter_uf).toUpperCase(),
    recipient_uf: uf || normalizeText(profile.recipient_uf).toUpperCase(),
  };
}

function suggestionCell(label: string, value: unknown) {
  return `
    <span class="suggestion-cell">
      <small>${label}</small>
      <strong>${displayValue(value, "-")}</strong>
    </span>
  `;
}

function suggestionTaxRow(title: string, tone: string, cells: string[]) {
  return `
    <tr class="suggestion-tax-row suggestion-tax-row--${tone}">
      <th scope="row">${escapeHtml(title)}</th>
      <td>${cells.join("")}</td>
    </tr>
  `;
}

function renderProductSuggestion(item: CatalogProductItem) {
  const loading = suggestionLoadingProductId === item.id;
  const error = suggestionErrorByProductId[item.id];
  const payload = suggestionByProductId[item.id];
  const suggestion = payload?.suggestion || {};
  const warnings = Array.isArray(payload?.warnings) ? payload.warnings : [];
  const ai = payload?.ai_assistance;

  if (loading) {
    return `
      <section class="product-suggestion product-suggestion--loading">
        <strong>Consultando sugestao tributaria...</strong>
        <p>Motor aplicando venda interna para consumidor final em ${escapeHtml(getOrganizationUF() || "UF da organizacao")} no regime ${escapeHtml(getTaxRegimeLabel())}.</p>
      </section>
    `;
  }

  if (error) {
    return `
      <section class="product-suggestion product-suggestion--error">
        <strong>Nao foi possivel gerar sugestao</strong>
        <p>${escapeHtml(error)}</p>
      </section>
    `;
  }

  if (!payload) {
    return `
      <section class="product-suggestion product-suggestion--empty">
        <strong>Sugestao ainda nao consultada</strong>
        <p>Clique em "Sugerir tributos" para calcular a saida conforme UF e regime da organizacao.</p>
      </section>
    `;
  }

  return `
    <section class="product-suggestion">
      <div class="product-suggestion__header">
        <div>
          <span>Sugestao tributaria</span>
          <strong>${displayValue(payload?.decision_summary?.title, "Resultado da consulta")}</strong>
          <p>${displayValue(payload?.decision_summary?.message, "Cenario padrao aplicado pela plataforma.")}</p>
        </div>
        <span class="product-chip product-chip--source">Conf. ${displayValue(payload?.confidence_score, "0")}</span>
      </div>
      <table class="suggestion-tax-table">
        <tbody>
        ${suggestionTaxRow(
          "Identidade fiscal",
          "teal",
          [
            suggestionCell("NCM", suggestion.ncm),
            suggestionCell("CEST", suggestion.cest),
            suggestionCell("cClasTrib", suggestion.cclas_trib),
          ],
        )}
        ${suggestionTaxRow(
          "Operacao e ICMS",
          "sky",
          [
            suggestionCell("CFOP", suggestion.cfop),
            suggestionCell("ICMS CST", suggestion.icms_cst),
            suggestionCell("CSOSN", suggestion.csosn),
            suggestionCell("Aliq. ICMS", suggestion.icms_rate),
          ],
        )}
        ${suggestionTaxRow(
          "PIS e COFINS",
          "rose",
          [
            suggestionCell("PIS CST", suggestion.pis_cst),
            suggestionCell("COFINS CST", suggestion.cofins_cst),
            suggestionCell("Aliq. PIS", suggestion.pis_rate),
            suggestionCell("Aliq. COFINS", suggestion.cofins_rate),
          ],
        )}
        ${suggestionTaxRow(
          "Reforma",
          "amber",
          [
            suggestionCell("IBS", suggestion.ibs_rate),
            suggestionCell("CBS", suggestion.cbs_rate),
          ],
        )}
        </tbody>
      </table>
      ${
        ai
          ? `<div class="product-ai-assist">
              <div>
                <span>OpenAI assistiva</span>
                <strong>${displayValue(ai.category, "Classificacao de apoio")}</strong>
                <p>${displayValue(ai.observation || ai.recommended_action, "IA usada apenas como apoio de triagem.")}</p>
              </div>
              <div class="product-ai-assist__meta">
                <span>Modelo <strong>${displayValue(ai.model, "-")}</strong></span>
                <span>Risco <strong>${displayValue(ai.risk, "-")}</strong></span>
                <span>Conf. IA <strong>${displayValue(ai.confidence, "-")}</strong></span>
              </div>
              ${
                Array.isArray(ai.signals) && ai.signals.length
                  ? `<p class="product-ai-assist__signals"><strong>Sinais:</strong> ${ai.signals.slice(0, 5).map((signal: string) => escapeHtml(signal)).join(", ")}</p>`
                  : ""
              }
            </div>`
          : ""
      }
      ${
        warnings.length
          ? `<div class="product-suggestion__warnings"><strong>Pontos de revisao</strong>${warnings.slice(0, 4).map((item: string) => `<p>${escapeHtml(item)}</p>`).join("")}</div>`
          : ""
      }
    </section>
  `;
}

type ProductDiagnostic = {
  area: string;
  status: "ready" | "attention" | "missing";
  title: string;
  action: string;
};

function buildProductDiagnostics(item: CatalogProductItem): ProductDiagnostic[] {
  const profile = item.profile || {};
  const diagnostics: ProductDiagnostic[] = [];

  diagnostics.push(
    hasValue(item.gtin)
      ? {
          area: "GTIN",
          status: "ready",
          title: "Codigo de barras definido",
          action: "Produto pode ser encontrado e consolidado em novas importacoes.",
        }
      : {
          area: "GTIN",
          status: "missing",
          title: "GTIN ausente",
          action: "Informe o codigo de barras para reduzir duplicidades.",
        },
  );

  if (hasValue(profile.ncm) && hasValue(profile.cest)) {
    diagnostics.push({
      area: "Classificacao",
      status: "ready",
      title: "NCM e CEST definidos",
      action: "Identidade fiscal suficiente para consultar regras por UF.",
    });
  } else if (hasValue(profile.ncm)) {
    diagnostics.push({
      area: "Classificacao",
      status: "attention",
      title: "CEST pendente",
      action: "CEST ajuda o motor, mas nao confirma ST sozinho.",
    });
  } else {
    diagnostics.push({
      area: "Classificacao",
      status: "missing",
      title: "NCM ausente",
      action: "Informe NCM para destravar a consulta tributaria.",
    });
  }

  if (hasValue(profile.icms_cst) || hasValue(profile.csosn)) {
    diagnostics.push({
      area: "ICMS",
      status: "ready",
      title: "Situacao ICMS base registrada",
      action: "A saida final sera decidida pela UF e regime da organizacao.",
    });
  } else if (hasValue(profile.cest)) {
    diagnostics.push({
      area: "ICMS",
      status: "attention",
      title: "ICMS depende do motor",
      action: "CEST existe, mas a regra de ST deve vir por UF/regime.",
    });
  } else {
    diagnostics.push({
      area: "ICMS",
      status: "missing",
      title: "Situacao ICMS base ausente",
      action: "Cadastre CST/CSOSN base ou deixe o motor sugerir pela regra.",
    });
  }

  const hasContributionCST = hasValue(profile.pis_cst) && hasValue(profile.cofins_cst);
  if (hasContributionCST) {
    diagnostics.push({
      area: "PIS/COFINS",
      status: "ready",
      title: "CST de contribuicoes registrado",
      action: "Aliquotas variam conforme regime e regra do produto.",
    });
  } else {
    diagnostics.push({
      area: "PIS/COFINS",
      status: "missing",
      title: "CST PIS/COFINS ausente",
      action: "Cadastre CST base para diferenciar normal, zero ou monofasico.",
    });
  }

  if (hasValue(profile.cclas_trib) && (hasValue(profile.ibs_rate) || hasValue(profile.cbs_rate))) {
    diagnostics.push({
      area: "Reforma",
      status: "ready",
      title: "IBS/CBS preparados",
      action: "Revise vigencia e regras transitorias.",
    });
  } else if (hasValue(profile.cclas_trib) || hasValue(profile.ibs_rate) || hasValue(profile.cbs_rate)) {
    diagnostics.push({
      area: "Reforma",
      status: "attention",
      title: "Reforma parcial",
      action: "Complete cClasTrib, IBS e CBS.",
    });
  } else {
    diagnostics.push({
      area: "Reforma",
      status: "missing",
      title: "IBS/CBS ausentes",
      action: "Alimente os dados da reforma tributaria.",
    });
  }

  return diagnostics;
}

function diagnosticLabel(status: ProductDiagnostic["status"]) {
  if (status === "ready") return "Pronto";
  if (status === "missing") return "Pendente";
  return "Atencao";
}

function renderProductDiagnostics(item: CatalogProductItem) {
  const diagnostics = buildProductDiagnostics(item);
  const ready = diagnostics.filter((entry) => entry.status === "ready").length;
  const attention = diagnostics.filter((entry) => entry.status === "attention").length;
  const missing = diagnostics.filter((entry) => entry.status === "missing").length;
  const score = Math.round((ready / diagnostics.length) * 100);

  return `
    <section class="product-diagnostic">
      <div class="product-diagnostic__header">
        <div>
          <span>Identidade do item</span>
          <strong>${score}% completo</strong>
        </div>
        <div class="product-diagnostic__counts">
          <span>${ready} prontos</span>
          <span>${attention} atencao</span>
          <span>${missing} pendentes</span>
        </div>
      </div>
      <div class="product-diagnostic__bar">
        <div style="width:${score}%"></div>
      </div>
      <div class="product-diagnostic__grid">
        ${diagnostics
          .map(
            (entry) => `
              <article class="product-diagnostic-card product-diagnostic-card--${entry.status}">
                <div>
                  <span>${escapeHtml(entry.area)}</span>
                  <strong>${diagnosticLabel(entry.status)}</strong>
                </div>
                <h5>${escapeHtml(entry.title)}</h5>
                <p>${escapeHtml(entry.action)}</p>
              </article>
            `,
          )
          .join("")}
      </div>
    </section>
  `;
}

function productMatchesQuickFilter(item: CatalogProductItem) {
  const profile = item.profile || {};
  switch (activeProductFilter) {
    case "missing_ncm":
      return !hasValue(profile.ncm);
    case "missing_cest":
      return hasValue(profile.ncm) && !hasValue(profile.cest);
    case "missing_gtin":
      return !hasValue(item.gtin);
    case "missing_cst":
      return !(hasValue(profile.icms_cst) || hasValue(profile.csosn) || hasValue(profile.pis_cst) || hasValue(profile.cofins_cst));
    case "missing_reform":
      return !(hasValue(profile.cclas_trib) && (hasValue(profile.ibs_rate) || hasValue(profile.cbs_rate)));
    default:
      return true;
  }
}

function productMatchesTextFilter(item: CatalogProductItem, query: string) {
  if (!query) return true;

  return [
    item.product_code,
    item.gtin,
    item.description,
    item.profile?.ncm,
    item.profile?.ncm_ex,
    item.profile?.cest,
    item.profile?.cfop,
    item.profile?.csosn,
    item.profile?.cclas_trib,
  ]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(query));
}

function statusBadge(item: CatalogProductItem) {
  const completion = getProductCompletion(item);
  const health = getProductHealth(item);
  return `<span class="product-status-badge product-status-badge--${health.tone}">${escapeHtml(health.label)} - ${completion}%</span>`;
}

function buildSuggestionDetailsRow(item: CatalogProductItem) {
  if (!suggestionExpandedByProductId[item.id]) return "";

  const hasSuggestionState = suggestionLoadingProductId === item.id || suggestionErrorByProductId[item.id] || suggestionByProductId[item.id];
  if (!hasSuggestionState) return "";

  return `
    <tr class="product-table__details" data-product-details="${escapeHtml(item.id)}">
      <td colspan="10">
        ${renderProductSuggestion(item)}
      </td>
    </tr>
  `;
}

function getSuggestionButtonLabel(item: CatalogProductItem) {
  if (suggestionLoadingProductId === item.id) return "Consultando...";
  if (suggestionExpandedByProductId[item.id]) return "Recolher";
  if (suggestionByProductId[item.id] || suggestionErrorByProductId[item.id]) return "Mostrar sugestao";
  return "Sugerir";
}

function buildProductRow(item: CatalogProductItem, index: number) {
  const profile = item.profile || {};
  const editPayload = escapeHtml(JSON.stringify(buildProductEditPayload(item)));
  const highlightClass = item.id === lastSavedProductId ? " product-table__row--saved" : "";
  const stripeClass = index % 2 === 0 ? " product-table__row--even" : " product-table__row--odd";
  const icmsSituationLabel = isSimplesNacional() ? "CSOSN base" : "CST ICMS base";
  const icmsSituationValue = isSimplesNacional() ? profile.csosn || profile.icms_cst : profile.icms_cst || profile.csosn;
  const pisCofins = `${displayValue(profile.pis_cst, "-")} / ${displayValue(profile.cofins_cst, "-")}`;

  return `
    <tr class="product-table__row${stripeClass}${highlightClass}" data-product-id="${escapeHtml(item.id)}">
      <td class="product-table__index">${(currentPage - 1) * PAGE_SIZE + index + 1}</td>
      <td class="product-table__product">
        <strong>${displayValue(item.description)}</strong>
        <span>Codigo: ${displayValue(item.product_code, "-")}</span>
      </td>
      <td>${displayValue(item.gtin, "-")}</td>
      <td>
        <strong class="product-table__ncm-code" title="${displayValue(profile.ncm_description, "Descricao NCM nao carregada")}">${displayValue(profile.ncm, "-")}</strong>
        <span>Passe o mouse</span>
      </td>
      <td>
        <strong>${displayValue(profile.cest, "-")}</strong>
        <span>${displayValue(profile.cest_description, "Sem descricao CEST")}</span>
      </td>
      <td>
        <strong>${displayValue(icmsSituationValue, "-")}</strong>
        <span>${escapeHtml(icmsSituationLabel)}</span>
      </td>
      <td>
        <strong>${pisCofins}</strong>
        <span>PIS / COFINS</span>
      </td>
      <td>
        <strong>${displayValue(profile.cclas_trib, "-")}</strong>
        <span>cClasTrib</span>
      </td>
      <td>${statusBadge(item)}</td>
      <td class="product-table__actions">
          <button class="product-suggest-button" type="button" data-product-suggest="${escapeHtml(item.id)}">${getSuggestionButtonLabel(item)}</button>
          <button class="product-edit-button" type="button" data-product-edit="${editPayload}">Editar</button>
      </td>
    </tr>
    ${buildSuggestionDetailsRow(item)}
  `;
}

function buildProductEditPayload(item: CatalogProductItem) {
  const profile = item.profile || {};
  return {
    product_id: item.id,
    product_code: item.product_code || "",
    gtin: item.gtin || "",
    description: item.description || "",
    ncm: profile.ncm || "",
    ncm_description: profile.ncm_description || "",
    ncm_ex: profile.ncm_ex || "",
    cest: profile.cest || "",
    cest_description: profile.cest_description || "",
    cclas_trib: profile.cclas_trib || "",
    cfop: profile.cfop || "",
    csosn: profile.csosn || "",
    icms_cst: profile.icms_cst || "",
    pis_cst: profile.pis_cst || "",
    cofins_cst: profile.cofins_cst || "",
    pis_revenue_code: profile.pis_revenue_code || "",
    cofins_revenue_code: profile.cofins_revenue_code || "",
    cbenef: profile.cbenef || "",
    icms_value: profile.icms_value || "",
    ipi_value: profile.ipi_value || "",
    pis_value: profile.pis_value || "",
    cofins_value: profile.cofins_value || "",
    pis_rate: profile.pis_rate || "",
    cofins_rate: profile.cofins_rate || "",
    icms_rate: profile.icms_rate || "",
    icms_base_reduction: profile.icms_base_reduction || "",
    fcp_rate: profile.fcp_rate || "",
    icms_st_rate: profile.icms_st_rate || "",
    ibs_rate: profile.ibs_rate || "",
    cbs_rate: profile.cbs_rate || "",
    selective_tax_code: profile.selective_tax_code || "",
    selective_tax_rate: profile.selective_tax_rate || "",
    operation_code: profile.operation_code || "sale_consumer_final",
    emitter_uf: profile.emitter_uf || "",
    recipient_uf: profile.recipient_uf || "",
    operation_nature: profile.operation_nature || "",
    target_tax_regime: profile.target_tax_regime || "",
    observed_tax_regime: profile.observed_tax_regime || "",
    target_crt: profile.target_crt || "",
    observed_crt: profile.observed_crt || "",
  };
}

function renderProducts(list: CatalogProductItem[]) {
  if (!result) return;

  if (!list.length) {
    const query = normalizeText(filterInput?.value);
    result.innerHTML = `
      <div class="dashboard-empty">
        <strong>${query ? "Nenhum produto localizado nessa busca." : "Nenhum produto tributario cadastrado."}</strong>
        <p>${
          query
            ? "Revise a descricao, GTIN, NCM, CEST, CFOP ou codigo informado para tentar novamente."
            : "Cadastre o primeiro item para alimentar o motor de sugestao fiscal e a tela de simulacao."
        }</p>
      </div>
    `;
    return;
  }

  result.innerHTML = `
    <div class="product-table-wrap">
      <table class="product-table">
        <colgroup>
          <col class="product-table__col-index" />
          <col class="product-table__col-product" />
          <col class="product-table__col-gtin" />
          <col class="product-table__col-ncm" />
          <col class="product-table__col-cest" />
          <col class="product-table__col-icms" />
          <col class="product-table__col-piscofins" />
          <col class="product-table__col-reform" />
          <col class="product-table__col-status" />
          <col class="product-table__col-actions" />
        </colgroup>
        <thead>
          <tr>
            <th>#</th>
            <th>Produto</th>
            <th>GTIN</th>
            <th>NCM</th>
            <th>CEST</th>
            <th>ICMS</th>
            <th>PIS/COFINS</th>
            <th>IBS/CBS</th>
            <th>Status</th>
            <th>Acao</th>
          </tr>
        </thead>
        <tbody>
          ${list.map((item, index) => buildProductRow(item, index)).join("")}
        </tbody>
      </table>
    </div>
  `;
  wireProductEditButtons();
  wireProductSuggestButtons();
}

function renderPagination() {
  if (!paginationBox) return;

  const from = items.length ? (currentPage - 1) * PAGE_SIZE + 1 : 0;
  const to = (currentPage - 1) * PAGE_SIZE + items.length;
  paginationBox.innerHTML = `
    <div class="catalog-pagination__info">
      <strong>Pagina ${currentPage}</strong>
      <span>Mostrando ${from}-${to}. Use a busca para ir direto ao produto quando a base estiver grande.</span>
    </div>
    <div class="catalog-pagination__actions">
      <button class="secondary-button" type="button" data-catalog-page="previous" ${currentPage <= 1 ? "disabled" : ""}>Anterior</button>
      <button class="primary-button" type="button" data-catalog-page="next" ${!catalogHasMore ? "disabled" : ""}>Proxima</button>
    </div>
  `;

  const previous = paginationBox.querySelector('[data-catalog-page="previous"]') as HTMLButtonElement | null;
  const next = paginationBox.querySelector('[data-catalog-page="next"]') as HTMLButtonElement | null;
  previous?.addEventListener("click", () => {
    if (currentPage <= 1) return;
    void refreshProducts(lastQuery, currentPage - 1);
  });
  next?.addEventListener("click", () => {
    if (!catalogHasMore) return;
    void refreshProducts(lastQuery, currentPage + 1);
  });
}

function wireProductEditButtons() {
  const buttons = Array.from(document.querySelectorAll("[data-product-edit]")) as HTMLButtonElement[];
  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      const raw = button.dataset.productEdit || "{}";
      try {
        const payload = JSON.parse(raw) as Record<string, string>;
        fillProductForm(payload);
      } catch {
        setMessage("Nao foi possivel carregar os dados do produto para edicao.", "error");
      }
    });
  });
}

function wireProductSuggestButtons() {
  const buttons = Array.from(document.querySelectorAll("[data-product-suggest]")) as HTMLButtonElement[];
  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      const productId = button.dataset.productSuggest || "";
      const item = items.find((entry) => entry.id === productId);
      if (!item) {
        setMessage("Produto nao encontrado para gerar sugestao.", "error");
        return;
      }
      void handleToggleProductSuggestion(item);
    });
  });
}

async function handleToggleProductSuggestion(item: CatalogProductItem) {
  const hasSuggestionState = Boolean(suggestionByProductId[item.id] || suggestionErrorByProductId[item.id]);
  if (suggestionExpandedByProductId[item.id] && hasSuggestionState) {
    suggestionExpandedByProductId[item.id] = false;
    applyFilter();
    return;
  }

  suggestionExpandedByProductId[item.id] = true;
  if (hasSuggestionState || suggestionLoadingProductId === item.id) {
    applyFilter();
    return;
  }

  await handleSuggestProduct(item);
}

async function handleSuggestProduct(item: CatalogProductItem) {
  suggestionLoadingProductId = item.id;
  suggestionExpandedByProductId[item.id] = true;
  delete suggestionErrorByProductId[item.id];
  applyFilter();

  try {
    const payload = buildSuggestionPayload(item);
    const response = await suggestTax(payload);
    suggestionByProductId[item.id] = response;
    setMessage(`Sugestao gerada para ${normalizeText(item.description) || "produto selecionado"}.`, "success");
  } catch (error) {
    suggestionErrorByProductId[item.id] = String(error);
    setMessage(`Falha ao gerar sugestao: ${String(error)}`, "error");
  } finally {
    suggestionLoadingProductId = "";
    applyFilter();
  }
}

function fillProductForm(payload: Record<string, string>) {
  if (!form) return;

  Object.entries(payload).forEach(([key, value]) => {
    const field = form.elements.namedItem(key);
    if (field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement || field instanceof HTMLSelectElement) {
      field.value = String(value || "");
    }
  });

  if (adminPanel) {
    adminPanel.open = true;
  }

  form.scrollIntoView({ behavior: "smooth", block: "start" });
  setMessage("Produto carregado para complementar o cadastro fiscal. Ao salvar, o perfil manual sera priorizado.", "success");
}

function startNewProduct() {
  if (!form) return;

  form.reset();
  lastSavedProductId = "";
  if (adminPanel) {
    adminPanel.open = true;
  }

  form.scrollIntoView({ behavior: "smooth", block: "start" });
  setMessage("Formulario limpo para cadastrar um novo produto fiscal.", "muted");
}

function applyFilter() {
  const query = normalizeText(filterInput?.value).toLowerCase();
  const filtered = items.filter((item) => productMatchesQuickFilter(item) && productMatchesTextFilter(item, query));

  renderSearchSummary(filtered);
  renderProducts(filtered);
  renderPagination();
}

function setActiveQuickFilter(filter: string) {
  activeProductFilter = filter || "all";
  quickFilterButtons.forEach((button) => {
    const isActive = button.dataset.productFilter === activeProductFilter;
    button.classList.toggle("quick-chip--active", isActive);
  });
  applyFilter();
}

async function handleSearchSubmit(event: SubmitEvent) {
  event.preventDefault();

  const query = normalizeText(filterInput?.value);
  lastQuery = query;

  try {
    if (result) {
      result.innerHTML = `<div class="dashboard-note">Consultando catalogo fiscal...</div>`;
    }
    renderSearchSummary([]);
    await refreshProducts(query, 1);
    setMessage(
      query
        ? `Consulta atualizada para "${query}".`
        : "Consulta atualizada com toda a memoria fiscal da organizacao.",
      "muted",
    );
  } catch (error) {
    if (result) {
      result.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao consultar o catalogo fiscal.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
    if (summaryBox) {
      summaryBox.innerHTML = `
        <strong>Falha na consulta</strong>
        <span>Nao foi possivel carregar os produtos para essa busca.</span>
      `;
    }
    setMessage("Nao foi possivel consultar o catalogo fiscal.", "error");
  }
}

async function loadOrganizationContext() {
  try {
    const response = (await listOrganizations()) as { organizations?: OrganizationItem[] };
    const orgId = getOrganizationId();
    const organizations = Array.isArray(response?.organizations) ? response.organizations : [];
    organization = organizations.find((item) => item.id === orgId) || organizations[0] || null;
  } catch {
    organization = null;
  }
}

function buildPayload(formData: FormData) {
  const payload = Object.fromEntries(formData.entries()) as Record<string, FormDataEntryValue>;
  const normalized: Record<string, string> = {};

  Object.entries(payload).forEach(([key, value]) => {
    normalized[key] = String(value || "").trim();
  });

  normalized.emitter_uf = normalized.emitter_uf.toUpperCase();
  normalized.recipient_uf = normalized.recipient_uf.toUpperCase();
  return normalized;
}

function readProductPrefill() {
  try {
    const raw = localStorage.getItem(PRODUCT_PREFILL_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Record<string, string>;
  } catch {
    return null;
  }
}

function clearProductPrefill() {
  try {
    localStorage.removeItem(PRODUCT_PREFILL_KEY);
  } catch {
    // ignora indisponibilidade de storage local
  }
}

function applyProductPrefill() {
  if (!form) return;

  const prefill = readProductPrefill();
  if (!prefill || typeof prefill !== "object") {
    return;
  }

  Object.entries(prefill).forEach(([key, value]) => {
    const field = form.elements.namedItem(key);
    if (field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement || field instanceof HTMLSelectElement) {
      field.value = String(value || "");
    }
  });

  if (adminPanel) {
    adminPanel.open = true;
  }

  form.scrollIntoView({ behavior: "smooth", block: "start" });
  setMessage("Cadastro preenchido automaticamente com dados trazidos da nota fiscal.", "success");
  clearProductPrefill();
}

async function refreshProducts(query = "", page = currentPage) {
  currentPage = Math.max(1, page);
  lastQuery = query;
  const response = await listCatalogProducts(query, {
    limit: PAGE_SIZE,
    offset: (currentPage - 1) * PAGE_SIZE,
  });
  items = Array.isArray(response?.items) ? response.items : [];
  catalogHasMore = Boolean(response?.has_more);
  renderStats();
  renderCoverage();
  renderGovernance();
  applyFilter();
}

async function handleSubmit(event: SubmitEvent) {
  event.preventDefault();

  if (!form) return;

  const description = normalizeText((form.elements.namedItem("description") as HTMLInputElement | null)?.value);
  if (!description) {
    setMessage("Informe a descricao do produto antes de salvar.", "error");
    return;
  }

  try {
    setMessage("Salvando produto tributario...", "success");
    const payload = buildPayload(new FormData(form));
    const response = await saveCatalogProduct(payload);
    lastSavedProductId = String(response?.product_id || payload.product_id || "");
    if (!lastSavedProductId) {
      const savedDescription = normalizeText(payload.description).toLowerCase();
      const savedGTIN = normalizeText(payload.gtin);
      const savedProduct = items.find((item) => {
        const sameGTIN = savedGTIN && normalizeText(item.gtin) === savedGTIN;
        const sameDescription = savedDescription && normalizeText(item.description).toLowerCase() === savedDescription;
        return Boolean(sameGTIN || sameDescription);
      });
      lastSavedProductId = savedProduct?.id || "";
    }
    if (filterInput) {
      filterInput.value = normalizeText(payload.description) || lastQuery;
    }
    lastQuery = normalizeText(payload.description) || lastQuery;
    activeProductFilter = "all";
    quickFilterButtons.forEach((button) => {
      button.classList.toggle("quick-chip--active", button.dataset.productFilter === "all");
    });
    await refreshProducts(lastQuery, 1);
    form.reset();
    setMessage(response?.message || "Produto tributario salvo com sucesso.", "success");
    if (lastSavedProductId) {
      setTimeout(() => {
        const savedCard = document.querySelector(`[data-product-id="${CSS.escape(lastSavedProductId)}"]`);
        savedCard?.scrollIntoView({ behavior: "smooth", block: "center" });
      }, 80);
    }
  } catch (error) {
    setMessage(`Falha ao salvar produto: ${String(error)}`, "error");
  }
}

async function initialize() {
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

  await loadOrganizationContext();
  renderGovernance();
  applyProductPrefill();

  try {
    await refreshProducts();
    renderSearchSummary(items);
    setMessage("Consulta pronta. Use a busca unica para localizar rapidamente o perfil tributario do produto.", "muted");
  } catch (error) {
    if (result) {
      result.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar o catalogo fiscal.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
    if (summaryBox) {
      summaryBox.innerHTML = `
        <strong>Falha ao preparar a consulta</strong>
        <span>O catalogo fiscal nao pode ser carregado no momento.</span>
      `;
    }
    setMessage("Nao foi possivel carregar o catalogo fiscal.", "error");
  }
}

filterInput?.addEventListener("input", applyFilter);
filterInput?.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    void handleSearchSubmit(event as unknown as SubmitEvent);
  }
});
searchForm?.addEventListener("submit", (event) => {
  void handleSearchSubmit(event as SubmitEvent);
});
form?.addEventListener("submit", (event) => {
  void handleSubmit(event as SubmitEvent);
});
quickFilterButtons.forEach((button) => {
  button.addEventListener("click", () => {
    setActiveQuickFilter(button.dataset.productFilter || "all");
  });
});
newProductButton?.addEventListener("click", startNewProduct);

void initialize();
