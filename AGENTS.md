# AGENTS.md

本文档提供给 AI Agent 使用，用于在使用 kubeagents-mcp 工具上报开发状态时的参考。

## kubeagents-mcp 工具使用说明

kubeagents-mcp 提供 MCP 协议工具，用于向 KubeAgents 系统上报会话状态。详细的工具使用说明请参考 [kubeagents-mcp/AGENTS.md](../kubeagents-mcp/AGENTS.md)。

### 可用工具

1. **start_session** - 启动新的会话，在任务开始时调用
2. **report_status** - 上报任务状态，在任务执行到关键节点时调用
3. **end_session** - 结束会话并上报最终状态，在任务完成或失败时调用

### 状态值说明

| 状态值 | 含义 | 使用场景 |
|--------|------|----------|
| `running` | 任务执行中 | 任务开始或执行到关键节点时 |
| `success` | 任务成功 | 任务成功完成（仅用于 end_session） |
| `failed` | 任务失败 | 任务失败（可用于 report_status 和 end_session） |
| `pending` | 任务等待中 | 任务需要等待外部资源或事件时 |

### 最佳实践

1. **保持参数一致性**：在整个任务流程中，`agent_id` 和 `session_topic` 必须保持一致
2. **关键节点上报**：在任务的重要节点及时调用 `report_status`
3. **会话管理**：每个会话必须有对应的 `start_session` 和 `end_session`

## kubeagents 项目特定配置

在 kubeagents 项目中进行开发时，使用以下参数：

- `agent_name`: "kubeagents-dev-agent"
- `agent_source`: "cursor-ai"（或实际使用的 AI 助手）
- `agent_id`: 使用会话主题（必填）
- `session_topic`: 使用会话主题（必填）

### 调用示例

#### 任务开始
```
调用：start_session
参数：
  - agent_id: "实现 /api/agents/{agent_id}/status 端点"
  - session_topic: "实现 /api/agents/{agent_id}/status 端点"
  - agent_name: "kubeagents-dev-agent"
  - agent_source: "cursor-ai"
  - ttl_minutes: 60
```

#### 关键节点进展
```
调用：report_status
参数：
  - agent_id: "实现 /api/agents/{agent_id}/status 端点"
  - session_topic: "实现 /api/agents/{agent_id}/status 端点"
  - status: "running"
  - message: "已完成数据模型定义"
```

#### 任务完成
```
调用：end_session
参数：
  - agent_id: "实现 /api/agents/{agent_id}/status 端点"
  - session_topic: "实现 /api/agents/{agent_id}/status 端点"
  - status: "success"
  - message: "功能实现完成，已通过测试"
```

## 相关文档

- [kubeagents-mcp/AGENTS.md](../kubeagents-mcp/AGENTS.md) - kubeagents-mcp 详细使用说明
- [specs/001-kubeagents-system/contracts/api.yaml](../specs/001-kubeagents-system/contracts/api.yaml) - API 规范
