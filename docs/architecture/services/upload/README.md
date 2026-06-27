# Upload 服务设计

## 事实来源

- Java `zhicore-upload` controller：`FileUploadController`。
- `03-file-upload-architecture.md`
- `file-service-integration.md` 和 `file-service-data-flow.md`

## 职责边界

`zhicore-upload` 是 ZhiCore 的统一文件入口，负责接收上传、文件类型和大小校验、访问级别参数转换、批量上传、URL 获取和删除代理。

Upload 不拥有业务实体和文件关联关系。用户头像、文章封面、评论媒体等引用关系归对应业务服务。

## API 保留范围

必须保留：

- `POST /api/v1/upload/image`
- `POST /api/v1/upload/audio`
- `POST /api/v1/upload/image/with-access`
- `POST /api/v1/upload/images/batch`
- `GET /api/v1/upload/file/{fileId}/url`
- `DELETE /api/v1/upload/file/{fileId}`

## 数据归属

当前 Go 目标第一阶段不默认引入 Upload 自有文件元数据表。

文件元数据、秒传、分片、对象存储路径和 CDN URL 归外部 File Service。Upload 只做统一入口和 adapter。

如果未来 Upload 需要拥有自己的文件元数据表，必须作为独立架构变更重新定义数据归属。

## 文件规则

目标规则承接当前上传入口约定：

- 图片：JPEG、PNG、GIF、WebP，大小限制按 Java 配置迁移。
- 音频：MP3、WAV、OGG，大小限制按 Java 配置迁移。
- 支持 PUBLIC/PRIVATE 访问级别。
- 批量上传需要明确部分失败语义。

## 跨服务依赖

- 外部 File Service：上传、删除、URL 解析、文件元数据。
- User/Content/Comment：保存文件引用和业务权限。

## Go 目标落点

- HTTP：`services/zhicore-upload/api/http`
- Application：`services/zhicore-upload/internal/upload/application`
- Domain：`services/zhicore-upload/internal/upload/domain`
- Ports：`services/zhicore-upload/internal/upload/ports`
- Infrastructure：`clients`
- Runtime：`services/zhicore-upload/internal/upload/runtime/module.go`

## 实现风险

- Multipart 请求字段、响应封装和错误码必须保持兼容，否则前端上传会直接失败。
- 临时文件清理必须可靠，避免大文件上传后占满磁盘。
- 删除文件必须考虑业务服务已经删除实体但外部 File Service 删除失败的补偿。

## 下一步

- 提取 Upload multipart contract。
- 明确 File Service Go client port。
- 写图片/音频验证、批量上传、删除失败补偿测试。
