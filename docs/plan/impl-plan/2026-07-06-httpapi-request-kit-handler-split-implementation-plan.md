# HTTP request kit 与 handler 拆分实现计划

> **给 agentic workers：** 本计划以用户提供的“handler request kit 与拆分计划”为事实输入。实现时先守住 `libs/kit/httpapi` 的窄边界，再迁移服务本地重复 helper，最后拆分超过 500 行的 handler。每个 checklist 完成后立即更新；如需提交，提交前必须使用 @committing-changes。

**目标：** 抽取跨服务稳定的 HTTP request 解析原语，并在不改变 HTTP path、response envelope、错误码、分页默认值、权限判断和 cookie/CSRF 语义的前提下，降低 `content`、`auth`、`user` 的 oversized handler 复杂度。

**非目标：**

- 不新增服务错误码。
- 不把业务 actor、roles、cookie/CSRF、application mapper、Gin 专用 helper 提升到 kit。
- 不改变 Auth register/login 的 strict JSON decode contract。
- 不强制拆分 `comment` 和 `file` handler。

## 输入与边界

- `AGENTS.md`
- `docs/architecture/go-service-design.md`
- `docs/architecture/repository-layout.md`
- `docs/architecture/testing.md`
- `docs/contracts/errors.md`
- `docs/architecture/error-handling.md`
- `docs/reviews/quality-gates.md`
- 用户提供的执行计划

## 任务 1：新增 `libs/kit/httpapi` request 原语

**测试立场：** TDD。JSON framing、body limit、query int 解析和时间格式化属于共享 contract 原语。

- [x] 先新增 `libs/kit/httpapi/request_test.go`，覆盖 malformed JSON、trailing JSON、limited body 超限、空 query 默认值、正整数上下限、UTC 时间格式化和 zero time。
- [x] 运行 `cd libs/kit && go test ./httpapi -count=1`，确认新增测试 RED。
- [x] 新增 `libs/kit/httpapi/request.go`，实现 `DecodeJSONBody`、`DecodeJSONBodyLimited`、`ParsePositiveInt`、`FormatRFC3339UTC`。
- [x] 运行 `cd libs/kit && go test ./httpapi -count=1`，确认 GREEN。

## 任务 2：迁移 Content HTTP handler helper 并拆分 oversized 文件

**测试立场：** R1/R2。保持行为不变，依赖现有 handler test 保护；迁移 helper 后运行包测试。

- [x] 用 kit 替换 `decodeJSONBody`、`optionalPositiveIntQuery` 的底层解析和 `formatTime`。
- [x] 保留服务本地 `writeValidationError`、body-too-large 错误码映射、actor/roles、path 参数和 application mapper。
- [x] 拆分 `handler.go`：保留 `Service`、`Handler`、`NewHandler`、`routes`；endpoint handler、错误映射、request helper、response mapper 分窄文件。
- [x] 运行 `cd services/zhicore-content && go test ./api/http -count=1`。

## 任务 3：迁移 User HTTP handler helper 并拆分 oversized 文件

**测试立场：** R1/R2。保持 PATCH nullable/optional contract 不变。

- [x] 用 kit 替换普通 `decodeJSONBody` 和 relationship `limit` 正整数解析。
- [x] 保留 `decodeUpdateProfileBody` 本地实现。
- [x] 拆分 `handler.go`：profile、relationship、internal batch、request helper、errors、mappers 分文件；构造和路由留在 `handler.go`。
- [x] 运行 `cd services/zhicore-user && go test ./api/http -count=1`。

## 任务 4：迁移 Auth helper 并拆分 oversized 文件

**测试立场：** R1/R2。严格 JSON decode、cookie/CSRF 和身份语义保持不变。

- [x] `parsePagination` 改用 `ParsePositiveInt`，保持默认 `page=1`、`size=20`、`maxPageSize=50`。
- [x] `formatTimePtr` 内部使用 `FormatRFC3339UTC`，nil 继续返回 nil。
- [x] 保留 register/login strict JSON decode 本地实现。
- [x] 拆分 cookie/CSRF、identity/request helper、response payload helper、errors 和 endpoint handler family。
- [x] 运行 `cd services/zhicore-auth && go test ./api/http -count=1`。

## 任务 5：迁移 Comment helper

**测试立场：** R1/R2。`handler.go` 当前低于 500 行，只做必要 helper 迁移，避免无收益 churn。

- [x] 用 kit 替换普通 JSON decode、分页正整数解析和 `formatTime`。
- [x] 必要时只抽 `request_helpers.go` / `mappers.go`，不做大拆。
- [x] 运行 `cd services/zhicore-comment && go test ./api/http -count=1`。

## 任务 6：集成验证与 review

- [x] 运行 `python3 scripts/check-test-size.py --root .`。
- [x] 运行 `make check`。
- [x] 完成独立 backend review，并把 review 归档到 `docs/reviews/backend/`。
- [x] 修复 material finding 后重跑受影响验证。
