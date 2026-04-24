import { getOrganizationId, getToken } from "../lib/auth";
import { listInvoices } from "../lib/invoices";

interface Invoice {
  id: string;
  number: string;
  series: string;
  emitter_name: string;
  issued_at: string;
  total_amount: string;
  status: string;
}

const wrapper = document.getElementById("invoice-table-wrapper") as HTMLDivElement | null;

function formatDate(value?: string) {
  if (!value) return "-";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(parsed);
}

function formatMoney(value?: string) {
  const numeric = Number(value || 0);
  if (Number.isNaN(numeric)) return value || "-";
  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  }).format(numeric);
}

async function loadInvoices() {
  if (!wrapper) return;

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

  try {
    const data = await listInvoices();
    const invoices: Invoice[] = data.invoices || [];

    if (!invoices.length) {
      wrapper.innerHTML = `
        <p>Nenhuma nota importada ainda.</p>
        <p><a href="/importacoes">Importe um XML</a> para popular esta tela.</p>
      `;
      return;
    }

    const rows = invoices
      .map(
        (invoice: Invoice) => `
          <tr>
            <td>${invoice.number || "-"}</td>
            <td>${invoice.series || "-"}</td>
            <td>${invoice.emitter_name || "-"}</td>
            <td>${formatDate(invoice.issued_at)}</td>
            <td>${formatMoney(invoice.total_amount)}</td>
            <td>${invoice.status || "-"}</td>
            <td><a href="/invoices/${invoice.id}">Ver</a></td>
          </tr>
        `
      )
      .join("");

    wrapper.innerHTML = `
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
    `;
  } catch (error) {
    wrapper.innerHTML = `
      <p>Erro ao carregar as notas.</p>
      <p>${String(error)}</p>
    `;
  }
}

loadInvoices();
