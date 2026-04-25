import { listFiscalWatcherEvents, listFiscalWatcherSources, runFiscalWatcherCheck } from "../lib/fiscal-watcher";

const sourcesBox = document.getElementById("watcher-sources");
const eventsBox = document.getElementById("watcher-events");
const feedbackBox = document.getElementById("watcher-feedback");
const runAllButton = document.getElementById("watcher-run-all") as HTMLButtonElement | null;

function setFeedback(message: string, tone: "muted" | "success" | "error" = "muted") {
  if (!feedbackBox) return;
  feedbackBox.textContent = message;
  feedbackBox.className =
    tone === "success"
      ? "watcher-feedback watcher-feedback--success"
      : tone === "error"
        ? "watcher-feedback watcher-feedback--error"
        : "watcher-feedback";
}

function formatDate(value?: string) {
  if (!value) return "-";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(parsed);
}

function severityClass(value: string) {
  switch (value) {
    case "high":
      return "watcher-badge watcher-badge--high";
    case "medium":
      return "watcher-badge watcher-badge--medium";
    default:
      return "watcher-badge watcher-badge--low";
  }
}

function renderSources(items: Awaited<ReturnType<typeof listFiscalWatcherSources>>["items"]) {
  if (!sourcesBox) return;

  sourcesBox.innerHTML = items
    .map(
      (item) => `
        <article class="watcher-source-card" data-filter-item>
          <div class="watcher-source-card__top">
            <div>
              <strong>${item.name}</strong>
              <p>${item.authority}</p>
            </div>
            <span class="watcher-source-card__type">${item.source_type}</span>
          </div>
          <div class="watcher-source-card__meta">
            <span><strong>Cadencia:</strong> ${item.cadence_hours}h</span>
            <span><strong>Ultima verificacao:</strong> ${formatDate(item.last_checked_at)}</span>
            <span><strong>Status:</strong> ${item.last_status}</span>
          </div>
          <div class="watcher-source-card__actions">
            <a class="watcher-link" href="${item.url}" target="_blank" rel="noreferrer">Abrir fonte</a>
            <button class="watcher-button" type="button" data-source-code="${item.code}">Verificar</button>
          </div>
        </article>
      `
    )
    .join("");

  sourcesBox.querySelectorAll<HTMLButtonElement>("[data-source-code]").forEach((button) => {
    button.addEventListener("click", async () => {
      const sourceCode = button.dataset.sourceCode || "";
      const originalLabel = button.textContent || "Verificar";

      button.disabled = true;
      button.textContent = "Verificando...";
      setFeedback("Executando verificacao da fonte selecionada.");

      try {
        const response = await runFiscalWatcherCheck(sourceCode);
        setFeedback(response.message || "Verificacao registrada.", "success");
        await loadWatcher();
      } catch (error) {
        setFeedback(`Falha ao verificar fonte: ${String(error)}`, "error");
      } finally {
        button.disabled = false;
        button.textContent = originalLabel;
      }
    });
  });
}

function renderEvents(items: Awaited<ReturnType<typeof listFiscalWatcherEvents>>["items"]) {
  if (!eventsBox) return;

  if (!items.length) {
    eventsBox.innerHTML = `<div class="dashboard-note">Nenhum evento de verificacao registrado.</div>`;
    return;
  }

  eventsBox.innerHTML = `
    <div class="watcher-event-list">
      ${items
        .map(
          (item) => `
            <article class="watcher-event-card" data-filter-item>
              <div class="watcher-event-card__top">
                <div>
                  <strong>${item.title}</strong>
                  <p>${item.source_name} · ${item.authority}</p>
                </div>
                <span class="${severityClass(item.severity)}">${item.severity}</span>
              </div>
              <p>${item.summary}</p>
              <div class="watcher-event-card__meta">
                <span><strong>Status:</strong> ${item.status}</span>
                <span><strong>Modo:</strong> ${item.detection_mode}</span>
                <span><strong>Detectado em:</strong> ${formatDate(item.detected_at)}</span>
              </div>
            </article>
          `
        )
        .join("")}
    </div>
  `;
}

async function loadWatcher() {
  try {
    const [sourcesResponse, eventsResponse] = await Promise.all([
      listFiscalWatcherSources(),
      listFiscalWatcherEvents("", 20),
    ]);

    renderSources(Array.isArray(sourcesResponse?.items) ? sourcesResponse.items : []);
    renderEvents(Array.isArray(eventsResponse?.items) ? eventsResponse.items : []);
  } catch (error) {
    setFeedback(`Falha ao carregar o watcher fiscal: ${String(error)}`, "error");
  }
}

runAllButton?.addEventListener("click", async () => {
  const originalLabel = runAllButton.textContent || "Verificar tudo";

  runAllButton.disabled = true;
  runAllButton.textContent = "Verificando...";
  setFeedback("Executando verificacao geral das fontes.");

  try {
    const response = await runFiscalWatcherCheck("");
    setFeedback(response.message || "Verificacao geral registrada.", "success");
    await loadWatcher();
  } catch (error) {
    setFeedback(`Falha ao verificar fontes: ${String(error)}`, "error");
  } finally {
    runAllButton.disabled = false;
    runAllButton.textContent = originalLabel;
  }
});

loadWatcher();
