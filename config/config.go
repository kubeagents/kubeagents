package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration
type Config struct {
	Port                   string
	CORSAllowedOrigins     []string
	NotificationWebhookURL string
	NotificationTimeout    time.Duration
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

	return &Config{
		Port:                   port,
		CORSAllowedOrigins:     origins,
		NotificationWebhookURL: notificationWebhookURL,
		NotificationTimeout:    notificationTimeout,
	}
}
