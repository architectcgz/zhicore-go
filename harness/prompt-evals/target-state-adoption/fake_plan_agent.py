#!/usr/bin/env python3
"""仅用于验证 plan eval runner 和评分器的确定性 fixture。"""

from __future__ import annotations

import sys


prompt = sys.stdin.read()

if "第一期只迁移 repository API" in prompt:
    print("""# 阶段边界
本期只让 GORM 承担 repository API；`sql.Open`、lib/pq 和 runtime wiring 是临时保留边界，禁止本期迁移 pgx。
# 实施步骤
迁移 repository 并补测试。
# 临时保留
记录 `sql.Open`、lib/pq 和第二期 owner 变更。
# 验证
运行 `go test ./...`。
# 退出条件
第二期完成 GORM 初始化与 pgx 切换后删除临时边界。""")
elif "用户只说：这个项目引入 GORM" in prompt:
    print("""# 已知目标
目标是减少数据库代码量，但范围尚未确定。
# 调研依据
先读取仓库代码、配置和现有实现，并核对 GORM 官方一手资料中的 PostgreSQL 初始化与连接池建议。
# 推荐方案
根据调查结果推荐先确定一个服务做完整纵向试点，因为能验证 runtime owner、事务和旧路径删除，再决定是否全面推广。
# 待确认
确认是试点、增量采用还是全面迁移，以及最终 runtime owner（运行时所有者）。
# 方案分支
试点保留明确边界；全面方案必须定义最终生产路径和旧路径删除。
# 验证
范围确认后为所选方案运行 `go test ./...`。""")
else:
    mappings = {
        "sql.Open + lib/pq": ("GORM 通过 gorm.Open(postgres.Open(dsn)) 和 pgx", "sql.Open、lib/pq、旧 constructor 初始化", "repository、transaction 事务、outbox、连接池 pool"),
        "Redux Toolkit": ("Zustand store", "Redux、Provider 和 package 依赖", "middleware、selector、初始化和测试"),
        "ts-jest": ("Vitest", "Jest、ts-jest、config 配置和 package 依赖", "setup、environment、coverage、CI shard 分片"),
        "aws-sdk v2": ("AWS SDK v3 @aws-sdk client", "aws-sdk v2 和全局 config", "S3、SQS、DynamoDB、retry 重试和 mock 测试"),
        "Express 创建 server": ("Fastify 作为生产 runtime owner 入口", "Express、middleware 和 package 依赖", "plugin、error handler、request context、OpenAPI 和 integration 集成测试"),
        "logrus logger": ("slog handler", "logrus、formatter、hook 和 module 依赖", "context field 字段、hook 迁移和测试"),
        "生产数据库是 MySQL": ("PostgreSQL schema", "MySQL driver 驱动和生产连接配置路径", "数据迁移、CDC 增量同步、reconciliation 校验和 cutover 切换"),
        "Docker Compose 部署": ("Kubernetes Deployment、Service", "生产 Compose 和 production 部署脚本", "readiness、liveness、Secret、PVC volume 和 rollout 滚动"),
    }
    target, removals, details = next(value for key, value in mappings.items() if key in prompt)
    print(f"""# 目标状态
{target} 是唯一生产路径和 runtime owner；覆盖 {details}。
# 实施步骤
逐模块迁移初始化、调用点、配置与测试，最后切换生产入口。
# 删除清单
删除 {removals}，并用源码搜索证明旧路径不可达。
# 验证
运行 `make test`、`make check` 和构建命令确认正向采用与负向清除。
# 回退
切换前保留可恢复制品或数据快照；失败时按已验证步骤恢复上一稳定版本。""")
