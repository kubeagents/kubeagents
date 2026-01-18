package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kubeagents/kubeagents/middleware"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

// APIKeyHandler handles API key management endpoints
type APIKeyHandler struct {
	store store.Store
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(st store.Store) *APIKeyHandler {
	return &APIKeyHandler{
		store: st,
	}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name      string `json:"name"`
	ExpiresIn *int   `json:"expires_in,omitempty"` // days, nil means never expires
}

// CreateAPIKeyResponse represents the response when creating an API key
// The raw key is only returned once at creation time
type CreateAPIKeyResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"`        // Raw key, only shown once
	KeyPrefix string     `json:"key_prefix"` // First 8 chars for identification
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// APIKeyInfo represents API key information (without the raw key)
type APIKeyInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	Revoked    bool       `json:"revoked"`
}

// Create handles API key creation
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate name
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Name) > 100 {
		respondError(w, http.StatusBadRequest, "name must be <= 100 characters")
		return
	}

	// Generate random API key (32 bytes = 256 bits)
	rawKey, err := generateAPIKey()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	// Hash the key for storage using SHA256 (allows fast lookup)
	keyHash := middleware.HashAPIKey(rawKey)

	// Calculate expiration if provided
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &exp
	}

	now := time.Now()
	apiKey := &models.APIKey{
		ID:        uuid.New().String(),
		UserID:    claims.UserID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: rawKey[:8],
		ExpiresAt: expiresAt,
		CreatedAt: now,
		Revoked:   false,
	}

	// Validate and save
	if err := apiKey.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.CreateAPIKey(apiKey); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	// Return response with raw key (only shown once)
	respondJSON(w, http.StatusCreated, CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       rawKey,
		KeyPrefix: apiKey.KeyPrefix,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
	})
}

// List handles listing API keys for the current user
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	keys, err := h.store.ListAPIKeysByUser(claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}

	// Convert to response format (without key hash)
	result := make([]APIKeyInfo, 0, len(keys))
	for _, key := range keys {
		result = append(result, APIKeyInfo{
			ID:         key.ID,
			Name:       key.Name,
			KeyPrefix:  key.KeyPrefix,
			ExpiresAt:  key.ExpiresAt,
			LastUsedAt: key.LastUsedAt,
			CreatedAt:  key.CreatedAt,
			Revoked:    key.Revoked,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_keys": result,
	})
}

// Revoke handles revoking an API key
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	keyID := chi.URLParam(r, "id")
	if keyID == "" {
		respondError(w, http.StatusBadRequest, "missing key id")
		return
	}

	// Get the key to verify ownership
	apiKey, err := h.store.GetAPIKeyByID(keyID)
	if err != nil {
		if err == store.ErrNotFound {
			respondError(w, http.StatusNotFound, "API key not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get API key")
		return
	}

	// Verify ownership
	if apiKey.UserID != claims.UserID {
		respondError(w, http.StatusNotFound, "API key not found")
		return
	}

	// Revoke the key
	if err := h.store.RevokeAPIKey(keyID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to revoke API key")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}

// generateAPIKey generates a random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding
	return base64.URLEncoding.EncodeToString(bytes), nil
}
