# ID Generator 页面设计

本文记录 ID Generator 的页面设计结论。当前 `zhicore-id-generator` 不迁移、不提供 HTTP API，也没有面向普通用户或管理端的产品页面。

当前状态：无产品页面；仅保留未来恢复集中发号能力时的运维诊断页占位。

## 当前页面结论

- 不设计普通用户页面。
- 不设计默认管理页面。
- 不在 Admin、Ops 或 Gateway 中展示 ID Generator 入口。
- 不为历史 Snowflake / segment endpoint 构建页面预览。

依据见 `README.md` 和 `services/zhicore-id-generator/api/http/README.md`：当前 Go 目标使用各服务数据库 identity、服务内公开 ID 和业务编号，不依赖集中发号服务。

## 恢复集中发号后的页面范围

只有未来重新确认集中发号服务需要迁移，且已更新 `docs/architecture/id-strategy.md` 与本服务 README 后，才补以下页面。

### 发号状态诊断

```text
┌────────────────────────────────────────────┐
│ ID generator diagnostics                   │
├────────────────────────────────────────────┤
│ Mode: snowflake · segment                  │
│ worker lease · clock drift · throughput    │
├────────────────────────────────────────────┤
│ Biz tags                                   │
│ tag · current segment · remaining · alerts │
├────────────────────────────────────────────┤
│ Recent failures / audit                    │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 加载 worker lease、时钟回拨、segment 剩余量和错误摘要。
2. 某个 bizTag segment 低水位时显示告警。
3. 发号失败只展示错误类别和 trace，不展示可预测内部序列细节。
4. 所有配置变更必须通过 Ops 审计流程，不在诊断页直接修改。

## 未来页面约束

- 诊断页只能内部可见。
- 不展示可用于推断业务量或连续 ID 的敏感细节，除非用户具备运维权限。
- 不能让业务服务页面直接依赖 ID Generator 页面判断可用性；服务调用方应通过运行时健康和 fallback 策略处理。
