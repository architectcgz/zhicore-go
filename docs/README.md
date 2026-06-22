# 文档索引

本目录是 `zhicore-go` 的文档入口。

## 阅读顺序

1. 先读 `docs/documentation-rules.md`，确认文档归属和放置规则。
2. 再通过本索引找到相关事实源。
3. 修改当前事实前，用 Java 源码、Go 代码、contract、配置、测试或运维记录做核对。

## 当前事实源

- `docs/architecture/`：当前服务边界和数据归属决策。
- `docs/contracts/`：跨服务 contract 归属、兼容性、版本和变更流程。
- `docs/migration/`：Java 到 Go 的服务迁移映射、迁移顺序和发布说明。

## 流程和历史

- `docs/reviews/`：review 证据和发现。
- `docs/todos/debt/`：迁移过程中不能丢失的未解决技术债。

## 部署说明

部署资产放在 `deploy/`：

- `deploy/docker/`
- `deploy/k8s/`
