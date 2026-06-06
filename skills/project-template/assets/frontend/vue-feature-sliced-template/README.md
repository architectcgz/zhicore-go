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

当前模板现在更接近“可直接起页面壳并继续补业务”的 starter：

- `tree.txt`：推荐目录树
- `starter-files/`：起步文件与代码片段
- `manifest.json`：模板元信息与占位变量说明

这次补上的最小闭环：

- `vite.config.ts`、`tsconfig.json`、`index.html`、`src/style.css`
- `src/api/request.ts` 与 `src/api/auth.ts` 的最小网络层
- `src/stores/auth.ts` 与 `src/runtime/globalErrorRuntime.ts`
- `src/features/auth/`、`src/entities/user/`、`src/widgets/dashboard/` 示例
- `router guard`、错误页路由与登录 / dashboard 页面壳
- `createDefaultErrorRuntimeOptions()`：给 401/500/route error 提供可替换的默认行为

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
