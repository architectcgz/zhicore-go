# Review Communication

Read this file before writing comments or summarizing a review.

## Feedback principles

- Review the code, not the author
- State impact and reasoning, not just preference
- Separate blockers from suggestions
- Use questions when they help the author see a risk clearly
- Acknowledge strong implementation choices when they materially improve the code

## Tone rules

- Avoid insults, sarcasm, or loaded phrasing
- Prefer: "This can race if two requests update the same state"
- Avoid: "This is wrong" without explanation
- Prefer: "What happens here if the input is empty or arrives twice"
- Avoid: "You forgot edge cases"

## Comment template

- `Severity`: Blocker / Major / Minor / Nit
- `Location`: file, function, or line context
- `Issue`: what is risky or unclear
- `Why it matters`: user, system, security, or maintenance impact
- `Suggestion`: fix direction, alternative pattern, or validating question

## Praise discipline

- Praise should be specific, not generic cheerleading
- Call out good simplifications, smart boundary placement, or useful tests
- Keep praise secondary to findings when the user asked for a strict review

## Review output shape

- Findings first, highest severity first
- Then open questions or assumptions
- Then brief overall summary or merge-readiness note
