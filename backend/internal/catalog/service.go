package catalog

import (
	"context"
	"errors"
	"strings"
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
	ProductID       string

	ProductCode string
	GTIN        string
	Description string

	NCM               string
	NCMEx             string
	CEST              string
	CFOP              string
	CClasTrib         string
	PISCST            string
	COFINSCST         string
	PISRevenueCode    string
	COFINSRevenueCode string
	ICMSCST           string
	CSOSN             string
	CBenef            string

	ICMSValue         string
	IPIValue          string
	PISValue          string
	COFINSValue       string
	PISRate           string
	COFINSRate        string
	ICMSRate          string
	ICMSBaseReduction string
	FCPRate           string
	ICMSSTRate        string
	IBSRate           string
	CBSRate           string
	SelectiveTaxCode  string
	SelectiveTaxRate  string

	OperationCode     string
	EmitterUF         string
	RecipientUF       string
	OperationNature   string
	TargetTaxRegime   string
	ObservedTaxRegime string
	TargetCRT         string
	ObservedCRT       string
	SourceType        string
}

type SaveManualProductParams struct {
	OrganizationID string
	ProductID      string
	ProductCode    string
	GTIN           string
	Description    string

	NCM               string
	NCMEx             string
	CEST              string
	CFOP              string
	CClasTrib         string
	PISCST            string
	COFINSCST         string
	PISRevenueCode    string
	COFINSRevenueCode string
	ICMSCST           string
	CSOSN             string
	CBenef            string

	ICMSValue         string
	IPIValue          string
	PISValue          string
	COFINSValue       string
	PISRate           string
	COFINSRate        string
	ICMSRate          string
	ICMSBaseReduction string
	FCPRate           string
	ICMSSTRate        string
	IBSRate           string
	CBSRate           string
	SelectiveTaxCode  string
	SelectiveTaxRate  string

	OperationCode     string
	EmitterUF         string
	RecipientUF       string
	OperationNature   string
	TargetTaxRegime   string
	ObservedTaxRegime string
	TargetCRT         string
	ObservedCRT       string
}

type SaveReviewedProductParams struct {
	SaveManualProductParams
	ConfidenceScore float64
	SourceType      string
}

func (s *Service) RegisterObservedItem(ctx context.Context, p RegisterObservedItemParams) error {
	productID, err := s.resolveProductID(ctx, p.ProductID, p.ProductCode, p.GTIN, p.Description)
	if err != nil {
		return err
	}

	confidence := 0.55
	if NormalizeGTIN(p.GTIN) != "" {
		confidence = 0.90
	}

	sourceType := strings.TrimSpace(p.SourceType)
	if sourceType == "" {
		sourceType = "invoice_import"
	}

	return s.repo.CreateTaxProfile(ctx, CreateTaxProfileParams{
		ProductID:       productID,
		OrganizationID:  p.OrganizationID,
		SourceInvoiceID: p.SourceInvoiceID,

		NCM:               p.NCM,
		NCMEx:             p.NCMEx,
		CEST:              p.CEST,
		CFOP:              p.CFOP,
		CClasTrib:         p.CClasTrib,
		PISCST:            p.PISCST,
		COFINSCST:         p.COFINSCST,
		PISRevenueCode:    p.PISRevenueCode,
		COFINSRevenueCode: p.COFINSRevenueCode,
		ICMSCST:           p.ICMSCST,
		CSOSN:             p.CSOSN,
		CBenef:            p.CBenef,

		ICMSValue:         p.ICMSValue,
		IPIValue:          p.IPIValue,
		PISValue:          p.PISValue,
		COFINSValue:       p.COFINSValue,
		PISRate:           p.PISRate,
		COFINSRate:        p.COFINSRate,
		ICMSRate:          p.ICMSRate,
		ICMSBaseReduction: p.ICMSBaseReduction,
		FCPRate:           p.FCPRate,
		ICMSSTRate:        p.ICMSSTRate,
		IBSRate:           p.IBSRate,
		CBSRate:           p.CBSRate,
		SelectiveTaxCode:  p.SelectiveTaxCode,
		SelectiveTaxRate:  p.SelectiveTaxRate,

		OperationCode:     p.OperationCode,
		EmitterUF:         p.EmitterUF,
		RecipientUF:       p.RecipientUF,
		OperationNature:   p.OperationNature,
		TargetTaxRegime:   p.TargetTaxRegime,
		ObservedTaxRegime: p.ObservedTaxRegime,
		TargetCRT:         p.TargetCRT,
		ObservedCRT:       p.ObservedCRT,
		ConfidenceScore:   confidence,
		SourceType:        sourceType,
	})
}

func (s *Service) SaveManualProduct(ctx context.Context, p SaveManualProductParams) (string, error) {
	p = normalizeManualProductParams(p)
	productID, err := s.resolveProductID(ctx, p.ProductID, p.ProductCode, p.GTIN, p.Description)
	if err != nil {
		return "", err
	}

	if err := s.repo.CreateTaxProfile(ctx, CreateTaxProfileParams{
		ProductID:       productID,
		OrganizationID:  p.OrganizationID,
		SourceInvoiceID: "",

		NCM:               p.NCM,
		NCMEx:             p.NCMEx,
		CEST:              p.CEST,
		CFOP:              p.CFOP,
		CClasTrib:         p.CClasTrib,
		PISCST:            p.PISCST,
		COFINSCST:         p.COFINSCST,
		PISRevenueCode:    p.PISRevenueCode,
		COFINSRevenueCode: p.COFINSRevenueCode,
		ICMSCST:           p.ICMSCST,
		CSOSN:             p.CSOSN,
		CBenef:            p.CBenef,

		ICMSValue:         p.ICMSValue,
		IPIValue:          p.IPIValue,
		PISValue:          p.PISValue,
		COFINSValue:       p.COFINSValue,
		PISRate:           p.PISRate,
		COFINSRate:        p.COFINSRate,
		ICMSRate:          p.ICMSRate,
		ICMSBaseReduction: p.ICMSBaseReduction,
		FCPRate:           p.FCPRate,
		ICMSSTRate:        p.ICMSSTRate,
		IBSRate:           p.IBSRate,
		CBSRate:           p.CBSRate,
		SelectiveTaxCode:  p.SelectiveTaxCode,
		SelectiveTaxRate:  p.SelectiveTaxRate,

		OperationCode:     p.OperationCode,
		EmitterUF:         p.EmitterUF,
		RecipientUF:       p.RecipientUF,
		OperationNature:   p.OperationNature,
		TargetTaxRegime:   p.TargetTaxRegime,
		ObservedTaxRegime: p.ObservedTaxRegime,
		TargetCRT:         p.TargetCRT,
		ObservedCRT:       p.ObservedCRT,
		ConfidenceScore:   0.99,
		SourceType:        "manual_entry",
	}); err != nil {
		return "", err
	}

	return productID, nil
}

func (s *Service) SaveReviewedProduct(ctx context.Context, p SaveReviewedProductParams) error {
	normalized := normalizeManualProductParams(p.SaveManualProductParams)
	productID, err := s.resolveProductID(ctx, normalized.ProductID, normalized.ProductCode, normalized.GTIN, normalized.Description)
	if err != nil {
		return err
	}

	confidence := p.ConfidenceScore
	if confidence <= 0 {
		confidence = 0.80
	}
	sourceType := strings.TrimSpace(p.SourceType)
	if sourceType == "" {
		sourceType = "auto_review_accepted"
	}

	return s.repo.CreateTaxProfile(ctx, CreateTaxProfileParams{
		ProductID:       productID,
		OrganizationID:  normalized.OrganizationID,
		SourceInvoiceID: "",

		NCM:               normalized.NCM,
		NCMEx:             normalized.NCMEx,
		CEST:              normalized.CEST,
		CFOP:              normalized.CFOP,
		CClasTrib:         normalized.CClasTrib,
		PISCST:            normalized.PISCST,
		COFINSCST:         normalized.COFINSCST,
		PISRevenueCode:    normalized.PISRevenueCode,
		COFINSRevenueCode: normalized.COFINSRevenueCode,
		ICMSCST:           normalized.ICMSCST,
		CSOSN:             normalized.CSOSN,
		CBenef:            normalized.CBenef,

		ICMSValue:         normalized.ICMSValue,
		IPIValue:          normalized.IPIValue,
		PISValue:          normalized.PISValue,
		COFINSValue:       normalized.COFINSValue,
		PISRate:           normalized.PISRate,
		COFINSRate:        normalized.COFINSRate,
		ICMSRate:          normalized.ICMSRate,
		ICMSBaseReduction: normalized.ICMSBaseReduction,
		FCPRate:           normalized.FCPRate,
		ICMSSTRate:        normalized.ICMSSTRate,
		IBSRate:           normalized.IBSRate,
		CBSRate:           normalized.CBSRate,
		SelectiveTaxCode:  normalized.SelectiveTaxCode,
		SelectiveTaxRate:  normalized.SelectiveTaxRate,

		OperationCode:     normalized.OperationCode,
		EmitterUF:         normalized.EmitterUF,
		RecipientUF:       normalized.RecipientUF,
		OperationNature:   normalized.OperationNature,
		TargetTaxRegime:   normalized.TargetTaxRegime,
		ObservedTaxRegime: normalized.ObservedTaxRegime,
		TargetCRT:         normalized.TargetCRT,
		ObservedCRT:       normalized.ObservedCRT,
		ConfidenceScore:   confidence,
		SourceType:        sourceType,
	})
}

func normalizeManualProductParams(p SaveManualProductParams) SaveManualProductParams {
	if strings.TrimSpace(p.OperationCode) == "" {
		p.OperationCode = "sale_consumer_final"
	}

	p.ICMSValue = normalizeDecimalInput(p.ICMSValue)
	p.IPIValue = normalizeDecimalInput(p.IPIValue)
	p.PISValue = normalizeDecimalInput(p.PISValue)
	p.COFINSValue = normalizeDecimalInput(p.COFINSValue)
	p.PISRate = normalizeDecimalInput(p.PISRate)
	p.COFINSRate = normalizeDecimalInput(p.COFINSRate)
	p.ICMSRate = normalizeDecimalInput(p.ICMSRate)
	p.ICMSBaseReduction = normalizeDecimalInput(p.ICMSBaseReduction)
	p.FCPRate = normalizeDecimalInput(p.FCPRate)
	p.ICMSSTRate = normalizeDecimalInput(p.ICMSSTRate)
	p.IBSRate = normalizeDecimalInput(p.IBSRate)
	p.CBSRate = normalizeDecimalInput(p.CBSRate)
	p.SelectiveTaxRate = normalizeDecimalInput(p.SelectiveTaxRate)
	return p
}

func normalizeDecimalInput(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "%", "")
	value = strings.ReplaceAll(value, " ", "")

	if strings.Contains(value, ",") && strings.Contains(value, ".") {
		value = strings.ReplaceAll(value, ".", "")
	}
	value = strings.ReplaceAll(value, ",", ".")
	return value
}

type CatalogProductPage struct {
	Items   []CatalogProductView `json:"items"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
	HasMore bool                 `json:"has_more"`
}

func (s *Service) ListCatalogProducts(ctx context.Context, organizationID, query string, limit, offset int) (CatalogProductPage, error) {
	if limit <= 0 {
		limit = 24
	}
	if limit > 80 {
		limit = 80
	}
	if offset < 0 {
		offset = 0
	}

	items, err := s.repo.ListCatalogProducts(ctx, organizationID, query, limit+1, offset)
	if err != nil {
		return CatalogProductPage{}, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	return CatalogProductPage{
		Items:   items,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

func (s *Service) resolveProductID(ctx context.Context, productID, productCode, gtin, description string) (string, error) {
	productID = strings.TrimSpace(productID)
	if productID != "" {
		normalizedGTIN := NormalizeGTIN(gtin)
		normalizedDescription := NormalizeDescription(description)
		if normalizedDescription == "" && normalizedGTIN == "" {
			return "", errors.New("cannot save product without gtin or description")
		}

		if err := s.repo.UpdateProduct(ctx, productID, strings.TrimSpace(productCode), strings.TrimSpace(gtin), normalizedGTIN, strings.TrimSpace(description), normalizedDescription); err != nil {
			return "", err
		}

		return productID, nil
	}

	normalizedGTIN := NormalizeGTIN(gtin)
	normalizedDescription := NormalizeDescription(description)

	if normalizedDescription == "" && normalizedGTIN == "" {
		return "", errors.New("cannot save product without gtin or description")
	}

	if normalizedGTIN != "" {
		product, err := s.repo.FindProductByNormalizedGTIN(ctx, normalizedGTIN)
		if err == nil && product != nil {
			if err := s.repo.UpdateProduct(ctx, product.ID, strings.TrimSpace(productCode), strings.TrimSpace(gtin), normalizedGTIN, strings.TrimSpace(description), normalizedDescription); err != nil {
				return "", err
			}
			return product.ID, nil
		}
	}

	if normalizedDescription != "" {
		product, err := s.repo.FindProductByNormalizedDescription(ctx, normalizedDescription)
		if err == nil && product != nil {
			if err := s.repo.UpdateProduct(ctx, product.ID, strings.TrimSpace(productCode), strings.TrimSpace(gtin), normalizedGTIN, strings.TrimSpace(description), normalizedDescription); err != nil {
				return "", err
			}
			return product.ID, nil
		}
	}

	id, err := s.repo.CreateProduct(
		ctx,
		strings.TrimSpace(productCode),
		strings.TrimSpace(gtin),
		normalizedGTIN,
		strings.TrimSpace(description),
		normalizedDescription,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}
