# Docker

这里放 Go 服务的 Docker 构建和本地编排资产。

## 快速开始

```bash
# 启动本地开发依赖（RabbitMQ + PostgreSQL）
docker compose -f deploy/docker/docker-compose.yml up -d

# 查看运行状态
docker compose -f deploy/docker/docker-compose.yml ps

# 停止
docker compose -f deploy/docker/docker-compose.yml down
```

启动后：

| 服务 | 地址 | 说明 |
| --- | --- | --- |
| RabbitMQ AMQP | `localhost:5672` | broker 连接地址 |
| RabbitMQ Management | `http://localhost:15672` | 管理界面，用户名密码均为 `zhicore` |
| PostgreSQL | `localhost:5432` | 数据库，用户名密码均为 `zhicore` |

## RabbitMQ 拓扑管理

启动时自动加载 `rabbitmq/definitions.json` 初始化以下资源：

- **Exchange**：`zhicore.events`（topic）、`zhicore.dlx`（dead-letter）
- **Queue**：按 `docs/contracts/events.md` 中的 Consumer 命名创建的 8 个 consumer queue，每个 queue 绑定 `zhicore.dlx` 作为 dead-letter exchange
- **Binding**：各 queue 到 `zhicore.events` 的 routing key 绑定

修改拓扑的流程：

1. 编辑 `rabbitmq/definitions.json`（新增 queue、修改 binding、调整 DLX 参数等）
2. 同步更新 `docs/contracts/events.md` 中的 Consumer 命名清单
3. 重启 RabbitMQ 容器加载新 definitions：
   ```bash
   docker compose -f deploy/docker/docker-compose.yml restart rabbitmq
   ```

## 服务连接配置

各服务本地开发时使用以下环境变量连接依赖：

```bash
# RabbitMQ
export ZHICORE_<SERVICE>_RABBITMQ_URL=amqp://zhicore:zhicore@localhost:5672/

# PostgreSQL（每个服务使用自己的 database）
export ZHICORE_<SERVICE>_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/<service_db>?sslmode=disable
```

其中 `<SERVICE>` 和 `<service_db>` 按服务替换，例如：

```bash
export ZHICORE_CONTENT_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/zhicore_content?sslmode=disable
export ZHICORE_USER_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/zhicore_user?sslmode=disable
```

## 后续

每个服务实现后，应至少补齐：

- 服务镜像构建方式（Dockerfile）
- 服务自身容器编排（加入 docker-compose 或独立 compose 文件）
- 环境变量示例文件 `services/<service>/configs/local.example.env`
- Go 服务本地开发端口约定
