package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kubeagents/kubeagents/auth"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

type contextKey string

// UserContextKey is the key used to store user claims in request context
const UserContextKey contextKey = "user"

// APIKeyContextKey is the key used to store API key ID in request context
const APIKeyContextKey contextKey = "api_key_id"

// AuthMiddleware handles JWT and API Key authentication
type AuthMiddleware struct {
	jwtService *auth.JWTService
	store      store.Store
}

// NewAuthMiddlewareWithStore creates a new authentication middleware with store for API key validation
func NewAuthMiddlewareWithStore(jwtService *auth.JWTService, st store.Store) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		store:      st,
	}
}

// RequireAuth is a middleware that requires a valid JWT token (for frontend API)
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondUnauthorized(w, "missing authorization header")
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondUnauthorized(w, "invalid authorization format")
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			respondUnauthorized(w, "missing token")
			return
		}

		// Validate JWT token
		claims, err := m.jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			respondUnauthorized(w, "invalid or expired token")
			return
		}

		// Add user claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthOrAPIKey is a middleware that accepts either JWT token or API key
// This is used for webhook endpoints that can be called by external tools
func (m *AuthMiddleware) RequireAuthOrAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondUnauthorized(w, "missing authorization header")
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondUnauthorized(w, "invalid authorization format")
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			respondUnauthorized(w, "missing token")
			return
		}

		// First try to validate as JWT token
		claims, err := m.jwtService.ValidateAccessToken(tokenString)
		if err == nil {
			// JWT token is valid
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If JWT validation failed and we have a store, try API key authentication
		if m.store != nil {
			if m.validateAPIKey(w, r, next, tokenString) {
				return
			}
		}

		respondUnauthorized(w, "invalid or expired token")
	})
}

// validateAPIKey validates an API key and sets the context if valid
func (m *AuthMiddleware) validateAPIKey(w http.ResponseWriter, r *http.Request, next http.Handler, keyString string) bool {
	if m.store == nil {
		return false
	}

	// Check key length (base64 encoded 32 bytes = 44 chars)
	if len(keyString) < 8 {
		return false
	}

	// Get the key prefix for quick lookup
	keyPrefix := keyString[:8]

	// Find API key by verifying against stored hashes
	apiKey := m.findAPIKeyByPrefixAndVerify(keyPrefix, keyString)
	if apiKey == nil {
		return false
	}

	// Check if key is valid
	if !apiKey.IsValid() {
		return false
	}

	// Get user info to create claims
	user, err := m.store.GetUserByID(apiKey.UserID)
	if err != nil {
		return false
	}

	// Update last used timestamp (async to not block request)
	go m.store.UpdateAPIKeyLastUsed(apiKey.ID)

	// Create claims for the user
	claims := &auth.AccessTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
	}

	// Add user claims and API key ID to context
	ctx := context.WithValue(r.Context(), UserContextKey, claims)
	ctx = context.WithValue(ctx, APIKeyContextKey, apiKey.ID)
	next.ServeHTTP(w, r.WithContext(ctx))
	return true
}

// findAPIKeyByPrefixAndVerify finds an API key by SHA256 hash lookup
func (m *AuthMiddleware) findAPIKeyByPrefixAndVerify(prefix, rawKey string) *models.APIKey {
	// Compute SHA256 hash of the raw key for lookup
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Look up by hash
	apiKey, err := m.store.GetAPIKeyByHash(keyHash)
	if err != nil {
		return nil
	}

	// Verify prefix matches (additional security check)
	if apiKey.KeyPrefix != prefix {
		return nil
	}

	return apiKey
}

// HashAPIKey computes SHA256 hash of an API key for storage
func HashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// GetUserFromContext retrieves user claims from request context
func GetUserFromContext(ctx context.Context) (*auth.AccessTokenClaims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*auth.AccessTokenClaims)
	return claims, ok
}

// respondUnauthorized sends a 401 response with error message
func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
