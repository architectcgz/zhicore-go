# Auth 页面设计

本文记录 Auth 相关页面的前端草稿、页面分区、接口编排、加载状态和降级规则。它是产品 / 前端交互设计事实源，不替代 HTTP 字段级 contract；字段、错误码和 path 仍以 `services/zhicore-auth/api/http/` 为准。

当前状态：本文固定页面初设计和加载逻辑，不表示前端已经实现。

## 设计原则

- Auth 页面优先保护凭证和会话安全，错误提示必须稳定、可理解，但不能泄露账号是否存在、密码校验细节、token、Redis key 或内部安全状态。
- 登录、注册、refresh、logout 和 session revoke 都是安全动作；按钮需要明确 pending 状态，重复提交必须被 UI 阻断。
- `refresh_token` 只通过 HttpOnly cookie 传输，前端页面不读取、不展示、不缓存。
- CSRF token 是会话变更请求的附加防护，不是登录态本身；CSRF 获取失败只影响需要写 session 的动作。
- 安全操作返回 `202 PROCESSING` 时，页面显示处理中和轮询入口，不能承诺旧 token 已立即失效。
- Auth 页面不做营销式 hero。登录 / 注册是任务型表单，安全设置是密集但清晰的账户管理界面。

## 页面范围

本文覆盖：

- 登录页。
- 注册页。
- 登录恢复和 refresh 处理中状态。
- 当前认证主体加载。
- 账户安全设置页。
- 会话列表与撤销会话。
- 登出和安全操作处理中页。

本文不覆盖：

- 用户公开资料、头像和关注关系，这些归 User 页面设计。
- 管理端账号禁用 / 启用，这些归 Admin 页面设计，Auth 只提供被委托的安全能力。

## 登录页

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Auth top bar · brand · help                │
├────────────────────────────────────────────┤
│ Login form                                 │
│ email                                      │
│ password                                   │
│ [Login]        register · forgot password  │
├────────────────────────────────────────────┤
│ Inline error / locked / rate limited       │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 页面进入时不主动调用 `GET /api/v1/auth/me`；如果应用壳已经有有效登录态，应由路由守卫跳转离开登录页。
2. 用户提交表单前只做本地必填、email 格式和密码长度提示。
3. 提交时调用 `POST /api/v1/auth/login`。
4. 成功后保存 `accessToken` 到前端认证状态，`refresh_token` 和 `csrf_token` 由 cookie 承载；前端只使用响应中的 `csrfToken` 初始化会话变更请求头。
5. 成功后调用或复用 `principal`，跳转到登录前目标页。

### 状态处理

| 场景 | 页面行为 |
| --- | --- |
| `2003` 凭证错误 | 表单级错误，文案不区分邮箱不存在或密码错误。 |
| `2004` / `2019` | 表单级阻断，提示账号不可用，提供帮助入口。 |
| `2014` | 显示锁定状态和可重试时间，不允许持续提交。 |
| `2015` | 显示限流提示，按钮进入短暂禁用倒计时。 |
| `1004` | 显示服务暂不可用，保留输入，允许稍后重试。 |
| 网络超时 | 显示网络异常，保留输入，不清空密码外的字段。 |

登录成功前不要进入全屏 loading；提交按钮使用 loading，表单仍保持稳定尺寸。

## 注册页

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Auth top bar · back to login               │
├────────────────────────────────────────────┤
│ Register form                              │
│ email · password · confirm password        │
│ nickname / optional profile seed           │
│ [Create account]                           │
├────────────────────────────────────────────┤
│ Pending retry / field errors               │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 本地校验 email、password policy 和确认密码一致。
2. 调用 `POST /api/v1/auth/register`。
3. 注册成功且返回登录态时，按登录成功逻辑初始化前端认证状态。
4. 如果后端返回注册 pending 或 profile 初始化待补偿，页面显示“账号创建处理中”，按后端建议进行轮询或允许用户稍后登录。

### 状态处理

| 场景 | 页面行为 |
| --- | --- |
| `2009` | email 字段错误，提示已被占用。 |
| `2010` / `2011` | 字段级错误，保留其他输入。 |
| `2012` | 显示注册处理中或可重试状态，不重复创建账号。 |
| `2015` | 表单级限流提示。 |
| `1004` | 服务暂不可用，保留输入。 |

注册页不能在未知状态下引导用户反复提交同一 email；如果服务返回 pending，应优先给出恢复路径。

## 登录恢复和当前主体

应用壳启动时由全局登录态管理执行恢复，不由每个页面重复实现。

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ App shell                                  │
├────────────────────────────────────────────┤
│ Session restore skeleton                   │
│ checking principal / refreshing token      │
├────────────────────────────────────────────┤
│ Continue as guest / retry                  │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 若本地有未过期 access token，先调用需要登录的主页面接口或 `GET /api/v1/auth/me` 校验当前主体。
2. 若 access token 过期且存在 refresh cookie，调用 `POST /api/v1/auth/refresh`。
3. refresh 成功后更新 `accessToken`、`principal` 和 `csrfToken`。
4. refresh 返回会话失效、replay 或账号不可用时，清理前端登录态并进入登录页。
5. `me` 或 refresh 因服务降级失败时，显示恢复失败提示；不要把用户错误地显示为已登录。

`AUTH_PRINCIPAL_UNAVAILABLE` 必须被视为未知主体，不允许继续执行需要明确身份的写操作。

## 账户安全设置页

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Settings nav: profile · security           │
├────────────────────────────────────────────┤
│ Account summary                            │
│ email · roles · account status             │
├────────────────────────────────────────────┤
│ Sessions                                   │
│ current device · other devices · revoke    │
├────────────────────────────────────────────┤
│ Security actions                           │
│ logout · revoke current · refresh status   │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 进入页面时并行加载 `GET /api/v1/auth/me` 和 `GET /api/v1/auth/sessions`。
2. `me` 失败且为未登录时跳转登录；`me` 服务降级时页面显示主体不可确认，不允许撤销其他会话。
3. `sessions` 失败时只降级会话列表，账户摘要仍可展示。
4. 执行撤销会话前先确保已获取 CSRF token；缺失时调用 `GET /api/v1/auth/csrf`。
5. 当前会话撤销或 logout 成功后，清理本地登录态并跳转登录页或公开首页。

### 状态处理

| 场景 | 页面行为 |
| --- | --- |
| `sessions` 加载中 | 表格骨架，保留账户摘要。 |
| `sessions` 为空 | 显示仅当前设备或暂无其他设备。 |
| 撤销其他会话 pending | 仅目标行按钮 loading，其他行可读。 |
| `202 PROCESSING` | 目标行显示处理中，按 `operationId` 轮询。 |
| CSRF 失败 | 提示刷新页面或重新登录，不重复提交。 |
| `1004` | 显示安全状态暂不可用，阻断高风险动作。 |

## 登出和安全操作处理中页

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Security operation                         │
├────────────────────────────────────────────┤
│ status: processing / succeeded / failed    │
│ operation id                               │
│ [Retry status] [Back to login]             │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. `POST /api/v1/auth/logout` 返回 `200` 时立即清理前端登录态。
2. 返回 `202` 时也清理本地登录态，但进入安全操作处理中页或 toast，按 `retryAfterSeconds` 查询 `GET /api/v1/auth/security-operations/{operationId}`。
3. `SUCCEEDED` 后提示安全操作完成；`FAILED` 后提示服务端撤销状态未完全确认，并提供重新登录 / 联系支持入口。

前端可以本地退出，但不能在 `202` 或 `FAILED` 时声明所有旧 access token 已服务端失效。

## 跨页面交互约定

- 需要登录的业务页遇到 `2006` 时跳转登录，并携带返回地址。
- 需要权限的业务页遇到 `2005` / `2007` 时显示权限不足，不自动跳登录。
- 需要明确安全状态的页面遇到 `1004` 或 `2016` 时 fail closed，不展示可提交的危险动作。
- 登录态变化后，Content、Comment、Message、Notification 等页面必须重新加载 viewer 状态和用户相关数据。
