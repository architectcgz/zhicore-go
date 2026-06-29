# API 演进流程

本文定义 `zhicore-go` 中已发布 HTTP API 的破坏性变更和废弃流程。

## 基本原则

- 已发布 API 默认不破坏兼容性。前端暂时不修改，所有对外兼容性约束以服务级 HTTP contract 为准。
- 破坏性变更必须作为独立演进任务处理，不能混入功能开发。
- 非破坏性变更（新增可选字段、新增 endpoint、放宽校验）不需要走本流程。

## 什么是破坏性变更

| 变更类型 | 是否破坏性 |
|---------|----------|
| 新增 endpoint | 否 |
| 新增响应中的可选字段 | 否 |
| 新增请求中的可选参数（有合理默认值） | 否 |
| 删除或重命名字段 | **是** |
| 改变字段类型或语义 | **是** |
| 删除 endpoint | **是** |
| 改变必填/选填 | **是**（必填变选填：否；选填变必填：是） |
| 改变错误码语义 | **是** |
| 改变 HTTP status 含义 | **是** |
| 缩紧权限（原来无需登录现在需要） | **是** |

## 演进流程

### 第一阶段：评估

1. 在服务级 HTTP contract（`services/<service>/api/http/endpoints/`）中标记受影响的 endpoint 为 `DEPRECATED`，注明废弃原因、新替代方案和计划废弃日期。
2. 在 `docs/reviews/` 或 `docs/todos/` 中创建一个演进任务文档，记录：变更内容、影响范围、前端/consumer 需要做的适配、验证方案。
3. 通知前端和已知 consumer。

### 第二阶段：并存

1. 新行为作为新 endpoint 或带版本参数的扩展实现，旧 endpoint 继续维持原有行为。
2. 新旧并存期默认 ≥ 2 个发版周期（个人项目可视情况缩短，但不得低于 1 个周期）。
3. 在 Gateway route 配置中记录旧 endpoint 的废弃状态，可选地在响应 header 中加 `Deprecation: date` 提示。

### 第三阶段：下线

1. 确认前端和所有 consumer 已切换到新行为（通过访问日志、metrics 或直接沟通确认）。
2. 从 Gateway route 移除旧 endpoint 的路由。
3. 在服务级 HTTP contract 中将该 endpoint 标记为 `REMOVED`，记录移除日期。
4. 后续可选：从服务代码中删除 handler（不影响外部，只影响代码维护）。

## 历史例外

已发布接口中存在使用 HTTP status code 作为 body `code` 的历史形态（`400`、`401`、`404`、`500` 等）。Go 服务承接这类接口时，在服务级 HTTP contract 中标记为**历史兼容例外**，不作为新接口的参考模式。如需改为业务错误码，走本文演进流程。

## 相关文档

- `docs/contracts/http.md`：HTTP 协议层规则
- `docs/contracts/errors.md`：错误响应和错误码规范
- `docs/contracts/events.md`：事件兼容性规则（事件不适用本流程，见该文件的兼容性章节）
