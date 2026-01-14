package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Config holds application configuration
type Config struct {
	Port                   string
	CORSAllowedOrigins     []string
	NotificationWebhookURL string
	NotificationTimeout    time.Duration
	Database               DatabaseConfig
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}

	origins := strings.Split(corsOrigins, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}

	// Notification webhook URL
	notificationWebhookURL := os.Getenv("NOTIFICATION_WEBHOOK_URL")

	// Notification timeout (default 5 seconds)
	notificationTimeout := 5 * time.Second
	if timeoutStr := os.Getenv("NOTIFICATION_TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
			notificationTimeout = time.Duration(timeout) * time.Second
		}
	}

	// Database configuration
	dbConfig := DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", ""),
		Password:        getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", ""),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
	}

	return &Config{
		Port:                   port,
		CORSAllowedOrigins:     origins,
		NotificationWebhookURL: notificationWebhookURL,
		NotificationTimeout:    notificationTimeout,
		Database:               dbConfig,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil && value > 0 {
			return value
		}
	}
	return defaultValue
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	value, _ := time.ParseDuration(defaultValue)
	return value
}
