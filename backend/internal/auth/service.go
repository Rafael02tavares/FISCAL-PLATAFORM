package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRegisterData = errors.New("invalid register data")
	ErrInvalidLoginData    = errors.New("invalid login data")
	ErrEmailAlreadyExists  = errors.New("email already exists")
)

const (
	minPasswordLength = 8
	maxNameLength     = 120
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, name, email, password string) error {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if err := validateRegisterInput(name, email, password); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.repo.CreateUser(ctx, name, email, string(hash)); err != nil {
		if isUniqueViolation(err) {
			return ErrEmailAlreadyExists
		}
		return err
	}

	return nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if err := validateLoginInput(email, password); err != nil {
		return "", err
	}

	id, _, hash, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return id, nil
}

func validateRegisterInput(name, email, password string) error {
	if name == "" || email == "" || password == "" {
		return ErrInvalidRegisterData
	}

	if len(name) > maxNameLength {
		return ErrInvalidRegisterData
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidRegisterData
	}

	if len(password) < minPasswordLength {
		return ErrInvalidRegisterData
	}

	return nil
}

func validateLoginInput(email, password string) error {
	if email == "" || password == "" {
		return ErrInvalidLoginData
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidLoginData
	}

	return nil
}

// isUniqueViolation deve ser ajustada conforme a forma como o repositório
// retorna os erros do PostgreSQL.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "already exists")
}