# vue-feature-sliced-template

基于 `ctf/code/frontend` 提炼的 Vue 前端架构模板。

目标：

- 保留 CTF 当前前端最稳定的页面壳、路由与切片边界
- 提供 Vue 3 + Vite + Pinia + Vue Router 的 Feature-Sliced 风格骨架
- 避免“只有空目录，没有真实起步文件”的模板

这套模板强调的边界：

- `src/pages/` 是 route composition surface
- `src/features/` 放用户动作与流程 owner
- `src/entities/` 放稳定业务对象表达
- `src/widgets/` 放页面级组合块
- `src/shared/` 放通用 UI、基础 model、基础库
- `src/router/routes/` 按路由命名空间拆分

测试放置规则：

- `src/shared/**/__tests__`：放 shared model、shared lib、shared UI 原语自己的行为测试；不得在这里测试具体 feature 或页面流程。
- `src/features/**/__tests__` 或 feature 内邻近 `__tests__`：放 feature owner 的状态、校验、权限、异步流程、组合逻辑和 feature 私有 UI 行为。
- `src/pages/**/__tests__`：放 route / page 入口集成测试，只覆盖页面装配、路由参数、入口状态和关键用户流程；不重复锁定 shared 组件或 feature 内部结构。
- `src/api/__tests__`：放 API adapter、请求参数、响应映射和后端 contract 对齐测试。
- `src/stores/__tests__`：放 Pinia store 行为、持久化、权限状态和跨页面共享状态。
- `src/router/__tests__`、`src/runtime/__tests__`、`src/config/__tests__`、`src/utils/__tests__`：分别放对应基础设施 owner 的测试，不承接业务场景。
- `src/__tests__`：只放架构边界、设计系统 guard、跨切面回归防线和无法归属到单一 feature / page / shared owner 的测试；新增前必须能说明它为什么不能贴近具体 owner。
- `src/test`：放 Vitest setup、测试环境适配和稳定复用的测试工具；只服务单个测试文件的 helper 先留在测试文件本地。
- TDD 写出的测试默认是行为规格和回归护栏，不因为对应功能已经实现就删除；只在行为信号重复、实现细节锁定、迁移 guard 到期，或目标行为明确废弃时合并或删除。

当前模板现在更接近“可直接起页面壳并继续补业务”的 starter：

- `tree.txt`：推荐目录树
- `starter-files/`：起步文件与代码片段
- `manifest.json`：模板元信息与占位变量说明
- `docs/reviews/` 与 `docs/todos/debt/`：review 证据和未收口技术债的默认目录骨架

这次补上的最小闭环：

- `vite.config.ts`、`tsconfig.json`、`index.html`、`src/style.css`
- `src/api/request.ts` 与 `src/api/auth.ts` 的最小网络层
- `src/stores/auth.ts` 与 `src/runtime/globalErrorRuntime.ts`
- `src/features/auth/`、`src/entities/user/`、`src/widgets/dashboard/` 示例
- `router guard`、错误页路由与登录 / dashboard 页面壳
- `createDefaultErrorRuntimeOptions()`：给 401/500/route error 提供可替换的默认行为
- `docs/reviews/README.md`：约定 review 证据的放置方式
- `docs/todos/debt/README.md` 与 `_template.md`：约定 unresolved debt 的目录、文件命名和条目格式

生成后建议先做：

- 把 `src/api/auth.ts` 的接口约定替换成项目真实 API 契约
- 检查 `src/runtime/globalErrorRuntime.ts` 的 401/500 跳转是否符合当前产品交互；如不适配，优先覆盖 `createDefaultErrorRuntimeOptions()`
- 从 `src/style.css` 的 theme token 开始替换品牌色、背景层级和文案语气
- 再决定是否需要扩 `entities/`、`widgets/`、权限路由和更细粒度 store

关键占位符：

- `__APP_NAME__`：应用名
- `__DEFAULT_AUTH_REDIRECT__`：默认鉴权后首页，例如 `/student/dashboard`
- `__DEFAULT_LOGIN_PATH__`：登录页，例如 `/login`

来源特征：

- 启动入口参考 `ctf/code/frontend/src/main.ts`
- 根壳参考 `ctf/code/frontend/src/App.vue`
- 路由骨架参考 `ctf/code/frontend/src/router/*`
- 页面壳与 layout bridge 参考 `ctf/code/frontend/src/pages/AppShellRoutePage.vue`
