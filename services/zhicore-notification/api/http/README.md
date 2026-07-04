# zhicore-notification HTTP Schema

本目录记录 `zhicore-notification` 的对外 HTTP contract。当前仅做计划化占位，字段级 endpoint schema 待后续按通知中心切片提取。

## Provider Owner

Notification 拥有通知收件箱、通知聚合状态、未读数、用户通知偏好、免打扰、作者订阅、campaign、delivery ledger 和实时 fanout 语义。它不拥有触发通知的用户、文章、评论、私信或榜单源事实。

## 首批 endpoint 候选

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/notifications` | 当前用户通知列表 | API 族已识别 |
| `GET` | `/api/v1/notifications/unread-count` | 当前用户未读数 | API 族已识别 |
| `POST` | `/api/v1/notifications/{notificationId}/read` | 标记单条已读 | API 族已识别 |
| `POST` | `/api/v1/notifications/read-all` | 全部已读 | API 族已识别 |
| `GET` | `/api/v1/notifications/preferences` | 通知偏好 | API 族已识别 |
| `PUT` | `/api/v1/notifications/preferences` | 更新通知偏好 | API 族已识别 |

## 待提取 contract

- 通知列表分页、聚合组、未读状态和 payload 展示快照。
- 偏好 / DND / 作者订阅字段和默认值。
- WebSocket / realtime fanout 与 HTTP 查询的一致性边界。

## 禁止规则

- 不复制 Content、Comment、User、Message 的源对象 DTO。
- Gateway 只能承载连接和转发，不拥有 Notification 收件箱或未读事实。
- 暂不创建前端 `src/api/notification.ts`，直到 endpoint 达到 `Contract 草案`。
