---
name: superpowers
description: Use when explicitly inspecting, maintaining, or explaining the Superpowers skill collection, or when choosing among Superpowers sub-skills
---

# Superpowers

Core development practices for effective AI-assisted software development. These skills form the foundation of disciplined, high-quality development workflows.

## Getting Oriented

Use this overview only when the Superpowers collection itself is relevant. For ordinary work, match the task directly against specific skill descriptions instead of treating this container as a default entrypoint.

## 📚 Core Skills

### Development Cycle & Quality

- **test-driven-development** - RED-GREEN-REFACTOR cycle for behavior-bearing features, bugfixes, and logic refactors
- **verification-before-completion** - Always verify changes work before claiming completion
- **systematic-debugging** - Root-cause analysis approach for bugs and failures

### Planning & Design

- **brainstorming** - Explore solution space and stress-test requirements before coding
- **writing-plans** - Structure implementation plans for multi-step tasks
- **executing-plans** - Execute plans in separate sessions with review checkpoints

### Code Review & Collaboration

- **requesting-code-review** - How to request effective code reviews
- **receiving-code-review** - How to handle review feedback with technical rigor

### Git & Workflow

- **using-git-worktrees** - Create isolated development environments for feature work
- **finishing-a-development-branch** - Clean branch completion with merge/PR/cleanup options

### Multi-Agent Coordination

- **subagent-driven-development** - Execute plans with independent tasks in current session
- **dispatching-parallel-agents** - Coordinate multiple agents for parallel work

### Meta-Skills

- **writing-skills** - Create and test new skills (TDD for documentation)
- **writing-plans** - Structure implementation plans

## Usage

1. **Skill routing questions**: Use `using-superpowers` when the task is about skill discovery or invocation discipline.
2. **Creative or implementation design**: Consider `brainstorming` when the task actually involves shaping new behavior.
3. **Feature or bugfix implementation**: Use `test-driven-development` when the change carries behavior or logic and test-first implementation is required by task or local policy. Do not route simple UI / presentation-only edits into TDD by default.
4. **Completion claims**: Apply `verification-before-completion` before claiming changes are complete or passing.
5. **Failures and bugs**: Follow `systematic-debugging` when investigating unexpected behavior.
6. **Skill authoring**: Use `writing-skills` when creating, editing, or validating skills unless a user instruction explicitly narrows the process.

## 📖 Philosophy

These skills enforce discipline through:
- **Explicit rules** (not suggestions)
- **Verification requirements** (evidence over assertions)
- **Test-first mindset** (RED-GREEN-REFACTOR for behavior-bearing code and docs)
- **Systematic approaches** (root-cause over quick fixes)

All superpowers skills are designed to resist rationalization under pressure. They include explicit counters to common excuses and enforce the letter of the rules, not just the spirit.
