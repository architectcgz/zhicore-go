# 提交信息规范

本文件定义 `zhicore-go` 的提交信息格式和 commit-msg 机械检查规则。

## 基本要求

- 提交信息使用“标题 + 正文”两段结构。
- 标题格式为 `type(scope): 中文描述` 或 `type: 中文描述`。
- `type` 必须使用英文，例如 `feat`、`fix`、`refactor`、`docs`、`test`、`chore`、`build`、`ci`、`perf`、`style`、`revert`。
- `scope` 可选，使用英文或稳定模块名，例如 `upload`、`runtime`、`配置` 不作为 scope；中文说明放在冒号后。
- 普通提交正文至少两行有效内容，并说明改动点、原因、影响或验证中的关键信息。
- 默认不添加 `Co-Authored-By` 或其他署名 trailer，除非用户明确要求或项目规则另行要求。

## 示例

```bash
git commit -m "docs(配置): 确立环境变量规范" \
  -m "新增服务配置、环境变量命名和密钥处理规则。" \
  -m "同步 AGENTS 路由、文档索引和结构检查，避免后续规则漂移。"
```

```bash
git commit -m "fix(upload): 修正文件删除错误映射" \
  -m "把对象存储 404 翻译为应用层文件不存在错误。" \
  -m "补充 handler 回归测试，确保对外错误码保持稳定。"
```

## Task 元数据

当前仓库还没有完整 task gate。未来如果接入 `scripts/check-startup-gate.sh`，并且当前暂存改动命中激活任务，提交正文必须单独写一行：

```text
Task: <task-slug>
```

`Task:` 行属于元数据，不计入正文有效说明行数。

## 机械检查

项目策略文件：

```text
harness/policies/commit-message.json
```

检查入口：

```bash
bash scripts/check-commit-message.sh <commit-message-file>
```

Git hook：

```text
.githooks/commit-msg
```

安装 hook：

```bash
bash scripts/install-githooks.sh
```

安装后，Git 会把 `core.hooksPath` 指向 `.githooks`，提交时自动执行 `commit-msg` 检查。

## 维护规则

- 修改提交信息格式、允许的 type、正文行数、Task 元数据或 hook 接线时，必须同步本文件、`harness/policies/commit-message.json`、`.githooks/README.md` 和 `AGENTS.md`。
- 修改 `scripts/check-commit-message.sh` 时，优先保持它是薄 wrapper；共享检查逻辑继续由 `~/.agents/harness/commit-message/check_commit_message.py` 负责。
- 不要把 pre-commit、测试门禁或 review 规则混进提交信息检查；这些规则分别归 `docs/reviews/quality-gates.md` 和 `docs/reviews/done-definition.md` 管理。
