#!/usr/bin/env bash
set -euo pipefail

hook_input="$(cat 2>/dev/null || true)"

cwd="$(
  HOOK_INPUT="$hook_input" python3 - <<'PY' 2>/dev/null || true
import json
import os

raw = os.environ.get("HOOK_INPUT", "")
if not raw:
    raise SystemExit(0)
try:
    payload = json.loads(raw)
except json.JSONDecodeError:
    raise SystemExit(0)
cwd = payload.get("cwd")
if isinstance(cwd, str):
    print(cwd)
PY
)"

if [[ -z "$cwd" ]]; then
  cwd="${PWD:-$(pwd)}"
fi

root="$(git -C "$cwd" rev-parse --show-toplevel 2>/dev/null || printf '%s' "$cwd")"

source_path="${CODEX_SESSION_THIN_SHELL_SOURCE:-AGENTS.md}"
if [[ "$source_path" = /* ]]; then
  entry="$source_path"
else
  entry="$root/$source_path"
fi

if [[ ! -f "$entry" ]]; then
  exit 0
fi

max_lines="${CODEX_SESSION_THIN_SHELL_MAX_LINES:-120}"
max_bytes="${CODEX_SESSION_THIN_SHELL_MAX_BYTES:-12000}"

python3 - "$entry" "$root" "$max_lines" "$max_bytes" <<'PY'
import json
import re
import sys
from pathlib import Path

entry = Path(sys.argv[1])
root = Path(sys.argv[2])
max_lines = int(sys.argv[3])
max_bytes = int(sys.argv[4])

text = entry.read_text(encoding="utf-8", errors="replace")

marker_re = re.compile(
    r"<!--\s*codex-session-thin-shell:start\s*-->(.*?)"
    r"<!--\s*codex-session-thin-shell:end\s*-->",
    re.IGNORECASE | re.DOTALL,
)
marker_match = marker_re.search(text)

if marker_match:
    body = marker_match.group(1).strip()
else:
    heading_re = re.compile(r"^(#{1,6})\s+(.+?)\s*$")
    wanted = (
        "quick routing",
        "session discipline",
        "auto-triggers",
        "auto triggers",
        "red flags",
        "skill routing",
        "task routing",
        "routing",
        "任务路由",
        "技能路由",
        "自动触发",
        "会话纪律",
        "防失忆",
        "红旗",
        "停止",
    )

    lines = text.splitlines()
    sections: list[str] = []
    i = 0
    while i < len(lines):
        match = heading_re.match(lines[i])
        if not match:
            i += 1
            continue
        title = match.group(2).strip().lower()
        if not any(key in title for key in wanted):
            i += 1
            continue

        level = len(match.group(1))
        section = [lines[i]]
        i += 1
        while i < len(lines):
            next_match = heading_re.match(lines[i])
            if next_match and len(next_match.group(1)) <= level:
                break
            section.append(lines[i])
            i += 1
        sections.append("\n".join(section).strip())

    if sections:
        body = "\n\n".join(part for part in sections if part).strip()
    else:
        body = "\n".join(text.splitlines()[:max_lines]).strip()

body_lines = body.splitlines()[:max_lines]
body = "\n".join(body_lines).strip()

encoded = body.encode("utf-8")
if len(encoded) > max_bytes:
    body = encoded[:max_bytes].decode("utf-8", errors="ignore").rstrip()
    body += "\n\n[truncated by session-thin-shell hook]"

if not body:
    raise SystemExit(0)

try:
    display_entry = entry.relative_to(root)
except ValueError:
    display_entry = entry

context = f"""<project-thin-shell source="{display_entry}">
SessionStart reloaded the project routing shell. Treat this as a compact reminder to re-read the project entry and route to matching skills/workflows before task actions.

{body}
</project-thin-shell>"""

print(
    json.dumps(
        {
            "hookSpecificOutput": {
                "hookEventName": "SessionStart",
                "additionalContext": context,
            }
        },
        ensure_ascii=False,
    )
)
PY
