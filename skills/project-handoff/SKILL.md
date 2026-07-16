---
name: project-handoff
description: Use when taking over an unfamiliar repository, preparing a codebase handoff, building project-understanding documentation, or analyzing runtime flows where threads, queues, state ownership, lifecycle, persistence, side effects, failure recovery, or evidence locations are easy to miss.
---

# Project Handoff

Build an evidence-backed map of an unfamiliar codebase that a maintainer can use before changing it. Treat diagrams as navigation only: every important runtime flow must also explain execution context, triggers, state ownership, synchronization, side effects, failure boundaries, observability, and unknowns.

## Use when

- taking over, understanding, onboarding to, or analyzing an unfamiliar project;
- existing architecture diagrams show relationships but not behavior;
- background workers, watchers, queues, async jobs, transactions, or external providers are involved;
- current-state documentation needs to be created, repaired, or checked against code.

Do not use it for a narrow code question whose owner and behavior are already obvious, or for a one-off session diary.

## Workflow

1. Read the repository entrypoint, documentation rules/indexes, current-state or architecture docs, build/configuration files, and the code paths named by them. Treat docs as hypotheses until checked against code.
2. Identify the main entrypoints and trace one real path end to end: caller → dispatch/queue → worker or service → persistence/external side effect → result/event.
3. For each important flow, fill the understanding-point contract from `references/understanding-point-template.md`.
4. Record evidence paths and separate confirmed facts, runtime-verified facts, assumptions, and unknowns. Do not infer a thread, retry, transaction, or delete API merely because a diagram suggests it.
5. Produce the smallest useful handoff: orientation, module boundaries, runtime/concurrency model, state/data flow, failure/recovery boundaries, risks, and recommended reading order.

## Required understanding points

Every non-trivial runtime flow must answer:

- Who executes it: caller thread, worker thread, watcher, process, scheduler, or external service?
- What triggers it, and how does it start, stop, drain, cancel, or restart?
- Who owns each mutable state, and how does in-memory state map to persisted state?
- What queue, lock, condition, transaction, idempotency key, or ordering rule protects it?
- What data and external side effects occur, and in what order?
- What happens on timeout, exception, partial success, duplicate event, or shutdown?
- What can a caller observe through logs, metrics, events, errors, or status APIs?
- Which statements are confirmed, which are unverified, and where is the evidence?

## Output shape

Use this compact structure unless the repository has a stronger local format:

1. **定位** — what the system actually does and its maturity/implementation-vs-intent gap.
2. **入口与边界** — public entrypoints, major modules, ownership, and storage locations.
3. **运行模型** — actors/threads/processes, queues, locks, lifecycle, and one concrete sequence.
4. **数据与副作用** — state transitions, persistence, external calls, and consistency boundaries.
5. **失败与风险** — retry, backpressure, shutdown, stale state, compatibility, and unverified behavior.
6. **证据索引** — exact files/symbols/commands supporting each important conclusion.

Use a table for actor/state mappings and a sequence or state diagram only when it clarifies relationships. A diagram without the corresponding behavioral explanation is incomplete.

## Reference routing

- Understanding-point fields and a copyable template → `references/understanding-point-template.md`
- Handoff document structure, evidence levels, and review checklist → `references/handoff-contract.md`
