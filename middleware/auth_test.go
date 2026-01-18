package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kubeagents/kubeagents/auth"
)

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)
	middleware := NewAuthMiddleware(jwtService)

	// Generate a valid token
	validToken, _ := jwtService.GenerateAccessToken("user-123", "test@example.com")

	tests := []struct {
		name           string
		authHeader     string
		wantStatusCode int
		wantUserID     string
	}{
		{
			name:           "valid bearer token",
			authHeader:     "Bearer " + validToken,
			wantStatusCode: http.StatusOK,
			wantUserID:     "user-123",
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid format - no bearer",
			authHeader:     validToken,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid format - wrong prefix",
			authHeader:     "Basic " + validToken,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "empty token",
			authHeader:     "Bearer ",
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that checks for user in context
			var gotUserID string
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, ok := GetUserFromContext(r.Context())
				if ok {
					gotUserID = claims.UserID
				}
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with auth middleware
			handler := middleware.RequireAuth(testHandler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("status code = %v, want %v", rr.Code, tt.wantStatusCode)
			}

			// Check user ID if expected
			if tt.wantUserID != "" && gotUserID != tt.wantUserID {
				t.Errorf("userID = %v, want %v", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Create service with very short expiry
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-chars", 1*time.Millisecond, 7*24*time.Hour)
	middleware := NewAuthMiddleware(jwtService)

	token, _ := jwtService.GenerateAccessToken("user-123", "test@example.com")

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Create request with expired token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAuth(testHandler)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %v", rr.Code)
	}
}

func TestGetUserFromContext_NoUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	claims, ok := GetUserFromContext(req.Context())

	if ok {
		t.Error("expected ok to be false")
	}
	if claims != nil {
		t.Error("expected claims to be nil")
	}
}
