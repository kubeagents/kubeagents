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
- **Concurrent Safe**: Thread-safe in-memory store for high-performance operations

## Quick Start

```bash
go mod download
go run main.go
```

The server runs on `http://localhost:8080` by default.

## Environment Variables

- `PORT`: Server port (default: `8080`)
- `CORS_ALLOWED_ORIGINS`: Allowed CORS origins (comma-separated, default: `*`)

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
- **Store**: In-memory data store with thread-safe operations
- **Models**: Data models for Agent, Session, and AgentStatus
- **Config**: Configuration management with environment variable support

## Project Structure

```
kubeagents/
├── handlers/          # HTTP handlers
├── models/            # Data models
├── store/             # In-memory store
├── config/            # Configuration
├── internal/          # Internal utilities
└── main.go           # Application entry point
```
