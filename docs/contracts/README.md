# Contract 变更规则

本文件定义 `zhicore-go` 中跨服务 contract 的变更方式。

## 范围

Contract 包括：

- `libs/contracts/clients/<provider-service>/` 下的同步 client contract。
- `libs/contracts/events/<domain>/` 下的事件 payload contract。
- `services/<service>/api/http` 下描述外部可见行为的 HTTP API schema。

服务私有 DTO、领域模型、数据库实体、仓储过滤条件、内部 command/query struct 不属于 contract。

## 专题事实源

本文件只负责 contract 治理规则和变更流程；具体协议规则按专题拆分：

| 专题 | 文档 | 负责内容 |
| --- | --- | --- |
| HTTP | `docs/contracts/http.md` | path、method、header、envelope、版本化和服务级 HTTP schema 放置规则。 |
| API 演进 | `docs/contracts/api-evolution.md` | 已发布 HTTP API 的破坏性变更、废弃和下线流程。 |
| API 设计文档 | `docs/contracts/api-design-documentation.md` | API 背后设计、HTTP contract、endpoint 文档和实现追踪的分层结构。 |
| HTTP schema 模板 | `docs/contracts/http-schema-template.md` | `services/<service>/api/http/` 下服务级 schema 和 endpoint 文档格式。 |
| 错误 | `docs/contracts/errors.md` | 对外错误响应、公开错误码、HTTP status 映射和字段级校验错误形态。 |
| 错误码 | `docs/contracts/error-codes.md` | Go 对外 `body.code` 的项目级错误码表。 |
| 通用数据类型 | `docs/contracts/data-types.md` | 时间、ID、枚举、空值、数字、布尔和 JSON 字段命名规则。 |
| 分页 | `docs/contracts/pagination.md` | page/cursor 分页、排序、过滤和返回形态。 |
| 事件 | `docs/contracts/events.md` | RabbitMQ exchange/routing key、事件 envelope、outbox、幂等和事件兼容性。 |

Go 服务内部错误分层不属于对外 contract，见 `docs/architecture/error-handling.md`。

## 外部 API 契约基线

Go 服务的对外 contract 优先由服务级 HTTP schema 和 provider-owned typed client 定义。前端暂时不修改，未登记 Go-first API reset 的服务在替换既有实现时必须让现有前端和已知调用方无需改造即可继续工作；Java 只作为核对已发布行为的参考来源之一。

当前 Go-first API reset：

- `zhicore-content`：`services/zhicore-content/api/http/` 是新的 Content HTTP contract 事实源；Java 只作为业务能力参考，不作为 path、字段或响应约束。

已发布外部 contract 必须保持的内容包括：

- 外部 API 路径、HTTP method、query/path/body 参数。
- 响应封装、状态码、错误码和错误信息语义。
- 字段名、字段类型、必填/可选语义、分页和排序语义。
- 认证、授权、幂等、可见性和权限失败行为。

Gateway 可以在部署层把路由指向对应 Go 服务。未登记的 API 形态变化不得传递给前端；Go-first API reset 服务以服务级 HTTP schema 作为新前端和 consumer contract。当前目标不规划 Java/Go 运行时并存。

如果某个已发布接口设计确实需要重做，必须作为独立的 API 演进任务处理：先新增兼容入口或版本化入口，保留旧入口，等前端和所有调用方明确切换完成后，再在独立清理变更中删除旧入口。若服务整体已登记为 Go-first API reset，则按该服务 HTTP schema 执行，不要求保留旧外部形态。

## 归属

Provider 拥有 contract。

例子：

- Content 提供的查询 DTO 和 typed client 放在 `libs/contracts/clients/content/`。
- Auth 提供的认证主体、账号状态和角色 DTO / typed client 放在 `libs/contracts/clients/auth/`。
- User 提供的用户资料 DTO 和 typed client 放在 `libs/contracts/clients/user/`。
- Content 的领域事件放在 `libs/contracts/events/content/`。

Consumer 可以依赖 contract，但不能在自己的服务里重新定义 provider 的数据模型。

## 变更分类

修改 contract 前，必须先判断变更类型。

### 兼容变更

不破坏现有 provider 或 consumer 的变更，可以原地修改。

允许的例子：

- 增加可选响应字段。
- 增加 nullable 字段，并且零值语义安全。
- 增加新 endpoint 或 client 方法，不删除旧方法。
- 增加新事件类型。
- 增加旧 consumer 可以忽略的可选事件字段。

### 破坏性变更

破坏性变更必须版本化并分阶段发布。

破坏性例子：

- 重命名、删除或改变字段含义。
- 把必填字段改成不同类型。
- 改变分页、排序、过滤、可见性、授权或幂等语义。
- 要求现有前端为后端内部重写修改路径、参数、响应解析或错误处理。
- 复用同一个事件名但改变语义。
- 删除仍被任何 consumer 使用的 endpoint、client 方法或事件字段。

## 必需变更流程

1. 确认 provider 和所有已知 consumer。
2. 阅读 `docs/architecture/service-boundaries.md`。
3. 判断变更是兼容还是破坏性。
4. 更新 `libs/contracts/...` 或 `services/<service>/api/http` 中 provider 拥有的 contract。
5. 在最小归属边界增加或更新 contract test。
6. 更新 provider 服务实现。
7. 只有在 provider 兼容路径存在后，才更新 consumer。
8. 当归属、语义或发布行为变化时，更新文档。
9. 先运行最窄相关服务测试，再运行 `make check`。

## 破坏性变更流程

不要原地破坏 consumer。

使用以下模式之一：

- 增加 `v2` DTO、client 方法、endpoint 或事件类型。
- 增加新字段，并在切换窗口保留旧字段。
- 增加新 endpoint，同时保留旧 endpoint 直到所有 consumer 切换完成。
- 增加新事件类型，同时让旧 consumer 继续接收旧事件。
- 对外 HTTP API 需要重构时，新增版本化或并行 endpoint；旧外部形态 endpoint 在前端和 consumer 切换完成前必须继续可用。已登记为 Go-first API reset 的服务按服务级 HTTP schema 执行，不要求保留旧外部形态。

切换顺序：

1. 增加新 contract，同时保留旧 contract。
2. 部署或合并 provider 的兼容实现。
3. 将 consumer 切换到新 contract。
4. 证明旧 contract 已无使用方。
5. 在独立清理变更中删除旧 contract。

## 事件 contract 规则

- 不要原地改变已有事件类型的语义。
- 语义变化时，优先增加新事件类型或显式版本。
- 事件 payload 应包含 consumer 需要的稳定事实，不包含 provider 私有持久化细节。
- Consumer 必须容忍未知字段。
- 新增字段默认必须可选，除非所有 consumer 在同一个受控变更中同步更新。

## Facade contract 规则

Facade 路由可以暴露 consumer 友好的形态，但不拥有 provider 数据。是否提供 facade 由对应服务设计决定，不能因为产品 URL 看起来方便就默认新增。

例如，管理端可以暴露面向后台操作的 facade 路由，但真实 mutation 必须委托给拥有该资源的 provider 服务。Facade 数据和权威查询仍属于 provider，facade 只能做浅层参数/返回转换。

当前 `zhicore-content` 明确不提供 User 文章 facade；用户主页文章列表直接调用 Content 作者过滤接口，例如 `GET /api/v1/posts?authorId={authorId}&limit=20`。

如果 facade 形态和 provider 形态不同，必须在 facade 边界记录浅层转换规则。

## 禁止事项

- 不要导入另一个服务的 `internal` 包。
- 不要为了绕过 `libs/contracts` 而把 provider DTO 复制到 consumer 服务。
- 不要在某个模型真正成为跨服务 contract 前，把服务私有模型提升到 `libs/contracts`。
- 不要在引入替代 contract 的同一个变更中删除旧 contract，除非能在同一个原子变更里证明所有 consumer 都已切换。
