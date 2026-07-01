# 测试策略

本文件定义 `zhicore-go` 的测试分层、风险分级和验证要求。测试是必要的，但本项目不把严格 TDD 作为所有代码改动的绝对前置条件。

## 核心原则

- 所有改动都必须有验证证据；验证可以是结构检查、现有测试、新增测试、运行时检查或手动验证记录。
- 不强制所有代码改动先写失败测试。是否 test-first 由风险、现有覆盖和回归价值决定。
- 行为变更必须能被某种测试或验证证明；无法自动化验证时必须说明原因和手动验证方式。
- 新增测试必须证明一个真实行为契约，不为满足流程而测试 getter、setter、纯 mock 调用或实现细节。
- 已有测试能覆盖同一行为时，不重复添加同信号测试；运行最窄相关测试即可。
- Bugfix、contract、权限、分页、事务、幂等、并发、worker / consumer 和 migration 属于高风险面，应优先先补失败用例或回归测试。
- 测试失败时先判断 owner、contract、分层或实现语义是否未收口；不要为了变绿而放宽断言、修改 fixture 或 mock 迁就错误实现。

## 风险分级

| 等级 | 场景 | 新增测试要求 | 验证要求 |
| --- | --- | --- | --- |
| R0 | 文档、注释、README、索引、脚手架路径登记 | 不新增测试 | 运行相关结构检查，例如 `bash scripts/check-structure.sh` |
| R1 | 无行为变化的机械重构、移动文件、拆包、改导入、改命名 | 不强制新增测试 | 运行受影响模块现有最窄测试 |
| R2 | 小行为调整，且已有测试覆盖目标行为 | 不强制新增测试 | 运行覆盖该行为的现有测试，并在交付说明中说明覆盖来源 |
| R3 | 新 endpoint、新 use case、新 repository、新错误码、新权限/分页/过滤/字段 contract | 必须新增 focused test；推荐 test-first，但不强制 | 运行新增测试和受影响包测试 |
| R4 | bugfix、并发、事务、幂等、worker / consumer、migration、跨服务 contract、数据一致性 | 必须新增回归或 contract test；应 test-first，除非先复现不可行 | 运行新增测试、受影响包测试；必要时加 `-race`、runtime 或 system 测试 |

## Test-first 触发条件

以下场景默认先写失败用例或 characterization test，再改实现：

- 修复已知 bug，尤其是线上/手动复现问题。
- 修改公开 HTTP contract、错误码、字段序列化、分页、排序、过滤或权限语义。
- 修改 repository 查询语义、transaction、幂等、outbox / inbox、worker / consumer ack / retry / dead-letter。
- 修改 migration、schema 约束、唯一索引、外部 ID 或数据归属。
- 修改并发逻辑、缓存一致性、锁、重试、超时、后台任务或事件消费。

可以不 test-first 的场景：

- 文档、注释、索引和目录登记。
- 已有测试清楚覆盖的小改动。
- 纯机械重构，行为由现有测试保护。
- 临时探索或 spike，但进入正式实现前必须补验证。

## 测试放置

| 位置 | 负责内容 |
| --- | --- |
| `services/<service>/**/*_test.go` | 服务本地 handler、application/use case、domain、repository、adapter、worker、consumer 行为。 |
| `libs/*/**/*_test.go` | 共享 contract 或 kit 原语，不测试服务私有业务。 |
| `tests/architecture` | 源码级架构边界检查，例如禁止跨服务导入 `internal`、`libs/kit` 不依赖服务私有包；当前入口是 `python3 tests/architecture/check_boundaries.py --root .`。 |
| `tests/system/http` | 黑盒 HTTP 场景，验证已实现服务的外部 API contract 和核心流程。 |
| `tests/runtime` | 需要真实服务、容器、端口、PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch 等基础设施的测试。 |
| `tests/testkit` | 黑盒测试 fixture、builder、HTTP client、断言辅助；不能承载业务规则。 |

新增测试前先判断它证明的是哪一层契约。需要多层覆盖时，每一层必须证明不同事实：handler / HTTP test 证明路由、鉴权、序列化和错误 envelope；application test 证明业务决策和状态迁移；repository test 证明查询、事务、约束和错误翻译；runtime / system test 证明真实依赖协作。

## 测试写法和规模控制

- 一个 test 只证明一个行为契约或一个失败信号。跨多个 endpoint、use case 或 repository 查询语义的测试必须拆开，除非它本身就是黑盒 workflow 测试。
- 测试函数命名使用 `Test<Subject><Behavior>`，让失败输出能直接定位行为，例如 `TestUploadImageRejectsUnsupportedContentType`。
- Table test 只用于同一规则的多个输入例子；不同业务分支、权限语义、错误类型或副作用不要塞进同一个 table。
- 断言优先面向外部可见结果、状态变化、错误码、事件、持久化结果或幂等效果；不要只验证 mock 被调用。
- 使用 fake / stub 时，fake 应贴近被测端口，记录必要输入和副作用即可；mock 调用顺序只有在顺序本身是 contract 时才断言。
- Helper 必须调用 `t.Helper()`，优先接收 `testing.TB`；不要在测试里用 `time.Sleep` 做同步，改用 context、channel、fake clock、hook 或可观察状态。
- 测试数据少量内联即可；重复出现两次后提取 package-local builder / fixture。只有被两个以上包复用、且不携带业务规则时，才提升到 `tests/testkit`。
- 单个 `*_test.go` 目标控制在 400 行以内。超过 400 行时，review 应要求按 endpoint、use case、repository query、worker 行为或 fixture/helper 拆分。
- 单个 `*_test.go` 超过 800 行属于测试结构问题，`make test-size` 和 `scripts/check-test-size.py` 会失败；不能用“只是测试代码”作为保留理由。
- 单个测试函数目标控制在 80 行以内。超过后优先拆子行为、提取请求构造/响应断言 helper，或把黑盒 workflow 移到 `tests/system/http`。
- `handler_test.go`、`repository_test.go` 只适合包还小的时候使用；同一包出现多个 endpoint 或多个查询族后，应拆成 `<operation>_test.go`、`<resource>_handler_test.go`、`<query>_repository_test.go` 等更窄文件。
- 新增测试后必须做一次整理：合并同失败信号的测试、删除过时断言、提取重复 fixture，并确认失败信息包含关键输入和实际输出。
- 拆分超长测试文件时，优先先移动顶层 `Test...` 到按行为命名的新文件，让原文件暂时保留 shared helper / fixture owner；不要第一步就做跨包 testkit 大抽取。
- 共享 schema、seed、HTTP 请求构造、响应断言或 fake 依赖如果已经被三个以上测试文件重复使用，应收口到稳定 helper；只服务单个文件的一次性封装继续留在本地。

## 改动类型要求

### HTTP handler 和 contract

- 新 endpoint 必须先有 `services/<service>/api/http/README.md` 和 `endpoints/<operation>.md` schema。
- Handler test 验证 path、method、请求字段、响应 envelope、`data` 字段、公开错误码和鉴权边界。
- 历史入口或兼容例外必须在 endpoint schema 中写明，并用测试锁定。

### Application / use case

- 测试业务分支、权限判断、状态机、幂等、事件写入和端口调用结果。
- 优先断言行为结果，避免只验证 mock 被调用。
- 多个输入例子属于同一规则时使用 table test；无关行为拆成独立测试。

### Repository / GORM / SQL

- 测试持久化 contract、查询过滤、唯一约束、事务语义和错误翻译。
- 使用 GORM 时，测试应验证 repository 行为，不验证 GORM 本身。
- 正式 schema 来自 migration；测试 fixture 可以用 migration 初始化测试库，或在临时 fixture 中使用受控 schema setup。

### Migration

- 每个正式 migration 至少验证 `up` 能在空测试库执行。
- 可逆 migration 应验证最近一条 `down 1`。
- 不可逆 migration 必须在 `.down.sql` 中显式失败并说明原因，review 时人工确认。

### Worker / consumer / job

- 测试 ack / nack、retry、dead-letter、幂等、重复投递、乱序和取消逻辑。
- 有 goroutine、共享状态、缓存或后台循环时，能跑 race detector 的包应考虑 `go test -race`。

## 验证命令选择

本节说明测试相关命令的选择；交付前本地质量门禁组合见 `docs/reviews/quality-gates.md`。

优先从最窄相关命令开始：

```bash
cd services/<service> && go test ./path/...
cd libs/<module> && go test ./...
make test-size
bash scripts/check-structure.sh
python3 tests/architecture/check_boundaries.py --root .
make check
```

规则：

- 只改服务内代码，先跑该服务最窄包测试。
- 改测试文件组织、测试 helper、测试目录或规模规则，先跑 `make test-size`。只查当前工作区改动可用 `python3 scripts/check-test-size.py --working-tree`，只查暂存内容可用 `python3 scripts/check-test-size.py --staged`，只查指定文件或目录可用 `python3 scripts/check-test-size.py --files <path...>`。
- 改共享库、contract、脚手架、文档入口或检查脚本，交付前跑 `bash scripts/check-structure.sh`；涉及 Go 源码依赖方向时跑 `python3 tests/architecture/check_boundaries.py --root .`；必要时跑 `make check`。
- 改并发、worker、consumer、缓存或共享状态，考虑 `go test -race`。
- 改解析器、校验器、协议输入或安全敏感输入，考虑补 seed regression test 或 focused fuzz test。

## 不新增测试时的说明

行为相关改动如果不新增测试，交付说明或 review 记录必须写明原因，例如：

- 已有 `services/zhicore-file/api/http` handler test 覆盖该行为。
- 改动只调整文档、索引或结构检查脚本，已运行 `bash scripts/check-structure.sh`。
- 改动只能通过手动验证，已记录命令、输入和结果。

不要用“改动很小”作为唯一理由；理由必须指向风险、覆盖或验证方式。
