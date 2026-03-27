package invoices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceItemTaxDecisionRepository struct {
	db *pgxpool.Pool
}

func NewInvoiceItemTaxDecisionRepository(db *pgxpool.Pool) (*InvoiceItemTaxDecisionRepository, error) {
	if db == nil {
		return nil, errors.New("invoice item tax decision repository: db is required")
	}

	return &InvoiceItemTaxDecisionRepository{
		db: db,
	}, nil
}

func (r *InvoiceItemTaxDecisionRepository) UpsertInvoiceItemTaxDecision(
	ctx context.Context,
	input UpsertInvoiceItemTaxDecisionInput,
) error {
	input = sanitizeUpsertInvoiceItemTaxDecisionInput(input)

	if err := validateUpsertInvoiceItemTaxDecisionInput(input); err != nil {
		return err
	}

	warningsJSON, err := marshalJSONArray(input.Warnings)
	if err != nil {
		return fmt.Errorf("invoice item tax decision repository: marshal warnings: %w", err)
	}

	explanationsJSON, err := marshalJSON(input.Explanations)
	if err != nil {
		return fmt.Errorf("invoice item tax decision repository: marshal explanations: %w", err)
	}

	auditTrailJSON, err := marshalJSON(input.AuditTrail)
	if err != nil {
		return fmt.Errorf("invoice item tax decision repository: marshal audit trail: %w", err)
	}

	query := `
		INSERT INTO invoice_item_tax_decisions (
			tenant_id,
			organization_id,
			invoice_id,
			invoice_item_id,

			classification_ncm,
			classification_extipi,
			classification_cest,
			classification_source,
			classification_confidence,
			needs_review,
			summary_status,

			icms_base_value,
			icms_rate,
			icms_amount,
			icms_cst,
			icms_rule_id,

			icms_st_base_value,
			icms_st_rate,
			icms_st_amount,
			icms_st_cst,
			icms_st_rule_id,

			fcp_base_value,
			fcp_rate,
			fcp_amount,
			fcp_rule_id,

			difal_base_value,
			difal_internal_rate,
			difal_interstate_rate,
			difal_amount_destination,
			difal_amount_origin,
			difal_rule_id,

			pis_base_value,
			pis_rate,
			pis_amount,
			pis_cst,
			pis_rule_id,

			cofins_base_value,
			cofins_rate,
			cofins_amount,
			cofins_cst,
			cofins_rule_id,

			ipi_base_value,
			ipi_rate,
			ipi_amount,
			ipi_cst,
			ipi_rule_id,

			warnings_json,
			explanations_json,
			audit_trail_json,
			updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25,
			$26, $27, $28, $29, $30, $31,
			$32, $33, $34, $35, $36,
			$37, $38, $39, $40, $41,
			$42, $43, $44, $45, $46,
			$47, $48, $49, NOW()
		)
		ON CONFLICT (invoice_item_id)
		DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			organization_id = EXCLUDED.organization_id,
			invoice_id = EXCLUDED.invoice_id,

			classification_ncm = EXCLUDED.classification_ncm,
			classification_extipi = EXCLUDED.classification_extipi,
			classification_cest = EXCLUDED.classification_cest,
			classification_source = EXCLUDED.classification_source,
			classification_confidence = EXCLUDED.classification_confidence,
			needs_review = EXCLUDED.needs_review,
			summary_status = EXCLUDED.summary_status,

			icms_base_value = EXCLUDED.icms_base_value,
			icms_rate = EXCLUDED.icms_rate,
			icms_amount = EXCLUDED.icms_amount,
			icms_cst = EXCLUDED.icms_cst,
			icms_rule_id = EXCLUDED.icms_rule_id,

			icms_st_base_value = EXCLUDED.icms_st_base_value,
			icms_st_rate = EXCLUDED.icms_st_rate,
			icms_st_amount = EXCLUDED.icms_st_amount,
			icms_st_cst = EXCLUDED.icms_st_cst,
			icms_st_rule_id = EXCLUDED.icms_st_rule_id,

			fcp_base_value = EXCLUDED.fcp_base_value,
			fcp_rate = EXCLUDED.fcp_rate,
			fcp_amount = EXCLUDED.fcp_amount,
			fcp_rule_id = EXCLUDED.fcp_rule_id,

			difal_base_value = EXCLUDED.difal_base_value,
			difal_internal_rate = EXCLUDED.difal_internal_rate,
			difal_interstate_rate = EXCLUDED.difal_interstate_rate,
			difal_amount_destination = EXCLUDED.difal_amount_destination,
			difal_amount_origin = EXCLUDED.difal_amount_origin,
			difal_rule_id = EXCLUDED.difal_rule_id,

			pis_base_value = EXCLUDED.pis_base_value,
			pis_rate = EXCLUDED.pis_rate,
			pis_amount = EXCLUDED.pis_amount,
			pis_cst = EXCLUDED.pis_cst,
			pis_rule_id = EXCLUDED.pis_rule_id,

			cofins_base_value = EXCLUDED.cofins_base_value,
			cofins_rate = EXCLUDED.cofins_rate,
			cofins_amount = EXCLUDED.cofins_amount,
			cofins_cst = EXCLUDED.cofins_cst,
			cofins_rule_id = EXCLUDED.cofins_rule_id,

			ipi_base_value = EXCLUDED.ipi_base_value,
			ipi_rate = EXCLUDED.ipi_rate,
			ipi_amount = EXCLUDED.ipi_amount,
			ipi_cst = EXCLUDED.ipi_cst,
			ipi_rule_id = EXCLUDED.ipi_rule_id,

			warnings_json = EXCLUDED.warnings_json,
			explanations_json = EXCLUDED.explanations_json,
			audit_trail_json = EXCLUDED.audit_trail_json,
			updated_at = NOW()
	`

	_, err = r.db.Exec(
		ctx,
		query,
		input.TenantID,
		input.OrganizationID,
		input.InvoiceID,
		input.InvoiceItemID,

		input.ClassificationNCM,
		normalizeOptionalString(input.ClassificationEXTIPI),
		normalizeOptionalString(input.ClassificationCEST),
		nullIfEmptyString(input.ClassificationSource),
		input.ClassificationConfidence,
		input.NeedsReview,
		nullIfEmptyString(input.SummaryStatus),

		input.ICMSBaseValue,
		input.ICMSRate,
		input.ICMSAmount,
		normalizeOptionalString(input.ICMSCST),
		normalizeOptionalString(input.ICMSRuleID),

		input.ICMSSTBaseValue,
		input.ICMSSTRate,
		input.ICMSSTAmount,
		normalizeOptionalString(input.ICMSSTCST),
		normalizeOptionalString(input.ICMSSTRuleID),

		input.FCPBaseValue,
		input.FCPRate,
		input.FCPAmount,
		normalizeOptionalString(input.FCPRuleID),

		input.DIFALBaseValue,
		input.DIFALInternalRate,
		input.DIFALInterstateRate,
		input.DIFALAmountDestination,
		input.DIFALAmountOrigin,
		normalizeOptionalString(input.DIFALRuleID),

		input.PISBaseValue,
		input.PISRate,
		input.PISAmount,
		normalizeOptionalString(input.PISCST),
		normalizeOptionalString(input.PISRuleID),

		input.COFINSBaseValue,
		input.COFINSRate,
		input.COFINSAmount,
		normalizeOptionalString(input.COFINSCST),
		normalizeOptionalString(input.COFINSRuleID),

		input.IPIBaseValue,
		input.IPIRate,
		input.IPIAmount,
		normalizeOptionalString(input.IPICST),
		normalizeOptionalString(input.IPIRuleID),

		nullIfEmptyJSON(warningsJSON),
		nullIfEmptyJSON(explanationsJSON),
		nullIfEmptyJSON(auditTrailJSON),
	)
	if err != nil {
		return fmt.Errorf("invoice item tax decision repository: upsert invoice item tax decision: %w", err)
	}

	return nil
}

func validateUpsertInvoiceItemTaxDecisionInput(input UpsertInvoiceItemTaxDecisionInput) error {
	if input.TenantID == "" {
		return errors.New("invoice item tax decision repository: tenant_id is required")
	}
	if input.OrganizationID == "" {
		return errors.New("invoice item tax decision repository: organization_id is required")
	}
	if input.InvoiceID == "" {
		return errors.New("invoice item tax decision repository: invoice_id is required")
	}
	if input.InvoiceItemID == "" {
		return errors.New("invoice item tax decision repository: invoice_item_id is required")
	}

	return nil
}

func sanitizeUpsertInvoiceItemTaxDecisionInput(input UpsertInvoiceItemTaxDecisionInput) UpsertInvoiceItemTaxDecisionInput {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.OrganizationID = strings.TrimSpace(input.OrganizationID)
	input.InvoiceID = strings.TrimSpace(input.InvoiceID)
	input.InvoiceItemID = strings.TrimSpace(input.InvoiceItemID)

	input.ClassificationNCM = strings.TrimSpace(input.ClassificationNCM)
	input.ClassificationEXTIPI = trimOptionalString(input.ClassificationEXTIPI)
	input.ClassificationCEST = trimOptionalString(input.ClassificationCEST)
	input.ClassificationSource = strings.TrimSpace(input.ClassificationSource)
	input.SummaryStatus = strings.TrimSpace(input.SummaryStatus)

	input.ICMSCST = trimOptionalString(input.ICMSCST)
	input.ICMSRuleID = trimOptionalString(input.ICMSRuleID)

	input.ICMSSTCST = trimOptionalString(input.ICMSSTCST)
	input.ICMSSTRuleID = trimOptionalString(input.ICMSSTRuleID)

	input.FCPRuleID = trimOptionalString(input.FCPRuleID)

	input.DIFALRuleID = trimOptionalString(input.DIFALRuleID)

	input.PISCST = trimOptionalString(input.PISCST)
	input.PISRuleID = trimOptionalString(input.PISRuleID)

	input.COFINSCST = trimOptionalString(input.COFINSCST)
	input.COFINSRuleID = trimOptionalString(input.COFINSRuleID)

	input.IPICST = trimOptionalString(input.IPICST)
	input.IPIRuleID = trimOptionalString(input.IPIRuleID)

	input.Warnings = compactStrings(input.Warnings)

	return input
}

func marshalJSONArray(values []string) ([]byte, error) {
	if len(values) == 0 {
		return json.Marshal([]string{})
	}
	return json.Marshal(values)
}

func marshalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func compactStrings(values []string) []string {
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
		if _, exists := seen[value]; exists {
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

func normalizeOptionalString(value *string) any {
	if value == nil {
		return nil
	}

	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}

	return v
}

func nullIfEmptyString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullIfEmptyJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

var _ TaxDecisionRepository = (*InvoiceItemTaxDecisionRepository)(nil)