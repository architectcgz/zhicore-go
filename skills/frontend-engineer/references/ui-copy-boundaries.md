# UI Copy Boundaries

## When to Read

Read this before adding or changing visible UI text in a page, dashboard, workspace, table header, empty state, helper block, card, dialog, or toolbar.

## Rule

Visible UI copy must be terminal-user product copy. Do not render implementation explanations, layout descriptions, feature walkthroughs, or obvious navigation instructions in the page.

Avoid copy that explains what the screen does instead of helping the user complete a concrete action.

## Disallowed Patterns

Do not add visible text like:

- "先从运维赛事目录中选择具体比赛，再进入单场运维台查看轮次、流量、大屏和榜单。"
- "进入具体赛事后查看轮次、流量、大屏和实时榜单"
- "这里展示了..."
- "该模块用于..."
- "点击左侧二级菜单..."
- "本页面采用..."
- "支持按..."

These belong in assistant replies, docs, comments, tests, or product documentation when truly needed, not in operational UI chrome.

## Allowed Patterns

Use concise labels, state, actions, and field-specific help:

- Page title: "赛事运维"
- Section title: "竞赛列表"
- Button: "进入运维台"
- Status: "进行中"
- Empty state title: "当前还没有可进入运维台的 AWD 赛事"
- Error recovery: "重试加载"

Empty-state descriptions are acceptable only when they tell the user the missing prerequisite or next action. Keep them specific and short.

## Review Before Shipping

Before finishing a UI change:

1. Scan new template text for explanatory prose.
2. Remove any sentence that describes layout, navigation flow, or feature purpose when the controls already make it clear.
3. Keep text that is a label, state, user action, field constraint, concrete error, or real empty-state recovery instruction.
4. If a sentence starts with "先", "进入...后", "这里", "该模块", "本页面", or "支持", challenge whether it belongs in visible UI at all.

## CTF Reference Case

The contest operations index previously rendered explanatory copy telling users to select a contest and then enter the single-contest operations workspace to view rounds, traffic, projector, and scoreboard. The page already had a "竞赛列表" heading and "进入运维台" action, so the sentences added noise without adding actionable information. The fix was to remove those visible explanations and keep the operational labels.
