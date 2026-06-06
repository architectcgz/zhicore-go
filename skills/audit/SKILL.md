---
name: audit
description: Run technical quality checks across accessibility, async state handling, performance, theming, responsive design, and frontend anti-patterns. Generates a scored report with P0-P3 severity ratings and actionable plan. Use when the user wants an accessibility check, performance audit, or technical quality review.
argument-hint: "[area (feature, page, component...)]"
---

## MANDATORY PREPARATION

Invoke $frontend-design — it contains design principles, anti-patterns, and the **Context Gathering Protocol**. Follow the protocol before proceeding — if no design context exists yet, you MUST run $teach-impeccable first.

---

Run systematic **technical** quality checks and generate a comprehensive report. Don't fix issues — document them for other commands to address.

This is a code-level audit, not a design critique. Check what's measurable and verifiable in the implementation.

## Diagnostic Scan

Run comprehensive checks across 6 dimensions. Score each dimension 0-4 using the criteria below.

### 1. Accessibility (A11y)

**Check for**:
- **Contrast issues**: Text contrast ratios < 4.5:1 (or 7:1 for AAA)
- **Missing ARIA**: Interactive elements without proper roles, labels, or states
- **Keyboard navigation**: Missing focus indicators, illogical tab order, keyboard traps
- **Semantic HTML**: Improper heading hierarchy, missing landmarks, divs instead of buttons
- **Alt text**: Missing or poor image descriptions
- **Form issues**: Inputs without labels, poor error messaging, missing required indicators

**Score 0-4**: 0=Inaccessible (fails WCAG A), 1=Major gaps (few ARIA labels, no keyboard nav), 2=Partial (some a11y effort, significant gaps), 3=Good (WCAG AA mostly met, minor gaps), 4=Excellent (WCAG AA fully met, approaches AAA)

### 2. Async State & Boundary Conditions

**Check for**:
- **Happy-path only UI**: Missing explicit loading, error, empty, or no-permission states
- **Unsafe re-entry**: Buttons, uploads, or multi-step flows that stay clickable during in-flight work
- **Missing request guards**: No debounce/throttle on search/filter inputs or repeat-fire interactions
- **Race conditions**: Fast tab/filter/route switches where an older response can overwrite the current selection
- **Request validity**: Missing `AbortController`, request-id guards, or "latest request wins" checks before assignment
- **State machine rollback**: Late responses that can move a multi-stage flow back to an older stage

**Score 0-4**: 0=Unsafe (multiple happy-path traps, race conditions likely), 1=Major gaps (missing states or request guards on core flows), 2=Partial (basic states exist, several weak async edges), 3=Good (core flows guarded, minor gaps), 4=Excellent (explicit states, safe async transitions, stale responses contained)

### 3. Performance & Lifecycle

**Check for**:
- **Layout thrashing**: Reading/writing layout properties in loops
- **Expensive animations**: Animating layout properties (width, height, top, left) instead of transform/opacity
- **Missing optimization**: Images without lazy loading, unoptimized assets, missing will-change
- **Bundle size**: Unnecessary imports, unused dependencies
- **Render performance**: Unnecessary re-renders, missing memoization
- **Huge DOMs**: Long lists, chat logs, ranking panels, or logs rendered wholesale without pagination or virtualization when counts are high
- **Uncleaned side effects**: `setInterval`, global listeners, observers, or third-party instances created on mount but never torn down on unmount
- **Third-party leaks**: ECharts, editors, charts, or media instances not disposed when route/component changes

**Score 0-4**: 0=Severe issues (layout thrash, unoptimized everything), 1=Major problems (no lazy loading, expensive animations), 2=Partial (some optimization, gaps remain), 3=Good (mostly optimized, minor improvements possible), 4=Excellent (fast, lean, well-optimized)

### 4. Theming

**Check for**:
- **Hard-coded colors**: Colors not using design tokens
- **Broken dark mode**: Missing dark mode variants, poor contrast in dark theme
- **Inconsistent tokens**: Using wrong tokens, mixing token types
- **Theme switching issues**: Values that don't update on theme change

**Score 0-4**: 0=No theming (hard-coded everything), 1=Minimal tokens (mostly hard-coded), 2=Partial (tokens exist but inconsistently used), 3=Good (tokens used, minor hard-coded values), 4=Excellent (full token system, dark mode works perfectly)

### 5. Responsive Design

**Check for**:
- **Fixed widths**: Hard-coded widths that break on mobile
- **Touch targets**: Interactive elements < 44x44px
- **Horizontal scroll**: Content overflow on narrow viewports
- **Text scaling**: Layouts that break when text size increases
- **Missing breakpoints**: No mobile/tablet variants

**Score 0-4**: 0=Desktop-only (breaks on mobile), 1=Major issues (some breakpoints, many failures), 2=Partial (works on mobile, rough edges), 3=Good (responsive, minor touch target or overflow issues), 4=Excellent (fluid, all viewports, proper touch targets)

### 6. Frontend Anti-Patterns & Architecture (CRITICAL)

Check both implementation anti-patterns and the **DON'T** guidelines in the frontend-design skill.

**Check for**:
- **Vue 3 reactivity traps**: Direct `props` destructuring in `<script setup>` without `toRef`/`toRefs`
- **Store abuse**: Purely local UI state pushed into Pinia/Vuex without a cross-component need
- **God components**: Large route/page components mixing fetch logic, validation, orchestration, and dense template concerns without subcomponents or composables
- **Deep prop drilling**: State or callbacks passed across many layers where `provide/inject` or a better boundary is warranted
- **Inconsistent async ownership**: View, component, and composable all mutating the same remote state without clear authority
- **Test blind spots**: Tests that only protect happy path while missing stale request, loading, error, empty, or cleanup behavior
- **AI slop tells**: AI color palette, gradient text, glassmorphism, hero metrics, card grids, generic fonts, and general design anti-patterns such as gray on color, nested cards, bounce easing, redundant copy

**Score 0-4**: 0=Severe architectural drift (multiple framework traps or AI tells), 1=Major problems (shared logic tangled, weak state boundaries), 2=Partial (some component/composable structure, notable traps remain), 3=Good (sound boundaries, minor cleanup opportunities), 4=Excellent (clean ownership, no obvious framework or design anti-patterns)

## Special Frontend Checks

Always include these extra passes when the target has data fetching, tabs, filters, uploads, charts, or long lists:

- Rapidly switch route params, tabs, or filter values and check whether stale responses can still render
- Repeatedly click primary actions and verify whether duplicate submissions are blocked while in flight
- Inspect whether each remote view has explicit `loading`, `error`, `empty`, and `success` ownership
- Search for mount-time side effects (`setInterval`, `addEventListener`, third-party instances) and confirm teardown
- Look for list sizes that justify virtualization or server-driven pagination rather than raw `v-for`
- Check whether props were destructured into inert locals inside `<script setup>`

## Generate Report

### Audit Health Score

| # | Dimension | Score | Key Finding |
|---|-----------|-------|-------------|
| 1 | Accessibility | ? | [most critical a11y issue or "--"] |
| 2 | Async State & Boundary Conditions | ? | |
| 3 | Performance & Lifecycle | ? | |
| 4 | Responsive Design | ? | |
| 5 | Theming | ? | |
| 6 | Frontend Anti-Patterns & Architecture | ? | |
| **Total** | | **??/24** | **[Rating band]** |

**Rating bands**: 21-24 Excellent (minor polish), 16-20 Good (address weak dimensions), 11-15 Acceptable (significant work needed), 6-10 Poor (major overhaul), 0-5 Critical (fundamental issues)

### Anti-Patterns & Architecture Verdict
**Start here.** Pass/fail: Does this show frontend engineering drift, AI-generated UI tells, or both? List the most concrete signals. Be brutally honest.

### Executive Summary
- Audit Health Score: **??/24** ([rating band])
- Total issues found (count by severity: P0/P1/P2/P3)
- Top 3-5 critical issues
- Recommended next steps

### Detailed Findings by Severity

Tag every issue with **P0-P3 severity**:
- **P0 Blocking**: Prevents task completion — fix immediately
- **P1 Major**: Significant difficulty or WCAG AA violation — fix before release
- **P2 Minor**: Annoyance, workaround exists — fix in next pass
- **P3 Polish**: Nice-to-fix, no real user impact — fix if time permits

For each issue, document:
- **[P?] Issue name**
- **Location**: Component, file, line
- **Category**: Accessibility / Async State / Performance / Lifecycle / Theming / Responsive / Anti-Pattern / Architecture
- **Impact**: How it affects users
- **WCAG/Standard**: Which standard it violates (if applicable)
- **Recommendation**: How to fix it
- **Suggested command**: Which command to use (prefer: $animate, $quieter, $optimize, $adapt, $clarify, $distill, $delight, $onboard, $normalize, $audit, $harden, $polish, $extract, $bolder, $arrange, $typeset, $critique, $colorize, $overdrive)

### Patterns & Systemic Issues

Identify recurring problems that indicate systemic gaps rather than one-off mistakes:
- "Hard-coded colors appear in 15+ components, should use design tokens"
- "Touch targets consistently too small (<44px) throughout mobile experience"

### Positive Findings

Note what's working well — good practices to maintain and replicate.

## Recommended Actions

List recommended commands in priority order (P0 first, then P1, then P2):

1. **[P?] `$command-name`** — Brief description (specific context from audit findings)
2. **[P?] `$command-name`** — Brief description (specific context)

**Rules**: Only recommend commands from: $animate, $quieter, $optimize, $adapt, $clarify, $distill, $delight, $onboard, $normalize, $audit, $harden, $polish, $extract, $bolder, $arrange, $typeset, $critique, $colorize, $overdrive. Map findings to the most appropriate command. End with `$polish` as the final step if any fixes were recommended.

After presenting the summary, tell the user:

> You can ask me to run these one at a time, all at once, or in any order you prefer.
>
> Re-run `$audit` after fixes to see your score improve.

**IMPORTANT**: Be thorough but actionable. Too many P3 issues creates noise. Focus on what actually matters.

**NEVER**:
- Report issues without explaining impact (why does this matter?)
- Provide generic recommendations (be specific and actionable)
- Skip positive findings (celebrate what works)
- Forget to prioritize (everything can't be P0)
- Report false positives without verification

Remember: You're a technical quality auditor. Document systematically, prioritize ruthlessly, cite specific code locations, and provide clear paths to improvement.
