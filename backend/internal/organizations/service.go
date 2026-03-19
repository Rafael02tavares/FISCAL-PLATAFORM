package organizations

import "context"

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