package store

import (
	"sync"
	"time"

	"github.com/kubeagents/kubeagents/models"
)

// MemoryStore is a thread-safe in-memory store for agents, sessions, and statuses
type MemoryStore struct {
	mu            sync.RWMutex
	agents        map[string]*models.Agent
	sessions      map[string]map[string]*models.Session       // agent_id -> session_topic
	statuses      map[string]map[string][]*models.AgentStatus // agent_id -> session_topic -> history
	users         map[string]*models.User                     // user_id -> user
	usersByEmail  map[string]*models.User                     // email -> user
	refreshTokens map[string]*models.RefreshToken             // token_hash -> token
	apiKeys       map[string]*models.APIKey                   // key_id -> api_key
	apiKeysByHash map[string]*models.APIKey                   // key_hash -> api_key
	config        map[string]string                           // key -> value
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents:        make(map[string]*models.Agent),
		sessions:      make(map[string]map[string]*models.Session),
		statuses:      make(map[string]map[string][]*models.AgentStatus),
		users:         make(map[string]*models.User),
		usersByEmail:  make(map[string]*models.User),
		refreshTokens: make(map[string]*models.RefreshToken),
		apiKeys:       make(map[string]*models.APIKey),
		apiKeysByHash: make(map[string]*models.APIKey),
		config:        make(map[string]string),
	}
}

// CreateOrUpdateAgent creates or updates an agent
func (s *MemoryStore) CreateOrUpdateAgent(agent *models.Agent) error {
	if err := agent.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents[agent.AgentID] = agent
	return nil
}

// GetAgent retrieves an agent by ID
func (s *MemoryStore) GetAgent(agentID string) (*models.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return nil, ErrNotFound
	}
	return agent, nil
}

// ListAgents returns all agents
func (s *MemoryStore) ListAgents() []*models.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*models.Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents
}

// CreateOrUpdateSession creates or updates a session
func (s *MemoryStore) CreateOrUpdateSession(session *models.Session) error {
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
func (s *MemoryStore) GetSession(agentID, sessionTopic string) (*models.Session, error) {
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
func (s *MemoryStore) ListSessions(agentID string, includeExpired bool) []*models.Session {
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
func (s *MemoryStore) AddStatus(status *models.AgentStatus) error {
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
func (s *MemoryStore) GetStatusHistory(agentID, sessionTopic string) ([]*models.AgentStatus, error) {
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
func (s *MemoryStore) GetLatestStatus(agentID, sessionTopic string) (*models.AgentStatus, error) {
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
func (s *MemoryStore) CheckExpiredSessions() {
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

// ListAgentsByUser returns all agents belonging to a specific user
func (s *MemoryStore) ListAgentsByUser(userID string) []*models.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*models.Agent, 0)
	for _, agent := range s.agents {
		if agent.UserID == userID {
			agents = append(agents, agent)
		}
	}
	return agents
}

// CreateUser creates a new user
func (s *MemoryStore) CreateUser(user *models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if email already exists
	if _, exists := s.usersByEmail[user.Email]; exists {
		return ErrDuplicateEmail
	}

	s.users[user.ID] = user
	s.usersByEmail[user.Email] = user
	return nil
}

// GetUserByID retrieves a user by ID
func (s *MemoryStore) GetUserByID(userID string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, ErrNotFound
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *MemoryStore) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.usersByEmail[email]
	if !exists {
		return nil, ErrNotFound
	}
	return user, nil
}

// GetUserByVerifyToken retrieves a user by verification token
func (s *MemoryStore) GetUserByVerifyToken(token string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.VerifyToken == token {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

// UpdateUser updates an existing user
func (s *MemoryStore) UpdateUser(user *models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existingUser, exists := s.users[user.ID]
	if !exists {
		return ErrNotFound
	}

	// If email changed, update email index
	if existingUser.Email != user.Email {
		// Check if new email already exists
		if _, exists := s.usersByEmail[user.Email]; exists {
			return ErrDuplicateEmail
		}
		delete(s.usersByEmail, existingUser.Email)
		s.usersByEmail[user.Email] = user
	}

	s.users[user.ID] = user
	return nil
}

// SaveRefreshToken saves a refresh token
func (s *MemoryStore) SaveRefreshToken(token *models.RefreshToken) error {
	if err := token.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[token.TokenHash] = token
	return nil
}

// GetRefreshToken retrieves a refresh token by hash
func (s *MemoryStore) GetRefreshToken(tokenHash string) (*models.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.refreshTokens[tokenHash]
	if !exists {
		return nil, ErrNotFound
	}
	return token, nil
}

// RevokeRefreshToken revokes a refresh token
func (s *MemoryStore) RevokeRefreshToken(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.refreshTokens[tokenHash]
	if !exists {
		return ErrNotFound
	}
	token.Revoked = true
	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (s *MemoryStore) RevokeAllUserTokens(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, token := range s.refreshTokens {
		if token.UserID == userID {
			token.Revoked = true
		}
	}
	return nil
}

// CreateAPIKey creates a new API key
func (s *MemoryStore) CreateAPIKey(apiKey *models.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiKeys[apiKey.ID] = apiKey
	s.apiKeysByHash[apiKey.KeyHash] = apiKey
	return nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (s *MemoryStore) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, exists := s.apiKeysByHash[keyHash]
	if !exists {
		return nil, ErrNotFound
	}
	return apiKey, nil
}

// GetAPIKeyByID retrieves an API key by its ID
func (s *MemoryStore) GetAPIKeyByID(keyID string) (*models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, exists := s.apiKeys[keyID]
	if !exists {
		return nil, ErrNotFound
	}
	return apiKey, nil
}

// ListAPIKeysByUser returns all API keys for a user
func (s *MemoryStore) ListAPIKeysByUser(userID string) ([]*models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*models.APIKey, 0)
	for _, apiKey := range s.apiKeys {
		if apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}
	return keys, nil
}

// RevokeAPIKey revokes an API key
func (s *MemoryStore) RevokeAPIKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKey, exists := s.apiKeys[keyID]
	if !exists {
		return ErrNotFound
	}
	apiKey.Revoked = true
	return nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp of an API key
func (s *MemoryStore) UpdateAPIKeyLastUsed(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKey, exists := s.apiKeys[keyID]
	if !exists {
		return ErrNotFound
	}
	now := time.Now()
	apiKey.LastUsedAt = &now
	return nil
}

// GetConfig retrieves a config value by key
func (s *MemoryStore) GetConfig(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.config[key]
	if !exists {
		return "", ErrNotFound
	}
	return value, nil
}

// SetConfig sets a config value
func (s *MemoryStore) SetConfig(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config[key] = value
	return nil
}
