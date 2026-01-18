package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kubeagents/kubeagents/auth"
	"github.com/kubeagents/kubeagents/email"
	"github.com/kubeagents/kubeagents/middleware"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	store        store.Store
	jwtService   *auth.JWTService
	emailService *email.EmailService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(st store.Store, jwtService *auth.JWTService, emailService *email.EmailService) *AuthHandler {
	return &AuthHandler{
		store:        st,
		jwtService:   jwtService,
		emailService: emailService,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"` // seconds
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate password
	if err := models.ValidatePassword(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	// Generate verify token
	verifyToken, err := generateToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate verify token")
		return
	}

	now := time.Now()
	user := &models.User{
		ID:            uuid.New().String(),
		Email:         req.Email,
		PasswordHash:  passwordHash,
		Name:          req.Name,
		EmailVerified: false,
		VerifyToken:   verifyToken,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Validate user
	if err := user.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create user
	if err := h.store.CreateUser(user); err != nil {
		if err == store.ErrDuplicateEmail {
			respondError(w, http.StatusConflict, "email already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Send verification email (async, don't fail registration if email fails)
	if h.emailService != nil {
		go h.emailService.SendVerificationEmail(user.Email, verifyToken)
	}

	// Respond with success (no tokens until email is verified)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "注册成功，请检查邮箱完成验证",
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "missing token")
		return
	}

	// Find user by verify token
	user, err := h.store.GetUserByVerifyToken(token)
	if err != nil {
		if err == store.ErrNotFound {
			respondError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to verify token")
		return
	}

	// Update user to verified
	user.EmailVerified = true
	user.VerifyToken = ""
	user.UpdatedAt = time.Now()

	if err := h.store.UpdateUser(user); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to verify email")
		return
	}

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	// Save refresh token
	refreshTokenHash, _ := auth.HashPassword(refreshToken)
	rt := &models.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}
	h.store.SaveRefreshToken(rt)

	respondJSON(w, http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get user by email
	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		if err == store.ErrNotFound {
			respondError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	// Verify password
	if !auth.VerifyPassword(req.Password, user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// Check if email is verified
	if !user.EmailVerified {
		respondError(w, http.StatusForbidden, "email not verified")
		return
	}

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	// Save refresh token
	refreshTokenHash, _ := auth.HashPassword(refreshToken)
	rt := &models.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}
	h.store.SaveRefreshToken(rt)

	respondJSON(w, http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate refresh token
	claims, err := h.jwtService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Get user
	user, err := h.store.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "user not found")
		return
	}

	// Generate new tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	newRefreshToken, err := h.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	// Save new refresh token
	refreshTokenHash, _ := auth.HashPassword(newRefreshToken)
	rt := &models.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}
	h.store.SaveRefreshToken(rt)

	respondJSON(w, http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    900, // 15 minutes
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Revoke all user's refresh tokens
	h.store.RevokeAllUserTokens(claims.UserID)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// Me returns the current user's information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.store.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ResendVerify resends the verification email
func (h *AuthHandler) ResendVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get user by email
	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		// Don't reveal if email exists
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "如果该邮箱已注册，您将收到一封验证邮件",
		})
		return
	}

	// Check if already verified
	if user.EmailVerified {
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "如果该邮箱已注册，您将收到一封验证邮件",
		})
		return
	}

	// Generate new verify token
	verifyToken, err := generateToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate verify token")
		return
	}

	// Update user
	user.VerifyToken = verifyToken
	user.UpdatedAt = time.Now()
	h.store.UpdateUser(user)

	// Send verification email
	if h.emailService != nil {
		go h.emailService.SendVerificationEmail(user.Email, verifyToken)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "如果该邮箱已注册，您将收到一封验证邮件",
	})
}

// generateToken generates a random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// respondError sends an error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
