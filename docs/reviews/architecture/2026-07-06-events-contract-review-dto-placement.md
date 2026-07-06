# 事件契约 DTO 归属 review

## Review 对象

- Diff：`eb3af89^..956b79d`
- 提交：
  - `eb3af89 refactor(content): 提取发布事件契约`
  - `d1e89aa refactor(auth): 提取注册事件契约`
  - `3db43ba refactor(user): 提取用户事件契约`
  - `205d408 refactor(comment): 提取评论事件契约`
  - `956b79d docs(review): 增加全集检查入口`
- 范围：`libs/contracts/events/*`、Auth/User/Comment/Content application outbox 映射、`REVIEW.md` 和文档索引登记。

## 分类判断

- 分类：结构性 / 跨服务 contract review。
- 触发原因：事件 payload 从 application 本地 map / struct 提升到 provider-owned `libs/contracts/events/<domain>`，同时新增根 review 全集检查入口。
- Gate verdict：`pass`。

## Findings

未发现 blocker / major finding。

### Minor

无。

### Note

- 超过 400 行的 touched production 文件仍包括 `services/zhicore-comment/internal/comment/application/service.go`、`services/zhicore-user/internal/user/application/service.go`、`services/zhicore-content/internal/content/application/service.go`。本次 diff 只迁移事件 payload owner，没有继续拆分 use case；长文件拆分可作为后续独立重构处理。
- Application 中仍存在 Content cursor token 的内部 JSON 序列化命中，它是 application 自有不可见分页状态，不属于 HTTP req/resp DTO 或跨服务 payload。

## Review 覆盖

- 全集 diff：检查了 `git diff --stat eb3af89^..956b79d`、`git diff --name-status eb3af89^..956b79d` 和 `git diff --check eb3af89^..956b79d`。
- 长文件：扫描 `services` 和 `libs` 下所有 Go 文件，人工核对 touched surface 中的长文件改动只涉及 outbox payload 映射。
- DTO / payload owner：扫描 `services/*/internal/*/application` 的 `json.Marshal`、`Payload`、`Outbox`、`map[string]any`、`Req/Resp/DTO`、`json` tag、HTTP/Gin 命中。
- 架构依赖方向：确认新增 `libs/contracts/events/*` 不依赖服务 `internal`；application 只导入 provider-owned event contract，不导入 HTTP/Gin/底层 SDK。
- DDD：domain event、integration payload、outbox row 仍分离。User relationship domain event 只表达关系事实，application 映射为 `userevents.*Payload` 后写 outbox；Content publish 仍由 domain 产出发布事实，application 写 `content.post.published` outbox。
- Contract：对照 `docs/contracts/events.md`、`docs/contracts/data-types.md`、`docs/architecture/module/*/data-events.md` 和 `libs/contracts/events/*/*.md`，字段名、时间格式和 optional 字段语义未见漂移。
- 代码质量和注释：未发现无关重构、死代码、兼容 wrapper 或注释与代码冲突；新增注释集中解释事件 payload owner 和敏感字段排除。

## 验证证据

已执行：

```bash
git diff --check eb3af89^..956b79d
python3 tests/architecture/check_boundaries.py --root .
python3 -m unittest tests/architecture/check_boundaries_test.py
make check
```

结果：

- `git diff --check eb3af89^..956b79d` 无输出。
- 架构边界检查输出 `architecture boundaries ok`。
- 架构检查单测 `Ran 8 tests ... OK`。
- `make check` 通过，包含结构检查、架构检查、测试规模检查和所有 Go workspace 模块 `go test ./...`。

## Required re-validation

如果后续继续修改事件 payload 字段、event type、payload version、routing key、outbox envelope 或 consumer contract，需要重新执行：

```bash
make check
```

并补充对应 provider / consumer contract 测试或 application outbox payload 测试。

## Residual risk

- 本次没有新增独立的 `libs/contracts/events` contract 单测；风险由现有 Auth/User/Comment/Content application outbox payload 测试和全量 `make check` 覆盖。若 contract 包后续承载版本演进、枚举或兼容 helper，应补 contract 层单测。
- 事件拓扑和 consumer 处理策略不在本次 diff 内，未做 RabbitMQ 运行时验证。

## 技术债状态

未登记新技术债。长文件拆分属于既有维护性问题，本次没有扩大其职责面。
