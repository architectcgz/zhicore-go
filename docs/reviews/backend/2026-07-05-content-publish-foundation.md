# Content 发布闭环基础 Review

## Review 对象

- 分支：`task/2026-07-05-content-publish-foundation`
- 范围：
  - Content core migration、domain、application、PostgreSQL repository、MongoDB body store。
  - `POST /api/v1/posts`
  - `PUT /api/v1/posts/{postId}/draft/body`
  - `POST /api/v1/posts/{postId}/publish`
  - `GET /api/v1/posts/{postId}/body`
  - Content runtime module 和 `cmd/server` 最小入口。
- 实施计划：`docs/plan/archive/impl-plan/2026-07-05-content-publish-foundation-implementation-plan.md`
- 独立 reviewer：
  - 预实现风险审查：`code-reviewer` subagent `019f318f-60e5-73d0-b6e5-6a5d94229a96`。
  - Task 5 代码复核：`code-reviewer` subagent `019f3198-62a6-7551-b5fa-9dee55ac2e6f`。

## 分类判断

- 分类：非琐碎后端发布闭环基础实现。
- 触发原因：新增 Content schema、domain/application、PostgreSQL/MongoDB adapters、HTTP contract、runtime 装配和 outbox/cleanup/repair 边界。
- Gate verdict：`pass with residual risks`。

## Findings

### Blocker

未发现未修复 blocker。

### 已修复 finding

- `GET /body` 曾在 canonical body 缺失 `blocks` 时返回成功空正文。
  - 状态：已修复。
  - 修复：handler 提取失败时返回 HTTP `500`、body `code=4024`，并补 `TestGetPostBodyRejectsMalformedCanonicalBody`。
- HTTP schema “已验证”表述过宽，容易让未固定 sentinel 的 File / Cover 错误看起来已覆盖。
  - 状态：已修复。
  - 修复：`services/zhicore-content/api/http/README.md` 和 endpoint 文档明确 handler contract 覆盖范围，并登记 `4012` / `4021` / `4023` 待 application / ports sentinel 固定后补测。

### Note

- cleanup / repair / outbox worker 在 runtime 中只返回 disabled descriptor，不伪装成可运行 worker。真实 worker 生命周期、claim、retry 和 dispatcher 后续应作为独立切片实现。
- `cmd/server` 当前只建立进程根和 fail-fast 边界；真实配置加载、依赖打开、HTTP server listen/shutdown 后续应作为可运行服务切片实现。

## 验证证据

已执行并通过：

```bash
cd services/zhicore-content && go test -count=1 ./api/http
cd services/zhicore-content && go test -count=1 ./...
cd services/zhicore-content && go test ./internal/content/runtime
cd services/zhicore-content && go test ./cmd/server
cd services/zhicore-content && go test ./...
python3 scripts/check-test-size.py --files services/zhicore-content
bash scripts/check-structure.sh
git diff --check
```

Task 1 migration 真实数据库验证已执行：

```bash
migrate up -> down 1 -> up
```

验证环境：隔离临时 PostgreSQL 容器 `postgres:16.14-alpine` 和 `migrate/migrate:v4.18.3`。最终 `schema_migrations` 为 `20260705093000 dirty=false`，关键表、索引和 `PUBLISHED` check 约束已确认。

## Required Re-validation

后续修改 Content runtime、HTTP handler、application 发布语义、PostgreSQL/MongoDB adapters 或 migration 时，至少重新执行：

```bash
cd services/zhicore-content && go test ./...
python3 scripts/check-test-size.py --files services/zhicore-content
bash scripts/check-structure.sh
```

触达共享边界、脚手架或跨模块 contract 时，再执行：

```bash
make check
```

## Residual Risk

- 未使用真实 MongoDB DSN 做端到端运行验证；当前 MongoDB adapter 由单元测试覆盖。
- `ZHICORE_CONTENT_POSTGRES_DSN` 未设置；真实 migration 验证使用隔离容器完成，不代表目标部署库已迁移。
- Content HTTP system test 仍待补；当前验证层级是 handler contract test 和服务内 Go test。
- File / Cover 依赖错误 `4012` / `4021` / `4023` 需要先在 application / ports 固定语义错误，再补 handler mapping 和 contract test。
- cleanup / repair / outbox worker 目前为 disabled descriptor，后续需要单独实现 worker runtime、claim、retry、dead-letter 和 shutdown 语义。

## 技术债状态

未新增必须立即处理的阻塞技术债。上述 residual risks 已通过 HTTP schema、runtime health details 和本 review 证据显式记录。
