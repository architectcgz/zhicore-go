# 技术债：Upload 孤儿文件清理机制

---
category: data-integrity
severity: medium
created: 2026-06-29
affects: zhicore-upload, zhicore-content, zhicore-comment, zhicore-user
exit-condition: Upload 服务或各业务服务实现文件生命周期补偿任务，孤儿文件可被定期识别和清理
---

## 问题描述

当业务服务（Content、Comment、User）删除实体时，对应的 Upload 文件（封面、正文媒体、评论图片、语音、头像）需要通过调用 Upload service 删除对象存储资源。

当前设计中，如果实体删除成功但 Upload 文件删除失败，没有任何补偿机制：

- 无清理 task 表或 retry 队列
- 无孤儿文件统计或告警
- 无 GC worker 或对账机制

## 影响

- 对象存储空间持续增长，产生不必要的存储费用
- 用户删除的内容对应的文件仍在存储中，存在潜在的数据合规风险
- 随时间累积，孤儿文件难以事后清理（无法判断哪些 file_id 已无业务引用）

## 退出条件（满足其一即可关闭）

1. **方案A（推荐）：Upload 侧 GC**：在 Upload service 实现一个 file reference counting 机制，或支持"soft delete + delayed hard delete"，业务服务标记 file 为"可回收"，Upload 定期清理无引用文件。

2. **方案B：业务侧补偿 task**：各业务服务（Content、Comment、User）在实体删除事务内写一条 `upload_cleanup_task`，后台 worker 重试调用 Upload 删除；Upload 删除幂等（同一 file_id 多次删除返回成功）。

3. **方案C：事件驱动**：业务服务发布 `{domain}.file_reference_released` 事件，Upload 消费后删除对象；Upload 侧幂等处理重复事件。

## 当前兜底

暂无。首期上线前至少在各服务 README 中标注"文件删除失败不重试"的已知行为，并在 Ops 侧记录存储账单监控基线。

## 相关文档

- `docs/architecture/security.md` → 上传、外部 URL 和文件安全
- `docs/architecture/service-boundaries.md` → `zhicore-upload` 归属
