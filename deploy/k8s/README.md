# Kubernetes

这里放 Go 服务的 Kubernetes 部署资产。

后续每个服务上 K8s 前，应明确：

- `Deployment`、`Service`、`ConfigMap`、`Secret` 的归属
- `readinessProbe` 和 `livenessProbe`
- CPU/内存资源限制
- 滚动发布和回滚策略
- Go 服务的路由和服务发现方式
