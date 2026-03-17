package config

import (
	"os"
	"strings"
)

// Config holds all configuration values for the transfer-service.
type Config struct {
	HTTPPort     string
	DBDSN        string
	RedisAddr    string
	KafkaBrokers []string
	KafkaGroupID string
	ZipkinURL    string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		HTTPPort:     getEnv("HTTP_PORT", "8082"),
		DBDSN:        getEnv("DB_DSN", "postgres://transfer_user:transfer_pass@localhost:5433/transfer_db?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaGroupID: getEnv("KAFKA_CONSUMER_GROUP", "transfer-service-group"),
		ZipkinURL:    getEnv("ZIPKIN_URL", "http://localhost:9411/api/v2/spans"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
