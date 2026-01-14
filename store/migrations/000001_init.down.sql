-- Rollback initial schema for KubeAgents
-- Version: 001

-- Drop agent_statuses table first (due to foreign key dependencies)
DROP TABLE IF EXISTS agent_statuses CASCADE;

-- Drop sessions table
DROP TABLE IF EXISTS sessions CASCADE;

-- Drop agents table
DROP TABLE IF EXISTS agents CASCADE;
