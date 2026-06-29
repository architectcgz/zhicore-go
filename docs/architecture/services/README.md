# 服务设计索引

本目录记录各个 Go 目标服务的模块级设计。这里的文档以 Go 服务边界、Go contract、Go 运行模型和数据归属为事实源；Java 侧 controller、schema、client contract、事件定义和架构文档只作为既有能力参考，不表示对应服务已经完成 Go 实现。

## 阅读顺序

1. 先读 `docs/architecture/service-boundaries.md`，确认服务边界和数据归属。
2. 再读 `docs/architecture/repository-layout.md`，确认 `api/http` 与 `internal` 的落点。
3. 再读对应服务文档。
4. 实现服务前，先固定服务级 Go contract；需要确认既有行为或已发布接口时，再核对 Java controller、DTO、schema 和测试。

## 服务文档

每个服务使用独立目录归档长期设计正文和导出图：

```text
docs/architecture/services/<service>/
├── README.md
├── <topic>.md
├── service-design.<service>.png
├── service-detail.drawio
└── service-detail.png
```

| 服务 | 文档 | 设计图 | 详细图源 | 详细图 |
| --- | --- | --- | --- | --- |
| Gateway | `gateway/README.md` | `gateway/service-design.gateway.png` | `gateway/service-detail.drawio` | `gateway/service-detail.png` |
| Auth | `auth/README.md` | 待补 | 待补 | 待补 |
| User | `user/README.md` | `user/service-design.user.png` | `user/service-detail.drawio` | `user/service-detail.png` |
| Content | `content/README.md` | `content/service-design.content.png` | `content/service-detail.drawio` | `content/service-detail.png` |
| Comment | `comment/README.md` | `comment/service-design.comment.png` | `comment/service-detail.drawio` | `comment/service-detail.png` |
| Message | `message/README.md` | `message/service-design.message.png` | `message/service-detail.drawio` | `message/service-detail.png` |
| Notification | `notification/README.md` | `notification/service-design.notification.png` | `notification/service-detail.drawio` | `notification/service-detail.png` |
| Search | `search/README.md` | `search/service-design.search.png` | `search/service-detail.drawio` | `search/service-detail.png` |
| Ranking | `ranking/README.md` | `ranking/service-design.ranking.png` | `ranking/service-detail.drawio` | `ranking/service-detail.png` |
| Admin | `admin/README.md` | `admin/service-design.admin.png` | `admin/service-detail.drawio` | `admin/service-detail.png` |
| Upload | `upload/README.md` | `upload/service-design.upload.png` | `upload/service-detail.drawio` | `upload/service-detail.png` |
| ID Generator | `id-generator/README.md` | `id-generator/service-design.id-generator.png` | `id-generator/service-detail.drawio` | `id-generator/service-detail.png` |
| Ops | `ops/README.md` | `ops/service-design.ops.png` | `ops/service-detail.drawio` | `ops/service-detail.png` |

Gateway 的服务级专题文档：

| 专题 | 文档 |
| --- | --- |
| 入口和关键结论 | `gateway/README.md` |
| Redis/Auth 降级下的路由风险策略 | `gateway/route-risk-policy.md` |

Content 服务文档已经拆成专题文件：

| 专题 | 文档 |
| --- | --- |
| 入口和关键结论 | `content/README.md` |
| 术语表 | `content/CONTEXT.md` |
| 架构决策记录 | `content/adr/` |
| 设计问答复盘 | `content/decision-log/` |
| 领域模型 | `content/domain-model.md` |
| 正文存储与发布 | `content/body-storage-and-publishing.md` |
| Application、Ports 与实现切片 | `content/application-and-ports.md` |
| 数据、事件和契约 | `content/data-events-contracts.md` |
| 限流和频控 | `content/rate-limiting.md` |
| 运行期 resilience | `content/runtime-resilience.md` |

Comment 的服务级文档只保留边界、API 族和模块入口；模块内部设计已迁移到 `docs/architecture/module/comment/`：

| 专题 | 文档 |
| --- | --- |
| 入口和关键结论 | `comment/README.md` |
| 模块总览 | `../module/comment/README.md` |
| API 背后设计 | `../module/comment/api.md` |
| Application service | `../module/comment/service.md` |
| Domain | `../module/comment/domain.md` |
| Ports | `../module/comment/ports.md` |
| 数据和事件 | `../module/comment/data-events.md` |

User 的服务级文档只保留服务边界、API 范围和模块入口；模块内部设计已迁移到 `docs/architecture/module/user/`：

| 专题 | 文档 |
| --- | --- |
| 入口和关键结论 | `user/README.md` |
| 模块总览 | `../module/user/README.md` |
| API 背后设计 | `../module/user/api.md` |
| Application service | `../module/user/service.md` |
| Domain | `../module/user/domain.md` |
| Ports | `../module/user/ports.md` |
| 数据和事件 | `../module/user/data-events.md` |
| 运行期 resilience | `../module/user/runtime-resilience.md` |
| 限流和频控 | `../module/user/rate-limiting.md` |
| 决策日志 | `../module/user/decision-log.md` |

Ranking 的服务级专题文档：

| 专题 | 文档 |
| --- | --- |
| 入口和关键结论 | `ranking/README.md` |
| 设计决策日志 | `ranking/decision-log/2026-06-29-ranking-design-decisions.md` |
| Admin rebuild | `ranking/admin-rebuild.md` |
| 领域模型 | `ranking/domain-model.md` |
| Application、Ports 与事务 | `ranking/application-and-ports.md` |
| 数据事件与投影 | `ranking/data-events-projections.md` |
| 事件顺序与分片 | `ranking/event-ordering-and-partitioning.md` |
| 查询、缓存与物化 | `ranking/query-materialization.md` |
| 运行期 resilience | `ranking/runtime-resilience.md` |
| Schema、配置与实现切片 | `ranking/schema-and-implementation.md` |

## 服务设计图

图表源文件：

- `_overview/service-design.drawio`：服务设计图集源文件，包含总览页和每个服务的目标设计页。

渲染图片：

| 视角 | 图片 |
| --- | --- |
| 总览 | `_overview/service-design.overview.png` |
| Gateway | `gateway/service-design.gateway.png` |
| Auth | 待补 |
| User | `user/service-design.user.png` |
| Content | `content/service-design.content.png` |
| Comment | `comment/service-design.comment.png` |
| Message | `message/service-design.message.png` |
| Notification | `notification/service-design.notification.png` |
| Search | `search/service-design.search.png` |
| Ranking | `ranking/service-design.ranking.png` |
| Admin | `admin/service-design.admin.png` |
| Upload | `upload/service-design.upload.png` |
| ID Generator | `id-generator/service-design.id-generator.png` |
| Ops | `ops/service-design.ops.png` |

## 当前设计状态

- 已明确：服务职责、数据归属、主要 API 族、跨服务依赖、事件方向和 Go 落点。
- 未完成：字段级 HTTP request/response schema、按 `docs/architecture/migrations.md` 落地的完整 migration SQL、服务级行为测试清单。
- 下一步：按 `docs/contracts/http-schema-template.md`，逐服务把目标 HTTP schema 固定到 `services/<service>/api/http/README.md` 和 `services/<service>/api/http/endpoints/`；需要承接已发布行为时再核对 Java controller 和 DTO；按 `docs/architecture/migrations.md` 补 migration 草案。
