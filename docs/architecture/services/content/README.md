# Content 服务设计

本目录记录 `zhicore-content` 的 Go 目标设计。本文是入口索引和关键结论摘要；详细事实按专题拆分到同目录文件，避免把领域模型、正文存储、应用流程、契约和实现切片都堆在一个 README 里。

## 事实来源

- Java `zhicore-content` controller：只作为业务能力参考，用于确认 Post command/query、like/favorite、tag、admin、outbox、reader presence 等能力存在；不作为 Go API path、字段或响应兼容约束。
- Java `content-service-design.md`、`content-visibility-and-projection-evolution.md`、`post-reading-presence.md`。
- Java `zhicore-content/src/main/resources/db/schema.sql`。
- `zhicore-client` 和 `zhicore-integration` 中 post 事件与 DTO。

## 阅读顺序

0. [CONTEXT.md](CONTEXT.md)：Content 领域语言和术语边界。
1. [domain-model.md](domain-model.md)：限界上下文、聚合、值对象、领域事件、领域服务和工厂。
2. [body-storage-and-publishing.md](body-storage-and-publishing.md)：PostgreSQL + MongoDB 正文指针、草稿、发布原子切换、blocks schema、cleanup / repair。
3. [application-and-ports.md](application-and-ports.md)：application use case、ports、包落点、事务边界和实现切片。
4. [data-events-contracts.md](data-events-contracts.md)：API 保留范围、数据归属、事件、跨服务依赖、发布校验、错误契约和链接预览后续项。
5. [engagement-design.md](engagement-design.md)：点赞、收藏、互动统计、当前用户状态、Redis 故障降级和产品展示语义。
6. [page-design.md](page-design.md)：公开浏览页面、文章详情页、列表页互动摘要和前端加载编排。
7. [rate-limiting.md](rate-limiting.md)：公开读、作者写路径、互动、presence、管理端和内部调用的限流矩阵、Redis 故障原则和观测要求。
8. [runtime-resilience.md](runtime-resilience.md)：Content 下游 provider / operation 的 timeout、retry、circuit breaker、max-in-flight 和降级策略矩阵。

相关 ADR：

- [ADR 0001: Content uses PostgreSQL body pointers for publish atomicity](adr/0001-body-pointer-publish-atomicity.md)
- [ADR 0002: Content stores body as structured blocks and rejects raw HTML](adr/0002-body-blocks-no-raw-html.md)
- [ADR 0003: Content defers link preview to a backend asynchronous feature](adr/0003-link-preview-deferred.md)

复盘日志：

- [2026-06-26 Content 设计压测问答重建日志](decision-log/2026-06-26-content-design-grill.md)

辅助图：

- [service-detail.drawio](service-detail.drawio)：Content 保存草稿与发布原子切换流程图源。
- [service-detail.png](service-detail.png)：流程图导出图。
- [service-design.content.png](service-design.content.png)：服务级设计图。

## 职责边界

`zhicore-content` 拥有文章主数据、文章发布生命周期、标签、分类、话题引用、文章互动写模型、文章统计、作者快照和内容服务内部投影。

Content 不拥有用户资料事实、评论树、搜索索引、热榜分数、通知收件箱或 Upload 文件对象。

## 关键设计结论

- **Content 是独立限界上下文**：统一语言以文章、草稿、正文、发布、定时发布、删除、恢复、标签、分类、话题引用、点赞、收藏、统计、作者快照和内部投影为核心。
- **PostgreSQL 是可见性真相源**：`posts.published_body_id` / `posts.draft_body_id` 指向 MongoDB `post_bodies._id`；MongoDB 不通过 `role=published` 决定线上正文。
- **发布是用户可见原子操作**：新标题、摘要、封面和正文一起上线；失败时线上 `published_*` 完全不变。
- **不做 PG + Mongo 分布式事务**：不使用 2PC / XA。选择原因是协调器、悬挂事务、锁占用和恢复复杂度过高；本阶段只需要 PG 事务作为上线开关。
- **不采用“PG 已 published 后 Mongo 正文补偿”**：正文是详情页核心数据，不能允许发布成功后用户点进去读不到正文。
- **正文写入 copy-on-write**：保存草稿写新 draft body，发布写新 snapshot body，然后由 PG 事务切换指针；旧 body 进入 cleanup task。
- **普通个人文章不做长期版本库**：正文 UUID 只是内部引用，不是产品版本号；旧 draft / old snapshot 按 body_id 清理。
- **正文使用结构化 blocks，不允许 raw HTML**：blocks 便于媒体引用、字数统计、审核、搜索抽取、AI summary 和 schema migration；raw HTML 会扩大 XSS 和样式污染风险。
- **链接预览第一阶段不做**：后续如果做，必须由后端异步生成并使用 SSRF-safe fetcher。
- **Engagement viewer 状态使用三值语义**：`liked/favorited=true` 表示确认已互动，`false` 表示确认未互动，`null + degraded=true` 表示当前无法确认；前端不能把 unknown 当成未点赞或未收藏。
- **浏览页面按主资源和附加资源分层加载**：文章列表 / 详情先加载主资源；文章不可用时不请求 engagement，文章可读后再加载互动状态。
- **Content 需要服务内业务限流**：Gateway 粗限流只挡 IP / route 洪水；Content 还要按 actor、post、session、service caller、operation 和高成本资源保护草稿、发布、正文读取、互动、presence、管理端和内部调用。
- **Content 需要按 provider + operation 声明 resilience policy**：User、Upload、MongoDB、Redis、RabbitMQ 和 PostgreSQL 的 timeout、retry、熔断、max-in-flight 与降级策略见 `runtime-resilience.md`，不能只在实现里临时写 timeout。

## 当前设计状态

- 已明确：服务职责、数据归属、主要 API 族、跨服务依赖、事件方向、Go 落点、正文发布原子切换设计。
- 已设计草案：Content Go-first HTTP contract，见 `services/zhicore-content/api/http/`；该 contract 是 Go 侧新事实源，不承诺兼容 Java path / DTO，且尚未由 Go handler/test 验证。
- 已设计草案：Content engagement、公开浏览页面、业务限流和运行期 resilience 策略，见 `engagement-design.md`、`page-design.md`、`rate-limiting.md` 与 `runtime-resilience.md`；当前只固定设计和实现准入条件，不表示 Go runtime 已落地。
- 未完成：完整 migration SQL、服务级行为测试清单、Go handler / application / repository 实现。

## 下一步

- 生成 Content migration 草案。
- 实现“创建草稿并发布文章”核心切片，详见 [application-and-ports.md](application-and-ports.md)。
- 先写 domain 层测试，再写 application 层编排测试，最后接入 PostgreSQL / MongoDB / HTTP。
