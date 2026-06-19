# Documentation Rules

## 目标

- `~/.agents/AGENTS.md` 只保留全局入口、总则和专题路由，不再承载大段专题正文。
- `~/.agents/docs/` 承接需要按主题维护的长文规则、索引和归属说明。
- `~/.agents/README.md` 继续说明仓库用途和安装方式，不替代规则正文。

## 路径归属

- `docs/README.md`：`docs/` 的导航索引。
- `docs/documentation-rules.md`：`docs/` 的规则源。
- `docs/agent-rules/`：从全局 `AGENTS.md` 下沉出来的专题规则正文。

## 维护规则

- 新增长期规则时，先判断它是入口路由还是专题正文。
- 只要内容属于长文说明、专项规则或按场景读取的延伸正文，就放进 `docs/agent-rules/`，并在 `AGENTS.md` 与 `docs/README.md` 注册入口。
- 只要内容属于仓库用途、安装方式或目录说明，就放进 `README.md`。
- 不让 `AGENTS.md` 和 `docs/agent-rules/*` 同时维护同一段正文；`AGENTS.md` 只保留必要摘要和链接。
- 新增 `docs/` 下路径后，同步更新 `docs/README.md`；如果归属规则变化，再更新本文件。
