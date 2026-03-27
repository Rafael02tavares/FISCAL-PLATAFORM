package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type RuleRepository struct {
	db *pgxpool.Pool
}

func NewRuleRepository(db *pgxpool.Pool) (*RuleRepository, error) {
	if db == nil {
		return nil, errors.New("taxengine postgres: db pool is required")
	}

	return &RuleRepository{db: db}, nil
}

func (r *RuleRepository) FindCandidateRules(
	ctx context.Context,
	filter domain.RuleFilter,
) ([]domain.Rule, error) {
	if err := validateRuleFilter(filter); err != nil {
		return nil, err
	}

	rules, err := r.fetchRules(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, nil
	}

	ruleIDs := make([]string, 0, len(rules))
	for _, rule := range rules {
		ruleIDs = append(ruleIDs, rule.ID)
	}

	conditionsByRuleID, err := r.fetchConditionsByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}

	actionsByRuleID, err := r.fetchActionsByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}

	for i := range rules {
		rules[i].Conditions = conditionsByRuleID[rules[i].ID]
		rules[i].Actions = actionsByRuleID[rules[i].ID]
	}

	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		if rules[i].SpecificityHint != rules[j].SpecificityHint {
			return rules[i].SpecificityHint > rules[j].SpecificityHint
		}
		if !rules[i].ValidFrom.Equal(rules[j].ValidFrom) {
			return rules[i].ValidFrom.After(rules[j].ValidFrom)
		}
		return rules[i].Code < rules[j].Code
	})

	return rules, nil
}

func validateRuleFilter(filter domain.RuleFilter) error {
	if strings.TrimSpace(string(filter.TaxType)) == "" {
		return errors.New("taxengine postgres: rule filter tax_type is required")
	}
	if strings.TrimSpace(string(filter.Jurisdiction)) == "" {
		return errors.New("taxengine postgres: rule filter jurisdiction is required")
	}
	if strings.TrimSpace(filter.ReferenceDate) == "" {
		return errors.New("taxengine postgres: rule filter reference_date is required")
	}
	return nil
}

func (r *RuleRepository) fetchRules(
	ctx context.Context,
	filter domain.RuleFilter,
) ([]domain.Rule, error) {
	var args []any
	args = append(args, string(filter.TaxType))
	args = append(args, string(filter.Jurisdiction))
	args = append(args, filter.ReferenceDate)

	query := `
		SELECT
			id,
			code,
			name,
			tax_type,
			jurisdiction_type,
			uf,
			priority,
			specificity_hint,
			valid_from,
			valid_to,
			status,
			legal_basis_ids,
			created_at,
			updated_at
		FROM tax_rules
		WHERE tax_type = $1
		  AND jurisdiction_type = $2
		  AND valid_from <= $3::date
		  AND (valid_to IS NULL OR valid_to >= $3::date)
	`

	argPos := 4

	if filter.OnlyActive {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, string(domain.RuleStatusActive))
		argPos++
	}

	if filter.UF != nil && strings.TrimSpace(*filter.UF) != "" {
		query += fmt.Sprintf(" AND (uf IS NULL OR uf = $%d)", argPos)
		args = append(args, strings.TrimSpace(strings.ToUpper(*filter.UF)))
		argPos++
	}

	query += `
		ORDER BY priority DESC, specificity_hint DESC, valid_from DESC, code ASC
	`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("taxengine postgres: query rules: %w", err)
	}
	defer rows.Close()

	rules := make([]domain.Rule, 0, 32)

	for rows.Next() {
		var rule domain.Rule
		var taxType string
		var jurisdictionType string
		var status string
		var uf *string
		var validTo *time.Time
		var legalBasisIDs []string

		err := rows.Scan(
			&rule.ID,
			&rule.Code,
			&rule.Name,
			&taxType,
			&jurisdictionType,
			&uf,
			&rule.Priority,
			&rule.SpecificityHint,
			&rule.ValidFrom,
			&validTo,
			&status,
			&legalBasisIDs,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("taxengine postgres: scan rule: %w", err)
		}

		rule.TaxType = domain.TaxType(strings.TrimSpace(taxType))
		rule.JurisdictionType = domain.JurisdictionType(strings.TrimSpace(jurisdictionType))
		rule.Status = domain.RuleStatus(strings.TrimSpace(status))
		rule.UF = normalizeOptionalUpper(uf)
		rule.ValidTo = validTo
		rule.LegalBasisIDs = cloneStringsCompact(legalBasisIDs)

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("taxengine postgres: iterate rules: %w", err)
	}

	return rules, nil
}

func (r *RuleRepository) fetchConditionsByRuleIDs(
	ctx context.Context,
	ruleIDs []string,
) (map[string][]domain.Condition, error) {
	if len(ruleIDs) == 0 {
		return map[string][]domain.Condition{}, nil
	}

	query := `
		SELECT
			rule_id,
			field_name,
			operator,
			value_text,
			value_number,
			value_bool,
			value_list,
			value_min,
			value_max,
			weight
		FROM tax_rule_conditions
		WHERE rule_id = ANY($1)
		ORDER BY rule_id ASC, weight DESC, field_name ASC
	`

	rows, err := r.db.Query(ctx, query, ruleIDs)
	if err != nil {
		return nil, fmt.Errorf("taxengine postgres: query rule conditions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]domain.Condition, len(ruleIDs))

	for rows.Next() {
		var ruleID string
		var condition domain.Condition
		var valueText *string
		var valueNumber *float64
		var valueBool *bool
		var valueList []string
		var valueMin *float64
		var valueMax *float64

		err := rows.Scan(
			&ruleID,
			&condition.Field,
			&condition.Operator,
			&valueText,
			&valueNumber,
			&valueBool,
			&valueList,
			&valueMin,
			&valueMax,
			&condition.Weight,
		)
		if err != nil {
			return nil, fmt.Errorf("taxengine postgres: scan rule condition: %w", err)
		}

		condition.Field = strings.TrimSpace(condition.Field)
		condition.Operator = strings.TrimSpace(condition.Operator)
		condition.ValueText = normalizeOptionalTrim(valueText)
		condition.ValueNumber = valueNumber
		condition.ValueBool = valueBool
		condition.ValueList = cloneStringsCompact(valueList)
		condition.ValueMin = valueMin
		condition.ValueMax = valueMax

		result[ruleID] = append(result[ruleID], condition)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("taxengine postgres: iterate rule conditions: %w", err)
	}

	return result, nil
}

func (r *RuleRepository) fetchActionsByRuleIDs(
	ctx context.Context,
	ruleIDs []string,
) (map[string][]domain.Action, error) {
	if len(ruleIDs) == 0 {
		return map[string][]domain.Action{}, nil
	}

	query := `
		SELECT
			rule_id,
			action_type,
			target_field,
			value_text,
			value_number,
			value_bool,
			value_json
		FROM tax_rule_actions
		WHERE rule_id = ANY($1)
		ORDER BY rule_id ASC, target_field ASC
	`

	rows, err := r.db.Query(ctx, query, ruleIDs)
	if err != nil {
		return nil, fmt.Errorf("taxengine postgres: query rule actions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]domain.Action, len(ruleIDs))

	for rows.Next() {
		var ruleID string
		var action domain.Action
		var valueText *string
		var valueNumber *float64
		var valueBool *bool
		var valueJSON []byte

		err := rows.Scan(
			&ruleID,
			&action.Type,
			&action.Target,
			&valueText,
			&valueNumber,
			&valueBool,
			&valueJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("taxengine postgres: scan rule action: %w", err)
		}

		action.Type = strings.TrimSpace(action.Type)
		action.Target = strings.TrimSpace(action.Target)
		action.ValueText = normalizeOptionalTrim(valueText)
		action.ValueNumber = valueNumber
		action.ValueBool = valueBool
		action.ValueJSON = cloneBytes(valueJSON)

		result[ruleID] = append(result[ruleID], action)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("taxengine postgres: iterate rule actions: %w", err)
	}

	return result, nil
}

func normalizeOptionalTrim(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeOptionalUpper(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(strings.ToUpper(*value))
	if v == "" {
		return nil
	}
	return &v
}

func cloneStringsCompact(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

var _ domain.RuleRepository = (*RuleRepository)(nil)