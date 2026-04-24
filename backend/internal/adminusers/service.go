package adminusers

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidUserData = errors.New("invalid user data")
	ErrInvalidRole     = errors.New("invalid role")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByOrganization(ctx context.Context, organizationID string) ([]OrganizationUser, error) {
	return s.repo.ListByOrganization(ctx, strings.TrimSpace(organizationID))
}

type CreateOrAttachUserParams struct {
	OrganizationID string
	Name           string
	Email          string
	Password       string
	Role           string
}

func (s *Service) CreateOrAttachUser(ctx context.Context, params CreateOrAttachUserParams) error {
	role := normalizeRole(params.Role)
	if role == "" {
		return ErrInvalidRole
	}

	email := strings.ToLower(strings.TrimSpace(params.Email))
	name := strings.TrimSpace(params.Name)
	password := strings.TrimSpace(params.Password)
	organizationID := strings.TrimSpace(params.OrganizationID)

	if organizationID == "" || email == "" {
		return ErrInvalidUserData
	}

	existing, err := s.repo.FindUserByEmail(ctx, email)
	switch {
	case err == nil && existing != nil:
		return s.repo.AddUserToOrganization(ctx, existing.ID, organizationID, role)
	case err != nil && !errors.Is(err, ErrUserNotFound):
		return err
	}

	if name == "" || len([]rune(name)) < 2 || len(password) < 6 {
		return ErrInvalidUserData
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	userID, err := s.repo.CreateUser(ctx, name, email, string(hash))
	if err != nil {
		return err
	}

	return s.repo.AddUserToOrganization(ctx, userID, organizationID, role)
}

func (s *Service) UpdateRole(ctx context.Context, organizationID, userID, role string) error {
	role = normalizeRole(role)
	if role == "" {
		return ErrInvalidRole
	}

	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(userID) == "" {
		return ErrInvalidUserData
	}

	return s.repo.UpdateRole(ctx, strings.TrimSpace(userID), strings.TrimSpace(organizationID), role)
}

func (s *Service) RemoveFromOrganization(ctx context.Context, organizationID, userID string) error {
	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(userID) == "" {
		return ErrInvalidUserData
	}

	return s.repo.RemoveFromOrganization(ctx, strings.TrimSpace(userID), strings.TrimSpace(organizationID))
}

func normalizeRole(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "owner":
		return "owner"
	case "admin":
		return "admin"
	case "analyst", "analista":
		return "analyst"
	case "viewer", "leitor":
		return "viewer"
	default:
		return ""
	}
}
