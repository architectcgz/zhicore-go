# 技术债：Content 事件正确性必需 worker 被建模为默认关闭的可选开关

状态：未处理
优先级：高
负责人：未分配
来源：2026-07-10 Content 模块补全独立 review 后续追问

## 影响

Content 的四个后台 worker（`content-body-cleanup`、`content-body-repair`、`content-engagement-stats`、`content-outbox-dispatcher`）共用同一套「默认 `false` 的可选开关」建模：

- `DefaultContentServerConfig()` 未设置任何 `Workers.*Enabled`，Go 零值使四个开关默认 `false`。
- `ZHICORE_CONTENT_WORKERS_*_ENABLED` 是 optional env 覆盖，不显式设置即保持关闭。
- `services/zhicore-content/configs/local.example.env` 与 `deploy/docker/zhicore/docker-compose.yml` 都把 outbox 显式设为 `false`。
- `deploy/` 下没有独立的 Content worker 进程或 deployment，outbox 并非「由独立进程运行」的有意拆分。

问题在于这四个 worker 的「强依赖」程度不同，却被一视同仁：

- `content-outbox-dispatcher` 是事务性发件箱的投递端，是 Content 对外发布领域事件的**唯一出口**。关闭它时，发布文章仍会在发布事务内写入 outbox 表（拦不住），但事件永远不会投递，在表中无限堆积，下游 Search 建索引、Ranking 热榜、Notification 通知、Comment 等全线静默停更。
- `content-engagement-stats` 关闭时，点赞/收藏动作成功但 `post_stats` 计数永不更新，功能退化为用户可见的 bug。
- `content-body-cleanup` / `content-body-repair` 是真正的可选运维/诊断能力，做成开关合理。

净效果：按当前示例与编排启动，四个 worker 全部关闭，服务 HTTP 全绿、`/health/ready` 报 ready、文章可发布，但**没有任何事件对外投递、互动计数不更新**，且没有任何机制阻止或告警这种错误配置。`internal/content/runtime/module.go:176` 只保护了「开启 outbox 却缺 publisher」的反向情形，不保护「关闭 outbox 仍照常接收发布请求」。

附带配置不一致：`deploy/docker/zhicore/docker-compose.yml` 只列出 cleanup/repair/outbox 三个开关，遗漏 `ZHICORE_CONTENT_WORKERS_ENGAGEMENT_STATS_ENABLED`；虽靠零值兜底为 false，但 compose 与代码的四个开关未对齐。

本条与 [content-worker-lifecycle-no-restart.md](content-worker-lifecycle-no-restart.md) 是不同层面：那条是 worker 启用后遇瞬时错误静默永久退出；本条是「事件正确性必需的 worker 被默认关闭且允许静默错误配置」，修复方式不同（改默认值 / 加启动校验，而非改重启逻辑）。

## 退出条件

- 明确 `content-outbox-dispatcher`（及 `content-engagement-stats`）是否应默认开启，或在关闭这类「事件正确性必需」worker 时于启动阶段给出显式告警或拒绝启动，避免「文章可发但事件永不外发」的静默错误配置。
- 保留 `content-body-cleanup` / `content-body-repair` 作为可选运维开关。
- 补齐 `deploy/docker/zhicore/docker-compose.yml` 缺失的 `ZHICORE_CONTENT_WORKERS_ENGAGEMENT_STATS_ENABLED`，使 compose 与代码四个开关对齐。
- 在服务 README 或 runtime 文档写清每个 worker 的依赖等级（正确性必需 vs 可选运维），说明默认值取舍。

## 备注

- 关键代码：`cmd/server/config_defaults.go`（无 Workers 默认）、`cmd/server/config_loader.go:373-407`（optional env 覆盖）、`internal/content/runtime/module.go:176`（仅反向保护）、`internal/content/runtime/module.go:243-296`（按开关点亮 descriptor）。
- 部署证据：`services/zhicore-content/configs/local.example.env:101-104`、`deploy/docker/zhicore/docker-compose.yml:64-66`。
- 确认无独立 Content worker 进程/deployment：`deploy/` 下仅 `docker/docker-compose.yml` 与 `docker/zhicore/docker-compose.yml`。
