# 服务目录

`services/` 下的每个目录都是独立可构建、可测试、可部署的 Go 服务。

每个服务固定拥有：

- `go.mod`
- `cmd/server/`
- `internal/`
- `api/http/`
- `configs/`
- `migrations/`

HTTP 入站层放在服务根目录的 `api/http/` 下。服务私有代码必须放在自己的 `internal/` 下。其他服务只能通过 `libs/contracts` 中的 contract 或对外 API 访问它，不允许导入另一个服务的 `internal`。
