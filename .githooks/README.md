# Git Hooks

本目录存放 `zhicore-go` 的可版本化 Git hooks。

安装：

```bash
bash scripts/install-githooks.sh
```

当前 hooks：

- `commit-msg`：运行 `scripts/check-commit-message.sh`。该脚本调用共享检查器 `~/.agents/harness/commit-message/check_commit_message.py`，并读取仓库内 `harness/policies/commit-message.json` 执行项目策略。

提交信息要求：

- 标题使用 `英文类型(可选 scope): 中文描述`，例如 `docs(配置): 确立环境变量规范`。
- 普通提交不能只有单行标题；正文至少两行有效内容，说明改动点、原因、影响或验证中的关键信息。
- 如果未来接入 task gate 且当前暂存改动命中激活任务，正文必须单独写一行 `Task: <task-slug>`；该元数据不计入正文说明行数。
- 默认不添加 `Co-Authored-By` 或其他署名 trailer，除非用户明确要求或项目规则另行要求。

推荐写法：

```bash
git commit -m "docs(门禁): 确立本地质量门禁规范" \
  -m "新增本地验证命令选择规则，明确 make check 的交付前职责。" \
  -m "同步 AGENTS、文档索引和结构检查，减少后续规范漂移。"
```
