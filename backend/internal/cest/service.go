package cest

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

func (s *Service) List(ctx context.Context, q string, ncmCode string, limit int) ([]CEST, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	return s.repo.List(ctx, strings.TrimSpace(q), normalizeNCMCode(ncmCode), limit)
}

func (s *Service) FindByCode(ctx context.Context, code string) (*CEST, error) {
	return s.repo.FindByCode(ctx, normalizeCESTCode(code))
}

func normalizeCESTCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}

func normalizeNCMCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, " ", "")
	return v
}
