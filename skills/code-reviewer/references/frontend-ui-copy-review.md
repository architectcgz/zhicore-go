# Frontend UI Copy Review

## Review Trigger

Use this when a frontend diff adds or changes visible UI text in templates, component props, empty states, dashboard/workspace copy, helper text, or page descriptions.

## Blocking Concern

Flag visible copy that explains implementation, layout, feature purpose, or obvious navigation flow instead of serving as terminal-user product text.

Examples to block:

- "先从运维赛事目录中选择具体比赛，再进入单场运维台查看轮次、流量、大屏和榜单。"
- "进入具体赛事后查看轮次、流量、大屏和实时榜单"
- "本页面用于..."
- "这里展示..."
- "点击左侧..."
- "支持按..."

These usually belong in docs, assistant explanations, code comments, or tests, not visible operational UI.

## Acceptable Copy

Do not block concise product labels and actionable states:

- page and section titles
- field labels
- buttons and menu item labels
- status badges
- concrete validation messages
- short empty-state recovery text

## Review Heuristic

If removing the sentence does not reduce the user's ability to act because the title, table columns, button labels, and state already communicate the workflow, ask for removal.

Phrase the finding as a UI quality issue, not a style preference: explanatory prose in operational chrome increases noise, ages poorly when features change, and violates the product-copy boundary.
