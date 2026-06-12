# Code-Workflow Plan 路径说明

## 当前硬编码

code-workflow 的共享脚本中硬编码了 `docs/plan/impl-plan/` 作为正式实施计划的默认位置。

涉及的文件：
- `managed/start-implementation.sh`：创建 plan 时使用
- `managed/archive-task-artifacts.sh`：归档 plan 时使用
- `managed/check-task-group-dependencies.sh`：检查 task group 依赖时使用

## 设计考虑

code-workflow 只负责**正式实施计划**的工作流，不处理探索性计划。所以硬编码 `docs/plan/impl-plan/` 是合理的。

探索性计划 (`docs/plan/exploratory/`) 不进入 code-workflow，由 writing-plans skill 直接写入，不绑定 task gate。

## 项目覆盖

如果未来有项目需要将正式实施计划放在其他位置（不是 `docs/plan/impl-plan/`），可以：

1. 在项目 `.harness/config.json` 中配置：
   ```json
   {
     "formal_impl_plan_dir": "docs/plans/formal/"
   }
   ```

2. code-workflow 脚本读取该配置并覆盖默认路径

当前暂不实现项目覆盖，因为 `docs/plan/impl-plan/` 作为约定已经足够清晰。

## 总结

- **正式实施计划**：`docs/plan/impl-plan/` (code-workflow 负责)
- **探索性计划**：`docs/plan/exploratory/` (writing-plans 负责)
- code-workflow 不需要修改，因为它只处理正式计划
