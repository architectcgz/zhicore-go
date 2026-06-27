# zhicore-ranking

`zhicore-ranking` 是排行榜服务的 Go 目标服务模块。

服务职责：

- 拥有榜单分数、榜单 ledger、榜单快照、热门文章、创作者榜、话题榜和排行榜归档。
- 消费 Content、Comment 等服务的互动事件，构建 Ranking 自己的读模型。
- 在需要展示详情时调用 Content 获取文章权威信息。

数据归属：

- Ranking Redis key
- Ranking MongoDB 归档
- Ranking PostgreSQL ledger/snapshot 表，如果后续正式落库

Go 设计注意点：

- Ranking 拥有分数计算和排序结果，不拥有文章、点赞、收藏、评论、用户或标签。
- 事件消费必须幂等，避免重复投递造成分数累加错误。
