import { getOrganizationId, getToken } from "../lib/auth";
import {
  createLegalRule,
  listLegalRules,
  listLegalSources,
  type LegalRule,
  type LegalSource,
} from "../lib/legalbasis";

const form = document.getElementById("reform-rule-form") as HTMLFormElement | null;
const feedback = document.getElementById("reform-feedback");
const sourceSelect = document.getElementById("reform-legal-source") as HTMLSelectElement | null;
const taxTypeField = document.getElementById("reform-tax-type") as HTMLSelectElement | null;
const listBox = document.getElementById("reform-rules-list");
const statsBox = document.getElementById("reform-stats");
const helperBox = document.getElementById("reform-json-preview");

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
    case "ibs":
      return "IBS";
    case "cbs":
      return "CBS";
    case "cclas_trib":
      return "cClasTrib";
    default:
      return value || "Sem tributo";
  }
}

function resolveValueType(taxType: string) {
  if (taxType === "cclas_trib") {
    return "classification_rule";
  }

  return "rate_rule";
}

function buildValueContent(taxType: string, formData: FormData) {
  const payload: Record<string, string> = {};

  if (taxType === "cclas_trib") {
    const cclasTrib = String(formData.get("suggested_cclas_trib") || "").trim();
    if (cclasTrib) {
      payload.cclas_trib = cclasTrib;
    }
  }

  if (taxType === "ibs") {
    const ibsRate = String(formData.get("suggested_ibs_rate") || "").trim();
    if (ibsRate) {
      payload.ibs_rate = ibsRate;
    }
  }

  if (taxType === "cbs") {
    const cbsRate = String(formData.get("suggested_cbs_rate") || "").trim();
    if (cbsRate) {
      payload.cbs_rate = cbsRate;
    }
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
  helperBox.textContent = JSON.stringify(buildValueContent(taxTypeField.value, formData), null, 2);
}

function syncFieldVisibility() {
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

function renderStats(items: LegalRule[]) {
  if (!statsBox) return;

  const ibs = items.filter((item) => item.tax_type === "ibs").length;
  const cbs = items.filter((item) => item.tax_type === "cbs").length;
  const cclasTrib = items.filter((item) => item.tax_type === "cclas_trib").length;
  const withNCM = items.filter((item) => item.ncm_code || item.cclas_trib).length;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Total</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>IBS</span>
      <strong>${ibs}</strong>
    </article>
    <article class="stat-card stat-card--sky">
      <span>CBS</span>
      <strong>${cbs}</strong>
    </article>
    <article class="stat-card stat-card--amber">
      <span>cClasTrib</span>
      <strong>${cclasTrib}</strong>
    </article>
    <article class="stat-card stat-card--rose">
      <span>Com contexto fiscal</span>
      <strong>${withNCM}</strong>
    </article>
  `;
}

function renderRules(items: LegalRule[]) {
  if (!listBox) return;

  renderStats(items);

  if (!items.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma regra de IBS, CBS ou cClasTrib cadastrada.</strong>
        <p>Cadastre regras da reforma tributaria para alimentar a sugestao fiscal e a simulacao.</p>
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
                  <p>${item.tax_regime || "Todos os regimes"} · ${item.value_type || "-"}</p>
                </div>
                <span class="rule-card__badge">prioridade ${item.priority || 100}</span>
              </div>

              <div class="rule-card__grid">
                <p><strong>NCM:</strong> ${item.ncm_code || "-"}</p>
                <p><strong>CFOP:</strong> ${item.cfop || "-"}</p>
                <p><strong>CEST:</strong> ${item.cest || "-"}</p>
                <p><strong>cClasTrib filtro:</strong> ${item.cclas_trib || "-"}</p>
                <p><strong>UFs:</strong> ${item.emitter_uf || "-"} → ${item.recipient_uf || "-"}</p>
                <p><strong>Vigencia:</strong> ${item.effective_from || "-"} ate ${item.effective_to || "indeterminada"}</p>
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
  cachedRules = (Array.isArray(response?.items) ? response.items : []).filter((item) => {
    if (!["ibs", "cbs", "cclas_trib"].includes(item.tax_type)) {
      return false;
    }

    return item.value_type === "rate_rule" || item.value_type === "classification_rule";
  });

  renderRules(cachedRules);
}

async function bootstrap() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  if (!getOrganizationId()) {
    setFeedback("Selecione uma organizacao antes de gerenciar IBS, CBS e cClasTrib.", "error");
    return;
  }

  try {
    await Promise.all([loadSources(), loadRules()]);
    updateJSONPreview();
  } catch (error) {
    setFeedback(`Falha ao carregar configuracoes da reforma: ${String(error)}`, "error");
    if (listBox) {
      listBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar regras da reforma.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

taxTypeField?.addEventListener("change", syncFieldVisibility);

form?.addEventListener("input", () => {
  updateJSONPreview();
});

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  if (!form || !taxTypeField) return;

  const formData = new FormData(form);
  const taxType = taxTypeField.value;
  const valueContent = buildValueContent(taxType, formData);

  if (!Object.keys(valueContent).length) {
    setFeedback("Informe um valor sugerido para salvar a regra da reforma.", "error");
    return;
  }

  const payload = {
    legal_source_id: String(formData.get("legal_source_id") || "").trim(),
    tax_type: taxType,
    operation_code: String(formData.get("operation_code") || "").trim(),
    tax_regime: String(formData.get("tax_regime") || "").trim(),
    ncm_code: String(formData.get("ncm_code") || "").trim(),
    cest: String(formData.get("cest") || "").trim(),
    cclas_trib: String(formData.get("input_cclas_trib") || "").trim(),
    cfop: String(formData.get("cfop") || "").trim(),
    pis_cst: "",
    cofins_cst: "",
    icms_cst: "",
    csosn: "",
    cbenef: "",
    emitter_uf: String(formData.get("emitter_uf") || "").trim().toUpperCase(),
    recipient_uf: String(formData.get("recipient_uf") || "").trim().toUpperCase(),
    value_type: resolveValueType(taxType),
    value_content: JSON.stringify(valueContent),
    priority: Number(formData.get("priority") || 100),
    confidence_base: String(formData.get("confidence_base") || "0.75").trim(),
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

  setFeedback("Salvando regra da reforma tributaria...", "muted");

  try {
    await createLegalRule(payload);
    setFeedback("Regra da reforma salva com sucesso.", "success");
    form.reset();
    if (taxTypeField) taxTypeField.value = taxType;
    syncFieldVisibility();
    await loadRules();
  } catch (error) {
    setFeedback(`Falha ao salvar regra da reforma: ${String(error)}`, "error");
  } finally {
    if (submit) {
      submit.disabled = false;
      submit.textContent = "Salvar regra";
    }
  }
});

void bootstrap();
