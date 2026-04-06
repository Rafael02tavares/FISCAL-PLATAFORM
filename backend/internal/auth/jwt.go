package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

const tokenTTL = 24 * time.Hour

type JWT struct {
	Secret string
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWT(secret string) *JWT {
	return &JWT{Secret: secret}
}

func (j *JWT) Generate(userID string) (string, error) {
	if j.Secret == "" {
		return "", fmt.Errorf("jwt secret is required")
	}

	if userID == "" {
		return "", fmt.Errorf("user id is required")
	}

	now := time.Now()

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(j.Secret))
}

func (j *JWT) Parse(tokenString string) (string, error) {
	if j.Secret == "" {
		return "", fmt.Errorf("jwt secret is required")
	}

	if tokenString == "" {
		return "", ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method == nil {
			return nil, ErrInvalidToken
		}

		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}

		return []byte(j.Secret), nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	if claims.UserID == "" {
		return "", ErrInvalidToken
	}

	return claims.UserID, nil
}