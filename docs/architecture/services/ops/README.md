# Ops 服务设计

## 事实来源

- Java `zhicore-ops` controller：`GrayReleaseController`。
- 当前迁移决策：不做灰度，不规划 Java/Go 运行时并存。

## 职责边界

`zhicore-ops` 在 Go 目标中不是普通业务服务。它保留为内部迁移、检查、对账、修复、回放或运维工具的落点。

Java 的灰度配置、用户灰度、推进、回滚接口不迁移为当前 Go 事实源。

## API 保留范围

Java 侧存在 `/api/gray` 相关接口：

- config 查询和更新。
- 用户灰度检查。
- rollback、advance、reconcile。
- reconciliation result。
- user flags 清理。

Go 目标当前不保留这些运行时灰度 API，除非后续重新确认 Java/Go 并存或灰度发布需求。

## 数据归属

Ops 不拥有业务事实表。

如果后续需要迁移运维能力，可以拥有：

- 对账任务表。
- 修复任务表。
- 迁移检查结果。
- CDC repair checkpoint。

这些数据只服务运维，不改变业务数据所有权。

## Go 目标落点

- HTTP：`services/zhicore-ops/api/http`
- Application：`services/zhicore-ops/internal/ops/application`
- Domain：`services/zhicore-ops/internal/ops/domain`
- Ports：`services/zhicore-ops/internal/ops/ports`
- Infrastructure：`postgres`、`clients`
- Runtime：`services/zhicore-ops/internal/ops/runtime/module.go`

## 迁移风险

- 把灰度接口迁移到 Go 会引入与当前目标相反的复杂度。
- Ops 工具若能写业务库，必须通过归属服务 contract 或受控 repair 任务，不能绕过服务边界随意改表。

## 下一步

- 暂不实现灰度 API。
- 后续如果需要迁移工具，先定义具体任务：对账、补偿、事件重放或数据修复。
