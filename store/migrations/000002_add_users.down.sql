-- Drop refresh_tokens table
DROP TABLE IF EXISTS refresh_tokens;

-- Remove foreign key constraint from agents
ALTER TABLE agents DROP CONSTRAINT IF EXISTS fk_agent_user;

-- Remove index on user_id
DROP INDEX IF EXISTS idx_agents_user_id;

-- Remove user_id column from agents
ALTER TABLE agents DROP COLUMN IF EXISTS user_id;

-- Drop users table (this will also drop related indexes)
DROP TABLE IF EXISTS users;
