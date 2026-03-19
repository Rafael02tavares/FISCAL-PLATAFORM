package catalog

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type RegisterObservedItemParams struct {
	OrganizationID  string
	SourceInvoiceID string

	GTIN        string
	Description string

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
}

func (s *Service) RegisterObservedItem(ctx context.Context, p RegisterObservedItemParams) error {
	normalizedGTIN := NormalizeGTIN(p.GTIN)
	normalizedDescription := NormalizeDescription(p.Description)

	if normalizedDescription == "" && normalizedGTIN == "" {
		return errors.New("cannot register item without gtin or description")
	}

	var productID string

	if normalizedGTIN != "" {
		product, err := s.repo.FindProductByNormalizedGTIN(ctx, normalizedGTIN)
		if err == nil && product != nil {
			productID = product.ID
		}
	}

	if productID == "" && normalizedDescription != "" {
		product, err := s.repo.FindProductByNormalizedDescription(ctx, normalizedDescription)
		if err == nil && product != nil {
			productID = product.ID
		}
	}

	if productID == "" {
		id, err := s.repo.CreateProduct(
			ctx,
			p.GTIN,
			normalizedGTIN,
			p.Description,
			normalizedDescription,
		)
		if err != nil {
			return err
		}
		productID = id
	}

	confidence := 0.55
	if normalizedGTIN != "" {
		confidence = 0.90
	}

	return s.repo.CreateTaxProfile(ctx, CreateTaxProfileParams{
		ProductID:       productID,
		OrganizationID:  p.OrganizationID,
		SourceInvoiceID: p.SourceInvoiceID,

		NCM:         p.NCM,
		CEST:        p.CEST,
		CFOP:        p.CFOP,
		ICMSValue:   p.ICMSValue,
		IPIValue:    p.IPIValue,
		PISValue:    p.PISValue,
		COFINSValue: p.COFINSValue,

		EmitterUF:       p.EmitterUF,
		RecipientUF:     p.RecipientUF,
		OperationNature: p.OperationNature,
		ConfidenceScore: confidence,
		SourceType:      "invoice_import",
	})
}
