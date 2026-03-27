package classifier

import (
	"context"
	"errors"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Service struct {
	memoryRepo domain.ClassificationMemoryRepository
}

func NewService(memoryRepo domain.ClassificationMemoryRepository) (*Service, error) {
	if memoryRepo == nil {
		return nil, errors.New("classifier: classification memory repository is required")
	}

	return &Service{
		memoryRepo: memoryRepo,
	}, nil
}

func (s *Service) Resolve(ctx context.Context, normalized domain.NormalizedContext) (*domain.ClassificationDecision, error) {
	if err := validateNormalizedContext(normalized); err != nil {
		return nil, err
	}

	if decision, err := s.resolveByGTIN(ctx, normalized); err != nil {
		return nil, err
	} else if decision != nil {
		return decision, nil
	}

	if decision, err := s.resolveBySupplierProduct(ctx, normalized); err != nil {
		return nil, err
	} else if decision != nil {
		return decision, nil
	}

	if decision, err := s.resolveByDescription(ctx, normalized); err != nil {
		return nil, err
	} else if decision != nil {
		return decision, nil
	}

	if decision := s.resolveByXMLFallback(normalized); decision != nil {
		return decision, nil
	}

	return s.resolveHardFallback(normalized), nil
}

func validateNormalizedContext(normalized domain.NormalizedContext) error {
	if strings.TrimSpace(normalized.TenantID) == "" {
		return errors.New("classifier: tenant_id is required")
	}
	if strings.TrimSpace(normalized.OrganizationID) == "" {
		return errors.New("classifier: organization_id is required")
	}
	if strings.TrimSpace(normalized.ProductDescriptionNorm) == "" {
		return errors.New("classifier: normalized product description is required")
	}

	return nil
}

func (s *Service) resolveByGTIN(ctx context.Context, normalized domain.NormalizedContext) (*domain.ClassificationDecision, error) {
	if normalized.GTIN == "" {
		return nil, nil
	}

	entry, err := s.memoryRepo.FindByGTIN(
		ctx,
		normalized.TenantID,
		normalized.OrganizationID,
		normalized.GTIN,
	)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	if !isUsableMemoryEntry(*entry) {
		return nil, nil
	}

	confidence := clampConfidence(maxFloat(entry.Confidence, 0.96))

	reasons := []string{
		"classificação encontrada por GTIN",
		"GTIN já classificado anteriormente",
	}

	if normalized.NCM != "" && entry.NCM != normalized.NCM {
		reasons = append(reasons, "NCM do XML diverge da memória por GTIN")
		return &domain.ClassificationDecision{
			NCM:         entry.NCM,
			EXTIPI:      cloneOptionalString(entry.EXTIPI),
			CEST:        cloneOptionalString(entry.CEST),
			Confidence:  minFloat(confidence, 0.90),
			Source:      domain.ClassificationSourceGTINMemory,
			NeedsReview: true,
			Reasons:     reasons,
		}, nil
	}

	return &domain.ClassificationDecision{
		NCM:         entry.NCM,
		EXTIPI:      cloneOptionalString(entry.EXTIPI),
		CEST:        cloneOptionalString(entry.CEST),
		Confidence:  confidence,
		Source:      domain.ClassificationSourceGTINMemory,
		NeedsReview: false,
		Reasons:     reasons,
	}, nil
}

func (s *Service) resolveBySupplierProduct(ctx context.Context, normalized domain.NormalizedContext) (*domain.ClassificationDecision, error) {
	if normalized.SupplierID == "" || normalized.SupplierProductCode == "" {
		return nil, nil
	}

	entry, err := s.memoryRepo.FindBySupplierProduct(
		ctx,
		normalized.TenantID,
		normalized.OrganizationID,
		normalized.SupplierID,
		normalized.SupplierProductCode,
	)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	if !isUsableMemoryEntry(*entry) {
		return nil, nil
	}

	confidence := clampConfidence(maxFloat(entry.Confidence, 0.93))
	needsReview := false
	reasons := []string{
		"classificação encontrada por fornecedor e código do produto",
	}

	if normalized.NCM != "" && entry.NCM != normalized.NCM {
		needsReview = true
		confidence = minFloat(confidence, 0.88)
		reasons = append(reasons, "NCM do XML diverge da memória do fornecedor")
	}

	return &domain.ClassificationDecision{
		NCM:         entry.NCM,
		EXTIPI:      cloneOptionalString(entry.EXTIPI),
		CEST:        cloneOptionalString(entry.CEST),
		Confidence:  confidence,
		Source:      domain.ClassificationSourceSupplierMemory,
		NeedsReview: needsReview,
		Reasons:     reasons,
	}, nil
}

func (s *Service) resolveByDescription(ctx context.Context, normalized domain.NormalizedContext) (*domain.ClassificationDecision, error) {
	if normalized.ProductDescriptionNorm == "" {
		return nil, nil
	}

	entry, err := s.memoryRepo.FindBestDescriptionMatch(
		ctx,
		normalized.TenantID,
		normalized.OrganizationID,
		normalized.ProductDescriptionNorm,
	)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	if !isUsableMemoryEntry(*entry) {
		return nil, nil
	}

	confidence := clampConfidence(entry.Confidence)
	if confidence == 0 {
		confidence = 0.82
	}
	confidence = minFloat(confidence, 0.89)

	needsReview := true
	reasons := []string{
		"classificação sugerida por similaridade de descrição",
		"exige conferência manual por se basear em descrição",
	}

	if normalized.NCM != "" && entry.NCM == normalized.NCM {
		confidence = maxFloat(confidence, 0.86)
		reasons = append(reasons, "NCM do XML confirma a memória por descrição")
	}

	return &domain.ClassificationDecision{
		NCM:         entry.NCM,
		EXTIPI:      cloneOptionalString(entry.EXTIPI),
		CEST:        cloneOptionalString(entry.CEST),
		Confidence:  confidence,
		Source:      domain.ClassificationSourceDescriptionMatch,
		NeedsReview: needsReview,
		Reasons:     reasons,
	}, nil
}

func (s *Service) resolveByXMLFallback(normalized domain.NormalizedContext) *domain.ClassificationDecision {
	if normalized.NCM == "" {
		return nil
	}

	reasons := []string{
		"classificação obtida do XML de origem",
	}

	confidence := 0.78
	needsReview := false

	if normalized.CEST == "" {
		reasons = append(reasons, "CEST ausente no XML")
		confidence = 0.74
	}

	if normalized.GTIN == "" {
		reasons = append(reasons, "GTIN ausente; sem confirmação por memória")
		confidence = minFloat(confidence, 0.72)
	}

	if normalized.EXTIPI == "" {
		reasons = append(reasons, "EX TIPI não informado no XML")
	}

	if normalized.GTIN == "" && normalized.CEST == "" {
		needsReview = true
		confidence = minFloat(confidence, 0.68)
		reasons = append(reasons, "dados auxiliares insuficientes para alta confiança")
	}

	var extipi *string
	if normalized.EXTIPI != "" {
		extipi = stringPtr(normalized.EXTIPI)
	}

	var cest *string
	if normalized.CEST != "" {
		cest = stringPtr(normalized.CEST)
	}

	return &domain.ClassificationDecision{
		NCM:         normalized.NCM,
		EXTIPI:      extipi,
		CEST:        cest,
		Confidence:  confidence,
		Source:      domain.ClassificationSourceXML,
		NeedsReview: needsReview,
		Reasons:     reasons,
	}
}

func (s *Service) resolveHardFallback(normalized domain.NormalizedContext) *domain.ClassificationDecision {
	reasons := []string{
		"nenhuma memória de classificação encontrada",
		"XML sem NCM válido",
		"revisão manual obrigatória",
	}

	return &domain.ClassificationDecision{
		NCM:         "",
		EXTIPI:      nil,
		CEST:        nil,
		Confidence:  0.10,
		Source:      domain.ClassificationSourceFallback,
		NeedsReview: true,
		Reasons:     reasons,
	}
}

func isUsableMemoryEntry(entry domain.ClassificationMemoryEntry) bool {
	return strings.TrimSpace(entry.NCM) != ""
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func stringPtr(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}