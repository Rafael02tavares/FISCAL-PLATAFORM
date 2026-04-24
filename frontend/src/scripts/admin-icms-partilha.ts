import { getOrganizationId, getToken } from "../lib/auth";
import {
  createDIFALRule,
  listICMSStateRates,
  listDIFALRules,
  resolveICMSRateReference,
  upsertICMSStateRate,
  type DIFALRule,
  type ICMSRateReference,
  type ICMSStateRate,
} from "../lib/admin-icms-partilha";

const form = document.getElementById("difal-rule-form") as HTMLFormElement | null;
const rateForm = document.getElementById("icms-rate-form") as HTMLFormElement | null;
const feedback = document.getElementById("difal-feedback");
const rateFeedback = document.getElementById("icms-rate-feedback");
const listBox = document.getElementById("difal-rules-list");
const statsBox = document.getElementById("difal-stats");
const ratesListBox = document.getElementById("icms-state-rates-list");
const referenceBox = document.getElementById("icms-rate-reference");

let cachedStateRates: ICMSStateRate[] = [];
let cachedRules: DIFALRule[] = [];

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

function setRateFeedback(message: string, tone: "muted" | "success" | "error" = "muted") {
  if (!rateFeedback) return;

  rateFeedback.textContent = message;
  rateFeedback.className =
    tone === "success"
      ? "feedback feedback--success"
      : tone === "error"
        ? "feedback feedback--error"
        : "dashboard-note";
}

function formatPercent(value: string) {
  if (!value) return "-";
  const normalized = value.replace(",", ".");
  const numeric = Number(normalized);
  if (Number.isNaN(numeric)) return value;
  return `${numeric.toFixed(2)}%`;
}

function formatMode(value: string) {
  switch (value) {
    case "yes":
      return "Sim";
    case "no":
      return "Nao";
    case "any":
    case "":
      return "Qualquer";
    default:
      return value;
  }
}

function renderStats(items: DIFALRule[]) {
  if (!statsBox) return;

  const active = items.filter((item) => item.status === "ACTIVE").length;
  const withFCP = items.filter((item) => item.fcp_rate && item.fcp_rate !== "0").length;
  const targeted = items.filter((item) => item.recipient_uf || item.issuer_uf).length;
  const stateCoverage = new Set(cachedStateRates.map((item) => item.uf)).size;

  statsBox.innerHTML = `
    <article class="stat-card">
      <span>Regras cadastradas</span>
      <strong>${items.length}</strong>
    </article>
    <article class="stat-card stat-card--teal">
      <span>Ativas</span>
      <strong>${active}</strong>
    </article>
    <article class="stat-card stat-card--gold">
      <span>Com FCP</span>
      <strong>${withFCP}</strong>
    </article>
    <article class="stat-card stat-card--blue">
      <span>Com UF alvo</span>
      <strong>${targeted}</strong>
    </article>
    <article class="stat-card">
      <span>UFs na base</span>
      <strong>${stateCoverage}</strong>
    </article>
  `;
}

function renderList(items: DIFALRule[]) {
  if (!listBox) return;

  renderStats(items);

  if (!items.length) {
    listBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma regra de partilha cadastrada.</strong>
        <p>Cadastre a primeira combinacao de DIFAL para controlar origem, destino e aliquotas da operacao.</p>
      </div>
    `;
    return;
  }

  listBox.innerHTML = `
    <div class="rule-list">
      ${items
        .map(
          (item) => `
            <article class="rule-card" data-filter-item>
              <div class="rule-card__top">
                <div>
                  <div class="rule-card__eyebrow">${item.code || "Sem codigo"}</div>
                  <h3>${item.name || "Regra sem nome"}</h3>
                  <p>${item.operation_scope || "INTERSTATE"} · ${item.operation_type || "EXIT"} · prioridade ${item.priority}</p>
                </div>
                <span class="rule-badge rule-badge--${String(item.status || "").toLowerCase()}">${item.status || "DRAFT"}</span>
              </div>

              <div class="rule-card__metrics">
                <div>
                  <span>Aliquota interna</span>
                  <strong>${formatPercent(item.internal_rate)}</strong>
                </div>
                <div>
                  <span>Aliquota interestadual</span>
                  <strong>${formatPercent(item.interstate_rate)}</strong>
                </div>
                <div>
                  <span>FCP</span>
                  <strong>${formatPercent(item.fcp_rate)}</strong>
                </div>
              </div>

              <div class="rule-card__grid">
                <p><strong>UFs:</strong> ${item.issuer_uf || "-"} → ${item.recipient_uf || item.uf || "-"}</p>
                <p><strong>Consumidor final:</strong> ${formatMode(item.final_consumer_mode)}</p>
                <p><strong>Destinatario contribuinte:</strong> ${formatMode(item.recipient_contributor)}</p>
                <p><strong>CRT:</strong> ${item.crt || "Todos"}</p>
                <p><strong>CFOP prefixo:</strong> ${item.cfop_prefix || "-"}</p>
                <p><strong>NCM prefixo:</strong> ${item.ncm_prefix || "-"}</p>
                <p><strong>Vigencia:</strong> ${item.valid_from || "-"} ate ${item.valid_to || "indeterminada"}</p>
                <p><strong>Especificidade:</strong> ${item.specificity_hint || 0}</p>
              </div>

              ${
                item.reason
                  ? `<div class="rule-card__reason"><strong>Justificativa:</strong> ${item.reason}</div>`
                  : ""
              }

              ${
                Array.isArray(item.legal_basis_ids) && item.legal_basis_ids.length
                  ? `<div class="rule-card__footer"><strong>Base legal:</strong> ${item.legal_basis_ids.join(", ")}</div>`
                  : ""
              }
            </article>
          `
        )
        .join("")}
    </div>
  `;
}

function renderStateRates(items: ICMSStateRate[]) {
  if (!ratesListBox) return;

  if (!items.length) {
    ratesListBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma base por UF cadastrada.</strong>
        <p>Cadastre as aliquotas internas para o motor reforcar o DIFAL automaticamente.</p>
      </div>
    `;
    return;
  }

  ratesListBox.innerHTML = `
    <div class="state-rate-list">
      ${items
        .map(
          (item) => `
            <article class="state-rate-card" data-filter-item>
              <div class="state-rate-card__top">
                <div>
                  <strong>${item.uf}</strong>
                  <p>Vigencia ${item.valid_from || "-"} ate ${item.valid_to || "indeterminada"}</p>
                </div>
                <button
                  type="button"
                  class="state-rate-card__apply"
                  data-apply-uf="${item.uf}"
                  data-apply-rate="${item.internal_rate}"
                  data-apply-fcp="${item.fcp_rate}"
                >
                  Usar no formulario
                </button>
              </div>

              <div class="state-rate-card__meta">
                <div>
                  <span>Aliquota interna</span>
                  <strong>${formatPercent(item.internal_rate)}</strong>
                </div>
                <div>
                  <span>FCP padrao</span>
                  <strong>${formatPercent(item.fcp_rate)}</strong>
                </div>
              </div>

              <div class="state-rate-card__footer">
                <span>${item.source_reference || "Sem fonte informada"}</span>
                ${item.source_url ? `<a class="secondary-link" href="${item.source_url}" target="_blank" rel="noreferrer">Abrir fonte</a>` : ""}
              </div>
            </article>
          `
        )
        .join("")}
    </div>
  `;
}

function renderReference(item: ICMSRateReference | null) {
  if (!referenceBox) return;

  if (!item) {
    referenceBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Referencia indisponivel.</strong>
        <p>Preencha origem e destino com UFs validas e confirme que a base estadual foi migrada.</p>
      </div>
    `;
    return;
  }

  referenceBox.innerHTML = `
    <div class="reference-card" data-filter-item>
      <div class="reference-card__header">
        <div>
          <strong>${item.issuer_uf} → ${item.recipient_uf}</strong>
          <p>${item.mode === "INTERNAL" ? "Operacao interna" : "Operacao interestadual"} · base resolvida automaticamente</p>
        </div>
        <button id="apply-reference-to-rule" type="button" class="primary-button">Aplicar na regra</button>
      </div>

      <div class="reference-card__grid">
        <div class="reference-pill">
          <span>Aliq. interna</span>
          <strong>${formatPercent(item.internal_rate)}</strong>
        </div>
        <div class="reference-pill">
          <span>Aliq. interestadual</span>
          <strong>${formatPercent(item.interstate_rate)}</strong>
        </div>
        <div class="reference-pill">
          <span>Diferenca</span>
          <strong>${formatPercent(item.difference_rate)}</strong>
        </div>
        <div class="reference-pill">
          <span>FCP</span>
          <strong>${formatPercent(item.fcp_rate)}</strong>
        </div>
      </div>

      <div class="rule-card__reason">
        <strong>Fundamento operacional:</strong> ${item.notes || "Referencia resolvida pela matriz estadual de ICMS."}
      </div>
    </div>
  `;

  const applyButton = document.getElementById("apply-reference-to-rule");
  applyButton?.addEventListener("click", () => applyReferenceToRule(item));
}

function applyReferenceToRule(item: ICMSRateReference) {
  if (!form) return;

  const internalRateField = form.querySelector<HTMLInputElement>('input[name="internal_rate"]');
  const interstateRateField = form.querySelector<HTMLInputElement>('input[name="interstate_rate"]');
  const fcpRateField = form.querySelector<HTMLInputElement>('input[name="fcp_rate"]');
  const ufField = form.querySelector<HTMLInputElement>('input[name="uf"]');
  const recipientUFField = form.querySelector<HTMLInputElement>('input[name="recipient_uf"]');

  if (internalRateField) internalRateField.value = item.internal_rate || "";
  if (interstateRateField) interstateRateField.value = item.interstate_rate || "";
  if (fcpRateField) fcpRateField.value = item.fcp_rate || "";
  if (ufField) ufField.value = item.recipient_uf || "";
  if (recipientUFField && !recipientUFField.value) recipientUFField.value = item.recipient_uf || "";

  setFeedback("Referencia automatica aplicada ao formulario de regra.", "success");
}

function wireStateRateActions() {
  document.querySelectorAll<HTMLButtonElement>("[data-apply-uf]").forEach((button) => {
    button.addEventListener("click", () => {
      if (!rateForm || !form) return;

      const uf = button.dataset.applyUf || "";
      const rate = button.dataset.applyRate || "";
      const fcp = button.dataset.applyFcp || "";

      const rateUFField = rateForm.querySelector<HTMLInputElement>('input[name="uf"]');
      const recipientUFField = form.querySelector<HTMLInputElement>('input[name="recipient_uf"]');
      const ufField = form.querySelector<HTMLInputElement>('input[name="uf"]');
      const internalRateField = form.querySelector<HTMLInputElement>('input[name="internal_rate"]');
      const fcpRateField = form.querySelector<HTMLInputElement>('input[name="fcp_rate"]');

      if (rateUFField) rateUFField.value = uf;
      if (recipientUFField) recipientUFField.value = uf;
      if (ufField) ufField.value = uf;
      if (internalRateField) internalRateField.value = rate;
      if (fcpRateField && !fcpRateField.value) fcpRateField.value = fcp;

      setFeedback(`Base de ${uf} aplicada ao formulario da regra detalhada.`, "success");
      void refreshReference();
    });
  });
}

async function loadRules() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  if (!getOrganizationId()) {
    setFeedback("Selecione uma organizacao antes de administrar a partilha de ICMS.", "error");
    return;
  }

  try {
    const response = await listDIFALRules();
    cachedRules = Array.isArray(response?.items) ? response.items : [];
    renderList(cachedRules);
  } catch (error) {
    if (!listBox) return;
    listBox.innerHTML = `
      <div class="dashboard-empty dashboard-empty--error">
        <strong>Falha ao carregar as regras de partilha.</strong>
        <p>${String(error)}</p>
      </div>
    `;
  }
}

async function loadStateRates() {
  try {
    const response = await listICMSStateRates();
    cachedStateRates = Array.isArray(response?.items) ? response.items : [];
    renderStateRates(cachedStateRates);
    if (cachedRules.length || statsBox) {
      renderStats(cachedRules);
    }
    wireStateRateActions();
  } catch (error) {
    if (!ratesListBox) return;
    ratesListBox.innerHTML = `
      <div class="dashboard-empty dashboard-empty--error">
        <strong>Falha ao carregar a base estadual.</strong>
        <p>${String(error)}</p>
      </div>
    `;
  }
}

async function refreshReference() {
  if (!form) return;

  const issuerUF = String(form.querySelector<HTMLInputElement>('input[name="issuer_uf"]')?.value || "").trim().toUpperCase();
  const recipientUF = String(form.querySelector<HTMLInputElement>('input[name="recipient_uf"]')?.value || "").trim().toUpperCase();

  if (!issuerUF || !recipientUF) {
    if (referenceBox) {
      referenceBox.innerHTML = `
        <div class="dashboard-note">
          Informe UF de origem e destino na regra abaixo para visualizar a referencia automatica de partilha.
        </div>
      `;
    }
    return;
  }

  try {
    const response = await resolveICMSRateReference(issuerUF, recipientUF);
    renderReference(response?.item || null);
  } catch (error) {
    if (!referenceBox) return;
    referenceBox.innerHTML = `
      <div class="dashboard-empty dashboard-empty--error">
        <strong>Falha ao resolver a referencia.</strong>
        <p>${String(error)}</p>
      </div>
    `;
  }
}

rateForm?.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(rateForm);
  const payload = {
    uf: String(formData.get("uf") || "").trim().toUpperCase(),
    internal_rate: String(formData.get("internal_rate") || "").trim(),
    fcp_rate: String(formData.get("fcp_rate") || "").trim(),
    valid_from: String(formData.get("valid_from") || "").trim(),
    valid_to: String(formData.get("valid_to") || "").trim(),
    source_reference: String(formData.get("source_reference") || "").trim(),
    source_url: String(formData.get("source_url") || "").trim(),
    notes: String(formData.get("notes") || "").trim(),
  };

  const submit = rateForm.querySelector<HTMLButtonElement>('button[type="submit"]');
  if (submit) {
    submit.disabled = true;
    submit.textContent = "Salvando...";
  }

  setRateFeedback("Salvando base estadual de ICMS...", "muted");

  try {
    const response = await upsertICMSStateRate(payload);
    setRateFeedback(response?.message || "Base estadual salva com sucesso.", "success");
    await loadStateRates();
    await loadRules();
    await refreshReference();
  } catch (error) {
    setRateFeedback(`Falha ao salvar a base estadual: ${String(error)}`, "error");
  } finally {
    if (submit) {
      submit.disabled = false;
      submit.textContent = "Salvar base por UF";
    }
  }
});

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(form);
  const legalBasisRaw = String(formData.get("legal_basis_ids") || "");
  const payload = {
    code: String(formData.get("code") || "").trim(),
    name: String(formData.get("name") || "").trim(),
    uf: String(formData.get("uf") || "").trim().toUpperCase(),
    priority: Number(formData.get("priority") || 100),
    status: String(formData.get("status") || "ACTIVE").trim().toUpperCase(),
    valid_from: String(formData.get("valid_from") || "").trim(),
    valid_to: String(formData.get("valid_to") || "").trim(),
    legal_basis_ids: legalBasisRaw
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean),
    issuer_uf: String(formData.get("issuer_uf") || "").trim().toUpperCase(),
    recipient_uf: String(formData.get("recipient_uf") || "").trim().toUpperCase(),
    operation_scope: String(formData.get("operation_scope") || "INTERSTATE").trim().toUpperCase(),
    operation_type: String(formData.get("operation_type") || "EXIT").trim().toUpperCase(),
    final_consumer_mode: String(formData.get("final_consumer_mode") || "any").trim().toLowerCase(),
    recipient_contributor: String(formData.get("recipient_contributor") || "any").trim().toLowerCase(),
    crt: String(formData.get("crt") || "").trim(),
    cfop_prefix: String(formData.get("cfop_prefix") || "").trim(),
    ncm_prefix: String(formData.get("ncm_prefix") || "").trim(),
    internal_rate: String(formData.get("internal_rate") || "").trim(),
    interstate_rate: String(formData.get("interstate_rate") || "").trim(),
    fcp_rate: String(formData.get("fcp_rate") || "").trim(),
    applies: formData.get("applies") === "on",
    reason: String(formData.get("reason") || "").trim(),
  };

  const submit = form.querySelector<HTMLButtonElement>('button[type="submit"]');
  if (submit) {
    submit.disabled = true;
    submit.textContent = "Salvando...";
  }

  setFeedback("Salvando regra de partilha de ICMS...", "muted");

  try {
    const response = await createDIFALRule(payload);
    setFeedback(response?.message || "Regra de partilha salva com sucesso.", "success");
    form.reset();

    const statusField = form.querySelector<HTMLSelectElement>('select[name="status"]');
    const operationScopeField = form.querySelector<HTMLSelectElement>('select[name="operation_scope"]');
    const operationTypeField = form.querySelector<HTMLSelectElement>('select[name="operation_type"]');
    const finalConsumerField = form.querySelector<HTMLSelectElement>('select[name="final_consumer_mode"]');
    const contributorField = form.querySelector<HTMLSelectElement>('select[name="recipient_contributor"]');
    const appliesField = form.querySelector<HTMLInputElement>('input[name="applies"]');

    if (statusField) statusField.value = "ACTIVE";
    if (operationScopeField) operationScopeField.value = "INTERSTATE";
    if (operationTypeField) operationTypeField.value = "EXIT";
    if (finalConsumerField) finalConsumerField.value = "yes";
    if (contributorField) contributorField.value = "any";
    if (appliesField) appliesField.checked = true;

    await loadRules();
  } catch (error) {
    setFeedback(`Falha ao salvar a regra: ${String(error)}`, "error");
  } finally {
    if (submit) {
      submit.disabled = false;
      submit.textContent = "Salvar regra";
    }
  }
});

form?.querySelectorAll<HTMLInputElement>('input[name="issuer_uf"], input[name="recipient_uf"]').forEach((field) => {
  field.addEventListener("change", () => {
    void refreshReference();
  });
  field.addEventListener("blur", () => {
    void refreshReference();
  });
});

void loadRules();
void loadStateRates();
void refreshReference();
