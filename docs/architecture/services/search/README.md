# Search 服务设计

## 事实来源

- Java `zhicore-search` controller：Search command/query。
- Java 架构文档中的 Search 职责。
- Content 事件和 Post client contract。

## 职责边界

`zhicore-search` 拥有搜索索引、搜索建议、热门搜索词、搜索历史和搜索相关读模型。

Search 不拥有文章事实、用户事实或排行分数。搜索结果可以返回索引中的预览数据，也可以调用 Content 补齐权威详情。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/search/posts`：文章搜索。
- `/api/v1/search/suggest`：搜索建议。
- `/api/v1/search/hot`：热门搜索词。
- `/api/v1/search/history`：搜索历史查询和清理。

## 数据归属

Search 拥有：

- Elasticsearch 文章索引。
- 搜索建议索引或 Redis/ES 读模型。
- 用户搜索历史本地读模型。

Search 不拥有 `posts`、`users`、`comments` 表。

## 事件

Search 消费：

- `content.post.published`
- `content.post.updated`
- `content.post.deleted`
- `content.post.tags.updated`

事件处理必须幂等，索引重建可以回源 Content。

## 跨服务依赖

- Content：索引修复、详情补齐和批量文章查询。
- User：如果搜索结果展示作者最新摘要，可以消费 User 事件或调用 User contract；默认优先使用 Content 事件里的作者快照。

## Go 目标落点

- HTTP：`services/zhicore-search/api/http`
- Application：`services/zhicore-search/internal/search/application`
- Domain：`services/zhicore-search/internal/search/domain`
- Ports：`services/zhicore-search/internal/search/ports`
- Infrastructure：`es`、`redis`、`rabbitmq`、`clients`
- Runtime：`services/zhicore-search/internal/search/runtime/module.go`

## 迁移风险

- 搜索排序、分页和高亮字段容易影响前端展示，字段级 contract 必须从 Java DTO 提取。
- 索引事件可能乱序，删除事件必须覆盖旧的更新事件。
- 回源 Content 不能变成每次搜索都同步 N+1 调用。

## 下一步

- 提取 Search API contract。
- 固定 Elasticsearch index mapping。
- 设计索引重建和事件消费幂等测试。
