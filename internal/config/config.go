package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv            string
	Port              string
	DatabaseURL       string
	JWTSecret         string
	FrontendOrigin    string
	RequestsPerMinute int
	JWTTTL            time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		FrontendOrigin:    os.Getenv("FRONTEND_ORIGIN"),
		RequestsPerMinute: getEnvInt("REQUESTS_PER_MINUTE", 10),
		JWTTTL:            8 * time.Hour,
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
