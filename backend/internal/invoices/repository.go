package invoices

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type CreateInvoiceParams struct {
	OrganizationID  string
	AccessKey       string
	Number          string
	Series          string
	IssuedAt        string
	EmitterName     string
	EmitterCNPJ     string
	EmitterUF       string
	RecipientName   string
	RecipientCNPJ   string
	RecipientUF     string
	OperationNature string
	TotalAmount     string
	XMLRaw          string
	Status          string
}

type CreateInvoiceItemParams struct {
	InvoiceID      string
	ItemNumber     int
	ProductCode    string
	GTIN           string
	GTINTributable string
	Description    string
	NCM            string
	CEST           string
	CFOP           string
	Unit           string
	Quantity       string
	UnitValue      string
	TotalValue     string
	ICMSValue      string
	IPIValue       string
	PISValue       string
	COFINSValue    string
}

func (r *Repository) CreateInvoice(ctx context.Context, p CreateInvoiceParams) (string, error) {
	query := `
		INSERT INTO invoices (
			organization_id,
			access_key,
			number,
			series,
			issued_at,
			emitter_name,
			emitter_cnpj,
			emitter_uf,
			recipient_name,
			recipient_cnpj,
			recipient_uf,
			operation_nature,
			total_amount,
			xml_raw,
			status
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5, '')::timestamp, $6, $7, $8, $9, $10, $11, $12,
			NULLIF($13, '')::numeric, $14, $15
		)
		RETURNING id
	`

	var invoiceID string

	err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(p.OrganizationID),
		strings.TrimSpace(p.AccessKey),
		strings.TrimSpace(p.Number),
		strings.TrimSpace(p.Series),
		strings.TrimSpace(p.IssuedAt),
		strings.TrimSpace(p.EmitterName),
		strings.TrimSpace(p.EmitterCNPJ),
		strings.ToUpper(strings.TrimSpace(p.EmitterUF)),
		strings.TrimSpace(p.RecipientName),
		strings.TrimSpace(p.RecipientCNPJ),
		strings.ToUpper(strings.TrimSpace(p.RecipientUF)),
		strings.TrimSpace(p.OperationNature),
		strings.TrimSpace(p.TotalAmount),
		strings.TrimSpace(p.XMLRaw),
		strings.TrimSpace(p.Status),
	).Scan(&invoiceID)
	if err != nil {
		return "", fmt.Errorf("create invoice: %w", err)
	}

	return invoiceID, nil
}

func (r *Repository) CreateInvoiceItem(ctx context.Context, p CreateInvoiceItemParams) error {
	query := `
		INSERT INTO invoice_items (
			invoice_id,
			item_number,
			product_code,
			gtin,
			gtin_tributable,
			description,
			ncm,
			cest,
			cfop,
			unit,
			quantity,
			unit_value,
			total_value,
			icms_value,
			ipi_value,
			pis_value,
			cofins_value
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			NULLIF($11, '')::numeric,
			NULLIF($12, '')::numeric,
			NULLIF($13, '')::numeric,
			NULLIF($14, '')::numeric,
			NULLIF($15, '')::numeric,
			NULLIF($16, '')::numeric,
			NULLIF($17, '')::numeric
		)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		strings.TrimSpace(p.InvoiceID),
		p.ItemNumber,
		strings.TrimSpace(p.ProductCode),
		strings.TrimSpace(p.GTIN),
		strings.TrimSpace(p.GTINTributable),
		strings.TrimSpace(p.Description),
		strings.TrimSpace(p.NCM),
		strings.TrimSpace(p.CEST),
		strings.TrimSpace(p.CFOP),
		strings.TrimSpace(p.Unit),
		strings.TrimSpace(p.Quantity),
		strings.TrimSpace(p.UnitValue),
		strings.TrimSpace(p.TotalValue),
		strings.TrimSpace(p.ICMSValue),
		strings.TrimSpace(p.IPIValue),
		strings.TrimSpace(p.PISValue),
		strings.TrimSpace(p.COFINSValue),
	)
	if err != nil {
		return fmt.Errorf("create invoice item: %w", err)
	}

	return nil
}

type InvoiceListItem struct {
	ID          string `json:"id"`
	Number      string `json:"number"`
	Series      string `json:"series"`
	IssuedAt    string `json:"issued_at"`
	EmitterName string `json:"emitter_name"`
	TotalAmount string `json:"total_amount"`
	Status      string `json:"status"`
}

func (r *Repository) ListInvoices(ctx context.Context, organizationID string) ([]InvoiceListItem, error) {
	query := `
		SELECT
			id,
			COALESCE(number, ''),
			COALESCE(series, ''),
			COALESCE(issued_at::text, ''),
			COALESCE(emitter_name, ''),
			COALESCE(total_amount::text, ''),
			COALESCE(status, '')
		FROM invoices
		WHERE organization_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(ctx, query, strings.TrimSpace(organizationID))
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	invoices := make([]InvoiceListItem, 0)

	for rows.Next() {
		var item InvoiceListItem

		if err := rows.Scan(
			&item.ID,
			&item.Number,
			&item.Series,
			&item.IssuedAt,
			&item.EmitterName,
			&item.TotalAmount,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan invoice list item: %w", err)
		}

		invoices = append(invoices, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice list: %w", err)
	}

	return invoices, nil
}

type InvoiceItemDetail struct {
	ItemNumber  int    `json:"item_number"`
	ProductCode string `json:"product_code"`
	GTIN        string `json:"gtin"`
	Description string `json:"description"`
	NCM         string `json:"ncm"`
	CEST        string `json:"cest"`
	CFOP        string `json:"cfop"`
	Unit        string `json:"unit"`
	Quantity    string `json:"quantity"`
	UnitValue   string `json:"unit_value"`
	TotalValue  string `json:"total_value"`
	ICMSValue   string `json:"icms_value"`
	IPIValue    string `json:"ipi_value"`
	PISValue    string `json:"pis_value"`
	COFINSValue string `json:"cofins_value"`
}

type InvoiceDetail struct {
	ID              string              `json:"id"`
	Number          string              `json:"number"`
	Series          string              `json:"series"`
	IssuedAt        string              `json:"issued_at"`
	EmitterName     string              `json:"emitter_name"`
	EmitterCNPJ     string              `json:"emitter_cnpj"`
	RecipientName   string              `json:"recipient_name"`
	RecipientCNPJ   string              `json:"recipient_cnpj"`
	OperationNature string              `json:"operation_nature"`
	TotalAmount     string              `json:"total_amount"`
	Status          string              `json:"status"`
	Items           []InvoiceItemDetail `json:"items"`
}

func (r *Repository) GetInvoiceByID(ctx context.Context, organizationID, invoiceID string) (*InvoiceDetail, error) {
	queryInvoice := `
		SELECT
			id,
			COALESCE(number, ''),
			COALESCE(series, ''),
			COALESCE(issued_at::text, ''),
			COALESCE(emitter_name, ''),
			COALESCE(emitter_cnpj, ''),
			COALESCE(recipient_name, ''),
			COALESCE(recipient_cnpj, ''),
			COALESCE(operation_nature, ''),
			COALESCE(total_amount::text, ''),
			COALESCE(status, '')
		FROM invoices
		WHERE id = $1 AND organization_id = $2
	`

	var detail InvoiceDetail

	err := r.db.QueryRow(
		ctx,
		queryInvoice,
		strings.TrimSpace(invoiceID),
		strings.TrimSpace(organizationID),
	).Scan(
		&detail.ID,
		&detail.Number,
		&detail.Series,
		&detail.IssuedAt,
		&detail.EmitterName,
		&detail.EmitterCNPJ,
		&detail.RecipientName,
		&detail.RecipientCNPJ,
		&detail.OperationNature,
		&detail.TotalAmount,
		&detail.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by id: %w", err)
	}

	queryItems := `
		SELECT
			item_number,
			COALESCE(product_code, ''),
			COALESCE(gtin, ''),
			COALESCE(description, ''),
			COALESCE(ncm, ''),
			COALESCE(cest, ''),
			COALESCE(cfop, ''),
			COALESCE(unit, ''),
			COALESCE(quantity::text, ''),
			COALESCE(unit_value::text, ''),
			COALESCE(total_value::text, ''),
			COALESCE(icms_value::text, ''),
			COALESCE(ipi_value::text, ''),
			COALESCE(pis_value::text, ''),
			COALESCE(cofins_value::text, '')
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY item_number ASC
	`

	rows, err := r.db.Query(ctx, queryItems, strings.TrimSpace(invoiceID))
	if err != nil {
		return nil, fmt.Errorf("get invoice items: %w", err)
	}
	defer rows.Close()

	detail.Items = make([]InvoiceItemDetail, 0)

	for rows.Next() {
		var item InvoiceItemDetail

		if err := rows.Scan(
			&item.ItemNumber,
			&item.ProductCode,
			&item.GTIN,
			&item.Description,
			&item.NCM,
			&item.CEST,
			&item.CFOP,
			&item.Unit,
			&item.Quantity,
			&item.UnitValue,
			&item.TotalValue,
			&item.ICMSValue,
			&item.IPIValue,
			&item.PISValue,
			&item.COFINSValue,
		); err != nil {
			return nil, fmt.Errorf("scan invoice item detail: %w", err)
		}

		detail.Items = append(detail.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice items: %w", err)
	}

	return &detail, nil
}