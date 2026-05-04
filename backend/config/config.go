package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL      string
	JWTSigningSecret string
	Env              string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSigningSecret: os.Getenv("JWT_SIGNING_SECRET"),
		Env:              getEnv("ENV", "development"),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSigningSecret == "" {
		return nil, fmt.Errorf("JWT_SIGNING_SECRET is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
