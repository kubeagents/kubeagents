package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kubeagents/kubeagents/auth"
)

type contextKey string

// UserContextKey is the key used to store user claims in request context
const UserContextKey contextKey = "user"

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtService *auth.JWTService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtService}
}

// RequireAuth is a middleware that requires a valid JWT token
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

		// Validate token
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
