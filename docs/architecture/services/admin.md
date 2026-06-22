# Admin 服务设计

## 事实来源

- Java `zhicore-admin` controller：User/Post/Comment/Report manage command/query。
- Java 全量初始化 SQL 中 `reports`、`audit_logs`。
- `service-boundaries.md` 中 Admin facade 规则。

## 职责边界

`zhicore-admin` 是管理端 facade 和审核编排服务，拥有举报处理流程、审核审计和管理端聚合查询。

Admin 不拥有用户、文章或评论业务状态。禁用用户、删除文章、删除评论必须委托对应归属服务。

## API 保留范围

必须保留以下 API 族：

- `/admin/users`：管理端用户查询、禁用、启用。
- `/admin/posts`：管理端文章查询、删除。
- `/admin/comments`：管理端评论查询、删除。
- `/admin/reports`：举报待处理查询、列表查询、处理。

Java 中这些路径没有统一 `/api/v1` 前缀，Go 迁移需要保持前端当前路径兼容。

## 数据归属

Admin 拥有：

- `reports`
- `audit_logs`

不拥有 `users`、`posts`、`comments`。

## 编排规则

管理操作流程：

1. 校验管理员身份和权限。
2. 调用归属服务 command contract，例如 User 禁用、Content 删除文章、Comment 删除评论。
3. 在 Admin 本地记录审核原因、处理结果和审计日志。
4. 返回管理端兼容响应。

如果归属服务操作失败，Admin 不应写成功审计；可以记录失败审计或操作日志。

## 跨服务依赖

- User：用户管理和用户摘要。
- Content：文章管理。
- Comment：评论管理。
- Notification：如后续需要管理通知公告，由 Notification 提供 contract。

## Go 目标落点

- HTTP：`services/zhicore-admin/api/http`
- Application：`services/zhicore-admin/internal/admin/application`
- Domain：`services/zhicore-admin/internal/admin/domain`
- Ports：`services/zhicore-admin/internal/admin/ports`
- Infrastructure：`postgres`、`clients`
- Runtime：`services/zhicore-admin/internal/admin/runtime/module.go`

## 迁移风险

- Admin facade 容易复制业务服务逻辑，必须保持“委托归属服务 + 本地审计”。
- 管理端路径兼容要单独核对，不能默认加 `/api/v1`。
- 审计日志需要记录操作者、目标、原因、结果和时间，不能只记成功状态。

## 下一步

- 提取 Admin API contract。
- 设计 reports/audit_logs migration。
- 补用户禁用、文章删除、评论删除、举报处理的编排测试。
