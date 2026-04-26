package fiscaloperations

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

func (s *Service) ListActive(ctx context.Context) ([]FiscalOperation, error) {
	return s.repo.ListActive(ctx)
}

func (s *Service) ResolveOperation(ctx context.Context, code string) (*FiscalOperation, error) {
	if code == "" {
		operation, err := s.repo.FindDefault(ctx)
		if err == nil {
			return operation, nil
		}
		return fallbackOperation("sale_consumer_final"), nil
	}

	operation, err := s.repo.FindByCode(ctx, code)
	if err == nil {
		return operation, nil
	}

	if fallback := fallbackOperation(code); fallback != nil {
		return fallback, nil
	}

	return nil, err
}

func fallbackOperation(code string) *FiscalOperation {
	switch strings.TrimSpace(code) {
	case "", "sale_consumer_final":
		return &FiscalOperation{
			Code:        "sale_consumer_final",
			Name:        "Venda a consumidor final",
			Direction:   "outbound",
			DefaultCFOP: "5102",
			IsDefault:   true,
			Active:      true,
		}
	case "sale_st_internal":
		return &FiscalOperation{
			Code:        "sale_st_internal",
			Name:        "Venda interna de mercadoria sujeita a ST",
			Direction:   "saida",
			DefaultCFOP: "5405",
			Active:      true,
		}
	case "sale_st_interstate":
		return &FiscalOperation{
			Code:        "sale_st_interstate",
			Name:        "Venda interestadual de mercadoria sujeita a ST",
			Direction:   "saida",
			DefaultCFOP: "6404",
			Active:      true,
		}
	case "sale_interstate":
		return &FiscalOperation{
			Code:        "sale_interstate",
			Name:        "Venda interestadual",
			Direction:   "saida",
			DefaultCFOP: "6102",
			Active:      true,
		}
	default:
		return nil
	}
}
