package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kubeagents/kubeagents/models"
)

// PostgresStore implements Store interface using PostgreSQL
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL store connection
func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Pool returns the underlying connection pool
func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// Close closes the database connection pool
func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

// CreateOrUpdateAgent creates or updates an agent
func (s *PostgresStore) CreateOrUpdateAgent(agent *models.Agent) error {
	if err := agent.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO agents (agent_id, user_id, name, source, registered, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (agent_id) DO UPDATE
		SET name = EXCLUDED.name,
		    source = EXCLUDED.source,
		    last_seen = EXCLUDED.last_seen,
		    user_id = COALESCE(agents.user_id, EXCLUDED.user_id)
	`

	_, err := s.pool.Exec(ctx, query,
		agent.AgentID,
		agent.UserID,
		agent.Name,
		agent.Source,
		agent.Registered,
		agent.LastSeen,
	)

	if err != nil {
		return fmt.Errorf("failed to create/update agent: %w", err)
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (s *PostgresStore) GetAgent(agentID string) (*models.Agent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT agent_id, COALESCE(user_id, ''), name, source, registered, last_seen
		FROM agents
		WHERE agent_id = $1
	`

	row := s.pool.QueryRow(ctx, query, agentID)

	var agent models.Agent
	err := row.Scan(
		&agent.AgentID,
		&agent.UserID,
		&agent.Name,
		&agent.Source,
		&agent.Registered,
		&agent.LastSeen,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return &agent, nil
}

// ListAgents returns all agents
func (s *PostgresStore) ListAgents() []*models.Agent {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT agent_id, COALESCE(user_id, ''), name, source, registered, last_seen
		FROM agents
		ORDER BY last_seen DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return []*models.Agent{}
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var agent models.Agent
		if err := rows.Scan(
			&agent.AgentID,
			&agent.UserID,
			&agent.Name,
			&agent.Source,
			&agent.Registered,
			&agent.LastSeen,
		); err != nil {
			continue
		}
		agents = append(agents, &agent)
	}

	return agents
}

// ListAgentsByUser returns all agents belonging to a specific user
func (s *PostgresStore) ListAgentsByUser(userID string) []*models.Agent {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT agent_id, COALESCE(user_id, ''), name, source, registered, last_seen
		FROM agents
		WHERE user_id = $1
		ORDER BY last_seen DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return []*models.Agent{}
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var agent models.Agent
		if err := rows.Scan(
			&agent.AgentID,
			&agent.UserID,
			&agent.Name,
			&agent.Source,
			&agent.Registered,
			&agent.LastSeen,
		); err != nil {
			continue
		}
		agents = append(agents, &agent)
	}

	return agents
}

// CreateOrUpdateSession creates or updates a session
func (s *PostgresStore) CreateOrUpdateSession(session *models.Session) error {
	if err := session.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO sessions (agent_id, session_topic, created, last_updated, expired, expired_at, ttl_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (agent_id, session_topic) DO UPDATE
		SET last_updated = EXCLUDED.last_updated,
		    expired = EXCLUDED.expired,
		    expired_at = EXCLUDED.expired_at,
		    ttl_minutes = EXCLUDED.ttl_minutes
	`

	_, err := s.pool.Exec(ctx, query,
		session.AgentID,
		session.SessionTopic,
		session.Created,
		session.LastUpdated,
		session.Expired,
		session.ExpiredAt,
		session.TTLMinutes,
	)

	if err != nil {
		return fmt.Errorf("failed to create/update session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by agent ID and session topic
func (s *PostgresStore) GetSession(agentID, sessionTopic string) (*models.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT agent_id, session_topic, created, last_updated, expired, expired_at, ttl_minutes
		FROM sessions
		WHERE agent_id = $1 AND session_topic = $2
	`

	row := s.pool.QueryRow(ctx, query, agentID, sessionTopic)

	var session models.Session
	err := row.Scan(
		&session.AgentID,
		&session.SessionTopic,
		&session.Created,
		&session.LastUpdated,
		&session.Expired,
		&session.ExpiredAt,
		&session.TTLMinutes,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// ListSessions returns all sessions for an agent
func (s *PostgresStore) ListSessions(agentID string, includeExpired bool) []*models.Session {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT agent_id, session_topic, created, last_updated, expired, expired_at, ttl_minutes
		FROM sessions
		WHERE agent_id = $1
	`

	if !includeExpired {
		query += " AND expired = false"
	}

	query += " ORDER BY last_updated DESC"

	rows, err := s.pool.Query(ctx, query, agentID)
	if err != nil {
		return []*models.Session{}
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		var session models.Session
		if err := rows.Scan(
			&session.AgentID,
			&session.SessionTopic,
			&session.Created,
			&session.LastUpdated,
			&session.Expired,
			&session.ExpiredAt,
			&session.TTLMinutes,
		); err != nil {
			continue
		}
		sessions = append(sessions, &session)
	}

	return sessions
}

// AddStatus adds a status record to history
func (s *PostgresStore) AddStatus(status *models.AgentStatus) error {
	if err := status.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO agent_statuses (agent_id, session_topic, status, timestamp, message, content)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, query,
		status.AgentID,
		status.SessionTopic,
		status.Status,
		status.Timestamp,
		status.Message,
		status.Content,
	)

	if err != nil {
		return fmt.Errorf("failed to add status: %w", err)
	}

	return nil
}

// GetStatusHistory returns all status records for a session
func (s *PostgresStore) GetStatusHistory(agentID, sessionTopic string) ([]*models.AgentStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, agent_id, session_topic, status, timestamp, message, content
		FROM agent_statuses
		WHERE agent_id = $1 AND session_topic = $2
		ORDER BY timestamp DESC
	`

	rows, err := s.pool.Query(ctx, query, agentID, sessionTopic)
	if err != nil {
		return nil, fmt.Errorf("failed to get status history: %w", err)
	}
	defer rows.Close()

	var statuses []*models.AgentStatus
	for rows.Next() {
		var status models.AgentStatus
		if err := rows.Scan(
			new(interface{}), // id - not used
			&status.AgentID,
			&status.SessionTopic,
			&status.Status,
			&status.Timestamp,
			&status.Message,
			&status.Content,
		); err != nil {
			continue
		}
		statuses = append(statuses, &status)
	}

	return statuses, nil
}

// GetLatestStatus returns the latest status for a session
func (s *PostgresStore) GetLatestStatus(agentID, sessionTopic string) (*models.AgentStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT agent_id, session_topic, status, timestamp, message, content
		FROM agent_statuses
		WHERE agent_id = $1 AND session_topic = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	row := s.pool.QueryRow(ctx, query, agentID, sessionTopic)

	var status models.AgentStatus
	err := row.Scan(
		&status.AgentID,
		&status.SessionTopic,
		&status.Status,
		&status.Timestamp,
		&status.Message,
		&status.Content,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get latest status: %w", err)
	}

	return &status, nil
}

// CheckExpiredSessions checks and marks expired sessions
func (s *PostgresStore) CheckExpiredSessions() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()

	query := `
		UPDATE sessions
		SET expired = true,
		    expired_at = $1
		WHERE expired = false
		  AND last_updated + (ttl_minutes || ' minutes')::interval < $1
	`

	_, err := s.pool.Exec(ctx, query, now)
	if err != nil {
		return
	}
}

// CreateUser creates a new user
func (s *PostgresStore) CreateUser(user *models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (id, email, password_hash, name, email_verified, verify_token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.EmailVerified,
		user.VerifyToken,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if isDuplicateKeyError(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (s *PostgresStore) GetUserByID(userID string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, email, password_hash, COALESCE(name, ''), email_verified, COALESCE(verify_token, ''), created_at, updated_at
		FROM users
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, query, userID)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.EmailVerified,
		&user.VerifyToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *PostgresStore) GetUserByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, email, password_hash, COALESCE(name, ''), email_verified, COALESCE(verify_token, ''), created_at, updated_at
		FROM users
		WHERE email = $1
	`

	row := s.pool.QueryRow(ctx, query, email)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.EmailVerified,
		&user.VerifyToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetUserByVerifyToken retrieves a user by verification token
func (s *PostgresStore) GetUserByVerifyToken(token string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, email, password_hash, COALESCE(name, ''), email_verified, COALESCE(verify_token, ''), created_at, updated_at
		FROM users
		WHERE verify_token = $1
	`

	row := s.pool.QueryRow(ctx, query, token)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.EmailVerified,
		&user.VerifyToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by verify token: %w", err)
	}

	return &user, nil
}

// UpdateUser updates an existing user
func (s *PostgresStore) UpdateUser(user *models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE users
		SET email = $2, password_hash = $3, name = $4, email_verified = $5, verify_token = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := s.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.EmailVerified,
		user.VerifyToken,
		user.UpdatedAt,
	)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// SaveRefreshToken saves a refresh token
func (s *PostgresStore) SaveRefreshToken(token *models.RefreshToken) error {
	if err := token.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
		token.Revoked,
	)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by hash
func (s *PostgresStore) GetRefreshToken(tokenHash string) (*models.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	row := s.pool.QueryRow(ctx, query, tokenHash)

	var token models.RefreshToken
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.Revoked,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &token, nil
}

// RevokeRefreshToken revokes a refresh token
func (s *PostgresStore) RevokeRefreshToken(tokenHash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE refresh_tokens
		SET revoked = true
		WHERE token_hash = $1
	`

	result, err := s.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (s *PostgresStore) RevokeAllUserTokens(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE refresh_tokens
		SET revoked = true
		WHERE user_id = $1
	`

	_, err := s.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	return nil
}

// CreateAPIKey creates a new API key
func (s *PostgresStore) CreateAPIKey(apiKey *models.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, expires_at, last_used_at, created_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.pool.Exec(ctx, query,
		apiKey.ID,
		apiKey.UserID,
		apiKey.Name,
		apiKey.KeyHash,
		apiKey.KeyPrefix,
		apiKey.ExpiresAt,
		apiKey.LastUsedAt,
		apiKey.CreatedAt,
		apiKey.Revoked,
	)

	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (s *PostgresStore) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, name, key_hash, key_prefix, expires_at, last_used_at, created_at, revoked
		FROM api_keys
		WHERE key_hash = $1
	`

	row := s.pool.QueryRow(ctx, query, keyHash)

	var apiKey models.APIKey
	err := row.Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.CreatedAt,
		&apiKey.Revoked,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// GetAPIKeyByID retrieves an API key by its ID
func (s *PostgresStore) GetAPIKeyByID(keyID string) (*models.APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, name, key_hash, key_prefix, expires_at, last_used_at, created_at, revoked
		FROM api_keys
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, query, keyID)

	var apiKey models.APIKey
	err := row.Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.CreatedAt,
		&apiKey.Revoked,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// ListAPIKeysByUser returns all API keys for a user
func (s *PostgresStore) ListAPIKeysByUser(userID string) ([]*models.APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, name, key_hash, key_prefix, expires_at, last_used_at, created_at, revoked
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		var apiKey models.APIKey
		if err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.Name,
			&apiKey.KeyHash,
			&apiKey.KeyPrefix,
			&apiKey.ExpiresAt,
			&apiKey.LastUsedAt,
			&apiKey.CreatedAt,
			&apiKey.Revoked,
		); err != nil {
			continue
		}
		keys = append(keys, &apiKey)
	}

	return keys, nil
}

// RevokeAPIKey revokes an API key
func (s *PostgresStore) RevokeAPIKey(keyID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE api_keys
		SET revoked = true
		WHERE id = $1
	`

	result, err := s.pool.Exec(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp of an API key
func (s *PostgresStore) UpdateAPIKeyLastUsed(keyID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE api_keys
		SET last_used_at = $2
		WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, query, keyID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// GetConfig retrieves a config value by key
func (s *PostgresStore) GetConfig(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT value FROM system_config WHERE key = $1`

	var value string
	err := s.pool.QueryRow(ctx, query, key).Scan(&value)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	return value, nil
}

// SetConfig sets a config value (upsert)
func (s *PostgresStore) SetConfig(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO system_config (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`

	_, err := s.pool.Exec(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// isDuplicateKeyError checks if the error is a duplicate key violation
func isDuplicateKeyError(err error) bool {
	// PostgreSQL unique violation error code is 23505
	return err != nil && (err.Error() == "ERROR: duplicate key value violates unique constraint" ||
		(len(err.Error()) > 0 && err.Error()[:5] == "ERROR"))
}
