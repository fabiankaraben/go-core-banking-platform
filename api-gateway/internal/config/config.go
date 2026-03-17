package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the api-gateway.
type Config struct {
	HTTPPort           string
	AccountServiceURL  string
	TransferServiceURL string
	RedisAddr          string
	RateLimitRPM       int
	ZipkinURL          string
	AllowedOrigins     []string
}

// Load reads configuration from environment variables.
func Load() *Config {
	rateLimit, _ := strconv.Atoi(getEnv("RATE_LIMIT_RPM", "100"))
	return &Config{
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		AccountServiceURL:  getEnv("ACCOUNT_SERVICE_URL", "http://localhost:8081"),
		TransferServiceURL: getEnv("TRANSFER_SERVICE_URL", "http://localhost:8082"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RateLimitRPM:       rateLimit,
		ZipkinURL:          getEnv("ZIPKIN_URL", "http://localhost:9411/api/v2/spans"),
		AllowedOrigins:     strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
