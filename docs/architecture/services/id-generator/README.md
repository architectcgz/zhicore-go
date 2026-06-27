# ID Generator 服务设计

## 事实来源

- Java `zhicore-id-generator` controller：`IdGeneratorController`。
- Java `zhicore-client` 中 `IdGeneratorFeignClient`。
- `docs/architecture/id-strategy.md` 中 Go 目标 ID 决策。

## 职责边界

`zhicore-id-generator` 当前不迁移、不实现 HTTP API，也不作为 Go 后端默认核心依赖。2026-06-27 已确认当前没有使用 Snowflake 的业务场景。

本目录仅保留旧服务映射和未来可选集中发号能力的设计记录；后续任务不得按默认服务交付顺序补 Snowflake / batch / segment contract，除非重新出现明确业务 owner。

普通业务服务内部主键默认使用各服务数据库 sequence / identity。外部公开 ID 和业务编号按各自服务设计。

## API 保留范围

Java 侧存在以下接口：

- `GET /api/v1/id/snowflake`
- `GET /api/v1/id/snowflake/batch`
- `GET /api/v1/id/segment/{bizTag}`

Go 第一阶段不保留这些 HTTP 兼容入口，不补 endpoint contract。若未来发现外部调用方仍依赖 `/api/v1/id/*`，必须先登记调用方、业务 owner、迁移原因和容量 / 时钟 / 高可用要求，再重新打开 API contract。

## 数据归属

如果未来实现集中发号，ID Generator 可以拥有：

- segment 号段分配表。
- worker 节点租约或发号实例状态。
- 发号审计和监控指标。

当前迁移不创建默认业务表，也不把它作为 User/Content/Comment 等服务的主键来源。

## Go 目标落点

- HTTP：`services/zhicore-id-generator/api/http/README.md`，当前仅记录“不迁移 / 不提供 API”状态。
- Application / Domain / Ports / Infrastructure / Runtime：当前不落地实现，保留目录骨架不代表服务已进入交付范围。

## 实现风险

- 直接迁移 Java 的 `IdGeneratorFeignClient` 会把所有服务重新耦合到中心发号服务，这和当前 Go ID 决策冲突。
- 如果对外保留 snowflake 接口，需要明确 worker id 分配、时钟回拨和多实例部署策略。
- 如果使用 segment，需要明确数据库扩容和号段缓存策略。

## 下一步

- 默认不继续实现。
- 如未来要恢复集中发号，先更新 `docs/architecture/id-strategy.md` 和本文件，再补独立 HTTP contract、容量模型和测试目标。
