# HTTP request kit 与 handler 拆分 Review

## Review 对象

- 范围：
  - 新增 `libs/kit/httpapi` request helper：JSON 单值 decode、limited body decode、正整数 query 解析、UTC RFC3339 时间格式化。
  - 拆分 `services/zhicore-content/api/http` oversized handler。
  - 拆分 `services/zhicore-user/api/http` oversized handler，并迁移普通 request helper。
  - 拆分 `services/zhicore-auth/api/http` oversized handler，并迁移 pagination/time helper。
  - 迁移 `services/zhicore-comment/api/http` 普通 JSON decode、分页和时间 helper。
- 实施计划：`docs/plan/impl-plan/2026-07-06-httpapi-request-kit-handler-split-implementation-plan.md`
- 独立 reviewer：`code-reviewer` subagent `019f365b-8cf5-73d1-948c-3a653fcaf48d`

## 分类判断

- 分类：结构性后端重构，触达共享 `libs/kit`、多服务 HTTP handler、用户输入解析和长期架构事实源。
- 触发原因：共享 kit 原语、多服务拆分、输入解析 helper 标准化和新增 contract regression tests。
- Gate verdict：`pass`。

## Findings

### Blocker

未发现 blocker。

### Major

未发现 major finding。

### Minor

- 少量 helper 与新确立的 handler 入出参标准不一致。
  - 证据：`comment` 的 path helper 和 `user` 的 internal caller helper 曾直接写响应。
  - 状态：已修复。
  - 修复：helper 改为返回 `error`，endpoint handler 统一调用 `writeValidationError` 或 `writeMappedError`。
- Auth strict JSON decode 缺少 endpoint 层 contract 覆盖。
  - 证据：register/login 本地 strict decoder 保留 `DisallowUnknownFields()`，但缺少 trailing JSON value 和 unknown field regression tests。
  - 状态：已修复。
  - 修复：新增 register/login strict decode 测试，验证返回 HTTP `400`、body `code=1001`，且 service 未被调用。
- Kit `ParsePositiveInt` 缺少 `value == max` 边界样例。
  - 状态：已修复。
  - 修复：新增 `TestParsePositiveIntAcceptsValueEqualToMax`。

## Material Findings

无未修复 material finding。

## 验证证据

已执行并通过：

```bash
cd libs/kit && go test ./httpapi -count=1
cd services/zhicore-user && go test ./api/http -count=1
cd services/zhicore-comment && go test ./api/http -count=1
cd services/zhicore-auth && go test ./api/http -count=1
python3 scripts/check-test-size.py --root .
make check
```

## Required Re-validation

后续修改 `libs/kit/httpapi` request helper、任一服务 `api/http` request helper、handler 错误映射、pagination 解析或时间序列化时，至少重新执行对应模块的 `go test ./api/http -count=1` 或 `go test ./httpapi -count=1`。

触达共享 kit、多个服务 handler 或文档/计划索引时，交付前重新执行：

```bash
python3 scripts/check-test-size.py --root .
make check
```

## Residual Risk

- Auth register/login 继续保留服务本地 strict JSON decode；本轮没有把 `DisallowUnknownFields()` 提升到 kit，避免静默改变其他服务 request contract。
- `comment` handler 本轮只迁移公共 helper，不做大拆；当前文件低于 500 行，后续只有继续增长时再按 API family 拆分。

## 技术债状态

未新增技术债。review 提出的非阻断 finding 已在本次 touched surface 内收口。
