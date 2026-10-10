#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

: "${OSS_REMOTE_URL:?Set OSS_REMOTE_URL to the public repository URL}"
: "${OSS_VERIFIED_TREE:?Stage and verify the public projection before publishing}"
tree="$(git write-tree)"
if [[ "$tree" != "$OSS_VERIFIED_TREE" ]]; then
  echo "Refusing a source tree changed since public projection verification." >&2
  exit 1
fi
# Only reviewed public roots may anchor the public history. Never use HEAD
# as a parent: it belongs to the private source repository.
trusted_roots="${OSS_TRUSTED_ROOTS:-8e3e6dc0ebf3695d1a60517d525b87850922f76e}"
git fetch --no-tags "$OSS_REMOTE_URL" refs/heads/main
parent="$(git rev-parse FETCH_HEAD)"
roots="$(git rev-list --max-parents=0 "$parent")"
while IFS= read -r root; do
  case " $trusted_roots " in
    *" $root "*) ;;
    *) echo "Refusing unreviewed public history root: $root" >&2; exit 1 ;;
  esac
done <<< "$roots"

temporary="$(mktemp -d "${TMPDIR:-/tmp}/businessos-oss-history.XXXXXX")"
trap 'rm -rf "$temporary"' EXIT
git rev-list HEAD | sort > "$temporary/private"
git rev-list "$parent" | sort > "$temporary/public"
if [[ -n "$(comm -12 "$temporary/private" "$temporary/public")" ]]; then
  echo "Refusing public history that shares private commit ancestry." >&2
  exit 1
fi

if [[ "$tree" == "$(git rev-parse "$parent^{tree}")" ]]; then
  echo "No public source changes; existing public history preserved."
  exit 0
fi

# Derive public descriptions from product areas, not potentially private
# client names or internal notes in source commit messages.
areas=()
for area in backend frontend desktop engine docs tooling; do
  case "$area" in
    backend) path=desktop/backend-go ;;
    frontend) path=frontend ;;
    desktop) path=desktop ;;
    engine) path=optimal-engine ;;
    docs) path=docs ;;
    tooling) path=scripts ;;
  esac
  excluded=()
  if [[ "$area" == desktop ]]; then
    excluded=(':(exclude)desktop/backend-go')
  fi
  if ! git diff --quiet "$parent" "$tree" -- "$path" "${excluded[@]}"; then
    areas+=("$area")
  fi
done
description="${areas[*]:-public source}"
files="$(git diff --name-only "$parent" "$tree" | wc -l | tr -d ' ')"
commit="$(printf 'sync: update %s (%s)\n\nPublish verified open-source changes across %s files.\n' \
  "$description" "$(date -u +%Y-%m-%d)" "$files" | git commit-tree "$tree" -p "$parent")"
[[ "$(git show -s --format=%P "$commit")" == "$parent" ]]
# A normal push preserves history and rejects a concurrent public update.
git push "$OSS_REMOTE_URL" "$commit:refs/heads/main"
echo "Published public commit $commit; prior public commits preserved."
