# 部署

`deploy/` 存放 Go 迁移过程中需要的部署资产。

- `docker/`：本地或测试环境使用的 Docker、Docker Compose 资产。
- `k8s/`：Kubernetes manifest 或 Helm chart 资产。

部署文件应服务于 Go 目标服务逐步落地：支持按服务配置目标路由、回滚到上一个 Go 版本或配置，并完成部署验证。迁移目标不规划 Java/Go 运行时并存。
