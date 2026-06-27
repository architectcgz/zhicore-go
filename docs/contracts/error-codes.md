# 错误码表

本文件记录 `zhicore-go` 对外响应 `body.code` 的项目级错误码规范。错误响应 envelope、HTTP status 映射和字段级校验形态见 `docs/contracts/errors.md`。

## 来源与定位

- 本文件是 Go 项目公开错误码的事实源。
- 既有错误编号和语义可参考 `../zhicore-microservice` 中的 `ResultCode`、`ApiResponse`、`GlobalExceptionHandler` 和 Upload 内部错误标识，但 Go 服务公开错误码以本文件和服务级 HTTP schema 为准。

## 使用规则

- Go 服务新增或重写 endpoint 时，优先按本文件选择 `body.code`。
- 只有某个 endpoint 明确要求承接已发布历史行为时，才在服务级 HTTP schema 中登记兼容例外。
- 服务级 HTTP schema 必须从本表中选择该服务公开的错误码子集，并补充 endpoint 级触发条件。
- HTTP status 风格数字只作为历史例外保留；Go 新增错误优先使用 `1xxx` 到 `8xxx` 业务错误码。
- Upload 的 `UPLOAD_001` 这类字符串是旧实现内部错误标识，不作为 Go 对外 `body.code`。

## 范围归属

| 范围 | 归属 | 说明 |
| --- | --- | --- |
| `200` | 成功 | ZhiCore envelope 成功值。 |
| `400`-`503` | HTTP status 风格例外码 | 只在服务级 contract 明确登记时使用。 |
| `1xxx` | 通用错误 | 参数、内部错误、降级、通用业务拒绝。 |
| `2xxx` | 认证授权 | token、登录、角色、资源访问。 |
| `3xxx` | User | 用户、关注、拉黑、签到。 |
| `4xxx` | Content | 文章、分类、点赞、收藏。 |
| `5xxx` | Comment | 评论、回复、评论点赞。 |
| `6xxx` | Message | 私信、会话、撤回、发信限制。 |
| `7xxx` | Notification | 通知。 |
| `8xxx` | Upload | 文件上传。 |

## HTTP 风格例外码

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `200` | `SUCCESS` | 操作成功 | 成功响应固定值。 |
| `400` | `BAD_REQUEST` | 请求参数错误 | 不作为 Go 新增错误默认值；参数错误优先用 `1001`。 |
| `401` | `UNAUTHORIZED` | 未授权 | 不作为 Go 新增错误默认值；登录态错误优先选 `2001`、`2002` 或 `2006`。 |
| `403` | `FORBIDDEN` | 禁止访问 | 不作为 Go 新增错误默认值；权限错误优先用 `2005` 或 `2008`。 |
| `404` | `NOT_FOUND` | 资源不存在 | 不作为 Go 新增错误默认值；有服务专属码时优先选专属码。 |
| `405` | `METHOD_NOT_ALLOWED` | 请求方法不允许 | 可用于框架层 method mismatch；业务错误不要用它。 |
| `409` | `CONFLICT` | 资源冲突 | 可用于通用乐观锁、幂等冲突或状态冲突；有业务专属码时优先选专属码。 |
| `429` | `TOO_MANY_REQUESTS` | 请求过于频繁 | 可用于网关或限流层；业务频控可用 `1003`。 |
| `500` | `FAIL` | 操作失败 | 不作为 Go 新增错误默认值；内部错误优先用 `1000` 或 `1007`。 |
| `503` | `SERVICE_UNAVAILABLE` | 服务不可用 | 可用于运行期依赖不可用；业务降级可用 `1004`。 |

## 通用错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `1000` | `INTERNAL_ERROR` | 服务器内部错误 | 未分类内部错误；不要暴露底层错误细节。 |
| `1001` | `PARAM_ERROR` | 参数校验失败 | 参数缺失、格式错误、枚举非法、分页游标非法等通用参数错误。 |
| `1002` | `PARAM_MISSING` | 缺少必要参数 | 需要区分“缺少参数”时使用；否则可归并到 `1001`。 |
| `1003` | `REQUEST_TOO_FREQUENT` | 请求过于频繁 | 业务层频控；网关或限流层的历史接口可能仍用 `429`。 |
| `1004` | `SERVICE_DEGRADED` | 服务暂时不可用 | 下游降级、fallback、核心依赖短暂不可用。 |
| `1005` | `DATA_NOT_FOUND` | 数据不存在 | 通用数据不存在；有服务专属 not found 时优先用专属码。 |
| `1006` | `DATA_ALREADY_EXISTS` | 数据已存在 | 通用重复数据；有服务专属 already exists 时优先用专属码。 |
| `1007` | `OPERATION_FAILED` | 操作失败 | 通用业务操作失败。 |
| `1008` | `OPERATION_NOT_ALLOWED` | 操作不允许 | 状态不允许、业务规则拒绝或非法操作。 |

## 认证授权错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `2001` | `TOKEN_INVALID` | Token无效 | token 解析失败、签名无效或 refresh token 无效。 |
| `2002` | `TOKEN_EXPIRED` | Token已过期 | token 过期。 |
| `2003` | `LOGIN_FAILED` | 登录失败 | 用户名/邮箱/密码不匹配。 |
| `2004` | `ACCOUNT_DISABLED` | 账号已禁用 | 登录或业务操作遇到禁用账号。 |
| `2005` | `PERMISSION_DENIED` | 权限不足 | 已登录但权限不足。 |
| `2006` | `LOGIN_REQUIRED` | 请先登录 | 未登录或登录态缺失。 |
| `2007` | `ROLE_REQUIRED` | 需要特定角色 | 需要管理员或特定角色。 |
| `2008` | `RESOURCE_ACCESS_DENIED` | 无权访问该资源 | 已登录但无权访问目标资源。 |

## User 错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `3001` | `USER_NOT_FOUND` | 用户不存在 | 用户查询、管理、关注、拉黑、签到等目标用户不存在。 |
| `3002` | `USER_ALREADY_EXISTS` | 用户已存在 | 用户重复创建。 |
| `3003` | `PASSWORD_ERROR` | 密码错误 | 密码校验失败；登录接口优先按服务级 schema 选择是否归并到 `2003`。 |
| `3004` | `EMAIL_ALREADY_EXISTS` | 邮箱已被注册 | 注册或资料修改时邮箱冲突。 |
| `3005` | `USERNAME_ALREADY_EXISTS` | 用户名已被使用 | 注册或资料修改时用户名冲突。 |
| `3006` | `USER_DISABLED` | 用户已被禁用 | 用户已禁用且不可执行业务操作。 |
| `3007` | `FOLLOW_SELF_NOT_ALLOWED` | 不能关注自己 | 关注自己。 |
| `3008` | `ALREADY_FOLLOWED` | 已经关注 | 重复关注。 |
| `3009` | `NOT_FOLLOWED` | 尚未关注 | 取消关注但关系不存在。 |
| `3010` | `USER_BLOCKED` | 用户已被拉黑 | 目标用户处于拉黑关系。 |
| `3011` | `BLOCK_SELF_NOT_ALLOWED` | 不能拉黑自己 | 拉黑自己。 |
| `3012` | `ALREADY_CHECKED_IN` | 今日已签到 | 重复签到。 |

## Content 错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `4001` | `POST_NOT_FOUND` | 文章不存在 | 文章不存在或不可见时的专属错误。 |
| `4002` | `POST_ALREADY_PUBLISHED` | 文章已发布 | 重复发布。 |
| `4003` | `POST_NOT_PUBLISHED` | 文章未发布 | 需要已发布状态但当前未发布。 |
| `4004` | `POST_ALREADY_DELETED` | 文章已删除 | 操作已删除文章。 |
| `4005` | `POST_TITLE_EMPTY` | 文章标题不能为空 | 标题为空。 |
| `4006` | `POST_CONTENT_EMPTY` | 文章内容不能为空 | 内容为空。 |
| `4007` | `POST_TITLE_TOO_LONG` | 文章标题过长 | 标题超过限制。 |
| `4008` | `POST_ALREADY_LIKED` | 已点赞该文章 | 重复点赞。 |
| `4009` | `POST_NOT_LIKED` | 未点赞该文章 | 取消点赞但未点赞。 |
| `4010` | `POST_ALREADY_FAVORITED` | 已收藏该文章 | 重复收藏。 |
| `4011` | `POST_NOT_FAVORITED` | 未收藏该文章 | 取消收藏但未收藏。 |
| `4012` | `CATEGORY_NOT_FOUND` | 分类不存在 | 分类不存在。 |
| `4013` | `BODY_SCHEMA_INVALID` | i18n: `content.body.schema_invalid` | 正文 blocks schema 不合法；字段级详情放在 `data.details[].code`。 |
| `4014` | `BLOCK_TYPE_NOT_ENABLED` | i18n: `content.body.block_type_not_enabled` | 请求包含当前阶段未启用的 block 类型，例如 `mention`、`poll`、`custom_widget`。 |
| `4015` | `BODY_TOO_LARGE` | i18n: `content.body.too_large` | 正文超过单篇大小限制。 |
| `4016` | `BODY_TEXT_TOO_SHORT` | i18n: `content.body.text_too_short` | 发布时正文有效文本不足，例如普通文章少于 10 个有效 rune。 |
| `4017` | `DRAFT_CONFLICT` | i18n: `content.draft.conflict` | `post_version`、`draft_body_id` 或 `draft_body_hash` 不匹配，草稿已被其他保存或发布修改。 |
| `4018` | `CONTENT_BODY_UNAVAILABLE` | i18n: `content.body.unavailable` | `published_body_id` 指向的 MongoDB 正文不可读；应创建 repair task 并告警。 |
| `4019` | `CONTENT_BODY_INCONSISTENT` | i18n: `content.body.inconsistent` | PostgreSQL body hash 与 MongoDB body hash 不一致，或发布前 draft body 不可信。 |
| `4020` | `EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED` | i18n: `content.body.external_embed_provider_not_allowed` | `external_embed` provider 不在白名单内。 |
| `4021` | `MEDIA_REF_INVALID` | i18n: `content.body.media_ref_invalid` | 正文中的 Upload `file_id`、附件、图片或媒体引用格式/权限/状态不合法。 |
| `4022` | `VALIDATION_ERROR_LIMIT_EXCEEDED` | i18n: `validation.error_limit_exceeded` | 字段级或 block 级校验错误超过返回上限。 |
| `4023` | `COVER_UNAVAILABLE` | i18n: `content.cover.unavailable` | 草稿封面引用已经不可用或不可发布；封面非必填，为空不触发该错误。 |
| `4024` | `BODY_SCHEMA_UNSUPPORTED` | i18n: `content.body.schema_unsupported` | MongoDB body 的 `schemaVersion` 当前服务不可读。 |

## Comment 错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `5001` | `COMMENT_NOT_FOUND` | 评论不存在 | 评论不存在。 |
| `5002` | `COMMENT_ALREADY_DELETED` | 评论已删除 | 操作已删除评论。 |
| `5003` | `COMMENT_CONTENT_EMPTY` | 评论内容不能为空 | 评论文本、图片、语音等内容整体为空。 |
| `5004` | `COMMENT_CONTENT_TOO_LONG` | 评论内容过长 | 评论文本超过限制。 |
| `5005` | `ROOT_COMMENT_NOT_FOUND` | 根评论不存在 | 回复目标根评论不存在。 |
| `5006` | `REPLY_TO_COMMENT_NOT_FOUND` | 被回复的评论不存在 | 被回复评论不存在。 |
| `5007` | `COMMENT_ALREADY_LIKED` | 已点赞该评论 | 重复点赞评论。 |
| `5008` | `COMMENT_NOT_LIKED` | 未点赞该评论 | 取消点赞但未点赞评论。 |

## Message 错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `6001` | `MESSAGE_NOT_FOUND` | 消息不存在 | 消息不存在。 |
| `6002` | `CONVERSATION_NOT_FOUND` | 会话不存在 | 会话不存在或当前用户不可见。 |
| `6003` | `MESSAGE_ALREADY_RECALLED` | 消息已撤回 | 重复撤回。 |
| `6004` | `MESSAGE_RECALL_TIMEOUT` | 消息发送超过2分钟，无法撤回 | 撤回窗口超时。 |
| `6005` | `MESSAGE_CONTENT_EMPTY` | 消息内容不能为空 | 私信内容为空。 |
| `6006` | `MESSAGE_CONTENT_TOO_LONG` | 消息内容过长 | 私信内容超过限制。 |
| `6007` | `CANNOT_MESSAGE_SELF` | 不能给自己发消息 | 给自己发私信。 |
| `6008` | `USER_BLOCKED_CANNOT_MESSAGE` | 对方已将你拉黑，无法发送消息 | 拉黑关系导致不能发信。 |

## Notification 错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `7001` | `NOTIFICATION_NOT_FOUND` | 通知不存在 | 通知不存在或当前用户不可见。 |

## Upload 错误

| code | 历史 symbol | 默认 message | Go 使用 |
| --- | --- | --- | --- |
| `8001` | `FILE_TOO_LARGE` | 文件过大 | 文件大小超过业务限制。 |
| `8002` | `FILE_TYPE_NOT_ALLOWED` | 文件类型不允许 | 文件 MIME type 或扩展名不允许。 |
| `8003` | `UPLOAD_FAILED` | 上传失败 | 文件服务上传、删除、分片或哈希等失败的对外上传错误。 |

## Upload 内部错误标识

这些标识来自既有 Upload 内部错误分类，用于日志、异常分类或内部映射。Go 对外响应不要把这些字符串放进 `body.code`；如需暴露给调用方，必须先映射到上面的数字码并写入服务级 HTTP schema。

| internal code | 含义 | 建议对外数字码 |
| --- | --- | --- |
| `UPLOAD_001` | 文件不能为空 | `1001`。 |
| `UPLOAD_002` | 不支持的文件类型 | `8002`。 |
| `UPLOAD_003` | 文件大小超过限制 | `8001`。 |
| `UPLOAD_004` | 文件哈希计算失败 | `8003` 或 `1000`。 |
| `UPLOAD_005` | 文件上传失败 | `8003`。 |
| `UPLOAD_006` | 文件服务不可用 | `503` 或 `1004`，按 endpoint 兼容要求选择。 |
| `UPLOAD_007` | 文件删除失败 | `8003` 或 `1007`。 |
| `UPLOAD_008` | 文件不存在 | `404`；如果后续新增文件专属 not found 码，再独立登记。 |
| `UPLOAD_009` | 上传任务不存在 | `404`；如果后续新增上传任务专属码，再独立登记。 |
| `UPLOAD_010` | 分片上传初始化失败 | `8003`。 |
| `UPLOAD_011` | 分片上传失败 | `8003`。 |
| `UPLOAD_012` | 网络超时 | `503` 或 `1004`，按 endpoint 兼容要求选择。 |
| `UPLOAD_999` | 系统内部错误 | `1000`。 |
