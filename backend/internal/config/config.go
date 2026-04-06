package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func Load() (Config, error) {
	// Carrega o .env se existir. Em produção, as variáveis
	// devem vir do ambiente do sistema/container.
	_ = godotenv.Load()

	cfg := Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	validEnvs := map[string]bool{
		"development": true,
		"test":        true,
		"production":  true,
	}

	if !validEnvs[cfg.AppEnv] {
		return fmt.Errorf("APP_ENV must be one of: development, test, production")
	}

	if cfg.Port == "" {
		return fmt.Errorf("PORT is required")
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return fmt.Errorf("PORT must be a valid number")
	}

	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}