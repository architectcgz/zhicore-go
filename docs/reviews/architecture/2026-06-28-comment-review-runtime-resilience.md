# Comment 运行韧性 review

## Review 对象

- Repository：`zhicore-go`
- Commit：`3889516 docs(comment): 固定运行韧性策略`
- Diff source：`git show 3889516`
- Review 方式：`codex review --commit 3889516 --title "docs(comment): 固定运行韧性策略"`

文件范围：

- `docs/architecture/module/comment/runtime-resilience.md`
- `docs/architecture/module/comment/decision-log.md`
- `docs/architecture/module/comment/ports.md`
- `docs/architecture/module/comment/service.md`
- `docs/architecture/module/comment/README.md`
- `docs/architecture/module/README.md`
- `docs/architecture/services/comment/README.md`
- `services/zhicore-comment/api/http/README.md`

## 分类判断

非琐碎 architecture / runtime contract 变更。改动固定 Comment 的 timeout、retry、熔断、降级、限流和健康检查语义，并触达 HTTP schema 摘要字段语义，按 `docs/reviews/done-definition.md` 需要正式 review。

## Gate Verdict

初始 review verdict：`blocked`。

修复后 verdict：`pass`。3 个 P2 material findings 已现场修复，未发现剩余 blocker / major finding。

## Findings

### Major: degraded 作者摘要缺少 `publicId` 契约不闭合

位置：`docs/architecture/module/comment/runtime-resilience.md`、`services/zhicore-comment/api/http/README.md`

User `BatchGetUserSimple` 不可用时，runtime 文档允许作者摘要降级，但 HTTP schema 原本把 `AuthorSummary.publicId` 标为绝对必填。Comment 又只持久化内部 `author_id`，不能从本地事实推导 `publicId`。如果不修正，首次实现要么违反 HTTP schema，要么伪造 User 拥有的公开标识。

修复：正常作者摘要仍必须返回 `publicId`；仅当 `author.unavailable=true` 且 User 摘要不可用、Comment 没有已确认快照时允许省略，并明确禁止伪造。`decision-log.md` 新增第 84 条固定该语义。

### Major: 取消点赞不应依赖 relation guard

位置：`docs/architecture/module/comment/runtime-resilience.md`

原 API 降级矩阵把 `LikeComment` 和 `UnlikeComment` 合并，导致 User relation 不可用时取消点赞也失败。但 `service.md` 和 decision-log 已确认取消点赞不校验拉黑关系，应允许用户撤销自己的历史点赞。否则 relation 故障会让旧互动无法解除。

修复：拆分 `LikeComment` 和 `UnlikeComment`。点赞在 relation 不可确认时 fail closed；取消点赞不调用 relation guard，PostgreSQL 不可用才失败。

### Major: `UpdateComment` 漏写 Content guard

位置：`docs/architecture/module/comment/runtime-resilience.md`

下游依赖矩阵和 decision-log 均要求创建 / 更新评论前同步校验 Content 状态，但 API 降级矩阵的 `UpdateComment` 行漏掉 Content。实现者按该行实现时，可能在 Content 不可确认时修改评论内容。

修复：`UpdateComment` 的关键依赖补充 Content，并说明不能在文章、权限和媒体事实不可确认时更新。

## Material Findings

以上 3 项均属于 material findings，已在本次修复中收口。

## Required Re-validation

- `bash scripts/check-structure.sh`
- `git diff --check`
- `bash scripts/check-commit-message.sh <message-file>`

## 验证证据

- `bash scripts/check-structure.sh`：通过，输出 `structure ok`。
- `git diff --cached --check`：通过，无输出。
- `bash scripts/check-commit-message.sh <temp-message-file>`：通过，无输出。

## Residual Risk

本次只固定设计事实源和 HTTP schema 文档，没有实现 handler、application、adapter 或 contract test。首次实现 Comment runtime 和 HTTP handler 时，仍需补对应 application test 与 handler contract test。

## 技术债状态

未触达已登记技术债；本次 review 发现的问题都属于当前 touched surface，已直接修复，不登记延期债务。
