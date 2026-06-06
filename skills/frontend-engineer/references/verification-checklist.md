# Verification Checklist

Read this file before claiming a frontend implementation is complete.

## Core checks

- Can a user click twice and trigger duplicate work
- Can an older response overwrite a newer selection
- Does the UI still make sense during loading, empty, and error states
- Did I accidentally create two sources of truth for the same value
- Does the form preserve meaningful user input and map validation failures back to something actionable
- Is every mount-time side effect cleaned up
- Did props destructuring accidentally break Vue reactivity
- Did local UI state get pushed into global store without a real cross-component need
- If I could not execute the UI flow, did I clearly separate verified behavior from inference
- Does any rendered UI text contain design notes, structural explanations, TODO markers, placeholder commentary, or implementation hints that should stay in code comments or assistant output instead

## Common implementation drift

- Shipping only the success state
- Forgetting to cancel or invalidate stale requests
- Copying props or remote payloads into local state without a real ownership reason
- Leaving timers, listeners, or chart instances alive after unmount
- Letting one `.vue` file become the API layer, controller, and template all at once
- Writing comments that explain what the code literally does instead of why the rule exists
