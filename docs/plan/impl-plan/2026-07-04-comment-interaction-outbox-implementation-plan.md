# Comment 互动与 Outbox 实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 在 `zhicore-comment` 中实现评论详情、回复分页、删除、点赞、取消点赞、点赞状态、计数 delta、outbox worker 和 runtime 收口。

**架构：** 删除、点赞和 outbox 都属于 Comment 本地事实。点赞请求只写点赞事实、delta 和事件，不在请求事务内同步刷新最终 `like_count`；worker 负责批量聚合和 rank 更新。

**技术栈：** Go 1.26、Gin HTTP router、PostgreSQL、RabbitMQ outbox、context cancellation。

---

## 背景依据

- `docs/architecture/module/comment/service.md`
- `docs/architecture/module/comment/runtime-resilience.md`
- `docs/architecture/module/comment/data-events.md`
- `docs/contracts/events.md`
- `docs/architecture/go-service-design.md`

## 文件结构

- 新增：`services/zhicore-comment/api/http/endpoints/get-comment-detail.md`
- 新增：`services/zhicore-comment/api/http/endpoints/list-replies-page.md`
- 新增：`services/zhicore-comment/api/http/endpoints/delete-comment.md`
- 新增：`services/zhicore-comment/api/http/endpoints/like-comment.md`
- 新增：`services/zhicore-comment/api/http/endpoints/unlike-comment.md`
- 新增：`services/zhicore-comment/api/http/endpoints/get-like-status.md`
- 新增：`services/zhicore-comment/api/http/comment_interaction_handler_test.go`
- 新增：`services/zhicore-comment/internal/comment/application/delete_comment_test.go`
- 新增：`services/zhicore-comment/internal/comment/application/like_comment_test.go`
- 新增：`services/zhicore-comment/internal/comment/application/outbox_worker_test.go`
- 修改：`services/zhicore-comment/internal/comment/application/service.go`
- 新增或修改：`services/zhicore-comment/internal/comment/infrastructure/postgres/*.go`
- 新增：`services/zhicore-comment/internal/comment/runtime/module.go`
- 新增：`services/zhicore-comment/cmd/server/main.go`

## 任务 1：后续 endpoint contract

**测试立场：** R0 文档切片，但后续 handler 必须按 contract 测试。

- [x] **步骤 1：提取详情和回复分页 contract**

  固定 `GET /api/v1/posts/{postId}/comments/{commentId}` 和 replies page 的响应、错误码、排序和 viewer 语义。

- [x] **步骤 2：提取删除和点赞 contract**

  固定作者删除、Admin 删除、重复删除、点赞幂等、取消点赞幂等和点赞状态查询。

- [x] **步骤 3：更新服务级 HTTP README 索引**

  新 endpoint 状态先标记为“草案”。

## 任务 2：删除和点赞 application

**测试立场：** TDD - 删除子树、统计、delta 和事件一致性属于 R4。

- [x] **步骤 1：编写删除失败测试**

  覆盖普通用户只能删自己的评论、普通用户重复删除返回 404、Admin 重复删除幂等成功、删除子树只发一条 `comment.deleted`。

- [x] **步骤 2：实现删除 application**

  删除必须维护 `reply_count`、`comment_post_stats` 和 rank 可见性。

- [x] **步骤 3：编写点赞失败测试**

  覆盖点赞 / 取消点赞重复调用幂等成功，但不重复写 delta 和事件。

- [x] **步骤 4：实现点赞 application**

  点赞请求只写 `comment_likes`、`comment_counter_deltas` 和 `comment.liked/comment.unliked` outbox。

- [x] **步骤 5：运行 application 测试**

  运行：`cd services/zhicore-comment && go test ./internal/comment/application -run 'TestDeleteComment|TestLikeComment'`

  预期：通过。

## 任务 3：HTTP handler

**测试立场：** TDD - 公开错误码和幂等语义必须锁定。

- [x] **步骤 1：编写 handler 失败测试**

  覆盖详情、回复分页、删除、点赞、取消点赞和点赞状态。

  运行：`cd services/zhicore-comment && go test ./api/http -run 'TestCommentDetail|TestReplies|TestDelete|TestLike'`

  预期：失败。

- [x] **步骤 2：实现 handler**

  非 Admin 公开 API 对不存在和已删除统一返回 404；Admin 重复删除返回当前删除元数据。

- [x] **步骤 3：更新 endpoint 状态**

  被 handler contract test 覆盖的 endpoint 标记为“已验证”。

## 任务 4：Outbox worker 和 runtime

**测试立场：** TDD - claim、重试、停机和幂等属于 R4。

- [x] **步骤 1：编写 outbox claim 失败测试**

  覆盖 `PENDING -> CLAIMING -> SENT/DEAD`、stale claim 重领、多实例不重复 claim、publish 失败退避和条件更新。

- [x] **步骤 2：实现 worker**

  使用 `FOR UPDATE SKIP LOCKED` claim 模式；publish RabbitMQ 不在持有 DB 行锁的事务里执行。

- [x] **步骤 3：编写 shutdown 测试**

  worker 必须响应 context cancellation，不在停止过程中继续 claim 新任务。

- [x] **步骤 4：补 runtime module 和 `cmd/server`**

  `runtime.Build` 返回 HTTP handler、workers 和健康检查；`cmd/server` 不放业务逻辑。

- [x] **步骤 5：运行收口测试**

  运行：`cd services/zhicore-comment && go test ./api/http ./internal/comment/...`

  预期：通过。

## 架构适配评估

- 点赞计数通过 delta worker 最终一致更新，不阻塞请求事务。
- outbox 是 Comment 本地事务事实，RabbitMQ 故障不回滚业务请求。
- runtime 只做装配，不承载业务规则。
