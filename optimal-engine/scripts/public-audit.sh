#!/usr/bin/env bash
set -euo pipefail

fail=0

say() {
  printf '%s\n' "$*"
}

check_tracked_paths() {
  local matches
  matches="$(git ls-files | rg '(^|/)node_modules/|(^|/)\.env(\.|$)|\.(db|sqlite|sqlite3)$|(^|/)tmp/|(^|/)\.optimal/' || true)"

  if [[ -n "$matches" ]]; then
    say "Tracked generated/private runtime paths found:"
    say "$matches"
    fail=1
  fi
}

check_private_workspace_paths() {
  local matches
  matches="$(git ls-files | rg '(^|/)nodes/|(^|/)\.system/|(^|/)packages/|(^|/)rhythm/' | rg -v '^sample-workspace/nodes/' || true)"

  if [[ -n "$matches" ]]; then
    say "Tracked private workspace paths found:"
    say "$matches"
    fail=1
  fi
}

check_sidecars() {
  local matches
  matches="$(git ls-files | rg '\.(abstract|overview)\.md$|(^|/)\.DS_Store$' || true)"

  if [[ -n "$matches" ]]; then
    say "Tracked generated sidecars found:"
    say "$matches"
    fail=1
  fi
}

check_secret_shapes() {
  local matches
  matches="$(
    git grep -nE \
      '(sk-[A-Za-z0-9]{20,}|xox[baprs]-[A-Za-z0-9-]{20,}|gh[pousr]_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----)' \
      -- . ':(exclude)deps/**' ':(exclude)_build/**' ':(exclude)**/node_modules/**' || true
  )"

  if [[ -n "$matches" ]]; then
    say "Possible secret material found:"
    say "$matches"
    fail=1
  fi
}

check_private_terms() {
  local matches
  matches="$(
    git grep -nE \
      '(Roberto H\. Luna|robertohluna|\<roberto\>|NFOrce|AMEX|Lunivate|AI Masters|AI MASTERS|Ed Honour|Robert Potter|robert-potter|Brooklyn|Talha|Acme|acme|Healthtech|healthtech|MIOSA Platform|MIOSA LLC|ClinicIQ|Mosaic Effect|01-roberto|01-founder|02-platform|03-services|04-academy|04-ai-masters|04-AI MASTERS|06-partners|08-media|09-new-stuff|11-money-revenue|miosa-platform|ai-masters|money-revenue|Agency Accelerants|OS Architect|OS Accelerator|Content Creators|new-stuff|Malu|gbrain|leafwiki|OpenViking|ByteDance|Mem0|Letta|Zep|Dust|Glean|Pinecone|Weaviate|Qdrant)' \
      -- . ':(exclude)scripts/public-audit.sh' ':(exclude)deps/**' ':(exclude)_build/**' ':(exclude)**/node_modules/**' || true
  )"

  if [[ -n "$matches" ]]; then
    say "Private workspace/company terms still present:"
    say "$matches"
    fail=1
  fi
}

check_tracked_paths
check_private_workspace_paths
check_sidecars
check_secret_shapes
check_private_terms

if [[ "$fail" -eq 0 ]]; then
  say "Public audit passed."
else
  say "Public audit failed."
fi

exit "$fail"
