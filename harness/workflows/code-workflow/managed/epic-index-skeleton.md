# __EPIC_TITLE__ Epic

**Epic Slug:** `__EPIC_SLUG__`

**Goal:** TODO

**Status:** `planning` <!-- planning | in-progress | completed | archived -->

**Created:** `__CREATED_AT__`

---

## Overview

- Background:
- Motivation:
- Scope:
- Non-Goals:

## Slices

### Slice 1: __SLICE_1_TITLE__

- Task Slug: `__SLICE_1_SLUG__`
- Status: `not-started` <!-- not-started | in-progress | completed -->
- Plan: [implementation-plan](__SLICE_1_SLUG__.md)
- Depends On: 无
- Notes:

### Slice 2: __SLICE_2_TITLE__

- Task Slug: `__SLICE_2_SLUG__`
- Status: `not-started`
- Plan: [implementation-plan](__SLICE_2_SLUG__.md)
- Depends On: `__SLICE_1_SLUG__`
- Notes:

### Slice 3: __SLICE_3_TITLE__

- Task Slug: `__SLICE_3_SLUG__`
- Status: `not-started`
- Plan: [implementation-plan](__SLICE_3_SLUG__.md)
- Depends On: `__SLICE_1_SLUG__`
- Notes:

## Dependencies Graph

```
Slice 1 (基础设施)
  ├─> Slice 2 (业务层依赖基础设施)
  └─> Slice 3 (并行于 Slice 2，同样依赖基础设施)
```

## Integration Validation

整个 epic 完成后的集成验证：

- [ ] 集成测试 1:
- [ ] 集成测试 2:
- [ ] 端到端场景:
- [ ] 性能/压力验证:
- [ ] 文档完整性检查:

## Completion Criteria

Epic 视为完成的条件：

- [ ] 所有 slice 的 implementation plan 已归档
- [ ] 所有 slice 的代码已合并到 main
- [ ] 集成验证全部通过
- [ ] 架构文档已更新
- [ ] 运维文档已更新
- [ ] 无 blocker 级别的 residual risk

## Progress Tracking

| Slice | Status | Started | Completed | Notes |
|-------|--------|---------|-----------|-------|
| 1 | not-started | - | - | |
| 2 | not-started | - | - | |
| 3 | not-started | - | - | |

## Notes

- Epic-level decisions:
- Cross-slice coordination:
- Known risks:
