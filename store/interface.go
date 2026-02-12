package store

import "github.com/kubeagents/kubeagents/models"

// Store defines the interface for data storage implementations
// Different storage backends (memory, postgres, etc.) can implement this interface
type Store interface {
	// User operations
	CreateUser(user *models.User) error
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByVerifyToken(token string) (*models.User, error)
	UpdateUser(user *models.User) error

	// Refresh token operations
	SaveRefreshToken(token *models.RefreshToken) error
	GetRefreshTokenByID(tokenID string) (*models.RefreshToken, error)
	GetRefreshToken(tokenHash string) (*models.RefreshToken, error)
	RevokeRefreshToken(tokenID string) error
	RevokeAllUserTokens(userID string) error

	// API Key operations
	CreateAPIKey(apiKey *models.APIKey) error
	GetAPIKeyByHash(keyHash string) (*models.APIKey, error)
	GetAPIKeyByID(keyID string) (*models.APIKey, error)
	ListAPIKeysByUser(userID string) ([]*models.APIKey, error)
	RevokeAPIKey(keyID string) error
	UpdateAPIKeyLastUsed(keyID string) error

	// Agent operations
	CreateOrUpdateAgent(agent *models.Agent) error
	GetAgent(agentID string) (*models.Agent, error)
	ListAgents() []*models.Agent
	ListAgentsByUser(userID string) []*models.Agent

	// Session operations
	CreateOrUpdateSession(session *models.Session) error
	GetSession(agentID, sessionTopic string) (*models.Session, error)
	ListSessions(agentID string, includeExpired bool) []*models.Session

	// Status operations
	AddStatus(status *models.AgentStatus) error
	GetStatusHistory(agentID, sessionTopic string) ([]*models.AgentStatus, error)
	GetLatestStatus(agentID, sessionTopic string) (*models.AgentStatus, error)

	// Maintenance
	CheckExpiredSessions()

	// System config operations
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error
}
