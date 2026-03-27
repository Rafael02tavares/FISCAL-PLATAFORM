package taxengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Service struct {
	normalizer domain.Normalizer
	classifier domain.Classifier
	evaluator  domain.RuleEvaluator
	calculator domain.Calculator
	explainer  domain.Explainer
	auditor    domain.Auditor
}

func NewService(
	normalizer domain.Normalizer,
	classifier domain.Classifier,
	evaluator domain.RuleEvaluator,
	calculator domain.Calculator,
	explainer domain.Explainer,
	auditor domain.Auditor,
) (*Service, error) {
	if normalizer == nil {
		return nil, errors.New("taxengine: normalizer is required")
	}
	if classifier == nil {
		return nil, errors.New("taxengine: classifier is required")
	}
	if evaluator == nil {
		return nil, errors.New("taxengine: evaluator is required")
	}
	if calculator == nil {
		return nil, errors.New("taxengine: calculator is required")
	}
	if explainer == nil {
		return nil, errors.New("taxengine: explainer is required")
	}
	if auditor == nil {
		return nil, errors.New("taxengine: auditor is required")
	}

	return &Service{
		normalizer: normalizer,
		classifier: classifier,
		evaluator:  evaluator,
		calculator: calculator,
		explainer:  explainer,
		auditor:    auditor,
	}, nil
}

func (s *Service) Evaluate(ctx context.Context, input domain.EvaluateInput) (*domain.EvaluateOutput, error) {
	normalized, err := s.normalizer.Normalize(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("taxengine: normalize input: %w", err)
	}

	classification, err := s.classifier.Resolve(ctx, *normalized)
	if err != nil {
		return nil, fmt.Errorf("taxengine: resolve classification: %w", err)
	}

	evaluation, err := s.evaluator.Evaluate(ctx, *normalized, *classification)
	if err != nil {
		return nil, fmt.Errorf("taxengine: evaluate rules: %w", err)
	}

	taxes, err := s.calculator.Calculate(ctx, *normalized, *classification, *evaluation)
	if err != nil {
		return nil, fmt.Errorf("taxengine: calculate taxes: %w", err)
	}

	explanations := s.explainer.Build(*normalized, *classification, *evaluation, *taxes)

	output := &domain.EvaluateOutput{
		Classification: *classification,
		Taxes:          *taxes,
		Summary:        buildSummary(*classification, *evaluation, *taxes),
		Explanations:   explanations,
		Warnings:       deduplicateStrings(evaluation.Warnings),
		AuditTrail:     evaluation.AuditTrail,
	}

	if err := s.auditor.RecordRun(
		ctx,
		input,
		*normalized,
		*classification,
		*evaluation,
		*taxes,
		*output,
	); err != nil {
		return nil, fmt.Errorf("taxengine: audit run: %w", err)
	}

	return output, nil
}

func buildSummary(
	classification domain.ClassificationDecision,
	evaluation domain.RuleEvaluationResult,
	taxes domain.TaxDecisionSet,
) domain.DecisionSummary {
	confidence := classification.Confidence

	status := domain.DecisionStatusAutomatic
	requiresManualReview := false
	hasWarnings := len(evaluation.Warnings) > 0

	if classification.NeedsReview {
		status = domain.DecisionStatusManualReview
		requiresManualReview = true
	} else if confidence < 0.80 {
		status = domain.DecisionStatusSuggested
	}

	if isMissingCriticalTaxData(classification, taxes) {
		status = domain.DecisionStatusBlockedMissingData
		requiresManualReview = true
	}

	return domain.DecisionSummary{
		Status:               status,
		Confidence:           confidence,
		RequiresManualReview: requiresManualReview,
		HasWarnings:          hasWarnings,
	}
}

func isMissingCriticalTaxData(
	classification domain.ClassificationDecision,
	taxes domain.TaxDecisionSet,
) bool {
	if classification.NCM == "" {
		return true
	}

	if taxes.ICMS == nil && taxes.PIS == nil && taxes.COFINS == nil && taxes.IPI == nil {
		return true
	}

	return false
}

func deduplicateStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}