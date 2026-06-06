# Example Module Layout

将 `example` 替换成真实模块名后，建议保留这一层次：

- `api/http/`
  - transport adapter，处理请求/响应映射
- `application/commands/`
  - 写用例与状态变更
- `application/queries/`
  - 读用例
- `contracts/`
  - 对外 DTO / contract model
- `domain/`
  - 领域规则、状态机、值对象
- `entity/`
  - 稳定业务实体表达
- `infrastructure/`
  - 仓储、缓存、第三方依赖适配
- `ports/`
  - consumer-side ports
- `runtime/`
  - composition wiring
- `testsupport/`
  - 模块测试帮助代码

生成后建议先处理这几个点：

- 把示例 `Repository` 从内存实现替换成真实 persistence
- 保留 `TxRunner` 边界，不要让 command service 直接依赖 GORM 事务细节
- 再按项目需要补 `testsupport/`、`shared/`、跨模块 contract
