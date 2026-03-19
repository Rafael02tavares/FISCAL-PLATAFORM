package legalbasis

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateLegalSource(ctx context.Context, p CreateLegalSourceParams) (string, error) {
	return s.repo.CreateLegalSource(ctx, p)
}

func (s *Service) ListLegalSources(ctx context.Context, limit int) ([]LegalSource, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

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
	if p.Priority == 0 {
		p.Priority = 100
	}
	if p.ConfidenceBase == "" {
		p.ConfidenceBase = "0.70"
	}
	if p.ValueContent == "" {
		p.ValueContent = "{}"
	}
	return s.repo.CreateLegalRuleMapping(ctx, p)
}

func (s *Service) ListLegalRuleMappings(ctx context.Context, limit int) ([]LegalRuleMapping, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

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
	items, err := s.repo.FindApplicableRules(ctx, p)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []ApplicableLegalRule{}, nil
	}

	return items, nil
}