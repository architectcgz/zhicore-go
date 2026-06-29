# Content Context

Content 上下文拥有文章主数据、正文、草稿、发布生命周期、标签、互动统计和作者快照。本文只记录领域语言；实现策略、存储和流程见同目录架构文档与 ADR。

## Language

**文章（Post）**:
Content 拥有的内容实体，包含发布生命周期、元数据、正文引用、作者快照和标签关系。
_Avoid_: content item, document（除非指共建文档）

**个人文章（Personal Article）**:
单作者普通文章，不保留完整历史版本，只保留当前线上正文和当前编辑草稿。
_Avoid_: wiki, collaborative doc

**共建文档（Collaborative Document）**:
多人共同维护的内容类型，预留完整 revision history、diff 和回滚能力。
_Avoid_: personal article with a flag

**正文（Body）**:
文章的主体内容，由结构化 blocks 表达，存放在 MongoDB body 文档中。
_Avoid_: raw HTML, blob

**正文引用（Body ID）**:
PostgreSQL `posts` 行保存的 UUID，用来指向 MongoDB `post_bodies._id`。
_Avoid_: version（普通个人文章中不要把它叫版本）

**线上正文（Published Body）**:
`posts.published_body_id` 指向的正文，是公开详情页唯一可读正文。
_Avoid_: latest Mongo published role

**草稿正文（Draft Body）**:
`posts.draft_body_id` 指向的编辑中正文，只有作者编辑器和草稿接口可读。
_Avoid_: unpublished published body

**正文快照（Snapshot Body）**:
由草稿正文复制出的不可变正文，用于发布时让 PostgreSQL 原子切换 `published_body_id`。
_Avoid_: historical version（普通文章不承诺长期保存）

**正文块（Block）**:
正文里的结构化内容单元，例如段落、标题、表格、公式、图片、外部嵌入或附件组。
_Avoid_: HTML tag

**作者快照（Owner Snapshot）**:
Content 本地保存的作者昵称、头像和资料版本快照，用于列表和详情展示。
_Avoid_: User profile source

**互动（Engagement）**:
文章的点赞、收藏、统计和当前用户视角状态。统计是文章事实，当前用户视角是查询结果。
_Avoid_: reaction（除非未来扩展多表情反馈）

**当前用户视角（Viewer Engagement）**:
登录用户对当前文章是否已点赞、是否已收藏的查询结果。
_Avoid_: post state, global engagement state

**互动状态未知（Unknown Engagement Status）**:
Redis 不可用且受控 DB fallback 无法确认时返回的查询降级状态，不是领域事实，不能当成未点赞或未收藏。
_Avoid_: false, unliked, default state

**正文清理任务（Body Cleanup Task）**:
删除未被 PostgreSQL `published_body_id` 或 `draft_body_id` 引用的 MongoDB body 的资源回收任务。
_Avoid_: repair task

**正文修复任务（Body Repair Task）**:
记录 `published_body_id` 指向的正文缺失或 hash 不一致等数据一致性事故的修复任务。
_Avoid_: cleanup task

## Relationships

- 一篇 **文章** 最多有一个当前 **线上正文**。
- 一篇 **文章** 最多有一个当前 **草稿正文**。
- 一个 **正文引用** 指向一个 MongoDB body 文档。
- **线上正文** 由 PostgreSQL `published_body_id` 决定，不由 MongoDB 字段决定。
- **草稿正文** 发布时复制成 **正文快照**，然后由 PostgreSQL 事务切换为新的 **线上正文**。
- **个人文章** 不保留完整正文历史；**共建文档** 才预留 revision history。
- **正文清理任务** 只删除 PostgreSQL 未引用的 body；**正文修复任务** 处理 PostgreSQL 指向的 body 不可读或不一致。
- **作者快照** 来自 User 事实，但不是 User 资料事实源。
- **互动状态未知** 只存在于查询响应，不会写入点赞、收藏、统计、outbox 或 Redis 事实缓存。

## Example Dialogue

> **Dev:** “发布成功后 MongoDB 里哪个 body 是真实线上正文？”
> **Domain expert:** “只看 **文章** 的 `published_body_id`。它指向的 **正文引用** 就是 **线上正文**，MongoDB 里的其他 body 只是草稿、orphan 或待清理快照。”

> **Dev:** “普通个人文章的 `body_id` 算版本号吗？”
> **Domain expert:** “不算。它只是内部 **正文引用**。完整历史版本只属于 **共建文档**。”

## Flagged Ambiguities

- “版本”曾被用来描述普通文章的正文 UUID；已统一为 **正文引用（Body ID）**，避免误解成产品历史版本。
- “补偿”曾被用来描述发布后补 Mongo 正文；已改为 **正文清理任务** 和 **正文修复任务** 两类，发布正文不走“PG 已发布后补正文”的正常流程。
- “未知互动状态”不能被产品、前端或后端实现解释成未点赞 / 未收藏；它只表示当前请求无法确认。
