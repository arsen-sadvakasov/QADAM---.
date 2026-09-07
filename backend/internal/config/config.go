// Package config загружает конфигурацию приложения из переменных окружения.
package config

import (
	"os"
	"strconv"
)

// Config содержит все настройки backend-сервиса QADAM.
type Config struct {
	AppEnv string // development | staging | production
	Port   string

	DatabaseURL    string
	MigrationsPath string

	JWTAccessSecret  string
	JWTRefreshSecret string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	RedisURL string
}

// Load читает конфигурацию из переменных окружения, применяя разумные значения
// по умолчанию для локальной разработки.
func Load() Config {
	return Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("PORT", "8080"),

		DatabaseURL:    getEnv("DATABASE_URL", "postgres://qadam:qadam@localhost:5432/qadam?sslmode=disable"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),

		JWTAccessSecret:  getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "qadam"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "qadam12345"),
		MinioBucket:    getEnv("MINIO_BUCKET", "qadam-materials"),
		MinioUseSSL:    getEnvBool("MINIO_USE_SSL", false),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
