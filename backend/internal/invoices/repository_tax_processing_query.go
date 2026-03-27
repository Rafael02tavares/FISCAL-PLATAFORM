package invoices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceTaxProcessingQueryRepository struct {
	db *pgxpool.Pool
}

func NewInvoiceTaxProcessingQueryRepository(db *pgxpool.Pool) (*InvoiceTaxProcessingQueryRepository, error) {
	if db == nil {
		return nil, errors.New("invoice tax processing query repository: db is required")
	}

	return &InvoiceTaxProcessingQueryRepository{
		db: db,
	}, nil
}

func (r *InvoiceTaxProcessingQueryRepository) GetInvoiceForTaxProcessing(
	ctx context.Context,
	invoiceID string,
	organizationID string,
) (*InvoiceTaxProcessingView, error) {
	invoiceID = strings.TrimSpace(invoiceID)
	organizationID = strings.TrimSpace(organizationID)

	if invoiceID == "" {
		return nil, errors.New("invoice tax processing query repository: invoice_id is required")
	}
	if organizationID == "" {
		return nil, errors.New("invoice tax processing query repository: organization_id is required")
	}

	invoice, err := r.fetchInvoice(ctx, invoiceID, organizationID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, nil
	}

	items, err := r.fetchInvoiceItems(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	invoice.Items = items
	return invoice, nil
}

func (r *InvoiceTaxProcessingQueryRepository) fetchInvoice(
	ctx context.Context,
	invoiceID string,
	organizationID string,
) (*InvoiceTaxProcessingView, error) {
	query := `
		SELECT
			i.id,
			COALESCE(i.number, ''),
			COALESCE(i.series, ''),
			COALESCE(i.access_key, ''),
			COALESCE(i.issue_date, i.created_at),

			COALESCE(i.issuer_cnpj, ''),
			COALESCE(i.issuer_cpf, ''),
			COALESCE(i.issuer_uf, ''),
			COALESCE(i.issuer_crt, ''),
			COALESCE(i.issuer_ie, ''),
			COALESCE(i.issuer_is_contributor_icms, false),
			COALESCE(i.issuer_is_final_consumer, false),
			COALESCE(i.issuer_tax_regime_code, ''),
			COALESCE(i.issuer_municipality_code, ''),
			COALESCE(i.issuer_country_code, ''),

			COALESCE(i.recipient_cnpj, ''),
			COALESCE(i.recipient_cpf, ''),
			COALESCE(i.recipient_uf, ''),
			COALESCE(i.recipient_crt, ''),
			COALESCE(i.recipient_ie, ''),
			COALESCE(i.recipient_is_contributor_icms, false),
			COALESCE(i.recipient_is_final_consumer, false),
			COALESCE(i.recipient_tax_regime_code, ''),
			COALESCE(i.recipient_municipality_code, ''),
			COALESCE(i.recipient_country_code, ''),

			COALESCE(i.document_type, 'NFE'),
			COALESCE(i.operation_type, 'EXIT'),
			COALESCE(i.operation_scope, ''),
			COALESCE(i.cfop, ''),
			COALESCE(i.fin_nfe, ''),
			COALESCE(i.presence_indicator, ''),
			COALESCE(i.purpose_code, ''),
			COALESCE(i.is_return, false),
			COALESCE(i.is_transfer, false),
			COALESCE(i.is_resale, false),
			COALESCE(i.is_import, false),
			COALESCE(i.is_export, false),
			COALESCE(i.has_interstate_delivery, false)
		FROM invoices i
		WHERE i.id = $1
		  AND i.organization_id = $2
		LIMIT 1
	`

	row := r.db.QueryRow(ctx, query, invoiceID, organizationID)

	var invoice InvoiceTaxProcessingView

	err := row.Scan(
		&invoice.InvoiceID,
		&invoice.Number,
		&invoice.Series,
		&invoice.AccessKey,
		&invoice.IssueDate,

		&invoice.Issuer.CNPJ,
		&invoice.Issuer.CPF,
		&invoice.Issuer.UF,
		&invoice.Issuer.CRT,
		&invoice.Issuer.IE,
		&invoice.Issuer.IsContributorICMS,
		&invoice.Issuer.IsFinalConsumer,
		&invoice.Issuer.TaxRegimeCode,
		&invoice.Issuer.MunicipalityCode,
		&invoice.Issuer.CountryCode,

		&invoice.Recipient.CNPJ,
		&invoice.Recipient.CPF,
		&invoice.Recipient.UF,
		&invoice.Recipient.CRT,
		&invoice.Recipient.IE,
		&invoice.Recipient.IsContributorICMS,
		&invoice.Recipient.IsFinalConsumer,
		&invoice.Recipient.TaxRegimeCode,
		&invoice.Recipient.MunicipalityCode,
		&invoice.Recipient.CountryCode,

		&invoice.Operation.DocumentType,
		&invoice.Operation.OperationType,
		&invoice.Operation.OperationScope,
		&invoice.Operation.CFOP,
		&invoice.Operation.FinNFe,
		&invoice.Operation.PresenceIndicator,
		&invoice.Operation.PurposeCode,
		&invoice.Operation.IsReturn,
		&invoice.Operation.IsTransfer,
		&invoice.Operation.IsResale,
		&invoice.Operation.IsImport,
		&invoice.Operation.IsExport,
		&invoice.Operation.HasInterstateDelivery,
	)
	if err != nil {
		if isNoRowsError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("invoice tax processing query repository: fetch invoice: %w", err)
	}

	sanitizeInvoiceTaxProcessingView(&invoice)
	return &invoice, nil
}

func (r *InvoiceTaxProcessingQueryRepository) fetchInvoiceItems(
	ctx context.Context,
	invoiceID string,
) ([]ProcessInvoiceTaxItem, error) {
	query := `
		SELECT
			ii.id,
			COALESCE(ii.item_number, 0),
			COALESCE(ii.supplier_id, ''),
			COALESCE(ii.supplier_product_code, ''),
			COALESCE(ii.description, ''),
			COALESCE(ii.additional_description, ''),
			COALESCE(ii.gtin, ''),
			COALESCE(ii.ncm, ''),
			COALESCE(ii.extipi, ''),
			COALESCE(ii.cest, ''),
			COALESCE(ii.origin_code, ''),
			COALESCE(ii.unit, ''),
			COALESCE(ii.commercial_unit, ''),
			COALESCE(ii.tributary_unit, ''),
			COALESCE(ii.quantity, 0),
			COALESCE(ii.unit_value, 0),
			COALESCE(ii.gross_value, 0),
			COALESCE(ii.discount_value, 0),
			COALESCE(ii.freight_value, 0),
			COALESCE(ii.insurance_value, 0),
			COALESCE(ii.other_expenses_value, 0),
			COALESCE(ii.ipi_value, 0),
			COALESCE(ii.icms_base_value, 0),
			COALESCE(ii.total_value, 0)
		FROM invoice_items ii
		WHERE ii.invoice_id = $1
		ORDER BY ii.item_number ASC, ii.id ASC
	`

	rows, err := r.db.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("invoice tax processing query repository: query invoice items: %w", err)
	}
	defer rows.Close()

	items := make([]ProcessInvoiceTaxItem, 0, 32)

	for rows.Next() {
		var item ProcessInvoiceTaxItem

		err := rows.Scan(
			&item.InvoiceItemID,
			&item.ItemNumber,
			&item.SupplierID,
			&item.SupplierProductCode,
			&item.Description,
			&item.AdditionalDescription,
			&item.GTIN,
			&item.NCM,
			&item.EXTIPI,
			&item.CEST,
			&item.OriginCode,
			&item.Unit,
			&item.CommercialUnit,
			&item.TributaryUnit,
			&item.Quantity,
			&item.UnitValue,
			&item.GrossValue,
			&item.DiscountValue,
			&item.FreightValue,
			&item.InsuranceValue,
			&item.OtherExpensesValue,
			&item.IPIValue,
			&item.ICMSBaseValue,
			&item.TotalValue,
		)
		if err != nil {
			return nil, fmt.Errorf("invoice tax processing query repository: scan invoice item: %w", err)
		}

		sanitizeProcessInvoiceTaxItem(&item)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("invoice tax processing query repository: iterate invoice items: %w", err)
	}

	return items, nil
}

func sanitizeInvoiceTaxProcessingView(v *InvoiceTaxProcessingView) {
	if v == nil {
		return
	}

	v.InvoiceID = strings.TrimSpace(v.InvoiceID)
	v.Number = strings.TrimSpace(v.Number)
	v.Series = strings.TrimSpace(v.Series)
	v.AccessKey = strings.TrimSpace(v.AccessKey)

	v.Issuer.CNPJ = strings.TrimSpace(v.Issuer.CNPJ)
	v.Issuer.CPF = strings.TrimSpace(v.Issuer.CPF)
	v.Issuer.UF = strings.TrimSpace(v.Issuer.UF)
	v.Issuer.CRT = strings.TrimSpace(v.Issuer.CRT)
	v.Issuer.IE = strings.TrimSpace(v.Issuer.IE)
	v.Issuer.TaxRegimeCode = strings.TrimSpace(v.Issuer.TaxRegimeCode)
	v.Issuer.MunicipalityCode = strings.TrimSpace(v.Issuer.MunicipalityCode)
	v.Issuer.CountryCode = strings.TrimSpace(v.Issuer.CountryCode)

	v.Recipient.CNPJ = strings.TrimSpace(v.Recipient.CNPJ)
	v.Recipient.CPF = strings.TrimSpace(v.Recipient.CPF)
	v.Recipient.UF = strings.TrimSpace(v.Recipient.UF)
	v.Recipient.CRT = strings.TrimSpace(v.Recipient.CRT)
	v.Recipient.IE = strings.TrimSpace(v.Recipient.IE)
	v.Recipient.TaxRegimeCode = strings.TrimSpace(v.Recipient.TaxRegimeCode)
	v.Recipient.MunicipalityCode = strings.TrimSpace(v.Recipient.MunicipalityCode)
	v.Recipient.CountryCode = strings.TrimSpace(v.Recipient.CountryCode)

	v.Operation.DocumentType = strings.TrimSpace(v.Operation.DocumentType)
	v.Operation.OperationType = strings.TrimSpace(v.Operation.OperationType)
	v.Operation.OperationScope = strings.TrimSpace(v.Operation.OperationScope)
	v.Operation.CFOP = strings.TrimSpace(v.Operation.CFOP)
	v.Operation.FinNFe = strings.TrimSpace(v.Operation.FinNFe)
	v.Operation.PresenceIndicator = strings.TrimSpace(v.Operation.PresenceIndicator)
	v.Operation.PurposeCode = strings.TrimSpace(v.Operation.PurposeCode)
}

func sanitizeProcessInvoiceTaxItem(item *ProcessInvoiceTaxItem) {
	if item == nil {
		return
	}

	item.InvoiceItemID = strings.TrimSpace(item.InvoiceItemID)
	item.SupplierID = strings.TrimSpace(item.SupplierID)
	item.SupplierProductCode = strings.TrimSpace(item.SupplierProductCode)
	item.Description = strings.TrimSpace(item.Description)
	item.AdditionalDescription = strings.TrimSpace(item.AdditionalDescription)
	item.GTIN = strings.TrimSpace(item.GTIN)
	item.NCM = strings.TrimSpace(item.NCM)
	item.EXTIPI = strings.TrimSpace(item.EXTIPI)
	item.CEST = strings.TrimSpace(item.CEST)
	item.OriginCode = strings.TrimSpace(item.OriginCode)
	item.Unit = strings.TrimSpace(item.Unit)
	item.CommercialUnit = strings.TrimSpace(item.CommercialUnit)
	item.TributaryUnit = strings.TrimSpace(item.TributaryUnit)
}

func isNoRowsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no rows in result set")
}

var _ InvoiceQueryService = (*InvoiceTaxProcessingQueryRepository)(nil)