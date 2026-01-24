# kubeagents

**用于 Agent 编排和状态管理的后端服务**

KubeAgents 是一个 REST API 服务器，用于接收和管理 AI Agent 的状态上报。它提供自动 Agent 注册、会话生命周期管理和实时状态跟踪功能。

## 许可证

根据 Apache License, Version 2.0 许可。详情请参见 [LICENSE](LICENSE)。

## 功能特性

- **状态 Webhook**: 通过 HTTP webhook 接收 Agent 状态上报
- **REST API**: 查询 Agent 和会话信息
- **自动管理**: 自动注册 Agent 并管理会话生命周期
- **会话过期**: 基于 TTL 自动过期非活跃会话
- **并发安全**: 线程安全的内存存储，支持高性能操作

## 快速开始

```bash
go mod download
go run main.go
```

服务器默认运行在 `http://localhost:8080`。

## 环境变量

- `PORT`: 服务器端口（默认: `8080`）
- `CORS_ALLOWED_ORIGINS`: 允许的 CORS 来源（逗号分隔，默认: `*`）

Webhook 通知地址在 Web 页面按用户配置，不再通过环境变量设置。

## API 文档

详细 API 规范请参见 [contracts/api.yaml](../specs/001-kubeagents-system/contracts/api.yaml)。

### 接口端点

- `GET /health` - 健康检查端点
- `POST /webhook/status` - 接收 Agent 状态上报
- `GET /api/agents` - 列出所有 Agent
- `GET /api/agents/{agent_id}` - 获取 Agent 详情
- `GET /api/agents/{agent_id}/sessions` - 列出 Agent 会话
- `GET /api/agents/{agent_id}/sessions/{session_topic}` - 获取会话详情
- `GET /api/agents/{agent_id}/status` - 获取 Agent 状态历史

## 测试

```bash
go test ./...
go test -race ./...  # 并发竞争测试
```

## 架构

- **Handlers**: API 端点的 HTTP 请求处理器
- **Store**: 线程安全操作的内存数据存储
- **Models**: Agent、Session 和 AgentStatus 的数据模型
- **Config**: 支持环境变量的配置管理

## 项目结构

```
kubeagents/
├── handlers/          # HTTP 处理器
├── models/            # 数据模型
├── store/             # 内存存储
├── config/            # 配置
├── internal/          # 内部工具
└── main.go           # 应用入口点
```
