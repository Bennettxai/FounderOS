#!/usr/bin/env bash
# =============================================================================
# Add collaborators to the Agency MIOSA BusinessOS repo.
#
# TEMPLATE - Roberto runs this. It is not executed automatically.
#
# Usage:
#   ./scripts/add-collaborators.sh <github-username> [<github-username> ...]
#
# Example:
#   ./scripts/add-collaborators.sh robert-handle floyd-handle
#
# Requires the GitHub CLI (gh) authenticated as a repo admin:
#   gh auth status
#
# Permission level defaults to "push" (write). Override with PERMISSION env var:
#   PERMISSION=maintain ./scripts/add-collaborators.sh robert-handle
# Valid: pull, triage, push, maintain, admin
# =============================================================================
set -euo pipefail

REPO="robertohluna/agency-miosa-businessos"
PERMISSION="${PERMISSION:-push}"

if [ "$#" -eq 0 ]; then
  echo "Usage: $0 <github-username> [<github-username> ...]" >&2
  echo "Adds each user to $REPO with '$PERMISSION' permission." >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh (GitHub CLI) is not installed - https://cli.github.com/" >&2
  exit 1
fi

for user in "$@"; do
  echo "==> Inviting '$user' to $REPO ($PERMISSION)..."
  gh api \
    --method PUT \
    -H "Accept: application/vnd.github+json" \
    "/repos/$REPO/collaborators/$user" \
    -f permission="$PERMISSION"
  echo "    invite sent to '$user'"
done

echo "Done. Invited users must accept the GitHub invitation to gain access."
