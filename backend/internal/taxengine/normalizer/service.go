package normalizer

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

var nonDigitRegex = regexp.MustCompile(`\D+`)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Normalize(ctx context.Context, input domain.EvaluateInput) (*domain.NormalizedContext, error) {
	_ = ctx

	if err := validateInput(input); err != nil {
		return nil, err
	}

	descriptionNorm := normalizeDescription(input.Product.Description)
	gtin := normalizeGTIN(input.Product.GTIN)
	ncm := normalizeNCM(input.Product.NCM)
	extipi := normalizeEX(input.Product.EXTIPI)
	cest := normalizeCEST(input.Product.CEST)
	cfop := normalizeCFOP(input.Operation.CFOP)
	issuerUF := normalizeUF(input.Issuer.UF)
	recipientUF := normalizeUF(input.Recipient.UF)

	scope := deriveOperationScope(input.Operation.OperationScope, issuerUF, recipientUF, input.Operation.IsImport, input.Operation.IsExport)

	normalized := &domain.NormalizedContext{
		TenantID:               strings.TrimSpace(input.TenantID),
		OrganizationID:         strings.TrimSpace(input.OrganizationID),
		InvoiceID:              input.InvoiceID,
		InvoiceItemID:          input.InvoiceItemID,

		DocumentType:           normalizeDocumentType(input.Operation.DocumentType),
		OperationType:          normalizeOperationType(input.Operation.OperationType),
		OperationScope:         scope,
		CFOP:                   cfop,
		FinNFe:                 strings.TrimSpace(input.Operation.FinNFe),
		PresenceIndicator:      strings.TrimSpace(input.Operation.PresenceIndicator),
		PurposeCode:            strings.TrimSpace(input.Operation.PurposeCode),

		IssuerUF:               issuerUF,
		RecipientUF:            recipientUF,
		IssuerCRT:              strings.TrimSpace(input.Issuer.CRT),
		RecipientContributor:   input.Recipient.IsContributorICMS,
		FinalConsumer:          input.Recipient.IsFinalConsumer,

		ProductDescription:     strings.TrimSpace(input.Product.Description),
		ProductDescriptionNorm: descriptionNorm,
		GTIN:                   gtin,
		NCM:                    ncm,
		NCMChapter:             deriveNCMChapter(ncm),
		NCMPosition:            deriveNCMPosition(ncm),
		EXTIPI:                 extipi,
		CEST:                   cest,
		OriginCode:             strings.TrimSpace(input.Product.OriginCode),

		SupplierID:             strings.TrimSpace(input.Product.SupplierID),
		SupplierProductCode:    strings.TrimSpace(input.Product.SupplierProductCode),

		Quantity:               sanitizeNonNegative(input.Product.Quantity),
		UnitValue:              sanitizeNonNegative(input.Values.UnitValue),
		GrossValue:             sanitizeNonNegative(input.Values.GrossValue),
		DiscountValue:          sanitizeNonNegative(input.Values.DiscountValue),
		FreightValue:           sanitizeNonNegative(input.Values.FreightValue),
		InsuranceValue:         sanitizeNonNegative(input.Values.InsuranceValue),
		OtherExpensesValue:     sanitizeNonNegative(input.Values.OtherExpensesValue),
		TotalValue:             sanitizeNonNegative(input.Values.TotalValue),

		IsReturn:               input.Operation.IsReturn,
		IsTransfer:             input.Operation.IsTransfer,
		IsResale:               input.Operation.IsResale,
		IsImport:               input.Operation.IsImport,
		IsExport:               input.Operation.IsExport,
		HasInterstateDelivery:  input.Operation.HasInterstateDelivery,

		AdditionalTags:         deriveAdditionalTags(input, scope, ncm, cest, gtin),
	}

	return normalized, nil
}

func validateInput(input domain.EvaluateInput) error {
	if strings.TrimSpace(input.TenantID) == "" {
		return errors.New("normalizer: tenant_id is required")
	}

	if strings.TrimSpace(input.OrganizationID) == "" {
		return errors.New("normalizer: organization_id is required")
	}

	if strings.TrimSpace(input.Product.Description) == "" {
		return errors.New("normalizer: product description is required")
	}

	if input.Product.Quantity < 0 {
		return errors.New("normalizer: product quantity cannot be negative")
	}

	if input.Values.TotalValue < 0 {
		return errors.New("normalizer: total value cannot be negative")
	}

	return nil
}

func normalizeDescription(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	value = strings.ToUpper(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, value)

	value = strings.Join(strings.Fields(value), " ")
	return value
}

func normalizeGTIN(value string) string {
	value = onlyDigits(value)

	switch len(value) {
	case 8, 12, 13, 14:
		return value
	default:
		return ""
	}
}

func normalizeNCM(value string) string {
	value = onlyDigits(value)
	if len(value) != 8 {
		return ""
	}
	return value
}

func normalizeEX(value string) string {
	value = onlyDigits(value)
	if value == "" {
		return ""
	}
	if len(value) > 3 {
		return value[:3]
	}
	return value
}

func normalizeCEST(value string) string {
	value = onlyDigits(value)
	if len(value) != 7 {
		return ""
	}
	return value
}

func normalizeCFOP(value string) string {
	value = onlyDigits(value)
	if len(value) != 4 {
		return ""
	}
	return value
}

func normalizeUF(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if len(value) != 2 {
		return ""
	}
	return value
}

func normalizeDocumentType(value domain.DocumentType) domain.DocumentType {
	switch value {
	case domain.DocumentTypeNFe:
		return value
	default:
		return domain.DocumentTypeNFe
	}
}

func normalizeOperationType(value domain.OperationType) domain.OperationType {
	switch value {
	case domain.OperationTypeEntry, domain.OperationTypeExit:
		return value
	default:
		return domain.OperationTypeExit
	}
}

func deriveOperationScope(
	current domain.OperationScope,
	issuerUF string,
	recipientUF string,
	isImport bool,
	isExport bool,
) domain.OperationScope {
	if current == domain.OperationScopeInternational {
		return current
	}

	if isImport || isExport {
		return domain.OperationScopeInternational
	}

	if issuerUF != "" && recipientUF != "" {
		if issuerUF == recipientUF {
			return domain.OperationScopeInternal
		}
		return domain.OperationScopeInterstate
	}

	if current == domain.OperationScopeInternal || current == domain.OperationScopeInterstate {
		return current
	}

	return domain.OperationScopeInternal
}

func deriveNCMChapter(ncm string) string {
	if len(ncm) < 2 {
		return ""
	}
	return ncm[:2]
}

func deriveNCMPosition(ncm string) string {
	if len(ncm) < 4 {
		return ""
	}
	return ncm[:4]
}

func deriveAdditionalTags(
	input domain.EvaluateInput,
	scope domain.OperationScope,
	ncm string,
	cest string,
	gtin string,
) []string {
	tags := make([]string, 0, 10)

	if scope == domain.OperationScopeInternal {
		tags = append(tags, "internal_operation")
	}
	if scope == domain.OperationScopeInterstate {
		tags = append(tags, "interstate_operation")
	}
	if scope == domain.OperationScopeInternational {
		tags = append(tags, "international_operation")
	}

	if input.Operation.IsReturn {
		tags = append(tags, "return")
	}
	if input.Operation.IsTransfer {
		tags = append(tags, "transfer")
	}
	if input.Operation.IsResale {
		tags = append(tags, "resale")
	}
	if input.Recipient.IsFinalConsumer {
		tags = append(tags, "final_consumer")
	}
	if input.Recipient.IsContributorICMS {
		tags = append(tags, "recipient_contributor")
	}
	if ncm != "" {
		tags = append(tags, fmt.Sprintf("ncm_chapter:%s", deriveNCMChapter(ncm)))
	}
	if cest != "" {
		tags = append(tags, "has_cest")
	}
	if gtin != "" {
		tags = append(tags, "has_gtin")
	}

	return deduplicate(tags)
}

func sanitizeNonNegative(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func onlyDigits(value string) string {
	return nonDigitRegex.ReplaceAllString(value, "")
}

func deduplicate(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}