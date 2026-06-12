---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code
---

# Writing Plans

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm using the writing-plans skill to create the implementation plan."

**Context:** This should be run in a dedicated worktree (created by brainstorming skill).

**Save plans to the appropriate directory based on plan type.**

Resolve the plan directory in this order:

1. **Default (exploratory plans)**: `docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`
   - Use this for quick drafts, technical exploration, prototyping, and temporary investigations
   - Does not require project to declare this location
   - Short lifecycle, can be deleted after completion

2. **Formal implementation plans**: Only use the project-defined formal plan location if ALL of these conditions are met:
   - The project explicitly defines a formal implementation plan location (e.g., via `<!-- FORMAL_IMPL_PLAN_DIR: docs/plan/impl-plan/ -->` marker in `AGENTS.md`, or explicit `formal_impl_plan_location` field)
   - The task is structural, cross-module, or requires formal review and task gate binding
   - The plan will be tracked through code-workflow with a task slug and startup gate

3. **Fallback for projects with existing `docs/plan/` but no explicit formal marker**: 
   - If the project has a `docs/plan/` directory and `docs/plan/README.md` that clearly distinguishes formal vs exploratory plans, follow that structure
   - Otherwise, default to `docs/superpowers/plans/` for safety

**Decision criteria:**

Ask yourself: "Is this a formal, structural change that will go through code-workflow with task gates and formal review?"
- **Yes** → Use project's formal plan location (if defined)
- **No** → Use `docs/superpowers/plans/`
- **Unsure** → Use `docs/superpowers/plans/` and mention that it can be promoted to formal plan if needed

Always report the actual saved path. When using `docs/superpowers/plans/`, briefly note: "This is an exploratory plan. If it evolves into a formal implementation, it should be promoted to the project's formal plan directory."

## Scope Check

If the spec covers multiple independent subsystems, it should have been broken into sub-project specs during brainstorming. If it wasn't, suggest breaking this into separate plans — one per subsystem. Each plan should produce working, testable software on its own.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

Every executable step must be represented by a checkbox. The executor is required to flip each checkbox from `- [ ]` to `- [x]` immediately after the step's expected result is reached, before continuing to later steps. Plans should make this easy by keeping steps small and objectively verifiable.

## Plan Document Header

**Every plan MUST start with this header:**

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

---
```

## Task Structure

````markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit this slice according to the target repository policy**

Check the repository's `AGENTS.md`, `CLAUDE.md`, or commit policy before writing the command. Include task metadata when the project requires it, use the project's required message shape, and do not copy a generic one-line commit example into repositories with stricter rules.
````

## Remember
- Exact file paths always
- Complete code in plan (not "add validation")
- Exact commands with expected output
- Reference relevant skills with @ syntax
- DRY, YAGNI, TDD, frequent commits

## Plan Review Loop

After writing the complete plan:

1. Run an explicit architecture-fit evaluation on the written plan before any implementation handoff. Check:
   - whether the target architecture boundary is explicit
   - whether shared layers, owners, reuse points, and abstraction landing zones are named
   - whether the plan is only aligning output behavior while quietly deferring structural convergence
   - whether following the plan would predictably cause an immediate second-round redesign after "completion"
   - if structural convergence is intentionally deferred, whether it is captured as its own tracked task with completion criteria
   If any answer is unclear, revise the plan first.
2. Dispatch a single plan-document-reviewer subagent (see plan-document-reviewer-prompt.md) with precisely crafted review context — never your session history. This keeps the reviewer focused on the plan, not your thought process.
   - Provide: path to the plan document, path to spec document
   - Default model: `gpt-5.5` with `medium`
3. If ❌ Issues Found: fix the issues, re-dispatch reviewer for the whole plan
4. If ✅ Approved: proceed to execution handoff

**Review loop guidance:**
- Same agent that wrote the plan fixes it (preserves context)
- If loop exceeds 3 iterations, surface to human for guidance
- Reviewers are advisory — explain disagreements if you believe feedback is incorrect

## Execution Handoff

After saving the plan, offer execution choice:

**"Plan complete and saved to `<actual-plan-path>`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?"**

**If Subagent-Driven chosen:**
- **REQUIRED SUB-SKILL:** Use superpowers:subagent-driven-development
- Fresh subagent per task + two-stage review

**If Inline Execution chosen:**
- **REQUIRED SUB-SKILL:** Use superpowers:executing-plans
- Batch execution with checkpoints for review
