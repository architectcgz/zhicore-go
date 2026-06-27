# Auth Event Contract

本目录放 `zhicore-auth` 作为 producer 拥有的跨服务事件 payload contract。

第一阶段待固定事件：

- `auth.account.registered`
- `auth.account.disabled`
- `auth.account.enabled`
- `auth.role.changed`
- `auth.password.changed`，仅在存在明确安全审计 consumer 时发布

事件 payload 不包含 password hash、JWT、refresh token、Authorization header 或完整请求 body。
