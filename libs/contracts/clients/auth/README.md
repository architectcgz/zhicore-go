# Auth Client Contract

本目录放 `zhicore-auth` 作为 provider 拥有的同步 typed client contract。

第一阶段待固定能力：

- 查询当前或指定账号认证主体。
- 查询账号状态和角色。
- 管理端账号禁用、启用和 token 全量失效命令。

这里不放 User profile DTO。昵称、头像、简介和用户摘要归 `libs/contracts/clients/user/`。
