# Ranking 页面设计

本文记录 Ranking 相关页面的前端草稿、页面分区、接口编排、加载状态和降级规则。它是产品 / 前端交互设计事实源，不替代 HTTP 字段级 contract；字段、错误码和 path 仍以 `services/zhicore-ranking/api/http/` 为准。

当前状态：本文固定页面初设计和加载逻辑，不表示前端已经实现。

## 设计原则

- Ranking 页面展示的是榜单排序和分数读模型，不拥有文章、用户、话题或评论事实。
- 榜单主资源是 rank item 列表；Content / User 详情补齐是附加资源。榜单可用但详情补齐失败时，应显示榜位、分数和轻量占位，而不是整页失败。
- 公开榜单必须过滤不可见文章；页面仍要接受最终一致性导致点击后 Content 详情不可用的情况。
- Redis 榜单降级或回源慢时使用稳定骨架和局部错误，不显示虚假的空榜。
- Admin rebuild 是高风险运维动作，页面必须显示状态机、互斥状态、审计原因和可恢复查询。

## 页面范围

本文覆盖：

- 热门文章榜。
- 日榜 / 周榜 / 月榜。
- 创作者榜。
- 话题榜。
- 单篇文章排名 / 分数摘要。
- 热门候选集的管理或调试视图。
- 管理端 rebuild 页面。

本文不覆盖：

- 文章详情、文章 engagement 和评论，这些归 Content / Comment 页面设计。
- Search 热门词，这些归 Search 页面设计。

## 热门文章榜

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Ranking header                             │
│ hot posts · daily · weekly · monthly       │
├────────────────────────────────────────────┤
│ Filters: period · category · refresh       │
├────────────────────────────────────────────┤
│ Rank list                                  │
│ # · title · score · trend · stats          │
│ # · detail degraded / loading              │
├────────────────────────────────────────────┤
│ Load more / snapshot time / retry          │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 进入页面时先调用 Ranking 榜单 endpoint，获取 `postId`、rank、score、period 和必要快照。
2. 榜单主资源成功后立即渲染榜位、分数和可用快照。
3. 收集当前页 `postId`，调用 Content 批量详情接口补齐标题、摘要、封面、作者快照和可见性。
4. Content 补齐成功后更新对应榜单项；过期响应丢弃。
5. 如果需要当前用户 engagement，必须在 Content 补齐确认文章可用后，再按 Content `batch-status` 规则加载。
6. 分页或 period 切换时清空旧榜单的补齐 pending，避免把旧详情写入新榜单。

### 状态处理

| 场景 | 页面行为 |
| --- | --- |
| Ranking 主资源失败 | 页面级错误和重试。 |
| Ranking 返回空榜 | 显示真实空榜态和最近更新时间。 |
| Ranking 降级但有 snapshot | 显示 snapshot 和降级提示。 |
| Content 批量补齐失败 | 榜单继续展示，详情字段显示占位或 Ranking 快照。 |
| 某文章 Content 不可用 | 当前行显示内容不可用，不请求 engagement。 |
| Engagement 失败 | 当前行互动状态 unknown，不影响榜位和详情。 |

## 周期榜和筛选

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Period ranking                             │
├────────────────────────────────────────────┤
│ Segmented: daily · weekly · monthly        │
│ date picker / week picker / month picker   │
├────────────────────────────────────────────┤
│ Rank table                                 │
│ rank · entity · score · delta              │
└────────────────────────────────────────────┘
```

规则：

- 周榜使用 ISO week-based year 和 week number；UI 需要显示清晰的周范围。
- 切换周期后重新加载 Ranking 主资源。
- 旧周期响应必须丢弃。
- 周期榜没有数据时显示真实空态，不展示热门总榜兜底，以免混淆榜单语义。

## 创作者榜和话题榜

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Creator / topic ranking                    │
├────────────────────────────────────────────┤
│ Tabs: creators · topics                    │
├────────────────────────────────────────────┤
│ rank · avatar/topic · name · score · trend │
│ ...                                        │
└────────────────────────────────────────────┘
```

加载逻辑：

1. Ranking 先返回创作者或话题 ID、rank、score 和快照。
2. 创作者详情通过 User 批量摘要补齐；User 失败时使用 Ranking 快照或 ID 占位。
3. 话题如果未来有 Topic 服务，则通过 Topic 补齐；当前可使用 Ranking 快照。
4. 点击创作者进入 User 主页，点击话题进入对应话题 / 标签页，由目标页面重新加载主资源。

## 单篇文章排名摘要

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Post ranking summary                       │
│ hot rank · score · daily / weekly / month  │
│ snapshot time · degraded state             │
└────────────────────────────────────────────┘
```

该组件可嵌入 Content 详情页。加载规则：

- Content 文章详情成功后再请求单篇文章 rank / score。
- Ranking 失败只隐藏或降级排名摘要，不影响文章阅读和 engagement。
- 文章不可用时不请求 Ranking 摘要。

## 热门候选集调试视图

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Hot candidates                             │
├────────────────────────────────────────────┤
│ Filters: source · window · limit           │
├────────────────────────────────────────────┤
│ candidate · score · reason · updated       │
└────────────────────────────────────────────┘
```

热门候选集主要服务 Comment 等下游缓存判定，不是前端公开分页榜单的替代品。普通用户页面默认不展示候选集调试信息；如后台开放，只显示必要字段和更新时间。

## Admin rebuild 页面

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Ranking rebuild                            │
├────────────────────────────────────────────┤
│ Current lock / active operation            │
├────────────────────────────────────────────┤
│ Form: scope · reason · dry-run             │
│ [Start rebuild]                            │
├────────────────────────────────────────────┤
│ Operation status · progress · errors       │
└────────────────────────────────────────────┘
```

加载逻辑：

1. 进入页面加载当前 active rebuild operation。
2. 启动 rebuild 前要求填写原因，并展示会暂停或影响 live ingestion 的范围。
3. 调用 Admin facade 或 Ranking admin endpoint 返回 `operationId`。
4. 页面按 operation status 轮询，展示 `PROCESSING`、`SUCCEEDED`、`FAILED`、`CANCELED`。
5. 操作失败时保留错误码、可重试建议和审计信息。

管理页面不能只靠前端禁用按钮防重复；后端 lock / barrier 是事实，页面只展示其状态。

## 跨服务页面约定

- Ranking 页面先加载榜单，再补 Content / User 详情；不要让 Content 补齐成为榜单主资源成功的前置条件。
- Content 文章不可用时，当前榜单项不再请求 engagement。
- Search 页面不能复用 Ranking 热榜作为热门搜索词。
- Admin rebuild 状态进入 Notification 或 Ops 面板时，只展示操作摘要；rebuild 事实仍归 Ranking。
