package fiscaloperations

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListActive(ctx context.Context) ([]FiscalOperation, error) {
	return s.repo.ListActive(ctx)
}

func (s *Service) ResolveOperation(ctx context.Context, code string) (*FiscalOperation, error) {
	if code == "" {
		return s.repo.FindDefault(ctx)
	}

	return s.repo.FindByCode(ctx, code)
}