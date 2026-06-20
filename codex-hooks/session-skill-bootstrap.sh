#!/usr/bin/env bash
set -euo pipefail

# Bootstrap 纪律 skill：跨工具的 Session Discipline 入口，与项目/全局列表无关，始终注入。
skill_path="${CODEX_SKILL_BOOTSTRAP_PATH:-$HOME/.agents/skills/superpowers/using-superpowers/SKILL.md}"

if [[ ! -f "$skill_path" && -f "$HOME/.agents/skills/using-superpowers/SKILL.md" ]]; then
  skill_path="$HOME/.agents/skills/using-superpowers/SKILL.md"
fi

if [[ -f "$skill_path" ]]; then
  skill_body="$(<"$skill_path")"
else
  skill_body="The bootstrap skill was not found at: $skill_path"
fi

# 定位“当前项目”：Codex SessionStart hook 通过 stdin JSON 传入 cwd；取不到则回退 $PWD。
# 不依赖 jq（本机无 jq），用 sed 抽取首个 "cwd": "..." 值。
hook_input="$(cat 2>/dev/null || true)"
cwd=""
if [[ -n "$hook_input" ]]; then
  cwd="$(printf '%s' "$hook_input" | sed -n 's/.*"cwd"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
fi
[[ -z "$cwd" ]] && cwd="${PWD:-$(pwd)}"

# 从 cwd 向上查找项目级 skill 目录：优先 <repo>/.agents/skills，回退 <repo>/.claude/skills。
# 这对齐 Claude 在项目内通过 .claude/skills 自动加载项目 skill 的行为。
# 关键排除：home 下的 .agents/skills 本身就是全局 root，.claude/skills 又软链到它；
# 用 realpath 比较，凡指向全局 root 的入口目录都不算“项目”，避免把 $HOME 误判成项目根。
global_root="${CODEX_SKILLS_ROOT:-$HOME/.agents/skills}"
global_real="$(readlink -f "$global_root" 2>/dev/null || printf '%s' "$global_root")"

is_global_entry() {
  local cand_real
  cand_real="$(readlink -f "$1" 2>/dev/null || printf '%s' "$1")"
  [[ "$cand_real" == "$global_real" ]]
}

project_skills_root=""
project_root=""
dir="$cwd"
while [[ -n "$dir" && "$dir" != "/" ]]; do
  if [[ -d "$dir/.agents/skills" ]] && ! is_global_entry "$dir/.agents/skills"; then
    project_skills_root="$dir/.agents/skills"; project_root="$dir"; break
  fi
  if [[ -d "$dir/.claude/skills" ]] && ! is_global_entry "$dir/.claude/skills"; then
    project_skills_root="$dir/.claude/skills"; project_root="$dir"; break
  fi
  dir="$(dirname "$dir")"
done

# 构建 skill 索引（name + description + 路径）。索引即路由表：抗上下文压缩，
# Codex 命中 description 后再按需读对应 SKILL.md（激活优于存储 / 按需加载）。
# 排除 .system/：Codex 系统级 skill，有独立加载路径。
build_skill_index() {
  local root="$1"
  local lines="" md name desc
  if [[ -d "$root" ]]; then
    while IFS= read -r md; do
      name="$(awk '/^name:[[:space:]]*/{sub(/^name:[[:space:]]*/,""); print; exit}' "$md")"
      [[ -z "$name" ]] && continue
      desc="$(awk '/^description:[[:space:]]*/{sub(/^description:[[:space:]]*/,""); print; exit}' "$md")"
      lines+="- ${name}: ${desc} [${md}]"$'\n'
    done < <(find "$root" -name SKILL.md -not -path '*/.system/*' | sort)
  fi
  printf '%s' "$lines"
}

# 默认注入“当前项目”的说明性/约束性 skill；不在任何项目内时才回退全局共享 skill 作为发现兜底。
index_root=""
scope_label=""
skill_index_lines=""
if [[ -n "$project_skills_root" ]]; then
  skill_index_lines="$(build_skill_index "$project_skills_root")"
  index_root="$project_skills_root"
  scope_label="当前项目 ($project_root)"
fi

if [[ -z "$skill_index_lines" ]]; then
  index_root="${CODEX_SKILLS_ROOT:-$HOME/.agents/skills}"
  skill_index_lines="$(build_skill_index "$index_root")"
  scope_label="全局共享（未检测到项目 skill，回退）"
fi

if [[ -z "$skill_index_lines" ]]; then
  skill_index_lines="(No skills were found under $index_root)"
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

<available-skills scope="$scope_label">
These are the descriptive/constraining skills for the current scope, indexed
from disk under $index_root. Match the user request against each description
below; when one matches, read that skill's SKILL.md (path in brackets) from disk
before acting — the index is the routing table, the SKILL.md body is read on
demand. When working inside a project this lists that project's own skills (the
same set Claude auto-loads via the project's .claude/skills); the global shared
skills are only listed here as a fallback when the session is not inside any
project.

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
