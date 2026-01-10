package store

import (
	"sync"
	"time"

	"github.com/kubeagents/kubeagents/models"
)

// Store is a thread-safe in-memory store for agents, sessions, and statuses
type Store struct {
	mu       sync.RWMutex
	agents   map[string]*models.Agent
	sessions map[string]map[string]*models.Session // agent_id -> session_topic
	statuses map[string]map[string][]*models.AgentStatus // agent_id -> session_topic -> history
}

// NewStore creates a new memory store
func NewStore() *Store {
	return &Store{
		agents:   make(map[string]*models.Agent),
		sessions: make(map[string]map[string]*models.Session),
		statuses: make(map[string]map[string][]*models.AgentStatus),
	}
}

// CreateOrUpdateAgent creates or updates an agent
func (s *Store) CreateOrUpdateAgent(agent *models.Agent) error {
	if err := agent.Validate(); err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.agents[agent.AgentID] = agent
	return nil
}

// GetAgent retrieves an agent by ID
func (s *Store) GetAgent(agentID string) (*models.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	agent, exists := s.agents[agentID]
	if !exists {
		return nil, ErrNotFound
	}
	return agent, nil
}

// ListAgents returns all agents
func (s *Store) ListAgents() []*models.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	agents := make([]*models.Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents
}

// CreateOrUpdateSession creates or updates a session
func (s *Store) CreateOrUpdateSession(session *models.Session) error {
	if err := session.Validate(); err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Ensure agent exists
	if _, exists := s.agents[session.AgentID]; !exists {
		return ErrNotFound
	}
	
	// Initialize session map for agent if needed
	if s.sessions[session.AgentID] == nil {
		s.sessions[session.AgentID] = make(map[string]*models.Session)
	}
	
	s.sessions[session.AgentID][session.SessionTopic] = session
	return nil
}

// GetSession retrieves a session by agent ID and session topic
func (s *Store) GetSession(agentID, sessionTopic string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	sessions, exists := s.sessions[agentID]
	if !exists {
		return nil, ErrNotFound
	}
	
	session, exists := sessions[sessionTopic]
	if !exists {
		return nil, ErrNotFound
	}
	return session, nil
}

// ListSessions returns all sessions for an agent
func (s *Store) ListSessions(agentID string, includeExpired bool) []*models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	sessions, exists := s.sessions[agentID]
	if !exists {
		return []*models.Session{}
	}
	
	result := make([]*models.Session, 0)
	for _, session := range sessions {
		if includeExpired || !session.Expired {
			result = append(result, session)
		}
	}
	return result
}

// AddStatus adds a status record to the history
func (s *Store) AddStatus(status *models.AgentStatus) error {
	if err := status.Validate(); err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Ensure session exists
	sessions, exists := s.sessions[status.AgentID]
	if !exists {
		return ErrNotFound
	}
	if _, exists := sessions[status.SessionTopic]; !exists {
		return ErrNotFound
	}
	
	// Initialize status map for agent if needed
	if s.statuses[status.AgentID] == nil {
		s.statuses[status.AgentID] = make(map[string][]*models.AgentStatus)
	}
	
	// Initialize status slice for session if needed
	if s.statuses[status.AgentID][status.SessionTopic] == nil {
		s.statuses[status.AgentID][status.SessionTopic] = make([]*models.AgentStatus, 0)
	}
	
	s.statuses[status.AgentID][status.SessionTopic] = append(
		s.statuses[status.AgentID][status.SessionTopic],
		status,
	)
	return nil
}

// GetStatusHistory returns all status records for a session
func (s *Store) GetStatusHistory(agentID, sessionTopic string) ([]*models.AgentStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	statuses, exists := s.statuses[agentID]
	if !exists {
		return []*models.AgentStatus{}, nil
	}
	
	history, exists := statuses[sessionTopic]
	if !exists {
		return []*models.AgentStatus{}, nil
	}
	return history, nil
}

// GetLatestStatus returns the latest status for a session
func (s *Store) GetLatestStatus(agentID, sessionTopic string) (*models.AgentStatus, error) {
	history, err := s.GetStatusHistory(agentID, sessionTopic)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, ErrNotFound
	}
	
	// Find latest by timestamp
	latest := history[0]
	for _, status := range history[1:] {
		if status.Timestamp.After(latest.Timestamp) {
			latest = status
		}
	}
	return latest, nil
}

// CheckExpiredSessions checks and marks expired sessions
func (s *Store) CheckExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for _, sessions := range s.sessions {
		for _, session := range sessions {
			if session.Expired {
				continue
			}
			
			ttl := session.TTLMinutes
			if ttl == 0 {
				ttl = 30 // default 30 minutes
			}
			
			expiryTime := session.LastUpdated.Add(time.Duration(ttl) * time.Minute)
			if now.After(expiryTime) {
				session.Expired = true
				expiredAt := now
				session.ExpiredAt = &expiredAt
			}
		}
	}
}

// Errors
var (
	ErrNotFound = &StoreError{Message: "not found"}
)

type StoreError struct {
	Message string
}

func (e *StoreError) Error() string {
	return e.Message
}
