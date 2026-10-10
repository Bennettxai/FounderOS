#!/usr/bin/env node
import { createHash } from "node:crypto";
import { promises as fs } from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const packageJson = JSON.parse(
  await fs.readFile(path.join(root, "package.json"), "utf8"),
);
const version = process.env.BUSINESSOS_RELEASE_VERSION || packageJson.version;
const makeDir =
  process.env.BUSINESSOS_MAKE_DIR || path.join(root, "out", "make");
const outputPath =
  process.env.BUSINESSOS_UPDATE_METADATA ||
  path.join(makeDir, "latest-mac.yml");

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await walk(full)));
    } else {
      files.push(full);
    }
  }
  return files;
}

async function sha512(file) {
  const hash = createHash("sha512");
  const handle = await fs.open(file, "r");
  try {
    for await (const chunk of handle.createReadStream()) {
      hash.update(chunk);
    }
  } finally {
    await handle.close();
  }
  return hash.digest("base64");
}

function yamlString(value) {
  return JSON.stringify(String(value));
}

let files;
try {
  files = await walk(makeDir);
} catch {
  console.error(`No make output found at ${makeDir}`);
  process.exit(1);
}

const allArtifacts = files
  .filter((file) => /\.(zip|dmg)$/i.test(file))
  .filter((file) => !path.basename(file).startsWith("."))
  .sort((a, b) => {
    const extA = path.extname(a).toLowerCase();
    const extB = path.extname(b).toLowerCase();
    if (extA === ".zip" && extB !== ".zip") return -1;
    if (extB === ".zip" && extA !== ".zip") return 1;
    return a.localeCompare(b);
  });

const artifacts = allArtifacts.filter((file) =>
  path.basename(file).includes(version),
);

if (artifacts.length === 0) {
  console.error(
    `No artifacts for version ${version} found under ${makeDir}. Found: ${allArtifacts
      .map((file) => path.basename(file))
      .join(", ")}`,
  );
  process.exit(1);
}

const zip = artifacts.find((file) => path.extname(file).toLowerCase() === ".zip");
if (!zip) {
  console.error(
    `No mac ZIP found under ${makeDir}. Electron mac auto-update requires a ZIP artifact.`,
  );
  process.exit(1);
}

const rows = [];
for (const file of artifacts) {
  const stat = await fs.stat(file);
  rows.push({
    url: path.basename(file),
    sha512: await sha512(file),
    size: stat.size,
  });
}

const primary = rows.find((row) => row.url === path.basename(zip));
const yml = [
  `version: ${yamlString(version)}`,
  "files:",
  ...rows.flatMap((row) => [
    `  - url: ${yamlString(row.url)}`,
    `    sha512: ${yamlString(row.sha512)}`,
    `    size: ${row.size}`,
  ]),
  `path: ${yamlString(primary.url)}`,
  `sha512: ${yamlString(primary.sha512)}`,
  `releaseDate: ${yamlString(new Date().toISOString())}`,
  "",
].join("\n");

await fs.mkdir(path.dirname(outputPath), { recursive: true });
await fs.writeFile(outputPath, yml, "utf8");
console.log(`Wrote ${outputPath}`);
