# 迁移映射

本文件记录 Java ZhiCore 模块到 Go 落点的映射。

## 源仓库

Java 事实源：`../zhicore-microservice`

## 可部署服务

| Java 模块 | Go 服务模块 |
| --- | --- |
| `zhicore-gateway` | `services/zhicore-gateway` |
| `zhicore-user` | `services/zhicore-user` |
| `zhicore-content` | `services/zhicore-content` |
| `zhicore-comment` | `services/zhicore-comment` |
| `zhicore-message` | `services/zhicore-message` |
| `zhicore-notification` | `services/zhicore-notification` |
| `zhicore-search` | `services/zhicore-search` |
| `zhicore-ranking` | `services/zhicore-ranking` |
| `zhicore-admin` | `services/zhicore-admin` |
| `zhicore-upload` | `services/zhicore-upload` |
| `zhicore-id-generator` | `services/zhicore-id-generator` |
| `zhicore-ops` | `services/zhicore-ops` |

## Java 共享模块

| Java 模块 | Go 落点 | 说明 |
| --- | --- | --- |
| `zhicore-common` | `libs/kit` | 响应封装、错误、认证、配置、持久化、可观测性、基础设施原语。 |
| `zhicore-client` | `libs/contracts/clients` | typed service client 和同步调用契约。 |
| `zhicore-integration` | `libs/contracts/events` | 跨服务事件 payload 和消息 contract。 |

## 推荐迁移顺序

1. `zhicore-id-generator`：HTTP 面最小，适合验证 Go 服务部署链路。
2. `zhicore-upload`：主要是代理/集成逻辑，API 边界相对清晰。
3. `zhicore-search`：查询型服务，依赖 Elasticsearch 和 RabbitMQ consumer。
4. `zhicore-ranking`：Redis 读模型和定时任务较多，适合在事件模型稳定后迁移。
5. `zhicore-user`、`zhicore-comment`、`zhicore-content`：核心写服务，涉及 PostgreSQL、Redis、事件和跨服务调用。
6. `zhicore-message`、`zhicore-notification`：涉及 WebSocket、推送和事件 fanout。
7. `zhicore-admin`、`zhicore-ops`、`zhicore-gateway`：在核心服务 contract 稳定后迁移。

## 兼容规则

- 在调用方被明确调整前，保留现有外部 API 路径和响应封装。
- 迁移期间 Java 和 Go 服务并行存在。
- 优先通过现有网关或部署路由逐个服务替换。
- 移除 Java 等价实现前，必须记录已迁移端点和验证证据。
