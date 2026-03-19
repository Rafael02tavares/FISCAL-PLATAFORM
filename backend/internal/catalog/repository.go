package catalog

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

type Product struct {
	ID                    string
	GTIN                  string
	NormalizedGTIN        string
	Description           string
	NormalizedDescription string
}

type ProductTaxProfile struct {
	ID              string
	ProductID       string
	OrganizationID  string
	SourceInvoiceID string

	NCM         string
	CEST        string
	CFOP        string
	ICMSValue   string
	IPIValue    string
	PISValue    string
	COFINSValue string

	EmitterUF       string
	RecipientUF     string
	OperationNature string
	ConfidenceScore float64
	SourceType      string
}

func (r *Repository) FindProductByNormalizedGTIN(ctx context.Context, normalizedGTIN string) (*Product, error) {
	query := `
		SELECT id, COALESCE(gtin, ''), COALESCE(normalized_gtin, ''), description, normalized_description
		FROM products
		WHERE normalized_gtin = $1
		LIMIT 1
	`

	var p Product
	err := r.db.QueryRow(ctx, query, normalizedGTIN).Scan(
		&p.ID,
		&p.GTIN,
		&p.NormalizedGTIN,
		&p.Description,
		&p.NormalizedDescription,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *Repository) FindProductByNormalizedDescription(ctx context.Context, normalizedDescription string) (*Product, error) {
	query := `
		SELECT id, COALESCE(gtin, ''), COALESCE(normalized_gtin, ''), description, normalized_description
		FROM products
		WHERE normalized_description = $1
		LIMIT 1
	`

	var p Product
	err := r.db.QueryRow(ctx, query, normalizedDescription).Scan(
		&p.ID,
		&p.GTIN,
		&p.NormalizedGTIN,
		&p.Description,
		&p.NormalizedDescription,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *Repository) CreateProduct(ctx context.Context, gtin, normalizedGTIN, description, normalizedDescription string) (string, error) {
	query := `
		INSERT INTO products (gtin, normalized_gtin, description, normalized_description)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var productID string
	err := r.db.QueryRow(ctx, query, gtin, normalizedGTIN, description, normalizedDescription).Scan(&productID)
	if err != nil {
		return "", fmt.Errorf("create product: %w", err)
	}

	return productID, nil
}

type CreateTaxProfileParams struct {
	ProductID       string
	OrganizationID  string
	SourceInvoiceID string

	NCM         string
	CEST        string
	CFOP        string
	ICMSValue   string
	IPIValue    string
	PISValue    string
	COFINSValue string

	EmitterUF       string
	RecipientUF     string
	OperationNature string
	ConfidenceScore float64
	SourceType      string
}

func (r *Repository) CreateTaxProfile(ctx context.Context, p CreateTaxProfileParams) error {
	query := `
		INSERT INTO product_tax_profiles (
			product_id,
			organization_id,
			source_invoice_id,
			ncm,
			cest,
			cfop,
			icms_value,
			ipi_value,
			pis_value,
			cofins_value,
			emitter_uf,
			recipient_uf,
			operation_nature,
			confidence_score,
			source_type
		)
		VALUES (
			$1, $2, NULLIF($3, '')::uuid, $4, $5, $6,
			NULLIF($7, '')::numeric,
			NULLIF($8, '')::numeric,
			NULLIF($9, '')::numeric,
			NULLIF($10, '')::numeric,
			$11, $12, $13, $14, $15
		)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		p.ProductID,
		p.OrganizationID,
		p.SourceInvoiceID,
		p.NCM,
		p.CEST,
		p.CFOP,
		p.ICMSValue,
		p.IPIValue,
		p.PISValue,
		p.COFINSValue,
		p.EmitterUF,
		p.RecipientUF,
		p.OperationNature,
		p.ConfidenceScore,
		p.SourceType,
	)
	if err != nil {
		return fmt.Errorf("create tax profile: %w", err)
	}

	return nil
}
