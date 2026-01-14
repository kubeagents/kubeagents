package store

import "github.com/kubeagents/kubeagents/models"

// Store defines the interface for data storage implementations
// Different storage backends (memory, postgres, etc.) can implement this interface
type Store interface {
	// Agent operations
	CreateOrUpdateAgent(agent *models.Agent) error
	GetAgent(agentID string) (*models.Agent, error)
	ListAgents() []*models.Agent

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
}
