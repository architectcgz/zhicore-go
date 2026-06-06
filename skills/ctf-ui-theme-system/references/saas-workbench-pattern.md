# SaaS Workspace Pattern

Read this file when a backend-style page is trying to split "overview" and "list" into parallel tabs, or when an admin workspace needs a dense but low-friction "top metrics, bottom data grid" structure.

This pattern is the default for CTF admin pages such as contest management, user governance, image management, instance management, and similar operational directories.

## Core Position

- Treat overview metrics and data directories as complementary layers of the same workspace, not as mutually exclusive views.
- Prefer one continuous operational page:
  page header -> key metrics -> seamless directory section.
- Use vertical page flow before introducing route-local tabs.

## Anti-Patterns

### 1. Pseudo-parallel overview/list tabs

- Symptom:
  The page exposes tabs such as "总览" and "列表/目录", forcing the operator to switch between seeing the big picture and touching concrete records.
- Why it is wrong:
  Metrics and rows are parent-child information, not competing work modes.
- Fix:
  Merge them into the same page and let the list live directly under the KPI strip.

### 2. Floating entry blocks

- Symptom:
  The overview area contains a standalone card or callout such as "进入工作台", "去处理", or "查看运行中对象".
- Why it is wrong:
  The action is detached from the record it applies to, so the user loses context.
- Fix:
  Push the action down into the row action area for the relevant data object.

### 3. Wasted white space

- Symptom:
  The overview page uses only a few cards and leaves a large empty canvas, while the list page feels dry and contextless.
- Why it is wrong:
  The layout artificially splits information that fits comfortably in one reading flow.
- Fix:
  Use the top half of the workspace for compact KPIs and the bottom half for the working directory.

## Canonical Pattern

### Page Header

- Keep one strong page title.
- Place global actions such as "新建", "刷新", "导入" on the right side of the header.
- Use the strongest filled button only for the primary global action.

### Key Metrics

- Show 3-4 high-value KPI cards immediately below the header.
- The first glance should answer:
  how many,
  how many active,
  how many abnormal,
  what changed recently.
- Keep these cards summary-only. Do not turn them into mini dashboards.

### Seamless Data Grid

- Put the directory toolbar and data table directly below the KPI band.
- Keep the list region decardified inside the workspace shell.
- Use search, filter, sort, and pagination as lightweight toolbar islands attached to the list section.

## Data-Driven Actions

- All meaningful actions should be tied to a specific data object.
- Do not create vague global shortcuts such as "去处理进行中的赛事".
- Prefer row-level contextual actions, for example:
  a running AWD contest row exposes "进入 AWD 赛区" in its action column.
- The user should understand which object they are about to operate on before clicking.

## Action Hierarchy

### Global actions

- Live in the page header.
- Use filled brand styling for the single primary action.
- Secondary global utilities such as refresh stay ghost or outline.

### Row actions

- Live in the rightmost action column.
- Frequent core transitions may use a stronger surface treatment.
- Ordinary actions such as edit, inspect, export, and delete should stay secondary or overflowed.

## Tab Review Rule

Before adding a tab rail, ask:

- Are these views truly mutually exclusive workflows?
- Would the page become genuinely crowded if these sections were placed one above another?

If the answer is no, remove the tab rail and use one continuous workspace.

## When Tabs Are Still Legitimate

- Peer workflows with different tools and mental models:
  for example environment management, monitoring, announcements, operations.
- Views that cannot fit into one coherent vertical reading flow without becoming noisy.
- Route-level stages where each panel owns distinct actions, forms, or context.

Do not use tabs just to separate metrics from the list they describe.

## UI Review Checklist

- [ ] If the page currently has "总览" and "列表/目录" tabs, can they be merged into one workspace?
- [ ] Are there any floating entry cards that should instead become row-level actions?
- [ ] Is the first glance focused on KPI cards and the second glance naturally led into the directory?
- [ ] Is the page using vertical space efficiently instead of splitting one workflow into two sparse screens?
- [ ] Are global actions in the header and object actions in the row action column?

