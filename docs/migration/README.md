# 迁移映射

本文件记录 Java ZhiCore 模块到 Go 落点的映射。

详细设计迁移盘点见 `docs/migration/java-design-migration.md`。该文档记录 Java 侧设计中哪些保留、哪些改写、哪些废弃，以及逐服务迁移风险。

迁移单个服务或服务内 API 族前，先读 `docs/migration/service-migration-workflow.md`，按事实提取、HTTP contract、schema migration、测试策略、Go 实现和交付验证的顺序推进。

## 源仓库

Java 事实源：`../zhicore-microservice`

Java 代码只作为接口、行为和数据模型参考；迁移目标不规划 Java/Go 运行时并存。

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

1. `zhicore-upload`：主要是代理/集成逻辑，API 边界相对清晰。
2. `zhicore-search`：查询型服务，依赖 Elasticsearch 和 RabbitMQ consumer。
3. `zhicore-ranking`：Redis 读模型和定时任务较多，适合在事件模型稳定后迁移。
4. `zhicore-user`、`zhicore-comment`、`zhicore-content`：核心写服务，涉及 PostgreSQL、Redis、事件和跨服务调用。
5. `zhicore-message`、`zhicore-notification`：涉及 WebSocket、推送和事件 fanout。
6. `zhicore-admin`、`zhicore-ops`、`zhicore-gateway`：在核心服务 contract 稳定后迁移。

`zhicore-id-generator` 当前不作为默认核心依赖。内部主键默认使用各服务数据库 `BIGINT` sequence / identity，外部公开 ID 策略见 `docs/architecture/id-strategy.md`；只有未来重新确认集中发号需求时，再把该服务纳入实现顺序。

## 兼容规则

- 前端暂时不修改。Go 服务替换 Java 服务时，必须保留现有外部 API 路径、请求参数、响应封装、字段语义、错误码和权限行为。
- 当前开发阶段不做灰度。Gateway 可以切换本地或部署路由，但不能把 API 形态变化传递给前端。
- 在所有调用方被明确调整并验证前，旧接口必须继续可用；需要重做的接口作为独立 API 演进任务处理。
- 运行时不规划 Java/Go 并存；Go 服务按模块逐步替换对应 Java 实现。
- 优先通过本地环境、网关或部署路由逐个服务接入 Go 目标实现。
- 移除 Java 等价实现前，必须记录已迁移端点和验证证据。
