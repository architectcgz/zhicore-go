# Comment 服务设计

`zhicore-comment` 是评论限界上下文的 Go 目标服务。服务级文档只记录边界、API 族和模块入口；模块内部的 API、application service、domain、ports、数据和事件设计见 `docs/architecture/module/comment/`。

## 事实来源

- Go 模块设计：`docs/architecture/module/comment/README.md`。
- Java `zhicore-comment` controller：`CommentCommandController`、`CommentQueryController`、`CommentLikeCommandController`、`CommentLikeQueryController`、`AdminComment*Controller`、`CommentOutboxAdminController`。
- Java `zhicore-comment` domain / application：`Comment`、`CommentStats`、`CommentLike`、评论命令、评论查询、点赞、首页缓存和 outbox 管理。
- Java `database/init-all-databases.sql` 和 `zhicore-comment/src/main/resources/db/schema.sql` 中 comment 表。
- Ranking、Content、Notification 对评论事件的消费设计。

## 职责边界

Comment 拥有：

- 评论、回复、评论树、评论状态和删除元数据。
- 评论文本、评论媒体引用、评论点赞、评论统计、文章级评论统计、点赞计数 delta、顶级评论 HOT 排序读模型和默认 RECOMMENDED 排序读模型。
- Comment 服务自己的 outbox 事件、首页评论缓存和热门候选本地缓存。

Comment 不拥有：

- 文章、用户资料、文件存储事实、通知收件箱或榜单分数。
- Admin 审核审计日志；Admin 可以暴露 facade，但删除评论必须委托 Comment mutation。
- File 文件元数据、对象存储路径或 URL 生成规则；Comment 只保存系统内媒体文件 ID，创建 / 更新时校验引用，展示 / 播放 URL 由读取时解析或缓存派生。Comment 不提供媒体上传 facade。

## 模块设计

| 文档 | 内容 |
| --- | --- |
| `docs/architecture/module/comment/README.md` | 模块职责、边界、API family、实现切片、关联服务和当前状态。 |
| `docs/architecture/module/comment/api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| `docs/architecture/module/comment/service.md` | Application service、事务边界、幂等、错误映射、缓存失效和实现切片。 |
| `docs/architecture/module/comment/domain.md` | 聚合、实体、值对象、不变量、领域服务和工厂。 |
| `docs/architecture/module/comment/ports.md` | repository、cache、client、event publisher、outbox 和 external adapter 端口归属。 |
| `docs/architecture/module/comment/data-events.md` | 数据归属、目标 schema 草案、缓存 key、事件 payload 和跨服务一致性。 |
| `docs/architecture/module/comment/runtime-resilience.md` | timeout、retry、熔断、降级、限流、健康检查和依赖故障语义。 |
| `docs/architecture/module/comment/decision-log.md` | 设计压测中已确认的决策、原因和后续依赖。 |
| [frontend pages/comment.md](../../../../../zhicore-frontend-vue/docs/design/pages/comment.md) | Comment 页面初设计、前端草稿、加载逻辑和降级规则。 |

## 设计复盘

| 文档 | 内容 |
| --- | --- |
| `decision-log/2026-06-27-comment-stats-hot-rank.md` | 记录 Comment 统计字段、点赞高 QPS、`comments + comment_stats` HOT join 排序和 `comment_hot_rank` 读模型的取舍。 |
| `docs/architecture/module/comment/decision-log.md` | 记录本轮 grillme 后确认的用户标识、删除、计数、排序、媒体、可见性、点赞和事务边界决策。 |

## 目标 API 范围

Comment 服务明确按 Go-first API 重做。目标 API 以文章为上级资源，用 `(postId, commentId)` 定位评论；`postId` 是 Content 对外文章 ID，`commentId` 是由 Comment 内部 `comments.id BIGINT IDENTITY` 派生的对外字符串。

| API family | 范围 | HTTP contract |
| --- | --- | --- |
| 创建和查询评论 | 创建根评论、传统分页、游标分页、增量补拉、详情、更新、删除 | `services/zhicore-comment/api/http/README.md` |
| 回复 | 创建回复已并入 `POST /api/v1/posts/{postId}/comments`；回复传统分页、回复游标分页、回复增量补拉 | 创建回复见 `services/zhicore-comment/api/http/endpoints/create-comment.md`；回复列表待提取 |
| 点赞 | 点赞、取消点赞、点赞状态、批量点赞状态、点赞数 | 待提取 |
| 管理 | 管理端查询、管理删除、outbox summary、dead retry | 待提取 |
| 媒体 | Comment 不提供媒体上传 facade；前端直接调用 File service 后把文件 ID 写入评论 | 无 Comment endpoint |

## 数据归属

Comment 拥有：

- `comments`
- `comment_stats`
- `comment_post_stats`
- `comment_likes`
- `comment_counter_deltas`
- `comment_hot_rank`
- `comment_recommended_rank`
- Comment 服务自己的 `outbox_events`

评论图片和语音只保存 File service 返回的文件 ID，例如 `imageFileIds`、`voiceFileId`。文件元数据、对象存储路径、URL 解析、签名 URL、CDN 规则和文件删除事实仍归 File service；Comment response 可以返回可展示 / 可播放 URL，但这些 URL 不是 Comment 的持久化事实。

`post_id` 在 Comment 本地表中保存 Content 对外 `postId` 字符串，用于 HTTP 定位、分区和查询条件。Comment 同时保存 Content 内部 `post_id BIGINT` opaque reference，用于跨服务事件让 Ranking 等下游直接落账；Comment 不依赖该内部 ID 的生成方式、连续性或可读含义。

## 跨服务依赖

| 依赖 | 用途 |
| --- | --- |
| Content | 创建评论前校验文章存在、可见性和是否允许评论；消费评论事件更新文章评论数。 |
| User | 查询作者摘要、用户状态、拉黑关系和互动权限。 |
| File | 校验评论图片和语音文件引用，查询时解析展示 / 播放 URL。 |
| Ranking | Comment 可以读取热门候选作为首页缓存输入，但不能拥有分数或榜单计算。 |
| Notification | 消费评论创建和点赞事件生成通知。 |
| Admin | 暴露管理 facade 和审计，删除命令委托 Comment。 |

## 实现风险

- Go 侧关键评论事件必须使用 producer outbox，不沿用 Java 历史中的直接 MQ 路径。
- 评论内部 ID 使用 PostgreSQL identity 生成；HTTP 对外 `commentId` 由内部 ID 派生，不使用 Redis、segment、每文章 counter 或独立发号服务。
- 回复模型使用 `root_id + parent_id`，不要再引入 `reply_to_comment_id` 或 `reply_to_user_id` 冗余字段。
- 顶级评论删除会影响所有回复，必须明确批量删除、统计修正和事件 `affectedCount` 语义。
- 点赞和取消点赞必须用 `comment_likes(comment_id, user_id)` 唯一约束保护幂等；点赞计数和 HOT / RECOMMENDED 排序通过 `comment_counter_deltas` 异步批量更新 `comment_stats` / rank 表，不要把高 QPS 点赞直接写到 `comments` 或同步更新同一统计行。
- 顶级评论 HOT 查询不做大范围 `comments + comment_stats` 排序 join；先从 `comment_hot_rank` 按 `(post_id, like_count DESC, comment_id ASC)` 取候选，再批量补评论正文、统计和作者摘要。
- 默认顶级评论流使用 `comment_recommended_rank`，按 `(post_id, recommended_score DESC, comment_id DESC)` 取候选；decay / recompute 必须过滤 `visible=true` 并使用锁或 claim 机制避免重复重算。
- 写路径外部 guard 不可确认时 fail closed；查询路径只允许作者摘要和 File URL 这类展示增强降级，具体矩阵见 `docs/architecture/module/comment/runtime-resilience.md`。
- 删除任意评论节点必须软删除整棵子树，并用本次实际从 `NORMAL` 变为 `DELETED` 的 `affectedCount` 维护统计和事件，避免重复删除导致重复扣减。
- Comment 与 Content 的评论总数需要 Ops 对账机制；Comment 的 `comment_post_stats` 是事实源，Content 的 `post_stats.comment_count` 是消费事件后的读模型。

## 下一步

1. 以 `services/zhicore-comment/api/http/README.md` 中首批 contract 为准，实现“创建根评论 / 回复 + 文章评论传统分页查询”；`POST` 已有 `parentCommentId`，实现不能只支持根评论。
2. 先补 domain / application 测试，验证评论内容、`commentId` 生成 / 编码、`parentCommentId` 解析、Content/User/File 校验、统计初始化、文章级统计、回复计数、outbox 写入和分页查询。
3. 再接入 PostgreSQL / HTTP；切片 2 再补删除、点赞、计数 delta worker、缓存失效和事件发布。
