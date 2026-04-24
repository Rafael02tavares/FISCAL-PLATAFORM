import { getOrganizationId, getToken } from "../lib/auth";
import {
  createLegalRule,
  listLegalRules,
  listLegalSources,
  type LegalRule,
  type LegalSource,
} from "../lib/legalbasis";

const form = document.getElementById("cbenef-rule-form") as HTMLFormElement | null;
const feedback = document.getElementById("cbenef-feedback");
const sourceSelect = document.getElementById("cbenef-legal-source") as HTMLSelectElement | null;
const listBox = document.getElementById("cbenef-rules-list");
const statsBox = document.getElementById("cbenef-stats");
const helperBox = document.getElementById("cbenef-json-preview");

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

function buildValueContent(formData: FormData) {
  const payload: Record<string, string> = {};
  const suggestedCBenef = String(formData.get("suggested_cbenef") || "").trim();

  if (suggestedCBenef) {
    payload.cbenef = suggestedCBenef;
  }

  return payload;
}

function updateSourceOptions() {
  if (!sourceSelect) return;

  const filtered = cachedSources.filter((item) => item.tax_type === "cbenef" || item.tax_type === "icms");

  if (!filtered.length) {
    sourceSelect.innerHTML = `<option value="">Nenhuma fonte legal de cBenef ou ICMS cadastrada</option>`;
    return;
  }

  sourceSelect.innerHTML = `
    <option value="">Selecione uma fonte legal</option>
    ${filtered
      .map(
        (item) => `
          <option value="${item.id}">
            ${item.title}${item.reference_code ? ` · ${item.reference_code}` : ""}${item.uf ? ` · ${item.uf}` : ""} · ${String(item.tax_type || "").toUpperCase()}
          </option>
        `
      )
      .join("")}
  `;
}

function updateJSONPreview() {
  if (!helperBox || !form) return;
  helperBox.textContent = JSON.stringify(buildValueContent(new FormData(form)), null, 2);
}

function renderStats(items: LegalRule[]) {
  if (!statsBox) return;

  const withUF = items.filter((item) => item.emitter_uf || item.recipient_uf).length;
  const withNCM = items.filter((item) => item.ncm_code || item.cclas_trib).length;
  const withCFOP = items.filter((item) => item.cfop).length;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Total de regras</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--amber">
      <span>Com filtro de UF</span>
      <strong>${withUF}</strong>
    </article>
    <article class="stat-card stat-card--sky">
      <span>Com NCM / cClasTrib</span>
      <strong>${withNCM}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Com CFOP</span>
      <strong>${withCFOP}</strong>
    </article>
  `;
}

function renderRules(items: LegalRule[]) {
  if (!listBox) return;

  renderStats(items);

  if (!items.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma regra de cBenef cadastrada.</strong>
        <p>Cadastre regras para orientar o beneficio fiscal por operacao, UF, classificacao e base legal.</p>
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
            <article class="rule-card">
              <div class="rule-card__top">
                <div>
                  <span class="rule-card__tag">cBenef</span>
                  <h3>${item.operation_code || "Regra geral"}</h3>
                  <p>${item.tax_regime || "Todos os regimes"} · prioridade ${item.priority || 100}</p>
                </div>
                <span class="rule-card__badge">${item.tax_type || "cbenef"}</span>
              </div>

              <div class="rule-card__grid">
                <p><strong>NCM:</strong> ${item.ncm_code || "-"}</p>
                <p><strong>CFOP:</strong> ${item.cfop || "-"}</p>
                <p><strong>CEST:</strong> ${item.cest || "-"}</p>
                <p><strong>cClasTrib:</strong> ${item.cclas_trib || "-"}</p>
                <p><strong>ICMS CST:</strong> ${item.icms_cst || "-"}</p>
                <p><strong>CSOSN:</strong> ${item.csosn || "-"}</p>
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
    const appliesTax = item.tax_type === "cbenef" || item.tax_type === "icms";
    return appliesTax && item.value_type === "cst_rule" && !!item.value_content && item.value_content.includes("cbenef");
  });

  renderRules(cachedRules);
}

async function bootstrap() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  if (!getOrganizationId()) {
    setFeedback("Selecione uma organizacao antes de gerenciar regras de cBenef.", "error");
    return;
  }

  try {
    await Promise.all([loadSources(), loadRules()]);
    updateJSONPreview();
  } catch (error) {
    setFeedback(`Falha ao carregar configuracoes de cBenef: ${String(error)}`, "error");
    if (listBox) {
      listBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar regras de cBenef.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

form?.addEventListener("input", () => {
  updateJSONPreview();
});

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  if (!form) return;

  const formData = new FormData(form);
  const valueContent = buildValueContent(formData);

  if (!Object.keys(valueContent).length) {
    setFeedback("Informe o cBenef sugerido para salvar a regra.", "error");
    return;
  }

  const payload = {
    legal_source_id: String(formData.get("legal_source_id") || "").trim(),
    tax_type: "cbenef",
    operation_code: String(formData.get("operation_code") || "").trim(),
    tax_regime: String(formData.get("tax_regime") || "").trim(),
    ncm_code: String(formData.get("ncm_code") || "").trim(),
    cest: String(formData.get("cest") || "").trim(),
    cclas_trib: String(formData.get("cclas_trib") || "").trim(),
    cfop: String(formData.get("cfop") || "").trim(),
    pis_cst: "",
    cofins_cst: "",
    icms_cst: String(formData.get("icms_cst") || "").trim(),
    csosn: String(formData.get("csosn") || "").trim(),
    cbenef: String(formData.get("input_cbenef") || "").trim(),
    emitter_uf: String(formData.get("emitter_uf") || "").trim().toUpperCase(),
    recipient_uf: String(formData.get("recipient_uf") || "").trim().toUpperCase(),
    value_type: "cst_rule",
    value_content: JSON.stringify(valueContent),
    priority: Number(formData.get("priority") || 100),
    confidence_base: String(formData.get("confidence_base") || "0.78").trim(),
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

  setFeedback("Salvando regra de cBenef...", "muted");

  try {
    await createLegalRule(payload);
    setFeedback("Regra de cBenef salva com sucesso.", "success");
    form.reset();
    updateJSONPreview();
    await loadRules();
  } catch (error) {
    setFeedback(`Falha ao salvar regra de cBenef: ${String(error)}`, "error");
  } finally {
    if (submit) {
      submit.disabled = false;
      submit.textContent = "Salvar regra";
    }
  }
});

void bootstrap();
