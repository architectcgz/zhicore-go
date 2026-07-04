# Docker

这里放 Go 服务的 Docker 构建和本地编排资产。`docker-compose.yml` 只提供本地开发依赖，不承担生产部署职责。

## 快速开始

```bash
# 启动本地开发依赖
docker compose -f deploy/docker/docker-compose.yml up -d

# 加载本地开发依赖环境变量
source deploy/docker/local.example.env

# 查看运行状态
docker compose -f deploy/docker/docker-compose.yml ps

# 停止并保留数据卷
docker compose -f deploy/docker/docker-compose.yml down

# 停止并清理本地数据卷
docker compose -f deploy/docker/docker-compose.yml down -v
```

启动后：

| 服务 | 地址 | 默认账号 | 说明 |
| --- | --- | --- | --- |
| PostgreSQL | `localhost:5432` | `zhicore` / `zhicore` | 各 Go 服务独立 database。 |
| Redis | `localhost:6379` | 无密码 | 缓存、限流、投影和锁。 |
| RabbitMQ AMQP | `localhost:5672` | `zhicore` / `zhicore` | 跨服务事件 broker。 |
| RabbitMQ Management | `http://localhost:15672` | `zhicore` / `zhicore` | RabbitMQ 管理界面。 |
| MongoDB | `localhost:27017` | `zhicore` / `zhicore` | Content 正文、Ranking 冷归档。 |
| Elasticsearch | `http://localhost:9200` | 无认证 | Search 索引，本地禁用 security。 |
| MinIO API | `http://localhost:9000` | `zhicore` / `zhicore123` | File service 本地对象存储。 |
| MinIO Console | `http://localhost:9001` | `zhicore` / `zhicore123` | MinIO 管理界面。 |

## PostgreSQL 数据库

首次初始化 `postgres_data` 数据卷时，`postgres/init-databases.sql` 会创建以下本地数据库：

| 服务 | database | DSN 示例 |
| --- | --- | --- |
| Gateway | `zhicore_gateway` | `postgres://zhicore:zhicore@localhost:5432/zhicore_gateway?sslmode=disable` |
| Auth | `zhicore_auth` | `postgres://zhicore:zhicore@localhost:5432/zhicore_auth?sslmode=disable` |
| User | `zhicore_user` | `postgres://zhicore:zhicore@localhost:5432/zhicore_user?sslmode=disable` |
| Content | `zhicore_content` | `postgres://zhicore:zhicore@localhost:5432/zhicore_content?sslmode=disable` |
| Comment | `zhicore_comment` | `postgres://zhicore:zhicore@localhost:5432/zhicore_comment?sslmode=disable` |
| Message | `zhicore_message` | `postgres://zhicore:zhicore@localhost:5432/zhicore_message?sslmode=disable` |
| Notification | `zhicore_notification` | `postgres://zhicore:zhicore@localhost:5432/zhicore_notification?sslmode=disable` |
| Search | `zhicore_search` | `postgres://zhicore:zhicore@localhost:5432/zhicore_search?sslmode=disable` |
| Ranking | `zhicore_ranking` | `postgres://zhicore:zhicore@localhost:5432/zhicore_ranking?sslmode=disable` |
| Admin | `zhicore_admin` | `postgres://zhicore:zhicore@localhost:5432/zhicore_admin?sslmode=disable` |
| File | `zhicore_file` | `postgres://zhicore:zhicore@localhost:5432/zhicore_file?sslmode=disable` |
| ID Generator | `zhicore_id_generator` | `postgres://zhicore:zhicore@localhost:5432/zhicore_id_generator?sslmode=disable` |
| Ops | `zhicore_ops` | `postgres://zhicore:zhicore@localhost:5432/zhicore_ops?sslmode=disable` |

`docker-entrypoint-initdb.d` 只在 PostgreSQL 数据卷首次创建时执行。新增 database 后如果本地已有旧数据卷，需要执行 `docker compose -f deploy/docker/docker-compose.yml down -v` 后重建，或手动连接 PostgreSQL 创建对应 database。

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
# PostgreSQL（每个服务使用自己的 database）
export ZHICORE_<SERVICE>_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/<service_db>?sslmode=disable

# Redis
export ZHICORE_<SERVICE>_REDIS_ADDR=localhost:6379

# RabbitMQ
export ZHICORE_<SERVICE>_RABBITMQ_URL=amqp://zhicore:zhicore@localhost:5672/

# MongoDB（Content / Ranking）
export ZHICORE_<SERVICE>_MONGO_URI=mongodb://zhicore:zhicore@localhost:27017/<service_db>?authSource=admin

# Elasticsearch（Search）
export ZHICORE_SEARCH_ES_URL=http://localhost:9200

# MinIO / S3（File）
export ZHICORE_FILE_S3_ENDPOINT=http://localhost:9000
export ZHICORE_FILE_S3_ACCESS_KEY=zhicore
export ZHICORE_FILE_S3_SECRET_KEY=zhicore123
export ZHICORE_FILE_S3_BUCKET=zhicore-files
export ZHICORE_FILE_S3_USE_SSL=false
```

其中 `<SERVICE>` 使用全大写服务名，连字符转下划线；`<service_db>` 按服务替换，例如：

```bash
export ZHICORE_AUTH_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/zhicore_auth?sslmode=disable
export ZHICORE_CONTENT_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/zhicore_content?sslmode=disable
export ZHICORE_USER_POSTGRES_DSN=postgres://zhicore:zhicore@localhost:5432/zhicore_user?sslmode=disable
export ZHICORE_CONTENT_MONGO_URI=mongodb://zhicore:zhicore@localhost:27017/zhicore_content?authSource=admin
export ZHICORE_RANKING_MONGO_URI=mongodb://zhicore:zhicore@localhost:27017/zhicore_ranking?authSource=admin
```

## MinIO 初始化

`minio-init` 会在 MinIO ready 后创建本地 bucket：

```text
zhicore-files
```

File service 后续应只通过自己的 MinIO / S3 adapter 访问对象存储。User、Content、Comment 等业务服务只保存 `fileId` 引用，不直接读写 MinIO。

## MongoDB 初始化

`mongo/init-databases.js` 会创建本地开发所需集合：

| database | collections |
| --- | --- |
| `zhicore_content` | `post_bodies`、`content_body_cleanup_tasks`、`content_body_repair_tasks` |
| `zhicore_ranking` | `ranking_archives` |

应用本地连接使用 `authSource=admin`，账号密码均为本地示例值 `zhicore`。

## 后续

每个服务实现后，应至少补齐：

- 服务镜像构建方式（Dockerfile）
- 服务自身容器编排（加入 docker-compose 或独立 compose 文件）
- 环境变量示例文件 `services/<service>/configs/local.example.env`
- Go 服务本地开发端口约定
