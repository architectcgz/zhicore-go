# Content Post Events

状态：草案。本文固定 `content.post.*` 事件的字段级 payload contract，尚未由 Go outbox publisher / consumer contract test 验证。

## Envelope

所有事件使用 `docs/contracts/events.md` 的统一 envelope。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `eventId` | string | 是 | 全局唯一事件 ID，consumer 幂等键。 |
| `eventType` | string | 是 | 事件名，例如 `content.post.published`。 |
| `payloadVersion` | int | 是 | 首版为 `1`。 |
| `producer` | string | 是 | 固定为 `zhicore-content`。 |
| `occurredAt` | string | 是 | 业务事实发生时间，RFC3339。 |
| `aggregateType` | string | 是 | 固定为 `post`。 |
| `aggregateId` | string | 是 | Content `publicPostId`。 |
| `aggregateVersion` | int | 条件必填 | 生命周期和可见性事件必填；互动事件可选。 |
| `payload` | object | 是 | 事件 payload。 |

## 通用 Payload 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`，外部公开文章 ID。 |
| `postId` | int64 | 否 | Content 内部 `post_id` opaque reference，仅用于 consumer 减少解析调用。 |
| `authorId` | int64 | 视事件而定 | 作者 User 内部 ID opaque reference。 |

如果事件缺少内部 `postId`，Ranking 等 consumer 必须通过 Content contract 解析 `publicPostId`；解析 transient 失败时按 consumer 重试 / DLQ 策略处理。

## 事件索引

| eventType / routing key | 触发事实 | Outbox | 主要 consumer |
| --- | --- | --- | --- |
| `content.post.published` | 文章公开发布或恢复为公开可见 | 必须 | Search、Ranking、Notification |
| `content.post.updated` | 已发布文章公开摘要、封面、作者快照或正文指针变化 | 必须 | Search、Ranking |
| `content.post.deleted` | 文章被删除或管理端删除 | 必须 | Search、Ranking |
| `content.post.visibility_changed` | 撤回、恢复、隐藏、下架、重新公开等可见性变化 | 必须 | Search、Ranking |
| `content.post.tags.updated` | 文章标签 / 话题引用变化 | 必须 | Search、Ranking |
| `content.post.liked` | 用户点赞文章 | 必须 | Notification、Ranking |
| `content.post.unliked` | 用户取消点赞文章 | 必须 | Ranking |
| `content.post.favorited` | 用户收藏文章 | 必须 | Ranking |
| `content.post.unfavorited` | 用户取消收藏文章 | 必须 | Ranking |
| `content.post.viewed` | 有效浏览事实 | 可批量 / 可降级 | Ranking |

## 生命周期和元数据事件

### `content.post.published`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 是 | 作者 User 内部 ID。 |
| `title` | string | 是 | 已发布标题。 |
| `summary` | string | 否 | 已发布摘要，可缺失。 |
| `coverFileId` | string | 否 | Upload 文件 ID，可缺失。 |
| `publishedAt` | string | 是 | 发布时间，RFC3339。 |
| `topicIds` | string[] | 否 | Content / Topic 拥有的话题公开 ID 列表。 |
| `publishedBodyId` | string | 否 | MongoDB published body 引用；consumer 不直接读取正文时可忽略。 |
| `publishedBodyHash` | string | 否 | 已发布正文 hash，用于 Search repair / audit。 |

语义：

- `publicVisible=true` 可由 consumer 从本事件推导，但需要完整可见性原因时仍以 `content.post.visibility_changed` 为准。
- Search 收到后通过 Content internal API 拉取正文；事件不携带正文全文。

### `content.post.updated`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 否 | 作者 User 内部 ID。 |
| `title` | string | 否 | 更新后的标题。 |
| `summary` | string | 否 | 更新后的摘要。 |
| `coverFileId` | string | 否 | 更新后的封面文件 ID。 |
| `publishedAt` | string | 否 | 当前发布时间。 |
| `topicIds` | string[] | 否 | 当前话题 ID 列表。 |
| `publishedBodyId` | string | 否 | 当前 published body 引用。 |
| `publishedBodyHash` | string | 否 | 当前正文 hash。 |
| `updatedAt` | string | 是 | 更新时间，RFC3339。 |

缺失字段表示本次事件不声明该字段变化，consumer 不得把缺失解释为空值。

### `content.post.deleted`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 否 | 作者 User 内部 ID。 |
| `deletedAt` | string | 是 | 删除时间，RFC3339。 |
| `deletedBy` | int64 | 否 | 操作者 User 内部 ID；系统任务可缺失。 |
| `deletedByRole` | string | 否 | `AUTHOR`、`ADMIN`、`SYSTEM`。 |
| `reason` | string | 否 | 删除原因机器可读值或审计摘要。 |

Ranking 消费本事件只更新可见性投影，不写热度 ledger。

### `content.post.visibility_changed`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 否 | 作者 User 内部 ID。 |
| `oldVisibility` | string | 是 | 变化前可见性，例如 `PUBLIC`、`UNPUBLISHED`、`HIDDEN`、`TAKEN_DOWN`、`DELETED`。 |
| `newVisibility` | string | 是 | 变化后可见性。 |
| `publicVisible` | boolean | 是 | 是否允许进入公开查询、Search 索引和 Ranking 公开榜单。 |
| `reason` | string | 是 | 机器可读原因，例如 `AUTHOR_UNPUBLISHED`、`ADMIN_TAKEN_DOWN`、`RESTORED`。 |
| `changedAt` | string | 是 | 可见性事实时间，RFC3339。 |
| `topicIds` | string[] | 否 | 可见性恢复时可携带当前话题快照。 |

`aggregateVersion` 对本事件必填。consumer 必须用 `aggregateVersion` 或 `changedAt` 防止旧事件覆盖新投影。

### `content.post.tags.updated`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 否 | 作者 User 内部 ID。 |
| `topicIds` | string[] | 是 | 更新后的完整话题 ID 列表。 |
| `updatedAt` | string | 是 | 更新时间，RFC3339。 |

## 互动事件

### `content.post.liked`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 是 | 文章作者 User 内部 ID。 |
| `likedBy` | int64 | 是 | 点赞用户 User 内部 ID。 |

上游保证同一用户对同一文章重复点赞不重复发布本事件。

### `content.post.unliked`

字段同 `content.post.liked`，但用户字段为 `unlikedBy`。事件表达 `LIKE` 指标 `-1`。

### `content.post.favorited`

字段同 `content.post.liked`，但用户字段为 `favoritedBy`。事件表达 `FAVORITE` 指标 `+1`。

### `content.post.unfavorited`

字段同 `content.post.liked`，但用户字段为 `unfavoritedBy`。事件表达 `FAVORITE` 指标 `-1`。

### `content.post.viewed`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `postId` | int64 | 否 | Content 内部 `post_id`。 |
| `authorId` | int64 | 否 | 作者 User 内部 ID。 |
| `publishedAt` | string | 否 | 当前发布时间，供 Ranking 衰减计算兜底。 |
| `viewerId` | int64 | 条件必填 | 登录用户 ID。 |
| `ipHash` | string | 条件必填 | 匿名浏览去重 hash，由 Gateway 或 Content 生成，不是原始 IP。 |

`viewerId` 和 `ipHash` 至少提供一个；两者都缺失时 Content 不应发布给 Ranking 的浏览热度事件。Ranking 可在 Redis 不可用时按运行期策略丢弃 view，不写 ledger。
