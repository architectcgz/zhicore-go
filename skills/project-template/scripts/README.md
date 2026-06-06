# project-template scripts

当前提供的机械命令：

- `apply_project_template.py`
  - 列出可用模板
  - 将 `project-template/assets/` 下的 starter asset 渲染到目标目录
  - 同时替换文件内容、输出路径以及 `_template-tree.txt` 中的占位符
- `bash ~/.agents/harness/project-template-init.sh`
  - 提供模板短名和更顺手的参数名
  - 适合日常工作流或 agent 调用

便捷入口示例：

```bash
bash ~/.agents/harness/project-template-init.sh --list

bash ~/.agents/harness/project-template-init.sh backend-go \
  --dest /tmp/example-backend \
  --module github.com/acme/example-service \
  --service example-service \
  --domain example

bash ~/.agents/harness/project-template-init.sh frontend-vue \
  --dest /tmp/example-frontend \
  --app-name example-frontend \
  --auth-redirect /student/dashboard \
  --login-path /login
```

示例：

```bash
python3 ~/.agents/skills/project-template/scripts/apply_project_template.py --list

python3 ~/.agents/skills/project-template/scripts/apply_project_template.py \
  --template backend/go-backend-onion-template \
  --dest /tmp/example-backend \
  --var __GO_MODULE__=github.com/acme/example-service \
  --var __SERVICE_NAME__=example-service \
  --var __DOMAIN_NAME__=example

python3 ~/.agents/skills/project-template/scripts/apply_project_template.py \
  --template frontend/vue-feature-sliced-template \
  --dest /tmp/example-frontend \
  --var __APP_NAME__=example-frontend \
  --var __DEFAULT_AUTH_REDIRECT__=/student/dashboard \
  --var __DEFAULT_LOGIN_PATH__=/login
```
