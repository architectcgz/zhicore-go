# Review 全集检查清单

本文件是仓库根 review 入口，要求 reviewer 先建立“内容全集”视图，再进入具体 finding。正式 review 证据仍写入 `docs/reviews/`，完成标准、finding 分级和验证要求以 `docs/reviews/README.md`、`docs/reviews/done-definition.md`、`docs/reviews/quality-gates.md` 为准。

## 先看全集

开始 review 前先确认完整对象，不只看用户点名的文件：

```bash
git status --short
git diff --stat
git diff --name-status
git diff --check
```

必须把以下内容纳入 review 范围：

- 本次 diff 的全部文件，包括新增、删除、重命名和测试 fixture。
- 受影响服务的 `api/http`、`internal/<domain>/application`、`domain`、`ports`、`infrastructure`、`runtime` 和 `migrations`。
- 受影响的 `libs/contracts`、`libs/kit`、`docs/architecture`、`docs/contracts`、`docs/reviews`、`scripts` 和 `tests`。
- 变更文件的调用方、被调用方、接口实现、测试替身和文档引用。
- 超过 400 行的文件，以及本次变更让文件接近或超过 400 行的文件。
- 架构依赖方向、DDD 模型、代码质量、注释质量、验证证据和文档同步。

## 高风险扫描

review 不得只按 `Req` / `Resp` 字面搜索。协议形状、跨服务 contract、事件 payload、cursor token、错误码、分页、身份 header 和 DTO mapper 都可能被藏在长文件或 map 里。

建议先跑：

```bash
find services libs -name '*.go' -not -path '*/vendor/*' -print0 \
  | xargs -0 wc -l \
  | awk '$1 > 400 && $2 != "total" {print $1, $2}' \
  | sort -nr

rg -n 'json\.Marshal|Payload|Outbox|map\[string\]any|map\[string\]interface|type .*Payload|type .*(Req|Resp|Request|Response|DTO|Dto)|json:"|http\.Request|gin\.Context' \
  services/*/internal/*/application \
  -g '*.go' -g '!*_test.go'
```

命中后按 owner 判断：

- HTTP request / response DTO 必须在 `services/<service>/api/http`。
- 跨服务同步 client DTO 必须在 provider-owned `libs/contracts/clients/<service>`。
- 跨服务事件 payload 必须在 provider-owned `libs/contracts/events/<domain>`。
- Application 可以暴露自有 command / query / result，但不能重新暴露 domain alias 或协议 DTO。
- Cursor token 等 application 自有内部序列化可以留在 application，但必须只服务 application 自己的不可见状态。

## 架构依赖方向

review 必须确认依赖方向符合 `docs/architecture/go-service-design.md` 和 `docs/architecture/repository-layout.md`。不能只看代码能编译。

允许方向：

```text
api/http -> application -> domain
application -> ports
infrastructure -> ports/domain mapping
runtime -> api/http/application/infrastructure wiring
```

重点检查：

- `domain` 没有导入 HTTP、Gin、SQL、Redis、RabbitMQ、MongoDB、Elasticsearch、Kubernetes、`libs/contracts` 或 provider client。
- `application` 没有导入 Gin、HTTP request/response、SQL driver、Redis client、RabbitMQ delivery、Mongo collection 或外部 SDK。
- `api/http` 没有直接访问 repository、database、cache、MQ、SDK 或绕过 application use case。
- `infrastructure` 没有反向调用 handler，也没有把底层错误、SQL 文本、SDK 类型暴露到 application API。
- `libs/contracts` 不依赖服务 `internal`，不包含 fallback、重试、缓存、熔断或 consumer 业务策略。
- `libs/kit` 不包含服务特定业务规则；不稳定的服务本地代码不要提前提升到共享库。

建议执行：

```bash
python3 tests/architecture/check_boundaries.py --root .
python3 -m unittest tests/architecture/check_boundaries_test.py
```

如果依赖方向规则本身变化，必须同步 `tests/architecture`、`docs/architecture/go-service-design.md` 和 `docs/architecture/repository-layout.md`。

## DDD 设计

review 必须判断代码是不是只按 service / repository 流水账组织，而没有领域模型承载业务语言。

先给变更命名：

- 聚合、实体和值对象是什么。
- 不变量和状态转换在哪里表达。
- use case 如何编排事务、权限、幂等和副作用。
- domain event、integration event、outbox row 三者是否分开。
- repository / cache / client / publisher port 是否由 application 消费方定义。

检查红线：

- Domain event 只表达业务事实，不携带 routing key、eventType、JSON tag、outbox status 或 broker 语义。
- Integration event contract 放在 `libs/contracts/events/<domain>`，application 负责从 domain fact 映射到 integration payload。
- Outbox row 只是持久化机制；不要让 domain 或 HTTP handler直接构造 outbox row。
- 领域规则优先在 domain 中表达；application 中大量重复 `if status/type/mode` 分支时，应检查是否缺少 domain 方法、值对象或策略。
- Repository 方法应按 use case 能力命名，不按表操作堆成 mega-interface。
- 事务边界由 application 拥有；repository 不隐藏跨聚合副作用或外部调用。
- 测试在 owner 层证明事实：domain test 证明不变量，application test 证明 use case 和 outbox 映射，handler test 证明 HTTP contract。

如果一个改动没有可命名的领域概念，只剩“调用几个仓储然后拼 DTO”，review 需要指出业务语义缺失，而不是只检查测试通过。

## 边界检查

对每个 material change，至少检查：

- `api/http` 是否只做路由、解包、鉴权上下文映射、DTO 转换和错误映射。
- `application` 是否拥有 use case、事务、权限、幂等、事件写入和端口调用，而不是 HTTP/Gin/数据库/SDK 细节。
- `domain` 是否保持框架无关，不依赖 HTTP、SQL、Redis、RabbitMQ、MongoDB、Elasticsearch 或 Kubernetes。
- `ports` 是否是 consumer-side 小接口，没有混入 provider contract、repository mega-interface 或技术 SDK 类型。
- `infrastructure` 是否只实现端口并翻译底层错误，没有构造业务成功 DTO 伪装降级。
- `runtime` / `cmd/server` 是否只负责组装、配置、健康、生命周期和真实依赖边界。

## Contract 和数据

以下变更必须按完整 contract surface review：

- HTTP path、method、query/path/body/multipart 字段、响应 envelope、错误码、分页、排序、过滤、header。
- `libs/contracts/clients` 的 path、caller operation、request / response DTO。
- `libs/contracts/events` 的 eventType、payloadVersion、routing key、payload 字段、时间格式、ID 语义和兼容性。
- migration、索引、唯一约束、内部主键、外部公开 ID、数据修复和 down migration。
- outbox / inbox / ledger、幂等键、事务边界、重试、补偿和乱序处理。

发现 contract 形状散落在 application、repository、测试匿名 struct 或 `map[string]any` 时，先判断是否应迁到 `api/http`、`libs/contracts` 或明确的本层 DTO。

## 代码质量

review 需要覆盖实现质量，不只覆盖架构位置：

- 改动是否 surgical：每条变更都能追溯到任务目标，没有顺手重构、格式 churn 或无关清理。
- 是否引入过度抽象：单用途 helper、空泛 manager、过早 shared package、无意义 interface、Java-style service hierarchy。
- 是否存在死代码、无 owner 方法、兼容 wrapper、只被测试引用的生产 API。
- 错误处理是否保留语义：必要时 `%w` 包装，不向公开 API 暴露 driver error、SQL、token、secret 或内部 sentinel。
- `context.Context` 是否从入口传递到 DB/cache/MQ/client；没有在普通业务路径里临时造 `context.Background()`。
- 事务是否只包数据库内工作；没有在事务内调用外部 HTTP、MQ publish、长耗时解析或不可控 SDK。
- SQL 是否参数化；rows 是否关闭并检查 `rows.Err()`；transaction 是否 `defer Rollback()`。
- goroutine、worker、retry、timeout、cancel、channel 和 map 访问是否有明确 owner、生命周期和并发安全。
- mapper 是否明确；机械字段复制不应散落在多处，业务语义也不应塞进生成 mapper。
- 文件是否过大：生产 Go 文件超过 400 行时，检查是否应该按 use case、endpoint、repository query、worker、mapper 或 fixture/helper 拆分。

建议补充扫描：

```bash
rg -n 'context\.Background\(|fmt\.Sprintf\(|go func|time\.Sleep\(|TODO|panic\(|interface \{|type .*Manager|type .*Helper' services libs -g '*.go'
rg -n 'func .*WithContext|Deprecated|compat|legacy|wrapper' services libs -g '*.go'
```

命中不一定是问题，但必须人工判断 owner、调用点和风险。

## 注释质量

review 需要检查注释是否帮助维护者理解业务和风险，而不是复述代码：

- 必须注释非显然的业务规则、领域不变量、状态转换、权限判断、幂等/补偿、事务顺序、降级策略和跨服务副作用。
- 必须注释为什么忽略错误、为什么允许 best-effort、为什么某个缺失字段可以省略。
- 不接受“// get user”“// set status”这类语法复述。
- 注释和代码冲突时以代码为当前事实，但 review 要求同步或删除误导注释。
- 公开 contract、错误码、事件 payload、migration 和运行策略变化时，注释不能替代文档；应同步事实源文档。
- 注释不得记录临时心理活动、过期计划、未验证猜测或“以后再说”的无 owner TODO。

新增关键业务逻辑但没有解释业务意图时，应作为 review finding；注释过量、重复和过时也应提出。

## 测试和验证

review 需要确认测试证明的是 owner 边界，不是只证明 mock 被调用：

- Handler / HTTP test：路由、身份、解包、序列化、envelope、公开错误。
- Application test：业务决策、状态迁移、事务、权限、幂等、事件写入。
- Repository test：SQL 查询、事务、约束、错误翻译。
- Runtime test：真实 wiring、健康、生命周期、配置失败语义。
- Contract test：跨服务 DTO、事件 payload、兼容字段和调用方 operation。

改动完成后按 `docs/reviews/quality-gates.md` 选择验证命令。结构、边界、共享 contract 或脚本变化优先运行：

```bash
make check
```

## Review 输出

正式输出必须先列 findings，再列总结。每条 finding 写清：

- 影响：会破坏 correctness、contract、安全、运行、数据一致性、测试有效性或维护边界的哪一项。
- 证据：文件路径、行号、调用路径、contract 文档或测试失败信号。
- 修复方向：最小正确修复，而不是泛泛建议。
- 验证：修复后必须重跑的命令或场景。

如果没有 finding，也要说明已覆盖的全集范围、实际验证命令和残余风险。
