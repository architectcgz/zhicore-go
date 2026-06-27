# 服务替换实施流程

本文件定义 `zhicore-go` 实现单个 Go 服务或服务内 API 族，并替换既有实现时的标准实施顺序。

本流程不是重型 task gate，也不强制所有改动采用绝对 TDD。它的目标是把 Go 目标设计、已发布外部 contract、schema migration、测试和 Go 实现按固定顺序收口，避免先写代码再补契约。Java 只作为核对既有行为的参考来源。

## 适用范围

- 实现一个完整 Go 目标服务，例如 `zhicore-upload`。
- 实现一个服务内 API 族，例如 `zhicore-user` 的 `/api/v1/auth`。
- 从服务设计、目标产品语义和已发布外部 contract 中固定 Go 实现需要的事实。
- 判断某个服务交付切片是否达到可交付状态。

不适用范围：

- 纯文档索引、目录占位或脚手架登记。
- 独立 API 重新设计；这类任务必须先按 `docs/contracts/README.md` 处理 contract 演进。
- 运行时 Java/Go 并存、灰度或 Gateway 兼容转换方案；当前目标不规划这类模式。

## 服务交付切片原则

- 默认一次只实现一个服务或一个服务内明确 API 族。
- 大服务应按 API 族、use case、worker / consumer 或 repository 边界拆分，不要求单次完成整个服务。
- 同一次切片不能混入多个服务的无关实现，除非它们是同一个 contract 变更的 provider / consumer 必要联动。
- 服务交付切片必须能说明 Go 落点、对外 contract、数据归属、测试范围、验证命令，以及是否需要参考既有实现。
- 前端暂时不修改。Go 服务替换既有实现时必须保持已发布外部 API 路径、字段、响应 envelope、错误码和权限行为，除非服务级 schema 已登记 Go-first API reset。

## 前置读取

开始实现前至少读取：

- `docs/migration/README.md`
- `docs/migration/java-design-migration.md`
- `docs/architecture/services/<service>/README.md`
- `docs/architecture/service-boundaries.md`
- `docs/architecture/go-service-design.md`
- `docs/contracts/http-schema-template.md`
- `docs/architecture/testing.md`
- `docs/reviews/quality-gates.md`

按改动面追加读取：

- HTTP contract：`docs/contracts/http.md`、`docs/contracts/errors.md`、`docs/contracts/error-codes.md`、`docs/contracts/data-types.md`、`docs/contracts/pagination.md`
- database migration：`docs/architecture/migrations.md`
- 安全权限：`docs/architecture/security.md`
- 运行期和配置：`docs/architecture/runtime-operations.md`、`docs/architecture/configuration.md`
- 事件和异步：`docs/contracts/events.md`
- 可观测性：`docs/architecture/observability.md`

## 标准顺序

### 1. 明确服务交付切片

先写清：

- 服务名和 Go 模块。
- 本次 API 族、endpoint、worker / consumer、repository 或 use case 范围。
- 不在本次处理的相邻能力。
- 已知 provider / consumer 和跨服务 contract。
- 预期验证命令。

如果范围无法在一段话里说清，先拆小再实现。

### 2. 固定 Go 目标事实

从服务设计、目标产品语义和已发布外部 contract 固定并记录：

- controller path、method、header、query、path variable、body、multipart 字段。
- request / response DTO 字段名、类型、必填、默认值、空值语义。
- `ApiResponse` envelope、HTTP status、错误码、错误 message 语义。
- 权限、登录态、角色、资源可见性、幂等、分页、排序和过滤语义。
- service / repository 行为、事务边界、状态机和副作用。
- SQL 表、索引、唯一约束、默认值、枚举和种子数据。
- typed client、事件 payload、routing key、producer 和已知 consumer。

规则：

- 不用猜测替代事实核对；找不到来源时记录“待核对”，不要写成已确认。
- 既有行为可能保留、改写或废弃，决策必须落到相应架构或 contract 文档。
- 不把既有全量初始化 SQL 原样复制到 Go 服务，必须按服务数据归属拆分。

### 3. 固定 HTTP contract

涉及 HTTP endpoint 时，先更新：

```text
services/<service>/api/http/README.md
services/<service>/api/http/endpoints/<operation>.md
```

要求：

- 每个 endpoint 单独记录 path、method、字段、响应 `data`、错误码、权限和分页/过滤规则。
- 历史入口、历史字段、异常 HTTP status 或错误码必须写成兼容例外。
- contract 状态先用“草案”；Go handler test 或 system HTTP test 证明后再标记“已验证”。
- 不要为了 Go 实现方便改变外部 API。确需重做时，按独立 API 演进任务处理。

### 4. 固定数据和 migration

涉及 PostgreSQL schema 时：

- 先确认表和字段归属符合 `docs/architecture/service-boundaries.md`。
- 正式 schema 放在 `services/<service>/migrations/`。
- 使用 `golang-migrate` 创建成对 `.up.sql` / `.down.sql`。
- 不创建跨服务外键，不跨服务数据库 join。
- GORM model、tag 或 `AutoMigrate` 不能替代正式 migration。

本阶段应同步判断：

- 内部主键和外部公开 ID 是否符合 `docs/architecture/id-strategy.md`。
- 是否需要 outbox / inbox / ledger。
- 是否存在不可逆 down migration、锁风险或数据修复风险。

### 5. 定义 application 行为和端口

实现前先收口服务内边界：

- handler 负责协议绑定、认证上下文映射、参数校验和响应转换。
- application 负责 use case、权限、事务、幂等、事件写入和端口调用。
- repository / adapter 负责基础设施细节和底层错误翻译。
- domain 只表达稳定业务规则，不依赖框架、数据库或 HTTP DTO。
- 跨服务调用通过 `libs/contracts/clients/<provider-service>` 或 provider-owned contract。

权限、错误、运行期和观测必须同步判断：

- 权限 owner：见 `docs/architecture/security.md`。
- 内部错误和对外错误码映射：见 `docs/architecture/error-handling.md` 和 `docs/contracts/error-codes.md`。
- context、timeout、retry、幂等和停机：见 `docs/architecture/runtime-operations.md`。
- operation、日志字段和脱敏：见 `docs/architecture/observability.md`。

### 6. 制定测试策略

测试按风险分级执行，不要求所有服务交付切片绝对 TDD。

必须新增 focused test 的常见情况：

- 新 endpoint、新 use case、新 repository、新错误码、新权限/分页/过滤/字段 contract。
- bugfix、事务、幂等、migration、worker / consumer、跨服务 contract、数据一致性。

优先测试层级：

- Handler / HTTP test：路由、鉴权、请求字段、响应 envelope、公开错误码。
- Application test：权限、状态机、幂等、事务编排、事件写入、端口调用结果。
- Repository test：查询、唯一约束、事务语义、错误翻译。
- Runtime / system test：真实依赖、端口、容器、消息、外部服务协作。

测试文件必须按行为、endpoint、use case、repository query 或 worker 场景拆分，遵守 `docs/architecture/testing.md` 的规模限制。

### 7. 实现 Go 代码

推荐实现顺序：

1. 补齐 contract / schema 文档。
2. 定义 application input / output、端口和错误映射。
3. 实现最小 domain / application 行为。
4. 实现 repository / adapter。
5. 实现 HTTP handler / middleware 接线。
6. 实现 runtime module 装配、配置校验和健康检查。
7. 接入事件、worker / consumer 或 outbox。
8. 更新 endpoint schema 状态和服务文档。

规则：

- 不在 `cmd/server` 放业务逻辑。
- 不让 handler 直接访问数据库、缓存、MQ 或外部 SDK。
- 不把服务私有模型提升到 `libs/contracts` 或 `libs/kit`。
- 不把 Gateway 做成 API 形态转换层。
- 不在普通服务启动路径自动执行 migration。

### 8. 验证和交付

验证从最窄相关命令开始：

```bash
cd services/<service> && go test ./path/...
bash scripts/check-structure.sh
make test-size
make check
```

选择规则：

- 只改文档、schema 或索引：至少运行 `bash scripts/check-structure.sh`。
- 改 handler / application / repository：运行受影响包测试。
- 改测试组织：运行 `make test-size`。
- 改 migration：验证 `up` 和最近一条 `down 1`；不可逆 down 写明人工确认点。
- 改共享 contract、脚手架、多个模块或服务边界：交付前优先运行 `make check`。
- 涉及并发、worker、consumer、cache 或共享状态时，说明是否运行 `go test -race`。

交付说明必须列出实际执行过的命令。没有运行的命令不能写成已通过。

## 完成标准

一个服务交付切片达到可交付状态时，应满足：

- 目标 Go contract 已固定；需要参考既有行为的部分已核对，未知点没有被写成确定事实。
- 受影响 HTTP contract 已记录到 `services/<service>/api/http`。
- 数据归属、migration、ID、错误码、权限和事件边界已收口。
- Go 实现符合 `api/http`、`internal`、`libs/contracts`、`libs/kit` 的边界。
- 行为改动有测试或明确手动验证证据。
- 需要 review 的高风险面已按 `docs/reviews/done-definition.md` 处理。
- 已知未收口问题只在不影响当前 touched surface 时登记到 `docs/todos/debt/`，并写清退出条件。

## 禁止事项

- 先实现 Go handler，再回头猜 contract。
- 为实现方便修改前端依赖的 path、字段、envelope、错误码或权限语义。
- 一次实现多个无关服务。
- 让 Gateway 承担业务聚合、资源权限或响应转换。
- 在服务启动路径执行 schema migration。
- 复制其他服务数据库表、repository 或 `internal` 包。
- 删除或放宽测试来迁就错误实现。
