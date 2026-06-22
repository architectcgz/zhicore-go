# zhicore-search

`zhicore-search` 是搜索服务的 Go 迁移模块。

服务职责：

- 拥有搜索索引、搜索建议、热门搜索词、搜索历史和搜索专用读模型。
- 消费 Content 事件更新或删除文章索引。
- 在索引修复或结果补全时调用 Content 获取文章权威详情。

数据归属：

- Elasticsearch index
- 搜索服务本地的 suggestion/history 存储

迁移注意点：

- Search 返回的是派生读模型，文章详情和可见性仍由 Content 决定。
- RabbitMQ 消费者需要处理重复投递、乱序事件和重建索引场景。
