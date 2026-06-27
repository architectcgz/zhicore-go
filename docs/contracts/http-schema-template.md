# 服务级 HTTP Schema 模板

本文件定义每个服务在 `services/<service>/api/http/` 下记录字段级 HTTP contract 的格式。API 背后设计、HTTP contract 和实现追踪的分层结构见 `docs/contracts/api-design-documentation.md`；通用 HTTP envelope、header、版本化和错误规则分别见 `docs/contracts/http.md`、`docs/contracts/errors.md`、`docs/contracts/error-codes.md`、`docs/contracts/data-types.md` 和 `docs/contracts/pagination.md`。

## 放置规则

每个已开始设计或实现 HTTP API 的服务必须维护：

```text
services/<service>/api/http/
├── README.md
└── endpoints/
    └── <operation>.md
```

规则：

- `README.md` 是该服务 HTTP schema 入口，只放服务级公共规则、endpoint 索引和来源说明。
- `endpoints/<operation>.md` 记录单个 endpoint 的字段级 contract。
- `<operation>` 使用小写短横线命名，例如 `upload-image.md`、`login.md`、`list-comments.md`。
- 默认一个文档只记录一个 endpoint。兼容别名 endpoint 可以放在同一文档中，但必须明确主路径和别名路径。
- 如果某个服务明确登记为 Go-first API reset，且一次设计需要固定完整 API 面，可以使用 `endpoints/<family>.md` 记录一个 API family 或完整服务 API 面；服务 README 必须索引该文件。后续实现单个 handler 时，可以再按 endpoint 拆出更窄文档。
- 如果 endpoint 尚未固定 Go 目标 schema，不要创建“占位完成”的 schema；在服务 README 或后续切片文档中记录待设计即可。

## 提取流程

为某个服务补 HTTP schema 时按以下顺序：

1. 读取该服务总览：`docs/architecture/services/<service>/README.md`。
2. 读取对应模块设计：`docs/architecture/module/<module>/README.md`、`api.md`、`service.md`、`domain.md`、`ports.md` 或 `data-events.md`；既有设计尚未迁移时，读取 `docs/architecture/services/<service>/` 下对应专题文档。
3. 读取通用 contract：`docs/contracts/README.md`、`docs/contracts/http.md`、`docs/contracts/errors.md`、`docs/contracts/error-codes.md`、`docs/contracts/data-types.md`。
4. 从模块设计、目标产品语义和已发布外部 contract 固定 path、method、字段、响应和错误；需要核对既有行为时再参考 Java controller、DTO、exception handler 和测试。
5. 将公共规则写入 `services/<service>/api/http/README.md`。
6. 每个 endpoint 写入独立 `endpoints/<operation>.md`。
7. 实现或修改 Go handler 前，先补对应 contract test。
8. 变更后运行最窄相关测试；脚手架或索引变更时运行 `bash scripts/check-structure.sh`。

## 服务 README 模板

“来源”默认列出服务总览、模块设计、目标 Go schema、Go handler / test 落点；需要承接已发布行为时，可附加 Java controller / DTO / 测试作为参考来源。

```markdown
# <service> HTTP Schema

本目录记录 `<service>` 的对外 HTTP contract。Go handler 实现必须以这里记录的字段级 schema 为准。

## 来源

- 服务总览：`docs/architecture/services/<service>/README.md`
- 模块设计：`docs/architecture/module/<module>/README.md`
- Go handler：`services/<service>/api/http/...`
- Go contract test：`services/<service>/...`
- 参考来源：`../zhicore-microservice/<module>/...`（仅在需要核对既有行为时填写）

## 公共规则

- 响应 envelope：见 `docs/contracts/http.md`。
- 错误码：见 `docs/contracts/error-codes.md`。
- 时间、ID、枚举、空值和 JSON 字段：见 `docs/contracts/data-types.md`。
- 分页、排序和过滤：见 `docs/contracts/pagination.md`。
- 鉴权上下文：<说明 userId、角色、匿名访问、管理员要求等服务级规则>

## Endpoint 索引

| 方法 | 路径 | 文档 | 状态 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/...` | `endpoints/<operation>.md` | 草案 / 已验证 |

## 服务级公开错误码

| code | 含义 | 适用场景 |
| --- | --- | --- |
| `1001` | 参数校验失败 | 请求字段缺失、格式错误。 |
```

## Endpoint 模板

Endpoint 来源默认列出服务总览、模块设计、当前 API schema 和 Go handler / test 落点；需要承接已发布行为时，可附加 Java 来源作为参考。

```markdown
# <operation>

## 来源

- 服务总览：`docs/architecture/services/<service>/README.md`
- 模块 API 设计：`docs/architecture/module/<module>/api.md`
- 模块 service 设计：`docs/architecture/module/<module>/service.md`
- 当前 API schema：`services/<service>/api/http/README.md`
- Go handler：`services/<service>/api/http/...`
- Go contract test：`services/<service>/...`
- 参考来源：`../zhicore-microservice/<module>/...`（仅在需要核对既有行为时填写）

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/...` |
| 兼容别名 | 无 / `/...` |
| Content-Type | `application/json` / `multipart/form-data` |
| 鉴权 | 匿名 / 登录用户 / 管理员 |
| 幂等 | 无 / `Idempotency-Key` / 业务唯一键 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 目标资源 ID。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | `1` | 页码。 |

## Body / Multipart 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `file` | file | 是 | 不允许为空 | 上传文件。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 资源 ID。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | 请求字段缺失或格式错误。 |

## 权限和可见性

- <说明资源归属、可见性过滤、管理员权限、匿名访问边界。>

## 排序、分页和过滤

- <列表接口必须说明排序稳定性、分页模型和过滤字段。非列表接口写“无”。>

## 测试要求

- Handler contract test：<测试文件或待补项>
- System HTTP test：<测试文件或待补项>
```

## 状态标记

Endpoint 文档状态只允许使用：

| 状态 | 含义 |
| --- | --- |
| 草案 | 已从服务设计、目标 Go schema 或已发布 contract 核对中提取，但尚未由 Go handler/test 验证。 |
| 已验证 | 已有 Go handler contract test 或 system HTTP test 证明。 |
| 兼容例外 | 明确保留历史 path、字段、HTTP status 或错误码例外；必须写明原因。 |
| 废弃候选 | 保留旧入口但计划切换；必须有替代 endpoint 和删除条件。 |
