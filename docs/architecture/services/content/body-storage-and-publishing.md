# Content 正文存储与发布流程

本文记录 Content 正文在 PostgreSQL 与 MongoDB 之间的存储、草稿、发布、清理、修复和 blocks schema 设计。

## 核心结论

```text
posts.published_body_id -> Mongo post_bodies._id
posts.draft_body_id     -> Mongo post_bodies._id
```

PostgreSQL 是可见性真相源，MongoDB 只保存正文 body。只有 `posts.published_body_id` 指向的 body 才是线上正文。

选择原因：

- 如果让 MongoDB 用 `role=published` 决定线上正文，多个 snapshot 会产生“哪个才是真实线上版本”的歧义。
- PostgreSQL 事务可以同时切换 `published_*` 元数据、body 指针、outbox 和 cleanup task，适合作为上线开关。
- 个人文章不把正文 UUID 暴露成产品版本号，也不长期保留历史版本；UUID 只是内部正文引用。

## 为什么不用 PG + Mongo 分布式事务

不使用 2PC / XA。原因：

- 需要额外协调器和恢复流程。
- 会引入 prepare / commit 阶段的悬挂事务、锁占用和运维复杂度。
- 本业务需要的是用户可见原子性，不是跨库物理强事务。

Content 的用户可见原子性由“MongoDB 先写 snapshot，PostgreSQL 事务切换指针”提供：

```text
发布成功：
  新 title / summary / cover / body 一起对外可见

发布失败：
  旧 published_* 完全不变
  草稿继续保留
```

## 为什么不采用 PG 先发布再补 Mongo

不采用：

```text
PG posts.status = published 成功
MongoDB 正文写入失败
后台补偿正文
```

原因是正文是文章详情页核心数据。如果 `published` 但正文不可读，用户点击进去会看到“发布成功但无法阅读”的异常状态，这不应是正常业务语义。补偿适合搜索索引、缓存、通知等非核心投影，不适合文章正文。

## 保存草稿

草稿保存采用 copy-on-write，不原地覆盖 MongoDB draft：

```text
1. 前端带 base_post_version / base_draft_hash 保存草稿。
2. application 校验 blocks schema，生成 canonical blocks、plain_text、content_hash。
3. MongoDB 写入新的 draft body UUID。
4. PostgreSQL transaction 校验 post_version / draft_body_hash：
   - draft_body_id = new_draft_body_id
   - draft_body_hash = new_hash
   - draft_size_bytes = size
   - draft_plain_text_length = length
   - post_version +1
   - content_body_cleanup_tasks(delete old draft)
5. commit 后 best-effort 删除旧 draft；定时任务最终清理。
```

选择 copy-on-write 的原因：

- 如果原地覆盖 MongoDB draft 后 PostgreSQL 更新失败，会出现 PG 仍记录旧 hash、MongoDB 却已经被新内容覆盖的不一致。
- copy-on-write 让失败方向可恢复：PG 未切指针时旧草稿仍有效，新 draft 只是 orphan。
- 保存频率由前端 debounce 和后端节流控制；`content_hash` 未变化时 no-op，避免制造大量 body。

## 发布文章

发布流程：

```text
1. Handler 解析 PublishPost 请求，携带 base_post_version、draft_body_id、draft_body_hash。
2. application 读取 posts 当前 draft_* 和 post_version，校验作者、状态和并发版本。
3. 读取 MongoDB draft body，校验 body hash、blocks schema、plain_text 长度、媒体引用和基础审核。
4. MongoDB 写入新的 snapshot body UUID。
5. PostgreSQL transaction 再次校验 post_version / draft_body_id / draft_body_hash：
   - published_title = draft_title
   - published_summary = draft_summary
   - published_cover_file_id = draft_cover_file_id
   - published_body_id = new_snapshot_body_id
   - published_body_hash = draft_body_hash
   - published_plain_text_length = draft_plain_text_length
   - draft_* 清空
   - post_version +1
   - outbox_event(content.post.published 或 content.post.updated)
   - content_body_cleanup_tasks(delete old draft / old snapshot)
6. PostgreSQL commit 成功后返回发布成功。
```

失败语义：

- MongoDB 写 snapshot 失败：PostgreSQL 不变，发布失败，草稿保留。
- MongoDB 写 snapshot 成功但 PostgreSQL transaction 失败：线上 `published_*` 不变，新 snapshot 没被 PG 引用，作为 orphan 清理。
- PostgreSQL commit 成功：发布成功，详情页只按 `published_body_id` 读取 MongoDB body。
- published body 读取 MongoDB miss：返回 `CONTENT_BODY_UNAVAILABLE`，创建 repair task 并告警；普通个人文章没有旧正文或 draft 降级可读。

## published / draft 元数据分离

`posts` 行同时保存 published 和 draft 两组元数据：

```text
post_version
status
published_title / published_summary / published_cover_file_id
published_body_id / published_body_hash / published_plain_text_length
draft_title / draft_summary / draft_cover_file_id
draft_body_id / draft_body_hash / draft_size_bytes / draft_plain_text_length
```

选择原因：

- 已发布文章再次编辑时，草稿标题、摘要、封面和正文不能污染线上列表或详情。
- 发布失败时线上 `published_*` 必须完全保持旧值。
- 列表页只读 PostgreSQL 快照字段，不为列表批量读取 MongoDB 正文。

## 清理与修复任务

### `content_body_cleanup_tasks`

用途：

- 删除 old draft。
- 删除 old snapshot。
- 删除 orphan draft / snapshot。

清理必须按 `body_id` 精确删除，不能按 `post_id + role=draft` 删除。删除前必须确认：

```sql
NOT EXISTS (
  SELECT 1 FROM posts
  WHERE published_body_id = $body_id
     OR draft_body_id = $body_id
)
```

这样可以避免用户发布后立刻再次编辑时，旧 draft 清理任务误删新 draft。

**清理 SLA 和告警：**

- cleanup worker 调度间隔：≤ 60s（配置化，首批默认值）。
- 单批次处理量：每轮最多 100 条（防止大批量写 MongoDB 影响正常读写）。
- 告警触发条件：
  - 待清理孤儿数量超过 500 条。
  - 最老待清理任务年龄超过 1 小时。
  - 连续 3 次 worker 失败。
- cleanup 失败不影响业务事务，但必须有结构化错误日志和 `content_cleanup_failed_total` metric。
- cleanup 是幂等的：重复删除不存在的 body 不是错误（MongoDB 删除操作返回 0 删除计数时正常 ack）。

### `content_body_repair_tasks`

用途：

- `published_body_missing`
- `draft_body_missing`
- `body_hash_mismatch`
- `mongo_read_error_after_pg_published`

repair task 是数据一致性事故入口，需要告警、人工介入、备份恢复或下架；它不是普通资源回收。

## 正文 blocks schema

MongoDB body 保存结构：

```json
{
  "schemaVersion": 1,
  "format": "blocks",
  "blocks": [],
  "plainText": "...",
  "contentHash": "sha256:...",
  "sizeBytes": 1234
}
```

使用结构化 blocks 的原因：

- 媒体引用、字数统计、审核、搜索抽取、AI summary 和 schema migration 都需要可控结构。
- raw HTML 会扩大 XSS、样式污染和 iframe/script 注入风险。
- 后端可以统一 canonicalize blocks 并计算 `content_hash`。

第一阶段可发布 block：

- `paragraph`
- `heading`
- `quote`
- `list`
- `code_block`
- `table`：简单二维表，不支持 `rowspan` / `colspan`
- `collapsible`：可展开/收起内容块，最大嵌套深度 2
- `math`：LaTeX 字符串，后端不执行公式
- `image`
- `external_embed`：只允许 provider 白名单，不保存任意 iframe / HTML
- `attachment_gallery`：只允许 File `file_id`

第一阶段仅预留、不开放发布：

- `mention`
- `poll`
- `custom_widget`

如果 blocks 中出现未启用类型，返回 `BLOCK_TYPE_NOT_ENABLED`。

## marks 与外链

行内 marks 第一阶段只支持：

- `bold`
- `italic`
- `underline`
- `strike`
- `inline_code`
- `link`

`link.href` 只允许安全的 `http` / `https`，禁止 `javascript:`、`data:`、`file:` 等 scheme。

外部媒体 / 外部链接只做安全格式、scheme、provider 白名单校验，不因为第三方站点短暂不可访问而阻止发布。外部媒体可用性不归 Content 所有。

## BodyParserRegistry

正文解析按 `schemaVersion` 使用 Strategy + Registry：

```text
BodyParserRegistry
  -> V1BodyParser
  -> V2BodyParser
```

application 不直接解析 blocks，只依赖 parser 输出的 `NormalizedBody`：

- `plainText`
- `mediaRefs`
- `externalLinks`
- `canonicalJSON`
- block 统计

`content_hash = SHA-256(canonical blocks)`。它只作为一致性指纹和幂等辅助，不作为安全证明。

## schema migration

schema 升级使用读兼容 + 分批 copy-on-write migration：

- 新写入使用最新 writable schema。
- 读路径在支持窗口内兼容旧 schema。
- 迁移当前 published / draft body 时先写新 body，再用 PostgreSQL 事务切换 body 指针，旧 body 进入 cleanup task。
- 迁移任务按批次、并发和 MongoDB 空间水位控制；冷数据可以在支持窗口内保留旧 schema，避免全库瞬时空间翻倍。

选择 copy-on-write migration 的原因：原地修改 published body 一旦写坏，会直接损坏线上正文；copy-on-write 写坏时 PG 不切指针，线上仍读旧正文。
