# 安全与权限规范

本文件定义 `zhicore-go` 的认证、授权、身份传播、审计、敏感输入和安全测试基线。

## 基本原则

- 安全规则按 owner 收口。Auth 负责账号、凭证、角色和 token 事实，Gateway 负责入口认证和基础拦截，User 负责用户资料和关系事实，业务服务负责资源归属和业务权限。
- Gateway 不判断资源归属权限，不复制业务模型，不把下游响应转换成另一套业务语义。
- 外部 HTTP 请求的 JWT 只在 Gateway 解析和校验。业务服务不解析客户端 `Authorization` 作为身份来源。
- 权限判断默认在 application 层完成；handler 只负责把认证上下文映射成 application input，repository 不做鉴权。
- 用户输入默认不可信。所有外部输入必须经过绑定、校验和明确的错误映射后才能进入 use case。
- 安全失败必须返回稳定公开错误码，不暴露 token、密码、底层 driver、外部 provider 或内部拓扑细节。
- 审计日志是业务事实，不是普通运行日志。需要审计的行为必须写入归属服务 schema 或明确的审计 owner。

相关文档：

- 服务边界和数据 owner：`docs/architecture/service-boundaries.md`
- HTTP 契约和 header：`docs/contracts/http.md`
- 对外错误响应和错误码：`docs/contracts/errors.md`、`docs/contracts/error-codes.md`
- 配置、密钥和环境变量：`docs/architecture/configuration.md`
- 日志脱敏和观测字段：`docs/architecture/observability.md`
- 运行期超时、重试和停机：`docs/architecture/runtime-operations.md`

## 身份认证

### Auth

`zhicore-auth` 拥有：

- 账号身份、登录凭证、角色、启用 / 禁用状态。
- JWT 签发、refresh、失效语义和账号状态变更后的 token 处理规则。
- 密码校验、密码 hash、凭证更新和登录失败错误映射。

规则：

- 密码只存 hash，不记录明文、hash 原文或可逆摘要。
- 登录失败对外返回稳定错误码，不区分“用户不存在”和“密码错误”的内部细节，除非服务级 contract 已明确登记历史差异。
- 禁用账号、角色变更、密码重置和 token 全量失效必须由 Auth 提供 command contract；其他服务不能自行修改账号身份状态。
- refresh token、临时授权 token 或高风险操作 token 不作为资源 ID 使用；编码和加密边界见 `docs/architecture/id-strategy.md`。

### User

`zhicore-user` 拥有用户资料、关注、拉黑、陌生人消息设置和签到事实。

规则：

- User 不保存密码 hash，不签发 JWT，不维护 refresh token 白名单，不写角色事实。
- User 需要账号可用性时通过 Auth contract 或 Auth 事件获得派生事实，不能跨库读取 Auth 表。
- User 对资料、关系和签到的业务权限仍在自己的 application 层判断。

### Gateway

`zhicore-gateway` 负责：

- 校验 JWT 签名、过期时间、issuer / audience 等基础 claims。
- 查询 token 黑名单或认证缓存。
- 为下游服务注入受信任的身份上下文 header。
- 处理匿名 endpoint 和登录 endpoint 的认证跳过规则。
- 按认证结果、路由规则和服务目标分流请求到下游服务。

规则：

- Gateway 只能信任自己校验后的 token 结果，不能把客户端传入的 `X-User-Id`、`X-User-Roles` 或类似身份 header 直接透传为可信身份。
- Gateway 转发给下游的身份 header 必须先清理客户端同名 header，再由 Gateway 重新写入。
- Gateway 是普通业务 HTTP 请求的唯一 JWT 校验点。下游业务服务不得为了直连、调试或兼容再从 `Authorization` 解析 JWT。
- Gateway 转发业务请求时不要求下游依赖原始 `Authorization`。如果某个 endpoint 确实需要接收原始凭证，必须在服务级 HTTP contract 中显式登记，并且不得把它当作当前用户身份来源。
- 认证失败使用 `docs/contracts/error-codes.md` 的认证授权错误码，并保持 ZhiCore 响应 envelope。
- Gateway 可以做入口级限流、CORS、body size、请求 ID 和可信 caller header 注入，但不做“文章是否属于当前用户”这类资源权限判断。

## 身份上下文传播

Gateway 注入给下游服务的身份上下文使用内部 header。当前固定名称：

| Header | 含义 | 要求 |
| --- | --- | --- |
| `X-User-Id` | 当前登录用户 ID | 登录态 endpoint 必填；下游按内部 `UserID` 解析。 |
| `X-User-Name` | 当前用户名或展示名 | 可选，仅用于兼容或轻量日志字段，不作为权限事实。 |
| `X-User-Roles` | 稳定角色集合 | 可选；多个角色用逗号分隔。 |
| `X-Request-Id` | 请求关联 ID | 有上游值则沿用，否则 Gateway 生成。 |
| `X-Trace-Id` | 链路关联 ID | 有上游值则沿用，否则 Gateway 生成或派生。 |

可选 `authSubject` 或 `tokenVersion` 只能在 Auth / Gateway 需要校验 token 版本时引入。新增身份 header 必须先更新 `docs/contracts/http.md` 和本文件。

服务间同步调用额外使用 `X-Caller-Service` 和 `X-Caller-Operation` 表达调用方服务和调用方稳定操作名。它们只用于 provider 侧限流、审计和观测，不表示当前用户身份。外部请求中的同名 header 必须由 Gateway 清理；服务间 typed client adapter 只能从本服务配置和调用点 operation 常量生成这两个 header，不能直接透传客户端输入。

规则：

- handler 把 header / middleware context 映射成明确 application input，例如 `Actor`、`AuthContext` 或 `Principal`。
- 下游服务的 auth middleware 只做可信 header 的提取、格式校验和 `context.Context` 注入；不得解析 JWT、查询 token 黑名单或自行决定 token 是否有效。
- 缺少必需身份上下文的登录态 endpoint 返回 `LOGIN_REQUIRED`；身份 header 格式非法按认证失败处理，不降级解析 `Authorization`。
- application 层不直接读 HTTP header，不依赖框架上下文查找权限事实。
- 服务间 typed client 需要传播 `X-Request-Id` / `X-Trace-Id`，并在 provider contract 要求时写入 `X-Caller-Service` / `X-Caller-Operation`；是否传播用户身份必须由 provider contract 明确。
- 服务间同步调用如果需要代表当前用户执行操作，consumer 必须显式把 `Actor`/`AuthContext` 映射为 provider contract 允许的身份 header；不能隐式透传客户端 header。
- 异步事件可以携带 `requestId` / `traceId` 做观测关联，但不能用这些字段做权限、幂等或业务分支判断。

## 授权与资源权限

权限判断分三类：

| 类型 | Owner | 示例 |
| --- | --- | --- |
| 登录态和角色 | Gateway / Auth | 未登录、token 过期、管理员角色 |
| 资源归属和可见性 | 资源归属服务 application | 文章作者才能编辑草稿、评论作者才能删除评论 |
| 跨服务关系事实 | 事实归属服务 contract | 拉黑关系、陌生人私信设置、账号禁用状态 |

规则：

- 业务服务必须在自己的 application use case 中判断资源归属、可见性、状态机和业务权限。
- repository 只负责数据查询、约束和错误翻译，不根据当前用户决定“是否允许”。
- handler 可以拒绝明显缺失认证上下文的请求，但不能把完整业务权限写在 handler 中。
- 跨服务权限事实必须通过归属服务 contract 查询或消费归属服务事件维护快照，不能跨库 join。
- 管理员权限不是业务权限的万能绕过。Admin facade 仍必须调用归属服务 command contract，让归属服务执行自己的状态和一致性校验。

## Admin 审计

`zhicore-admin` 拥有举报处理流程、审核审计和管理端聚合查询。

管理操作流程：

1. 校验管理员身份和权限。
2. 调用归属服务 command contract。
3. 根据归属服务结果记录审计事实。
4. 返回管理端兼容响应。

审计记录至少包含：

- 操作者 ID、角色或管理身份。
- 目标类型和目标 ID。
- 操作类型、原因、结果和发生时间。
- 失败原因分类；不能记录 token、密码、完整请求 body 或底层敏感错误。

规则：

- 归属服务操作失败时，Admin 不能写成功审计。
- 高风险管理操作需要保留失败审计或运行日志，便于追溯误操作和外部依赖失败。
- 审计日志使用 UTC 业务时间，时间规则见 `docs/contracts/data-types.md`。

## 上传、外部 URL 和文件安全

`zhicore-upload` 是统一文件入口，但不拥有业务实体和文件关联关系。

规则：

- 上传必须校验 multipart 字段名、文件大小、MIME type、扩展名和业务允许的访问级别。
- 文件类型不能只信任客户端 `Content-Type`；至少结合扩展名和文件头判断。
- 临时文件必须在成功、失败和 panic recovery 路径中清理。
- `GET /api/v1/upload/file/{fileId}/url` 返回的 URL 不能泄漏对象存储签名参数到日志。
- 删除文件需要考虑外部 File Service 失败；业务服务删除实体成功但文件删除失败时必须有补偿或技术债记录。
- 服务端发起外部 URL 请求时必须防 SSRF：限制 scheme、host、端口、重定向和内网地址段；没有明确白名单时不要拉取用户提交的任意 URL。

## 敏感信息和密钥

禁止进入 Git、日志、metrics label、trace attribute 或错误响应：

- 密码、验证码、JWT、refresh token、session、cookie、Authorization header。
- private key、access key、secret key、生产 DSN、对象存储签名 URL。
- 完整请求 body、完整文件 URL、连接串密码和云厂商凭证。

规则：

- 密钥和凭证只能通过配置或 secret 注入；不允许代码默认值。
- 配置模板只能出现占位符或本地开发安全默认值。
- 日志字段、脱敏和采样规则见 `docs/architecture/observability.md`。
- 安全错误的对外 message 使用公开、稳定、可本地化的业务说明，不包含内部 provider、SQL、stack trace 或文件路径。

## `libs/kit/auth` 边界

允许放入：

- JWT 解析、签名校验、claims 基础结构和错误分类原语，仅供 Auth 签发/解析 refresh token 和 Gateway 校验 access token 使用。
- 认证上下文类型、context helper 和可信 header 提取 middleware 的可复用小工具。
- 密码 hash / verify 的薄封装，前提是参数由 Auth owner 明确配置。

禁止放入：

- 服务私有角色模型、资源权限规则、Admin 操作清单。
- 直接访问 Auth / User 数据库、Redis 黑名单或业务 repository 的逻辑。
- 让业务服务从客户端 `Authorization` 解析 JWT 的通用 fallback。
- 会替代业务 owner 决策的通用 `CanEdit`、`IsOwner`、`IsAdminBypass`。

共享库只提供认证原语；Auth、Gateway、User 和业务服务分别拥有自己的安全决策。

## 测试和 review 门槛

以下改动属于高风险面：

- 登录、注册、refresh、JWT 校验、token 黑名单、账号禁用和角色变更。
- 权限、资源可见性、Admin 管理命令、审计日志。
- 上传校验、文件删除、外部 URL 拉取、对象存储签名 URL。
- 密钥加载、脱敏、认证 header 传播、跨服务身份上下文。

要求：

- 优先新增 focused test 或回归测试；是否 test-first 按 `docs/architecture/testing.md` 的风险分级执行。
- Handler test 覆盖认证缺失、权限失败、公开错误码和响应 envelope。
- Gateway handler / middleware test 必须覆盖客户端伪造 `X-User-*` 被清理或覆盖。
- 业务服务 handler test 必须覆盖：缺少 Gateway 注入身份时拒绝、存在 `Authorization` 但缺少可信身份 header 时不解析 JWT。
- Application test 覆盖资源归属、角色、状态机、审计结果和跨服务 contract 结果。
- Upload / 外部 URL 相关测试必须覆盖非法类型、超限、清理路径和 SSRF 防护边界。
- 修改安全敏感面时，应按 `docs/reviews/done-definition.md` 判断是否需要正式 review。
- 仅更新本文件、索引或结构检查时，运行 `bash scripts/check-structure.sh`；涉及代码时运行最窄相关 `go test`，必要时运行 `make check`。
