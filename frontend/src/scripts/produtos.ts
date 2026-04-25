import { getOrganizationId, getToken } from "../lib/auth";
import { listCatalogProducts, saveCatalogProduct, type CatalogProductItem } from "../lib/catalog-products";
import { listOrganizations } from "../lib/organizations";

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
const adminPanel = document.querySelector(".catalog-admin-panel") as HTMLDetailsElement | null;
const quickFilterButtons = Array.from(document.querySelectorAll("[data-product-filter]")) as HTMLButtonElement[];
const newProductButton = document.getElementById("new-product-button") as HTMLButtonElement | null;

let items: CatalogProductItem[] = [];
let organization: OrganizationItem | null = null;
let lastQuery = "";
let activeProductFilter = "all";
let lastSavedProductId = "";
const PRODUCT_PREFILL_KEY = "catalog_product_prefill";

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
    case "missing_icms":
      return "pendentes de ICMS";
    case "missing_contributions":
      return "sem PIS/COFINS";
    case "missing_reform":
      return "sem IBS/CBS";
    default:
      return "todos";
  }
}

function isSimplesNacional() {
  return getTaxRegimeLabel().toLowerCase().includes("simples");
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
  return `
    <article class="metric-card ${tone ? `metric-card--${tone}` : ""}">
      <span>${label}</span>
      <strong>${value}</strong>
      <p>${items.length ? `${Math.round((value / items.length) * 100)}% da base` : "Aguardando cadastro"}</p>
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
    metricCard("Com NCM", countWhere((item) => hasValue(item.profile?.ncm)), "teal"),
    metricCard("Com PIS/COFINS", countWhere((item) => hasValue(item.profile?.pis_cst) || hasValue(item.profile?.cofins_cst)), "amber"),
    metricCard("Com ICMS/CSOSN", countWhere((item) => hasValue(item.profile?.icms_cst) || hasValue(item.profile?.csosn)), "sky"),
    metricCard("Com reforma", countWhere((item) => hasValue(item.profile?.ibs_rate) || hasValue(item.profile?.cbs_rate)), "rose"),
    metricCard("Com operacao", countWhere((item) => hasValue(item.profile?.operation_code) || hasValue(item.profile?.cfop)), "sky"),
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
    coverageItem("Classificacao", countWhere((item) => hasValue(item.profile?.ncm) && hasValue(item.profile?.cclas_trib))),
    coverageItem("Contribuicoes", countWhere((item) => hasValue(item.profile?.pis_cst) && hasValue(item.profile?.cofins_cst))),
    coverageItem("ICMS / regime", countWhere((item) => hasValue(item.profile?.icms_cst) || hasValue(item.profile?.csosn))),
    coverageItem("Reforma", countWhere((item) => hasValue(item.profile?.ibs_rate) || hasValue(item.profile?.cbs_rate) || hasValue(item.profile?.selective_tax_code))),
  ].join("");
}

function renderGovernance() {
  if (!governanceBox) return;

  const cards = [
    {
      title: "Classificacao fiscal",
      text: "NCM, EX, CEST e cClasTrib prontos para consulta e sugestao.",
    },
    {
      title: "Regime e incidencias",
      text: isSimplesNacional()
        ? "A base privilegia CSOSN, PIS, COFINS e cenarios do Simples."
        : "A base privilegia CST, CFOP e memoria de aliquotas por operacao.",
    },
    {
      title: "Reforma tributaria",
      text: "IBS, CBS e imposto seletivo ficam cadastrados no mesmo ativo fiscal.",
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
      <strong>${list.length} de ${items.length} produtos disponiveis para consulta</strong>
      <span>
        Filtro atual: ${escapeHtml(getProductFilterLabel())}. Contexto: ${escapeHtml(getTaxRegimeLabel())}.
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
      Filtro atual: ${escapeHtml(getProductFilterLabel())}. Primeiro destaque: ${displayValue(highlighted?.description)} | NCM ${displayValue(profile.ncm)} | ${flowLabel} ${displayValue(flowValue)}.
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

type ProductDiagnostic = {
  area: string;
  status: "ready" | "attention" | "missing";
  title: string;
  action: string;
};

function buildProductDiagnostics(item: CatalogProductItem): ProductDiagnostic[] {
  const profile = item.profile || {};
  const diagnostics: ProductDiagnostic[] = [];

  if (hasValue(profile.ncm) && hasValue(profile.cest)) {
    diagnostics.push({
      area: "Classificacao",
      status: "ready",
      title: "NCM e CEST preenchidos",
      action: "Base pronta para regras por produto e ICMS ST.",
    });
  } else if (hasValue(profile.ncm)) {
    diagnostics.push({
      area: "Classificacao",
      status: "attention",
      title: "CEST pendente",
      action: "Revise CEST quando houver possibilidade de ICMS ST.",
    });
  } else {
    diagnostics.push({
      area: "Classificacao",
      status: "missing",
      title: "NCM ausente",
      action: "Informe NCM para destravar a consulta tributaria.",
    });
  }

  if (hasValue(profile.cfop) || hasValue(profile.csosn) || hasValue(profile.operation_code)) {
    diagnostics.push({
      area: "Operacao",
      status: "ready",
      title: "Fluxo fiscal informado",
      action: "Operacao pronta para sugestao e simulacao.",
    });
  } else {
    diagnostics.push({
      area: "Operacao",
      status: "missing",
      title: "CFOP/operacao ausente",
      action: "Vincule CFOP ou operacao padrao do produto.",
    });
  }

  if (isSimplesNacional()) {
    diagnostics.push(
      hasValue(profile.csosn)
        ? {
            area: "ICMS",
            status: "ready",
            title: "CSOSN definido",
            action: "Revise ST, FCP e beneficios quando aplicavel.",
          }
        : {
            area: "ICMS",
            status: "missing",
            title: "CSOSN pendente",
            action: "Preencha CSOSN para o regime Simples.",
          },
    );
  } else if (hasValue(profile.icms_cst) && hasValue(profile.icms_rate)) {
    diagnostics.push({
      area: "ICMS",
      status: "ready",
      title: "CST e aliquota definidos",
      action: "Revise reducao, FCP, ST e excecoes estaduais.",
    });
  } else if (hasValue(profile.icms_cst) || hasValue(profile.icms_rate) || hasValue(profile.fcp_rate)) {
    diagnostics.push({
      area: "ICMS",
      status: "attention",
      title: "ICMS parcial",
      action: "Complete CST, aliquota e regras estaduais.",
    });
  } else {
    diagnostics.push({
      area: "ICMS",
      status: "missing",
      title: "ICMS ausente",
      action: "Cadastre CST/aliquota de ICMS ou CSOSN.",
    });
  }

  const hasContributionCST = hasValue(profile.pis_cst) && hasValue(profile.cofins_cst);
  const hasContributionRates = hasValue(profile.pis_rate) && hasValue(profile.cofins_rate);
  if (hasContributionCST && hasContributionRates) {
    diagnostics.push({
      area: "PIS/COFINS",
      status: "ready",
      title: "Contribuicoes completas",
      action: "Valide receitas e excecoes monofasicas.",
    });
  } else if (hasContributionCST || hasContributionRates) {
    diagnostics.push({
      area: "PIS/COFINS",
      status: "attention",
      title: "Contribuicoes parciais",
      action: "Complete CST e aliquotas por regime.",
    });
  } else {
    diagnostics.push({
      area: "PIS/COFINS",
      status: "missing",
      title: "Contribuicoes ausentes",
      action: "Cadastre PIS/COFINS para consulta completa.",
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
          <span>Cobertura fiscal</span>
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
    case "missing_icms":
      return isSimplesNacional()
        ? !hasValue(profile.csosn)
        : !(hasValue(profile.icms_cst) && hasValue(profile.icms_rate));
    case "missing_contributions":
      return !(hasValue(profile.pis_cst) && hasValue(profile.cofins_cst) && hasValue(profile.pis_rate) && hasValue(profile.cofins_rate));
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

function buildProductCard(item: CatalogProductItem) {
  const profile = item.profile || {};
  const flowLabel = isSimplesNacional() ? "CSOSN" : "CFOP";
  const flowValue = isSimplesNacional() ? profile.csosn : profile.cfop;
  const editPayload = escapeHtml(JSON.stringify(buildProductEditPayload(item)));
  const highlightClass = item.id === lastSavedProductId ? " product-card--saved" : "";

  return `
    <article class="product-card${highlightClass}" data-product-id="${escapeHtml(item.id)}">
      <div class="product-card__top">
        <div>
          <span class="product-chip">${displayValue(item.product_code, "Sem codigo")}</span>
          <h4>${displayValue(item.description)}</h4>
          <p class="product-card__meta">
            GTIN ${displayValue(item.gtin, "nao informado")} | Regime ${escapeHtml(getTaxRegimeLabel())}
          </p>
        </div>
        <span class="product-chip product-chip--source">${displayValue(profile.source_type, "manual_entry")}</span>
      </div>

      <div class="catalog-field-grid">
        ${taxPill("NCM", profile.ncm, "teal")}
        ${taxPill("NCM EX", profile.ncm_ex, "teal")}
        ${taxPill("CEST", profile.cest, "amber")}
        ${taxPill("cClasTrib", profile.cclas_trib, "sky")}
        ${taxPill(flowLabel, flowValue, "sky")}
        ${taxPill("ICMS CST", profile.icms_cst, "sky")}
        ${taxPill("PIS CST", profile.pis_cst, "rose")}
        ${taxPill("COFINS CST", profile.cofins_cst, "rose")}
        ${taxPill("Aliq. ICMS", profile.icms_rate, "sky")}
        ${taxPill("Aliq. PIS", profile.pis_rate, "rose")}
        ${taxPill("Aliq. COFINS", profile.cofins_rate, "rose")}
        ${taxPill("FCP", profile.fcp_rate, "amber")}
        ${taxPill("IBS", profile.ibs_rate, "teal")}
        ${taxPill("CBS", profile.cbs_rate, "teal")}
        ${taxPill("Imp. seletivo", profile.selective_tax_code, "amber")}
        ${taxPill("Aliq. seletivo", profile.selective_tax_rate, "amber")}
      </div>

      <div class="product-card__footer">
        <span>Operacao: ${displayValue(profile.operation_code)}</span>
        <span>UF: ${displayValue(profile.emitter_uf, "--")} -> ${displayValue(profile.recipient_uf, "--")}</span>
        <span>Regime alvo: ${displayValue(profile.target_tax_regime || getTaxRegimeLabel())}</span>
        <span>CRT: ${displayValue(profile.target_crt)}</span>
        <span>Confianca: ${hasValue(profile.confidence_score) ? escapeHtml(profile.confidence_score) : "0.99"}</span>
        <button class="product-edit-button" type="button" data-product-edit="${editPayload}">Completar cadastro</button>
      </div>

      ${renderProductDiagnostics(item)}
    </article>
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
    ncm_ex: profile.ncm_ex || "",
    cest: profile.cest || "",
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

  result.innerHTML = `<div class="catalog-products-list">${list.map(buildProductCard).join("")}</div>`;
  wireProductEditButtons();
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
    await refreshProducts(query);
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
  if (!normalized.target_tax_regime && organization?.tax_regime) {
    normalized.target_tax_regime = normalizeText(organization.tax_regime);
  }
  if (!normalized.target_crt && organization?.crt) {
    normalized.target_crt = normalizeText(organization.crt);
  }
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

async function refreshProducts(query = "") {
  const response = await listCatalogProducts(query);
  items = Array.isArray(response?.items) ? response.items : [];
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
    items = Array.isArray(response?.items) ? response.items : [];
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
    renderStats();
    renderCoverage();
    renderGovernance();
    if (filterInput) {
      filterInput.value = normalizeText(payload.description) || lastQuery;
    }
    activeProductFilter = "all";
    quickFilterButtons.forEach((button) => {
      button.classList.toggle("quick-chip--active", button.dataset.productFilter === "all");
    });
    applyFilter();
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
