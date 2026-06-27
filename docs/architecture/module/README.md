# 模块架构文档索引

本目录记录单个模块内部的长期架构设计。这里的“模块”通常对应一个 Go 服务内的业务模块，例如 `content`、`upload`、`user`，也可以对应服务内一个足够大的 API family。

## 定位

- `docs/architecture/services/` 保留服务级总览、服务边界、跨服务依赖和设计图。
- `docs/architecture/module/<module>/` 承载模块内部的 API 背后设计、application service、domain、ports、数据和事件细节。
- `services/<service>/api/http/` 只记录 HTTP 字段级 contract，不写完整 application / domain 设计。

## 推荐结构

```text
docs/architecture/module/<module>/
├── README.md
├── api.md
├── service.md
├── domain.md
├── ports.md
├── data-events.md
├── adr/
└── decision-log/
```

| 文件 | 内容 |
| --- | --- |
| `README.md` | 模块职责、边界、API family、实现切片、关联服务和当前状态。 |
| `api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪；不写字段级 HTTP schema。 |
| `service.md` | application service / use case、事务边界、幂等、并发和错误映射。 |
| `domain.md` | 聚合、实体、值对象、不变量和状态转换。 |
| `ports.md` | repository、cache、client、event publisher、outbox、external adapter 等端口归属。 |
| `data-events.md` | 数据归属、缓存 key、事件 payload 归属、typed client 和一致性规则。 |
| `adr/` | 模块内难以逆转、需要解释取舍的架构决策。 |
| `decision-log/` | 重要设计压测、评审或讨论的复盘记录。 |

## 关联方式

HTTP contract 通过 `services/<service>/api/http/README.md` 的“API 到设计追踪”表关联到本目录：

```markdown
| Endpoint | Use case | 设计文档 | Contract 状态 | 测试状态 |
| --- | --- | --- | --- | --- |
| `POST /api/v1/posts/{postId}/publish` | `PublishDraft` | `docs/architecture/module/content/service.md` | 草案 | 待补 |
```

Endpoint 文档在“来源”和“设计追踪”区块中引用对应模块文档：

```markdown
- 模块 API 设计：`docs/architecture/module/content/api.md`
- 模块 service 设计：`docs/architecture/module/content/service.md`
```

## 迁移约定

既有 `docs/architecture/services/<service>/` 下已经拆出的专题设计不要求一次性搬迁。后续编辑某个模块专题时，优先把新增或重写的 API / service / domain / ports 设计落到本目录，并在原服务 README 中保留链接。
