# 服务设计索引

本目录记录各个 Go 目标服务的模块级设计。这里的文档是从 Java 侧 controller、schema、client contract、事件定义和架构文档迁移出的 Go 目标设计，不表示对应服务已经完成 Go 实现。

## 阅读顺序

1. 先读 `docs/architecture/service-boundaries.md`，确认服务边界和数据归属。
2. 再读 `docs/architecture/repository-layout.md`，确认 `api/http` 与 `internal` 的落点。
3. 再读对应服务文档。
4. 实现服务前，继续从 Java controller、DTO、schema 和测试提取字段级 contract。

## 服务文档

| 服务 | 文档 |
| --- | --- |
| Gateway | `gateway.md` |
| User | `user.md` |
| Content | `content.md` |
| Comment | `comment.md` |
| Message | `message.md` |
| Notification | `notification.md` |
| Search | `search.md` |
| Ranking | `ranking.md` |
| Admin | `admin.md` |
| Upload | `upload.md` |
| ID Generator | `id-generator.md` |
| Ops | `ops.md` |

## 当前迁移状态

- 已迁移：服务职责、数据归属、主要 API 族、跨服务依赖、事件方向和 Go 落点。
- 未完成：字段级 HTTP request/response schema、按 `docs/architecture/migrations.md` 落地的完整 migration SQL、服务级行为测试清单。
- 下一步：按 `docs/contracts/http-schema-template.md`，逐服务把 Java controller 和 DTO 提取到 `services/<service>/api/http/README.md` 和 `services/<service>/api/http/endpoints/`；按 `docs/architecture/migrations.md` 补 migration 草案。
