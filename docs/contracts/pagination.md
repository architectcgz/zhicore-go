# 分页契约

本文件定义分页、排序和过滤的通用规则。默认兼容迁移的已有 Java 接口保持现状；新接口和 Go-first API reset 服务按本文件设计，并在服务级 HTTP schema 中固定最终参数。

## 基本原则

- 列表接口必须明确分页模型，不能返回无限列表。
- 对外排序必须稳定；同分值或同时间必须有次级排序字段。
- 查询参数语义由 provider 拥有，consumer 不自行解释 provider 私有过滤条件。

## Page 分页

适用场景：

- 管理后台。
- 低频列表。
- 需要跳页的查询。

常用参数：

```text
page
size
sort
order
```

规则：

- 默认兼容迁移的 `page` 起始值保持 Java 当前接口语义；新接口和 Go-first API reset 服务默认从 `1` 开始。
- `size` 必须有最大值。
- `sort` 只能接受 provider 明确列出的字段。
- `order` 使用 `asc` / `desc`。

## Cursor 分页

适用场景：

- 信息流。
- 评论流。
- 排行榜滚动加载。
- 时间线和高频翻页。

常用参数：

```text
cursor
limit
direction
```

规则：

- cursor 对 consumer 不透明，不要求 consumer 解析。
- cursor 内部必须包含稳定排序锚点，例如 `created_at + id`。
- 返回结果必须包含下一页 cursor。
- 不允许只用 offset 模拟 cursor。

## 返回形态

默认兼容迁移的已有接口保持 Java DTO。新接口和 Go-first API reset 服务推荐：

```json
{
  "items": [],
  "page": 1,
  "size": 20,
  "total": 100
}
```

或 cursor：

```json
{
  "items": [],
  "nextCursor": "opaque-cursor",
  "hasMore": true
}
```

## 过滤

- 过滤字段必须列入服务级 HTTP schema。
- 时间范围使用 RFC3339 字符串。
- 多值过滤使用重复 query 参数或逗号分隔时，必须在服务 schema 中固定一种。
