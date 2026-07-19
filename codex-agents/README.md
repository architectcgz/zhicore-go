`~/.agents/codex-agents/` 是 Codex agent 定义的主体目录。

入口约定：

- `~/.codex/agents -> ~/.agents/codex-agents`

说明：

- 这里保留 Codex agent 的原生定义文件
- 目前以 `toml` 为官方运行时入口
- 部分 `md` 文件仍作为 legacy runner 兼容层保留

模型策略：

| 类型 | agent 定义 | 模型不可用时 |
|---|---|---|
| 默认 | 不声明 `model`，继承父会话模型 | 继续使用父会话已验证可用的 provider 和模型 |
| 偏好 | 在调度规则中声明偏好，不写入 agent 文件 | 回退到父会话模型，并向用户说明实际选择 |
| 硬要求 | 仅在任务确实依赖特定模型能力时声明 `model` | 调用前确认 provider 支持；失败后不得静默换模 |

- agent 的职责和专业能力由 `developer_instructions` 与 skills 定义，不通过模型名称表达。
- `toml` 与 legacy `md` 定义都不得为了角色分类默认固定模型。
- provider 不支持硬要求模型时，优先改由主 agent 完成；这会改变交付能力时再向用户说明限制。
