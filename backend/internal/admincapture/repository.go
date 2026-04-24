package admincapture

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Candidate struct {
	InvoiceID         string `json:"invoice_id"`
	InvoiceItemID     string `json:"invoice_item_id"`
	InvoiceNumber     string `json:"invoice_number"`
	InvoiceSeries     string `json:"invoice_series"`
	IssuedAt          string `json:"issued_at"`
	EmitterName       string `json:"emitter_name"`
	EmitterUF         string `json:"emitter_uf"`
	RecipientUF       string `json:"recipient_uf"`
	OperationNature   string `json:"operation_nature"`
	ItemNumber        int    `json:"item_number"`
	ProductCode       string `json:"product_code"`
	GTIN              string `json:"gtin"`
	Description       string `json:"description"`
	NCM               string `json:"ncm"`
	CEST              string `json:"cest"`
	CFOP              string `json:"cfop"`
	ICMSCST           string `json:"icms_cst"`
	CSOSN             string `json:"csosn"`
	ICMSRate          string `json:"icms_rate"`
	PISCST            string `json:"pis_cst"`
	PISRate           string `json:"pis_rate"`
	COFINSCST         string `json:"cofins_cst"`
	COFINSRate        string `json:"cofins_rate"`
	ICMSValue         string `json:"icms_value"`
	IPIValue          string `json:"ipi_value"`
	PISValue          string `json:"pis_value"`
	COFINSValue       string `json:"cofins_value"`
	HasObservedTaxes  bool   `json:"has_observed_taxes"`
	ObservedTaxRegime string `json:"observed_tax_regime"`
	ObservedCRT       string `json:"observed_crt"`
}

func (r *Repository) ListCandidates(ctx context.Context, organizationID string, limit int) ([]Candidate, error) {
	query := `
		SELECT
			i.id::text,
			ii.id::text,
			COALESCE(i.number, ''),
			COALESCE(i.series, ''),
			COALESCE(i.issued_at::text, ''),
			COALESCE(i.emitter_name, ''),
			COALESCE(i.emitter_uf, ''),
			COALESCE(i.recipient_uf, ''),
			COALESCE(i.operation_nature, ''),
			ii.item_number,
			COALESCE(ii.product_code, ''),
			COALESCE(ii.gtin, ''),
			COALESCE(ii.description, ''),
			COALESCE(ii.ncm, ''),
			COALESCE(ii.cest, ''),
			COALESCE(ii.cfop, ''),
			COALESCE(ii.icms_cst, COALESCE(ii.cst, ''), ''),
			COALESCE(ii.csosn, ''),
			COALESCE(ii.icms_rate::text, ''),
			COALESCE(ii.pis_cst, ''),
			COALESCE(ii.pis_rate::text, ''),
			COALESCE(ii.cofins_cst, ''),
			COALESCE(ii.cofins_rate::text, ''),
			COALESCE(ii.icms_value::text, ''),
			COALESCE(ii.ipi_value::text, ''),
			COALESCE(ii.pis_value::text, ''),
			COALESCE(ii.cofins_value::text, ''),
			COALESCE(o.tax_regime, ''),
			COALESCE(o.crt, '')
		FROM invoice_items ii
		INNER JOIN invoices i ON i.id = ii.invoice_id
		INNER JOIN organizations o ON o.id = i.organization_id
		WHERE i.organization_id = $1
		  AND (
			COALESCE(ii.ncm, '') <> ''
			OR COALESCE(ii.cest, '') <> ''
			OR COALESCE(ii.cfop, '') <> ''
			OR COALESCE(ii.icms_cst, COALESCE(ii.cst, ''), '') <> ''
			OR COALESCE(ii.csosn, '') <> ''
			OR COALESCE(ii.pis_cst, '') <> ''
			OR COALESCE(ii.cofins_cst, '') <> ''
		  )
		ORDER BY i.created_at DESC, ii.item_number ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, organizationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list capture candidates: %w", err)
	}
	defer rows.Close()

	items := make([]Candidate, 0)
	for rows.Next() {
		var item Candidate
		if err := rows.Scan(
			&item.InvoiceID,
			&item.InvoiceItemID,
			&item.InvoiceNumber,
			&item.InvoiceSeries,
			&item.IssuedAt,
			&item.EmitterName,
			&item.EmitterUF,
			&item.RecipientUF,
			&item.OperationNature,
			&item.ItemNumber,
			&item.ProductCode,
			&item.GTIN,
			&item.Description,
			&item.NCM,
			&item.CEST,
			&item.CFOP,
			&item.ICMSCST,
			&item.CSOSN,
			&item.ICMSRate,
			&item.PISCST,
			&item.PISRate,
			&item.COFINSCST,
			&item.COFINSRate,
			&item.ICMSValue,
			&item.IPIValue,
			&item.PISValue,
			&item.COFINSValue,
			&item.ObservedTaxRegime,
			&item.ObservedCRT,
		); err != nil {
			return nil, fmt.Errorf("scan capture candidate: %w", err)
		}

		item.HasObservedTaxes =
			item.NCM != "" ||
				item.CEST != "" ||
				item.CFOP != "" ||
				item.ICMSCST != "" ||
				item.CSOSN != "" ||
				item.PISCST != "" ||
				item.COFINSCST != ""

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate capture candidates: %w", err)
	}

	return items, nil
}

func (r *Repository) GetCandidateByInvoiceItemID(ctx context.Context, organizationID, invoiceItemID string) (*Candidate, error) {
	query := `
		SELECT
			i.id::text,
			ii.id::text,
			COALESCE(i.number, ''),
			COALESCE(i.series, ''),
			COALESCE(i.issued_at::text, ''),
			COALESCE(i.emitter_name, ''),
			COALESCE(i.emitter_uf, ''),
			COALESCE(i.recipient_uf, ''),
			COALESCE(i.operation_nature, ''),
			ii.item_number,
			COALESCE(ii.product_code, ''),
			COALESCE(ii.gtin, ''),
			COALESCE(ii.description, ''),
			COALESCE(ii.ncm, ''),
			COALESCE(ii.cest, ''),
			COALESCE(ii.cfop, ''),
			COALESCE(ii.icms_cst, COALESCE(ii.cst, ''), ''),
			COALESCE(ii.csosn, ''),
			COALESCE(ii.icms_rate::text, ''),
			COALESCE(ii.pis_cst, ''),
			COALESCE(ii.pis_rate::text, ''),
			COALESCE(ii.cofins_cst, ''),
			COALESCE(ii.cofins_rate::text, ''),
			COALESCE(ii.icms_value::text, ''),
			COALESCE(ii.ipi_value::text, ''),
			COALESCE(ii.pis_value::text, ''),
			COALESCE(ii.cofins_value::text, ''),
			COALESCE(o.tax_regime, ''),
			COALESCE(o.crt, '')
		FROM invoice_items ii
		INNER JOIN invoices i ON i.id = ii.invoice_id
		INNER JOIN organizations o ON o.id = i.organization_id
		WHERE i.organization_id = $1
		  AND ii.id = $2::uuid
	`

	var item Candidate
	if err := r.db.QueryRow(ctx, query, organizationID, invoiceItemID).Scan(
		&item.InvoiceID,
		&item.InvoiceItemID,
		&item.InvoiceNumber,
		&item.InvoiceSeries,
		&item.IssuedAt,
		&item.EmitterName,
		&item.EmitterUF,
		&item.RecipientUF,
		&item.OperationNature,
		&item.ItemNumber,
		&item.ProductCode,
		&item.GTIN,
		&item.Description,
		&item.NCM,
		&item.CEST,
		&item.CFOP,
		&item.ICMSCST,
		&item.CSOSN,
		&item.ICMSRate,
		&item.PISCST,
		&item.PISRate,
		&item.COFINSCST,
		&item.COFINSRate,
		&item.ICMSValue,
		&item.IPIValue,
		&item.PISValue,
		&item.COFINSValue,
		&item.ObservedTaxRegime,
		&item.ObservedCRT,
	); err != nil {
		return nil, fmt.Errorf("get capture candidate by invoice item id: %w", err)
	}

	item.HasObservedTaxes =
		item.NCM != "" ||
			item.CEST != "" ||
			item.CFOP != "" ||
			item.ICMSCST != "" ||
			item.CSOSN != "" ||
			item.PISCST != "" ||
			item.COFINSCST != ""

	return &item, nil
}
