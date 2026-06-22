# zhicore-id-generator

`zhicore-id-generator` 是 ID 生成服务的 Go 迁移模块。

服务职责：

- 当前不作为默认核心依赖。
- 保留 Snowflake、批量 ID 和按业务 tag 的 segment ID 生成能力作为未来可选落点。
- 对齐 Java 迁移映射，避免迁移评估时丢失模块。

数据归属：

- ID worker 配置
- segment 分配状态

迁移注意点：

- 当前默认 ID 策略见 `docs/architecture/id-strategy.md`：内部主键使用各服务数据库 `BIGINT` sequence / identity；外部公开 ID 使用独立 `public_id`、`public_no` 或 `order_no`。
- ID 发出后，具体业务实体归调用方服务所有。
- 只有未来重新确认跨数据库、离线、多主写入或集中发号需求时，才实现并接入该服务。
