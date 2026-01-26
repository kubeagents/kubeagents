# kubeagents

**用于 Agent 编排和状态管理的后端服务**

## 许可证

根据 Apache License, Version 2.0 许可。详情请参见 [LICENSE](LICENSE)。

## 概述

KubeAgents 是一个 REST API 服务器，专门用于接收和管理 AI Agent 的状态上报。它作为 AI 助手协调的中心编排层，提供会话生命周期管理、实时状态跟踪和多 Agent 监控功能。

## 核心功能

KubeAgents 作为 AI Agent 协调的中心枢纽：

- **接收状态上报**：通过 HTTP webhook 接收来自 AI Agent 的状态报告
- **自动注册 Agent**：首次接触时自动注册新 Agent
- **管理会话生命周期**：基于 TTL 自动过期非活跃会话
- **存储状态历史**：跟踪和分析 Agent 活动
- **提供 REST API**：查询 Agent 和会话信息
- **支持多种存储后端**：生产环境使用 PostgreSQL，开发环境使用内存存储

## 工作原理

KubeAgents 与 [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp) 协同工作，构建完整的 Agent 编排系统：

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────────┐
│  AI Agent       │ ──────> │  kubeagents-mcp  │ ──────> │   kubeagents    │
│  (Cursor/Claude)│         │  (MCP 服务器)     │         │   (REST API)    │
└─────────────────┘         └──────────────────┘         └─────────────────┘
     (调用 MCP 工具)                                     (接收 HTTP
                                                          Webhooks)
```

### 工作流程

1. **kubeagents 服务器**启动并监听 HTTP webhook 请求（`POST /webhook/status`）

2. **kubeagents-mcp**提供三个 MCP 工具供 AI Agent 调用：
   - `start_session` - 任务开始时启动新会话
   - `report_status` - 任务执行过程中上报中间状态
   - `end_session` - 标记会话完成并上报最终状态

3. **AI Agent**（如 Cursor 或 Claude Desktop）在对话过程中调用这些 MCP 工具

4. **kubeagents-mcp**通过 HTTP webhook 将状态转发给 kubeagents

5. **kubeagents**处理请求：
   - 如果 Agent 不存在则自动注册
   - 创建或更新会话
   - 存储带时间戳的状态
   - 根据 TTL 管理会话过期

6. **用户**可以通过以下方式查询状态：
   - [kubeagents-web](https://github.com/kubeagents/kubeagents-web) - 可视化 Web 界面
   - REST API 端点用于程序化访问

### 会话生命周期

AI Agent 中的每个任务执行都被跟踪为一个会话：

```
start_session ──> report_status (running) ──> ... ──> end_session (success/failed)
     │                  │                           │
     v                  v                           v
   会话创建           状态更新                     会话关闭
```

会话在配置的 TTL 后自动过期（默认 60 分钟），除非有新的状态报告续期。

## 功能特性

### 核心能力

- **自动 Agent 注册**：无需手动设置，首次状态报告时自动注册
- **会话管理**：完整的生命周期管理，支持自动过期
- **实时跟踪**：在任务执行过程中捕获详细的状态更新
- **状态历史**：查询任何 Agent 或会话的历史状态
- **并发安全**：多 Agent 操作的线程安全支持

### 存储选项

- **内存存储**（默认）：快速，无需数据库，适合开发和测试
- **PostgreSQL 存储**：持久化存储，自动迁移，适合生产环境

### 集成特性

- **Webhook 通知**：状态更新时推送到外部服务
- **CORS 支持**：可配置的 CORS 来源，支持跨域请求
- **灵活的 TTL**：为不同任务类型配置会话级别的 TTL

## 部署

### 开发环境部署

本地开发使用内存存储：

```bash
# 安装依赖
go mod download

# 运行服务器
go run main.go
```

服务器默认在 `http://localhost:8080` 启动。

### 生产环境部署

生产环境使用 PostgreSQL 持久化存储：

#### 1. 启动 PostgreSQL

使用 Docker Compose（推荐）：

```bash
docker-compose up -d
```

这将启动 PostgreSQL 并使用默认凭据：
- 主机：`localhost:5432`
- 用户：`kubeagents`
- 密码：`kubeagents`
- 数据库：`kubeagents`

或使用您自己的 PostgreSQL 实例。

#### 2. 配置环境变量

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=kubeagents
export DB_PASSWORD=kubeagents
export DB_NAME=kubeagents

# 可选：自定义服务器设置
export PORT=8080
export CORS_ALLOWED_ORIGINS=http://localhost:3000,https://yourdomain.com
export NOTIFICATION_TIMEOUT_SECONDS=5
```

#### 3. 运行服务器

```bash
# 构建生产版本
go build -o kubeagents-server main.go

# 运行服务器
./kubeagents-server
```

服务器会在启动时自动运行数据库迁移。

### Docker 部署

使用 Docker 构建和运行：

```bash
# 构建镜像
docker build -t kubeagents:latest .

# 使用 PostgreSQL 运行
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

### Kubernetes 部署

对于 Kubernetes 部署，使用提供的部署清单（如果可用）或创建自己的清单，需要考虑以下事项：

- 使用 PostgreSQL StatefulSet 或外部托管数据库
- 在 `/health` 端点配置就绪/存活探针
- 根据预期负载设置资源限制
- 使用 ConfigMap 管理环境变量

## 环境变量

### 服务器配置

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `PORT` | 服务器端口 | `8080` |
| `CORS_ALLOWED_ORIGINS` | 允许的 CORS 来源（逗号分隔） | `*` |
| `NOTIFICATION_TIMEOUT_SECONDS` | Webhook 通知超时时间 | `5` |

### 数据库配置（可选）

如果设置了任何 `DB_*` 变量，则使用 PostgreSQL 存储。否则使用内存存储。

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_USER` | 数据库用户 | - |
| `DB_PASSWORD` | 数据库密码 | - |
| `DB_NAME` | 数据库名称 | - |
| `DB_SSLMODE` | SSL 模式 | `disable` |
| `DB_MAX_OPEN_CONNS` | 最大打开连接数 | `25` |
| `DB_MAX_IDLE_CONNS` | 最大空闲连接数 | `5` |
| `DB_CONN_MAX_LIFETIME` | 连接最大生命周期 | `5m` |

## 与 kubeagents-mcp 集成

要将 kubeagents 与 AI Agent 连接，需要设置 [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp)：

### 快速设置

1. **构建 kubeagents-mcp 服务器**：
   ```bash
   cd kubeagents-mcp
   go mod download
   go build -o kubeagents-mcp-server ./cmd/mcp-server
   ```

2. **在 Cursor 中配置**：
   添加到 Cursor 的 MCP 服务器设置：
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

3. **重启 Cursor**

现在 Cursor 可以调用 MCP 工具向 kubeagents 上报状态了！

### 配置选项

使用以下环境变量配置 kubeagents-mcp：

- `KUBEAGENTS_SERVER_URL`：kubeagents 服务器 URL（默认：`http://localhost:8080`）
- `KUBEAGENTS_AUTO_REPORT_ENABLED`：启用自动上报（默认：`false`）

## Web 界面

使用 [kubeagents-web](https://github.com/kubeagents/kubeagents-web) 进行可视化界面监控 Agent 活动：

- 实时 Agent 状态
- 会话历史和详情
- 状态时间线可视化
- 按用户配置 webhook 通知

## 健康检查

检查服务器状态：

```bash
curl http://localhost:8080/health
```

如果服务器正在运行，返回 `200 OK`。

## 下一步

- 设置 [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp) 连接您的 AI Agent
- 部署 [kubeagents-web](https://github.com/kubeagents/kubeagents-web) 进行可视化监控
- 配置 webhook 通知以获取状态更新
- 查看 [AGENTS.md](./AGENTS.md) 了解 Agent 集成指南

## 相关项目

- [kubeagents-mcp](https://github.com/kubeagents/kubeagents-mcp) - 用于 Agent 状态上报的 MCP 服务器
- [kubeagents-web](https://github.com/kubeagents/kubeagents-web) - 用于监控和管理的 Web 界面
