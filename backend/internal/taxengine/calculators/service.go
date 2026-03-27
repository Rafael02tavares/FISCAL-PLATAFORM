package calculators

import (
	"context"
	"math"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Calculate(
	ctx context.Context,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
	evaluation domain.RuleEvaluationResult,
) (*domain.TaxDecisionSet, error) {
	_ = ctx

	if classification.NCM == "" {
		return &domain.TaxDecisionSet{}, nil
	}

	taxes := &domain.TaxDecisionSet{}

	if evaluation.Decisions.ICMS != nil {
		taxes.ICMS = calculateICMS(normalized, evaluation.Decisions.ICMS)
	}

	if evaluation.Decisions.ICMSST != nil {
		taxes.ICMSST = calculateICMSST(normalized, evaluation.Decisions.ICMSST)
	}

	if evaluation.Decisions.FCP != nil {
		taxes.FCP = calculateFCP(normalized, evaluation.Decisions.FCP)
	}

	if evaluation.Decisions.DIFAL != nil {
		taxes.DIFAL = calculateDIFAL(normalized, evaluation.Decisions.DIFAL)
	}

	if evaluation.Decisions.PIS != nil {
		taxes.PIS = calculatePIS(normalized, evaluation.Decisions.PIS)
	}

	if evaluation.Decisions.COFINS != nil {
		taxes.COFINS = calculateCOFINS(normalized, evaluation.Decisions.COFINS)
	}

	if evaluation.Decisions.IPI != nil {
		taxes.IPI = calculateIPI(normalized, evaluation.Decisions.IPI)
	}

	return taxes, nil
}

func calculateICMS(normalized domain.NormalizedContext, base *domain.ICMSDecision) *domain.ICMSDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.Amount = 0
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = taxableBase(normalized)
	}

	if decision.ReducedBaseRate != nil && *decision.ReducedBaseRate > 0 {
		decision.BaseValue = round2(decision.BaseValue * (1 - (*decision.ReducedBaseRate / 100)))
	}

	decision.Amount = percentage(decision.BaseValue, decision.Rate)
	return &decision
}

func calculateICMSST(normalized domain.NormalizedContext, base *domain.ICMSSTDecision) *domain.ICMSSTDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.Amount = 0
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = taxableBase(normalized)
	}

	if decision.MVARate != nil && *decision.MVARate > 0 {
		decision.BaseValue = round2(decision.BaseValue * (1 + (*decision.MVARate / 100)))
	}

	decision.Amount = percentage(decision.BaseValue, decision.Rate)
	return &decision
}

func calculateFCP(normalized domain.NormalizedContext, base *domain.FCPDecision) *domain.FCPDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.Amount = 0
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = taxableBase(normalized)
	}

	decision.Amount = percentage(decision.BaseValue, decision.Rate)
	return &decision
}

func calculateDIFAL(normalized domain.NormalizedContext, base *domain.DIFALDecision) *domain.DIFALDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.AmountDestinationUF = 0
		decision.AmountOriginUF = 0
		if decision.FCPRate != nil {
			v := 0.0
			decision.FCPAmount = &v
		}
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = taxableBase(normalized)
	}

	diffRate := decision.InternalRate - decision.InterstateRate
	if diffRate < 0 {
		diffRate = 0
	}

	decision.AmountDestinationUF = percentage(decision.BaseValue, diffRate)
	decision.AmountOriginUF = 0

	if decision.FCPRate != nil {
		fcpAmount := percentage(decision.BaseValue, *decision.FCPRate)
		decision.FCPAmount = &fcpAmount
	}

	return &decision
}

func calculatePIS(normalized domain.NormalizedContext, base *domain.PISDecision) *domain.PISDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.Amount = 0
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = federalContributionBase(normalized)
	}

	decision.Amount = percentage(decision.BaseValue, decision.Rate)
	return &decision
}

func calculateCOFINS(normalized domain.NormalizedContext, base *domain.COFINSDecision) *domain.COFINSDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.Amount = 0
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = federalContributionBase(normalized)
	}

	decision.Amount = percentage(decision.BaseValue, decision.Rate)
	return &decision
}

func calculateIPI(normalized domain.NormalizedContext, base *domain.IPIDecision) *domain.IPIDecision {
	if base == nil {
		return nil
	}

	decision := *base
	if !decision.Applies {
		decision.BaseValue = 0
		decision.Amount = 0
		return &decision
	}

	if decision.BaseValue <= 0 {
		decision.BaseValue = taxableBase(normalized)
	}

	decision.Amount = percentage(decision.BaseValue, decision.Rate)
	return &decision
}

func taxableBase(normalized domain.NormalizedContext) float64 {
	base := normalized.TotalValue
	if base <= 0 {
		base = normalized.GrossValue
	}

	base = base + normalized.FreightValue + normalized.InsuranceValue + normalized.OtherExpensesValue - normalized.DiscountValue
	if base < 0 {
		return 0
	}

	return round2(base)
}

func federalContributionBase(normalized domain.NormalizedContext) float64 {
	base := normalized.TotalValue
	if base <= 0 {
		base = normalized.GrossValue
	}

	base = base - normalized.DiscountValue
	if base < 0 {
		return 0
	}

	return round2(base)
}

func percentage(base float64, rate float64) float64 {
	if base <= 0 || rate <= 0 {
		return 0
	}
	return round2(base * (rate / 100))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

var _ domain.Calculator = (*Service)(nil)