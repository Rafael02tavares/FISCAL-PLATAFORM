package rules

import (
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Matcher struct{}

func NewMatcher() *Matcher {
	return &Matcher{}
}

type ActionValue struct {
	Text   *string
	Number *float64
	Bool   *bool
}

type SelectedRule struct {
	Rule        domain.Rule
	MatchedRule domain.MatchedRule
	ActionMap   map[string]ActionValue
}

func (m *Matcher) Match(
	taxType domain.TaxType,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
	rules []domain.Rule,
) []*SelectedRule {

	results := make([]*SelectedRule, 0, len(rules))

	for _, rule := range rules {
		ok, score, facts := m.evaluateConditions(rule.Conditions, normalized, classification)
		if !ok {
			continue
		}

		actionMap := buildActionMap(rule.Actions)

		matched := domain.MatchedRule{
			RuleID:        rule.ID,
			RuleCode:      rule.Code,
			RuleName:      rule.Name,
			TaxType:       taxType,
			Score:         score,
			Priority:      rule.Priority,
			MatchedFacts:  facts,
			LegalBasisIDs: rule.LegalBasisIDs,
		}

		results = append(results, &SelectedRule{
			Rule:        rule,
			MatchedRule: matched,
			ActionMap:   actionMap,
		})
	}

	return results
}

func (m *Matcher) evaluateConditions(
	conditions []domain.Condition,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
) (bool, int, []string) {

	totalScore := 0
	facts := make([]string, 0, len(conditions))

	for _, cond := range conditions {
		ok := m.evaluateCondition(cond, normalized, classification)
		if !ok {
			return false, 0, nil
		}

		totalScore += cond.Weight
		facts = append(facts, buildFact(cond))
	}

	return true, totalScore, facts
}

func (m *Matcher) evaluateCondition(
	cond domain.Condition,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
) bool {

	fieldValue := m.resolveField(cond.Field, normalized, classification)

	switch cond.Operator {

	case "eq":
		return fieldValue == getText(cond)

	case "neq":
		return fieldValue != getText(cond)

	case "prefix":
		return strings.HasPrefix(fieldValue, getText(cond))

	case "not_prefix":
		return !strings.HasPrefix(fieldValue, getText(cond))

	case "contains":
		return strings.Contains(fieldValue, getText(cond))

	case "in":
		for _, v := range cond.ValueList {
			if fieldValue == v {
				return true
			}
		}
		return false

	case "not_in":
		for _, v := range cond.ValueList {
			if fieldValue == v {
				return false
			}
		}
		return true

	case "is_true":
		return fieldValue == "true"

	case "is_false":
		return fieldValue == "false"

	default:
		return false
	}
}

func (m *Matcher) resolveField(
	field string,
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
) string {

	switch field {

	case "issuer_uf":
		return normalized.IssuerUF

	case "recipient_uf":
		return normalized.RecipientUF

	case "cfop":
		return normalized.CFOP

	case "operation_scope":
		return string(normalized.OperationScope)

	case "operation_type":
		return string(normalized.OperationType)

	case "crt":
		return normalized.IssuerCRT

	case "final_consumer":
		if normalized.FinalConsumer {
			return "true"
		}
		return "false"

	case "recipient_contributor":
		if normalized.RecipientContributor {
			return "true"
		}
		return "false"

	case "ncm":
		return classification.NCM

	case "ncm_chapter":
		return normalized.NCMChapter

	case "ncm_position":
		return normalized.NCMPosition

	case "cest":
		return normalized.CEST

	case "gtin":
		return normalized.GTIN

	case "supplier_id":
		return normalized.SupplierID

	case "supplier_product_code":
		return normalized.SupplierProductCode

	case "has_gtin":
		if normalized.GTIN != "" {
			return "true"
		}
		return "false"

	case "has_cest":
		if normalized.CEST != "" {
			return "true"
		}
		return "false"

	default:
		return ""
	}
}

func buildActionMap(actions []domain.Action) map[string]ActionValue {
	result := make(map[string]ActionValue, len(actions))

	for _, action := range actions {
		result[action.Target] = ActionValue{
			Text:   action.ValueText,
			Number: action.ValueNumber,
			Bool:   action.ValueBool,
		}
	}

	return result
}

func buildFact(cond domain.Condition) string {
	if cond.ValueText != nil {
		return cond.Field + "=" + *cond.ValueText
	}
	if cond.ValueBool != nil {
		if *cond.ValueBool {
			return cond.Field + "=true"
		}
		return cond.Field + "=false"
	}
	if len(cond.ValueList) > 0 {
		return cond.Field + "=list"
	}
	return cond.Field
}

func getText(cond domain.Condition) string {
	if cond.ValueText != nil {
		return *cond.ValueText
	}
	return ""
}