-- Initial schema for KubeAgents
-- Version: 001

-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    agent_id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(200),
    source VARCHAR(200),
    registered TIMESTAMP NOT NULL,
    last_seen TIMESTAMP NOT NULL
);

-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    agent_id VARCHAR(100) NOT NULL,
    session_topic VARCHAR(500) NOT NULL,
    created TIMESTAMP NOT NULL,
    last_updated TIMESTAMP NOT NULL,
    expired BOOLEAN NOT NULL DEFAULT FALSE,
    expired_at TIMESTAMP,
    ttl_minutes INTEGER CHECK (ttl_minutes >= 0 AND ttl_minutes <= 1440),
    PRIMARY KEY (agent_id, session_topic),
    CONSTRAINT fk_session_agent FOREIGN KEY (agent_id)
        REFERENCES agents(agent_id)
        ON DELETE CASCADE
);

-- Create agent_statuses table
CREATE TABLE IF NOT EXISTS agent_statuses (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(100) NOT NULL,
    session_topic VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('running', 'success', 'failed', 'pending')),
    timestamp TIMESTAMP NOT NULL,
    message VARCHAR(1000),
    content TEXT,
    CONSTRAINT fk_status_session FOREIGN KEY (agent_id, session_topic)
        REFERENCES sessions(agent_id, session_topic)
        ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_agent_statuses_agent_session
    ON agent_statuses(agent_id, session_topic);

CREATE INDEX IF NOT EXISTS idx_agent_statuses_timestamp
    ON agent_statuses(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_agent_id
    ON sessions(agent_id);

CREATE INDEX IF NOT EXISTS idx_sessions_expired
    ON sessions(expired);
