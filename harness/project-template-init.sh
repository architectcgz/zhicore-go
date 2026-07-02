#!/usr/bin/env bash
set -euo pipefail

template_script="/home/azhi/.agents/skills/project-template/scripts/apply_project_template.py"

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/project-template-init.sh --list

  bash ~/.agents/harness/project-template-init.sh backend-go \
    --dest <dir> \
    --module <go-module> \
    --service <service-name> \
    --domain <domain-name> \
    [--git-user-name <name>] \
    [--git-user-email <email>] \
    [--skip-git-init] \
    [--force] [--dry-run]

  bash ~/.agents/harness/project-template-init.sh frontend-vue \
    --dest <dir> \
    --app-name <app-name> \
    [--auth-redirect <route>] \
    [--login-path <route>] \
    [--git-user-name <name>] \
    [--git-user-email <email>] \
    [--skip-git-init] \
    [--force] [--dry-run]

Description:
  Convenience wrapper around project-template starter assets.

Aliases:
  backend-go   -> backend/go-backend-onion-template
  frontend-vue -> frontend/vue-feature-sliced-template
EOF
}

resolve_template() {
  case "$1" in
    backend-go|go-backend|backend/go-backend-onion-template)
      printf '%s\n' "backend/go-backend-onion-template"
      ;;
    frontend-vue|vue-frontend|frontend/vue-feature-sliced-template)
      printf '%s\n' "frontend/vue-feature-sliced-template"
      ;;
    *)
      printf '%s\n' "$1"
      ;;
  esac
}

is_nonempty_value() {
  local value="$1"
  [[ -n "${value//[[:space:]]/}" ]]
}

resolve_git_identity() {
  local field="$1"
  local value="$2"
  local prompt=""

  case "$field" in
    name)
      prompt="Git user.name"
      ;;
    email)
      prompt="Git user.email"
      ;;
    *)
      echo "FAIL: unsupported git identity field: $field" >&2
      exit 1
      ;;
  esac

  if is_nonempty_value "$value"; then
    printf '%s\n' "$value"
    return 0
  fi

  if [[ -t 0 ]]; then
    read -r -p "$prompt: " value
  fi

  if ! is_nonempty_value "$value"; then
    echo "FAIL: git user $field is required for new repositories; pass --git-user-name and --git-user-email or rerun interactively" >&2
    exit 1
  fi

  printf '%s\n' "$value"
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

first_arg="$1"
if [[ "$first_arg" == "--list" ]]; then
  echo "[project-template-init] aliases"
  echo "- backend-go -> backend/go-backend-onion-template"
  echo "- frontend-vue -> frontend/vue-feature-sliced-template"
  echo
  exec python3 "$template_script" --list
fi

if [[ "$first_arg" == "-h" || "$first_arg" == "--help" ]]; then
  usage
  exit 0
fi

template="$(resolve_template "$first_arg")"
shift

dest=""
go_module=""
service_name=""
domain_name=""
app_name=""
auth_redirect="/student/dashboard"
login_path="/login"
git_user_name=""
git_user_email=""
skip_git_init=0
force=0
dry_run=0
passthrough=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dest)
      [[ $# -ge 2 ]] || { echo "FAIL: --dest requires a value" >&2; exit 1; }
      dest="$2"
      shift 2
      ;;
    --module)
      [[ $# -ge 2 ]] || { echo "FAIL: --module requires a value" >&2; exit 1; }
      go_module="$2"
      shift 2
      ;;
    --service)
      [[ $# -ge 2 ]] || { echo "FAIL: --service requires a value" >&2; exit 1; }
      service_name="$2"
      shift 2
      ;;
    --domain)
      [[ $# -ge 2 ]] || { echo "FAIL: --domain requires a value" >&2; exit 1; }
      domain_name="$2"
      shift 2
      ;;
    --app-name)
      [[ $# -ge 2 ]] || { echo "FAIL: --app-name requires a value" >&2; exit 1; }
      app_name="$2"
      shift 2
      ;;
    --auth-redirect)
      [[ $# -ge 2 ]] || { echo "FAIL: --auth-redirect requires a value" >&2; exit 1; }
      auth_redirect="$2"
      shift 2
      ;;
    --login-path)
      [[ $# -ge 2 ]] || { echo "FAIL: --login-path requires a value" >&2; exit 1; }
      login_path="$2"
      shift 2
      ;;
    --git-user-name)
      [[ $# -ge 2 ]] || { echo "FAIL: --git-user-name requires a value" >&2; exit 1; }
      git_user_name="$2"
      shift 2
      ;;
    --git-user-email)
      [[ $# -ge 2 ]] || { echo "FAIL: --git-user-email requires a value" >&2; exit 1; }
      git_user_email="$2"
      shift 2
      ;;
    --skip-git-init)
      skip_git_init=1
      shift
      ;;
    --force)
      force=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "FAIL: unknown arg: $1" >&2
      usage
      exit 1
      ;;
    *)
      passthrough+=("$1")
      shift
      ;;
  esac
done

if [[ ${#passthrough[@]} -gt 0 ]]; then
  echo "FAIL: unexpected positional args: ${passthrough[*]}" >&2
  usage
  exit 1
fi

if [[ -z "$dest" ]]; then
  echo "FAIL: --dest is required" >&2
  usage
  exit 1
fi

cmd=(
  python3 "$template_script"
  --template "$template"
  --dest "$dest"
)

case "$template" in
  backend/go-backend-onion-template)
    [[ -n "$go_module" ]] || { echo "FAIL: backend-go requires --module" >&2; exit 1; }
    [[ -n "$service_name" ]] || { echo "FAIL: backend-go requires --service" >&2; exit 1; }
    [[ -n "$domain_name" ]] || { echo "FAIL: backend-go requires --domain" >&2; exit 1; }
    cmd+=(
      --var "__GO_MODULE__=$go_module"
      --var "__SERVICE_NAME__=$service_name"
      --var "__DOMAIN_NAME__=$domain_name"
    )
    ;;
  frontend/vue-feature-sliced-template)
    [[ -n "$app_name" ]] || { echo "FAIL: frontend-vue requires --app-name" >&2; exit 1; }
    cmd+=(
      --var "__APP_NAME__=$app_name"
      --var "__DEFAULT_AUTH_REDIRECT__=$auth_redirect"
      --var "__DEFAULT_LOGIN_PATH__=$login_path"
    )
    ;;
  *)
    echo "FAIL: unsupported template alias for wrapper: $template" >&2
    echo "Use apply_project_template.py directly for custom templates." >&2
    exit 1
    ;;
esac

if [[ "$force" -eq 1 ]]; then
  cmd+=(--force)
fi
if [[ "$dry_run" -eq 1 ]]; then
  cmd+=(--dry-run)
fi

"${cmd[@]}"

if [[ "$dry_run" -eq 1 ]]; then
  exit 0
fi

dest_root="$(cd "$dest" && pwd)"
git_root=""
if git_root="$(git -C "$dest_root" rev-parse --show-toplevel 2>/dev/null)"; then
  if [[ "$git_root" == "$dest_root" ]]; then
    echo "[project-template-init] keep existing git repository: $git_root"
  else
    echo "[project-template-init] destination already belongs to git repository: $git_root"
  fi
  exit 0
fi

if [[ "$skip_git_init" -eq 1 ]]; then
  echo "[project-template-init] skip git initialization for new directory: $dest_root"
  exit 0
fi

# New project roots must carry an explicit local commit identity instead of inheriting an agent default.
git_user_name="$(resolve_git_identity name "$git_user_name")"
git_user_email="$(resolve_git_identity email "$git_user_email")"

git init "$dest_root" >/dev/null
git -C "$dest_root" config user.name "$git_user_name"
git -C "$dest_root" config user.email "$git_user_email"

echo "[project-template-init] initialized git repository: $dest_root"
echo "[project-template-init] configured local git identity: $git_user_name <$git_user_email>"
