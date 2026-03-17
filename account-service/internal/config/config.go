package config

import (
	"os"
	"strings"
)

// Config holds all configuration values for the account-service.
type Config struct {
	HTTPPort       string
	DBDSN          string
	KafkaBrokers   []string
	KafkaGroupID   string
	ZipkinURL      string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		HTTPPort:     getEnv("HTTP_PORT", "8081"),
		DBDSN:        getEnv("DB_DSN", "postgres://account_user:account_pass@localhost:5432/account_db?sslmode=disable"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaGroupID: getEnv("KAFKA_CONSUMER_GROUP", "account-service-group"),
		ZipkinURL:    getEnv("ZIPKIN_URL", "http://localhost:9411/api/v2/spans"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
