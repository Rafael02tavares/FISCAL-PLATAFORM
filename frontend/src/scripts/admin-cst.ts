import { getOrganizationId, getToken } from "../lib/auth";
import {
  createLegalRule,
  listLegalRules,
  listLegalSources,
  type LegalRule,
  type LegalSource,
} from "../lib/legalbasis";

const form = document.getElementById("cst-rule-form") as HTMLFormElement | null;
const feedback = document.getElementById("cst-feedback");
const sourceSelect = document.getElementById("cst-legal-source") as HTMLSelectElement | null;
const listBox = document.getElementById("cst-rules-list");
const statsBox = document.getElementById("cst-stats");
const taxTypeField = document.getElementById("cst-tax-type") as HTMLSelectElement | null;
const helperBox = document.getElementById("cst-json-preview");

let cachedSources: LegalSource[] = [];
let cachedRules: LegalRule[] = [];

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

function normalizeTaxLabel(value: string) {
  switch (value) {
    case "icms":
      return "ICMS";
    case "pis":
      return "PIS";
    case "cofins":
      return "COFINS";
    default:
      return value || "Sem tributo";
  }
}

function getSuggestedPayload(taxType: string, formData: FormData) {
  const payload: Record<string, string> = {};
  const icmsCST = String(formData.get("suggested_icms_cst") || "").trim();
  const csosn = String(formData.get("suggested_csosn") || "").trim();
  const pisCST = String(formData.get("suggested_pis_cst") || "").trim();
  const cofinsCST = String(formData.get("suggested_cofins_cst") || "").trim();
  const cbenef = String(formData.get("suggested_cbenef") || "").trim();

  if (taxType === "icms") {
    if (icmsCST) payload.icms_cst = icmsCST;
    if (csosn) payload.csosn = csosn;
    if (cbenef) payload.cbenef = cbenef;
  }

  if (taxType === "pis" && pisCST) {
    payload.pis_cst = pisCST;
  }

  if (taxType === "cofins" && cofinsCST) {
    payload.cofins_cst = cofinsCST;
  }

  return payload;
}

function updateSourceOptions() {
  if (!sourceSelect || !taxTypeField) return;

  const taxType = taxTypeField.value;
  const filtered = cachedSources.filter((item) => item.tax_type === taxType);

  if (!filtered.length) {
    sourceSelect.innerHTML = `<option value="">Nenhuma fonte legal de ${normalizeTaxLabel(taxType)} cadastrada</option>`;
    return;
  }

  sourceSelect.innerHTML = `
    <option value="">Selecione uma fonte legal</option>
    ${filtered
      .map(
        (item) => `
          <option value="${item.id}">
            ${item.title}${item.reference_code ? ` · ${item.reference_code}` : ""}${item.uf ? ` · ${item.uf}` : ""}
          </option>
        `
      )
      .join("")}
  `;
}

function updateJSONPreview() {
  if (!helperBox || !form || !taxTypeField) return;

  const formData = new FormData(form);
  const payload = getSuggestedPayload(taxTypeField.value, formData);
  helperBox.textContent = JSON.stringify(payload, null, 2);
}

function renderStats(items: LegalRule[]) {
  if (!statsBox) return;

  const icms = items.filter((item) => item.tax_type === "icms").length;
  const pis = items.filter((item) => item.tax_type === "pis").length;
  const cofins = items.filter((item) => item.tax_type === "cofins").length;
  const withUF = items.filter((item) => item.emitter_uf || item.recipient_uf).length;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Total de regras</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--sky">
      <span>ICMS</span>
      <strong>${icms}</strong>
    </article>
    <article class="stat-card stat-card--rose">
      <span>PIS / COFINS</span>
      <strong>${pis + cofins}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Com filtro por UF</span>
      <strong>${withUF}</strong>
    </article>
  `;
}

function renderRules(items: LegalRule[]) {
  if (!listBox) return;

  renderStats(items);

  if (!items.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma regra CST cadastrada.</strong>
        <p>Cadastre regras de ICMS, PIS e COFINS para orientar a sugestao fiscal por contexto.</p>
      </div>
    `;
    return;
  }

  listBox.innerHTML = `
    <div class="rule-list">
      ${items
        .map((item) => {
          let parsed = "{}";
          try {
            parsed = JSON.stringify(JSON.parse(item.value_content || "{}"), null, 2);
          } catch {
            parsed = item.value_content || "{}";
          }

          return `
            <article class="rule-card rule-card--${item.tax_type || "default"}">
              <div class="rule-card__top">
                <div>
                  <span class="rule-card__tag">${normalizeTaxLabel(item.tax_type)}</span>
                  <h3>${item.operation_code || "Regra geral"}</h3>
                  <p>${item.tax_regime || "Todos os regimes"} · prioridade ${item.priority || 100}</p>
                </div>
                <span class="rule-card__badge">${item.value_type || "cst_rule"}</span>
              </div>

              <div class="rule-card__grid">
                <p><strong>NCM:</strong> ${item.ncm_code || "-"}</p>
                <p><strong>CFOP:</strong> ${item.cfop || "-"}</p>
                <p><strong>CEST:</strong> ${item.cest || "-"}</p>
                <p><strong>UFs:</strong> ${item.emitter_uf || "-"} → ${item.recipient_uf || "-"}</p>
                <p><strong>ICMS CST:</strong> ${item.icms_cst || "-"}</p>
                <p><strong>CSOSN:</strong> ${item.csosn || "-"}</p>
                <p><strong>PIS CST:</strong> ${item.pis_cst || "-"}</p>
                <p><strong>COFINS CST:</strong> ${item.cofins_cst || "-"}</p>
              </div>

              <pre class="rule-card__json">${parsed}</pre>
            </article>
          `;
        })
        .join("")}
    </div>
  `;
}

async function loadSources() {
  const response = await listLegalSources(300);
  cachedSources = Array.isArray(response?.items) ? response.items : [];
  updateSourceOptions();
}

async function loadRules() {
  const response = await listLegalRules(300);
  cachedRules = (Array.isArray(response?.items) ? response.items : []).filter(
    (item) => item.value_type === "cst_rule" && ["icms", "pis", "cofins"].includes(item.tax_type)
  );
  renderRules(cachedRules);
}

function syncTaxFormVisibility() {
  if (!form || !taxTypeField) return;

  const taxType = taxTypeField.value;
  form.querySelectorAll<HTMLElement>("[data-tax-only]").forEach((element) => {
    const allowed = String(element.dataset.taxOnly || "")
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);

    element.hidden = allowed.length > 0 && !allowed.includes(taxType);
  });

  updateSourceOptions();
  updateJSONPreview();
}

async function bootstrap() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  if (!getOrganizationId()) {
    setFeedback("Selecione uma organizacao antes de gerenciar regras CST.", "error");
    return;
  }

  try {
    await Promise.all([loadSources(), loadRules()]);
    updateJSONPreview();
  } catch (error) {
    setFeedback(`Falha ao carregar configuracoes CST: ${String(error)}`, "error");
    if (listBox) {
      listBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar regras CST.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

taxTypeField?.addEventListener("change", syncTaxFormVisibility);

form?.addEventListener("input", () => {
  updateJSONPreview();
});

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  if (!form || !taxTypeField) return;

  const formData = new FormData(form);
  const taxType = taxTypeField.value;
  const valueContentObject = getSuggestedPayload(taxType, formData);

  if (!Object.keys(valueContentObject).length) {
    setFeedback("Informe pelo menos um CST sugerido para salvar a regra.", "error");
    return;
  }

  const payload = {
    legal_source_id: String(formData.get("legal_source_id") || "").trim(),
    tax_type: taxType,
    operation_code: String(formData.get("operation_code") || "").trim(),
    tax_regime: String(formData.get("tax_regime") || "").trim(),
    ncm_code: String(formData.get("ncm_code") || "").trim(),
    cest: String(formData.get("cest") || "").trim(),
    cclas_trib: String(formData.get("cclas_trib") || "").trim(),
    cfop: String(formData.get("cfop") || "").trim(),
    pis_cst: String(formData.get("input_pis_cst") || "").trim(),
    cofins_cst: String(formData.get("input_cofins_cst") || "").trim(),
    icms_cst: String(formData.get("input_icms_cst") || "").trim(),
    csosn: String(formData.get("input_csosn") || "").trim(),
    cbenef: String(formData.get("input_cbenef") || "").trim(),
    emitter_uf: String(formData.get("emitter_uf") || "").trim().toUpperCase(),
    recipient_uf: String(formData.get("recipient_uf") || "").trim().toUpperCase(),
    value_type: "cst_rule",
    value_content: JSON.stringify(valueContentObject),
    priority: Number(formData.get("priority") || 100),
    confidence_base: String(formData.get("confidence_base") || "0.80").trim(),
    effective_from: String(formData.get("effective_from") || "").trim(),
    effective_to: String(formData.get("effective_to") || "").trim(),
  };

  if (!payload.legal_source_id) {
    setFeedback("Selecione uma fonte legal para vincular a regra.", "error");
    return;
  }

  const submit = form.querySelector<HTMLButtonElement>('button[type="submit"]');
  if (submit) {
    submit.disabled = true;
    submit.textContent = "Salvando...";
  }

  setFeedback("Salvando regra CST...", "muted");

  try {
    await createLegalRule(payload);
    setFeedback("Regra CST salva com sucesso.", "success");
    form.reset();
    if (taxTypeField) taxTypeField.value = taxType;
    syncTaxFormVisibility();
    await loadRules();
  } catch (error) {
    setFeedback(`Falha ao salvar regra CST: ${String(error)}`, "error");
  } finally {
    if (submit) {
      submit.disabled = false;
      submit.textContent = "Salvar regra CST";
    }
  }
});

void bootstrap();
