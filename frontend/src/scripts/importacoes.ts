import { listInvoices, uploadInvoices } from "../lib/invoices";

const form = document.getElementById("upload-form");
const fileInput = document.getElementById("xml-file") as HTMLInputElement | null;
const message = document.getElementById("upload-message");
const sessionBox = document.getElementById("upload-session");
const invoicesBox = document.getElementById("upload-invoices");

function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function setMessage(text: string, tone = "muted"): void {
  if (!message) return;
  message.textContent = text;
  message.className =
    tone === "error"
      ? "message-box message-box--error"
      : tone === "success"
        ? "message-box message-box--success"
        : "dashboard-note";
}

function renderInvoices(invoices: any[]): void {
  if (!invoicesBox) return;

  if (!invoices.length) {
    invoicesBox.innerHTML = `
      <div class="dashboard-empty">
        <strong>Nenhuma nota importada.</strong>
        <p>Envie um XML valido para popular o historico de invoices.</p>
      </div>
    `;
    return;
  }

  const rows = invoices
    .map(
      (item) => `
        <tr>
          <td>${item.number || "-"}</td>
          <td>${item.series || "-"}</td>
          <td>${item.emitter_name || "-"}</td>
          <td>${item.issued_at || "-"}</td>
          <td>${item.total_amount || "-"}</td>
          <td>${item.status || "-"}</td>
          <td><a href="/invoices/${item.id}">Abrir</a></td>
        </tr>
      `
    )
    .join("");

  invoicesBox.innerHTML = `
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Numero</th>
            <th>Serie</th>
            <th>Emitente</th>
            <th>Data</th>
            <th>Total</th>
            <th>Status</th>
            <th>Acao</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    </div>
  `;
}

async function loadInvoicesPanel(): Promise<void> {
  const token = readStorage("token");
  const organizationId = readStorage("organization_id");

  if (!token || !organizationId) {
    if (sessionBox) {
      sessionBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Sessao incompleta.</strong>
          <p>Faca login e selecione uma organizacao antes de enviar XMLs.</p>
        </div>
      `;
    }
    renderInvoices([]);
    return;
  }

  if (sessionBox) {
    sessionBox.innerHTML = `
      <div class="session-card">
        <span><strong>Token:</strong> ativo</span>
        <span><strong>Organizacao ativa:</strong> ${organizationId}</span>
      </div>
    `;
  }

  try {
    const response = await listInvoices();
    renderInvoices(response.invoices || []);
  } catch (error) {
    if (invoicesBox) {
      invoicesBox.innerHTML = `
        <div class="dashboard-empty dashboard-empty--error">
          <strong>Falha ao carregar notas.</strong>
          <p>${String(error)}</p>
        </div>
      `;
    }
  }
}

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  const files = Array.from(fileInput?.files || []);
  if (!files.length) {
    setMessage("Selecione ao menos um arquivo XML antes de enviar.", "error");
    return;
  }

  try {
    setMessage(`Enviando ${files.length} XML(s) para processamento...`, "success");
    const response = await uploadInvoices(files);
    const results = Array.isArray(response.results) ? response.results : [];
    const created = results.filter((item: any) => item.success);
    const failed = results.filter((item: any) => !item.success);

    if (message && failed.length) {
      message.className = "message-box message-box--error";
      message.innerHTML = `
        <strong>Lote concluido.</strong><br />
        ${created.length} importado(s) com sucesso e ${failed.length} com falha.<br />
        ${failed
          .slice(0, 5)
          .map((item: any) => `${item.file_name || "arquivo"}: ${item.error || "falha ao processar"}`)
          .join("<br />")}
      `;
    } else {
      setMessage(
        `Lote concluido com sucesso. ${created.length} XML(s) importado(s).`,
        "success"
      );
    }

    if (form instanceof HTMLFormElement) {
      form.reset();
    }
    await loadInvoicesPanel();
  } catch (error) {
    setMessage(`Falha ao processar o lote de XMLs: ${String(error)}`, "error");
  }
});

loadInvoicesPanel();
