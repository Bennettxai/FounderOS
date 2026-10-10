#!/usr/bin/env bash
set -euo pipefail

echo "Direct pushes from the private checkout are disabled." >&2
echo "Use the verified Sync to open-source mirror workflow:" >&2
echo "https://github.com/Miosa-osa/businessos-5/actions/workflows/sync-oss.yml" >&2
exit 1
