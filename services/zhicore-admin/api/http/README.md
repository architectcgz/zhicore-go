# zhicore-admin HTTP Schema

本目录记录 `zhicore-admin` 的对外 HTTP contract。当前仅做计划化占位，字段级 endpoint schema 待后续按管理端切片提取。

## Provider Owner

Admin 拥有举报处理流程、审核审计和管理端聚合查询。Admin 不拥有用户、文章或评论业务状态；真实 mutation 必须委托 User、Content、Comment、Auth 等 provider。

## 首批 endpoint 候选

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/admin/users` | 管理端用户查询 facade | API 族已识别 |
| `POST` | `/admin/users/{userId}/disable` | 委托 Auth/User 执行停用 | API 族已识别 |
| `GET` | `/admin/posts` | 管理端文章查询 facade | API 族已识别 |
| `DELETE` | `/admin/posts/{postId}` | 委托 Content 删除文章 | API 族已识别 |
| `GET` | `/admin/comments` | 管理端评论查询 facade | API 族已识别 |
| `DELETE` | `/admin/comments/{commentId}` | 委托 Comment 删除评论 | API 族已识别 |
| `GET` | `/admin/reports` | 举报列表 | API 族已识别 |
| `POST` | `/admin/reports/{reportId}/resolve` | 处理举报并记录审计 | API 族已识别 |

## Facade / Orchestration 规则

- Admin facade 只能做浅层参数转换、聚合查询和审计记录。
- 用户禁用、账号封禁、角色调整归 Auth；资料删除归 User；文章删除归 Content；评论删除归 Comment。
- 如果归属服务 mutation 失败，Admin 不得写成功审计。

## 待提取 contract

- 管理端路径兼容性，当前旧路径没有统一 `/api/v1` 前缀。
- 审计字段、处理原因、操作者、目标资源和失败响应。
- facade 返回中 provider DTO 的引用边界和浅层转换规则。

## 禁止规则

- 不复制 provider DTO 为 Admin 第二事实源。
- 不绕过 provider service 直接改业务表。
- 暂不创建前端 `src/api/admin.ts`，直到 endpoint 达到 `Contract 草案`。
