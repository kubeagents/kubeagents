-- Migration 000005: Drop all tables and recreate with TIMESTAMPTZ
-- This fixes timezone issues by using TIMESTAMP WITH TIME ZONE

-- Drop all tables (in correct order due to foreign keys)
DROP TABLE IF EXISTS agent_statuses CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS system_config CASCADE;

-- Recreate users table with TIMESTAMPTZ
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(200),
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verify_token VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create agents table with TIMESTAMPTZ
CREATE TABLE agents (
    agent_id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(36),
    name VARCHAR(200),
    source VARCHAR(200),
    registered TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT fk_agent_user FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Create sessions table with TIMESTAMPTZ
CREATE TABLE sessions (
    agent_id VARCHAR(100) NOT NULL,
    session_topic VARCHAR(500) NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL,
    expired BOOLEAN NOT NULL DEFAULT FALSE,
    expired_at TIMESTAMP WITH TIME ZONE,
    ttl_minutes INTEGER CHECK (ttl_minutes >= 0 AND ttl_minutes <= 1440),
    PRIMARY KEY (agent_id, session_topic),
    CONSTRAINT fk_session_agent FOREIGN KEY (agent_id)
        REFERENCES agents(agent_id)
        ON DELETE CASCADE
);

-- Create agent_statuses table with TIMESTAMPTZ
CREATE TABLE agent_statuses (
    id BIGSERIAL PRIMARY KEY,
    agent_id VARCHAR(100) NOT NULL,
    session_topic VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('running', 'success', 'failed', 'pending')),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    message VARCHAR(1000),
    content TEXT,
    CONSTRAINT fk_status_session FOREIGN KEY (agent_id, session_topic)
        REFERENCES sessions(agent_id, session_topic)
        ON DELETE CASCADE
);

-- Create refresh_tokens table with TIMESTAMPTZ
CREATE TABLE refresh_tokens (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create api_keys table with TIMESTAMPTZ
CREATE TABLE api_keys (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(8) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create system_config table with TIMESTAMPTZ
CREATE TABLE system_config (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_verify_token ON users(verify_token);
CREATE INDEX idx_agents_user_id ON agents(user_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_agent_statuses_agent_session ON agent_statuses(agent_id, session_topic);
CREATE INDEX idx_agent_statuses_timestamp ON agent_statuses(timestamp DESC);
CREATE INDEX idx_sessions_agent_id ON sessions(agent_id);
CREATE INDEX idx_sessions_expired ON sessions(expired);
CREATE INDEX idx_system_config_key ON system_config(key);
