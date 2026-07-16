# Handoff Contract

## Purpose

A handoff is a current, evidence-backed map for the next maintainer. It is not a transcript, a generic README, or a list of files read.

## Minimum coverage

| Area | Minimum answer |
| --- | --- |
| Actual purpose | What the system does today, not only what its README intends |
| Entrypoints | How a caller starts, invokes, stops, and observes the system |
| Boundaries | Which module owns orchestration, domain state, persistence, integrations, and public API |
| Runtime model | Threads/processes/workers, queue topology, scheduling, synchronization, and shutdown |
| State model | Important states, transitions, persistence, deduplication, idempotency, and ordering |
| Data flow | Input → transformation → storage/index/cache → output, including side effects |
| Failure model | Exceptions, timeouts, retries, partial success, stale state, recovery, and known loss paths |
| Evidence | File/symbol/command references and whether facts were statically or runtime verified |

## Evidence levels

- **Code-confirmed**: directly supported by implementation and callers.
- **Config-confirmed**: supported by active configuration/build/deployment wiring.
- **Runtime-confirmed**: observed through a test, command, trace, log, or reproducible scenario.
- **Intent-only**: stated in README, roadmap, or design but not confirmed by implementation.
- **Unknown**: cannot be established from available local evidence.

Never upgrade intent-only or inferred concurrency/retry/transaction behavior to a current fact. State the gap explicitly.

## Completion check

Before handing off, ask:

- Can a maintainer explain one real request or background job from entrypoint to side effect?
- Can they name the thread/process that executes each asynchronous step?
- Can they identify the owner and synchronization of every mutable state in that path?
- Can they predict what happens on duplicate events, failure, restart, and shutdown?
- Can each important claim be traced to evidence, with unverified behavior labeled?
- Are diagrams accompanied by prose, tables, or state transitions that explain behavior?
