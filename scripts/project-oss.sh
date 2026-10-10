#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STUBS="$ROOT/.oss-stubs"

if [[ "${BUSINESSOS_OSS_ALLOW_IN_PLACE:-}" != "1" ]]; then
  echo "Refusing to strip this checkout without BUSINESSOS_OSS_ALLOW_IN_PLACE=1." >&2
  echo "Run this only inside CI or an expendable copied working tree." >&2
  exit 1
fi

if [[ ! -f "$ROOT/.oss-exclude" || ! -d "$STUBS" ]]; then
  echo "OSS projection inputs are missing." >&2
  exit 1
fi

STUB_COPY="$(mktemp -d "${TMPDIR:-/tmp}/businessos-oss-stubs.XXXXXX")"
trap 'rm -rf "$STUB_COPY"' EXIT
cp -R "$STUBS/." "$STUB_COPY/"
cp "$ROOT/.oss-exclude" "$STUB_COPY/exclude-manifest"

while IFS= read -r path; do
  [[ -z "$path" || "$path" == \#* ]] && continue
  path="${path%%#*}"
  path="${path% }"
  [[ -z "$path" ]] && continue
  rm -rf "$ROOT/$path"
done < "$STUB_COPY/exclude-manifest"

install -m 0644 "$STUB_COPY/handlers/billing.go" \
  "$ROOT/desktop/backend-go/internal/handlers/billing.go"
install -m 0644 "$STUB_COPY/handlers/platform_admin.go" \
  "$ROOT/desktop/backend-go/internal/handlers/platform_admin.go"
install -m 0644 "$STUB_COPY/config/workspaceModuleProfiles.ts" \
  "$ROOT/frontend/src/lib/config/workspaceModuleProfiles.ts"
install -m 0644 "$STUB_COPY/config/workspaceModules.test.ts" \
  "$ROOT/frontend/src/lib/config/workspaceModules.test.ts"
install -m 0644 "$STUB_COPY/config/workspaceContentDefaults.ts" \
  "$ROOT/frontend/src/lib/modules/content/workspaceContentDefaults.ts"
install -m 0644 "$STUB_COPY/config/relationshipsFieldPack.ts" \
  "$ROOT/frontend/src/routes/(app)/relationships/field-pack.ts"
install -m 0644 "$STUB_COPY/migrations/123_project_template.sql" \
  "$ROOT/desktop/backend-go/internal/database/migrations/123_project_template.sql"

echo "OSS projection applied."
