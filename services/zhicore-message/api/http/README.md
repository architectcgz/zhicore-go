# zhicore-message HTTP Schema

本目录记录 `zhicore-message` 的对外 HTTP contract。当前仅做计划化占位，字段级 endpoint schema 待后续按私信和会话切片提取。

## Provider Owner

Message 拥有私信发送规则、会话投影、消息中心聚合和未读摘要。Notification 拥有系统通知收件箱；User 拥有关系、拉黑和私信设置源事实；外部 IM provider 可拥有消息历史源事实。

## 首批 endpoint 候选

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/messages` | 发送私信 | API 族已识别 |
| `GET` | `/api/v1/messages/conversations` | 会话列表 | API 族已识别 |
| `GET` | `/api/v1/messages/conversations/{conversationId}` | 会话详情 | API 族已识别 |
| `POST` | `/api/v1/messages/conversations/{conversationId}/read` | 标记会话已读 | API 族已识别 |
| `GET` | `/api/v1/messages/unread-count` | 私信未读数 | API 族已识别 |

## 待提取 contract

- 私信发送请求、附件、风控、拉黑和陌生人设置错误码。
- 会话分页、会话摘要、历史消息 provider 未接入时的空列表兼容语义。
- 未读数与外部 IM provider 已读态同步边界。

## 禁止规则

- 不把 Notification 的系统通知 DTO 放入 Message。
- 不复制 User 关系表；权限判断通过 User contract。
- 暂不创建前端 `src/api/message.ts`，直到 endpoint 达到 `Contract 草案`。
