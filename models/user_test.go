package models

import (
	"testing"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user with all fields",
			user: User{
				ID:            "uuid-123",
				Email:         "test@example.com",
				PasswordHash:  "hashed_password",
				Name:          "Test User",
				EmailVerified: false,
			},
			wantErr: false,
		},
		{
			name: "valid user without name",
			user: User{
				ID:            "uuid-456",
				Email:         "user@example.com",
				PasswordHash:  "hashed_password",
				EmailVerified: true,
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			user: User{
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name: "empty email",
			user: User{
				ID:           "uuid-123",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "invalid email format - missing @",
			user: User{
				ID:           "uuid-123",
				Email:        "invalid-email",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "invalid email format - missing domain",
			user: User{
				ID:           "uuid-123",
				Email:        "test@",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "invalid email format - missing TLD",
			user: User{
				ID:           "uuid-123",
				Email:        "test@domain",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "email too long",
			user: User{
				ID:           "uuid-123",
				Email:        "very_long_email_address_that_exceeds_the_maximum_allowed_length_of_255_characters_abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789@example.com",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "email must be <= 255 characters",
		},
		{
			name: "name too long",
			user: User{
				ID:           "uuid-123",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Name:         "This is a very long name that exceeds the maximum allowed length of 200 characters abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789",
			},
			wantErr: true,
			errMsg:  "name must be <= 200 characters",
		},
		{
			name: "empty password hash",
			user: User{
				ID:    "uuid-123",
				Email: "test@example.com",
			},
			wantErr: true,
			errMsg:  "password_hash is required",
		},
		{
			name: "ID too long",
			user: User{
				ID:           "this_is_a_very_long_id_that_exceeds_36_characters",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
			errMsg:  "id must be <= 36 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password - 8 characters",
			password: "abcdefgh",
			wantErr:  false,
		},
		{
			name:     "valid password - long",
			password: "this_is_a_very_long_password_with_many_characters",
			wantErr:  false,
		},
		{
			name:     "password too short - 7 characters",
			password: "1234567",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters",
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
