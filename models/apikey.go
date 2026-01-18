package models

import (
	"errors"
	"time"
)

// APIKey represents a long-lived API key for external integrations
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`          // Never expose in JSON, stored as hash
	KeyPrefix  string     `json:"key_prefix"` // First 8 chars for identification
	ExpiresAt  *time.Time `json:"expires_at"` // nil means never expires
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	Revoked    bool       `json:"revoked"`
}

// Validate validates APIKey fields
func (k *APIKey) Validate() error {
	if k.ID == "" {
		return errors.New("id is required")
	}
	if len(k.ID) > 36 {
		return errors.New("id must be <= 36 characters")
	}
	if k.UserID == "" {
		return errors.New("user_id is required")
	}
	if k.Name == "" {
		return errors.New("name is required")
	}
	if len(k.Name) > 100 {
		return errors.New("name must be <= 100 characters")
	}
	if k.KeyHash == "" {
		return errors.New("key_hash is required")
	}
	if k.KeyPrefix == "" {
		return errors.New("key_prefix is required")
	}
	if len(k.KeyPrefix) != 8 {
		return errors.New("key_prefix must be exactly 8 characters")
	}
	return nil
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid checks if the API key is valid (not revoked and not expired)
func (k *APIKey) IsValid() bool {
	return !k.Revoked && !k.IsExpired()
}
