# 技术债：统一 HTTP payload 文件组织

状态：已处理
优先级：中
负责人：未分配
来源：2026-07-06 Content 发布生命周期切片中确立 `api/http/payloads.go` 标准后的跨服务检查。

## 影响

部分已实现服务曾把命名的 JSON request / response payload 类型放在 `handler.go` 中，导致 handler 同时承担路由流程、参数校验、application 映射和 payload 类型归属，后续新增 endpoint 时容易继续膨胀。

当前检查结果：

- `zhicore-comment`：已移入 `services/zhicore-comment/api/http/payloads.go`。
- `zhicore-file`：已移入 `services/zhicore-file/api/http/payloads.go`。
- `zhicore-user`：已移入 `services/zhicore-user/api/http/payloads.go`。
- `zhicore-auth`：当前主要使用 handler 函数内匿名 decode struct，未出现命名 `Req` / `Resp` 类型；但 `handler.go` 已较大，后续新增命名 payload 时应直接放入 `payloads.go` 或 `<family>_payloads.go`。
- `zhicore-content`：已使用 `api/http/payloads.go`，符合新标准。

## 处理记录

- 2026-07-06：完成 `zhicore-comment`、`zhicore-file`、`zhicore-user` 命名 HTTP payload 类型搬移，并运行对应服务 `go test ./api/http -count=1`。

## 退出条件

- 已实现 HTTP 服务的命名 request / response payload 类型移入 `api/http/payloads.go` 或按 API family 拆分的 `<family>_payloads.go`。
- `handler.go` 只保留路由注册、handler 流程、参数校验调用、application command/query 映射和必要 helper。
- 每个迁移服务运行对应 `go test ./api/http -count=1`。
- 保留少量函数内匿名 decode struct 时，确认它只服务单个 handler 且没有复用或测试引用需求。

## 备注

标准事实源：`docs/architecture/go-service-design.md` 的 HTTP 入站层文件组织规则。
