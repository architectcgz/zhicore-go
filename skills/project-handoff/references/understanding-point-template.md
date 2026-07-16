# Understanding Point Template

Use one card for each non-trivial runtime mechanism: a worker, watcher, queue, scheduler, async request, transaction boundary, cache refresh, lifecycle transition, or external side effect.

## Mechanism

- Name:
- Owner module:
- Current fact sources:

## Behavior

- Execution context: caller / worker / watcher / scheduler / process / external service:
- Trigger:
- Inputs and outputs:
- Lifecycle: start, normal completion, stop, cancellation, restart:
- State owner: in-memory state, persisted state, and their mapping:
- Concurrency and synchronization: threads, queues, locks, conditions, transactions, ordering:
- Side effects: files, database, cache, index, network, messages, events:
- Failure behavior: timeout, exception, duplicate, partial success, backpressure, shutdown:
- Retry, idempotency, compensation, or absence of them:
- Observability: logs, metrics, events, status, error surface:

## Sequence

```text
trigger -> dispatch/queue -> processing -> persistence/side effect -> success/failure
```

## Evidence and uncertainty

- Code evidence: exact files and symbols:
- Configuration/build evidence:
- Runtime evidence: command, test, trace, or explicitly “not run”:
- Confirmed facts:
- Assumptions:
- Unknowns and follow-up checks:
