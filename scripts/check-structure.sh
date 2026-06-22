#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

required_dirs=(
  "services/zhicore-gateway/cmd/server"
  "services/zhicore-gateway/internal/gateway"
  "services/zhicore-gateway/api/http"
  "services/zhicore-gateway/configs"
  "services/zhicore-gateway/migrations"
  "services/zhicore-user/cmd/server"
  "services/zhicore-user/internal/user"
  "services/zhicore-user/api/http"
  "services/zhicore-user/configs"
  "services/zhicore-user/migrations"
  "services/zhicore-content/cmd/server"
  "services/zhicore-content/internal/content"
  "services/zhicore-content/api/http"
  "services/zhicore-content/configs"
  "services/zhicore-content/migrations"
  "services/zhicore-comment/cmd/server"
  "services/zhicore-comment/internal/comment"
  "services/zhicore-comment/api/http"
  "services/zhicore-comment/configs"
  "services/zhicore-comment/migrations"
  "services/zhicore-message/cmd/server"
  "services/zhicore-message/internal/message"
  "services/zhicore-message/api/http"
  "services/zhicore-message/configs"
  "services/zhicore-message/migrations"
  "services/zhicore-notification/cmd/server"
  "services/zhicore-notification/internal/notification"
  "services/zhicore-notification/api/http"
  "services/zhicore-notification/configs"
  "services/zhicore-notification/migrations"
  "services/zhicore-search/cmd/server"
  "services/zhicore-search/internal/search"
  "services/zhicore-search/api/http"
  "services/zhicore-search/configs"
  "services/zhicore-search/migrations"
  "services/zhicore-ranking/cmd/server"
  "services/zhicore-ranking/internal/ranking"
  "services/zhicore-ranking/api/http"
  "services/zhicore-ranking/configs"
  "services/zhicore-ranking/migrations"
  "services/zhicore-admin/cmd/server"
  "services/zhicore-admin/internal/admin"
  "services/zhicore-admin/api/http"
  "services/zhicore-admin/configs"
  "services/zhicore-admin/migrations"
  "services/zhicore-upload/cmd/server"
  "services/zhicore-upload/internal/upload"
  "services/zhicore-upload/api/http"
  "services/zhicore-upload/configs"
  "services/zhicore-upload/migrations"
  "services/zhicore-id-generator/cmd/server"
  "services/zhicore-id-generator/internal/idgenerator"
  "services/zhicore-id-generator/api/http"
  "services/zhicore-id-generator/configs"
  "services/zhicore-id-generator/migrations"
  "services/zhicore-ops/cmd/server"
  "services/zhicore-ops/internal/ops"
  "services/zhicore-ops/api/http"
  "services/zhicore-ops/configs"
  "services/zhicore-ops/migrations"
  "libs/kit/httpapi"
  "libs/kit/auth"
  "libs/kit/config"
  "libs/kit/observability"
  "libs/kit/postgres"
  "libs/kit/redis"
  "libs/kit/mongo"
  "libs/kit/rabbitmq"
  "libs/kit/es"
  "libs/contracts/events"
  "libs/contracts/clients"
  "deploy/docker"
  "deploy/k8s"
  "docs/architecture"
  "docs/contracts"
  "docs/migration"
  "docs/reviews"
  "docs/todos/debt"
  "tests/architecture"
  "tests/system/http"
  "tests/runtime"
  "tests/testkit"
)

for dir in "${required_dirs[@]}"; do
  if [[ ! -d "$ROOT/$dir" ]]; then
    echo "missing directory: $dir" >&2
    exit 1
  fi
done

required_files=(
  "AGENTS.md"
  "CLAUDE.md"
  "README.md"
  "go.work"
  "libs/contracts/go.mod"
  "libs/kit/go.mod"
  "services/zhicore-gateway/go.mod"
  "services/zhicore-user/go.mod"
  "services/zhicore-content/go.mod"
  "services/zhicore-comment/go.mod"
  "services/zhicore-message/go.mod"
  "services/zhicore-notification/go.mod"
  "services/zhicore-search/go.mod"
  "services/zhicore-ranking/go.mod"
  "services/zhicore-admin/go.mod"
  "services/zhicore-upload/go.mod"
  "services/zhicore-id-generator/go.mod"
  "services/zhicore-ops/go.mod"
  "docs/README.md"
  "docs/documentation-rules.md"
  "docs/architecture/repository-layout.md"
  "docs/architecture/go-service-design.md"
  "docs/architecture/service-boundaries.md"
  "docs/architecture/id-strategy.md"
  "docs/contracts/README.md"
  "docs/migration/README.md"
  "docs/migration/java-design-migration.md"
)

for file in "${required_files[@]}"; do
  if [[ ! -e "$ROOT/$file" ]]; then
    echo "missing file: $file" >&2
    exit 1
  fi
done

if [[ ! -L "$ROOT/CLAUDE.md" ]]; then
  echo "CLAUDE.md must be a symlink to AGENTS.md" >&2
  exit 1
fi

if [[ "$(readlink "$ROOT/CLAUDE.md")" != "AGENTS.md" ]]; then
  echo "CLAUDE.md must point to AGENTS.md" >&2
  exit 1
fi

if [[ -e "$ROOT/go.mod" ]]; then
  echo "root go.mod is not allowed; use go.work plus per-service modules" >&2
  exit 1
fi

if [[ -d "$ROOT/cmd" || -d "$ROOT/internal" ]]; then
  echo "root cmd/ and internal/ are not allowed; use services/<service> and libs/" >&2
  exit 1
fi

echo "structure ok"
