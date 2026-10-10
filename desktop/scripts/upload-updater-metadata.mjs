#!/usr/bin/env node
import { existsSync } from "node:fs";
import { readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const owner = process.env.BUSINESSOS_RELEASE_OWNER || "Miosa-osa";
const repo = process.env.BUSINESSOS_RELEASE_REPO || "businessos-5";
const releaseVersion =
  process.env.BUSINESSOS_RELEASE_VERSION ||
  JSON.parse(await readFile(path.join(root, "package.json"), "utf8")).version;
const tag = process.env.BUSINESSOS_RELEASE_TAG || `v${releaseVersion}`;
const makeDir =
  process.env.BUSINESSOS_MAKE_DIR || path.join(root, "out", "make");
const metadataPath =
  process.env.BUSINESSOS_UPDATE_METADATA ||
  path.join(makeDir, "latest-mac.yml");

if (!existsSync(metadataPath)) {
  console.error(`Missing updater metadata at ${metadataPath}`);
  console.error("Run npm run release:metadata first.");
  process.exit(1);
}

const result = spawnSync(
  "gh",
  [
    "release",
    "upload",
    tag,
    metadataPath,
    "--repo",
    `${owner}/${repo}`,
    "--clobber",
  ],
  { stdio: "inherit" },
);

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);
