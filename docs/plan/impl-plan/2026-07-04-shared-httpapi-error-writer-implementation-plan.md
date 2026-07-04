# 共享 HTTP 错误写入能力实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 让 `libs/kit/httpapi` 支持 HTTP status 与响应 body 业务错误码分离，解除 Comment 等服务不能直接使用 `WriteError` 的阻塞。

**架构：** `libs/kit/httpapi` 只提供通用 envelope writer，不包含任何服务业务错误映射。各服务在 `api/http` 层把 application error 映射成 HTTP status、body `code` 和 `message`。

**技术栈：** Go 1.22、标准库 `net/http`、`encoding/json`、`httptest`。

---

## 背景依据

- 当前实现：`libs/kit/httpapi/response.go`。
- 触发原因：`services/zhicore-comment/api/http/README.md` 明确要求 `5001` 等业务码不能被写成 HTTP status。
- Contract 规则：`docs/contracts/errors.md`、`docs/contracts/error-codes.md`。

## 文件结构

- 修改：`libs/kit/httpapi/response.go`
  - 新增 `WriteErrorCode`、`ErrorDetail`、`WithTraceID`、`WithDetails`。
  - 保留 `WriteError` 向后兼容现有调用。
- 新增：`libs/kit/httpapi/response_test.go`
  - 覆盖成功 envelope、旧 `WriteError` 行为、业务错误码写入、traceId 和 details。

## 任务 1：业务错误码 writer

**测试立场：** TDD - 公开响应 envelope 是跨服务 contract。

- [x] **步骤 1：编写失败测试**

  测试应断言 `WriteErrorCode(rr, http.StatusNotFound, 5001, "Comment not found")` 时：

  - HTTP status 是 `404`
  - body `code` 是 `5001`
  - body `message` 是 `Comment not found`
  - `timestamp` 非零

  运行：`cd libs/kit && go test ./httpapi -run TestWriteErrorCodeUsesBusinessCode`

  预期：失败，因为函数尚不存在。

- [x] **步骤 2：实现最小 API**

  在 `response.go` 中新增：

  ```go
  type ErrorDetail struct {
      Path       string `json:"path,omitempty"`
      Code       string `json:"code,omitempty"`
      MessageKey string `json:"messageKey,omitempty"`
  }

  type ErrorOption func(*Response)

  func WithTraceID(traceID string) ErrorOption
  func WithDetails(details []ErrorDetail) ErrorOption
  func WriteErrorCode(w http.ResponseWriter, status int, code int, message string, opts ...ErrorOption)
  ```

  `WriteError` 改为调用 `WriteErrorCode(w, status, status, message)`。

- [x] **步骤 3：补充 option 测试**

  覆盖 `traceId` 和路径级 `details` 会进入响应 body；空 traceId 和空 details 不输出字段。

  运行：`cd libs/kit && go test ./httpapi`

  预期：通过。

- [x] **步骤 4：运行共享库测试**

  运行：`cd libs/kit && go test ./...`

  预期：通过。

- [x] **步骤 5：按提交规则提交**

## 架构适配评估

- 只修改 `libs/kit/httpapi` 技术原语，不引入服务错误码常量。
- 不破坏 File 服务现有 `WriteError` 调用。
- 后续服务可以在各自 `api/http` 层维护错误映射，不需要复制 envelope writer。
