package adminpartilha

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type DIFALRule struct {
	ID                   string   `json:"id"`
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	UF                   string   `json:"uf"`
	Priority             int      `json:"priority"`
	SpecificityHint      int      `json:"specificity_hint"`
	Status               string   `json:"status"`
	ValidFrom            string   `json:"valid_from"`
	ValidTo              string   `json:"valid_to"`
	LegalBasisIDs        []string `json:"legal_basis_ids"`
	IssuerUF             string   `json:"issuer_uf"`
	RecipientUF          string   `json:"recipient_uf"`
	OperationScope       string   `json:"operation_scope"`
	OperationType        string   `json:"operation_type"`
	FinalConsumerMode    string   `json:"final_consumer_mode"`
	RecipientContributor string   `json:"recipient_contributor"`
	CRT                  string   `json:"crt"`
	CFOPPrefix           string   `json:"cfop_prefix"`
	NCMPrefix            string   `json:"ncm_prefix"`
	InternalRate         string   `json:"internal_rate"`
	InterstateRate       string   `json:"interstate_rate"`
	FCPRate              string   `json:"fcp_rate"`
	Applies              bool     `json:"applies"`
	Reason               string   `json:"reason"`
}

type CreateDIFALRuleParams struct {
	Code                 string
	Name                 string
	UF                   string
	Priority             int
	Status               string
	ValidFrom            string
	ValidTo              string
	LegalBasisIDs        []string
	IssuerUF             string
	RecipientUF          string
	OperationScope       string
	OperationType        string
	FinalConsumerMode    string
	RecipientContributor string
	CRT                  string
	CFOPPrefix           string
	NCMPrefix            string
	InternalRate         string
	InterstateRate       string
	FCPRate              string
	Applies              bool
	Reason               string
	SpecificityHint      int
}

func (r *Repository) ListDIFALRules(ctx context.Context, limit int) ([]DIFALRule, error) {
	query := `
		SELECT
			id::text,
			COALESCE(code, ''),
			COALESCE(name, ''),
			COALESCE(uf, ''),
			priority,
			specificity_hint,
			COALESCE(status, ''),
			COALESCE(valid_from::text, ''),
			COALESCE(valid_to::text, ''),
			COALESCE(legal_basis_ids, '{}')
		FROM tax_rules
		WHERE tax_type = 'DIFAL'
		ORDER BY priority DESC, valid_from DESC, created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list difal rules: %w", err)
	}
	defer rows.Close()

	items := make([]DIFALRule, 0)
	for rows.Next() {
		var item DIFALRule
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Name,
			&item.UF,
			&item.Priority,
			&item.SpecificityHint,
			&item.Status,
			&item.ValidFrom,
			&item.ValidTo,
			&item.LegalBasisIDs,
		); err != nil {
			return nil, fmt.Errorf("scan difal rule: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate difal rules: %w", err)
	}

	if len(items) == 0 {
		return items, nil
	}

	ruleIDs := make([]string, 0, len(items))
	index := make(map[string]int, len(items))
	for i, item := range items {
		ruleIDs = append(ruleIDs, item.ID)
		index[item.ID] = i
	}

	if err := r.fillConditions(ctx, items, index, ruleIDs); err != nil {
		return nil, err
	}
	if err := r.fillActions(ctx, items, index, ruleIDs); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) fillConditions(ctx context.Context, items []DIFALRule, index map[string]int, ruleIDs []string) error {
	query := `
		SELECT
			rule_id::text,
			field_name,
			operator,
			COALESCE(value_text, ''),
			COALESCE(value_list, '{}')
		FROM tax_rule_conditions
		WHERE rule_id = ANY($1)
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, ruleIDs)
	if err != nil {
		return fmt.Errorf("list difal rule conditions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ruleID string
		var fieldName string
		var operator string
		var valueText string
		var valueList []string
		if err := rows.Scan(&ruleID, &fieldName, &operator, &valueText, &valueList); err != nil {
			return fmt.Errorf("scan difal rule condition: %w", err)
		}

		idx, ok := index[ruleID]
		if !ok {
			continue
		}

		switch strings.TrimSpace(fieldName) {
		case "issuer_uf":
			items[idx].IssuerUF = valueText
		case "recipient_uf":
			items[idx].RecipientUF = valueText
		case "operation_scope":
			items[idx].OperationScope = valueText
		case "operation_type":
			items[idx].OperationType = valueText
		case "crt":
			items[idx].CRT = valueText
		case "cfop":
			items[idx].CFOPPrefix = valueText
		case "ncm":
			items[idx].NCMPrefix = valueText
		case "final_consumer":
			if operator == "is_true" {
				items[idx].FinalConsumerMode = "yes"
			} else if operator == "is_false" {
				items[idx].FinalConsumerMode = "no"
			}
		case "recipient_contributor":
			if operator == "is_true" {
				items[idx].RecipientContributor = "yes"
			} else if operator == "is_false" {
				items[idx].RecipientContributor = "no"
			}
		}

		if len(valueList) > 0 && strings.TrimSpace(fieldName) == "crt" {
			items[idx].CRT = strings.Join(valueList, ", ")
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate difal rule conditions: %w", err)
	}

	return nil
}

func (r *Repository) fillActions(ctx context.Context, items []DIFALRule, index map[string]int, ruleIDs []string) error {
	query := `
		SELECT
			rule_id::text,
			target_field,
			COALESCE(value_text, ''),
			COALESCE(value_number::text, ''),
			value_bool
		FROM tax_rule_actions
		WHERE rule_id = ANY($1)
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, ruleIDs)
	if err != nil {
		return fmt.Errorf("list difal rule actions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ruleID string
		var target string
		var valueText string
		var valueNumber string
		var valueBool *bool
		if err := rows.Scan(&ruleID, &target, &valueText, &valueNumber, &valueBool); err != nil {
			return fmt.Errorf("scan difal rule action: %w", err)
		}

		idx, ok := index[ruleID]
		if !ok {
			continue
		}

		switch strings.TrimSpace(target) {
		case "internal_rate":
			items[idx].InternalRate = valueNumber
		case "interstate_rate":
			items[idx].InterstateRate = valueNumber
		case "fcp_rate":
			items[idx].FCPRate = valueNumber
		case "reason":
			items[idx].Reason = valueText
		case "applies":
			items[idx].Applies = valueBool != nil && *valueBool
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate difal rule actions: %w", err)
	}

	return nil
}

func (r *Repository) CreateDIFALRule(ctx context.Context, p CreateDIFALRuleParams) (string, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin difal rule transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	ruleID, err := r.insertRule(ctx, tx, p)
	if err != nil {
		return "", err
	}

	if err := r.insertConditions(ctx, tx, ruleID, p); err != nil {
		return "", err
	}

	if err := r.insertActions(ctx, tx, ruleID, p); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit difal rule transaction: %w", err)
	}

	return ruleID, nil
}

func (r *Repository) insertRule(ctx context.Context, tx pgx.Tx, p CreateDIFALRuleParams) (string, error) {
	query := `
		INSERT INTO tax_rules (
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
			legal_basis_ids
		)
		VALUES (
			$1,
			$2,
			'DIFAL',
			'STATE',
			NULLIF($3, ''),
			$4,
			$5,
			$6::date,
			NULLIF($7, '')::date,
			$8,
			$9
		)
		RETURNING id::text
	`

	var ruleID string
	err := tx.QueryRow(
		ctx,
		query,
		p.Code,
		p.Name,
		p.UF,
		p.Priority,
		p.SpecificityHint,
		p.ValidFrom,
		p.ValidTo,
		p.Status,
		p.LegalBasisIDs,
	).Scan(&ruleID)
	if err != nil {
		return "", fmt.Errorf("insert difal rule: %w", err)
	}

	return ruleID, nil
}

func (r *Repository) insertConditions(ctx context.Context, tx pgx.Tx, ruleID string, p CreateDIFALRuleParams) error {
	type conditionRow struct {
		field    string
		operator string
		text     string
		weight   int
	}

	rows := []conditionRow{
		{field: "operation_scope", operator: "eq", text: firstNonEmpty(strings.TrimSpace(p.OperationScope), "INTERSTATE"), weight: 90},
		{field: "operation_type", operator: "eq", text: firstNonEmpty(strings.TrimSpace(p.OperationType), "EXIT"), weight: 70},
	}

	if strings.TrimSpace(p.IssuerUF) != "" {
		rows = append(rows, conditionRow{field: "issuer_uf", operator: "eq", text: p.IssuerUF, weight: 100})
	}
	if strings.TrimSpace(p.RecipientUF) != "" {
		rows = append(rows, conditionRow{field: "recipient_uf", operator: "eq", text: p.RecipientUF, weight: 100})
	}
	if strings.TrimSpace(p.CRT) != "" {
		rows = append(rows, conditionRow{field: "crt", operator: "eq", text: p.CRT, weight: 45})
	}
	if strings.TrimSpace(p.CFOPPrefix) != "" {
		rows = append(rows, conditionRow{field: "cfop", operator: "prefix", text: p.CFOPPrefix, weight: 55})
	}
	if strings.TrimSpace(p.NCMPrefix) != "" {
		rows = append(rows, conditionRow{field: "ncm", operator: "prefix", text: p.NCMPrefix, weight: 40})
	}
	switch strings.TrimSpace(p.FinalConsumerMode) {
	case "yes":
		rows = append(rows, conditionRow{field: "final_consumer", operator: "is_true", weight: 80})
	case "no":
		rows = append(rows, conditionRow{field: "final_consumer", operator: "is_false", weight: 80})
	}
	switch strings.TrimSpace(p.RecipientContributor) {
	case "yes":
		rows = append(rows, conditionRow{field: "recipient_contributor", operator: "is_true", weight: 65})
	case "no":
		rows = append(rows, conditionRow{field: "recipient_contributor", operator: "is_false", weight: 65})
	}

	query := `
		INSERT INTO tax_rule_conditions (
			rule_id,
			field_name,
			operator,
			value_text,
			weight
		)
		VALUES ($1::uuid, $2, $3, NULLIF($4, ''), $5)
	`

	for _, row := range rows {
		if _, err := tx.Exec(ctx, query, ruleID, row.field, row.operator, row.text, row.weight); err != nil {
			return fmt.Errorf("insert difal rule condition %s: %w", row.field, err)
		}
	}

	return nil
}

func (r *Repository) insertActions(ctx context.Context, tx pgx.Tx, ruleID string, p CreateDIFALRuleParams) error {
	type actionRow struct {
		target string
		text   string
		number string
		boolV  *bool
	}

	applies := p.Applies
	rows := []actionRow{
		{target: "applies", boolV: &applies},
		{target: "internal_rate", number: p.InternalRate},
		{target: "interstate_rate", number: p.InterstateRate},
	}

	if strings.TrimSpace(p.FCPRate) != "" {
		rows = append(rows, actionRow{target: "fcp_rate", number: p.FCPRate})
	}
	if strings.TrimSpace(p.Reason) != "" {
		rows = append(rows, actionRow{target: "reason", text: p.Reason})
	}

	query := `
		INSERT INTO tax_rule_actions (
			rule_id,
			action_type,
			target_field,
			value_text,
			value_number,
			value_bool
		)
		VALUES (
			$1::uuid,
			'set',
			$2,
			NULLIF($3, ''),
			NULLIF($4, '')::numeric,
			$5
		)
	`

	for _, row := range rows {
		if _, err := tx.Exec(ctx, query, ruleID, row.target, row.text, row.number, row.boolV); err != nil {
			return fmt.Errorf("insert difal rule action %s: %w", row.target, err)
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
