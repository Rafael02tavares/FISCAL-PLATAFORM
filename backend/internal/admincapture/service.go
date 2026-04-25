package admincapture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/catalog"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
)

type Service struct {
	repo              *Repository
	catalogService    *catalog.Service
	legalBasisService *legalbasis.Service
}

func NewService(repo *Repository, catalogService *catalog.Service, legalBasisService *legalbasis.Service) *Service {
	return &Service{
		repo:              repo,
		catalogService:    catalogService,
		legalBasisService: legalBasisService,
	}
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
