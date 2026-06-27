# zhicore-id-generator

`zhicore-id-generator` 是旧 ID 生成服务在 Go 工作区中的保留映射目录。当前不迁移、不实现 HTTP API，也不接入其他 Go 服务。

服务职责：

- 当前不作为默认核心依赖。
- 当前没有使用 Snowflake 的业务场景，不补 Snowflake、批量 ID 或 segment ID endpoint。
- 保留 Snowflake、批量 ID 和按业务 tag 的 segment ID 生成能力作为未来可选设计落点。
- 作为 Go 服务目录中的可选能力落点，避免集中发号能力在设计中丢失。

数据归属：

- ID worker 配置
- segment 分配状态

Go 设计注意点：

- 当前默认 ID 策略见 `docs/architecture/id-strategy.md`：内部主键使用各服务数据库 `BIGINT` sequence / identity；外部公开 ID 使用独立 `public_id`、`public_no` 或 `order_no`。
- ID 发出后，具体业务实体归调用方服务所有。
- 只有未来重新确认跨数据库、离线、多主写入或集中发号需求，并登记业务 owner 后，才实现并接入该服务。
