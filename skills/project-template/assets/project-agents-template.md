# Project Collaboration Rules

## Quick Routing — Task Entry Points

| Task type | Required reads | Workflow / Skill |
|-----------|---------------|------------------|
| Backend feature (API/Service/Repository) | `[backend patterns doc]` + `tests/README.md` + `harness/policies/reuse-first.yaml` | `backend-engineer` skill → `code-workflow` |
| Frontend feature (Page/Component) | `[frontend patterns doc]` + Frontend Local Rules (below) | `frontend-engineer` skill → `code-workflow` |
| Code review | `docs/文档规范.md` (review 章节) | `code-reviewer` skill |
| Bug fix (Backend) | `tests/README.md` + backend patterns | `systematic-debugging` → `backend-engineer` |
| Bug fix (Frontend) | Frontend Local Rules (below) | `systematic-debugging` → `frontend-engineer` |
| Add/Edit test | `tests/README.md` | `test-driven-development` skill |
| Architecture change | `docs/architecture/` + `brainstorming` + `writing-plans` | Plan first, then `code-workflow` |
| Documentation update | `docs/文档规範.md` | Direct edit (no worktree unless part of impl task) |
| New non-trivial task | `bash scripts/check-task-intake.sh` → `bash scripts/start-implementation.sh <topic>` | `brainstorming` → `grill-with-docs` → `writing-plans` |
| Other | Read this AGENTS.md fully, then ask user for clarification | Start with `harness-router` skill |

## Auto-Triggers — Session Discipline

- **New task in same session** → Re-read this AGENTS.md + relevant skill SKILL.md
- **Context compact/clear** → SessionStart hook reloads skill bootstrap (if configured)
- **Edit to harness/policies/\*.yaml** → PreToolUse hook blocks non-approved changes
- **Edit to AGENTS.md / docs/文档规范.md** → PreToolUse hook blocks non-approved changes
- **Task complete (non-trivial)** → Run completion validation gate, then AAR, update `feedback/` if new patterns found
- **Before commit** → Run `bash scripts/check-commit-message.sh` + relevant pre-commit checks

## Red Flags — STOP

These rationalizations mean STOP — re-read the relevant rules instead:

| Rationalization | Reality |
|----------------|---------|
| "就这一次跳过 reuse-first" | 没有例外，每次都要先搜索既有模式 |
| "时间紧，先不写测试" | 测试是完成定义的一部分，不写 = 未完成 |
| "这个改动太小，不用开 worktree" | 判断标准是"是否触达受保护 surface"，不是改动行数 |
| "我记得这个规则怎么写" | 规则会演化，必须读当前版本 |
| "Leader 让我加的，可以跳过 review" | 规则不因权威改变 |
| "用户着急，先违反一次架构约束" | 技术债是债，不是捷径 |
| "只改样式，不用看规则" | 样式改动触达共享组件 contract 时仍需遵守规则 |
| "测试太多了，删几个也没关系" | 删除测试需要明确的移除条件（见 TDD 规则） |

## 1. Scope and Inheritance

- This file defines repository-level instructions for agents working in this project.
- Follow this file after higher-priority system and user instructions.
- Keep the repository root `CLAUDE.md` as a symlink to `AGENTS.md` when Claude/Codex auto-discovery is part of the local workflow.
- Treat the global `AGENTS.md` as the default policy. Use this file only for repository-specific constraints, commands, architecture boundaries, documentation entry points, and routing rules.
- Prefer project-specific evidence over assumptions. Read the relevant source, configuration, and documentation before making implementation decisions.

## 2. Project Overview

- Project type: `[fill in: frontend / backend / full-stack / library / CLI / service / other]`
- Main language(s): `[fill in]`
- Framework(s): `[fill in]`
- Package manager / build tool: `[fill in]`
- Runtime requirements: `[fill in]`
- Primary entry points: `[fill in]`

## 3. Setup and Common Commands

Use the project's existing package manager and scripts. Do not invent commands that are not present in the repository.

```bash
# Install dependencies
[fill in]

# Start development server or local service
[fill in]

# Run tests
[fill in]

# Run lint
[fill in]

# Run typecheck
[fill in]

# Build
[fill in]
```

If a command is unavailable or dependencies are missing, report the exact blocked command and the reason.

- If this repository standardizes on a repo-scoped workflow layer such as Trellis, record the exact install and initialization commands here or in the nearest setup document. If not, remove this note instead of implying Trellis is already present.

## 4. Architecture and Change Boundaries

- Follow the existing module layout, naming conventions, layering, error handling, logging style, and test patterns.
- Describe the repository's actual code boundaries here, such as frontend/backend ownership, service layers, generated-file policy, migration flow, API compatibility rules, or directories that should normally not be edited.
- Do not introduce new abstractions, dependencies, frameworks, or architectural layers unless the current task requires them.
- Keep generated files, vendored code, build artifacts, and lockfiles consistent with the repository's existing policy.
- Do not render implementation notes, TODO markers, design explanations, or internal commentary into user-visible UI or API responses.

## 4.5. Code Quality Verification (Verification Questions)

After completing code changes, verify by asking these questions:

### Surgical Changes
- [ ] 每一行改动都能追溯到用户的请求吗？
- [ ] 这个改动是否引入了用户没要求的功能或优化？
- [ ] 如果用户说"撤销最后一个功能"，是否能干净删除？

### Avoid Premature Abstraction
- [ ] 这个抽象是为几个用例设计的？（少于 3 个 = 过早）
- [ ] 如果只有一个用例，为什么现在就要抽象？
- [ ] 这个接口是否比它要解决的问题更复杂？

### Test Quality
- [ ] 删除这个测试后，是否还能检测到相同的回归？
- [ ] 这个测试是在验证行为还是在验证实现细节？
- [ ] 这个测试失败时，错误信息是否说明了失败原因？

### Naming and Clarity
- [ ] 这个函数/变量名是否清楚说明了它的职责和副作用？
- [ ] 删除所有注释后，代码是否仍然能被6个月后的你理解？
- [ ] 这个魔法数字是否有业务含义？如果有，是否应该是常量？

### Dependencies and Coupling
- [ ] 改动这个模块会影响几个其他模块？
- [ ] 这个模块是依赖具体实现还是依赖接口？
- [ ] 添加新功能时是否需要修改现有代码？

Refer to `~/.agents/harness/docs/verification-questions-guide.md` for the complete guide on writing verification questions.

## 5. Project-Specific Verification

Use the global verification rules, then list the smallest project-specific command sequence that usually proves a change is correct.

```bash
[fill in project-specific verification sequence]
```

When the repository has tests, include an explicit testing prompt here:

- Write or update the narrowest relevant tests first.
- Treat TDD tests as maintained behavior specifications and regression guards. Do not delete them after implementation unless the behavior signal is duplicated by a clearer test, obsolete, implementation-coupled, or intentionally moved to a better owner/layer.
- Put each test at the owner/layer that proves the contract:
  - Backend: package-local tests for module semantics and unexported details; internal test utilities for helpers that need private access; system/API tests for black-box route or transport behavior; runtime/integration tests for real databases, external processes, or containers; architecture tests for boundary guardrails; testkit helpers for stable cross-test builders, fixtures, and assertions.
  - Frontend: colocated `__tests__` near shared, feature, page, store, API, router, runtime, config, or utility owners; root-level `src/__tests__` only for architecture, design-system, or cross-cutting guardrails; shared test setup and reusable helpers under the project's test support directory.
- Do not duplicate the same behavior signal across layers. If multiple layers need coverage, each layer must prove a distinct contract.
- After writing tests, run the smallest relevant test command that covers the changed surface.
- If the repository has verification scripts such as `scripts/check-*.sh`, `scripts/check-*.py`, or another documented guard command, run the relevant script check after the test command before claiming completion.
- If the repository already has `scripts/check-consistency.sh`, git hooks, CI validation, or another mechanical guardrail, wire the relevant test-related script check into that enforcement path instead of leaving it as prompt text only.
- If no mechanical enforcement path exists yet, say that explicitly and treat harness setup or guardrail wiring as still pending.

Add any repository-specific constraints here, such as required test order, snapshot update policy, browser checks, fixture refresh steps, or commands that are known to be expensive.

## 6. Git and Delivery Notes

Use the global git and worktree rules, then record only repository-specific delivery constraints here.

- Branch or commit naming convention: `[fill in if different from the global default]`
- Generated files or lockfiles that must be included or excluded: `[fill in]`
- Release, migration, or deployment coupling that affects how changes should be grouped: `[fill in]`

## 7. Documentation

- Documentation rules are owned by `docs/documentation-rules.md`; documentation navigation is owned by `docs/README.md`.
- Before creating, moving, deleting, or editing documentation, read `docs/documentation-rules.md` when it exists, then the nearest existing index for the target area.
- When code behavior, APIs, configuration, database shape, setup flow, or user-visible behavior changes, plan the documentation owner and target path before implementation.
- When the project uses recurring post-delivery review governance, keep review reports under `docs/reviews/` and unresolved technical debt under `docs/todos/debt/`.
- Keep technical debt as separate debt files plus a debt index under `docs/todos/debt/`; do not grow one root-level `DEBT.md` forever.
- Do not duplicate the full documentation taxonomy in this file. If a new durable documentation path changes agent routing, add only the route here and keep the detailed rule in the documentation owner file.
