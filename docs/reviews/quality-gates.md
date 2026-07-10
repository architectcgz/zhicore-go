# 本地质量门禁与 CI 规范

本文件定义 `zhicore-go` 的本地质量门禁、验证命令选择和未来 CI 最低要求。

## 基本原则

- 质量门禁按风险选择，不把所有小改动都升级成完整回归。
- `make check` 是统一交付前门禁，不是开发内循环里每次保存都必须运行的命令。
- 当前仓库还没有 CI；提供可安装的 `commit-msg` hook 用于提交信息检查。在 CI 建立前，交付说明中的手动验证证据仍是交付质量门禁的权威依据。
- 新增或调整检查命令时，必须同步 `Makefile`、`AGENTS.md`、相关事实源文档和必要的结构检查。
- 自动门禁不能替代人工判断。公开 contract、migration、runtime、并发、安全和跨服务边界变化仍按 `docs/reviews/done-definition.md` 判断是否需要正式 review。

## 命令职责

| 命令 | 职责 | 使用场景 |
| --- | --- | --- |
| `make check` | 组合运行结构检查、架构依赖方向检查、测试规模检查和所有 Go 模块测试 | 交付前统一门禁；脚手架、共享边界、contract、脚本、AGENTS 或跨模块改动后优先运行 |
| `bash scripts/check-structure.sh` | 检查服务入口、模块目录、文档入口和 agent 入口是否齐全 | 文档路径、索引、目录骨架、必备文件清单和入口规则变化 |
| `python3 tests/architecture/check_boundaries.py --root .` | 检查 Go 源码的服务间和服务内依赖方向 | 修改服务边界、`api/http`、`internal` 分层、`libs/kit`、`libs/contracts` 或相关检查规则 |
| `python3 scripts/check-inline-sql.py --root .` | 检查存储层是否把 SQL 硬编码在 Go 代码里 | 修改 `services/<service>/internal/<domain>/infrastructure/postgres` 下的 repository、store 或查询 |
| `make test-size` | 全量检查 `*_test.go` 文件规模 | 测试文件新增、拆分、合并或测试组织方式变化 |
| `make test` | 在每个 Go workspace 模块内运行 `go test ./...` | 需要仓库级 Go 测试证据，但不需要结构检查和测试规模检查的场景 |
| `cd <module> && go test ./...` | 运行单个模块测试 | 服务或共享库的开发内循环 |
| `cd <module> && go test ./path/...` | 运行最窄相关测试 | 单个 handler、service、repository、worker、adapter 或 contract 改动的首轮验证 |
| `cd <module> && go test -race ./path/...` | 检查数据竞争 | 并发、worker、consumer、cache、共享状态或 goroutine 生命周期变化 |
| `bash scripts/check-commit-message.sh <message-file>` | 检查提交信息格式 | commit-msg hook 或提交前手动验证提交信息 |

## 选择规则

- 只改文档索引、路径登记、目录说明或 agent 路由：至少运行 `bash scripts/check-structure.sh`。
- 改 `AGENTS.md`、`Makefile`、检查脚本、质量门禁、review 完成标准或文档结构检查：优先运行 `make check`；如果确实只影响路径存在性，可说明原因并至少运行 `bash scripts/check-structure.sh`。
- 改测试写法、测试目录归属、测试文件拆分或合并：运行 `make test-size` 和最窄相关 `go test`。
- 改单个服务行为：先运行最窄相关 `go test`；触达共享边界、公开 contract、脚手架或多个模块时，交付前再运行 `make check`。
- 改 `libs/*`、跨服务 contract、错误码、分页、事件 payload 或 typed client：运行相关模块测试，并在交付前运行 `make check`。
- 改 migration：验证 `up` 和 `down 1`；不可逆 migration 必须写明人工确认点。路径或索引变化时再运行 `bash scripts/check-structure.sh`。
- 改 worker、consumer、goroutine、cache、幂等或共享状态：说明是否运行 `go test -race`；不运行时写明原因。
- 需要真实服务、容器、端口或外部依赖的场景：自动化不足时写出手动验证输入、步骤和观察结果。

## CI 和 Hook 要求

当前没有 CI；仓库提供 `commit-msg` hook。未来新增或调整时遵守以下规则：

- CI 至少运行 `make check`，可以拆分阶段，但不能绕过 `Makefile` 中的统一入口。
- `commit-msg` hook 只负责提交信息格式，具体规则见 `docs/reviews/commit-message.md`。
- Git hook 可以作为本地提醒或快速失败机制，但不能成为唯一验证来源。
- CI 或 hook 新增、删除、重命名命令时，必须同步本文件、`AGENTS.md` 和 `docs/reviews/done-definition.md` 中的验证说明。
- CI 失败不能通过放宽测试断言、删除检查或把失败命令移出 `make check` 来规避；需要修复根因，或在确认外部依赖故障时记录残余风险。

## 维护规则

- 新增检查脚本时，优先按职责创建独立入口，并由 `make check` 组合。
- `scripts/check-structure.sh` 只检查仓库结构、固定入口和必备路径，不承载测试规模、源码扫描、契约语义或运行时规则。
- 涉及递归扫描、解析、聚合错误、白名单、差异模式或后续扩展的检查优先使用 Python。
- 如果某个文档规则需要机械防线，文档必须写明对应命令；如果某个检查守护必备路径，`scripts/check-structure.sh` 必须同步登记。
- 交付说明只列实际执行过的命令；没有运行的命令不能写成已通过。
