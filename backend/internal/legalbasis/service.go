package legalbasis

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidLegalSource = errors.New("invalid legal source")
	ErrInvalidLegalRule   = errors.New("invalid legal rule")
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateLegalSource(ctx context.Context, p CreateLegalSourceParams) (string, error) {
	p.normalize()

	if err := p.validate(); err != nil {
		return "", ErrInvalidLegalSource
	}

	return s.repo.CreateLegalSource(ctx, p)
}

func (s *Service) ListLegalSources(ctx context.Context, limit int) ([]LegalSource, error) {
	limit = normalizeLimit(limit)

	items, err := s.repo.ListLegalSources(ctx, limit)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []LegalSource{}, nil
	}

	return items, nil
}

func (s *Service) CreateLegalRuleMapping(ctx context.Context, p CreateLegalRuleMappingParams) (string, error) {
	p.normalize()

	if err := p.validate(); err != nil {
		return "", ErrInvalidLegalRule
	}

	if p.Priority == 0 {
		p.Priority = 100
	}

	if strings.TrimSpace(p.ConfidenceBase) == "" {
		p.ConfidenceBase = "0.70"
	}

	if strings.TrimSpace(p.ValueContent) == "" {
		p.ValueContent = "{}"
	}

	return s.repo.CreateLegalRuleMapping(ctx, p)
}

func (s *Service) ListLegalRuleMappings(ctx context.Context, limit int) ([]LegalRuleMapping, error) {
	limit = normalizeLimit(limit)

	items, err := s.repo.ListLegalRuleMappings(ctx, limit)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []LegalRuleMapping{}, nil
	}

	return items, nil
}

func (s *Service) FindApplicableRules(ctx context.Context, p FindApplicableRulesParams) ([]ApplicableLegalRule, error) {
	p.normalize()

	items, err := s.repo.FindApplicableRules(ctx, p)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []ApplicableLegalRule{}, nil
	}

	return items, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}