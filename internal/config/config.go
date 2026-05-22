package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv            string
	Port              string
	DatabaseURL       string
	JWTSecret         string
	FrontendOrigin    string
	RequestsPerMinute int
	JWTTTL            time.Duration
	EnableWorkers     bool // New: control whether to start background workers
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		FrontendOrigin:    getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		RequestsPerMinute: getEnvInt("REQUESTS_PER_MINUTE", 10),

		JWTTTL:        getEnvDuration("JWT_TTL", 8*time.Hour),
		EnableWorkers: getEnvBool("ENABLE_WORKERS", true), // Default: enabled
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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
