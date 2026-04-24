package adminpartilha

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListDIFALRules(ctx context.Context, limit int) ([]DIFALRule, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}

	return s.repo.ListDIFALRules(ctx, limit)
}

func (s *Service) CreateDIFALRule(ctx context.Context, p CreateDIFALRuleParams) (string, error) {
	p.Code = strings.TrimSpace(strings.ToUpper(p.Code))
	p.Name = strings.TrimSpace(p.Name)
	p.UF = normalizeUF(p.UF)
	p.IssuerUF = normalizeUF(p.IssuerUF)
	p.RecipientUF = normalizeUF(p.RecipientUF)
	p.OperationScope = strings.TrimSpace(strings.ToUpper(p.OperationScope))
	p.OperationType = strings.TrimSpace(strings.ToUpper(p.OperationType))
	p.FinalConsumerMode = strings.TrimSpace(strings.ToLower(p.FinalConsumerMode))
	p.RecipientContributor = strings.TrimSpace(strings.ToLower(p.RecipientContributor))
	p.CRT = strings.TrimSpace(p.CRT)
	p.CFOPPrefix = onlyDigits(strings.TrimSpace(p.CFOPPrefix))
	p.NCMPrefix = onlyDigits(strings.TrimSpace(p.NCMPrefix))
	p.Status = strings.TrimSpace(strings.ToUpper(p.Status))
	p.ValidFrom = strings.TrimSpace(p.ValidFrom)
	p.ValidTo = strings.TrimSpace(p.ValidTo)
	p.InternalRate = normalizeDecimalInput(p.InternalRate)
	p.InterstateRate = normalizeDecimalInput(p.InterstateRate)
	p.FCPRate = normalizeDecimalInput(p.FCPRate)
	p.Reason = strings.TrimSpace(p.Reason)
	p.LegalBasisIDs = normalizeStringList(p.LegalBasisIDs)

	if p.Name == "" {
		return "", errors.New("name is required")
	}
	if p.ValidFrom == "" {
		return "", errors.New("valid_from is required")
	}
	if _, err := time.Parse("2006-01-02", p.ValidFrom); err != nil {
		return "", errors.New("valid_from must be in YYYY-MM-DD format")
	}
	if p.ValidTo != "" {
		if _, err := time.Parse("2006-01-02", p.ValidTo); err != nil {
			return "", errors.New("valid_to must be in YYYY-MM-DD format")
		}
	}
	if p.OperationScope == "" {
		p.OperationScope = "INTERSTATE"
	}
	if p.OperationType == "" {
		p.OperationType = "EXIT"
	}
	if p.Status == "" {
		p.Status = "ACTIVE"
	}
	if p.Status != "ACTIVE" && p.Status != "INACTIVE" && p.Status != "DRAFT" {
		return "", errors.New("status must be ACTIVE, INACTIVE or DRAFT")
	}
	if p.FinalConsumerMode != "" && p.FinalConsumerMode != "yes" && p.FinalConsumerMode != "no" && p.FinalConsumerMode != "any" {
		return "", errors.New("final_consumer_mode must be yes, no or any")
	}
	if p.RecipientContributor != "" && p.RecipientContributor != "yes" && p.RecipientContributor != "no" && p.RecipientContributor != "any" {
		return "", errors.New("recipient_contributor must be yes, no or any")
	}
	if p.InternalRate == "" {
		return "", errors.New("internal_rate is required")
	}
	if p.InterstateRate == "" {
		return "", errors.New("interstate_rate is required")
	}
	if p.Code == "" {
		p.Code = buildRuleCode(p)
	}
	if p.Priority == 0 {
		p.Priority = 100
	}
	p.SpecificityHint = calculateSpecificityHint(p)

	id, err := s.repo.CreateDIFALRule(ctx, p)
	if err != nil {
		return "", fmt.Errorf("create difal rule: %w", err)
	}

	return id, nil
}

func normalizeUF(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if len(value) != 2 {
		return ""
	}
	return value
}

func onlyDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeDecimalInput(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", ".")
	return value
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func buildRuleCode(p CreateDIFALRuleParams) string {
	issuer := firstNonEmptyString(p.IssuerUF, "XX")
	recipient := firstNonEmptyString(p.RecipientUF, p.UF, "XX")
	datePart := strings.ReplaceAll(p.ValidFrom, "-", "")
	return fmt.Sprintf("DIFAL_%s_%s_%s", issuer, recipient, datePart)
}

func calculateSpecificityHint(p CreateDIFALRuleParams) int {
	score := 10
	if p.IssuerUF != "" {
		score += 25
	}
	if p.RecipientUF != "" {
		score += 25
	}
	if p.CRT != "" {
		score += 10
	}
	if p.CFOPPrefix != "" {
		score += 10
	}
	if p.NCMPrefix != "" {
		score += 10
	}
	if p.FinalConsumerMode == "yes" || p.FinalConsumerMode == "no" {
		score += 8
	}
	if p.RecipientContributor == "yes" || p.RecipientContributor == "no" {
		score += 8
	}
	return score
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
