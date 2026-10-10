const ignoredStdioErrorCodes = new Set(["EIO", "EPIPE", "EBADF"]);

const consoleMethods = ["debug", "error", "info", "log", "warn"] as const;

type ConsoleMethod = (typeof consoleMethods)[number];

function getErrorCode(error: unknown): string | undefined {
  if (!error || typeof error !== "object" || !("code" in error)) {
    return undefined;
  }

  const code = (error as { code?: unknown }).code;
  return typeof code === "string" ? code : undefined;
}

function isIgnoredStdioError(error: unknown): boolean {
  const code = getErrorCode(error);
  if (code && ignoredStdioErrorCodes.has(code)) {
    return true;
  }

  return error instanceof Error && /^write (EIO|EPIPE|EBADF)\b/.test(error.message);
}

function writeFallback(message: string): void {
  try {
    process.stderr.write(`${message}\n`);
  } catch {
    // If stdio is already broken, there is nowhere reliable left to report it.
  }
}

function handleStdioError(error: unknown): void {
  if (!isIgnoredStdioError(error)) {
    writeFallback(`[BusinessOS] stdio error: ${String(error)}`);
  }
}

function handleUncaughtException(error: unknown): void {
  if (isIgnoredStdioError(error)) {
    return;
  }

  const processEvents = process as NodeJS.EventEmitter;
  processEvents.removeListener("uncaughtException", handleUncaughtException);
  throw error;
}

function wrapConsoleMethod(method: ConsoleMethod): void {
  const original = console[method].bind(console);

  console[method] = (...args: unknown[]) => {
    try {
      original(...args);
    } catch (error) {
      if (!isIgnoredStdioError(error)) {
        writeFallback(`[BusinessOS] console.${method} failed: ${String(error)}`);
      }
    }
  };
}

export function installSafeConsole(): void {
  const globalState = globalThis as typeof globalThis & {
    __businessosSafeConsoleInstalled?: boolean;
  };

  if (globalState.__businessosSafeConsoleInstalled) {
    return;
  }

  globalState.__businessosSafeConsoleInstalled = true;

  process.stdout.on("error", handleStdioError);
  process.stderr.on("error", handleStdioError);
  const processEvents = process as NodeJS.EventEmitter;
  processEvents.prependListener("uncaughtException", handleUncaughtException);

  for (const method of consoleMethods) {
    wrapConsoleMethod(method);
  }
}
