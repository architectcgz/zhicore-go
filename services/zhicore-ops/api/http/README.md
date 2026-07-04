# zhicore-ops HTTP Schema

本目录记录 `zhicore-ops` 的内部运维 HTTP contract。当前仅做计划化占位；Java 灰度接口不迁移为当前 Go 事实源。

## Provider Owner

Ops 只作为内部迁移、检查、对账、修复、回放或运维工具落点。Ops 不拥有业务事实表，不改变业务数据所有权。

## 首批 endpoint 候选

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/ops/health` | Ops 自身健康检查 | API 族已识别 |
| `POST` | `/api/v1/ops/reconcile` | 受控对账任务 | 待确认 |
| `POST` | `/api/v1/ops/repair-tasks` | 受控修复任务 | 待确认 |
| `POST` | `/api/v1/ops/event-replay` | 事件回放任务 | 待确认 |

## 不迁移项

- Java `/api/gray` 灰度配置、用户灰度、推进、回滚和 reconcile 接口当前不迁移。
- 当前开发阶段不规划 Java/Go 运行时并存或灰度发布。

## 禁止规则

- Ops 工具如需写业务数据，必须通过归属服务 contract 或受控 repair 任务，不能绕过服务边界随意改表。
- 不把一次性排障脚本伪装成长期公开 API。
- 暂不创建前端 `src/api/ops.ts`，直到具体运维 endpoint 达到 `Contract 草案`。
