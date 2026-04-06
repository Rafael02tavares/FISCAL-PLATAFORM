package organizations

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidOrganizationData = errors.New("invalid organization data")
)

const ownerRole = "owner"

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
	userID = strings.TrimSpace(userID)
	name = strings.TrimSpace(name)
	cnpj = onlyDigits(cnpj)
	taxRegime = strings.TrimSpace(taxRegime)
	crt = strings.TrimSpace(crt)
	stateRegistration = strings.TrimSpace(stateRegistration)
	homeUF = strings.ToUpper(strings.TrimSpace(homeUF))

	if err := validateCreateOrganizationInput(userID, name, cnpj, homeUF); err != nil {
		return nil, err
	}

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

	if err := s.repo.AddUserToOrganization(ctx, userID, organizationID, ownerRole); err != nil {
		return nil, err
	}

	return &Organization{
		ID:                organizationID,
		Name:              name,
		CNPJ:              cnpj,
		Role:              ownerRole,
		TaxRegime:         taxRegime,
		CRT:               crt,
		StateRegistration: stateRegistration,
		HomeUF:            homeUF,
	}, nil
}

func (s *Service) ListOrganizations(ctx context.Context, userID string) ([]Organization, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidOrganizationData
	}

	return s.repo.ListOrganizationsByUser(ctx, userID)
}

func (s *Service) UserBelongsToOrganization(ctx context.Context, userID, organizationID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	organizationID = strings.TrimSpace(organizationID)

	if userID == "" || organizationID == "" {
		return false, ErrInvalidOrganizationData
	}

	return s.repo.UserBelongsToOrganization(ctx, userID, organizationID)
}

func validateCreateOrganizationInput(userID, name, cnpj, homeUF string) error {
	switch {
	case userID == "":
		return ErrInvalidOrganizationData
	case name == "":
		return ErrInvalidOrganizationData
	case len([]rune(name)) < 2:
		return ErrInvalidOrganizationData
	case len([]rune(name)) > 150:
		return ErrInvalidOrganizationData
	case cnpj != "" && len(cnpj) != 14:
		return ErrInvalidOrganizationData
	case homeUF != "" && len(homeUF) != 2:
		return ErrInvalidOrganizationData
	default:
		return nil
	}
}

func onlyDigits(value string) string {
	var b strings.Builder
	b.Grow(len(value))

	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	return b.String()
}