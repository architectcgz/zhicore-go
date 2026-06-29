# Content 数据、事件和契约设计

本文记录 `zhicore-content` 的 API 保留范围、数据归属、主写流程、事件、跨服务依赖、发布校验、错误契约和后续链接预览设计。

## API 设计范围

Content API 采用 Go-first 设计，Java 只作为业务能力参考，不作为 path、字段或响应兼容约束。字段级 HTTP schema 见 `services/zhicore-content/api/http/`，当前状态为草案，尚未由 Go handler / contract test 验证。

必须覆盖以下 API 族：

- `/api/v1/posts`：公开文章列表、作者文章过滤、创建草稿、文章详情、published body、批量摘要。
- `/api/v1/me/posts`、`/api/v1/me/drafts`：作者工作台列表。
- `/api/v1/posts/{postId}/draft/*`：草稿元数据、正文 blocks、草稿读取和删除。
- `/api/v1/posts/{postId}/publish`、`unpublish`、`schedule`、`restore`：发布生命周期。
- `/api/v1/posts/{postId}/like`、`favorite`、`engagement`：点赞、收藏、互动状态和计数。
- `/api/v1/posts/{postId}/reader-*`：阅读 presence session、leave、presence 查询。
- `/api/v1/tags`：标签详情、列表、搜索、热门和标签文章。
- `/api/v1/admin/content/posts`：管理端文章查询和删除。
- `/api/v1/admin/content/outbox-events`：outbox dead/failed 查询和 retry。

User 不暴露用户文章 facade；用户主页需要文章列表时直接调用 Content 作者过滤接口，例如 `GET /api/v1/posts?authorId={authorId}&limit=20`。

## 数据归属

PostgreSQL：

- `posts`
- `content_body_cleanup_tasks`
- `content_body_repair_tasks`
- `post_stats`
- `post_likes`
- `post_favorites`
- `categories`
- `tags`
- `post_tags`
- `tag_stats`
- `scheduled_publish_event`
- `outbox_event`
- `domain_event_task`
- `outbox_retry_audit`
- `consumed_events`

MongoDB：

- `post_bodies`：文章正文 blocks、`plain_text`、`schemaVersion`、`content_hash`、`size_bytes`

Redis：

- 文章详情缓存
- 点赞 / 收藏状态和计数缓存
- 阅读 presence 短生命周期状态

作者快照字段如 `owner_name`、`owner_avatar_id`、`owner_profile_version` 保留为列，不默认改成 JSON。原因是这些字段需要参与补偿、查询、索引和精确更新。

## 主写流程

文章创建、编辑、发布、删除、恢复由 application use case 拥有事务边界：

```text
api/http -> application command -> postgres repository
                         -> mongo body store（先写 draft/snapshot，再由 PG 切指针）
                         -> domain_event_task
                         -> outbox_event
                         -> content_body_cleanup_tasks / repair_tasks
```

公开列表和详情元数据只读 PostgreSQL `published_*` 字段；草稿列表只读 PostgreSQL `draft_*` 字段，不为列表批量读取 MongoDB 正文。

MongoDB 只在这些路径读取正文全文：

- 文章详情读取当前 published body。
- 编辑器读取当前 draft 或从 published body 派生编辑内容。
- 发布前审核、字数统计、媒体引用提取。
- Search consumer 拉取当前 published body 做全文索引。

发布事件不携带正文全文，只携带轻量字段：`postId`、作者、标题、摘要、封面、`publishedBodyId`、`publishedBodyHash` 和发布时间。选择原因是正文可能很大，事件会被多个服务持久化和重试，带全文会放大存储、隐私和治理成本。

## 事件

Content 生产：

- `content.post.published`
- `content.post.updated`
- `content.post.deleted`
- `content.post.visibility_changed`
- `content.post.tags.updated`
- `content.post.liked`
- `content.post.unliked`
- `content.post.favorited`
- `content.post.unfavorited`
- `content.post.viewed`

字段级事件 contract 见 `libs/contracts/events/content/README.md` 和 `libs/contracts/events/content/post-events.md`。本文只保留服务设计和事件方向。

`content.post.visibility_changed` 用于表达不一定产生正文或互动变化、但会改变公开可见性的生命周期事实，例如撤回、恢复、管理端下架、隐藏或重新公开。当前 Go API 已登记 `publish`、`unpublish`、`delete`、`restore`；未来若补管理端隐藏 / 下架能力，仍应复用该可见性事件，而不是让 Search / Ranking 在查询时临时回源 Content 判断。事件 payload 至少包含 `publicPostId`、可选内部 `postId`、`oldVisibility`、`newVisibility`、`publicVisible`、`reason`、`occurredAt`；事件 envelope 应携带 `aggregateVersion`，用于 consumer 处理乱序或迟到事件；`eventId` 仍用于 consumer 幂等。

Content 消费：

- `user.profile.updated`：刷新作者快照。
- Comment 事件可以更新评论计数，但评论事实仍归 Comment。

关键跨服务事件统一走 producer outbox + RabbitMQ topic exchange。

Search consumer 收到 `content.post.published` / `content.post.updated` 后通过 Content internal API 拉取当前 published body。如果 Content 返回 `CONTENT_BODY_UNAVAILABLE` 或 5xx，Search 应重试或进入 DLQ，不应 ack 成功。Ranking / Notification 只消费轻量字段，不读取正文。

## 跨服务依赖

- User：创建文章和刷新作者快照时读取用户资料摘要。
- Upload：文章图片、封面和正文资源只保存文件引用。
- Comment：文章详情可聚合评论计数，但不直接读评论库。
- Search：消费 Content 事件后拉取 published body 做索引。
- Ranking / Notification：消费 Content 事件，不读取正文。

## 发布校验

普通个人文章发布规则：

- `title` 必填；草稿阶段可以为空，发布时 `trim(title)` 必须非空。
- `summary` 默认非必填，优先用户手填。
- AI summary 只是可配置辅助能力，不能在保存或发布时隐式覆盖用户摘要；用户明确触发并接受后才写入 summary。
- 正文 blocks 提取 `plain_text` 后，去掉空白和格式标记，有效 rune 数至少 10。
- 媒体块不能替代文字正文要求。
- 封面非必填；封面上传由前端调用 Upload 完成，保存草稿时绑定 `draft_cover_file_id`；发布时只校验当前引用未失效，不在发布链路执行上传。
- internal media 使用 Upload `file_id`，保存和发布需要校验文件 facts。
- external media / 外部链接只做 URL、scheme、provider 白名单和安全格式校验，不因为第三方站点短暂不可访问而阻止发布。

选择原因：

- 标题、摘要、封面用于列表、搜索事件、通知和分享卡片，必须由 PostgreSQL 充当真相源。
- 外部媒体可用性不归 Content 所有，不能让第三方站点波动决定文章是否能发布。
- AI summary 是产品增强，不应成为默认保存/发布副作用。

## 错误契约

公开错误码登记在 `docs/contracts/error-codes.md`。Content 正文相关错误包括：

- `BODY_SCHEMA_INVALID`
- `BLOCK_TYPE_NOT_ENABLED`
- `BODY_TOO_LARGE`
- `BODY_TEXT_TOO_SHORT`
- `DRAFT_CONFLICT`
- `CONTENT_BODY_UNAVAILABLE`
- `CONTENT_BODY_INCONSISTENT`
- `EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED`
- `MEDIA_REF_INVALID`
- `VALIDATION_ERROR_LIMIT_EXCEEDED`
- `COVER_UNAVAILABLE`
- `BODY_SCHEMA_UNSUPPORTED`

错误响应必须使用稳定英文机器码，不把中文文案写死在协议里。字段级 / block 级校验错误放在结构化 details 中，前端按 `code` 或可选 `messageKey` 做 i18n：

```json
{
  "code": 1001,
  "data": {
    "details": [
      {
        "path": "blocks[3].children[1].latex",
        "code": "MATH_LATEX_TOO_LONG",
        "messageKey": "content.body.math_latex_too_long"
      }
    ]
  }
}
```

一次校验尽量返回多个路径级错误，但最多返回 20 个。超过上限返回 `VALIDATION_ERROR_LIMIT_EXCEEDED`。选择批量返回的原因是编辑器可以一次性标出多个 block 问题；设置上限是为了避免恶意请求制造巨大错误响应。

## 链接预览后续项

普通文章第一阶段不实现链接预览；正文中的外部链接只按安全链接渲染，不在保存草稿或发布时抓取外链。

后续如果实现链接预览：

- 必须由后端异步生成并缓存 preview 数据。
- 前端只消费后端返回的标题、描述、站点名和封面等结果。
- 后端抓取用户提交 URL 时必须使用 SSRF-safe fetcher。
- SSRF-safe fetcher 必须限制 scheme、内网地址、重定向、超时、响应大小和请求头。
- 链接预览生成失败不得阻塞保存草稿或发布流程。

选择后置的原因是链接预览会引入 SSRF、安全抓取、缓存、重试、图片代理和治理复杂度；第一阶段普通安全链接已能满足正文基础展示。

对应待办：[Content 后续实现链接预览](../../../todo/2026-06-26-1448-content.md)。

## 实现风险

- 既有 schema 来源存在漂移，Go migration 必须以服务归属重新整理，不原样复制全量初始化 SQL。
- 点赞 / 收藏链路必须保留 outbox 同事务语义，否则 Ranking / Notification 会丢事件。
- `domain_event_task` 是服务内投影，不应和跨服务 outbox 混用。
- Presence 只能作为附加能力，不能影响文章正文主链路。
- 正文 copy-on-write、cleanup、repair 等基础设施状态不能泄漏到领域模型；`Post` 聚合只表达业务状态和可见性规则。
- `PostStats` 是独立聚合根，不属于 `Post` 聚合，确保点赞 / 收藏不锁文章聚合。
