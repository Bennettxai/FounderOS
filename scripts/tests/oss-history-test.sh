#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
temporary="$(mktemp -d "${TMPDIR:-/tmp}/businessos-oss-test.XXXXXX")"
trap 'rm -rf "$temporary"' EXIT
export GIT_AUTHOR_NAME="OSS Test" GIT_AUTHOR_EMAIL="oss-test@example.invalid"
export GIT_COMMITTER_NAME="$GIT_AUTHOR_NAME" GIT_COMMITTER_EMAIL="$GIT_AUTHOR_EMAIL"
git init -q --bare "$temporary/public.git"
git init -q "$temporary/source"
cd "$temporary/source"
mkdir -p scripts desktop/backend-go
cp "$ROOT/scripts/publish-oss.sh" scripts/publish-oss.sh
printf 'private data\n' > private.txt
git add -A
git commit -qm 'private source history'
private="$(git rev-parse HEAD)"
git rm -q private.txt
printf 'first version\n' > desktop/backend-go/example.txt
git add -A
root="$(printf 'reviewed public snapshot\n' | git commit-tree "$(git write-tree)")"
git push -q "$temporary/public.git" "$root:refs/heads/main"
export OSS_REMOTE_URL="$temporary/public.git" OSS_TRUSTED_ROOTS="$root"

publish() {
  OSS_VERIFIED_TREE="$(git write-tree)" bash scripts/publish-oss.sh
}

printf 'second version\n' > desktop/backend-go/example.txt
git add -A
publish
first="$(git --git-dir="$temporary/public.git" rev-parse main)"
[[ "$(git show -s --format=%P "$first")" == "$root" ]]
[[ "$(git --git-dir="$temporary/public.git" rev-list --count main)" == 2 ]]
if git --git-dir="$temporary/public.git" cat-file -e "$first:private.txt" 2>/dev/null; then
  echo 'Private file leaked into public tree' >&2; exit 1
fi
if git --git-dir="$temporary/public.git" merge-base --is-ancestor "$private" main 2>/dev/null; then
  echo 'Private ancestry leaked into public history' >&2; exit 1
fi

publish
[[ "$(git --git-dir="$temporary/public.git" rev-parse main)" == "$first" ]]
printf 'third version\n' > desktop/backend-go/example.txt
git add -A
publish
[[ "$(git --git-dir="$temporary/public.git" rev-list --count main)" == 3 ]]

if OSS_TRUSTED_ROOTS=unreviewed publish; then
  echo 'Unreviewed root was accepted' >&2; exit 1
fi
if OSS_VERIFIED_TREE="$root" bash scripts/publish-oss.sh; then
  echo 'Unverified tree was accepted' >&2; exit 1
fi
git push -q --force "$temporary/public.git" "$private:refs/heads/main"
if OSS_TRUSTED_ROOTS="$private" publish; then
  echo 'Private ancestry was accepted' >&2; exit 1
fi
echo 'OSS history tests passed'
