# Validation Evidence

Use this file before claiming a stage is ready.

## Per-task Evidence

For each slice, capture the smallest relevant evidence set:
- compile or type-check result when relevant
- direct test result when relevant
- manual behavior check when automation is unavailable
- explicit note of anything left unverified

## Integration Evidence

For integrated work, capture evidence for:
- end-to-end path correctness
- contract agreement across boundaries
- state transitions or data lifecycle correctness
- user-visible behavior where relevant
- logs, metrics, or runtime signals where relevant

## Review Readiness

Before final code review, make sure the following are explicit:
- what was actually validated
- what was not validated
- which risks remain
- whether rollout, migration, or rollback notes are needed

## Evidence Rule

Never replace evidence with confidence language. If you did not run it, say you did not run it.
