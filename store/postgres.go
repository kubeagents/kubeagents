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
		INSERT INTO agents (agent_id, name, source, registered, last_seen)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (agent_id) DO UPDATE
		SET name = EXCLUDED.name,
		    source = EXCLUDED.source,
		    last_seen = EXCLUDED.last_seen
	`

	_, err := s.pool.Exec(ctx, query,
		agent.AgentID,
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
		SELECT agent_id, name, source, registered, last_seen
		FROM agents
		WHERE agent_id = $1
	`

	row := s.pool.QueryRow(ctx, query, agentID)

	var agent models.Agent
	err := row.Scan(
		&agent.AgentID,
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
		SELECT agent_id, name, source, registered, last_seen
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
