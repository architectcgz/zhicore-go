# 前后端 API 基础 Req/Resp 实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 API contract、请求解包、错误映射、feature workflow 的切片按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 为 `zhicore-go` 与 `zhicore-frontend-vue` 补齐基础 API `Req` / `Resp`、统一 envelope / pagination / error 类型、provider adapter 和首批 feature 接入边界，让后续服务实现按同一 contract 入口推进。

**架构：** 后端以 provider 拥有 contract 为原则：HTTP 字段 schema 和 handler request / response struct 归 `services/<service>/api/http`，跨服务 typed client 才进入 `libs/contracts`。前端以 `src/api` 作为 provider HTTP adapter，feature workflow 作为业务调用入口；页面、布局和组件不直接 import `src/api`。

**技术栈：** Go 1.23 workspace、标准库 `net/http`、`libs/kit/httpapi`、Vue 3、TypeScript strict mode、Axios、Vitest。

---

## 背景依据

- 后端 contract 规则：`docs/contracts/README.md`、`docs/contracts/http.md`、`docs/contracts/api-design-documentation.md`、`docs/contracts/http-schema-template.md`。
- 后端服务落点：`services/<service>/api/http` 放 HTTP 入站层和服务级 HTTP schema；`libs/kit/httpapi` 放通用 envelope。
- 前端规则：`zhicore-frontend-vue/AGENTS.md`、`zhicore-frontend-vue/docs/architecture/frontend-engineering-guidelines.md`、`zhicore-frontend-vue/src/api/README.md`。
- 当前已存在：File handler 与部分 contract test、Content Go-first 大草案、Auth endpoint 草案、Comment / Ranking 部分 schema。

## 已锁定决策

- `src/api` 不是 feature 私有目录；它是后端 provider 的可复用 HTTP adapter。
- `src/features/<feature>/api` 只用于 feature-private endpoint。只要 endpoint 属于 Content、Auth、User、File、Comment 等 provider 的稳定资源能力，就放 `src/api/<provider>.ts`。
- 前端通用 `ApiEnvelope<T>`、分页、错误详情进入 `src/types/api/`；具体接口 `Req` / `Resp` 贴近对应 provider adapter。
- 后端 HTTP `Req` / `Resp` struct 贴近 handler，默认放 `services/<service>/api/http`。不要把 HTTP DTO 放入 domain、repository、migration 或跨服务 contracts。
- 计划按服务切片推进，不一次性实现所有 endpoint。每个服务切片必须先达到 `Contract 草案`，再补 handler / adapter。

## 文件结构

### `zhicore-go`

- 修改：`libs/kit/httpapi/response.go`
  - 视 File 错误码迁移时机，补业务错误码写入能力；本计划第一阶段只要求不破坏现有 envelope。
- 修改：`services/zhicore-file/api/http/handler.go`
  - 命名并稳定 `UploadFileResp` 等 response struct。
- 修改：`services/zhicore-file/api/http/handler_test.go`
- 修改：`services/zhicore-file/api/http/upload_images_batch_test.go`
  - 补齐 audio / batch contract test。
- 修改：`services/zhicore-file/api/http/README.md`
- 修改：`services/zhicore-file/api/http/endpoints/*.md`
  - 同步已验证状态。
- 修改：`services/zhicore-content/api/http/README.md`
- 新增或拆分：`services/zhicore-content/api/http/endpoints/create-post.md`
- 新增或拆分：`services/zhicore-content/api/http/endpoints/save-draft-body.md`
- 新增或拆分：`services/zhicore-content/api/http/endpoints/publish-post.md`
- 新增或拆分：`services/zhicore-content/api/http/endpoints/get-post-body.md`
- 修改：`services/zhicore-auth/api/http/README.md`
- 修改：`services/zhicore-auth/api/http/endpoints/login.md`
- 修改：`services/zhicore-auth/api/http/endpoints/me.md`
- 修改：`services/zhicore-auth/api/http/endpoints/csrf.md`
- 修改：`services/zhicore-auth/api/http/endpoints/refresh.md`
- 修改：`services/zhicore-auth/api/http/endpoints/logout.md`
- 修改：`services/zhicore-auth/api/http/endpoints/register.md`
- 修改：`services/zhicore-user/api/http/README.md`
- 新增：`services/zhicore-user/api/http/endpoints/get-me.md`
- 新增：`services/zhicore-user/api/http/endpoints/get-profile.md`
- 新增：`services/zhicore-user/api/http/endpoints/update-profile.md`
- 修改：`services/zhicore-comment/api/http/README.md`
- 修改：`services/zhicore-comment/api/http/endpoints/create-comment.md`
- 修改：`services/zhicore-comment/api/http/endpoints/list-comments-page.md`
- 修改：`services/zhicore-ranking/api/http/README.md`
- 修改：`services/zhicore-ranking/api/http/endpoints/ranking-api.md`
- 后续新增：`services/zhicore-search/api/http/README.md`
- 后续新增：`services/zhicore-notification/api/http/README.md`
- 后续新增：`services/zhicore-message/api/http/README.md`
- 后续新增：`services/zhicore-admin/api/http/README.md`
- 后续新增：`services/zhicore-gateway/api/http/README.md`
- 后续新增：`services/zhicore-ops/api/http/README.md`

### `zhicore-frontend-vue`

- 新增：`src/types/api/response.ts`
- 新增：`src/types/api/pagination.ts`
- 新增：`src/types/api/error.ts`
- 新增：`src/types/api/index.ts`
- 修改：`src/types/README.md`
- 修改：`src/api/request.ts`
  - 统一 Go envelope 解包和 `ApiError` 标准化。
- 修改：`src/api/post.ts`
  - Content/Post provider adapter；保留 editor 当前 `saveDraftBody`，补齐 create / publish / read body 的类型。
- 新增：`src/api/file.ts`
- 修改：`src/api/auth.ts`
- 新增：`src/api/user.ts`
- 新增：`src/api/comment.ts`
- 新增：`src/api/ranking.ts`
- 后续新增：`src/api/search.ts`
- 后续新增：`src/api/notification.ts`
- 后续新增：`src/api/message.ts`
- 后续新增：`src/api/admin.ts`
- 后续新增：`src/api/ops.ts`
- 修改或新增：`src/api/__tests__/*.spec.ts`
- 修改：`src/features/editor/lib/editorDraftSavePayload.ts`
- 修改：`src/features/editor/composables/useEditorDraftSaveWorkflow.ts`
- 修改：`src/features/auth/composables/useLoginForm.ts`
- 修改或新增：对应 feature workflow 测试。

## 任务 1：前端 API 通用类型和 request 解包

**测试立场：** TDD - HTTP envelope、错误映射和类型出口是跨模块行为。

**文件：**

- 新增：`zhicore-frontend-vue/src/types/api/response.ts`
- 新增：`zhicore-frontend-vue/src/types/api/error.ts`
- 新增：`zhicore-frontend-vue/src/types/api/pagination.ts`
- 新增：`zhicore-frontend-vue/src/types/api/index.ts`
- 修改：`zhicore-frontend-vue/src/types/README.md`
- 修改：`zhicore-frontend-vue/src/api/request.ts`
- 新增或修改：`zhicore-frontend-vue/src/api/__tests__/request.spec.ts`

- [x] 编写失败测试：`request` 成功响应能从 Go envelope 中返回 `data`。

  建议断言：

  ```ts
  expect(await unwrapApiEnvelope({ code: 200, message: "操作成功", data: { id: "p1" }, timestamp: 1 })).toEqual({ id: "p1" });
  ```

- [x] 编写失败测试：错误响应保留 `code`、`message`、`traceId`、`details` 和 HTTP status。

  运行：`pnpm exec vitest run src/api/__tests__/request.spec.ts`

  预期：测试失败，指出当前 `ApiError` 只保留 `status` 和 `requestUrl`。

- [x] 新增 `ApiEnvelope<T>`、`ApiErrorDetail`、`ApiPageReq`、`ApiPageResp<T>`、`ApiCursorReq`、`ApiCursorResp<T>` 类型。

- [x] 修改 `ApiError`，保留公开错误码和字段级错误详情；不要在 request 层做 toast、跳转或业务权限判断。

- [x] 将 `post.ts` 内联的 `ApiEnvelope<T>` 改为引用 `src/types/api`。

- [x] 运行目标测试。

  运行：`pnpm exec vitest run src/api/__tests__/request.spec.ts src/api/__tests__/post.spec.ts`

  预期：全部通过。

- [x] 运行类型检查。

  运行：`pnpm typecheck`

  预期：通过。

## 任务 2：File 服务 Req/Resp 和前端 `file` adapter

**测试立场：** TDD - 上传、URL、delete 的 HTTP contract 与 multipart 字段必须被测试锁住。

**文件：**

- 修改：`zhicore-go/services/zhicore-file/api/http/handler.go`
- 修改：`zhicore-go/services/zhicore-file/api/http/handler_test.go`
- 修改：`zhicore-go/services/zhicore-file/api/http/upload_images_batch_test.go`
- 修改：`zhicore-go/services/zhicore-file/api/http/README.md`
- 修改：`zhicore-go/services/zhicore-file/api/http/endpoints/upload-audio.md`
- 修改：`zhicore-go/services/zhicore-file/api/http/endpoints/upload-images-batch.md`
- 新增：`zhicore-frontend-vue/src/api/file.ts`
- 新增：`zhicore-frontend-vue/src/api/__tests__/file.spec.ts`

- [x] 后端编写失败测试：`POST /api/v1/files/audio` 成功返回 envelope 和 `UploadFileResp`。

  运行：`cd zhicore-go/services/zhicore-file && go test ./api/http -run TestUploadAudio`

  预期：失败，因为 audio 成功场景 contract test 尚未补齐。

- [x] 后端编写失败测试：`POST /api/v1/files/images/batch` 覆盖字段名 `files`、默认 `PUBLIC`、`PRIVATE` 和部分失败语义。

- [x] 将 `uploadResponse` 重命名为导出或清晰的 `UploadFileResp`；保持 JSON 字段与 schema 一致。

- [x] 更新 File endpoint 文档状态：已有 test 覆盖的 endpoint 标为“已验证”，未覆盖的保持“草案”。

- [x] 运行 File HTTP 测试。

  运行：`cd zhicore-go/services/zhicore-file && go test ./api/http`

  预期：通过。

- [x] 前端编写失败测试：`uploadImage(file)` 使用 `multipart/form-data` 字段名 `file` 并返回 `UploadFileResp`。

  运行：`cd zhicore-frontend-vue && pnpm exec vitest run src/api/__tests__/file.spec.ts`

  预期：失败，因为 `src/api/file.ts` 不存在。

- [x] 新增 `src/api/file.ts`，包含 `UploadFileResp`、`UploadImageReq`、`UploadAudioReq`、`UploadImagesBatchReq`、`getFileUrl()`、`deleteFile()`。

- [x] 运行前端 File adapter 测试和类型检查。

  运行：

  ```bash
  pnpm exec vitest run src/api/__tests__/file.spec.ts
  pnpm typecheck
  ```

  预期：全部通过。

## 任务 3：Content 编辑器最小闭环

**测试立场：** TDD - 草稿保存、发布和正文读取是核心 contract 与 feature workflow。

**文件：**

- 修改：`zhicore-go/services/zhicore-content/api/http/README.md`
- 拆分或新增：`zhicore-go/services/zhicore-content/api/http/endpoints/create-post.md`
- 拆分或新增：`zhicore-go/services/zhicore-content/api/http/endpoints/save-draft-body.md`
- 拆分或新增：`zhicore-go/services/zhicore-content/api/http/endpoints/publish-post.md`
- 拆分或新增：`zhicore-go/services/zhicore-content/api/http/endpoints/get-post-body.md`
- 修改：`zhicore-frontend-vue/src/api/post.ts`
- 修改：`zhicore-frontend-vue/src/api/__tests__/post.spec.ts`
- 修改：`zhicore-frontend-vue/src/features/editor/lib/editorDraftSavePayload.ts`
- 修改：`zhicore-frontend-vue/src/features/editor/__tests__/editorDraftSavePayload.spec.ts`
- 修改：`zhicore-frontend-vue/src/features/editor/composables/useEditorDraftSaveWorkflow.ts`
- 修改：`zhicore-frontend-vue/src/features/editor/__tests__/useEditorDraftSaveWorkflow.spec.ts`

- [x] 后端把 Content 大草案中编辑器最小闭环拆成单 endpoint schema，状态保持“草案”。

- [x] 在 `save-draft-body.md` 明确 `SaveDraftBodyReq`、`SaveDraftBodyResp`、并发基线字段和错误码。

- [x] 前端编写失败测试：`createPost()`、`saveDraftBody()`、`publishPost()`、`getPostBody()` 使用 Go schema 的 path、body 和 envelope。

- [x] 修改 `src/api/post.ts`，补齐 `CreatePostReq`、`CreatePostResp`、`SaveDraftBodyReq`、`SaveDraftBodyResp`、`PublishPostReq`、`PublishPostResp`、`PostBodyResp`。

- [x] 收敛 editor feature 的 server save port：feature 只依赖 `EditorDraftServerSaveClient`，不在页面或组件中直接 import `src/api/post`。

- [x] 运行 Content 前端目标测试。

  运行：

  ```bash
  pnpm exec vitest run \
    src/api/__tests__/post.spec.ts \
    src/features/editor/__tests__/editorDraftSavePayload.spec.ts \
    src/features/editor/__tests__/useEditorDraftSaveWorkflow.spec.ts
  pnpm typecheck
  ```

  预期：全部通过。

- [x] 后端运行结构检查。

  运行：`cd zhicore-go && bash scripts/check-structure.sh`

  预期：`structure ok`。

## 任务 4：Auth 基础会话接口

**测试立场：** TDD - 登录、当前主体、CSRF、refresh、logout、register 涉及认证状态和错误分支。

**文件：**

- 修改：`zhicore-go/services/zhicore-auth/api/http/README.md`
- 修改：`zhicore-go/services/zhicore-auth/api/http/endpoints/login.md`
- 修改：`zhicore-go/services/zhicore-auth/api/http/endpoints/me.md`
- 修改：`zhicore-go/services/zhicore-auth/api/http/endpoints/csrf.md`
- 修改：`zhicore-go/services/zhicore-auth/api/http/endpoints/refresh.md`
- 修改：`zhicore-go/services/zhicore-auth/api/http/endpoints/logout.md`
- 修改：`zhicore-go/services/zhicore-auth/api/http/endpoints/register.md`
- 修改：`zhicore-frontend-vue/src/api/auth.ts`
- 新增或修改：`zhicore-frontend-vue/src/api/__tests__/auth.spec.ts`
- 修改：`zhicore-frontend-vue/src/features/auth/composables/useLoginForm.ts`
- 修改或新增：`zhicore-frontend-vue/src/features/auth/__tests__/useLoginForm.spec.ts`

- [x] 后端逐 endpoint 补齐 `Req` / `Resp` 表：login、me、csrf、refresh、logout、register。

- [x] 明确 cookie / token 字段边界：refresh token 不进 body，CSRF 只走约定 header / cookie。

- [x] 前端编写失败测试：`login()` 解 Go envelope，并返回 `AuthPrincipalResp` 或映射后的 `AuthUser`。

- [x] 修改 `src/api/auth.ts`，不要继续假设后端直接返回裸 `AuthUser`。

- [x] 修改 `useLoginForm`，确认提交、store 写入、导航仍在 feature workflow 内。

- [x] 运行 Auth 前端测试。

  运行：

  ```bash
  pnpm exec vitest run src/api/__tests__/auth.spec.ts src/features/auth
  pnpm typecheck
  ```

  预期：全部通过。

- [x] 运行后端结构检查。

  运行：`cd zhicore-go && bash scripts/check-structure.sh`

  预期：`structure ok`。

## 任务 5：User、Comment、Ranking 首批查询接口

**测试立场：** TDD - profile、评论分页、排行榜分页涉及 DTO、分页和可见性 contract。

**文件：**

- 新增或修改：`zhicore-go/services/zhicore-user/api/http/README.md`
- 新增：`zhicore-go/services/zhicore-user/api/http/endpoints/get-me.md`
- 新增：`zhicore-go/services/zhicore-user/api/http/endpoints/get-profile.md`
- 新增：`zhicore-go/services/zhicore-user/api/http/endpoints/update-profile.md`
- 修改：`zhicore-go/services/zhicore-comment/api/http/README.md`
- 修改：`zhicore-go/services/zhicore-comment/api/http/endpoints/create-comment.md`
- 修改：`zhicore-go/services/zhicore-comment/api/http/endpoints/list-comments-page.md`
- 修改：`zhicore-go/services/zhicore-ranking/api/http/README.md`
- 修改：`zhicore-go/services/zhicore-ranking/api/http/endpoints/ranking-api.md`
- 新增：`zhicore-frontend-vue/src/api/user.ts`
- 新增：`zhicore-frontend-vue/src/api/comment.ts`
- 新增：`zhicore-frontend-vue/src/api/ranking.ts`
- 新增：`zhicore-frontend-vue/src/api/__tests__/user.spec.ts`
- 新增：`zhicore-frontend-vue/src/api/__tests__/comment.spec.ts`
- 新增：`zhicore-frontend-vue/src/api/__tests__/ranking.spec.ts`

- [x] User：补 `getMe`、`getProfile`、`updateProfile` 字段级 contract 草案。

- [x] Comment：补 create / list 的 request、response、排序、分页、可见性和错误码。

- [x] Ranking：从 `ranking-api.md` 中优先固定公开热榜、周期榜和 score item 的分页响应。

- [x] 前端编写失败测试：User / Comment / Ranking adapter 都复用 `src/types/api` 分页和 envelope 类型。

- [x] 新增三个 provider adapter，避免任何 feature 复制 provider DTO。

- [x] 运行前端 API adapter 测试。

  运行：

  ```bash
  pnpm exec vitest run \
    src/api/__tests__/user.spec.ts \
    src/api/__tests__/comment.spec.ts \
    src/api/__tests__/ranking.spec.ts
  pnpm typecheck
  ```

  预期：全部通过。

- [x] 运行后端结构检查。

  运行：`cd zhicore-go && bash scripts/check-structure.sh`

  预期：`structure ok`。

## 任务 6：Search、Notification、Message、Admin、Gateway、Ops 计划化占位

**测试立场：** TDD 仅用于真实 adapter 或 handler；本任务主要是 contract 草案和禁止错误 owner。

**文件：**

- 新增：`zhicore-go/services/zhicore-search/api/http/README.md`
- 新增：`zhicore-go/services/zhicore-notification/api/http/README.md`
- 新增：`zhicore-go/services/zhicore-message/api/http/README.md`
- 新增：`zhicore-go/services/zhicore-admin/api/http/README.md`
- 新增：`zhicore-go/services/zhicore-gateway/api/http/README.md`
- 新增：`zhicore-go/services/zhicore-ops/api/http/README.md`
- 后续新增：`zhicore-frontend-vue/src/api/search.ts`
- 后续新增：`zhicore-frontend-vue/src/api/notification.ts`
- 后续新增：`zhicore-frontend-vue/src/api/message.ts`
- 后续新增：`zhicore-frontend-vue/src/api/admin.ts`
- 后续新增：`zhicore-frontend-vue/src/api/ops.ts`

- [x] 为每个服务 README 写清 provider owner、首批 endpoint、待提取 contract 和禁止 facade 复制规则。

- [x] Gateway README 只记录 Gateway 自有能力：health、route diagnostics、认证失败 envelope、限流失败 envelope；不要定义 Content / User / Comment DTO。

- [x] Admin README 写清 facade / orchestration 只能浅层转换，真实 mutation 仍委托 provider。

- [x] 暂不创建前端 adapter，除非对应 endpoint 已达到 `Contract 草案`。

- [x] 运行后端结构检查。

  运行：`cd zhicore-go && bash scripts/check-structure.sh`

  预期：`structure ok`。

## 任务 7：架构边界测试和最终验证

**测试立场：** TDD - import 边界和 contract 文档入口是长期防线。

**文件：**

- 新增或修改：`zhicore-frontend-vue/src/__tests__/apiBoundary.spec.ts`
- 修改：`zhicore-frontend-vue/AGENTS.md`
- 修改：`zhicore-frontend-vue/docs/architecture/frontend-engineering-guidelines.md`
- 修改：`zhicore-go/docs/README.md`
- 修改：`zhicore-go/docs/documentation-rules.md`
- 修改：`zhicore-go/docs/contracts/api-design-documentation.md`

- [x] 前端编写失败测试：`src/pages`、`src/components`、`src/layouts` 不直接 import `@/api/*`。

- [x] 实现边界测试扫描，允许 `src/features/**`、`src/stores/**` 和 `src/api/**` 使用 API adapter。

- [x] 检查后端 docs 索引是否能路由到 `docs/plan/impl-plan` 和所有新增 `api/http` README。

- [x] 运行前端验证。

  运行：

  ```bash
  cd zhicore-frontend-vue
  pnpm exec vitest run src/__tests__/apiBoundary.spec.ts src/api
  pnpm typecheck
  ```

  预期：全部通过。

- [x] 运行后端验证。

  运行：

  ```bash
  cd zhicore-go
  bash scripts/check-structure.sh
  make test-size
  ```

  预期：全部通过。

- [x] 如本轮包含 Go handler 代码变更，运行对应服务最窄 `go test`；如果只改 schema 文档，记录未跑 `go test` 的原因。

## 架构适配评估

- 计划遵守 provider owns contract：Go HTTP schema、Go HTTP struct、前端 provider adapter 和 feature workflow 的 owner 明确分开。
- 计划没有把 `src/api` 塞进 feature，也没有允许页面直接调 API；feature 是业务入口，provider adapter 是下游基础设施。
- 计划没有把服务私有 DTO 提升到 `libs/contracts`；跨服务 typed client 和事件 payload 仍按独立 contract 管理。
- 计划先做 File / Content / Auth 的最小闭环，再推进 User / Comment / Ranking，避免一次性铺开所有服务导致字段级 schema 质量下降。
- 计划对 Gateway / Admin 明确禁止复制 provider DTO，避免 facade 变成第二事实源。

## 交付说明

- 本计划是正式实现计划，执行时按任务顺序推进。
- 每个服务切片完成后，先更新对应 checkbox，再进入下一切片。
- 每个可运行切片都要留下最小验证证据；没有跑全量测试时，不得声称全仓回归已完成。
