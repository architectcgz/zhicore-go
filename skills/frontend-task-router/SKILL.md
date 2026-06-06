---
name: frontend-task-router
description: Use when the user asks for frontend optimization, UI polish, visual cleanup, responsive fixes, design QA, component refinement, or equivalent Chinese requests such as 前端优化, 页面打磨, 样式统一, 响应式修复, and the assistant should route work to specific frontend skills instead of giving generic advice.
---

# Frontend Task Router

## Overview

This skill routes broad frontend requests to exact skills that are already installed.

Use it when the user's wording is vague, broad, or overloaded:
- "optimize the frontend"
- "polish this page"
- "make the UI better"
- "fix the layout"
- "improve the design"
- "做前端优化"
- "把这个页面打磨一下"
- "统一一下样式"
- "修一下响应式"
- "帮我看看这个页面该怎么优化"

The goal is to stop generic frontend guidance and choose concrete skills with clear scope.

If the request is primarily about implementing, refactoring, or fixing frontend code behavior rather than visual polish, route to `frontend-engineer` instead of keeping the work inside design or polish skills.

## Core Rule

Do not stay at the level of "I will improve the frontend."

Translate the request into one or more exact skills. Prefer the impeccable/Codex skills below before falling back to generic frontend skills.

When you choose a route, name the exact skill or skills you are using in your commentary so the user can see the decision.

## Chinese Trigger Phrases

Treat the following Chinese phrasing as strong signals for this router:

- `前端优化` -> usually `$audit` first, then targeted fix skills, then `$polish`
- `页面打磨` or `页面精修` -> usually `$polish`, possibly with `$typeset` or `$arrange`
- `样式统一` or `设计系统统一` -> `$normalize`
- `视觉升级` or `界面改版` -> `$frontend-design` or `$bolder`
- `排版优化` or `层级优化` -> `$typeset`
- `布局优化` or `间距调整` -> `$arrange`
- `响应式修复` or `移动端适配` -> `$adapt`
- `动效优化` or `加一点动画` -> `$animate`
- `配色优化` or `颜色看起来不对` -> `$colorize`
- `统一成 workspace 风格` / `和平台概览保持一致` / `目录页风格对齐` / `工具栏变量化` in this CTF repo -> `$ctf-ui-theme-system`
- `设计审查` or `UI 走查` -> `$audit` or `$critique`
- `提取公共组件` or `组件收敛` -> `$extract`
- `性能优化` when clearly about frontend experience -> `$optimize`

If the Chinese request mixes several concerns, split it into a small skill chain instead of picking one generic frontend label.

## Routing Table

Use the first matching route as the default, then add a second or third skill only if the request clearly spans multiple concerns.

| User intent | Primary skill | Add when needed |
| --- | --- | --- |
| Implement or refactor frontend code behavior, async flows, component logic, or shared UI state | `$frontend-engineer` | `$harden`, `$audit` |
| Align a CTF page in this repo to the established workspace hero, directory list, toolbar, or theme patterns | `$ctf-ui-theme-system` | `$ctf-dark-surface-alignment`, `$frontend-engineer` |
| Build a new page, section, landing page, or strong visual UI | `$frontend-design` | `$teach-impeccable`, `$polish` |
| Gather product/brand/audience design context before UI work | `$teach-impeccable` | `$frontend-design` |
| Audit a page for design, accessibility, responsiveness, or frontend quality issues | `$audit` | `$normalize`, `$polish` |
| Standardize styles, tokens, spacing, consistency, or design-system drift | `$normalize` | `$extract`, `$polish` |
| Final ship pass, detail cleanup, or visual refinement | `$polish` | `$typeset`, `$arrange` |
| Improve typography, hierarchy, sizing, or font choices | `$typeset` | `$polish` |
| Fix layout, spacing, rhythm, grouping, or composition | `$arrange` | `$adapt`, `$polish` |
| Fix mobile behavior, breakpoints, or responsive adaptation | `$adapt` | `$arrange`, `$audit` |
| Add or improve motion and transitions | `$animate` | `$polish` |
| Improve color palette, contrast, or visual energy | `$colorize` | `$polish` |
| Make the design bolder or less generic | `$bolder` | `$colorize`, `$frontend-design` |
| Tone down an overdesigned UI | `$quieter` | `$polish` |
| Simplify a cluttered UI | `$distill` | `$arrange`, `$polish` |
| Critique UX, hierarchy, clarity, or emotional tone | `$critique` | `$normalize` |
| Improve loading, errors, empty states, edge cases, or UX hardening | `$harden` | `$polish` |
| Improve performance of a frontend experience | `$optimize` | `$audit` |
| Extract reusable components or patterns | `$extract` | `$normalize` |
| Improve onboarding or first-run experience | `$onboard` | `$clarify`, `$polish` |
| Improve copy clarity, labels, or UX writing | `$clarify` | `$polish` |
| Push visuals or interactions further than normal polish | `$overdrive` | `$animate`, `$colorize` |

## Default Recipes

### Broad "frontend optimization"

Route to:
1. `$audit`
2. The targeted fix skills revealed by the audit, usually one or more of `$normalize`, `$typeset`, `$arrange`, `$adapt`, `$colorize`, `$animate`, `$harden`, `$optimize`
3. `$polish`

Do not jump straight to implementation without first deciding what kind of optimization is actually needed.

### Broad "make this page better"

Route to:
1. `$audit` if the request is corrective
2. `$frontend-design` if the request is transformative or asks for a new look
3. `$polish` for the closing pass

### Design-system cleanup

Route to:
1. `$audit`
2. `$normalize`
3. `$extract` if repeated UI should become reusable
4. `$polish`

### Responsive cleanup

Route to:
1. `$audit`
2. `$adapt`
3. `$arrange` if spacing or composition also breaks
4. `$polish`

### Typography or layout-only request

Keep the route narrow:
- Typography only: `$typeset`
- Layout only: `$arrange`
- Color only: `$colorize`
- Motion only: `$animate`

Do not drag in `frontend-design` unless the user is asking for a broader redesign.

## Fallback Rules

- Prefer the exact impeccable skills above for design, polish, critique, and optimization work.
- Use `frontend-engineer` as the default destination for frontend implementation and refactoring, especially for framework-specific code changes.
- Use `ui-ux-designer` only when the user wants conceptual options, UX structure, or design exploration before code.
- If the user explicitly names a skill, obey that request even if another route would also work.
- If the request is not actually frontend work, do not force this router.

## Examples

- "Optimize this dashboard UI" -> `$audit` + targeted fix skills + `$polish`
- "The page feels bland" -> `$bolder` or `$frontend-design`, depending on whether it needs refinement or redesign
- "Mobile layout is broken" -> `$adapt` and likely `$arrange`
- "Clean up spacing and hierarchy" -> `$arrange` + `$typeset`
- "Make the checkout feel production-ready" -> `$audit` + `$harden` + `$polish`
- "做一下前端优化" -> `$audit` + targeted fix skills + `$polish`
- "把样式统一一下" -> `$normalize` and possibly `$extract`
- "这个页面再精修一轮" -> `$polish`, with `$typeset` or `$arrange` if needed
- "移动端有问题，顺便把布局理顺" -> `$adapt` + `$arrange`
- "这个后台太素了，做得更有设计感" -> `$bolder` or `$frontend-design`
