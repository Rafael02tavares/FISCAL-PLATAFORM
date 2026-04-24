import { getToken } from "../lib/auth";
import { listImportBatches, uploadCFOPCatalog, uploadNCMCatalog } from "../lib/admin-imports";
import * as XLSX from "xlsx";

type CatalogType = "ncm" | "cfop";

type ImportConfig = {
  type: CatalogType;
  formId: string;
  sourceInputId: string;
  versionInputId: string;
  fileInputId: string;
  feedbackId: string;
  submitButtonId: string;
  defaultSourceName: string;
  importingLabel: string;
  successFallback: string;
  upload: (file: File, sourceName: string, versionLabel: string) => Promise<any>;
};

const batchesBox = document.getElementById("import-batches");

const configs: ImportConfig[] = [
  {
    type: "ncm",
    formId: "ncm-import-form",
    sourceInputId: "source-name",
    versionInputId: "version-label",
    fileInputId: "ncm-file",
    feedbackId: "import-feedback",
    submitButtonId: "submit-import",
    defaultSourceName: "NCM CSV",
    importingLabel: "Importando NCM...",
    successFallback: "Importacao de NCM concluida com sucesso.",
    upload: uploadNCMCatalog,
  },
  {
    type: "cfop",
    formId: "cfop-import-form",
    sourceInputId: "cfop-source-name",
    versionInputId: "cfop-version-label",
    fileInputId: "cfop-file",
    feedbackId: "cfop-import-feedback",
    submitButtonId: "submit-cfop-import",
    defaultSourceName: "CFOP XLSX",
    importingLabel: "Importando CFOP...",
    successFallback: "Importacao de CFOP concluida com sucesso.",
    upload: uploadCFOPCatalog,
  },
];

function isSpreadsheetFile(file: File) {
  return file.name.toLowerCase().endsWith(".xlsx");
}

async function normalizeImportFile(file: File) {
  if (!isSpreadsheetFile(file)) {
    return file;
  }

  const buffer = await file.arrayBuffer();
  const workbook = XLSX.read(buffer, { type: "array" });
  const firstSheetName = workbook.SheetNames[0];

  if (!firstSheetName) {
    throw new Error("o arquivo XLSX nao possui planilhas validas");
  }

  const firstSheet = workbook.Sheets[firstSheetName];
  const csv = XLSX.utils.sheet_to_csv(firstSheet, {
    FS: ";",
    blankrows: false,
  });

  if (!csv.trim()) {
    throw new Error("a primeira planilha do XLSX esta vazia");
  }

  const normalizedName = file.name.replace(/\.xlsx$/i, ".csv");
  return new File([csv], normalizedName, { type: "text/csv;charset=utf-8" });
}

function setFeedback(element: HTMLElement | null, message: string, tone: "muted" | "success" | "error" = "muted") {
  if (!element) return;

  element.textContent = message;
  element.className =
    tone === "success"
      ? "feedback feedback--success"
      : tone === "error"
        ? "feedback feedback--error"
        : "dashboard-note";
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

function renderBatches(items: any[]) {
  if (!batchesBox) return;

  if (!items.length) {
    batchesBox.innerHTML = `
      <div class="dashboard-note">
        Nenhum lote encontrado ainda. A primeira importacao aparecera aqui.
      </div>
    `;
    return;
  }

  const rows = items
    .map(
      (item) => `
        <article class="batch-card">
          <div class="batch-card__top">
            <strong>${item.source_name || "Importacao"}</strong>
            <span class="batch-badge">${item.success_rows || 0} ok</span>
          </div>
          <div class="batch-card__meta">
            <span><strong>Versao:</strong> ${item.version_label || "-"}</span>
            <span><strong>Arquivo:</strong> ${item.file_name || "-"}</span>
            <span><strong>Tipo:</strong> ${item.source_type || "-"}</span>
            <span><strong>Processado em:</strong> ${formatDate(item.imported_at)}</span>
            <span><strong>Linhas:</strong> ${item.total_rows || 0} totais / ${item.failed_rows || 0} falhas</span>
          </div>
        </article>
      `
    )
    .join("");

  batchesBox.innerHTML = `<div class="batch-list">${rows}</div>`;
}

async function loadBatches() {
  if (!getToken()) {
    window.location.href = "/login";
    return;
  }

  try {
    const response = await listImportBatches("", 12);
    renderBatches(Array.isArray(response?.items) ? response.items : []);
  } catch (error) {
    if (!batchesBox) return;

    batchesBox.innerHTML = `
      <div class="feedback feedback--error">
        Falha ao consultar historico de importacoes: ${String(error)}
      </div>
    `;
  }
}

function wireImportForm(config: ImportConfig) {
  const form = document.getElementById(config.formId) as HTMLFormElement | null;
  const sourceInput = document.getElementById(config.sourceInputId) as HTMLInputElement | null;
  const versionInput = document.getElementById(config.versionInputId) as HTMLInputElement | null;
  const fileInput = document.getElementById(config.fileInputId) as HTMLInputElement | null;
  const feedback = document.getElementById(config.feedbackId) as HTMLElement | null;
  const submitButton = document.getElementById(config.submitButtonId) as HTMLButtonElement | null;

  form?.addEventListener("submit", async (event) => {
    event.preventDefault();

    const file = fileInput?.files?.[0];
    if (!file) {
      setFeedback(feedback, "Selecione um arquivo CSV ou XLSX antes de importar.", "error");
      return;
    }

    const sourceName = sourceInput?.value?.trim() || config.defaultSourceName;
    const versionLabel = versionInput?.value?.trim() || "";
    const originalLabel = submitButton?.textContent || config.importingLabel;

    if (submitButton) {
      submitButton.disabled = true;
      submitButton.textContent = config.importingLabel;
    }

    setFeedback(feedback, `Processando arquivo de ${config.type.toUpperCase()}. Isso pode levar alguns instantes.`, "muted");

    try {
      const normalizedFile = await normalizeImportFile(file);
      const response = await config.upload(normalizedFile, sourceName, versionLabel);
      setFeedback(
        feedback,
        typeof response?.message === "string" ? response.message : config.successFallback,
        "success"
      );

      renderBatches(Array.isArray(response?.items) ? response.items : []);
      form.reset();
      if (sourceInput) {
        sourceInput.value = config.defaultSourceName;
      }
    } catch (error) {
      setFeedback(feedback, `Falha na importacao: ${String(error)}`, "error");
    } finally {
      if (submitButton) {
        submitButton.disabled = false;
        submitButton.textContent = originalLabel;
      }
    }
  });
}

configs.forEach(wireImportForm);
loadBatches();
