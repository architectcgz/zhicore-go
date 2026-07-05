# contracts

`contracts` 存放跨服务契约。

- `clients/`：同步服务调用的 typed client、请求 DTO 和响应 DTO。
- `events/`：跨服务事件 payload。

契约由数据和行为的提供方拥有。例如内容服务提供的文章查询契约放在 `clients/content`，用户服务提供的用户摘要契约放在 `clients/user`。

同步 HTTP client contract 中的 endpoint path、caller operation、请求 DTO 和响应 DTO 应由 provider 目录维护。Consumer 的 `infrastructure/clients` 只实现传输、错误映射和本地 port 适配，不复制 path 字符串或 DTO 结构。

不要把服务私有领域模型、数据库实体或仓储查询条件提升到这里。只有真实跨服务使用的稳定形态才应该成为 contract。
