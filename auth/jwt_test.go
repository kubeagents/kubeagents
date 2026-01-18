package auth

import (
	"testing"
	"time"
)

func TestJWTService_GenerateAccessToken(t *testing.T) {
	svc := NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)

	tests := []struct {
		name    string
		userID  string
		email   string
		wantErr bool
	}{
		{
			name:    "valid token generation",
			userID:  "user-123",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "empty user ID",
			userID:  "",
			email:   "test@example.com",
			wantErr: true,
		},
		{
			name:    "empty email",
			userID:  "user-123",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateAccessToken(tt.userID, tt.email)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if token == "" {
				t.Error("expected non-empty token")
			}
		})
	}
}

func TestJWTService_GenerateRefreshToken(t *testing.T) {
	svc := NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "valid token generation",
			userID:  "user-123",
			wantErr: false,
		},
		{
			name:    "empty user ID",
			userID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateRefreshToken(tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if token == "" {
				t.Error("expected non-empty token")
			}
		})
	}
}

func TestJWTService_ValidateAccessToken(t *testing.T) {
	svc := NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)

	// Generate a valid token
	validToken, err := svc.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantErr    bool
		wantUserID string
		wantEmail  string
	}{
		{
			name:       "valid token",
			token:      validToken,
			wantErr:    false,
			wantUserID: "user-123",
			wantEmail:  "test@example.com",
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := svc.ValidateAccessToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if claims.UserID != tt.wantUserID {
				t.Errorf("UserID = %v, want %v", claims.UserID, tt.wantUserID)
			}
			if claims.Email != tt.wantEmail {
				t.Errorf("Email = %v, want %v", claims.Email, tt.wantEmail)
			}
		})
	}
}

func TestJWTService_ValidateRefreshToken(t *testing.T) {
	svc := NewJWTService("test-secret-key-at-least-32-chars", 15*time.Minute, 7*24*time.Hour)

	// Generate a valid refresh token
	validToken, err := svc.GenerateRefreshToken("user-123")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantErr    bool
		wantUserID string
	}{
		{
			name:       "valid token",
			token:      validToken,
			wantErr:    false,
			wantUserID: "user-123",
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := svc.ValidateRefreshToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if claims.UserID != tt.wantUserID {
				t.Errorf("UserID = %v, want %v", claims.UserID, tt.wantUserID)
			}
		})
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	// Create service with very short expiry
	svc := NewJWTService("test-secret-key-at-least-32-chars", 1*time.Millisecond, 1*time.Millisecond)

	token, err := svc.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = svc.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestJWTService_WrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-key-1-at-least-32-chars!!", 15*time.Minute, 7*24*time.Hour)
	svc2 := NewJWTService("secret-key-2-different-secret!!!", 15*time.Minute, 7*24*time.Hour)

	token, err := svc1.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validating with different secret should fail
	_, err = svc2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}
