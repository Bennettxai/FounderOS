#!/usr/bin/env bash
# Rebuilds testdata/founderos-os.seed.sql.gz: FounderOS v1's own seed (lib/db.ts +
# lib/seed.ts), dumped as SQL. Run with the Node whose ABI matches the FounderOS
# v1 checkout's installed better-sqlite3; never npm install/rebuild there.
#
#   ./build-fixture.sh ~/path/to/FounderOS-v1
#
# Sanitised on the way out: proposal access codes are real StatiCrypt gate
# codes, so they are replaced with fake ones. bank/ledger/paykit are not
# included (the PayKit seed row is a reconstruction of real customers); the
# tests synthesise those side databases instead.
#
# The checked-in dump is in the public demo world: ventures Vantage and
# Launchpad Cohort, operator Alex, demo agent ids (postly-publisher,
# adsmith-creative, ...), and dummy names for every person, handle and
# proposal recipient. A rebuild from any other seed must get the same
# treatment before it is committed; the Go roster test
# (api.TestRosterMetasAreTheSeededAgentRows) pins the agent rows.
set -euo pipefail
BOS="${1:?path to the FounderOS v1 checkout}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"; rm -f "$BOS/.tmp-etl-fixture.ts"' EXIT

cat > "$BOS/.tmp-etl-fixture.ts" <<'TS'
import { openDb } from './lib/db';
import { seedDatabase } from './lib/seed';
const db = openDb(process.argv[2]);
seedDatabase(db);
db.close();
TS
(cd "$BOS" && FOUNDEROS_OS_DB="$TMP/founderos-os.db" DATA_DIR="$TMP" npx tsx ./.tmp-etl-fixture.ts "$TMP/founderos-os.db")

sqlite3 "$TMP/founderos-os.db" "PRAGMA journal_mode=DELETE; UPDATE proposals SET access_code = 'fixture-code-' || rowid WHERE access_code <> '';"
sqlite3 "$TMP/founderos-os.db" .dump | gzip -9n > "$HERE/founderos-os.seed.sql.gz"
echo "wrote $HERE/founderos-os.seed.sql.gz"
