package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "MySecurePassword123",
			wantErr:  false,
		},
		{
			name:     "minimum length password",
			password: "abcdefgh",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: "this_is_a_very_long_password_with_many_special_chars!@#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if hash == "" {
				t.Error("expected non-empty hash")
			}
			// Hash should be different from password
			if hash == tt.password {
				t.Error("hash should be different from password")
			}
			// Same password should produce different hashes (bcrypt includes salt)
			hash2, _ := HashPassword(tt.password)
			if hash == hash2 {
				t.Error("same password should produce different hashes due to salt")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "MySecurePassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		wantOK   bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			wantOK:   true,
		},
		{
			name:     "incorrect password",
			password: "WrongPassword",
			hash:     hash,
			wantOK:   false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			wantOK:   false,
		},
		{
			name:     "invalid hash",
			password: password,
			hash:     "invalid_hash",
			wantOK:   false,
		},
		{
			name:     "empty hash",
			password: password,
			hash:     "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := VerifyPassword(tt.password, tt.hash)
			if ok != tt.wantOK {
				t.Errorf("VerifyPassword() = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}
