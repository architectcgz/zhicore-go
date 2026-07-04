# Comment 创建与分页基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 在 `zhicore-comment` 中实现创建根评论 / 回复、顶级评论传统分页、核心 schema 和首批 HTTP endpoints。

**架构：** Comment 拥有评论树、评论统计、rank 读模型和评论事件；Content/User/File 只通过 ports 提供写入前置 guard。外部 guard 在本地事务外执行，本地父评论和树结构校验在 Comment 事务内闭合。

**技术栈：** Go 1.22、标准库 `net/http`、PostgreSQL、`libs/kit/httpapi`。

---

## 背景依据

- `docs/architecture/module/comment/README.md`
- `docs/architecture/module/comment/service.md`
- `docs/architecture/module/comment/domain.md`
- `docs/architecture/module/comment/data-events.md`
- `docs/architecture/module/comment/comment-id.md`
- `services/zhicore-comment/api/http/README.md`

## 文件结构

- 新增：`services/zhicore-comment/internal/comment/domain/comment.go`
- 新增：`services/zhicore-comment/internal/comment/domain/media.go`
- 新增：`services/zhicore-comment/internal/comment/domain/errors.go`
- 新增：`services/zhicore-comment/internal/comment/ports/*.go`
- 新增：`services/zhicore-comment/internal/comment/application/service.go`
- 新增：`services/zhicore-comment/internal/comment/application/create_comment_test.go`
- 新增：`services/zhicore-comment/internal/comment/application/list_comments_test.go`
- 新增：`services/zhicore-comment/api/http/handler.go`
- 新增：`services/zhicore-comment/api/http/create_comment_handler_test.go`
- 新增：`services/zhicore-comment/api/http/list_comments_page_handler_test.go`
- 修改：`services/zhicore-comment/api/http/README.md`
- 修改：`services/zhicore-comment/api/http/endpoints/create-comment.md`
- 修改：`services/zhicore-comment/api/http/endpoints/list-comments-page.md`
- 新增：`services/zhicore-comment/migrations/<timestamp>_create_comment_core_tables.up.sql`
- 新增：`services/zhicore-comment/migrations/<timestamp>_create_comment_core_tables.down.sql`

## 任务 1：Comment domain、ports 和创建用例

**测试立场：** TDD - 评论树、媒体规则、统计和 outbox 是 R4。

- [ ] **步骤 1：编写创建评论失败测试**

  覆盖根评论、回复、空内容、文本过长、图片超过 9 张、图片语音互斥、父评论已删除、Content/User/File guard 失败不写入、成功写 `comment.created` outbox。

  运行：`cd services/zhicore-comment && go test ./internal/comment/application -run TestCreateComment`

  预期：失败。

- [ ] **步骤 2：实现 ID codec 和 domain**

  按 `comment-id.md` 固定内部 `BIGINT IDENTITY` 与外部 `commentId` 编码 / 解码边界。

- [ ] **步骤 3：定义 ports**

  至少定义 `ContentPostClient`、`UserProfileClient`、`UserRelationClient`、`FileReferenceClient`、`RateLimiter`、`CommentIDCodec`、`CommentCommandRepository`、`CommentStatsRepository`、`CommentPostStatsRepository`、`OutboxPublisher`、`TransactionRunner`、`Clock`。

- [ ] **步骤 4：实现创建 application**

  关键注释说明外部 guard 失败时 fail closed，不写本地评论事实、不消耗本地 ID 之外的业务副作用。

- [ ] **步骤 5：运行创建用例测试**

  运行：`cd services/zhicore-comment && go test ./internal/comment/application -run TestCreateComment`

  预期：通过。

## 任务 2：顶级评论分页用例

**测试立场：** TDD - 分页、排序、计数和 viewer.liked 是公开查询 contract。

- [ ] **步骤 1：编写分页 application 失败测试**

  覆盖 `RECOMMENDED`、`HOT`、`TIME` 排序，空列表，作者摘要降级，登录用户 `viewer.liked`。

- [ ] **步骤 2：实现查询 ports 和 application**

  `RECOMMENDED` 走 `comment_recommended_rank`，`HOT` 走 `comment_hot_rank`，`TIME` 走 `comments.id DESC`。

- [ ] **步骤 3：运行分页 application 测试**

  运行：`cd services/zhicore-comment && go test ./internal/comment/application -run TestListTopLevelComments`

  预期：通过。

## 任务 3：HTTP endpoints 和 migration

**测试立场：** TDD - Go-first HTTP contract 必须由 handler test 锁定。

- [ ] **步骤 1：编写 handler 失败测试**

  覆盖 `POST /api/v1/posts/{postId}/comments` 和 `GET /api/v1/posts/{postId}/comments/page` 的成功、鉴权、参数、错误码和 envelope。

  运行：`cd services/zhicore-comment && go test ./api/http`

  预期：失败。

- [ ] **步骤 2：实现 handler 和错误映射**

  使用 `WriteErrorCode`，确保 `5001` 等业务错误码不会被写成 HTTP status。

- [ ] **步骤 3：编写核心 migration**

  至少包含 `comments`、`comment_stats`、`comment_post_stats`、`comment_likes`、`comment_counter_deltas`、`comment_hot_rank`、`comment_recommended_rank`、`outbox_events`。

- [ ] **步骤 4：更新 contract 状态**

  `create-comment.md` 和 `list-comments-page.md` 被测试覆盖后标记为“已验证”。

- [ ] **步骤 5：运行 Comment 首批测试**

  运行：`cd services/zhicore-comment && go test ./api/http ./internal/comment/...`

  预期：通过。

## 架构适配评估

- Comment 不导入 Content/User/File 的内部模型。
- 写路径依赖不可确认时 fail closed。
- 查询增强失败可以降级，但不能伪造 User `publicId`。

