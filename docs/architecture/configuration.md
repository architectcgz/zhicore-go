# 配置与环境变量规范

本文件定义 `zhicore-go` 的服务配置、环境变量、配置模板、密钥处理和配置加载边界。

## 适用范围

- `services/<service>/cmd/server` 进程入口。
- `services/<service>/internal/<domain>/runtime` 运行时组装。
- `services/<service>/configs/` 本地配置模板。
- `libs/kit/config` 跨服务配置加载和校验原语。
- PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储、HTTP client、日志、追踪、健康检查和 worker 运行参数。

运行时启动、健康检查、超时、重试、幂等和停机规则见 `docs/architecture/runtime-operations.md`。日志字段、metrics、trace 和脱敏规则见 `docs/architecture/observability.md`。

## 配置来源优先级

配置按以下优先级合并，越靠前优先级越高：

1. 测试中显式传入的配置值。
2. 启动参数，只有确实需要本地覆盖时使用。
3. 环境变量。
4. 本地开发配置文件模板，例如 `services/<service>/configs/local.example.*`。
5. 代码内安全默认值。

规则：

- 生产部署默认通过环境变量、ConfigMap 或 Secret 注入，不依赖 Nacos。
- 本地配置文件只服务开发和示例，不作为生产事实源。
- 代码默认值只允许用于安全、非敏感、可预测的参数，例如本地监听地址、日志格式、默认 timeout。
- 密钥、密码、token、私钥、JWT secret、对象存储凭证、生产 DSN 不允许有代码默认值。
- 服务启动前必须完成配置校验；必填配置缺失、格式错误或组合不合法时直接 fail fast。

## 配置 owner

配置结构按职责放置：

- 服务私有配置结构由对应服务拥有，优先放在 `internal/<domain>/runtime` 或靠近运行时组装的位置。
- `cmd/server` 只负责调用加载、校验、打开运行时依赖和启动服务，不承载业务配置规则。
- application 可以接收已经解析好的业务参数，例如上传大小限制、允许的 MIME 类型、分页上限；它不能读取环境变量或配置文件。
- infrastructure 接收已经解析好的 DSN、timeout、pool size、client option；它不能自行查环境变量补配置。
- `libs/kit/config` 只放跨服务通用原语，例如读取 env、解析 duration / size、必填校验、默认值合并、敏感字段脱敏和错误聚合。
- `libs/kit/config` 不放任何服务专属环境变量名、业务默认值、表名、bucket 名或路由配置。

禁止在 handler、domain、repository、client adapter 或普通构造函数中直接读取环境变量。

## 环境变量命名

服务级环境变量使用统一前缀：

```text
ZHICORE_<SERVICE>_<SECTION>_<NAME>
```

规则：

- `<SERVICE>` 使用服务名去掉 `zhicore-` 前缀后的大写形式，连字符转下划线，例如 `UPLOAD`、`ID_GENERATOR`、`GATEWAY`。
- `<SECTION>` 使用能力或依赖名，例如 `HTTP`、`POSTGRES`、`REDIS`、`RABBITMQ`、`MONGO`、`ES`、`LOG`、`JWT`、`FILE_SERVICE`。
- `<NAME>` 使用具体字段名，例如 `ADDR`、`DSN`、`URL`、`TIMEOUT`、`MAX_OPEN_CONNS`、`SECRET`。
- 跨所有服务共享的部署级变量可以使用 `ZHICORE_<NAME>`，例如 `ZHICORE_ENV`，但不要把服务私有配置提升成全局变量。
- 布尔值使用 `true` / `false`。
- duration 使用 Go duration 字符串，例如 `500ms`、`2s`、`30s`。
- size 使用显式单位，例如 `10MiB`、`100MB`；不要用无单位整数表达字节大小。
- list 使用逗号分隔并在解析时 trim 空白；如果值本身可能包含逗号，改用配置文件模板或更明确的结构。

示例：

```text
ZHICORE_ENV=local
ZHICORE_UPLOAD_HTTP_ADDR=:8080
ZHICORE_UPLOAD_HTTP_READ_TIMEOUT=5s
ZHICORE_UPLOAD_POSTGRES_DSN=postgres://...
ZHICORE_UPLOAD_REDIS_ADDR=127.0.0.1:6379
ZHICORE_UPLOAD_FILE_SERVICE_BASE_URL=http://127.0.0.1:9000
ZHICORE_GATEWAY_JWT_SECRET=...
```

## 必填、默认值和 optional

每个配置字段必须明确属于以下一类：

| 类型 | 含义 | 处理规则 |
| --- | --- | --- |
| Required | 服务没有它不能正确运行 | 启动前校验；缺失直接失败 |
| Defaulted | 有安全默认值 | 写明默认值和适用环境 |
| Optional | 功能可关闭或依赖可缺省 | 需要明确 `enabled` 或等价开关 |
| Derived | 从其他配置派生 | 派生规则集中在加载/校验层 |

规则：

- 不用“空字符串”隐式表示 optional；可选依赖必须有明确开关或明确的禁用语义。
- 生产敏感项不能 default；本地开发可在 `local.example.*` 写示例值。
- 默认 timeout、连接池大小、最大请求体、上传大小上限、分页上限、retry / circuit breaker / max in-flight 等运行期策略必须在服务文档或配置模板中可见。
- 派生配置不能隐藏失败。例如从 public URL 派生 callback URL 失败时，应返回配置错误。
- 启动日志可以记录配置摘要，但只能记录脱敏后的值和关键非敏感参数。

## 密钥和敏感信息

敏感信息包括但不限于：

- 密码、JWT secret、签名 key、private key、access key、secret key、token。
- 数据库、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储的生产凭证或完整生产 DSN。
- `Authorization` header、cookie、验证码、临时签名 URL。

规则：

- 真实密钥不得提交到仓库、docs、README、测试 fixture、截图或 review 记录。
- 示例值必须明显是假的，例如 `change-me`、`example-secret`、`postgres://user:password@localhost:5432/zhicore`。
- 日志、错误、panic、配置摘要和测试失败输出必须对敏感字段脱敏。
- 含敏感字段的 config struct 不实现直接暴露明文的 `String()`。
- 需要打印配置时使用显式 `Redacted()`、`Summary()` 或等价方法，只输出非敏感字段。

`.gitignore` 必须忽略真实本地配置文件；只允许 `.env.example`、`local.example.*` 或其他明确 example 模板进入仓库。

## `services/<service>/configs/`

`configs/` 用于服务本地配置模板和说明，不是生产配置仓库。

允许：

- `local.example.env`
- `local.example.yaml`
- `.env.example`
- 非敏感的本地开发示例和字段注释。

禁止：

- `local.env`、`.env`、`prod.yaml`、`production.yaml` 等真实环境配置。
- 任何真实密码、token、secret、生产地址或生产 DSN。
- 把 Kubernetes Secret、Helm values 或云厂商凭证原样放入服务仓库。

服务实现进入可运行阶段时，README 或 `configs/` 模板必须列出本地运行所需的最小配置。

## 运行依赖配置

常见运行依赖配置必须显式：

- HTTP server：监听地址、read/write/header/idle timeout、shutdown timeout、最大请求体。
- PostgreSQL：DSN、最大连接数、最大空闲连接数、连接生命周期、query timeout；连接时区必须显式使用 UTC。
- Redis：地址、DB、dial/read/write timeout、pool size。
- RabbitMQ：URL、exchange、queue、routing key、publish confirm timeout、consumer shutdown timeout。
- MongoDB / Elasticsearch：URL、认证信息、request timeout、index / collection 名称。
- HTTP client：base URL、timeout、retry policy、可重试错误分类、circuit breaker 参数、max in-flight、降级策略标识。
- 日志：level、format、service、env；字段语义见 `docs/architecture/observability.md`。
- worker / consumer：并发数、batch size、poll interval、lease timeout、retry / dead-letter 策略、下游调用 resilience policy。

这些值可以有本地开发默认值，但生产依赖地址和凭证必须来自环境注入。

## 测试和验证

修改配置加载、默认值、必填项或环境变量命名时：

- 优先补配置加载单元测试，覆盖 required、defaulted、optional、非法格式和敏感字段脱敏。
- 涉及服务启动流程时，增加最窄相关 runtime / system 验证，确保缺失必填配置会启动失败。
- 涉及 `libs/kit/config` 时，运行 `cd libs/kit && go test ./...`。
- 涉及单个服务配置时，运行该服务最窄相关 `go test`。
- 改文档、索引、`.gitignore` 或结构检查时，运行 `bash scripts/check-structure.sh`；交付前按 `docs/reviews/quality-gates.md` 选择是否运行 `make check`。

不要只通过手动设置本机环境变量证明配置正确；需要记录命令、输入和观察结果。
