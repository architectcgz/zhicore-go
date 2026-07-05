# Comment 创建分页基础实现 Review

## 范围

- `zhicore-comment` 创建根评论 / 回复 application、HTTP handler 和核心 migration。
- 顶级评论传统分页 application 和 HTTP handler。
- 实施计划：`docs/plan/archive/impl-plan/2026-07-04-comment-create-page-foundation-implementation-plan.md`。

## Review 对象

- diff source：当前 worktree `task/2026-07-04-comment-create-page-foundation`。
- 实施计划：`docs/plan/archive/impl-plan/2026-07-04-comment-create-page-foundation-implementation-plan.md`。
- 风险分类：公开 HTTP contract、事件 payload、transaction / outbox 和 migration，属于高风险 backend 切片。

## Gate Verdict

`pass`

## Findings

- Blocker：未发现。
- Major：未发现。
- Minor：无。
- Note：真实 PostgreSQL `up` / `down 1` 仍需在提供 `ZHICORE_COMMENT_POSTGRES_DSN` 后补跑。

## Material Findings

无。首轮 blocking 已在复审前修复并关闭。

## 独立 Review 过程

独立 reviewer 首轮结论：未通过，发现 2 个 blocking。

1. `comment.created` outbox payload 与 `libs/contracts/events/comment/comment-events.md` 不兼容。
2. 回复父评论 / 根评论校验存在事务外短路路径，不符合本地树结构校验必须在事务内闭合的要求。

复审结论：通过。原 2 个 blocking 均已关闭，未发现新的 blocker / major finding。

## 修复结果

- `comment.created` payload 已改为事件 contract 字段：内部 `commentId`、Content `publicId` / `internalId`、`rootId` / `parentId`、`hasImages` / `hasVoice` 和 `createdAt`。
- outbox `AggregateID` 改为内部 comment id 字符串。
- 回复创建的事务外读取改为非权威 `FindReplyGuardPreview`，只用于获取 `parentAuthorId` 做 User relation guard；父评论存在性、状态和树结构仍由事务内 `FindReplyTarget` 返回。
- 增加空 `postId` application 边界校验。
- `NewReplyDraft` 增加 parent 为顶级评论但 root 不一致的防御。

## 验证

- `cd services/zhicore-comment && go test ./...`：通过。
- `bash scripts/check-structure.sh`：通过。
- `python3 tests/architecture/check_boundaries.py --root .`：通过。
- `make test-size`：通过。
- `make check`：通过。
- `git diff --check`：通过。
- `rg '^- \[ \]' docs/plan/archive/impl-plan/2026-07-04-comment-create-page-foundation-implementation-plan.md`：无未完成 checklist。

## Required Re-Validation

- 如后续接入 PostgreSQL repository 或 runtime wiring，必须补跑真实数据库 migration `up` / `down 1`。
- 如继续修改本切片代码，至少重跑 `cd services/zhicore-comment && go test ./...`；触达 contract、migration 或分层边界时重跑 `make check`。

## 残余风险

- 当前 migration 只有字符串 contract test，尚未在真实 PostgreSQL 上执行 `up` / `down 1`；需要可用 `ZHICORE_COMMENT_POSTGRES_DSN` 后补跑。
- 当前切片只实现 application 和 HTTP contract，不包含 PostgreSQL repository、runtime wiring 或真实下游 client adapter。

## 技术债状态

未新增需要登记到 `docs/todos/debt/` 的 touched-surface 技术债；PostgreSQL 实跑验证属于外部 DSN 缺口，已作为残余风险记录。
