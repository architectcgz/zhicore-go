# 架构测试

这里放源码级边界检查，用于防止服务迁移过程中破坏架构约束。

典型检查包括：

- 服务之间不得导入彼此的 `internal`
- `libs/kit` 不得依赖服务私有包
- contract 不得依赖服务私有模型
- handler、application、domain、infrastructure 的依赖方向符合项目约定
