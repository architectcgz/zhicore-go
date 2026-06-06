# Over-Broad Repository Or Port Case

Use this case when a repository, service, port, or gateway is named around one aggregate but is used by several unrelated command and query flows.

## Example Shape

A team repository may mix:

- team creation or mutation
- member management
- list or detail queries
- user lookup
- existence or uniqueness checks
- policy checks

This is a responsibility smell when callers only need one slice but must depend on the whole port.

## Analysis Steps

1. Read the interface, implementation, callers, and tests.
2. Classify each method by use case, not by table name.
3. Identify which methods are command-side writes, query-side reads, lookups, uniqueness/existence checks, and policy checks.
4. Check transaction boundaries and consistency assumptions before proposing extraction.
5. Preserve existing external behavior; prefer extracting smaller internal ports before changing APIs.

## Split Direction

Prefer names that match consumers and responsibilities, for example:

- command-side team mutation port
- member query or member command port
- user lookup port
- uniqueness or existence checker
- read-model query port

Avoid splitting mechanically by database table if the use cases do not align with table boundaries.
