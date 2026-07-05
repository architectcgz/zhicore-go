# zhicore-content

`zhicore-content` 是内容服务的 Go 目标服务模块。

## 服务职责

- 拥有文章、草稿、文章内容、发布生命周期、定时发布、删除恢复、标签、分类和话题引用。
- 拥有文章点赞、收藏、统计、作者快照和内容服务内部读模型。
- 发布内容相关事件，供 Search、Ranking、Notification、Comment 等服务消费。

## 当前实现状态

已完成 Content 发布闭环 foundation：

| 方法 | 路径 | 状态 |
| --- | --- | --- |
| `POST` | `/api/v1/posts` | 已实现并由 handler contract test 覆盖 |
| `PUT` | `/api/v1/posts/{postId}/draft/body` | 已实现并由 handler contract test 覆盖 |
| `POST` | `/api/v1/posts/{postId}/publish` | 已实现并由 handler contract test 覆盖 |
| `GET` | `/api/v1/posts/{postId}/body` | 已实现并由 handler contract test 覆盖 |

当前 `cmd/server` 只建立进程根和 runtime 装配边界；真实配置加载、PostgreSQL / MongoDB / RabbitMQ / User / File 依赖打开、HTTP server listen / shutdown 和真实 readiness 仍待后续切片实现。

## 数据归属

- `posts`
- `content_body_cleanup_tasks`
- `content_body_repair_tasks`
- `post_stats`
- `post_likes`
- `post_favorites`
- `categories`
- `tags`
- `post_tags`
- `tag_stats`
- `scheduled_publish_event`
- `outbox_event`
- `outbox_retry_audit`
- `consumed_events`
- `domain_event_task`

## Go 设计注意点

- 用户资料归 User，`posts` 中的作者昵称和头像只是 Content 拥有的快照。
- 文件资源归 File service，Content 只保存 `file_id`。
- 查询某个用户发表的文章由 Content 提供权威查询；当前不提供 User facade，用户主页直接调用 Content 作者过滤接口。

## 验证

发布闭环 foundation 已通过：

```bash
(cd services/zhicore-content && go test -count=1 ./...)
python3 scripts/check-test-size.py --files services/zhicore-content
bash scripts/check-structure.sh
make check
```

真实 PostgreSQL migration 已用隔离容器执行 `up -> down 1 -> up`；真实 MongoDB 端到端和黑盒 HTTP system test 仍待补。

## 后续计划

- 发布闭环 foundation 计划：`docs/plan/archive/impl-plan/2026-07-05-content-publish-foundation-implementation-plan.md`
- 模块补全计划：`docs/plan/impl-plan/2026-07-05-content-module-completion-implementation-plan.md`
