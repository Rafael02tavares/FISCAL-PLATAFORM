import {
  getCosmosIntegration,
  getOpenAIIntegration,
  saveCosmosIntegration,
  saveOpenAIIntegration,
  searchCosmosProducts,
  testCosmosIntegration,
  testOpenAIIntegration,
} from "../lib/integrations";

const statusBadge = document.getElementById("cosmos-status");
const enabledInput = document.getElementById("cosmos-enabled") as HTMLInputElement | null;
const baseURLInput = document.getElementById("cosmos-base-url") as HTMLInputElement | null;
const tokenInput = document.getElementById("cosmos-token") as HTMLInputElement | null;
const notesInput = document.getElementById("cosmos-notes") as HTMLTextAreaElement | null;
const saveForm = document.getElementById("cosmos-form") as HTMLFormElement | null;
const saveFeedback = document.getElementById("cosmos-feedback");
const testForm = document.getElementById("cosmos-test-form") as HTMLFormElement | null;
const testGTINInput = document.getElementById("cosmos-test-gtin") as HTMLInputElement | null;
const testFeedback = document.getElementById("cosmos-test-feedback");
const resultBox = document.getElementById("cosmos-test-result");
const cosmosSearchInput = document.getElementById("cosmos-search-query") as HTMLInputElement | null;
const cosmosSearchButton = document.getElementById("cosmos-search-button") as HTMLButtonElement | null;
const cosmosSearchFeedback = document.getElementById("cosmos-search-feedback");
const cosmosSearchResultBox = document.getElementById("cosmos-search-result");
const openAIStatusBadge = document.getElementById("openai-status");
const openAIEnabledInput = document.getElementById("openai-enabled") as HTMLInputElement | null;
const openAIBaseURLInput = document.getElementById("openai-base-url") as HTMLInputElement | null;
const openAIModelInput = document.getElementById("openai-model") as HTMLInputElement | null;
const openAITokenInput = document.getElementById("openai-token") as HTMLInputElement | null;
const openAINotesInput = document.getElementById("openai-notes") as HTMLTextAreaElement | null;
const openAIForm = document.getElementById("openai-form") as HTMLFormElement | null;
const openAIFeedback = document.getElementById("openai-feedback");
const openAITestButton = document.getElementById("openai-test-button") as HTMLButtonElement | null;
const openAITestFeedback = document.getElementById("openai-test-feedback");
const openAIResultBox = document.getElementById("openai-test-result");
const openAITestDescription = document.getElementById("openai-test-description") as HTMLInputElement | null;
const openAITestGTIN = document.getElementById("openai-test-gtin") as HTMLInputElement | null;
const openAITestNCM = document.getElementById("openai-test-ncm") as HTMLInputElement | null;
const openAITestCEST = document.getElementById("openai-test-cest") as HTMLInputElement | null;
const openAITestUF = document.getElementById("openai-test-uf") as HTMLInputElement | null;
const openAITestRegime = document.getElementById("openai-test-regime") as HTMLSelectElement | null;
const openAITestOperation = document.getElementById("openai-test-operation") as HTMLInputElement | null;

function setFeedback(element: HTMLElement | null, message: string, tone: "muted" | "success" | "error" = "muted") {
  if (!element) return;
  element.textContent = message;
  element.className =
    tone === "success"
      ? "integration-feedback integration-feedback--success"
      : tone === "error"
        ? "integration-feedback integration-feedback--error"
        : "integration-feedback";
}

function setStatus(enabled: boolean, hasToken: boolean) {
  if (!statusBadge) return;
  statusBadge.textContent = enabled && hasToken ? "Conectada" : enabled ? "Pendente de token" : "Desativada";
  statusBadge.className = enabled && hasToken ? "integration-status integration-status--ok" : "integration-status";
}

function setOpenAIStatus(enabled: boolean, hasToken: boolean) {
  if (!openAIStatusBadge) return;
  openAIStatusBadge.textContent = enabled && hasToken ? "Conectada" : enabled ? "Pendente de token" : "Desativada";
  openAIStatusBadge.className = enabled && hasToken ? "integration-status integration-status--ok" : "integration-status";
}

function escapeHTML(value: unknown) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function renderList(value: unknown) {
  if (Array.isArray(value)) {
    return value.map((item) => escapeHTML(item)).join(", ");
  }
  return escapeHTML(value || "-");
}

async function loadCosmos() {
  try {
    const item = await getCosmosIntegration();
    if (enabledInput) enabledInput.checked = Boolean(item.enabled);
    if (baseURLInput) baseURLInput.value = item.base_url || "https://api.cosmos.bluesoft.com.br";
    if (notesInput) notesInput.value = item.notes || "";
    if (tokenInput) tokenInput.placeholder = item.has_token ? `Token salvo: ${item.token_preview}` : "Cole o token Cosmos";
    setStatus(Boolean(item.enabled), Boolean(item.has_token));
    setFeedback(saveFeedback, item.has_token ? "Token salvo no backend. Deixe em branco para manter o token atual." : "Configure o token para ativar a consulta por codigo de barras.");
  } catch (error) {
    setFeedback(saveFeedback, `Falha ao carregar integracao Cosmos: ${String(error)}`, "error");
  }
}

async function loadOpenAI() {
  try {
    const item = await getOpenAIIntegration();
    if (openAIEnabledInput) openAIEnabledInput.checked = Boolean(item.enabled);
    if (openAIBaseURLInput) openAIBaseURLInput.value = item.base_url || "https://api.openai.com/v1";
    if (openAIModelInput) openAIModelInput.value = item.model_name || "gpt-5.4-mini";
    if (openAINotesInput) openAINotesInput.value = item.notes || "";
    if (openAITokenInput) openAITokenInput.placeholder = item.has_token ? `Token salvo: ${item.token_preview}` : "Cole a OPENAI_API_KEY";
    setOpenAIStatus(Boolean(item.enabled), Boolean(item.has_token));
    setFeedback(openAIFeedback, item.has_token ? "Chave salva no backend. Deixe em branco para manter a chave atual." : "Configure a chave para ativar classificacao assistida.");
  } catch (error) {
    setFeedback(openAIFeedback, `Falha ao carregar integracao OpenAI: ${String(error)}`, "error");
  }
}

saveForm?.addEventListener("submit", async (event) => {
  event.preventDefault();

  try {
    const response = await saveCosmosIntegration({
      enabled: Boolean(enabledInput?.checked),
      base_url: baseURLInput?.value?.trim() || "https://api.cosmos.bluesoft.com.br",
      api_token: tokenInput?.value?.trim() || "",
      notes: notesInput?.value?.trim() || "",
    });

    if (tokenInput) {
      tokenInput.value = "";
      tokenInput.placeholder = response.item.has_token ? `Token salvo: ${response.item.token_preview}` : "Cole o token Cosmos";
    }
    setStatus(Boolean(response.item.enabled), Boolean(response.item.has_token));
    setFeedback(saveFeedback, response.message || "Integracao salva com sucesso.", "success");
  } catch (error) {
    setFeedback(saveFeedback, `Falha ao salvar Cosmos: ${String(error)}`, "error");
  }
});

testForm?.addEventListener("submit", async (event) => {
  event.preventDefault();

  const gtin = testGTINInput?.value?.trim() || "";
  if (!gtin) {
    setFeedback(testFeedback, "Informe um GTIN para testar a consulta.", "error");
    return;
  }

  setFeedback(testFeedback, "Consultando Cosmos...", "muted");
  if (resultBox) resultBox.innerHTML = "";

  try {
    const result = await testCosmosIntegration({
      gtin,
      api_token: tokenInput?.value?.trim() || "",
    });

    setFeedback(testFeedback, result.ok ? "Consulta Cosmos executada com sucesso." : result.message, result.ok ? "success" : "error");
    if (resultBox) {
      resultBox.innerHTML = `
        <div class="cosmos-result-grid">
          <span><strong>Status</strong>${result.status_code}</span>
          <span><strong>GTIN</strong>${result.gtin || "-"}</span>
          <span><strong>Descricao</strong>${result.description || "-"}</span>
          <span><strong>NCM</strong>${result.ncm || "-"}</span>
        </div>
      `;
    }
  } catch (error) {
    setFeedback(testFeedback, `Falha no teste Cosmos: ${String(error)}`, "error");
  }
});

cosmosSearchButton?.addEventListener("click", async () => {
  const query = cosmosSearchInput?.value?.trim() || "";
  if (!query) {
    setFeedback(cosmosSearchFeedback, "Informe uma descricao para buscar produtos.", "error");
    return;
  }

  const originalLabel = cosmosSearchButton.textContent || "Buscar produtos";
  cosmosSearchButton.disabled = true;
  cosmosSearchButton.textContent = "Buscando...";
  setFeedback(cosmosSearchFeedback, "Consultando Cosmos por descricao...", "muted");
  if (cosmosSearchResultBox) cosmosSearchResultBox.innerHTML = "";

  try {
    const result = await searchCosmosProducts({
      query,
      api_token: tokenInput?.value?.trim() || "",
      limit: 12,
    });

    setFeedback(
      cosmosSearchFeedback,
      result.ok ? `${result.items.length} candidato(s) encontrados para integrar ao motor fiscal.` : result.message,
      result.ok ? "success" : "error",
    );

    if (cosmosSearchResultBox) {
      if (!result.items.length) {
        cosmosSearchResultBox.innerHTML = `<div class="integration-feedback">Nenhum candidato retornado para essa busca.</div>`;
      } else {
        cosmosSearchResultBox.innerHTML = `
          <div class="cosmos-search-table" role="table" aria-label="Produtos encontrados na Cosmos">
            <div class="cosmos-search-table__head" role="row">
              <span>Produto</span>
              <span>GTIN</span>
              <span>NCM</span>
              <span>CEST</span>
              <span>Marca</span>
            </div>
            ${result.items
              .map(
                (item) => `
                  <div class="cosmos-search-table__row" role="row">
                    <span>
                      <strong>${escapeHTML(item.description || "-")}</strong>
                      <small>${escapeHTML(item.ncm_description || "NCM usado para validar regras internas")}</small>
                    </span>
                    <span>${escapeHTML(item.gtin || "-")}</span>
                    <span>${escapeHTML(item.ncm || "-")}</span>
                    <span>${escapeHTML(item.cest || "-")}</span>
                    <span>${escapeHTML(item.brand || "-")}</span>
                  </div>
                `,
              )
              .join("")}
          </div>
        `;
      }
    }
  } catch (error) {
    setFeedback(cosmosSearchFeedback, `Falha na busca Cosmos: ${String(error)}`, "error");
  } finally {
    cosmosSearchButton.disabled = false;
    cosmosSearchButton.textContent = originalLabel;
  }
});

openAIForm?.addEventListener("submit", async (event) => {
  event.preventDefault();

  try {
    const response = await saveOpenAIIntegration({
      enabled: Boolean(openAIEnabledInput?.checked),
      base_url: openAIBaseURLInput?.value?.trim() || "https://api.openai.com/v1",
      model_name: openAIModelInput?.value?.trim() || "gpt-5.4-mini",
      api_token: openAITokenInput?.value?.trim() || "",
      notes: openAINotesInput?.value?.trim() || "",
    });

    if (openAITokenInput) {
      openAITokenInput.value = "";
      openAITokenInput.placeholder = response.item.has_token ? `Token salvo: ${response.item.token_preview}` : "Cole a OPENAI_API_KEY";
    }
    setOpenAIStatus(Boolean(response.item.enabled), Boolean(response.item.has_token));
    setFeedback(openAIFeedback, response.message || "Integracao OpenAI salva com sucesso.", "success");
  } catch (error) {
    setFeedback(openAIFeedback, `Falha ao salvar OpenAI: ${String(error)}`, "error");
  }
});

openAITestButton?.addEventListener("click", async () => {
  const originalLabel = openAITestButton.textContent || "Classificar item";
  openAITestButton.disabled = true;
  openAITestButton.textContent = "Classificando...";
  setFeedback(openAITestFeedback, "Testando classificacao assistida...", "muted");
  if (openAIResultBox) openAIResultBox.innerHTML = "";

  try {
    const result = await testOpenAIIntegration({
      api_token: openAITokenInput?.value?.trim() || "",
      model_name: openAIModelInput?.value?.trim() || "gpt-5.4-mini",
      description: openAITestDescription?.value?.trim() || "",
      gtin: openAITestGTIN?.value?.trim() || "",
      ncm: openAITestNCM?.value?.trim() || "",
      cest: openAITestCEST?.value?.trim() || "",
      uf: openAITestUF?.value?.trim() || "",
      tax_regime: openAITestRegime?.value?.trim() || "",
      operation: openAITestOperation?.value?.trim() || "",
    });

    setFeedback(openAITestFeedback, result.ok ? "OpenAI respondeu com sucesso." : result.message, result.ok ? "success" : "error");
    if (openAIResultBox) {
      const classification = result.classification || {};
      openAIResultBox.innerHTML = `
        <div class="ai-classification">
          <div class="ai-classification__head">
            <strong>${escapeHTML(classification.produto_normalizado || "Classificacao assistida")}</strong>
            <span class="ai-risk">Risco ${escapeHTML(classification.risco || "-")}</span>
          </div>
          <div class="ai-fields">
            <span><strong>Status</strong>${escapeHTML(result.status_code)}</span>
            <span><strong>Modelo</strong>${escapeHTML(result.model || "-")}</span>
            <span><strong>Categoria</strong>${escapeHTML(classification.categoria_fiscal_provavel || "-")}</span>
            <span><strong>Confianca</strong>${escapeHTML(classification.confianca ?? "-")}</span>
            <span><strong>NCM</strong>${escapeHTML(classification.ncm_informado || "-")}</span>
            <span><strong>CEST</strong>${escapeHTML(classification.cest_informado || "-")}</span>
            <span><strong>Sinais</strong>${renderList(classification.sinais)}</span>
            <span><strong>Acao recomendada</strong>${escapeHTML(classification.acao_recomendada || "-")}</span>
          </div>
          <div class="integration-feedback">
            ${escapeHTML(classification.observacao || result.output || "-")}
          </div>
        </div>
      `;
    }
  } catch (error) {
    setFeedback(openAITestFeedback, `Falha no teste OpenAI: ${String(error)}`, "error");
  } finally {
    openAITestButton.disabled = false;
    openAITestButton.textContent = originalLabel;
  }
});

void loadCosmos();
void loadOpenAI();
