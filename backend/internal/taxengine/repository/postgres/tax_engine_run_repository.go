package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type TaxEngineRunRepository struct {
	db *pgxpool.Pool
}

func NewTaxEngineRunRepository(db *pgxpool.Pool) (*TaxEngineRunRepository, error) {
	if db == nil {
		return nil, errors.New("taxengine postgres: db pool is required")
	}

	return &TaxEngineRunRepository{db: db}, nil
}

func (r *TaxEngineRunRepository) CreateRun(
	ctx context.Context,
	run domain.TaxEngineRun,
) (string, error) {
	run = sanitizeTaxEngineRun(run)

	if run.TenantID == "" {
		return "", errors.New("taxengine postgres: tax engine run tenant_id is required")
	}
	if run.OrganizationID == "" {
		return "", errors.New("taxengine postgres: tax engine run organization_id is required")
	}
	if strings.TrimSpace(run.Status) == "" {
		run.Status = "SUCCESS"
	}

	query := `
		INSERT INTO tax_engine_runs (
			tenant_id,
			organization_id,
			invoice_id,
			invoice_item_id,
			input_payload,
			normalized_payload,
			classification_json,
			evaluation_json,
			taxes_json,
			output_payload,
			status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		RETURNING id
	`

	var runID string

	err := r.db.QueryRow(
		ctx,
		query,
		run.TenantID,
		run.OrganizationID,
		nullIfEmptyPtr(run.InvoiceID),
		nullIfEmptyPtr(run.InvoiceItemID),
		nullIfEmptyJSON(run.InputPayload),
		nullIfEmptyJSON(run.NormalizedPayload),
		nullIfEmptyJSON(run.ClassificationJSON),
		nullIfEmptyJSON(run.EvaluationJSON),
		nullIfEmptyJSON(run.TaxesJSON),
		nullIfEmptyJSON(run.OutputPayload),
		strings.TrimSpace(run.Status),
	).Scan(&runID)
	if err != nil {
		return "", fmt.Errorf("taxengine postgres: create tax engine run: %w", err)
	}

	return strings.TrimSpace(runID), nil
}

func (r *TaxEngineRunRepository) CreateAuditSteps(
	ctx context.Context,
	runID string,
	steps []domain.AuditStepRecord,
) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return errors.New("taxengine postgres: run_id is required")
	}
	if len(steps) == 0 {
		return nil
	}

	var batch pgx.Batch

	for _, step := range steps {
		step = sanitizeAuditStepRecord(step)

		query := `
			INSERT INTO tax_engine_audit_steps (
				run_id,
				step_order,
				step_name,
				status,
				message,
				matched_rule_id,
				payload_json
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7
			)
		`

		batch.Queue(
			query,
			runID,
			step.Order,
			step.Step,
			string(step.Status),
			step.Message,
			nullIfEmptyPtr(step.MatchedRuleID),
			nullIfEmptyJSON(step.PayloadJSON),
		)
	}

	results := r.db.SendBatch(ctx, &batch)
	defer results.Close()

	for range steps {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("taxengine postgres: create audit steps: %w", err)
		}
	}

	return nil
}

func sanitizeTaxEngineRun(run domain.TaxEngineRun) domain.TaxEngineRun {
	run.TenantID = strings.TrimSpace(run.TenantID)
	run.OrganizationID = strings.TrimSpace(run.OrganizationID)
	run.InvoiceID = trimOptionalString(run.InvoiceID)
	run.InvoiceItemID = trimOptionalString(run.InvoiceItemID)
	run.Status = strings.TrimSpace(run.Status)

	run.InputPayload = cloneBytes(run.InputPayload)
	run.NormalizedPayload = cloneBytes(run.NormalizedPayload)
	run.ClassificationJSON = cloneBytes(run.ClassificationJSON)
	run.EvaluationJSON = cloneBytes(run.EvaluationJSON)
	run.TaxesJSON = cloneBytes(run.TaxesJSON)
	run.OutputPayload = cloneBytes(run.OutputPayload)

	return run
}

func sanitizeAuditStepRecord(step domain.AuditStepRecord) domain.AuditStepRecord {
	step.Step = strings.TrimSpace(step.Step)
	step.Message = strings.TrimSpace(step.Message)
	step.MatchedRuleID = trimOptionalString(step.MatchedRuleID)
	step.PayloadJSON = cloneBytes(step.PayloadJSON)

	if strings.TrimSpace(string(step.Status)) == "" {
		step.Status = domain.AuditStepStatusSuccess
	}

	return step
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}

	return &v
}

func nullIfEmptyPtr(value *string) any {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return v
}

func nullIfEmptyJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

var _ domain.TaxEngineRunRepository = (*TaxEngineRunRepository)(nil)