# 完成标准

本文件定义 `zhicore-go` 的交付完成门槛。完成不是“代码写完”，而是改动、文档、验证和 review 风险都达到可交付状态。

## 基本原则

- 所有改动必须有验证证据；没有执行的命令不能写成已通过。
- 行为、contract、migration、runtime、worker、事务、幂等、权限、分页和跨服务边界改动必须有测试或明确的手动验证记录。
- 文档、contract、脚本、测试和代码必须同步；不能让 `AGENTS.md`、`docs/README.md`、事实源文档和机械检查互相漂移。
- 已知 blocker 和 major finding 不能靠技术债登记绕过；只有确实不在本次 touched surface，或收口会扩大任务边界时，才登记到 `docs/todos/debt/`。
- 如果本次改动触达已有技术债所在 surface，默认必须在本次收口该债务；无法收口时先拆任务或回到方案阶段。

## Review 触发条件

以下改动在交付前必须做正式 review，并按 `docs/reviews/README.md` 记录证据：

- 修改公开 HTTP contract、错误码、分页、排序、过滤、字段序列化、事件 payload 或 typed client contract。
- 新增或修改 database migration、数据归属、唯一约束、外部公开 ID、回填或不可逆 down migration。
- 修改 runtime 启动、配置、context 传播、健康检查、优雅停机、超时、重试、熔断、幂等、worker / consumer / goroutine 生命周期。
- 修改事务边界、outbox / inbox / ledger、跨资源一致性、缓存失效、分布式锁或重复提交处理。
- 修改跨服务边界、服务职责、共享 `libs/*` contract / kit 原语或目录结构。
- 修改安全敏感面，例如认证、授权、用户输入解析、文件上传、对象存储、外部 URL、密钥和审计日志。
- 单次 diff 跨多个服务、多个职责边界，或新增/修改超过 5 个非文档文件。

纯文档索引、注释、目录登记和脚手架占位等低风险改动可以不写正式 review 记录，但仍要运行对应结构检查并在交付说明中写出验证证据。

## Finding 分级

| 等级 | 含义 | 交付要求 |
| --- | --- | --- |
| Blocker | 会导致错误行为、安全风险、数据损坏、契约破坏、无法运行、无法回滚或违反项目硬规则 | 必须修复并重新验证，不能交付 |
| Major | 有明确回归风险、维护风险、测试缺口、owner 未收口或高风险场景未覆盖 | 默认修复；如延期，必须说明不在 touched surface 或重新拆任务 |
| Minor | 不阻塞交付，但会增加局部维护成本、可读性成本或后续小风险 | 可修复或记录为非阻塞建议 |
| Note | 背景说明、可选改进或后续观察点 | 不阻塞交付 |

Review 记录必须先列 finding，再写总结。没有 material finding 时也要明确写“未发现 blocker / major finding”，并列出剩余验证缺口或假设。

## 验证证据

交付说明、review 记录或提交前说明必须列出实际执行的命令和结果。

常见证据：

```bash
bash scripts/check-structure.sh
make test-size
make check
cd services/<service> && go test ./path/...
cd libs/<module> && go test ./...
```

要求：

- 只跑了窄测试时，说明为什么足够，以及覆盖哪个行为。
- 没有新增测试时，按 `docs/architecture/testing.md` 说明原因。
- 无法自动化验证时，写出手动验证输入、步骤和观察结果。
- 修改并发、worker、consumer、缓存或共享状态时，说明是否需要 `go test -race`；不跑时写明原因。
- 修改 migration 时，说明 `up` / `down 1` 验证情况；不可逆 migration 说明人工确认点。

## 不可交付状态

出现以下任一情况，不得声称完成：

- 相关测试、结构检查、脚本检查或手动验证失败。
- 公开 contract、schema、错误码、事件、API 文档或 AGENTS 路由没有同步。
- 新增高风险行为但没有测试、回归用例或明确的手动验证记录。
- Review 存在 blocker 或未处理 major finding。
- 通过放宽测试断言、修改 fixture / mock 或删除测试来迁就错误实现。
- 把已知 touched surface 技术债留成“后续再说”，但没有明确退出条件和风险边界。
- 运行期必需能力缺失，例如配置校验、timeout、context 传播、幂等、worker 停机或健康检查，却没有记录为阻塞或技术债。

## 技术债登记

只有满足以下条件时，才能把问题登记到 `docs/todos/debt/`：

- 问题真实存在，且当前任务不触达该 surface，或收口会明显扩大任务边界。
- 已写清影响、负责人、来源和退出条件。
- 不影响当前改动的 correctness、contract、安全、数据一致性或运行可恢复性。

技术债不是 review finding 的自动豁免。review 发现的 blocker / major 必须先判断是否属于当前 touched surface；属于则修，不属于才考虑登记。
