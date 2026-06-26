# Context Map

本仓库按服务限界上下文组织领域语言。各服务的 `CONTEXT.md` 只记录术语和关系，不承载实现方案；实现决策放在架构文档和 ADR。

## Contexts

- [Content](docs/architecture/services/content/CONTEXT.md) — 拥有文章主数据、正文、草稿、发布生命周期、标签、互动统计和作者快照。

## Relationships

- **Content -> Upload**：Content 只保存 Upload 文件引用，Upload 拥有文件对象和物理清理。
- **Content -> User**：Content 保存作者快照，但 User 拥有用户资料事实。
- **Content -> Search**：Content 发布文章事件，Search 拉取当前 published body 做全文索引。
- **Content -> Ranking / Notification**：Ranking 和 Notification 消费 Content 轻量事件，不读取正文。
