import { randomBytes } from "node:crypto";
import {
  chmodSync,
  mkdirSync,
  readFileSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";

const CONNECTOR_KEY_PATTERN = /^[0-9a-fA-F]{64}$/;

function validateConnectorKey(value: string, source: string): string {
  const key = value.trim();
  if (!CONNECTOR_KEY_PATTERN.test(key)) {
    throw new Error(`Invalid connector key in ${source}`);
  }
  return key;
}

export function loadOrCreateConnectorKey(
  dataDir: string,
  configuredKey?: string,
): string {
  if (configuredKey?.trim()) {
    return validateConnectorKey(configuredKey, "CONNECTOR_KEY");
  }

  mkdirSync(dataDir, { recursive: true });
  const keyPath = path.join(dataDir, "connector_key");

  try {
    const key = validateConnectorKey(readFileSync(keyPath, "utf8"), keyPath);
    if (process.platform !== "win32") chmodSync(keyPath, 0o600);
    return key;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "ENOENT") throw error;
  }

  const generated = randomBytes(32).toString("hex");

  try {
    writeFileSync(keyPath, `${generated}\n`, {
      encoding: "utf8",
      flag: "wx",
      mode: 0o600,
    });
    return generated;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "EEXIST") throw error;
    const key = validateConnectorKey(readFileSync(keyPath, "utf8"), keyPath);
    if (process.platform !== "win32") chmodSync(keyPath, 0o600);
    return key;
  }
}
