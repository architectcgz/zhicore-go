# API 设计文档规范

本文件定义 `zhicore-go` 中 API 背后设计、HTTP contract 和实现追踪的文档结构。它回答“一个 API 是否已经可以实现”的判定标准。

## 分层事实源

| 层级 | 路径 | 负责内容 | 不负责内容 |
| --- | --- | --- | --- |
| 服务总览 | `docs/architecture/services/<service>/` | 服务职责、API 族范围、跨服务依赖、数据归属总览、实现风险、设计图索引 | 字段级 HTTP request/response、模块内部完整 service/domain/ports 设计 |
| 模块设计 | `docs/architecture/module/<module>/` | API 背后设计、application service、领域模型、状态机、事务边界、ports、数据归属、事件、实现切片 | 字段级 HTTP request/response |
| HTTP contract | `services/<service>/api/http/` | path、method、字段、错误码、权限、分页、排序、幂等、测试要求 | repository 查询细节、handler 内部流程、数据库字段清单 |
| API 追踪索引 | `services/<service>/api/http/README.md` | endpoint 到 use case、聚合、设计文档、测试状态的映射 | 重复展开完整领域设计 |

服务总览、模块设计和 HTTP contract 都是事实源，但 owner 不同：服务总览说明“这个服务拥有和依赖什么”，模块设计说明“为什么这样做”和“业务状态怎么变”，HTTP contract 说明“外部如何调用”和“返回什么”。

## 文件结构

默认结构：

```text
docs/architecture/services/<service>/
├── README.md
├── service-detail.drawio
└── service-detail.png

docs/architecture/module/<module>/
├── README.md
├── api.md
├── service.md
├── domain.md
├── ports.md
├── data-events.md
├── <api-family-or-topic>.md
├── adr/
└── decision-log/

services/<service>/api/http/
├── README.md
└── endpoints/
    ├── <family>-<operation>.md
    └── <family>.md
```

| 文件 | 使用条件 | 内容边界 |
| --- | --- | --- |
| `docs/architecture/services/<service>/README.md` | 每个服务必备 | 服务职责、API 族范围、数据归属总览、跨服务依赖、实现风险、下一步切片、模块设计链接 |
| `docs/architecture/module/<module>/README.md` | 模块开始设计或实现时必备 | 模块职责、边界、API family、实现切片、关联服务和当前状态 |
| `docs/architecture/module/<module>/api.md` | API 背后业务设计较多时拆出 | API family 背后的业务流程、权限、状态机和副作用，不写字段级 schema |
| `docs/architecture/module/<module>/service.md` | 用例、事务或 application service 较多时拆出 | application use case、事务边界、幂等、错误映射、包落点、测试切片 |
| `docs/architecture/module/<module>/domain.md` | 聚合、值对象或状态机较多时拆出 | 聚合和值对象、不变量、状态转换、领域服务 |
| `docs/architecture/module/<module>/ports.md` | ports 或 adapters 较多时拆出 | repository、cache、client、outbox、event publisher、external adapter 的端口归属 |
| `docs/architecture/module/<module>/data-events.md` | 数据归属、事件或跨服务 contract 较多时拆出 | 表归属、缓存、事件、typed client、公开错误和兼容约束 |
| `docs/architecture/module/<module>/<api-family-or-topic>.md` | 某个 API 族背后业务设计复杂时拆出 | 该 API 族背后的业务流程、权限、状态机和副作用，不写字段级 schema |
| `services/<service>/api/http/README.md` | 服务开始设计或实现 HTTP API 时必备 | 服务级公共规则、endpoint 索引、API 到设计追踪、服务级错误码 |
| `services/<service>/api/http/endpoints/<family>-<operation>.md` | 默认 endpoint 文档形式 | 单个 endpoint 的字段级 contract |
| `services/<service>/api/http/endpoints/<family>.md` | Go-first API reset 或一次固定完整 API family 草案时可用 | 一个 API family 或完整服务 API 面；后续实现时可再拆单 endpoint |

不要为尚未固定目标 schema 的 endpoint 创建“占位完成”文档；应在模块设计或 HTTP README 的待提取项中记录。

## 服务设计写法

服务总览文档只写服务级边界和模块入口；模块设计文档优先使用表格写清 API 背后的 owner 和边界。

| 区块 | 必须写明 |
| --- | --- |
| 职责边界 | 本服务拥有和不拥有的业务状态、禁止 facade、委托边界 |
| API 范围 | API family 或关键 endpoint，说明保留旧形态还是 Go-first reset |
| 领域模型 | 聚合、值对象、状态字段、不变量和生命周期 |
| Application 用例 | 每个 API 背后的 command/query use case、权限、事务和副作用 |
| Ports | repository、cache、client、outbox、external adapter 的接口 owner |
| 数据归属 | 表、缓存 key、外部系统事实、是否允许冗余快照 |
| 事件 | 生产事件、消费事件、outbox、幂等和失败处理 |
| 实现切片 | 首个最小 API 族、后续切片、暂不实现项 |
| 风险 | 兼容、并发、权限、分页、缓存一致性和降级风险 |

模块设计不重复 HTTP 字段表。字段名、必填、默认值、错误码和分页参数进入 `services/<service>/api/http/`。

## HTTP README 写法

每个 `services/<service>/api/http/README.md` 至少包含：

| 区块 | 内容 |
| --- | --- |
| 来源 | 服务总览、模块设计、Go handler / test、Java 参考来源；Java 只在需要核对已发布行为时列出 |
| 公共规则 | envelope、认证 header、ID、时间、错误码、分页、排序、幂等 |
| Endpoint 索引 | 方法、路径、文档、状态 |
| API 到设计追踪 | endpoint、use case、设计文档、测试状态 |
| 服务级公开错误码 | code、HTTP status、含义、适用场景 |
| 待提取 contract | 尚未固定字段级 schema 的 API family 或 endpoint |

推荐追踪表：

| Endpoint | Use case | 设计文档 | Contract 状态 | 测试状态 |
| --- | --- | --- | --- | --- |
| `POST /api/v1/...` | `Create...` | `docs/architecture/module/<module>/service.md` | 草案 | 待补 |

## Endpoint 文档写法

单个 endpoint 文档按 `docs/contracts/http-schema-template.md` 编写，并额外保持设计追踪。

| 区块 | 必须写明 |
| --- | --- |
| 来源 | 服务总览、模块设计、当前 HTTP README、handler、contract test、必要 Java 参考 |
| 请求 | method、path、Content-Type、鉴权、幂等 |
| 参数 | path、query、body、multipart 字段；类型、必填、默认值、空值语义 |
| 响应 | `data` 字段、必填、空值语义、ID / 时间格式 |
| 错误 | code、HTTP status、message 语义、触发条件 |
| 权限和可见性 | owner 校验、管理员、匿名访问、资源状态过滤 |
| 排序、分页和过滤 | 列表接口必须说明稳定排序、分页模型和过滤字段；非列表写“无” |
| 设计追踪 | use case、聚合、ports、事件、事务边界 |
| 测试要求 | handler contract test、system HTTP test、待补状态 |

## 状态判定

| 状态 | 判定标准 | 是否可实现 |
| --- | --- | --- |
| API 族已识别 | 服务总览或模块设计只列出 API family 或路径范围 | 否，只能作为计划 |
| 设计已说明 | 已写明 use case、权限、状态机、事务和数据归属 | 仍需补字段级 contract |
| Contract 草案 | 已写明 path、字段、响应、错误、权限和测试要求 | 可以开始写 handler / test |
| Contract 已验证 | 已有 handler contract test 或 system HTTP test 覆盖 | 可以作为当前实现事实源 |
| 兼容例外 | 明确保留历史 path、字段、HTTP status 或错误码 | 可以实现，但必须写清原因和删除条件 |

实现或修改 HTTP handler 前，最低要求是达到 `Contract 草案`。已经存在 handler 但缺少 HTTP schema 的服务，必须先补 schema，再继续扩展 handler。

## 落地优先级

| 优先级 | 服务 | 原因 | 推荐动作 |
| --- | --- | --- | --- |
| 1 | Upload | 已有 handler 和测试，但缺 HTTP schema | 补 `api/http/README.md` 和 6 个 endpoint contract |
| 2 | Content | 已有 Go-first 大草案 | 按实现切片拆出单 endpoint 或 API family contract |
| 3 | User / Comment / Notification / Ranking | 服务设计细，但字段级 contract 未固定 | 按首个实现切片提取 HTTP schema |
| 4 | Gateway / Admin / Message / Search / ID Generator / Ops | 需先确认保留范围或是否实现 | 先补服务级 HTTP README 的待提取项，再按切片展开 |

## 验证

| 变更类型 | 最小验证 |
| --- | --- |
| 只新增或修改 API 设计规范 | `bash scripts/check-structure.sh`、`git diff --check` |
| 新增服务级 HTTP README 或 endpoint 文档 | `bash scripts/check-structure.sh`、`git diff --check` |
| 修改已有 handler 对应 contract | 最窄相关 `go test`，再视路径变化运行 `bash scripts/check-structure.sh` |
| 标记 endpoint 为已验证 | 必须有 handler contract test 或 system HTTP test 证据 |
