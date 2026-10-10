#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "primitive smoke failed: $*" >&2
  exit 1
}

require_file() {
  [[ -f "$1" ]] || fail "missing $1"
}

require_grep() {
  local pattern="$1"
  local file="$2"
  local label="$3"
  rg -q "$pattern" "$file" || fail "$label missing in $file"
}

require_file desktop/forge.config.ts
require_file desktop/src/main/terminal/env.ts
require_file desktop/src/main/terminal/pty-manager.ts
require_file desktop/src/main/ipc/index.ts
require_file desktop/src/preload/index.ts
require_file desktop/scripts/generate-updater-metadata.mjs
require_file desktop/scripts/upload-updater-metadata.mjs
require_file frontend/src/lib/stores/workspaces.ts

require_grep 'node-pty' desktop/forge.config.ts "node-pty package copy/rebuild"
require_grep 'better-sqlite3' desktop/forge.config.ts "better-sqlite3 package copy/rebuild"
require_grep 'Miosa-osa' desktop/forge.config.ts "real GitHub release owner"
require_grep 'businessos-5' desktop/forge.config.ts "real GitHub release repo"
require_grep 'latest-mac.yml' desktop/scripts/generate-updater-metadata.mjs "mac updater metadata generator"
require_grep '"gh"' desktop/scripts/upload-updater-metadata.mjs "mac updater metadata upload CLI"
require_grep '"release"' desktop/scripts/upload-updater-metadata.mjs "mac updater metadata upload command"
require_grep '"upload"' desktop/scripts/upload-updater-metadata.mjs "mac updater metadata upload command"
require_grep '"--clobber"' desktop/scripts/upload-updater-metadata.mjs "mac updater metadata upload replacement"
require_grep 'resolveShellCandidates' desktop/src/main/terminal/pty-manager.ts "terminal shell fallback"
require_grep '/bin/zsh' desktop/src/main/terminal/env.ts "macOS zsh fallback"
require_grep '/bin/bash' desktop/src/main/terminal/env.ts "macOS bash fallback"
require_grep 'diagnostics:collect' desktop/src/main/ipc/index.ts "diagnostics IPC handler"
require_grep 'diagnostics:collect' desktop/src/preload/index.ts "diagnostics preload bridge"
require_grep 'setFeedURL' desktop/src/main/updater/auto-update.ts "auto-updater feed configuration"
require_grep 'BUSINESSOS_MIN_SUPPORTED_VERSION' desktop/src/main/updater/auto-update.ts "minimum supported desktop version"
require_grep 'updates:get-info' desktop/src/main/ipc/index.ts "update runtime info IPC handler"
require_grep 'updates:get-info' desktop/src/preload/index.ts "update runtime info preload bridge"
require_grep 'localStorage.removeItem\("businessos_current_workspace_id"\)' frontend/src/lib/stores/workspaces.ts "stale workspace ID cleanup"
require_grep 'getActiveWorkspaceHeaders' frontend/src/lib/api/base.ts "workspace-aware API headers"

critical_raw_fetches="$(
  rg -n "fetch\\s*\\(\\s*['\\\"]/(api/(chat|terminal|transcribe)|api/v1/osa/health|api/ai/models/local)" \
    frontend/src desktop/src \
    --glob '!desktop/src/renderer/**' || true
)"
if [[ -n "$critical_raw_fetches" ]]; then
  echo "$critical_raw_fetches" >&2
  fail "critical packaged-unsafe raw /api fetches found"
fi

if command -v node >/dev/null 2>&1; then
  node <<'NODE'
const fs = require("node:fs");
const path = require("node:path");
const pkg = JSON.parse(fs.readFileSync(path.join("desktop", "package.json"), "utf8"));
for (const dep of ["node-pty", "better-sqlite3", "electron-store"]) {
  if (!pkg.dependencies || !pkg.dependencies[dep]) {
    console.error(`missing desktop runtime dependency: ${dep}`);
    process.exit(1);
  }
}
NODE
fi

echo "primitive smoke passed"
