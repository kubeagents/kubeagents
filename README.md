# kubeagents

**A backend service for agent orchestration and status management**

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

KubeAgents is a REST API server that receives and manages status reports from AI agents. It provides automatic agent registration, session lifecycle management, and real-time status tracking capabilities.

## Features

- **Status Webhook**: Receives agent status reports via HTTP webhook
- **REST API**: Query agent and session information
- **Automatic Management**: Auto-registers agents and manages session lifecycles
- **Session Expiration**: Automatically expires inactive sessions based on TTL
- **Concurrent Safe**: Thread-safe storage operations
- **Multiple Storage Backends**: Supports PostgreSQL and in-memory storage
- **Automatic Migrations**: Database migrations run automatically on startup

## Quick Start

### Using In-Memory Storage (Default)

```bash
go mod download
go run main.go
```

The server runs on `http://localhost:8080` by default.

### Using PostgreSQL Storage

First, start PostgreSQL using Docker Compose:

```bash
docker-compose up -d
```

Then, run the application with database configuration:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=kubeagents
export DB_PASSWORD=kubeagents
export DB_NAME=kubeagents

go run main.go
```

The application will automatically run database migrations on startup.

## Environment Variables

- `PORT`: Server port (default: `8080`)
- `CORS_ALLOWED_ORIGINS`: Allowed CORS origins (comma-separated, default: `*`)
- `NOTIFICATION_WEBHOOK_URL`: Optional webhook URL for status notifications
- `NOTIFICATION_TIMEOUT_SECONDS`: Notification timeout in seconds (default: `5`)

### Database Configuration (Optional)

If database configuration is provided, the application will use PostgreSQL storage. Otherwise, it will use in-memory storage.

- `DB_HOST`: Database host (default: `localhost`)
- `DB_PORT`: Database port (default: `5432`)
- `DB_USER`: Database user
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name
- `DB_SSLMODE`: SSL mode for database connection (default: `disable`)
- `DB_MAX_OPEN_CONNS`: Maximum number of open database connections (default: `25`)
- `DB_MAX_IDLE_CONNS`: Maximum number of idle database connections (default: `5`)
- `DB_CONN_MAX_LIFETIME`: Maximum lifetime of a database connection (default: `5m`)

#### Example: Using PostgreSQL

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=kubeagents
export DB_PASSWORD=kubeagents
export DB_NAME=kubeagents
export DB_SSLMODE=disable
```

#### Example: Using In-Memory Storage

Simply don't set any `DB_*` environment variables, and the application will use in-memory storage (useful for development and testing).

## API Documentation

See [contracts/api.yaml](../specs/001-kubeagents-system/contracts/api.yaml) for detailed API specifications.

### Endpoints

- `GET /health` - Health check endpoint
- `POST /webhook/status` - Receive agent status reports
- `GET /api/agents` - List all agents
- `GET /api/agents/{agent_id}` - Get agent details
- `GET /api/agents/{agent_id}/sessions` - List agent sessions
- `GET /api/agents/{agent_id}/sessions/{session_topic}` - Get session details
- `GET /api/agents/{agent_id}/status` - Get agent status history

## Testing

```bash
go test ./...
go test -race ./...  # Race condition testing
```

## Architecture

- **Handlers**: HTTP request handlers for API endpoints
- **Store**: Storage interface with PostgreSQL and in-memory implementations
- **Models**: Data models for Agent, Session, and AgentStatus
- **Config**: Configuration management with environment variable support
- **Migrations**: Automatic database schema migrations

## Project Structure

```
kubeagents/
├── handlers/          # HTTP handlers
├── models/            # Data models
├── store/             # Storage implementations (memory, postgres)
│   ├── migrations/   # Database migration scripts
├── config/            # Configuration
├── internal/          # Internal utilities
├── docker-compose.yml  # Docker Compose configuration for PostgreSQL
└── main.go           # Application entry point
```
