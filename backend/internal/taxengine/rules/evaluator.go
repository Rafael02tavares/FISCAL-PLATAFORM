package rules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Evaluator struct {
	ruleRepo domain.RuleRepository
	matcher  *Matcher
	resolver *ConflictResolver
	clock    Clock
}

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func NewEvaluator(
	ruleRepo domain.RuleRepository,
	matcher *Matcher,
	resolver *ConflictResolver,
	clock Clock,
) (*Evaluator, error) {
	if ruleRepo == nil {
		return nil, errors.New("rules evaluator: rule repository is required")
	}
	if matcher == nil {
		return nil, errors.New("rules evaluator: matcher is required")
	}
	if resolver == nil {
		return nil, errors.New("rules evaluator: conflict resolver is required")
	}
	if clock == nil {
		clock = systemClock{}
	}

	return &Evaluator{
		ruleRepo: ruleRepo,
		matcher:  matcher,
		resolver: resolver,
		clock:    clock,
	}, nil
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
) (*domain.RuleEvaluationResult, error) {
	referenceDate := e.clock.Now()

	result := &domain.RuleEvaluationResult{
		Decisions:    domain.PreliminaryTaxDecisions{},
		MatchedRules: make([]domain.MatchedRule, 0, 8),
		Warnings:     make([]string, 0, 8),
		AuditTrail:   make([]domain.AuditStep, 0, 16),
	}

	taxTypes := []domain.TaxType{
		domain.TaxTypeIPI,
		domain.TaxTypePIS,
		domain.TaxTypeCOFINS,
		domain.TaxTypeICMS,
		domain.TaxTypeICMSST,
		domain.TaxTypeFCP,
		domain.TaxTypeDIFAL,
	}

	order := 1

	for _, taxType := range taxTypes {
		stepPrefix := fmt.Sprintf("evaluate_%s", taxType)

		result.AuditTrail = append(result.AuditTrail, domain.AuditStep{
			Order:     order,
			Step:      stepPrefix + "_start",
			Status:    domain.AuditStepStatusSuccess,
			Message:   fmt.Sprintf("iniciando avaliação de %s", taxType),
			CreatedAt: referenceDate,
		})
		order++

		filter := buildRuleFilter(taxType, normalized, referenceDate)
		candidates, err := e.ruleRepo.FindCandidateRules(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("rules evaluator: find candidate rules for %s: %w", taxType, err)
		}

		if len(candidates) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("nenhuma regra candidata encontrada para %s", taxType))
			result.AuditTrail = append(result.AuditTrail, domain.AuditStep{
				Order:     order,
				Step:      stepPrefix + "_no_candidates",
				Status:    domain.AuditStepStatusWarning,
				Message:   fmt.Sprintf("nenhuma regra candidata encontrada para %s", taxType),
				CreatedAt: referenceDate,
			})
			order++
			continue
		}

		matches := e.matcher.Match(taxType, normalized, classification, candidates)

		if len(matches) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("nenhuma regra compatível encontrada para %s", taxType))
			result.AuditTrail = append(result.AuditTrail, domain.AuditStep{
				Order:     order,
				Step:      stepPrefix + "_no_match",
				Status:    domain.AuditStepStatusWarning,
				Message:   fmt.Sprintf("nenhuma regra compatível encontrada para %s", taxType),
				CreatedAt: referenceDate,
			})
			order++
			continue
		}

		selected := e.resolver.Resolve(matches)
		if selected == nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("não foi possível resolver conflito de regras para %s", taxType))
			result.AuditTrail = append(result.AuditTrail, domain.AuditStep{
				Order:     order,
				Step:      stepPrefix + "_conflict_unresolved",
				Status:    domain.AuditStepStatusWarning,
				Message:   fmt.Sprintf("não foi possível resolver conflito de regras para %s", taxType),
				CreatedAt: referenceDate,
			})
			order++
			continue
		}

		result.MatchedRules = append(result.MatchedRules, selected.MatchedRule)
		applySelectedDecision(&result.Decisions, taxType, selected)

		matchedRuleID := selected.Rule.ID
		result.AuditTrail = append(result.AuditTrail, domain.AuditStep{
			Order:         order,
			Step:          stepPrefix + "_selected_rule",
			Status:        domain.AuditStepStatusSuccess,
			Message:       fmt.Sprintf("regra %s selecionada para %s", selected.Rule.Code, taxType),
			MatchedRuleID: &matchedRuleID,
			CreatedAt:     referenceDate,
		})
		order++
	}

	return result, nil
}

func buildRuleFilter(
	taxType domain.TaxType,
	normalized domain.NormalizedContext,
	referenceDate time.Time,
) domain.RuleFilter {
	jurisdiction := jurisdictionByTaxType(taxType)

	var uf *string
	if jurisdiction == domain.JurisdictionState {
		if normalized.RecipientUF != "" {
			uf = &normalized.RecipientUF
		} else if normalized.IssuerUF != "" {
			uf = &normalized.IssuerUF
		}
	}

	return domain.RuleFilter{
		TaxType:       taxType,
		Jurisdiction:  jurisdiction,
		UF:            uf,
		ReferenceDate: referenceDate.Format("2006-01-02"),
		OnlyActive:    true,
	}
}

func jurisdictionByTaxType(taxType domain.TaxType) domain.JurisdictionType {
	switch taxType {
	case domain.TaxTypePIS, domain.TaxTypeCOFINS, domain.TaxTypeIPI:
		return domain.JurisdictionFederal
	default:
		return domain.JurisdictionState
	}
}

func applySelectedDecision(
	decisions *domain.PreliminaryTaxDecisions,
	taxType domain.TaxType,
	selected *SelectedRule,
) {
	if decisions == nil || selected == nil {
		return
	}

	switch taxType {
	case domain.TaxTypeICMS:
		decisions.ICMS = buildICMSDecision(selected)
	case domain.TaxTypeICMSST:
		decisions.ICMSST = buildICMSSTDecision(selected)
	case domain.TaxTypeFCP:
		decisions.FCP = buildFCPDecision(selected)
	case domain.TaxTypeDIFAL:
		decisions.DIFAL = buildDIFALDecision(selected)
	case domain.TaxTypePIS:
		decisions.PIS = buildPISDecision(selected)
	case domain.TaxTypeCOFINS:
		decisions.COFINS = buildCOFINSDecision(selected)
	case domain.TaxTypeIPI:
		decisions.IPI = buildIPIDecision(selected)
	}
}

func buildICMSDecision(selected *SelectedRule) *domain.ICMSDecision {
	decision := &domain.ICMSDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyICMSActions(decision, selected.ActionMap)
	return decision
}

func buildICMSSTDecision(selected *SelectedRule) *domain.ICMSSTDecision {
	decision := &domain.ICMSSTDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyICMSSTActions(decision, selected.ActionMap)
	return decision
}

func buildFCPDecision(selected *SelectedRule) *domain.FCPDecision {
	decision := &domain.FCPDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyFCPActions(decision, selected.ActionMap)
	return decision
}

func buildDIFALDecision(selected *SelectedRule) *domain.DIFALDecision {
	decision := &domain.DIFALDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyDIFALActions(decision, selected.ActionMap)
	return decision
}

func buildPISDecision(selected *SelectedRule) *domain.PISDecision {
	decision := &domain.PISDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyPISActions(decision, selected.ActionMap)
	return decision
}

func buildCOFINSDecision(selected *SelectedRule) *domain.COFINSDecision {
	decision := &domain.COFINSDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyCOFINSActions(decision, selected.ActionMap)
	return decision
}

func buildIPIDecision(selected *SelectedRule) *domain.IPIDecision {
	decision := &domain.IPIDecision{
		Applies:       true,
		RuleID:        selected.Rule.ID,
		LegalBasisIDs: cloneStrings(selected.Rule.LegalBasisIDs),
		Reason:        fmt.Sprintf("regra %s aplicada", selected.Rule.Code),
	}

	applyIPIActions(decision, selected.ActionMap)
	return decision
}

func applyICMSActions(decision *domain.ICMSDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getString(actionMap, "cst"); ok {
		decision.CST = value
	}
	if value, ok := getString(actionMap, "csosn"); ok {
		decision.CSOSN = value
	}
	if value, ok := getString(actionMap, "mod_bc"); ok {
		decision.ModBC = &value
	}
	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "rate"); ok {
		decision.Rate = value
	}
	if value, ok := getFloat(actionMap, "reduced_base_rate"); ok {
		decision.ReducedBaseRate = &value
	}
	if value, ok := getFloat(actionMap, "deferred_rate"); ok {
		decision.DeferredRate = &value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func applyICMSSTActions(decision *domain.ICMSSTDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getString(actionMap, "cst"); ok {
		decision.CST = value
	}
	if value, ok := getString(actionMap, "mod_bc_st"); ok {
		decision.ModBCST = &value
	}
	if value, ok := getFloat(actionMap, "mva_rate"); ok {
		decision.MVARate = &value
	}
	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "rate"); ok {
		decision.Rate = value
	}
	if value, ok := getFloat(actionMap, "fcp_st_rate"); ok {
		decision.FCPSTRate = &value
	}
	if value, ok := getFloat(actionMap, "fcp_st_amount"); ok {
		decision.FCPSTAmount = &value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func applyFCPActions(decision *domain.FCPDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "rate"); ok {
		decision.Rate = value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func applyDIFALActions(decision *domain.DIFALDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "internal_rate"); ok {
		decision.InternalRate = value
	}
	if value, ok := getFloat(actionMap, "interstate_rate"); ok {
		decision.InterstateRate = value
	}
	if value, ok := getFloat(actionMap, "fcp_rate"); ok {
		decision.FCPRate = &value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func applyPISActions(decision *domain.PISDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getString(actionMap, "cst"); ok {
		decision.CST = value
	}
	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "rate"); ok {
		decision.Rate = value
	}
	if value, ok := getString(actionMap, "nature_code"); ok {
		decision.NatureCode = &value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func applyCOFINSActions(decision *domain.COFINSDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getString(actionMap, "cst"); ok {
		decision.CST = value
	}
	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "rate"); ok {
		decision.Rate = value
	}
	if value, ok := getString(actionMap, "nature_code"); ok {
		decision.NatureCode = &value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func applyIPIActions(decision *domain.IPIDecision, actionMap map[string]ActionValue) {
	if decision == nil {
		return
	}

	if value, ok := getString(actionMap, "cst"); ok {
		decision.CST = value
	}
	if value, ok := getFloat(actionMap, "base_value"); ok {
		decision.BaseValue = value
	}
	if value, ok := getFloat(actionMap, "rate"); ok {
		decision.Rate = value
	}
	if value, ok := getString(actionMap, "enquadramento_code"); ok {
		decision.EnquadramentoCode = &value
	}
	if value, ok := getBool(actionMap, "applies"); ok {
		decision.Applies = value
	}
	if value, ok := getString(actionMap, "reason"); ok && value != "" {
		decision.Reason = value
	}
}

func getString(actionMap map[string]ActionValue, key string) (string, bool) {
	value, ok := actionMap[key]
	if !ok || value.Text == nil {
		return "", false
	}
	return *value.Text, true
}

func getFloat(actionMap map[string]ActionValue, key string) (float64, bool) {
	value, ok := actionMap[key]
	if !ok || value.Number == nil {
		return 0, false
	}
	return *value.Number, true
}

func getBool(actionMap map[string]ActionValue, key string) (bool, bool) {
	value, ok := actionMap[key]
	if !ok || value.Bool == nil {
		return false, false
	}
	return *value.Bool, true
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, len(values))
	copy(out, values)
	return out
}