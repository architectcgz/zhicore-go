# 后端 Review

这里存放 Go 迁移切片的后端 review 证据。

统一完成标准见 `docs/reviews/done-definition.md`，review 记录格式见 `docs/reviews/README.md`。

记录内容应包括：

- review 对象或 commit
- 发现的问题
- 风险等级
- 修复结论
- 相关验证命令和结果

后端 review 重点：

- handler / application / repository / infrastructure 的 owner 是否清晰。
- HTTP contract、错误码、分页、字段序列化和服务级 schema 是否同步。
- migration、事务、幂等、outbox / inbox、缓存失效和 worker / consumer 生命周期是否可恢复。
- 测试是否证明真实行为，而不是只验证 mock 调用或实现细节。
