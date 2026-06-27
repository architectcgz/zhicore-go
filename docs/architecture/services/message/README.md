# Message 服务设计

## 事实来源

- Java `zhicore-message` controller：Message command/query、Conversation query。
- `blog-message-im-integration.md`
- `zhicore-message-im-history-boundary.md`
- Java `zhicore-message/src/main/resources/db/schema.sql`

## 职责边界

`zhicore-message` 是博客消息业务编排层和 IM 适配层，负责私信发送规则、会话投影、消息中心聚合和未读摘要。

Message 不拥有系统通知事实；通知收件箱归 Notification。Message 可以聚合通知摘要，但不能修改 Notification 的通知状态。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/messages`：发送私信、召回、标记会话已读。
- `/api/v1/messages/conversations` 和 `/api/v1/conversations`：会话列表、会话详情、用户会话、未读数。
- 消息历史查询接口保留协议，但当前边界允许默认 provider 返回空列表，后续接外部 IM provider。

## 数据归属

Message 拥有：

- `conversations`
- `messages`：历史遗留表，新 Go 目标不默认把它作为新消息真相源。
- `message_outbox_task`

目标边界：

- 会话摘要、参与者、未读投影归 Message。
- 历史消息真相源逐步让位给外部 IM provider。
- 本地 outbox 负责发送后统计、审计或外部同步补偿。

## 主写流程

发送私信：

1. 校验用户关系、拉黑、私信设置和频率。
2. 过滤内容。
3. 调用外部 IM provider 发送消息。
4. 本地更新 `conversations` 摘要和未读投影。
5. 写 `message_outbox_task` 处理统计或补偿。

标记已读：

- 清理本地会话未读投影。
- 如果 IM provider 支持已读态，同步通过 adapter 调用。

## 跨服务依赖

- User：拉黑、私信设置、关注关系、用户摘要。
- Notification：消息中心聚合时读取通知摘要。
- 外部 IM provider：消息发送、历史消息、召回和多端同步。

## 事件

Message 生产：

- `message.sent`
- `message.read`

这些事件第一阶段不是核心跨服务事实；如后续 Notification 或运营分析需要消费，再提升到 `libs/contracts/events/message`。

## Go 目标落点

- HTTP：`services/zhicore-message/api/http`
- Application：`services/zhicore-message/internal/message/application`
- Domain：`services/zhicore-message/internal/message/domain`
- Ports：`services/zhicore-message/internal/message/ports`
- Infrastructure：`postgres`、`redis`、`rabbitmq`、`clients`
- Runtime：`services/zhicore-message/internal/message/runtime/module.go`

## 实现风险

- 既有实现存在“新消息不再沉淀本地 messages 明细，但召回仍依赖本地消息表”的裂缝。Go 设计必须明确召回真相源。
- 历史消息 provider 未接入前，接口可返回空列表，但前端兼容行为必须验证。
- 私信权限涉及 User 多个事实，不能在 Message 中复制用户关系表。

## 下一步

- 明确外部 IM provider 契约。
- 提取 Message HTTP 字段级 contract。
- 写发送、会话摘要、未读数、历史空列表兼容测试。
