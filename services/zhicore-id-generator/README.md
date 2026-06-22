# zhicore-id-generator

`zhicore-id-generator` 是 ID 生成服务的 Go 迁移模块。

服务职责：

- 提供 Snowflake ID 生成。
- 提供批量 ID 生成。
- 提供按业务 tag 的 segment ID 生成。

数据归属：

- ID worker 配置
- segment 分配状态

迁移注意点：

- ID 发出后，具体业务实体归调用方服务所有。
- 该服务适合作为第一个迁移目标，用于验证 Go 服务构建、部署、健康检查和网关路由。
