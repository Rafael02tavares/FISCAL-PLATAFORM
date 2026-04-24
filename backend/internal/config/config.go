package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AllowedOrigins []string
}

func Load() (Config, error) {

	// carrega .env se existir
	_ = godotenv.Load()

	cfg := Config{
		AppEnv:         normalizeEnv(getEnv("APP_ENV", "development")),
		Port:           strings.TrimSpace(getEnv("APP_PORT", "8081")),
		DatabaseURL:    strings.TrimSpace(getEnv("DATABASE_URL", "")),
		JWTSecret:      strings.TrimSpace(getEnv("JWT_SECRET", "")),
		AllowedOrigins: parseCSVEnv(getEnv("ALLOWED_ORIGINS", "http://localhost:4321")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("APP_PORT is required")
	}

	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_ORIGINS must contain at least one origin")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func normalizeEnv(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "production", "prod":
		return "production"
	case "staging", "stage":
		return "staging"
	case "test", "testing":
		return "test"
	default:
		return "development"
	}
}

func parseCSVEnv(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		if _, exists := seen[origin]; exists {
			continue
		}

		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}

	return origins
}
