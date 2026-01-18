package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original values
	originalPort := os.Getenv("PORT")
	originalCORS := os.Getenv("CORS_ALLOWED_ORIGINS")

	// Clean up after test
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
		if originalCORS != "" {
			os.Setenv("CORS_ALLOWED_ORIGINS", originalCORS)
		} else {
			os.Unsetenv("CORS_ALLOWED_ORIGINS")
		}
	}()

	// Test default values
	os.Unsetenv("PORT")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Load() default port = %v, want 8080", cfg.Port)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
		t.Errorf("Load() default CORS = %v, want [*]", cfg.CORSAllowedOrigins)
	}

	// Test custom values
	os.Setenv("PORT", "9090")
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,https://example.com")

	cfg = Load()
	if cfg.Port != "9090" {
		t.Errorf("Load() custom port = %v, want 9090", cfg.Port)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("Load() custom CORS count = %v, want 2", len(cfg.CORSAllowedOrigins))
	}
	if cfg.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("Load() CORS[0] = %v, want http://localhost:5173", cfg.CORSAllowedOrigins[0])
	}
	if cfg.CORSAllowedOrigins[1] != "https://example.com" {
		t.Errorf("Load() CORS[1] = %v, want https://example.com", cfg.CORSAllowedOrigins[1])
	}
}

func TestLoad_NotificationWebhookURL(t *testing.T) {
	// Save original value
	originalURL := os.Getenv("NOTIFICATION_WEBHOOK_URL")
	defer func() {
		if originalURL != "" {
			os.Setenv("NOTIFICATION_WEBHOOK_URL", originalURL)
		} else {
			os.Unsetenv("NOTIFICATION_WEBHOOK_URL")
		}
	}()

	tests := []struct {
		name   string
		envVal string
		want   string
	}{
		{
			name:   "webhook URL set",
			envVal: "https://example.com/webhook",
			want:   "https://example.com/webhook",
		},
		{
			name:   "webhook URL empty",
			envVal: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv("NOTIFICATION_WEBHOOK_URL", tt.envVal)
			} else {
				os.Unsetenv("NOTIFICATION_WEBHOOK_URL")
			}

			cfg := Load()
			if cfg.NotificationWebhookURL != tt.want {
				t.Errorf("Load() NotificationWebhookURL = %v, want %v", cfg.NotificationWebhookURL, tt.want)
			}
		})
	}
}

func TestLoad_NotificationTimeout(t *testing.T) {
	// Save original value
	originalTimeout := os.Getenv("NOTIFICATION_TIMEOUT_SECONDS")
	defer func() {
		if originalTimeout != "" {
			os.Setenv("NOTIFICATION_TIMEOUT_SECONDS", originalTimeout)
		} else {
			os.Unsetenv("NOTIFICATION_TIMEOUT_SECONDS")
		}
	}()

	tests := []struct {
		name   string
		envVal string
		want   time.Duration
	}{
		{
			name:   "custom timeout",
			envVal: "10",
			want:   10 * time.Second,
		},
		{
			name:   "default timeout",
			envVal: "",
			want:   5 * time.Second,
		},
		{
			name:   "invalid timeout uses default",
			envVal: "invalid",
			want:   5 * time.Second,
		},
		{
			name:   "negative timeout uses default",
			envVal: "-5",
			want:   5 * time.Second,
		},
		{
			name:   "zero timeout uses default",
			envVal: "0",
			want:   5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv("NOTIFICATION_TIMEOUT_SECONDS", tt.envVal)
			} else {
				os.Unsetenv("NOTIFICATION_TIMEOUT_SECONDS")
			}

			cfg := Load()
			if cfg.NotificationTimeout != tt.want {
				t.Errorf("Load() NotificationTimeout = %v, want %v", cfg.NotificationTimeout, tt.want)
			}
		})
	}
}
