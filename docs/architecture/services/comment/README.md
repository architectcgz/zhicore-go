# Comment 服务设计

`zhicore-comment` 是评论限界上下文的 Go 目标服务。服务级文档只记录边界、API 族和模块入口；模块内部的 API、application service、domain、ports、数据和事件设计见 `docs/architecture/module/comment/`。

## 事实来源

- Go 模块设计：`docs/architecture/module/comment/README.md`。
- Java `zhicore-comment` controller：`CommentCommandController`、`CommentQueryController`、`CommentLikeCommandController`、`CommentLikeQueryController`、`CommentMediaController`、`AdminComment*Controller`、`CommentOutboxAdminController`。
- Java `zhicore-comment` domain / application：`Comment`、`CommentStats`、`CommentLike`、评论命令、评论查询、点赞、媒体上传、首页缓存和 outbox 管理。
- Java `database/init-all-databases.sql` 和 `zhicore-comment/src/main/resources/db/schema.sql` 中 comment 表。
- Ranking、Content、Notification 对评论事件的消费设计。

## 职责边界

Comment 拥有：

- 评论、回复、评论树、评论状态和删除元数据。
- 评论文本、评论媒体引用、评论点赞和评论统计。
- Comment 服务自己的 outbox 事件、首页评论缓存和热门候选本地缓存。

Comment 不拥有：

- 文章、用户资料、文件存储事实、通知收件箱或榜单分数。
- Admin 审核审计日志；Admin 可以暴露 facade，但删除评论必须委托 Comment mutation。
- Upload 文件元数据；Comment 只保存展示 / 播放 URL 快照，不保存媒体文件 ID、对象存储路径或 URL 生成规则。

## 模块设计

| 文档 | 内容 |
| --- | --- |
| `docs/architecture/module/comment/README.md` | 模块职责、边界、API family、实现切片、关联服务和当前状态。 |
| `docs/architecture/module/comment/api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| `docs/architecture/module/comment/service.md` | Application service、事务边界、幂等、错误映射、缓存失效和实现切片。 |
| `docs/architecture/module/comment/domain.md` | 聚合、实体、值对象、不变量、领域服务和工厂。 |
| `docs/architecture/module/comment/ports.md` | repository、cache、client、event publisher、outbox 和 external adapter 端口归属。 |
| `docs/architecture/module/comment/data-events.md` | 数据归属、目标 schema 草案、缓存 key、事件 payload 和跨服务一致性。 |

## 目标 API 范围

Comment 服务明确按 Go-first API 重做，不保留以全局 `commentId` 为外部资源 ID 的旧 API 形态。目标 API 以文章为上级资源，用 `(postId, floor)` 定位评论；`postId` 是 Content 对外文章 ID，`floor` 是文章内单调递增楼层号。

| API family | 范围 | HTTP contract |
| --- | --- | --- |
| 创建和查询评论 | 创建根评论、传统分页、游标分页、增量补拉、详情、更新、删除 | `services/zhicore-comment/api/http/README.md` |
| 回复 | 创建回复、回复传统分页、回复游标分页、回复增量补拉 | 待提取 |
| 点赞 | 点赞、取消点赞、点赞状态、批量点赞状态、点赞数 | 待提取 |
| 管理 | 管理端查询、管理删除、outbox summary、dead retry | 待提取 |
| 媒体 facade | 评论图片和语音上传 facade，可由目标前端直接改用 Upload | 待定 |

## 数据归属

Comment 拥有：

- `comments`
- `comment_stats`
- `comment_likes`
- `comment_post_counters`
- Comment 服务自己的 `outbox_events`

评论图片和语音只保存 Upload / File Service 返回的可展示 / 可播放 CDN URL。文件元数据、对象存储路径、URL 解析和文件删除事实仍归 Upload / File Service。

## 跨服务依赖

| 依赖 | 用途 |
| --- | --- |
| Content | 创建评论前校验文章存在、可见性和是否允许评论；消费评论事件更新文章评论数。 |
| User | 查询作者摘要、用户状态、拉黑关系和互动权限。 |
| Upload | 评论图片和语音上传、文件 URL 解析。 |
| Ranking | Comment 可以读取热门候选作为首页缓存输入，但不能拥有分数或榜单计算。 |
| Notification | 消费评论创建和点赞事件生成通知。 |
| Admin | 暴露管理 facade 和审计，删除命令委托 Comment。 |

## 实现风险

- Go 侧关键评论事件必须使用 producer outbox，不沿用 Java 历史中的直接 MQ 路径。
- `floor` 是文章内外部定位号，必须在同一 `post_id` 下唯一、单调递增、删除不复用，不能用动态行号或回复列表内序号替代。
- 回复模型使用 `root_id + parent_id`，不要再引入 `reply_to_comment_id` 或 `reply_to_user_id` 冗余字段。
- 顶级评论删除会影响所有回复，必须明确批量删除、统计修正和事件 `affectedCount` 语义。
- 点赞和取消点赞必须用唯一约束和原子计数保护幂等，不要只依赖 Redis。
- 媒体上传 facade 如果保留，必须明确文件归属不转移，并处理上传成功但评论创建失败的补偿策略。

## 下一步

1. 以 `services/zhicore-comment/api/http/README.md` 中首批 contract 为准，实现“创建根评论 + 文章评论传统分页查询”。
2. 先补 domain / application 测试，验证评论内容、楼层分配、Content/User 校验、统计初始化和分页查询。
3. 再接入 PostgreSQL / HTTP；切片 2 再补删除、点赞、outbox、缓存失效和事件发布。
