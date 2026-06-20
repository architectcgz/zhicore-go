#!/usr/bin/env bash
set -euo pipefail

skill_path="${CODEX_SKILL_BOOTSTRAP_PATH:-$HOME/.agents/skills/superpowers/using-superpowers/SKILL.md}"

if [[ ! -f "$skill_path" && -f "$HOME/.agents/skills/using-superpowers/SKILL.md" ]]; then
  skill_path="$HOME/.agents/skills/using-superpowers/SKILL.md"
fi

if [[ -f "$skill_path" ]]; then
  skill_body="$(<"$skill_path")"
else
  skill_body="The bootstrap skill was not found at: $skill_path"
fi

# 生成全量共享 skill 索引（name + description），让 Codex 与 Claude 的可见 skill 集合对齐。
# 排除 .system/：那是 Codex 系统级 skill，有独立加载路径，Claude 也忽略该目录。
# 不加 find -L：根目录指向 superpowers/* 的软链接不会被递归进入，天然避免与实体路径重复。
skills_root="${CODEX_SKILLS_ROOT:-$HOME/.agents/skills}"
skill_index_lines=""
if [[ -d "$skills_root" ]]; then
  while IFS= read -r md; do
    name="$(awk '/^name:[[:space:]]*/{sub(/^name:[[:space:]]*/,""); print; exit}' "$md")"
    [[ -z "$name" ]] && continue
    desc="$(awk '/^description:[[:space:]]*/{sub(/^description:[[:space:]]*/,""); print; exit}' "$md")"
    skill_index_lines+="- ${name}: ${desc} [${md}]"$'\n'
  done < <(find "$skills_root" -name SKILL.md -not -path '*/.system/*' | sort)
fi
if [[ -z "$skill_index_lines" ]]; then
  skill_index_lines="(No shared skills were found under $skills_root)"
fi

session_context="$(cat <<EOF
<codex-skill-bootstrap>
This Codex thread has just started, resumed, cleared, or compacted. Previously
loaded skill bodies may no longer be present in context.

Before any non-trivial task:
- Re-run skill matching from the current user request.
- If a skill is explicitly named or its description matches, read that skill's
  current SKILL.md from disk before acting.
- Do not rely on "I already read it earlier" after startup, resume, clear, or
  compact.
- Keep loaded context minimal: read only the matching skill and the references
  it explicitly requires.

Below is the bootstrap skill that defines the required skill-use discipline.

Skill path: $skill_path

$skill_body

<available-skills>
These shared skills exist on disk under $skills_root. Match the user request
against each description below; when one matches, read that skill's SKILL.md
(path in brackets) from disk before acting. This index mirrors what Claude
auto-loads, so both agents discover the same skill set — do not assume a skill
is unavailable just because its body is not yet in context.

$skill_index_lines
</available-skills>
</codex-skill-bootstrap>
EOF
)"

escape_for_json() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}

escaped_context="$(escape_for_json "$session_context")"

printf '{\n'
printf '  "hookSpecificOutput": {\n'
printf '    "hookEventName": "SessionStart",\n'
printf '    "additionalContext": "%s"\n' "$escaped_context"
printf '  }\n'
printf '}\n'
