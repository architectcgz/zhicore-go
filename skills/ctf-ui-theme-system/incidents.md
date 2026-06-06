# CTF UI Theme System Incidents

## 2026-04-15 - verification-gap

1. Date
   2026-04-15
2. Category
   `verification-gap`
3. Symptom
   C 端队伍弹窗把图标、标题、输入框和按钮直接铺在整屏遮罩视觉上，缺少明确的白色承载卡片，用户会误以为弹窗背景丢失或 UI 渲染异常。
4. Triggering action
   为了去掉全屏可见遮罩，只保留了透明 overlay，没有同步检查内容层是否仍然有清晰的 surface 承载。
5. Root cause
   只验证了 backdrop 行为，没有把“遮罩层”和“内容承载层”拆开审查；共享 dialog shell 缺少“显式 surface 必须存在”的 guardrail。
6. Correct rule
   所有 modal / drawer / floating panel 都必须同时满足两层约束：
   - overlay 只负责聚焦、遮罩和关闭交互
   - content 必须落在独立 surface / panel 上，明确给出背景、圆角、阴影或边框
7. Where the rule belongs
   existing skill: `ctf-ui-theme-system`
8. Verification that the guardrail was applied
   已将 modal / drawer surface 规则写入 `ctf-ui-theme-system/SKILL.md`，并在本轮新增 B 端统一弹窗壳回归测试，约束后续组件接入共享 surface shell。
