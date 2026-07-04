# File 服务设计

## 事实来源

- 当前 Go-first HTTP contract：`services/zhicore-file/api/http/`。
- 历史 Java 文件上传 controller、`03-file-upload-architecture.md`、`file-service-integration.md` 和 `file-service-data-flow.md` 只作为业务能力参考。

## 职责边界

`zhicore-file` 是 ZhiCore 的统一文件服务，负责接收上传、文件类型和大小校验、访问级别参数转换、批量上传、文件元数据、URL 获取、对象删除和生命周期清理。

File service 不拥有业务实体和业务绑定关系。用户头像、文章封面、评论媒体等引用关系归对应业务服务；但文件对象、对象存储 key、访问 URL、签名策略、临时 / 未绑定状态和物理删除语义归 `zhicore-file`。

## API 范围

`zhicore-file` 已登记为 Go-first API reset，不保留旧 `/api/v1/upload/...` 兼容入口。当前 HTTP schema 固定以下路径：

- `POST /api/v1/files/image`
- `POST /api/v1/files/audio`
- `POST /api/v1/files/image/with-access`
- `POST /api/v1/files/images/batch`
- `GET /api/v1/files/{fileId}/url`
- `DELETE /api/v1/files/{fileId}`

## 页面设计

- [frontend pages/file.md](../../../../../zhicore-frontend-vue/docs/design/pages/file.md)：文件上传组件、批量上传、URL 解析、移除引用和媒体管理入口初设计。

## 数据归属

`zhicore-file` 拥有文件元数据、秒传、分片、对象存储路径、访问级别、CDN / 签名 URL 派生规则和文件生命周期状态。后续接入 MinIO 时，MinIO adapter 放在 `zhicore-file` infrastructure 层，业务服务不得直接读写对象存储。

## 文件规则

目标规则承接当前上传入口约定：

- 图片：JPEG、PNG、GIF、WebP，大小限制按 Java 配置迁移。
- 音频：MP3、WAV、OGG，大小限制按 Java 配置迁移。
- 支持 PUBLIC/PRIVATE 访问级别。
- 批量上传需要明确部分失败语义。

## 跨服务依赖

- User/Content/Comment：保存文件引用和业务权限。
- MinIO / 对象存储：作为基础设施 adapter 提供对象读写、删除和 URL 签名能力。

## Go 目标落点

- HTTP：`services/zhicore-file/api/http`
- Application：`services/zhicore-file/internal/file/application`
- Domain：`services/zhicore-file/internal/file/domain`
- Ports：`services/zhicore-file/internal/file/ports`
- Infrastructure：`clients`
- Runtime：`services/zhicore-file/internal/file/runtime/module.go`

## 实现风险

- Multipart 请求字段、响应封装和错误码必须和 `services/zhicore-file/api/http` 保持一致，否则前端上传会直接失败。
- 临时文件清理必须可靠，避免大文件上传后占满磁盘。
- 删除文件必须考虑业务服务已经删除实体但对象存储删除失败的补偿。

## 下一步

- 补齐 File metadata migration 和 repository 设计。
- 明确 MinIO adapter port、URL 签名策略和临时 / 未绑定 TTL。
- 写图片/音频验证、批量上传、删除失败补偿和孤儿文件清理测试。
