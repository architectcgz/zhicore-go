# Completion Review

Read this file before finalizing frontend implementation, refactor, interaction behavior, or maintainability work.

## Review gate

- Do not treat typecheck or a few happy-path tests as sufficient closure when the leader or pipeline has classified the work as non-trivial; completion requires a distinct review pass for interaction regressions, state-ownership drift, oversized component debt, contract mismatches, and test gaps.
- Report frontend risk signals instead of redefining trivial/non-trivial policy locally: async flow, form, route sync, store, modal or drawer state, cross-component contract, user-visible workflow, extraction, or oversized component/service growth.
- If a reusable frontend rule gap, repeated miss, missing checklist item, or frontend engineering optimization should be preserved for future runs, use `improvement-tracker` to record it under `docs/improvements/`.

## Output protocol

Use this structure unless the user explicitly asks for another format.

Always include:

- Result
- Change Surface
- User-Facing Behavior
- State / Async Ownership
- Component Contract
- Verification
- Review / Completion Gate
- Risks / Unverified Points
- Improvement Records

Include only when relevant:

- Accessibility / Keyboard Behavior
- Responsive / Overflow Behavior
- Styling / Token Impact
- Lifecycle / Cleanup
- API / DTO Mapping
- Performance / Rendering Pressure
- Copy / User-Visible Content

Rules:

1. Start with the result.
2. Separate verified behavior from inference.
3. Name the owner of each async workflow, state source of truth, validation path, and remote mutation.
4. Keep route views, components, composables, and stores use-case-oriented; do not extract code only to reduce line count.
5. Do not include irrelevant conditional sections.
6. If a reusable frontend rule gap is discovered, use `improvement-tracker` and list the created file under `Improvement Records`.

Field guidance:

- `Change Surface`: list affected route views, components, composables, stores, API clients, styles, tests, and docs when applicable.
- `User-Facing Behavior`: state what users can now do, what changed, and what remains unchanged.
- `State / Async Ownership`: state where loading, error, empty, success, cancellation, stale-response, and duplicate-action handling live.
- `Component Contract`: state relevant props, emits, `v-model`, local draft state, and API-to-UI mapping boundaries.
- `Verification`: list only commands or checks actually run. If not run, state why and name the highest-risk unverified paths.
- `Review / Completion Gate`: state frontend risk signals, self-review result, independent review status when required, and whether leader or pipeline gating remains.
- `Improvement Records`: list created `docs/improvements/...` files, or state `None` if no reusable agent or policy gap was found.

## Output expectations

- The implemented surface handles real user interaction, not just the ideal path.
- Async ownership is clear enough that stale responses and duplicate actions are contained.
- Component contracts remain understandable and do not hide prop mutation, dual state ownership, or unclear `v-model` flow.
- Styling choices stay within the project's token and variable system instead of drifting into hardcoded local exceptions.
- The answer states what was tested, what was inferred, and what remains unverified.
