package models

import (
	"errors"
	"regexp"
	"time"
)

// User represents a system user
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"` // Never expose in JSON
	Name          string    `json:"name,omitempty"`
	EmailVerified bool      `json:"email_verified"`
	VerifyToken   string    `json:"-"` // Never expose in JSON
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Validate validates User fields
func (u *User) Validate() error {
	if u.ID == "" {
		return errors.New("id is required")
	}
	if len(u.ID) > 36 {
		return errors.New("id must be <= 36 characters")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	if !emailRegex.MatchString(u.Email) {
		return errors.New("invalid email format")
	}
	if len(u.Email) > 255 {
		return errors.New("email must be <= 255 characters")
	}
	if len(u.Name) > 200 {
		return errors.New("name must be <= 200 characters")
	}
	if u.PasswordHash == "" {
		return errors.New("password_hash is required")
	}
	return nil
}

// ValidatePassword validates password strength requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

// RefreshToken represents a refresh token for session management
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"` // Never expose in JSON
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

// Validate validates RefreshToken fields
func (rt *RefreshToken) Validate() error {
	if rt.ID == "" {
		return errors.New("id is required")
	}
	if rt.UserID == "" {
		return errors.New("user_id is required")
	}
	if rt.TokenHash == "" {
		return errors.New("token_hash is required")
	}
	if rt.ExpiresAt.IsZero() {
		return errors.New("expires_at is required")
	}
	return nil
}
