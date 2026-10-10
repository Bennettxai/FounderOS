import assert from "node:assert/strict";
import { mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

import { loadOrCreateConnectorKey } from "./credential-key.ts";

function withTempDir(run: (dir: string) => void): void {
  const dir = mkdtempSync(path.join(tmpdir(), "businessos-engine-key-"));
  try {
    run(dir);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

test("creates and reuses a protected connector key", () => {
  withTempDir((dir) => {
    const first = loadOrCreateConnectorKey(dir);
    const second = loadOrCreateConnectorKey(dir);
    const keyPath = path.join(dir, "connector_key");

    assert.match(first, /^[0-9a-f]{64}$/);
    assert.equal(second, first);
    assert.equal(readFileSync(keyPath, "utf8").trim(), first);

    if (process.platform !== "win32") {
      assert.equal(statSync(keyPath).mode & 0o777, 0o600);
    }
  });
});

test("uses a valid configured key without writing it to disk", () => {
  withTempDir((dir) => {
    const configured = "ab".repeat(32);
    assert.equal(loadOrCreateConnectorKey(dir, configured), configured);
    assert.throws(() => readFileSync(path.join(dir, "connector_key")));
  });
});

test("rejects an invalid persisted key instead of silently rotating it", () => {
  withTempDir((dir) => {
    writeFileSync(path.join(dir, "connector_key"), "broken-key\n", { mode: 0o600 });
    assert.throws(() => loadOrCreateConnectorKey(dir), /invalid connector key/i);
  });
});
