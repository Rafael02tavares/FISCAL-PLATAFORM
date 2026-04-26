package admincapture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/catalog"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
	"github.com/rafa/fiscal-platform/backend/internal/tax"
)

type Service struct {
	repo              *Repository
	catalogService    *catalog.Service
	legalBasisService *legalbasis.Service
	taxService        *tax.Service
}

func NewService(repo *Repository, catalogService *catalog.Service, legalBasisService *legalbasis.Service, taxService *tax.Service) *Service {
	return &Service{
		repo:              repo,
		catalogService:    catalogService,
		legalBasisService: legalBasisService,
		taxService:        taxService,
	}
}

type ProductReview struct {
	ProductID       string                    `json:"product_id"`
	ProductCode     string                    `json:"product_code"`
	GTIN            string                    `json:"gtin"`
	Description     string                    `json:"description"`
	CurrentProfile  catalog.ProductTaxProfile `json:"current_profile"`
	Suggestion      tax.Suggestion            `json:"suggestion"`
	ConfidenceScore float64                   `json:"confidence_score"`
	Status          string                    `json:"status"`
	CanAccept       bool                      `json:"can_accept"`
	Warnings        []string                  `json:"warnings"`
}

func (s *Service) ListCandidates(ctx context.Context, organizationID string, limit int) ([]Candidate, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, errors.New("organization_id is required")
	}
	if limit <= 0 || limit > 300 {
		limit = 120
	}

	return s.repo.ListCandidates(ctx, organizationID, limit)
}

func (s *Service) AcceptCandidate(ctx context.Context, organizationID, invoiceItemID, targetTaxRegime, targetCRT string) error {
	organizationID = strings.TrimSpace(organizationID)
	invoiceItemID = strings.TrimSpace(invoiceItemID)

	if organizationID == "" {
		return errors.New("organization_id is required")
	}
	if invoiceItemID == "" {
		return errors.New("invoice_item_id is required")
	}

	item, err := s.repo.GetCandidateByInvoiceItemID(ctx, organizationID, invoiceItemID)
	if err != nil {
		return fmt.Errorf("get candidate: %w", err)
	}

	if !item.HasObservedTaxes {
		return errors.New("o item da nota nao possui informacoes tributarias suficientes para captura")
	}

	operationCode := inferOperationCodeFromCFOP(item.CFOP)

	err = s.catalogService.RegisterObservedItem(ctx, catalog.RegisterObservedItemParams{
		OrganizationID:    organizationID,
		SourceInvoiceID:   item.InvoiceID,
		ProductCode:       item.ProductCode,
		GTIN:              item.GTIN,
		Description:       item.Description,
		NCM:               item.NCM,
		CEST:              item.CEST,
		CFOP:              item.CFOP,
		CClasTrib:         "",
		PISCST:            item.PISCST,
		COFINSCST:         item.COFINSCST,
		ICMSCST:           item.ICMSCST,
		CSOSN:             item.CSOSN,
		CBenef:            "",
		ICMSValue:         item.ICMSValue,
		IPIValue:          item.IPIValue,
		PISValue:          item.PISValue,
		COFINSValue:       item.COFINSValue,
		PISRate:           item.PISRate,
		COFINSRate:        item.COFINSRate,
		ICMSRate:          item.ICMSRate,
		IBSRate:           "",
		CBSRate:           "",
		OperationCode:     operationCode,
		EmitterUF:         strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
		RecipientUF:       strings.ToUpper(strings.TrimSpace(item.RecipientUF)),
		OperationNature:   item.OperationNature,
		TargetTaxRegime:   strings.TrimSpace(targetTaxRegime),
		ObservedTaxRegime: strings.TrimSpace(item.ObservedTaxRegime),
		TargetCRT:         strings.TrimSpace(targetCRT),
		ObservedCRT:       strings.TrimSpace(item.ObservedCRT),
		SourceType:        "invoice_import_entry",
	})
	if err != nil {
		return fmt.Errorf("integrate observed rule into catalog: %w", err)
	}

	if s.legalBasisService == nil {
		return nil
	}

	if err := s.integrateIntoLegalRules(ctx, item, strings.TrimSpace(targetTaxRegime), operationCode); err != nil {
		return fmt.Errorf("integrate observed rule into legal rules: %w", err)
	}

	return nil
}

func (s *Service) ReviewCatalogProducts(ctx context.Context, organizationID, taxRegime, crt, homeUF string, limit int) ([]ProductReview, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, errors.New("organization_id is required")
	}
	if s.taxService == nil {
		return nil, errors.New("tax service is required")
	}
	if limit <= 0 || limit > 300 {
		limit = 150
	}

	page, err := s.catalogService.ListCatalogProducts(ctx, organizationID, "", limit, 0)
	if err != nil {
		return nil, fmt.Errorf("list catalog products: %w", err)
	}
	products := page.Items

	reviews := make([]ProductReview, 0, len(products))
	for _, product := range products {
		resp, err := s.taxService.Suggest(ctx, buildProductReviewSuggestRequest(organizationID, product, taxRegime, crt, homeUF))
		if err != nil {
			reviews = append(reviews, ProductReview{
				ProductID:       product.ID,
				ProductCode:     product.ProductCode,
				GTIN:            product.GTIN,
				Description:     product.Description,
				CurrentProfile:  product.Profile,
				Status:          "error",
				CanAccept:       false,
				Warnings:        []string{err.Error()},
				ConfidenceScore: 0,
			})
			continue
		}

		status := "review"
		canAccept := resp.ConfidenceScore >= 0.70 && !isEmptySuggestion(resp.Suggestion)
		if resp.ConfidenceScore >= 0.90 {
			status = "ready"
		} else if resp.ConfidenceScore < 0.70 {
			status = "low_confidence"
		}

		reviews = append(reviews, ProductReview{
			ProductID:       product.ID,
			ProductCode:     product.ProductCode,
			GTIN:            product.GTIN,
			Description:     product.Description,
			CurrentProfile:  product.Profile,
			Suggestion:      resp.Suggestion,
			ConfidenceScore: resp.ConfidenceScore,
			Status:          status,
			CanAccept:       canAccept,
			Warnings:        resp.Warnings,
		})
	}

	return reviews, nil
}

func (s *Service) AcceptProductReviews(ctx context.Context, organizationID string, productIDs []string, acceptAll bool, minConfidence float64, taxRegime, crt, homeUF string) (int, []string, error) {
	if minConfidence <= 0 {
		minConfidence = 0.70
	}

	reviews, err := s.ReviewCatalogProducts(ctx, organizationID, taxRegime, crt, homeUF, 300)
	if err != nil {
		return 0, nil, err
	}

	allowed := map[string]bool{}
	for _, id := range productIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			allowed[id] = true
		}
	}

	accepted := 0
	failures := []string{}
	for _, review := range reviews {
		if !acceptAll && !allowed[review.ProductID] {
			continue
		}
		if review.ConfidenceScore < minConfidence || !review.CanAccept {
			continue
		}

		if err := s.acceptProductReview(ctx, organizationID, review, taxRegime, crt, homeUF); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", review.Description, err))
			continue
		}
		accepted++
	}

	return accepted, failures, nil
}

func (s *Service) acceptProductReview(ctx context.Context, organizationID string, review ProductReview, taxRegime, crt, homeUF string) error {
	suggestion := review.Suggestion
	return s.catalogService.SaveReviewedProduct(ctx, catalog.SaveReviewedProductParams{
		SaveManualProductParams: catalog.SaveManualProductParams{
			OrganizationID: organizationID,
			ProductID:      review.ProductID,
			ProductCode:    review.ProductCode,
			GTIN:           review.GTIN,
			Description:    review.Description,

			NCM:               suggestion.NCM,
			NCMEx:             suggestion.NCMEx,
			CEST:              suggestion.CEST,
			CFOP:              suggestion.CFOP,
			CClasTrib:         suggestion.CClasTrib,
			PISCST:            suggestion.PISCST,
			COFINSCST:         suggestion.COFINSCST,
			PISRevenueCode:    suggestion.PISRevenueCode,
			COFINSRevenueCode: suggestion.COFINSRevenueCode,
			ICMSCST:           suggestion.ICMSCST,
			CSOSN:             suggestion.CSOSN,
			CBenef:            suggestion.CBenef,

			ICMSValue:         suggestion.ICMSValue,
			IPIValue:          suggestion.IPIValue,
			PISValue:          suggestion.PISValue,
			COFINSValue:       suggestion.COFINSValue,
			PISRate:           suggestion.PISRate,
			COFINSRate:        suggestion.COFINSRate,
			ICMSRate:          suggestion.ICMSRate,
			ICMSBaseReduction: suggestion.ICMSBaseReduction,
			FCPRate:           suggestion.FCPRate,
			ICMSSTRate:        suggestion.ICMSSTRate,
			IBSRate:           suggestion.IBSRate,
			CBSRate:           suggestion.CBSRate,
			SelectiveTaxCode:  suggestion.SelectiveTaxCode,
			SelectiveTaxRate:  suggestion.SelectiveTaxRate,

			OperationCode:   "sale_consumer_final",
			EmitterUF:       strings.ToUpper(strings.TrimSpace(firstNonEmpty(review.CurrentProfile.EmitterUF, homeUF))),
			RecipientUF:     strings.ToUpper(strings.TrimSpace(firstNonEmpty(review.CurrentProfile.RecipientUF, homeUF))),
			OperationNature: "Venda interna para consumidor final",
			TargetTaxRegime: strings.TrimSpace(taxRegime),
			TargetCRT:       strings.TrimSpace(crt),
		},
		ConfidenceScore: review.ConfidenceScore,
		SourceType:      "auto_review_accepted",
	})
}

func buildProductReviewSuggestRequest(organizationID string, product catalog.CatalogProductView, taxRegime, crt, homeUF string) tax.SuggestRequest {
	profile := product.Profile
	uf := strings.ToUpper(strings.TrimSpace(firstNonEmpty(profile.EmitterUF, profile.RecipientUF, homeUF)))
	return tax.SuggestRequest{
		OrganizationID:   organizationID,
		GTIN:             strings.TrimSpace(product.GTIN),
		Description:      strings.TrimSpace(product.Description),
		NCMCode:          strings.TrimSpace(profile.NCM),
		OperationCode:    "sale_consumer_final",
		EmitterUF:        uf,
		RecipientUF:      uf,
		TaxRegime:        strings.TrimSpace(firstNonEmpty(profile.TargetTaxRegime, taxRegime)),
		TargetCRT:        strings.TrimSpace(firstNonEmpty(profile.TargetCRT, crt)),
		SourceICMSCST:    strings.TrimSpace(profile.ICMSCST),
		SourceICMSCSOSN:  strings.TrimSpace(profile.CSOSN),
		SourceICMSRate:   strings.TrimSpace(profile.ICMSRate),
		SourcePISCST:     strings.TrimSpace(profile.PISCST),
		SourcePISRate:    strings.TrimSpace(profile.PISRate),
		SourceCOFINSCST:  strings.TrimSpace(profile.COFINSCST),
		SourceCOFINSRate: strings.TrimSpace(profile.COFINSRate),
		SourceCFOP:       strings.TrimSpace(profile.CFOP),
	}
}

func isEmptySuggestion(s tax.Suggestion) bool {
	return strings.TrimSpace(s.NCM) == "" &&
		strings.TrimSpace(s.CEST) == "" &&
		strings.TrimSpace(s.CFOP) == "" &&
		strings.TrimSpace(s.ICMSCST) == "" &&
		strings.TrimSpace(s.CSOSN) == "" &&
		strings.TrimSpace(s.PISCST) == "" &&
		strings.TrimSpace(s.COFINSCST) == ""
}

func (s *Service) integrateIntoLegalRules(ctx context.Context, item *Candidate, targetTaxRegime string, operationCode string) error {
	title := fmt.Sprintf("Regra observada NF %s serie %s item %d", item.InvoiceNumber, item.InvoiceSeries, item.ItemNumber)
	description := fmt.Sprintf(
		"Regra capturada automaticamente da nota fiscal para o produto %s. Emitente %s -> destino %s.",
		strings.TrimSpace(item.Description),
		strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
		strings.ToUpper(strings.TrimSpace(item.RecipientUF)),
	)

	legalSourceID, err := s.legalBasisService.CreateLegalSource(ctx, legalbasis.CreateLegalSourceParams{
		TaxType:       "observed_rule",
		SourceType:    "invoice_capture",
		Jurisdiction:  "BR",
		UF:            strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
		Title:         title,
		ReferenceCode: fmt.Sprintf("NF-%s-%s-ITEM-%d", item.InvoiceNumber, item.InvoiceSeries, item.ItemNumber),
		Description:   description,
		Notes:         "Fonte automatica criada a partir da triagem de regras capturadas.",
	})
	if err != nil {
		return fmt.Errorf("create legal source: %w", err)
	}

	mappings := []legalbasis.CreateLegalRuleMappingParams{}

	if item.NCM != "" || item.CEST != "" || item.CFOP != "" {
		valueContent, err := marshalValueContent(map[string]string{
			"ncm":  strings.TrimSpace(item.NCM),
			"cest": strings.TrimSpace(item.CEST),
			"cfop": strings.TrimSpace(item.CFOP),
		})
		if err != nil {
			return err
		}

		mappings = append(mappings, legalbasis.CreateLegalRuleMappingParams{
			LegalSourceID:  legalSourceID,
			OperationCode:  operationCode,
			TaxType:        "ncm",
			TaxRegime:      targetTaxRegime,
			NCMCode:        strings.TrimSpace(item.NCM),
			CEST:           strings.TrimSpace(item.CEST),
			CFOP:           strings.TrimSpace(item.CFOP),
			EmitterUF:      strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
			RecipientUF:    strings.ToUpper(strings.TrimSpace(item.RecipientUF)),
			ValueType:      "classification_rule",
			ValueContent:   valueContent,
			Priority:       40,
			ConfidenceBase: "0.95",
		})
	}

	if item.ICMSCST != "" || item.CSOSN != "" || item.ICMSRate != "" || item.CFOP != "" {
		valueContent, err := marshalValueContent(map[string]string{
			"icms_cst":  strings.TrimSpace(item.ICMSCST),
			"csosn":     strings.TrimSpace(item.CSOSN),
			"icms_rate": strings.TrimSpace(item.ICMSRate),
			"cfop":      strings.TrimSpace(item.CFOP),
		})
		if err != nil {
			return err
		}

		mappings = append(mappings, legalbasis.CreateLegalRuleMappingParams{
			LegalSourceID:  legalSourceID,
			OperationCode:  operationCode,
			TaxType:        "icms",
			TaxRegime:      targetTaxRegime,
			NCMCode:        strings.TrimSpace(item.NCM),
			CEST:           strings.TrimSpace(item.CEST),
			CFOP:           strings.TrimSpace(item.CFOP),
			ICMSCST:        strings.TrimSpace(item.ICMSCST),
			CSOSN:          strings.TrimSpace(item.CSOSN),
			EmitterUF:      strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
			RecipientUF:    strings.ToUpper(strings.TrimSpace(item.RecipientUF)),
			ValueType:      "cst_rule",
			ValueContent:   valueContent,
			Priority:       50,
			ConfidenceBase: "0.95",
		})
	}

	if item.PISCST != "" || item.PISRate != "" {
		valueContent, err := marshalValueContent(map[string]string{
			"pis_cst":  strings.TrimSpace(item.PISCST),
			"pis_rate": strings.TrimSpace(item.PISRate),
		})
		if err != nil {
			return err
		}

		mappings = append(mappings, legalbasis.CreateLegalRuleMappingParams{
			LegalSourceID:  legalSourceID,
			OperationCode:  operationCode,
			TaxType:        "pis",
			TaxRegime:      targetTaxRegime,
			NCMCode:        strings.TrimSpace(item.NCM),
			CEST:           strings.TrimSpace(item.CEST),
			CFOP:           strings.TrimSpace(item.CFOP),
			PISCST:         strings.TrimSpace(item.PISCST),
			EmitterUF:      strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
			RecipientUF:    strings.ToUpper(strings.TrimSpace(item.RecipientUF)),
			ValueType:      "rate_rule",
			ValueContent:   valueContent,
			Priority:       60,
			ConfidenceBase: "0.95",
		})
	}

	if item.COFINSCST != "" || item.COFINSRate != "" {
		valueContent, err := marshalValueContent(map[string]string{
			"cofins_cst":  strings.TrimSpace(item.COFINSCST),
			"cofins_rate": strings.TrimSpace(item.COFINSRate),
		})
		if err != nil {
			return err
		}

		mappings = append(mappings, legalbasis.CreateLegalRuleMappingParams{
			LegalSourceID:  legalSourceID,
			OperationCode:  operationCode,
			TaxType:        "cofins",
			TaxRegime:      targetTaxRegime,
			NCMCode:        strings.TrimSpace(item.NCM),
			CEST:           strings.TrimSpace(item.CEST),
			CFOP:           strings.TrimSpace(item.CFOP),
			COFINSCST:      strings.TrimSpace(item.COFINSCST),
			EmitterUF:      strings.ToUpper(strings.TrimSpace(item.EmitterUF)),
			RecipientUF:    strings.ToUpper(strings.TrimSpace(item.RecipientUF)),
			ValueType:      "rate_rule",
			ValueContent:   valueContent,
			Priority:       70,
			ConfidenceBase: "0.95",
		})
	}

	for _, mapping := range mappings {
		if _, err := s.legalBasisService.CreateLegalRuleMapping(ctx, mapping); err != nil {
			return fmt.Errorf("create legal rule mapping: %w", err)
		}
	}

	return nil
}

func inferOperationCodeFromCFOP(cfop string) string {
	switch onlyDigits(cfop) {
	case "5403", "5405":
		return "sale_st_internal"
	case "6403", "6404":
		return "sale_st_interstate"
	case "5101", "5102":
		return "sale_consumer_final"
	case "6101", "6102":
		return "sale_interstate"
	default:
		return ""
	}
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func marshalValueContent(payload map[string]string) (string, error) {
	normalized := make(map[string]string)
	for key, value := range payload {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized[key] = value
	}

	if len(normalized) == 0 {
		return "{}", nil
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal value content: %w", err)
	}

	return string(raw), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
