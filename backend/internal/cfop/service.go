package cfop

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

func (s *Service) List(ctx context.Context, q string, operationType string, limit int) ([]CFOP, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	return s.repo.List(ctx, strings.TrimSpace(q), normalizeOperationType(operationType), limit)
}

func (s *Service) FindByCode(ctx context.Context, code string) (*CFOP, error) {
	return s.repo.FindByCode(ctx, normalizeCode(code))
}

func normalizeCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func normalizeOperationType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "entrada":
		return "entrada"
	case "saida":
		return "saida"
	default:
		return ""
	}
}
