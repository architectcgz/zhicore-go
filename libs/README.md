# 共享库

`libs/` 存放 Go 服务之间共享的独立模块。

- `contracts`：跨服务 DTO、事件 payload 和 typed client 契约。
- `kit`：小型共享技术原语，例如 HTTP 响应封装、认证、配置、可观测性、PostgreSQL、Redis、MongoDB、RabbitMQ、Elasticsearch 客户端封装。

业务规则必须留在 `services/<service>/internal`。不要为了复用方便把服务私有模型、仓储、查询条件或业务决策放进共享库。
