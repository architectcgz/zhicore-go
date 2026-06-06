# Harness Adaptation Notes

The reference project `deusyu/harness-engineering` remains an important upstream reference. It demonstrates these reusable patterns:

- repo-as-source-of-truth: knowledge lives in versioned files
- progressive navigation: root `AGENTS.md` points to smaller directory `AGENTS.md` files
- feedback capture: practical failures become durable records
- mechanical enforcement: scripts and hooks check claims that otherwise drift
- agent readability: directory shape and file names make next actions obvious

Current local default is the CTF harness shape, because the workspace is still exploring the right long-term convention. Use the upstream strict reference shape when the user explicitly asks to follow `deusyu/harness-engineering` structurally.

Current CTF-derived default mapping:

- root `AGENTS.md` -> repository entry map and project-specific overrides
- `.harness/` -> current-task scratch/state only
- `harness/policies/` -> project-local rules that can feed mechanical checks
- `harness/templates/` -> reusable project decision templates
- `harness/prompts/` -> validated project prompt assets
- `harness/checks/` -> deterministic guard scripts
- `.harness/reuse-index/` -> user-local durable reuse index, ignored by Git and mirrored from source paths with `README.md` secondary indexes
- `feedback/` -> workflow mistakes, corrections, and reusable lessons
- `scripts/check-consistency.sh` -> consistency guard for the chosen harness shape

Upstream strict reference mapping:

- concepts -> a project-local constraints index plus links to architecture and AGENTS rules
- thinking -> design/review rationale already present in architecture, plan, and review docs
- practice -> implementation plans and focused experiments
- feedback -> `docs/improvements/`, review findings, incident notes
- prompts -> project-local prompts or skills only when they are actually reused
- references -> `docs/refs/`, contracts, external research
- scripts/check-consistency.sh -> a project-tailored consistency script
