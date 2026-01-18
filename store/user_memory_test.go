package store

import (
	"testing"
	"time"

	"github.com/kubeagents/kubeagents/models"
)

func TestMemoryStore_CreateUser(t *testing.T) {
	st := NewMemoryStore()

	tests := []struct {
		name    string
		user    *models.User
		wantErr bool
		errType error
	}{
		{
			name: "valid user",
			user: &models.User{
				ID:           "user-1",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Name:         "Test User",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "duplicate email",
			user: &models.User{
				ID:           "user-2",
				Email:        "test@example.com", // Same email
				PasswordHash: "hashed_password",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
			errType: ErrDuplicateEmail,
		},
		{
			name: "invalid user - missing email",
			user: &models.User{
				ID:           "user-3",
				PasswordHash: "hashed_password",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := st.CreateUser(tt.user)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("error = %v, want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMemoryStore_GetUserByID(t *testing.T) {
	st := NewMemoryStore()

	user := &models.User{
		ID:           "user-1",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = st.CreateUser(user)

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "existing user",
			userID:  "user-1",
			wantErr: false,
		},
		{
			name:    "non-existing user",
			userID:  "user-999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetUserByID(tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if got.ID != tt.userID {
					t.Errorf("ID = %v, want %v", got.ID, tt.userID)
				}
			}
		})
	}
}

func TestMemoryStore_GetUserByEmail(t *testing.T) {
	st := NewMemoryStore()

	user := &models.User{
		ID:           "user-1",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = st.CreateUser(user)

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "existing email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "non-existing email",
			email:   "notfound@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetUserByEmail(tt.email)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if got.Email != tt.email {
					t.Errorf("Email = %v, want %v", got.Email, tt.email)
				}
			}
		})
	}
}

func TestMemoryStore_GetUserByVerifyToken(t *testing.T) {
	st := NewMemoryStore()

	user := &models.User{
		ID:           "user-1",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		VerifyToken:  "verify-token-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = st.CreateUser(user)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "existing token",
			token:   "verify-token-123",
			wantErr: false,
		},
		{
			name:    "non-existing token",
			token:   "invalid-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetUserByVerifyToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if got.VerifyToken != tt.token {
					t.Errorf("VerifyToken = %v, want %v", got.VerifyToken, tt.token)
				}
			}
		})
	}
}

func TestMemoryStore_UpdateUser(t *testing.T) {
	st := NewMemoryStore()

	user := &models.User{
		ID:           "user-1",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = st.CreateUser(user)

	// Create another user for duplicate email test
	user2 := &models.User{
		ID:           "user-2",
		Email:        "other@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = st.CreateUser(user2)

	tests := []struct {
		name    string
		user    *models.User
		wantErr bool
	}{
		{
			name: "update name",
			user: &models.User{
				ID:            "user-1",
				Email:         "test@example.com",
				PasswordHash:  "hashed_password",
				Name:          "Updated Name",
				EmailVerified: true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "update email to existing",
			user: &models.User{
				ID:           "user-1",
				Email:        "other@example.com", // Already exists
				PasswordHash: "hashed_password",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "update non-existing user",
			user: &models.User{
				ID:           "user-999",
				Email:        "new@example.com",
				PasswordHash: "hashed_password",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := st.UpdateUser(tt.user)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMemoryStore_RefreshTokens(t *testing.T) {
	st := NewMemoryStore()

	// Create user first
	user := &models.User{
		ID:           "user-1",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = st.CreateUser(user)

	// Test SaveRefreshToken
	token := &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: "hash-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	err := st.SaveRefreshToken(token)
	if err != nil {
		t.Fatalf("SaveRefreshToken failed: %v", err)
	}

	// Test GetRefreshToken
	got, err := st.GetRefreshToken("hash-123")
	if err != nil {
		t.Fatalf("GetRefreshToken failed: %v", err)
	}
	if got.ID != "token-1" {
		t.Errorf("ID = %v, want token-1", got.ID)
	}

	// Test GetRefreshToken - not found
	_, err = st.GetRefreshToken("invalid-hash")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Test RevokeRefreshToken
	err = st.RevokeRefreshToken("hash-123")
	if err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}
	got, _ = st.GetRefreshToken("hash-123")
	if !got.Revoked {
		t.Error("token should be revoked")
	}

	// Test RevokeAllUserTokens
	token2 := &models.RefreshToken{
		ID:        "token-2",
		UserID:    "user-1",
		TokenHash: "hash-456",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}
	_ = st.SaveRefreshToken(token2)

	err = st.RevokeAllUserTokens("user-1")
	if err != nil {
		t.Fatalf("RevokeAllUserTokens failed: %v", err)
	}
	got2, _ := st.GetRefreshToken("hash-456")
	if !got2.Revoked {
		t.Error("token2 should be revoked")
	}
}

func TestMemoryStore_ListAgentsByUser(t *testing.T) {
	st := NewMemoryStore()

	// Create agents for different users
	agent1 := &models.Agent{
		AgentID:    "agent-1",
		UserID:     "user-1",
		Name:       "Agent 1",
		Registered: time.Now(),
		LastSeen:   time.Now(),
	}
	agent2 := &models.Agent{
		AgentID:    "agent-2",
		UserID:     "user-1",
		Name:       "Agent 2",
		Registered: time.Now(),
		LastSeen:   time.Now(),
	}
	agent3 := &models.Agent{
		AgentID:    "agent-3",
		UserID:     "user-2",
		Name:       "Agent 3",
		Registered: time.Now(),
		LastSeen:   time.Now(),
	}

	_ = st.CreateOrUpdateAgent(agent1)
	_ = st.CreateOrUpdateAgent(agent2)
	_ = st.CreateOrUpdateAgent(agent3)

	// Test ListAgentsByUser
	user1Agents := st.ListAgentsByUser("user-1")
	if len(user1Agents) != 2 {
		t.Errorf("expected 2 agents for user-1, got %d", len(user1Agents))
	}

	user2Agents := st.ListAgentsByUser("user-2")
	if len(user2Agents) != 1 {
		t.Errorf("expected 1 agent for user-2, got %d", len(user2Agents))
	}

	user3Agents := st.ListAgentsByUser("user-3")
	if len(user3Agents) != 0 {
		t.Errorf("expected 0 agents for user-3, got %d", len(user3Agents))
	}
}
