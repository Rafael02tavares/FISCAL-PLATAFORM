package fiscaloperations

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrOperationNotFound = errors.New("fiscal operation not found")
)

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
	code = strings.TrimSpace(code)

	var (
		op  *FiscalOperation
		err error
	)

	if code == "" {
		op, err = s.repo.FindDefault(ctx)
	} else {
		op, err = s.repo.FindByCode(ctx, code)
	}

	if err != nil {
		return nil, ErrOperationNotFound
	}

	if op == nil {
		return nil, ErrOperationNotFound
	}

	return op, nil
}