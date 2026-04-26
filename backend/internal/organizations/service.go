package organizations

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateOrganization(
	ctx context.Context,
	userID, name, cnpj, taxRegime, crt, stateRegistration, homeUF string,
) (*Organization, error) {
	organizationID, err := s.repo.CreateOrganization(
		ctx,
		name,
		cnpj,
		taxRegime,
		crt,
		stateRegistration,
		homeUF,
	)
	if err != nil {
		return nil, err
	}

	err = s.repo.AddUserToOrganization(ctx, userID, organizationID, "owner")
	if err != nil {
		return nil, err
	}

	return &Organization{
		ID:                organizationID,
		Name:              name,
		CNPJ:              cnpj,
		Role:              "owner",
		TaxRegime:         taxRegime,
		CRT:               crt,
		StateRegistration: stateRegistration,
		HomeUF:            homeUF,
	}, nil
}

func (s *Service) ListOrganizations(ctx context.Context, userID string) ([]Organization, error) {
	return s.repo.ListOrganizationsByUser(ctx, userID)
}

func (s *Service) UserBelongsToOrganization(ctx context.Context, userID, organizationID string) (bool, error) {
	return s.repo.UserBelongsToOrganization(ctx, userID, organizationID)
}

func (s *Service) GetOrganizationByID(ctx context.Context, organizationID string) (*Organization, error) {
	return s.repo.GetOrganizationByID(ctx, organizationID)
}

func (s *Service) UpdateOrganization(
	ctx context.Context,
	userID, organizationID, name, cnpj, taxRegime, crt, stateRegistration, homeUF string,
) (*Organization, error) {
	userID = strings.TrimSpace(userID)
	organizationID = strings.TrimSpace(organizationID)
	name = strings.TrimSpace(name)
	homeUF = strings.ToUpper(strings.TrimSpace(homeUF))

	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if organizationID == "" {
		return nil, errors.New("organization_id is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if homeUF != "" && len(homeUF) != 2 {
		return nil, errors.New("home_uf must contain 2 characters")
	}

	allowed, err := s.repo.UserBelongsToOrganization(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("user does not belong to organization")
	}

	return s.repo.UpdateOrganization(
		ctx,
		organizationID,
		name,
		strings.TrimSpace(cnpj),
		strings.TrimSpace(taxRegime),
		strings.TrimSpace(crt),
		strings.TrimSpace(stateRegistration),
		homeUF,
	)
}
