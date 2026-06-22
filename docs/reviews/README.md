# Review 规则

本目录存放 review 规则、完成标准和正式 review 证据。

完成门槛见 `docs/reviews/done-definition.md`。本文件负责 review 流程、记录格式和归档位置。

## 适用范围

以下场景需要正式 review：

- 公开 HTTP contract、错误码、分页、排序、过滤、事件 payload 或 typed client contract 变化。
- database migration、数据归属、外部公开 ID、唯一约束或不可逆数据变化。
- runtime、worker、consumer、goroutine、重试、幂等、事务、outbox / inbox / ledger 和跨资源一致性变化。
- 跨服务边界、共享库、服务目录结构、脚本检查、AGENTS 路由或长期文档事实源变化。
- 安全敏感面，例如认证、授权、文件上传、外部 URL、密钥、审计日志和用户输入解析。

低风险文档索引、注释、目录登记可以不写正式 review 记录，但仍要保留验证证据。

## 记录位置

按主要风险面创建子目录，例如：

- `backend/`
- `architecture/`
- `security/`
- `general/`

命名建议：

```text
docs/reviews/<category>/YYYY-MM-DD-<scope>-review-<topic>.md
```

示例：

```text
docs/reviews/backend/2026-06-22-upload-http-review-contract.md
docs/reviews/architecture/2026-06-22-service-boundary-review-user.md
```

## 记录内容

正式 review 记录至少包含：

- Review 对象：commit、diff source、文件列表或任务切片。
- 分类判断：低风险、非琐碎、结构性或高风险。
- Gate verdict：`pass`、`pass_with_minor_issues`、`blocked`。
- Findings：按 `Blocker`、`Major`、`Minor`、`Note` 排序。
- Material findings：交付前必须修复的项。
- 验证证据：实际执行的命令和结果。
- Required re-validation：修复后必须重新跑的命令或场景。
- Residual risk：假设、未覆盖场景或明确延期项。
- 技术债状态：是否触达已有 debt；是否已收口；若延期，链接到 `docs/todos/debt/` 条目。

Finding 分级和不可交付状态以 `docs/reviews/done-definition.md` 为准。

## Review 要求

- Findings 先于总结，避免把风险藏在段落末尾。
- 每条 material finding 必须说明影响、证据和期望修复方向。
- 不把风格偏好升级成 blocker；只有会影响 correctness、contract、安全、运行、数据一致性、测试有效性或维护边界的事项才阻塞。
- 不把已确认的长期规则只留在 review 记录里；确认后的事实必须提升到 `docs/architecture/`、`docs/contracts/`、`docs/migration/` 或其他事实源文档。
- Review 发现测试是为了适配坏实现而放宽时，必须回到 owner、contract、实现语义或 fixture 归属排查，不直接接受测试变绿。
