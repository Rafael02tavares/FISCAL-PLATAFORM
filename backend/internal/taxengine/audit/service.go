package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Service struct {
	runRepo domain.TaxEngineRunRepository
}

func NewService(runRepo domain.TaxEngineRunRepository) (*Service, error) {
	if runRepo == nil {
		return nil, errors.New("taxengine audit: run repository is required")
	}

	return &Service{
		runRepo: runRepo,
	}, nil
}

func (s *Service) RecordRun(
	ctx context.Context,
	input domain.EvaluateInput,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
	evaluation domain.RuleEvaluationResult,
	taxes domain.TaxDecisionSet,
	output domain.EvaluateOutput,
) error {
	run, err := buildTaxEngineRun(input, normalized, classification, evaluation, taxes, output)
	if err != nil {
		return fmt.Errorf("taxengine audit: build run payload: %w", err)
	}

	runID, err := s.runRepo.CreateRun(ctx, run)
	if err != nil {
		return fmt.Errorf("taxengine audit: create run: %w", err)
	}

	steps, err := buildAuditStepRecords(evaluation.AuditTrail)
	if err != nil {
		return fmt.Errorf("taxengine audit: build audit steps: %w", err)
	}

	if err := s.runRepo.CreateAuditSteps(ctx, runID, steps); err != nil {
		return fmt.Errorf("taxengine audit: create audit steps: %w", err)
	}

	return nil
}

func buildTaxEngineRun(
	input domain.EvaluateInput,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
	evaluation domain.RuleEvaluationResult,
	taxes domain.TaxDecisionSet,
	output domain.EvaluateOutput,
) (domain.TaxEngineRun, error) {
	inputJSON, err := marshalJSON(input)
	if err != nil {
		return domain.TaxEngineRun{}, fmt.Errorf("marshal input: %w", err)
	}

	normalizedJSON, err := marshalJSON(normalized)
	if err != nil {
		return domain.TaxEngineRun{}, fmt.Errorf("marshal normalized context: %w", err)
	}

	classificationJSON, err := marshalJSON(classification)
	if err != nil {
		return domain.TaxEngineRun{}, fmt.Errorf("marshal classification: %w", err)
	}

	evaluationJSON, err := marshalJSON(evaluation)
	if err != nil {
		return domain.TaxEngineRun{}, fmt.Errorf("marshal evaluation: %w", err)
	}

	taxesJSON, err := marshalJSON(taxes)
	if err != nil {
		return domain.TaxEngineRun{}, fmt.Errorf("marshal taxes: %w", err)
	}

	outputJSON, err := marshalJSON(output)
	if err != nil {
		return domain.TaxEngineRun{}, fmt.Errorf("marshal output: %w", err)
	}

	status := deriveRunStatus(output)

	return domain.TaxEngineRun{
		TenantID:           strings.TrimSpace(input.TenantID),
		OrganizationID:     strings.TrimSpace(input.OrganizationID),
		InvoiceID:          input.InvoiceID,
		InvoiceItemID:      input.InvoiceItemID,
		InputPayload:       inputJSON,
		NormalizedPayload:  normalizedJSON,
		ClassificationJSON: classificationJSON,
		EvaluationJSON:     evaluationJSON,
		TaxesJSON:          taxesJSON,
		OutputPayload:      outputJSON,
		Status:             status,
	}, nil
}

func buildAuditStepRecords(auditTrail []domain.AuditStep) ([]domain.AuditStepRecord, error) {
	if len(auditTrail) == 0 {
		return nil, nil
	}

	records := make([]domain.AuditStepRecord, 0, len(auditTrail))

	for _, step := range auditTrail {
		payload, err := marshalJSON(map[string]any{
			"created_at": step.CreatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal audit step payload: %w", err)
		}

		record := domain.AuditStepRecord{
			Order:         step.Order,
			Step:          strings.TrimSpace(step.Step),
			Status:        step.Status,
			Message:       strings.TrimSpace(step.Message),
			MatchedRuleID: cloneOptionalString(step.MatchedRuleID),
			PayloadJSON:   payload,
		}

		if strings.TrimSpace(record.Step) == "" {
			record.Step = "unknown_step"
		}
		if strings.TrimSpace(string(record.Status)) == "" {
			record.Status = domain.AuditStepStatusSuccess
		}

		records = append(records, record)
	}

	return records, nil
}

func deriveRunStatus(output domain.EvaluateOutput) string {
	switch output.Summary.Status {
	case domain.DecisionStatusBlockedMissingData:
		return "BLOCKED"
	case domain.DecisionStatusManualReview:
		return "MANUAL_REVIEW_REQUIRED"
	case domain.DecisionStatusSuggested:
		return "SUGGESTED"
	case domain.DecisionStatusAutomatic:
		return "SUCCESS"
	default:
		return "SUCCESS"
	}
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
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

var _ domain.Auditor = (*Service)(nil)