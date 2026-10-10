#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

if (process.platform !== "darwin") {
  process.exit(0);
}

const npm = process.platform === "win32" ? "npm.cmd" : "npm";

for (const moduleName of ["macos-alias", "fs-xattr"]) {
  const result = spawnSync(npm, ["rebuild", moduleName], {
    stdio: "inherit",
    env: process.env,
  });

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}
