package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kubeagents/kubeagents/auth"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)
	middleware := NewAuthMiddlewareWithStore(jwtService, nil)

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
	middleware := NewAuthMiddlewareWithStore(jwtService, nil)

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

// TestHashAPIKey_VerifyCorrectAlgorithm tests that HashAPIKey uses SHA256
// This prevents regression where bcrypt or other algorithms might be used
func TestHashAPIKey_VerifyCorrectAlgorithm(t *testing.T) {
	rawKey := "test-api-key-12345678"

	// Hash the key using the same function used in production
	hash := HashAPIKey(rawKey)

	// Verify the hash is a SHA256 hash (64 hex characters)
	if len(hash) != 64 {
		t.Errorf("HashAPIKey() length = %v, want 64 (SHA256 hex length)", len(hash))
	}

	// Verify it's valid hex
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("HashAPIKey() returned invalid hex: %q", hash)
		}
	}

	// Verify the hash is deterministic (same input = same output)
	hash2 := HashAPIKey(rawKey)
	if hash != hash2 {
		t.Errorf("HashAPIKey() not deterministic, got different hashes: %q vs %q", hash, hash2)
	}
}

// TestFindAPIKeyByPrefixAndVerify_VerifyHashAlgorithm tests the full API key verification flow
// This ensures the hash used for verification matches the hash used for storage
func TestFindAPIKeyByPrefixAndVerify_VerifyHashAlgorithm(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)

	// Create a test store
	st := store.NewMemoryStore()

	// Create a test user with password hash
	user := &models.User{
		ID:           "test-user-123",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz123456", // Fake bcrypt hash
	}
	if err := st.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	middleware := NewAuthMiddlewareWithStore(jwtService, st)

	// Generate an API key using the production method
	rawKey := "OJBwmmPSTestApiKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	prefix := rawKey[:8]
	keyHash := HashAPIKey(rawKey)

	// Store the API key
	apiKey := &models.APIKey{
		ID:        "test-key-123",
		UserID:    user.ID,
		Name:      "test-key",
		KeyHash:   keyHash,
		KeyPrefix: prefix,
		Revoked:   false,
	}

	if err := st.CreateAPIKey(apiKey); err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Verify the key can be found using the production verification method
	foundKey := middleware.findAPIKeyByPrefixAndVerify(prefix, rawKey)

	if foundKey == nil {
		t.Fatalf("findAPIKeyByPrefixAndVerify() returned nil, expected to find the key")
	}

	if foundKey.ID != apiKey.ID {
		t.Errorf("findAPIKeyByPrefixAndVerify() returned key ID %q, want %q", foundKey.ID, apiKey.ID)
	}
}

// TestFindAPIKeyByPrefixAndVerify_RejectsWrongKey tests that verification fails for incorrect keys
func TestFindAPIKeyByPrefixAndVerify_RejectsWrongKey(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)
	st := store.NewMemoryStore()

	// Create a test user with password hash
	user := &models.User{
		ID:           "test-user-456",
		Email:        "test2@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz123456", // Fake bcrypt hash
	}
	if err := st.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	middleware := NewAuthMiddlewareWithStore(jwtService, st)

	// Generate an API key
	correctKey := "CorrectKeyValue1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	prefix := correctKey[:8]
	keyHash := HashAPIKey(correctKey)

	// Store the API key
	apiKey := &models.APIKey{
		ID:        "test-key-456",
		UserID:    user.ID,
		Name:      "test-key-2",
		KeyHash:   keyHash,
		KeyPrefix: prefix,
		Revoked:   false,
	}

	if err := st.CreateAPIKey(apiKey); err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Try to verify with wrong key (same prefix but different value)
	wrongKey := "CorrectKeyDIFFERENTVALUE7890ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	foundKey := middleware.findAPIKeyByPrefixAndVerify(prefix, wrongKey)

	if foundKey != nil {
		t.Errorf("findAPIKeyByPrefixAndVerify() found a key with wrong value, expected nil")
	}
}

// TestHashAPIKey_NotBcrypt tests that HashAPIKey does NOT use bcrypt
// Bcrypt hashes are variable length and contain special characters
func TestHashAPIKey_NotBcrypt(t *testing.T) {
	rawKey := "test-api-key-12345678"
	hash := HashAPIKey(rawKey)

	// Bcrypt hashes always start with "$2a$", "$2b$", or "$2y$" followed by cost and salt
	// SHA256 hashes are 64 hex characters
	if len(hash) < 60 {
		// Most bcrypt hashes are at least 60 characters
		t.Errorf("HashAPIKey() returned hash shorter than typical bcrypt: length=%d", len(hash))
	}

	// Check it doesn't start with bcrypt prefix
	if len(hash) >= 4 && hash[0:4] == "$2a$" || hash[0:4] == "$2b$" || hash[0:4] == "$2y$" {
		t.Errorf("HashAPIKey() appears to use bcrypt (starts with %q), should use SHA256", hash[0:4])
	}
}
