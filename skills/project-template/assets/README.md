# Project Template Assets

本目录存放 `project-template` 统一维护的项目代码模板资产。

当前模板：

- `backend/go-backend-onion-template/`
  - 基于 `ctf/code/backend` 提炼的 Go + Onion/Clean Architecture 骨架
- `frontend/vue-feature-sliced-template/`
  - 基于 `ctf/code/frontend` 提炼的 Vue 3 + Vite + Feature-Sliced 风格骨架

约定：

- 模板文件以“真实 starter”为目标，至少要让入口、依赖方向和关键骨架闭合，而不是只给空目录
- 需要替换的变量统一使用 `__TOKEN__` 形式
- harness 初始化、workflow 安装、文档脚手架不放在这里，仍分别由 `harness-*` 和 `documentation-architecture` 负责
- 如果模板需要默认保留 post-review evidence / debt 目录，优先提供最小可工作的 `docs/reviews/` 与 `docs/todos/debt/` 骨架，并在目录内给出 README / 模板文件；不要只在 AGENTS 里口头提示

日常生成模板时，优先使用：

- `bash ~/.agents/harness/project-template-init.sh --list`
- `bash ~/.agents/harness/project-template-init.sh backend-go ...`
- `bash ~/.agents/harness/project-template-init.sh frontend-vue ...`

它只是对 `scripts/apply_project_template.py` 的外层包装，不复制模板渲染逻辑。
