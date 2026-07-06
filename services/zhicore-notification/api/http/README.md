# zhicore-notification HTTP Schema

本目录记录 `zhicore-notification` 的对外 HTTP contract。通知中心首批 endpoint 当前为 `Contract 草案`，Go handler 和 system HTTP test 完成后再标记为已验证。

## Provider Owner

Notification 拥有通知收件箱、通知聚合状态、未读数、用户通知偏好、免打扰、作者订阅、campaign、delivery ledger 和实时 fanout 语义。它不拥有触发通知的用户、文章、评论、私信或榜单源事实。

## 首批通知中心 endpoint

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/notifications` | 当前用户聚合通知列表 | Contract 草案 |
| `POST` | `/api/v1/notifications/{notificationId}/read` | 标记单条通知已读 | Contract 草案 |
| `POST` | `/api/v1/notifications/read-all` | 全部已读；canonical path | Contract 草案 |
| `POST` | `/api/v1/notifications/mark-all-read` | 全部已读；兼容 alias | Contract 草案 |
| `GET` | `/api/v1/notifications/unread-count` | 当前用户未读总数；canonical path | Contract 草案 |
| `GET` | `/api/v1/notifications/unread/count` | 当前用户未读总数；兼容 alias | Contract 草案 |
| `GET` | `/api/v1/notifications/unread/breakdown` | 当前用户按 category 的未读数 | Contract 草案 |
| `GET` | `/api/v1/notifications/preferences` | 通知偏好；后续偏好切片细化 | API 族已识别 |
| `PUT` | `/api/v1/notifications/preferences` | 更新通知偏好；后续偏好切片细化 | API 族已识别 |

## ID 约定

- HTTP path 和 response 中的 `notificationId` 都是 `notifications.public_id` 字符串，例如 `n1...`。
- 内部 `notifications.id BIGINT` 只用于 Notification 服务内事务、索引和表关联，不进入外部 HTTP contract。
- `notificationId` 解析失败属于参数错误；解析成功但不属于当前用户时，application 按可见性规则返回不存在或无权限。

## 待提取 contract

- 通知列表分页、聚合组、未读状态和 payload 展示快照。
- 偏好 / DND / 作者订阅字段和默认值。
- WebSocket / realtime fanout 与 HTTP 查询的一致性边界。

## 禁止规则

- 不复制 Content、Comment、User、Message 的源对象 DTO。
- Gateway 只能承载连接和转发，不拥有 Notification 收件箱或未读事实。
- 暂不创建前端 `src/api/notification.ts`，直到 endpoint 达到 `Contract 草案`。
