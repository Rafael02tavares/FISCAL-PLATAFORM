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

let items: CatalogProductItem[] = [];
let organization: OrganizationItem | null = null;
let lastQuery = "";
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
      <strong>${items.length} produtos disponiveis para consulta</strong>
      <span>
        Contexto atual: ${escapeHtml(getTaxRegimeLabel())}.
        Pesquise por descricao, GTIN, NCM, CEST, ${flowLabel} ou codigo interno para localizar o perfil fiscal.
      </span>
    `;
    return;
  }

  if (!list.length) {
    summaryBox.innerHTML = `
      <strong>Nenhum produto encontrado para "${escapeHtml(query)}"</strong>
      <span>
        Tente buscar por um termo mais curto, GTIN, NCM, CEST, ${flowLabel} ou descricao parcial do item.
      </span>
    `;
    return;
  }

  summaryBox.innerHTML = `
    <strong>${list.length} resultado(s) para "${escapeHtml(query)}"</strong>
    <span>
      Primeiro destaque: ${displayValue(highlighted?.description)} | NCM ${displayValue(profile.ncm)} | ${flowLabel} ${displayValue(flowValue)}.
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

function buildProductCard(item: CatalogProductItem) {
  const profile = item.profile || {};
  const flowLabel = isSimplesNacional() ? "CSOSN" : "CFOP";
  const flowValue = isSimplesNacional() ? profile.csosn : profile.cfop;

  return `
    <article class="product-card">
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
        <span>Confianca: ${hasValue(profile.confidence_score) ? escapeHtml(profile.confidence_score) : "0.99"}</span>
      </div>
    </article>
  `;
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
}

function applyFilter() {
  const query = normalizeText(filterInput?.value).toLowerCase();
  if (!query) {
    renderSearchSummary(items);
    renderProducts(items);
    return;
  }

  const filtered = items.filter((item) =>
    [
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
      .some((value) => String(value).toLowerCase().includes(query)),
  );

  renderSearchSummary(filtered);
  renderProducts(filtered);
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
    const response = await saveCatalogProduct(buildPayload(new FormData(form)));
    items = Array.isArray(response?.items) ? response.items : [];
    renderStats();
    renderCoverage();
    renderGovernance();
    if (filterInput) {
      filterInput.value = lastQuery;
    }
    applyFilter();
    form.reset();
    setMessage(response?.message || "Produto tributario salvo com sucesso.", "success");
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

void initialize();
