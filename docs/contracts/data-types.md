# 数据类型契约

本文件定义 HTTP API、typed client contract 和事件 payload 中通用字段的序列化规则。单个服务的字段级 schema 放在 `services/<service>/api/http` 或 `libs/contracts/...`，不要堆到本文件。

## 基本原则

- 默认兼容迁移优先保持 Java 外部接口兼容；已有字段的类型、名称和空值语义不得因为 Go 重写而改变。
- 已明确登记为 Go-first API reset 的服务以服务级 HTTP schema 或 typed client schema 为准，Java 只作为业务能力参考。
- 新增字段必须默认向后兼容，旧前端和旧 consumer 能忽略。
- 默认兼容迁移的 API 字段名、事件字段名和协议字段保持原有大小写和拼写，不因 Go 命名习惯改变 JSON 形态；Go-first API reset 服务的新字段默认使用 lowerCamelCase。

## 时间

时间分两类：

| 场景 | 格式 |
| --- | --- |
| 统一响应 envelope 的 `timestamp` | Unix epoch milliseconds，保持 Java `ApiResponse` 行为 |
| 业务字段和事件字段 | RFC3339 / ISO-8601 字符串，必须带时区 |

规则：

- 新业务时间字段优先使用 UTC RFC3339，例如 `2026-06-22T10:30:00Z`。
- 读取时必须接受带 offset 的 RFC3339，例如 `2026-06-22T18:30:00+08:00`。
- 不新增无时区的本地时间字符串。
- 数据库存储优先使用 `TIMESTAMPTZ`，应用内统一按 UTC 处理。
- 会写入数据库、Redis、JSON、审计日志、事件 payload，或参与跨请求比较的业务 `time.Time`，创建时使用 UTC，输出前归一到 UTC。
- 纯运行时测量可以保留单进程内时间语义，例如耗时统计、deadline、timer、随机后缀和临时文件名；这类值不得作为业务时间返回给外部 contract。
- 事件 JSON payload 使用 lowerCamelCase 时间字段，例如 `occurredAt` / `scheduledAt`。
- 数据库、outbox 和 inbox 列名属于对应服务的 migration/schema；默认按 `docs/architecture/go-service-design.md` 的数据库命名规则处理。
- 定时任务、延迟投递和事件的发生/计划时间必须使用业务时间，不使用发送时间或消费时间替代。

## ID

内部主键策略见 `docs/architecture/id-strategy.md`。对外契约规则如下：

- 已有 Java 接口返回 `Long` / JSON number 的 ID，迁移阶段保持原类型，除非作为独立 API 演进任务处理。
- Go-first API reset 服务的外部公开资源 ID 优先使用 string；当前 Content 对外 `postId` 是 string 公开 ID，不暴露数据库内部自增主键。
- 新增对外公开资源 ID 优先使用 string，例如 `public_id`、`public_no`、`order_no`。
- 跨服务内部 contract 可以使用 `int64` / JSON number 表示内部引用 ID，但字段必须明确归属服务。
- 不把数据库自增序列的数量暴露问题交给 Base64/Base62 直接编码解决；需要隐藏时使用独立公开 ID 或可逆混淆方案。

## 枚举

- 对外枚举默认使用大写字符串，例如 `PUBLIC`、`PRIVATE`、`PUBLISHED`。
- 不使用 Go 的 `iota` 数字值作为公开枚举。
- 新增枚举值必须确认旧 consumer 的未知值处理策略；旧 consumer 不能容忍时，应增加新字段或新版本 contract。

## 空值和可选字段

- 默认兼容迁移的已有接口保持 Java 当前 JSON 行为；Go-first API reset 服务按服务级 schema 记录空值语义。
- 新字段默认设计为可选字段，旧 consumer 可忽略。
- `null` 只在“空值”和“字段缺失”有不同业务含义时使用。
- 空列表返回 `[]`，不返回 `null`。
- 空字符串不能用来表达“字段不存在”，除非旧接口已经如此定义。

## 数字和布尔值

- 计数、分页大小、游标内偏移等使用整数。
- 金额、积分或分数如果需要精确计算，不用浮点数表达；排行榜分数这种展示/排序值可以使用浮点数，但必须只由归属服务解释。
- 布尔字段使用 JSON boolean，不使用 `"true"` / `"false"` 字符串。

## 字段命名

- 默认兼容迁移的 JSON 字段沿用 Java DTO 当前字段名。
- 新增 HTTP、typed client、事件 JSON 字段以及 Go-first API reset 服务的新 schema 使用 lowerCamelCase。
- 本文件不规定 Go 内部标识符、数据库列名或 Redis key 命名；这些属于实现风格、migration/schema 或运行时设计规则。
