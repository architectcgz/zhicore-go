# 部署

`deploy/` 存放 Go 迁移过程中需要的部署资产。

- `docker/`：本地或测试环境使用的 Docker、Docker Compose 资产。
- `k8s/`：Kubernetes manifest 或 Helm chart 资产。

部署文件应服务于逐步迁移：允许 Java 和 Go 服务在一段时间内并存，并支持按服务切流、回滚和验证。
