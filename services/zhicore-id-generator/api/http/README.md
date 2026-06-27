# zhicore-id-generator HTTP Schema

当前状态：不迁移 / 不提供 HTTP API。

## 决策

2026-06-27 已确认当前没有使用 Snowflake 的业务场景，`zhicore-id-generator` 不需要作为 Go 服务实现，也不补以下历史 endpoint contract：

- `GET /api/v1/id/snowflake`
- `GET /api/v1/id/snowflake/batch`
- `GET /api/v1/id/segment/{bizTag}`

## 依据

- 当前默认 ID 策略见 `docs/architecture/id-strategy.md`：内部主键使用各服务数据库 `BIGINT` sequence / identity。
- 外部公开 ID 和业务编号由各资源 owner 自己设计，例如 `public_id`、`public_no`、`order_no`。
- 没有已知 Go 服务需要集中 Snowflake、批量发号或 segment 号段。

## 恢复条件

只有未来同时满足以下条件时，才重新打开本服务 HTTP contract：

- 存在明确业务 owner 和调用方。
- 数据库 sequence / identity 不能满足该场景。
- 已说明跨数据库、离线、多主写入、时钟回拨、worker 分配、号段缓存、容量和高可用要求。
- 已更新 `docs/architecture/id-strategy.md` 和 `docs/architecture/services/id-generator/README.md`。
