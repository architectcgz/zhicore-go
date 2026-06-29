# Comment 限流设计

本文是 Comment 模块限流策略的事实源。

## 两层限流

| 层级 | 归属 | 职责 |
|------|------|------|
| Gateway 粗限流 | `zhicore-gateway` | 按 IP、route、method 做突发流量拦截。 |
| Comment 业务限流 | `zhicore-comment` | 按 userId、postId、IP 做评论频率控制；可识别用户状态和评论内容类型。 |

## API 矩阵

| API | Gateway 粗限流 | Comment 业务限流 | Redis 不可用时 |
|-----|--------------|-----------------|--------------|
| `POST /api/v1/posts/{postId}/comments`（创建根评论） | IP + route | `userId`：同一用户 60s 内创建评论数；`postId + userId`：同一用户对同一文章 60s 内创建数；`IP`：匿名或兜底 | fail-closed：Redis 不可用时使用本机限流，阈值不超过 Redis 正常阈值的 20%；不能 fail-open |
| `POST /api/v1/posts/{postId}/comments`（创建回复，`parentCommentId` 非空） | IP + route | `userId`：60s 内全局回复数；`rootCommentId + userId`：对同一根评论的回复频率 | 同上 |
| `PUT /api/v1/comments/{commentId}`（更新评论） | IP + route | `userId`：60s 内更新数；防止频繁刷新规避内容过滤 | fail-closed |
| `DELETE /api/v1/comments/{commentId}`（删除评论） | IP + route | `userId` 轻量读限流，允许合理批量清理 | 可短时 fail-open，删除是降低风险动作 |
| `POST /api/v1/comments/{commentId}/like`（点赞） | IP + route | `userId`：60s 内点赞总数；防止刷热度 | fail-closed |
| `DELETE /api/v1/comments/{commentId}/like`（取消点赞） | IP + route | `userId` 轻量 | 可短时 fail-open |

## 默认阈值（首批，可配置）

所有阈值必须配置化，不能写死在 handler 中。

| 维度 | 窗口 | 阈值 | 说明 |
|------|------|------|------|
| `userId` 全局发评 | 60s | 10 条 | 正常用户浏览后评论频率 |
| `userId + postId` 发评 | 60s | 5 条 | 防止单帖刷楼 |
| `userId + postId` 回复 | 60s | 10 条 | 对话场景稍高 |
| `userId` 全局点赞 | 60s | 30 次 | 允许批量浏览点赞 |
| IP 匿名发评 | 60s | 3 条 | 匿名路径最严格 |

## Redis 故障策略

- 评论创建是核心写路径，不能 fail-open（禁止在 Redis 完全不可用时放行无限制的写入）。
- Redis 短时不可用（< 30s）：切换到本机内存限流，阈值 = Redis 阈值 × 20%；记录 `comment_ratelimit_degraded_total` metric。
- Redis 长时不可用（> 30s）：直接返回 `1004 SERVICE_DEGRADED`，写审计日志；不允许用本机内存兜底超过 30s（防止单实例内存不一致）。
- 点赞类操作可以在 Redis 不可用时适度放宽，但需记录 degraded metric。

## 限流 key 规则

- key 只使用规范化标识符，禁止记录原始 IP、用户输入文本、Authorization、token 或 cookie。
- IP 需规范化：IPv4 直接使用，IPv6 截取 /48 前缀。
- `userId` 使用数值字符串，不使用账号名或邮箱。

## 观测

每类限流至少记录：

- `allow` / `reject` 计数，标签：`route`、`limitType`（`user_global` | `user_post` | `ip`）
- `degraded_fallback` 计数（使用本机内存兜底次数）
- `redis_unavailable` 计数

metrics label 不包含 `userId` 具体值、帖子 ID 具体值、IP 具体值（只保留聚合维度名称）。
