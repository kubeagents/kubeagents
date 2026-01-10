package config

import (
	"os"
	"strings"
)

// Config holds application configuration
type Config struct {
	Port              string
	CORSAllowedOrigins []string
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
	
	return &Config{
		Port:              port,
		CORSAllowedOrigins: origins,
	}
}
