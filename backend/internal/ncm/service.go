package ncm

import (
	"context"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, limit int) ([]NCM, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	return s.repo.List(ctx, limit)
}

func (s *Service) FindByCode(ctx context.Context, code string) (*NCM, error) {
	code = normalizeCode(code)
	return s.repo.FindByCode(ctx, code)
}

func (s *Service) Search(ctx context.Context, q string, limit int) ([]NCM, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return s.repo.Search(ctx, q, limit)
}

func normalizeCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}