#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

required_routes=(
  dashboard
  agents
  knowledge
  glossary
  inbox
  calendar
  communication
  relationships
  projects
  tasks
  rhythm
  team
  connectors
  help
  modules
)

for route in "${required_routes[@]}"; do
  path="frontend/src/routes/(app)/${route}"
  if [[ ! -d "$path" ]]; then
    echo "::error::OSS projection is missing foundational route: $route"
    exit 1
  fi
done

profile_file="frontend/src/lib/config/workspaceModuleProfiles.ts"
if grep -RniIE --exclude='*.test.*' \
  'agency-miosa|agency miosa|terrawatt|project atlas|robert potter|betterstem|miami stem|greice|gracie|pritt|scott tripp|panther defense|clinic.?iq|lunivate|roberto|vantage|meridian' \
  frontend/src desktop/backend-go/internal/database/migrations; then
  echo "::error::OSS projection contains private workspace configuration or seeded client data"
  exit 1
fi

if ! grep -q 'value: "primitive"' "$profile_file"; then
  echo "::error::OSS projection does not expose the generic primitive module profile"
  exit 1
fi

if [[ ! -f "frontend/src/routes/(app)/modules/[id]/+page.svelte" ]]; then
  echo "::error::OSS projection is missing the custom module operating surface"
  exit 1
fi

private_paths=(
  .github/workflows/deploy-backend.yml
  .github/workflows/deploy-frontend.yml
  desktop/backend-go/internal/handlers/platform_admin_control.go
  desktop/backend-go/internal/handlers/platform_admin_miosa.go
  desktop/backend-go/internal/database/migrations/162_repair_private_workspace_module_profiles.sql
)

for path in "${private_paths[@]}"; do
  if [[ -e "$path" ]]; then
    echo "::error::OSS projection still contains private source: $path"
    exit 1
  fi
done

echo "OSS projection verified: foundational shell present; private workspace configuration absent."
