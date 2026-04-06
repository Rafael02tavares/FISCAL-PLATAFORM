package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func AuthMiddleware(jwtService *JWT, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "header de autorização é obrigatório")
			return
		}

		tokenType, tokenString, ok := strings.Cut(authHeader, " ")
		if !ok || tokenType != "Bearer" || strings.TrimSpace(tokenString) == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "header de autorização inválido")
			return
		}

		userID, err := jwtService.Parse(strings.TrimSpace(tokenString))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "token inválido")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDContextKey).(string)
	return userID
}