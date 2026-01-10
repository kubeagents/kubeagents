package config

import (
	"os"
	"testing"
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
