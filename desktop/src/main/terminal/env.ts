import fs from "fs";
import os from "os";
import path from "path";

function isUsableShell(shell?: string): shell is string {
  if (!shell) return false;
  try {
    fs.accessSync(shell, fs.constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

function resolveExecutable(command?: string): string | undefined {
  if (!command) return undefined;
  if (path.isAbsolute(command)) return isUsableShell(command) ? command : undefined;
  for (const directory of terminalPath().split(path.delimiter)) {
    const candidate = path.join(directory, command);
    if (isUsableShell(candidate)) return candidate;
  }
  return undefined;
}

function unique(values: string[]): string[] {
  return Array.from(new Set(values.filter(Boolean)));
}

export function terminalPath(basePath = process.env.PATH || ""): string {
  if (process.platform === "win32") return basePath;

  const home = os.homedir();
  const dirs = [
    basePath,
    "/opt/homebrew/bin",
    "/opt/homebrew/sbin",
    "/usr/local/bin",
    "/usr/local/sbin",
    "/usr/bin",
    "/bin",
    "/usr/sbin",
    "/sbin",
    path.join(home, ".local", "bin"),
    path.join(home, ".npm-global", "bin"),
    path.join(home, ".bun", "bin"),
    path.join(home, ".deno", "bin"),
    path.join(home, ".cargo", "bin"),
  ].filter(Boolean);

  return Array.from(new Set(dirs.join(path.delimiter).split(path.delimiter)))
    .filter(Boolean)
    .join(path.delimiter);
}

export function resolveDefaultShell(): string {
  if (process.platform === "win32") {
    return process.env.COMSPEC || "powershell.exe";
  }

  return resolveShellCandidates()[0] || "/bin/sh";
}

export function resolveShellCandidates(preferredShell?: string): string[] {
  if (process.platform === "win32") {
    return unique([preferredShell || "", process.env.COMSPEC || "powershell.exe"]);
  }

  const detectedShell =
    preferredShell === "auto"
      ? process.env.TMUX
        ? "tmux"
        : process.env.SHELL
      : preferredShell;
  const candidates = unique([
    detectedShell || "",
    process.env.SHELL,
    "/bin/zsh",
    "/bin/bash",
    "/bin/sh",
  ].filter(Boolean) as string[]);

  return unique(candidates.map(resolveExecutable).filter(Boolean) as string[]);
}

export function loginShellArgs(shell: string): string[] {
  if (process.platform === "win32") return [];

  const name = path.basename(shell);
  if (name === "fish") return ["--login"];
  if (name === "zsh" || name === "bash" || name === "sh") return ["-l"];
  return [];
}

export function buildPtyEnv(
  shell: string,
  extraEnv: Record<string, string> = {},
): Record<string, string> {
  const merged = {
    ...process.env,
    ...extraEnv,
  } as Record<string, string>;

  const env: Record<string, string> = {};
  for (const [key, value] of Object.entries(merged)) {
    if (typeof value === "string" && !value.includes("\0")) {
      env[key] = value;
    }
  }

  env.SHELL = shell;
  env.HOME = env.HOME || os.homedir();
  env.USER = env.USER || os.userInfo().username;
  env.LOGNAME = env.LOGNAME || env.USER;
  env.TERM = "xterm-256color";
  env.COLORTERM = env.COLORTERM || "truecolor";
  env.PATH = terminalPath(env.PATH);

  return env;
}
