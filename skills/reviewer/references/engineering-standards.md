# Engineering Standards

Read this file when judging maintainability, readability, and operational quality.

## Clean code

- Names should be self-explanatory enough that the reviewer can follow the logic without extra narration
- Flag hard-coded values and unexplained magic numbers
- In frontend review, treat raw UI constants in component CSS as maintainability risk when the project already has a token system. Check `z-index`, motion durations, focus ring width, spacing, radius, modal width/height, overlay color, and shadow values before approving.
- Good example: when a component needs a new modal/focus overlay layer, prefer adding or reusing semantic tokens such as `--ui-dialog-z-index`, `--ui-motion-fast`, `--ui-focus-ring-width`, `--ui-dialog-wide-width`, `--ui-dialog-shadow`, and `--ui-control-radius-*`, then update shared shells to consume the same token. This is better than leaving `z-index: 140`, `transition: 0.18s ease`, `outline: 2px`, or ad hoc rem values inside one component.
- Identify overly deep nesting, high cyclomatic complexity, and functions that mix orchestration with low-level detail
- Prefer small, coherent units with clear ownership over sprawling helper chains
- Flag files whose size now blocks effective review. As a working signal, scrutinize Vue SFCs or route views above roughly 700 lines, backend services or repositories above roughly 800 lines, and functions above roughly 80 lines when the diff adds more behavior to them. These are review triggers, not automatic blockers.
- When a large file is touched, check whether the diff should extract a component, composable, helper, query object, mapper, or smaller service around a real ownership boundary.
- For quantitative complexity assessment, consider running static analysis tools (e.g., `gocyclo`, `eslint-plugin-complexity`, SonarQube) to flag functions with cyclomatic complexity >10 or maintainability index <20. Human review should then focus on whether the complexity is justified by the problem domain or can be decomposed.

## Maintainability

- Ask how hard the next change will be if this lands as written
- If a simple feature would require touching many files, coupling is probably too high
- Check whether the change makes extension easy without forcing edit cascades in unrelated modules
- Check whether tests still match the new structure. After extraction or decomposition, tests should prove user-visible behavior, contracts, state transitions, and failure handling instead of only asserting that new child components exist.
- Compare the implementation to the surrounding architecture as a senior maintainer would. Prefer recommendations that improve ownership, explicit contracts, failure handling, testability, and future change cost.
- When suggesting a more elegant implementation, name the concrete lower-risk shape: move state to the route owner, extract a composable for one async workflow, split a repository port by capability, add a mapper boundary, introduce a small domain object, or remove an unnecessary abstraction.
- Do not recommend broad rewrites just because the reviewer would write it differently. The alternative must be tied to a specific bug risk, reviewability problem, coupling cost, or testability gap.
- Check cognitive load: functions that mix multiple abstraction levels (e.g., raw DB queries + high-level business rules + UI formatting in one body), or nest control flow deeper than 3 levels, impose high mental overhead even when cyclomatic complexity is moderate. Flag when reading the function requires holding more than 3 context switches in working memory.

## Logging and observability

- Review log level choice: `info`, `warn`, and `error` should match operational impact
- Important failure paths should not fail silently
- Check whether critical flows need metrics, tracing, or audit logging and whether this change weakens or omits them
- Avoid logging sensitive payloads just to gain debuggability

## Documentation and comments

- Public contract changes should usually be reflected in nearby docs, types, or usage examples
- Comments should explain why a rule or workaround exists, not paraphrase the code
- If behavior changed, check whether tests and docs still describe the same system

## Tooling mindset

- If the same formatting or low-value nit appears repeatedly, suggest linting, formatting, or static analysis instead of manual review repetition
- Prefer process and tooling improvements over recurring reviewer labor
