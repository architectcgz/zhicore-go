# zhicore-admin

`zhicore-admin` 是管理和审核编排服务的 Go 迁移模块。

服务职责：

- 拥有举报、举报处理流程和审核审计日志。
- 提供管理端查询和命令 facade。
- 调用 User、Content、Comment 等归属服务完成真实业务操作。

数据归属：

- `reports`
- `audit_logs`

迁移注意点：

- Admin 不直接拥有用户、文章或评论的业务状态。
- 禁用用户、删除文章、删除评论等操作必须委托给对应归属服务。
- Admin 本地只记录审核原因、处理结果和审计轨迹。
