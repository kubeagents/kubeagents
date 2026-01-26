# kubeagents

**A backend service for agent orchestration and status management**

[中文文档](./README_zh.md)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Overview

KubeAgents is a REST API server designed to receive and manage status reports from AI agents. It serves as the central orchestration layer for tracking agent activities, managing session lifecycles, and providing real-time status visibility across multiple AI assistants.

## What It Does

KubeAgents acts as a hub for AI agent coordination:

- **Receives status reports** from AI agents via HTTP webhooks
- **Automatically registers** new agents on first contact
- **Manages session lifecycles** with automatic expiration based on TTL
- **Stores status history** for tracking and analysis
- **Provides REST APIs** for querying agent and session information
- **Supports multiple storage backends** - PostgreSQL for production, in-memory for development

## How It Works

KubeAgents works together with [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp) to create a complete agent orchestration system:

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────────┐
│  AI Agent       │ ──────> │  kubeagents-mcp  │ ──────> │   kubeagents    │
│  (Cursor/Claude)│         │  (MCP Server)    │         │   (REST API)    │
└─────────────────┘         └──────────────────┘         └─────────────────┘
     (Calls MCP Tools)                                    (Receives HTTP
                                                          Webhooks)
```

### The Workflow

1. **kubeagents Server** starts and listens for HTTP webhook requests on `POST /webhook/status`

2. **kubeagents-mcp** provides three MCP tools that AI agents can call:
   - `start_session` - Starts a new session when a task begins
   - `report_status` - Reports intermediate status during task execution
   - `end_session` - Marks the session as complete with final status

3. **AI Agent** (like Cursor or Claude Desktop) calls these MCP tools during conversations

4. **kubeagents-mcp** forwards the status to kubeagents via HTTP webhook

5. **kubeagents** processes the request:
   - Auto-registers the agent if it doesn't exist
   - Creates or updates the session
   - Stores the status with timestamp
   - Manages session expiration based on TTL

6. **Users** can query the status through:
   - [kubeagents-web](https://github.com/kubeagents/kubeagents-web) - Web UI for visualization
   - REST API endpoints for programmatic access

### Session Lifecycle

Each task execution in an AI agent is tracked as a session:

```
start_session ──> report_status (running) ──> ... ──> end_session (success/failed)
     │                  │                           │
     v                  v                           v
   Session            Status                      Session
   Created            Updated                    Closed
```

Sessions automatically expire after the configured TTL (default: 60 minutes) unless renewed by new status reports.

## Features

### Core Capabilities

- **Automatic Agent Registration**: No manual setup needed - agents auto-register on first status report
- **Session Management**: Full lifecycle management with automatic expiration
- **Real-time Tracking**: Capture detailed status updates throughout task execution
- **Status History**: Query historical status for any agent or session
- **Concurrent Safe**: Thread-safe operations for multiple agents

### Storage Options

- **In-Memory Storage** (default): Fast, no database required, perfect for development and testing
- **PostgreSQL Storage**: Persistent storage with automatic migrations, ideal for production

### Integration Features

- **Webhook Notifications**: Push notifications to external services on status updates
- **CORS Support**: Configurable CORS origins for cross-origin requests
- **Flexible TTL**: Per-session TTL configuration for different task types

## Deployment

### Development Deployment

For local development, use in-memory storage:

```bash
# Install dependencies
go mod download

# Run the server
go run main.go
```

The server starts on `http://localhost:8080` by default.

### Production Deployment

For production, use PostgreSQL for persistent storage:

#### 1. Start PostgreSQL

Using Docker Compose (recommended):

```bash
docker-compose up -d
```

This starts PostgreSQL with default credentials:
- Host: `localhost:5432`
- User: `kubeagents`
- Password: `kubeagents`
- Database: `kubeagents`

Or use your own PostgreSQL instance.

#### 2. Configure Environment Variables

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=kubeagents
export DB_PASSWORD=kubeagents
export DB_NAME=kubeagents

# Optional: Customize server settings
export PORT=8080
export CORS_ALLOWED_ORIGINS=http://localhost:3000,https://yourdomain.com
export NOTIFICATION_TIMEOUT_SECONDS=5
```

#### 3. Run the Server

```bash
# Build for production
go build -o kubeagents-server main.go

# Run the server
./kubeagents-server
```

The server will automatically run database migrations on startup.

### Docker Deployment

Build and run with Docker:

```bash
# Build the image
docker build -t kubeagents:latest .

# Run with PostgreSQL
docker run -d \
  --name kubeagents \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=kubeagents \
  -e DB_PASSWORD=kubeagents \
  -e DB_NAME=kubeagents \
  --link postgres:postgres \
  kubeagents:latest
```

### Kubernetes Deployment

For Kubernetes deployments, use the provided deployment manifests (if available) or create your own with the following considerations:

- Use PostgreSQL StatefulSet or external managed database
- Configure readiness/liveness probes on `/health` endpoint
- Set resource limits based on expected load
- Use ConfigMap for environment variables

## Environment Variables

### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins (comma-separated) | `*` |
| `NOTIFICATION_TIMEOUT_SECONDS` | Webhook notification timeout | `5` |

### Database Configuration (Optional)

If any `DB_*` variable is set, PostgreSQL storage is used. Otherwise, in-memory storage is used.

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | - |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | - |
| `DB_SSLMODE` | SSL mode | `disable` |
| `DB_MAX_OPEN_CONNS` | Max open connections | `25` |
| `DB_MAX_IDLE_CONNS` | Max idle connections | `5` |
| `DB_CONN_MAX_LIFETIME` | Connection max lifetime | `5m` |

## Integration with kubeagents-mcp

To connect kubeagents with AI agents, you need to set up [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp):

### Quick Setup

1. **Build kubeagents-mcp server**:
   ```bash
   cd ../kubeagents-mcp
   go mod download
   go build -o kubeagents-mcp-server ./cmd/mcp-server
   ```

2. **Configure in Cursor**:
   Add to Cursor's MCP server settings:
   ```json
   {
     "mcpServers": {
       "kubeagents-reporter": {
         "command": "/path/to/kubeagents-mcp-server",
         "env": {
           "KUBEAGENTS_SERVER_URL": "http://localhost:8080"
         }
       }
     }
   }
   ```

3. **Restart Cursor**

Now Cursor can call the MCP tools to report status to kubeagents!

### Configuration Options

Configure kubeagents-mcp with these environment variables:

- `KUBEAGENTS_SERVER_URL`: URL of kubeagents server (default: `http://localhost:8080`)
- `KUBEAGENTS_AUTO_REPORT_ENABLED`: Enable automatic reporting (default: `false`)

## Web UI

Use [kubeagents-web](https://github.com/kubeagents/kubeagents-web) for a visual interface to monitor agent activities:

- Real-time agent status
- Session history and details
- Status timeline visualization
- Configure webhook notifications per user

## Health Check

Check server status:

```bash
curl http://localhost:8080/health
```

Returns `200 OK` if the server is running.

## Next Steps

- Set up [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp) to connect your AI agents
- Deploy [kubeagents-web](https://github.com/kubeagents/kubeagents-web) for visual monitoring
- Configure webhook notifications for status updates
- Review [AGENTS.md](./AGENTS.md) for agent integration guidelines

## Related Projects

- [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp) - MCP server for agent status reporting
- [kubeagents-web](https://github.com/kubeagents/kubeagents-web) - Web UI for monitoring and management
- [specs](../specs/) - API specifications and design documents
