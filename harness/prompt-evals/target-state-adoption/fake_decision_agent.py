#!/usr/bin/env python3
"""仅用于验证 decision eval runner 和评分器的确定性 fixture。"""

from __future__ import annotations

import json
import sys


prompt = sys.stdin.read()

if "所有服务全面改用 GORM" in prompt:
    response = {
        "choice": "A",
        "final_state": "使用 gorm.Open(postgres.Open(dsn)) 和 pgx 的唯一生产路径",
        "legacy_removals": ["删除 sql.Open 初始化", "删除 lib/pq"],
        "retained_boundaries": [],
        "reason": "全面迁移要求 runtime owner、repository、transaction 和 outbox 一起收敛。",
    }
elif "所有状态管理全面采用 Zustand" in prompt:
    response = {
        "choice": "A",
        "final_state": "Zustand 成为唯一 runtime store",
        "legacy_removals": ["删除 Redux store、Provider、middleware 和依赖"],
        "retained_boundaries": [],
        "reason": "用户明确禁止长期并存两套状态体系。",
    }
elif "全项目迁移到 slog" in prompt:
    response = {
        "choice": "A",
        "final_state": "slog 成为唯一日志实现",
        "legacy_removals": ["删除 logrus hooks、formatter 和依赖"],
        "retained_boundaries": [],
        "reason": "调用点和底层输出必须同时迁移。",
    }
elif "第一期只把 repository API 改成 GORM" in prompt:
    response = {
        "choice": "B",
        "final_state": "第一期由 GORM 承担 repository API",
        "legacy_removals": [],
        "retained_boundaries": [
            "临时保留 sql.Open、lib/pq 和现有 runtime wiring；第二期迁移 pgx 后退出"
        ],
        "reason": "尊重用户明确的分期边界，并记录临时层退出条件。",
    }
else:
    raise SystemExit("unsupported fixture prompt")

json.dump(response, sys.stdout, ensure_ascii=False)
sys.stdout.write("\n")
