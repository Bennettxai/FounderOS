import {
  ipcMain,
  app,
  shell,
  dialog,
  desktopCapturer,
  screen,
  session,
} from "electron";
import { BackendManager } from "../backend/manager";
import { getMainWindow } from "../window";
import {
  checkForUpdates,
  downloadUpdate,
  getUpdateRuntimeInfo,
  installUpdate,
} from "../updater/auto-update";
import {
  setupDatabaseHandlers,
  initializeDatabaseSystem,
  startSync,
  stopSync,
} from "./database";
import { setupTerminalHandlers } from "../terminal/pty-manager";
import { promises as fs } from "fs";
import { existsSync, mkdirSync } from "fs";
import { exec } from "child_process";
import path from "path";
import os from "os";

// --- BusinessOS home directory -------------------------------------------
// The folder the built-in terminal opens in and agents cd into for context.
// Resolution order: BUSINESSOS_HOME env -> user override (persisted) ->
// dev repo root -> ~/BusinessOS (created for packaged users) -> home.
// Previously a path was hardcoded to Roberto's dev machine
// (~/Desktop/OptimalOS/businessos), which was wrong for every other user.
let businessOSHomeOverride: string | null = null;
let bosSettingsLoaded = false;

function bosSettingsPath(): string {
  return path.join(app.getPath("userData"), "businessos-settings.json");
}

async function loadBosSettings(): Promise<void> {
  if (bosSettingsLoaded) return;
  bosSettingsLoaded = true;
  try {
    const raw = await fs.readFile(bosSettingsPath(), "utf8");
    const obj = JSON.parse(raw) as { home?: string };
    if (obj.home) businessOSHomeOverride = obj.home;
  } catch {
    // no settings file yet - fine
  }
}

export async function resolveBusinessOSHomeDir(): Promise<string> {
  await loadBosSettings();
  return resolveBusinessOSHome();
}

function resolveBusinessOSHome(): string {
  const envHome = process.env.BUSINESSOS_HOME;
  if (envHome && existsSync(envHome)) return envHome;
  if (businessOSHomeOverride && existsSync(businessOSHomeOverride)) {
    return businessOSHomeOverride;
  }
  // Dev: the repo root (one level up from desktop/), which holds CLAUDE.md.
  if (!app.isPackaged) {
    const repo = path.resolve(app.getAppPath(), "..");
    if (
      existsSync(path.join(repo, "CLAUDE.md")) ||
      existsSync(path.join(repo, "package.json"))
    ) {
      return repo;
    }
  }
  // Packaged: a real folder the user owns, created on first use.
  const home = path.join(os.homedir(), "BusinessOS");
  try {
    mkdirSync(home, { recursive: true });
    return home;
  } catch {
    return os.homedir();
  }
}

// Re-export database functions for use in main process
export { initializeDatabaseSystem, startSync, stopSync };

// ── Optimal Engine local mirror ──────────────────────────────────────────────
// Per-workspace engine connection cached on the user's machine so the desktop
// can write to a LOCAL engine (localhost) that the cloud backend cannot reach.
interface EngineConn {
  base_url: string;
  api_key: string;
  workspace: string;
  enabled: boolean;
}
const engineConfigs = new Map<string, EngineConn>();
let engineConfigsLoaded = false;

function engineConfigPath(): string {
  return path.join(app.getPath("userData"), "engine-configs.json");
}
async function loadEngineConfigs(): Promise<void> {
  if (engineConfigsLoaded) return;
  engineConfigsLoaded = true;
  try {
    const raw = await fs.readFile(engineConfigPath(), "utf8");
    const obj = JSON.parse(raw) as Record<string, EngineConn>;
    for (const [k, v] of Object.entries(obj)) engineConfigs.set(k, v);
  } catch {
    /* no file yet */
  }
}
async function saveEngineConfigs(): Promise<void> {
  const obj: Record<string, EngineConn> = {};
  for (const [k, v] of engineConfigs) obj[k] = v;
  try {
    await fs.writeFile(engineConfigPath(), JSON.stringify(obj), "utf8");
  } catch {
    /* best effort */
  }
}
function isLocalHostUrl(url: string): boolean {
  try {
    const h = new URL(url).hostname;
    return (
      h === "localhost" ||
      h === "127.0.0.1" ||
      h === "::1" ||
      h.endsWith(".local")
    );
  } catch {
    return false;
  }
}

// ── Bundled engine model connection ──────────────────────────────────────────
// The bundled OptimalEngine needs a model for its AI features. Users bring their
// OWN model: a cloud API key (Anthropic / OpenAI) or a local Ollama URL. The
// choice is persisted on this machine so the engine can use it. Kept parallel to
// the engine-configs pattern above, deliberately NOT touching those functions.
interface ModelConfig {
  provider: "anthropic" | "openai" | "ollama" | "";
  apiKey?: string;
  ollamaUrl?: string;
}
let modelConfig: ModelConfig | null = null;
let modelConfigLoaded = false;

function modelConfigPath(): string {
  return path.join(app.getPath("userData"), "model-config.json");
}
async function loadModelConfig(): Promise<ModelConfig> {
  if (modelConfigLoaded && modelConfig) return modelConfig;
  modelConfigLoaded = true;
  try {
    const raw = await fs.readFile(modelConfigPath(), "utf8");
    modelConfig = JSON.parse(raw) as ModelConfig;
  } catch {
    modelConfig = { provider: "" };
  }
  return modelConfig;
}
async function saveModelConfig(cfg: ModelConfig): Promise<void> {
  modelConfig = cfg;
  modelConfigLoaded = true;
  await fs.writeFile(modelConfigPath(), JSON.stringify(cfg, null, 2), "utf8");
}

// ── Bundled engine status accessor ───────────────────────────────────────────
// The live EngineManager instance lives in main/index.ts, out of reach of this
// module. main registers a lightweight status provider here so engine:status can
// report the real running/url/port. If nothing is registered (e.g. the wiring is
// not in place yet), engine:status falls back to the OPTIMAL_ENGINE_URL env var
// that main sets when the bundled engine becomes healthy - so the panel still
// shows a truthful status either way.
interface EngineStatusProvider {
  isRunning(): boolean;
  isAvailable(): boolean;
  getUrl(): string;
  getPort(): number;
  start(): Promise<boolean>;
  stop(): Promise<void>;
}
let engineStatusProvider: EngineStatusProvider | null = null;
export function setEngineStatusProvider(p: EngineStatusProvider | null): void {
  engineStatusProvider = p;
}

function bundledEngineUrl(): string {
  if (engineStatusProvider?.isRunning()) {
    return engineStatusProvider.getUrl();
  }
  return process.env.OPTIMAL_ENGINE_URL || "";
}

function resolveMemoryEngineConfig(workspaceId: string): EngineConn | null {
  const cfg = engineConfigs.get(workspaceId);
  if (cfg?.enabled && cfg.base_url && isLocalHostUrl(cfg.base_url)) {
    return cfg;
  }

  const baseUrl = bundledEngineUrl();
  if (!baseUrl || !isLocalHostUrl(baseUrl)) return null;
  return {
    base_url: baseUrl,
    api_key: "",
    workspace: workspaceId,
    enabled: true,
  };
}

// The bundled engine's per-user data directory. Kept in sync with EngineManager,
// which derives its data dir the same way (app.getPath("userData")/optimal-engine).
function engineDataDir(): string {
  return path.join(app.getPath("userData"), "optimal-engine");
}

async function collectDiagnostics(backendManager: BackendManager | null) {
  await Promise.allSettled([loadBosSettings(), loadEngineConfigs()]);

  const userData = app.getPath("userData");
  const bosHome = resolveBusinessOSHome();
  const backend = {
    running: backendManager?.isRunning() ?? false,
    port: backendManager?.getPort() ?? 0,
    url: backendManager?.getUrl() ?? "",
  };
  const engine = (() => {
    const dataDir = engineDataDir();
    if (engineStatusProvider) {
      return {
        running: engineStatusProvider.isRunning(),
        available: engineStatusProvider.isAvailable(),
        url: engineStatusProvider.getUrl(),
        port: engineStatusProvider.getPort(),
        dataDir,
      };
    }
    const envUrl = process.env.OPTIMAL_ENGINE_URL || "";
    let port = 0;
    try {
      if (envUrl) port = Number(new URL(envUrl).port) || 0;
    } catch {
      /* malformed url */
    }
    return { running: Boolean(envUrl), url: envUrl, port, dataDir };
  })();
  const engineWorkspaces = Array.from(engineConfigs.entries()).map(
    ([workspaceId, cfg]) => ({
      workspaceId,
      base_url: cfg.base_url,
      workspace: cfg.workspace,
      enabled: cfg.enabled,
      local: isLocalHostUrl(cfg.base_url),
      has_api_key: Boolean(cfg.api_key),
    }),
  );
  const terminal = {
    envShell: process.env.SHELL || "",
    path: process.env.PATH || "",
    businessOSHome: bosHome,
  };

  return {
    generatedAt: new Date().toISOString(),
    app: {
      version: app.getVersion(),
      isPackaged: app.isPackaged,
      platform: process.platform,
      arch: process.arch,
      appPath: app.getAppPath(),
      userData,
    },
    backend,
    engine,
    engineWorkspaces,
    terminal,
    paths: {
      businessOSHome: bosHome,
      settings: bosSettingsPath(),
      engineConfig: engineConfigPath(),
      modelConfig: modelConfigPath(),
    },
  };
}

/**
 * Set up all IPC handlers for communication with the renderer process
 */
export function setupIpcHandlers(backendManager: BackendManager | null): void {
  // Resilient registration: a failure in one subsystem (e.g. the node-pty
  // native module failing to load in a dev Electron, or the local SQLite setup)
  // must NOT abort registration of every handler that follows it. Before, a
  // throw here left shell:open-external + engine:* + dialogs unregistered, which
  // broke "Sign in with Google" (openExternal) and much more.
  try {
    setupDatabaseHandlers();
  } catch (e) {
    console.error("[ipc] setupDatabaseHandlers failed (continuing):", e);
  }
  try {
    setupTerminalHandlers();
  } catch (e) {
    console.error("[ipc] setupTerminalHandlers failed (continuing):", e);
  }
  // App info
  ipcMain.handle("app:get-version", () => {
    return app.getVersion();
  });

  ipcMain.handle("app:get-platform", () => {
    return {
      platform: process.platform,
      arch: process.arch,
      isPackaged: app.isPackaged,
    };
  });

  ipcMain.handle("app:get-path", (_, name: string) => {
    return app.getPath(name as any);
  });

  // BusinessOS home directory (terminal cwd + agent working dir).
  ipcMain.handle("app:get-businessos-home", async () => {
    await loadBosSettings();
    return resolveBusinessOSHome();
  });
  ipcMain.handle("app:set-businessos-home", async (_, dir: string) => {
    await loadBosSettings();
    businessOSHomeOverride = dir;
    try {
      await fs.writeFile(
        bosSettingsPath(),
        JSON.stringify({ home: dir }, null, 2),
        "utf8",
      );
      return { ok: true, home: dir };
    } catch (e) {
      return { ok: false, error: String(e) };
    }
  });

  // Backend status
  ipcMain.handle("backend:get-status", () => {
    return {
      running: backendManager?.isRunning() ?? false,
      port: backendManager?.getPort() ?? 0,
      url: backendManager?.getUrl() ?? "",
    };
  });

  ipcMain.handle("backend:get-url", () => {
    return backendManager?.getUrl() ?? "http://localhost:8001";
  });

  ipcMain.handle("backend:restart", async () => {
    if (backendManager) {
      await backendManager.restart();
      return true;
    }
    return false;
  });

  ipcMain.handle("diagnostics:collect", async () => {
    return collectDiagnostics(backendManager);
  });

  // Network status
  ipcMain.handle("network:get-status", () => {
    // Check if online by attempting to reach the remote server
    return {
      online: true, // Simplified - in production, implement actual check
    };
  });

  // Optimal Engine connectivity test, run from the main process so it reaches
  // engines on the user's own machine (e.g. http://localhost:4200) that the
  // cloud backend cannot. Local-first: the desktop reaches the local engine.
  ipcMain.handle("engine:test", async (_, baseUrl: string, apiKey?: string) => {
    if (!baseUrl) return { reachable: false, message: "No engine URL set" };
    const url = baseUrl.replace(/\/+$/, "") + "/api/health";
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 5000);
    try {
      const res = await fetch(url, {
        headers: apiKey ? { Authorization: `Bearer ${apiKey}` } : {},
        signal: controller.signal,
      });
      return {
        reachable: res.ok,
        status: res.status,
        message: res.ok ? "Connected" : `Engine returned HTTP ${res.status}`,
      };
    } catch (e) {
      return {
        reachable: false,
        message: e instanceof Error ? e.message : "Engine unreachable",
      };
    } finally {
      clearTimeout(timer);
    }
  });

  // List the workspaces an Optimal Engine has, from the user's machine, so
  // BusinessOS can detect them and offer to create matching workspaces.
  ipcMain.handle(
    "engine:workspaces",
    async (_, baseUrl: string, apiKey?: string) => {
      if (!baseUrl)
        return { ok: false, workspaces: [], message: "No engine URL" };
      const url = baseUrl.replace(/\/+$/, "") + "/api/workspaces?status=active";
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), 6000);
      try {
        const res = await fetch(url, {
          headers: apiKey ? { Authorization: `Bearer ${apiKey}` } : {},
          signal: controller.signal,
        });
        if (!res.ok) {
          return { ok: false, workspaces: [], message: `HTTP ${res.status}` };
        }
        const raw: unknown = await res.json();
        const list = Array.isArray(raw)
          ? raw
          : ((raw as { workspaces?: unknown[]; data?: unknown[] })
              ?.workspaces ??
            (raw as { data?: unknown[] })?.data ??
            []);
        const workspaces = (list as Record<string, unknown>[]).map((w) => ({
          id: String(w.id ?? w.slug ?? ""),
          slug: String(w.slug ?? w.id ?? ""),
          name: String(w.name ?? w.slug ?? w.id ?? ""),
        }));
        return { ok: true, workspaces, message: "ok" };
      } catch (e) {
        return {
          ok: false,
          workspaces: [],
          message: e instanceof Error ? e.message : "Engine unreachable",
        };
      } finally {
        clearTimeout(timer);
      }
    },
  );

  // Create a workspace in a local or user-supplied Optimal Engine. The
  // renderer lists first and only calls this for a missing slug, so ordinary
  // workspace creation is idempotent across app restarts.
  ipcMain.handle(
    "engine:createWorkspace",
    async (
      _,
      baseUrl: string,
      apiKey: string | undefined,
      workspace: { slug: string; name: string; description?: string },
    ) => {
      if (!baseUrl || !workspace?.slug || !workspace?.name) {
        return { ok: false, message: "Engine URL, slug, and name are required" };
      }
      const url = baseUrl.replace(/\/+$/, "") + "/api/workspaces";
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), 8000);
      try {
        const res = await fetch(url, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}),
          },
          body: JSON.stringify(workspace),
          signal: controller.signal,
        });
        const body = (await res.json().catch(() => ({}))) as Record<string, unknown>;
        if (!res.ok) {
          return {
            ok: false,
            message: String(body.error ?? `HTTP ${res.status}`),
          };
        }
        return {
          ok: true,
          workspace: {
            id: String(body.id ?? body.slug ?? workspace.slug),
            slug: String(body.slug ?? workspace.slug),
            name: String(body.name ?? workspace.name),
          },
          message: "created",
        };
      } catch (e) {
        return {
          ok: false,
          message: e instanceof Error ? e.message : "Engine unreachable",
        };
      } finally {
        clearTimeout(timer);
      }
    },
  );

  // Cache a workspace's engine connection on this machine (called on Save), so
  // the desktop can write to a local engine the cloud backend cannot reach.
  ipcMain.handle(
    "engine:setConfig",
    async (_, workspaceId: string, cfg: EngineConn) => {
      if (!workspaceId || !cfg) return { ok: false };
      await loadEngineConfigs();
      if (cfg.enabled && cfg.base_url) engineConfigs.set(workspaceId, cfg);
      else engineConfigs.delete(workspaceId);
      await saveEngineConfigs();
      return { ok: true };
    },
  );

  // Persist the bundled engine's model connection on this machine. The user
  // brings their own model: a cloud API key (Anthropic / OpenAI) or a local
  // Ollama URL. Returns { ok } so the settings panel can confirm the save.
  ipcMain.handle("engine:setModelConfig", async (_, cfg: ModelConfig) => {
    if (!cfg || typeof cfg !== "object") return { ok: false };
    const clean: ModelConfig = {
      provider: cfg.provider ?? "",
      apiKey: cfg.apiKey?.trim() || undefined,
      ollamaUrl: cfg.ollamaUrl?.trim() || undefined,
    };
    try {
      await saveModelConfig(clean);
      return { ok: true };
    } catch (e) {
      return { ok: false, error: e instanceof Error ? e.message : String(e) };
    }
  });

  // Read back the stored model connection so the panel can show current status.
  ipcMain.handle("engine:getModelConfig", async () => {
    try {
      return await loadModelConfig();
    } catch {
      return { provider: "" } as ModelConfig;
    }
  });

  // Report the bundled engine's status so users can SEE it is running and know
  // where their data lives. Prefers the live EngineManager (registered by main);
  // falls back to the OPTIMAL_ENGINE_URL env var main sets when healthy.
  ipcMain.handle("engine:status", () => {
    const dataDir = engineDataDir();
    if (engineStatusProvider) {
      return {
        running: engineStatusProvider.isRunning(),
        url: engineStatusProvider.getUrl(),
        port: engineStatusProvider.getPort(),
        dataDir,
      };
    }
    const envUrl = process.env.OPTIMAL_ENGINE_URL || "";
    let port = 0;
    try {
      if (envUrl) port = Number(new URL(envUrl).port) || 0;
    } catch {
      /* malformed url - leave port at 0 */
    }
    return { running: Boolean(envUrl), available: false, url: envUrl, port, dataDir };
  });

  // The built-in engine is a separate, private on-device runtime. Keep its
  // lifecycle explicit so users can decide whether to run it alongside, or
  // instead of, an external Optimal Engine connection.
  ipcMain.handle("engine:start", async () => {
    if (!engineStatusProvider) {
      return { ok: false, message: "Built-in engine manager is unavailable" };
    }
    if (!engineStatusProvider.isAvailable()) {
      return {
        ok: false,
        message: "The bundled engine runtime is not included in this development build",
      };
    }
    const ok = await engineStatusProvider.start();
    return {
      ok,
      message: ok
        ? "Built-in engine is running"
        : "The built-in engine could not start. Check the desktop logs and retry.",
    };
  });

  ipcMain.handle("engine:stop", async () => {
    if (!engineStatusProvider) {
      return { ok: false, message: "Built-in engine manager is unavailable" };
    }
    await engineStatusProvider.stop();
    return { ok: true, message: "Built-in engine stopped" };
  });

  // Open the bundled engine's data directory in Finder/Explorer so users can
  // reach their files. Creates it first so the reveal never fails on a fresh
  // install that has not started the engine yet.
  ipcMain.handle("engine:reveal-data", async () => {
    const dataDir = engineDataDir();
    try {
      mkdirSync(dataDir, { recursive: true });
    } catch {
      /* best effort - openPath still reports the real error below */
    }
    const result = await shell.openPath(dataDir);
    return { ok: result === "", error: result || undefined, dataDir };
  });

  // Mirror a content write into the workspace's LOCAL engine from this machine.
  // Prefer an explicit workspace engine link, but fall back to the bundled
  // engine when it is running. Without that fallback, downloaded users can save
  // content in BusinessOS while nothing ever reaches Optimal Engine memory.
  ipcMain.handle(
    "engine:memory",
    async (
      _,
      workspaceId: string,
      payload: {
        content: string;
        citation?: string;
        metadata?: Record<string, unknown>;
      },
    ) => {
      if (!workspaceId || !payload?.content) return { ok: false };
      await loadEngineConfigs();
      const cfg = resolveMemoryEngineConfig(workspaceId);
      if (!cfg) {
        return {
          ok: false,
          skipped: true,
          message: "No local Optimal Engine configured or running",
        };
      }
      const url = cfg.base_url.replace(/\/+$/, "") + "/api/memory/remember";
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), 8000);
      try {
        const res = await fetch(url, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(cfg.api_key ? { Authorization: `Bearer ${cfg.api_key}` } : {}),
          },
          body: JSON.stringify({
            content: payload.content,
            workspace: cfg.workspace || workspaceId,
            citation_uri: payload.citation,
            metadata: {
              ...(payload.metadata || {}),
              source_app: "businessos",
              source_workspace_id: workspaceId,
            },
            force: true,
          }),
          signal: controller.signal,
        });
        const body = await res.json().catch(() => undefined);
        return { ok: res.ok, status: res.status, body };
      } catch (e) {
        return {
          ok: false,
          message: e instanceof Error ? e.message : "write failed",
        };
      } finally {
        clearTimeout(timer);
      }
    },
  );

  // Shell operations
  ipcMain.handle("shell:open-external", async (_, url: string) => {
    console.log("[ipc] shell:open-external invoked ->", url);
    await shell.openExternal(url);
  });

  ipcMain.handle("shell:open-path", async (_, path: string) => {
    await shell.openPath(path);
  });

  // Clear the persisted auth session cookie so logout actually logs out.
  // The cookie is flushed to disk and would otherwise survive a reload and
  // silently re-authenticate the user. Remove every copy of it (any domain),
  // then flush so the change is durable.
  ipcMain.handle("auth:clear-session", async () => {
    try {
      const ses = session.defaultSession;
      const cookies = await ses.cookies.get({
        name: "better-auth.session_token",
      });
      for (const c of cookies) {
        const host = (c.domain || "").replace(/^\./, "");
        const scheme = c.secure ? "https" : "http";
        const url = `${scheme}://${host}${c.path || "/"}`;
        try {
          await ses.cookies.remove(url, c.name);
        } catch (err) {
          console.warn("[ipc] auth:clear-session remove failed", url, err);
        }
      }
      await ses.cookies.flushStore();
      return { ok: true };
    } catch (err) {
      console.error("[ipc] auth:clear-session failed", err);
      return { ok: false, error: String(err) };
    }
  });

  // Dialog operations
  ipcMain.handle(
    "dialog:show-open",
    async (_, options: Electron.OpenDialogOptions) => {
      const mainWindow = getMainWindow();
      if (!mainWindow) return { canceled: true, filePaths: [] };
      return dialog.showOpenDialog(mainWindow, options);
    },
  );

  ipcMain.handle(
    "dialog:show-save",
    async (_, options: Electron.SaveDialogOptions) => {
      const mainWindow = getMainWindow();
      if (!mainWindow) return { canceled: true, filePath: undefined };
      return dialog.showSaveDialog(mainWindow, options);
    },
  );

  ipcMain.handle(
    "dialog:show-message",
    async (_, options: Electron.MessageBoxOptions) => {
      const mainWindow = getMainWindow();
      if (!mainWindow) return { response: 0 };
      return dialog.showMessageBox(mainWindow, options);
    },
  );

  // Window state persistence
  ipcMain.handle("window:get-state", () => {
    const mainWindow = getMainWindow();
    if (!mainWindow) return null;

    const bounds = mainWindow.getBounds();
    return {
      x: bounds.x,
      y: bounds.y,
      width: bounds.width,
      height: bounds.height,
      isMaximized: mainWindow.isMaximized(),
      isFullScreen: mainWindow.isFullScreen(),
    };
  });

  ipcMain.on(
    "window:set-state",
    (_, state: { width: number; height: number; x?: number; y?: number }) => {
      const mainWindow = getMainWindow();
      if (!mainWindow) return;

      if (state.x !== undefined && state.y !== undefined) {
        mainWindow.setBounds({
          x: state.x,
          y: state.y,
          width: state.width,
          height: state.height,
        });
      } else {
        mainWindow.setSize(state.width, state.height);
      }
    },
  );

  // Sync status placeholder. Manual sync is owned by setupDatabaseHandlers().
  ipcMain.handle("sync:get-status", () => {
    return {
      status: "synced",
      lastSync: new Date().toISOString(),
      pendingChanges: 0,
    };
  });

  // Update operations (to be implemented with auto-updater)
  ipcMain.handle("updates:get-info", async () => {
    return getUpdateRuntimeInfo();
  });

  ipcMain.handle("updates:check", async () => {
    return checkForUpdates();
  });

  ipcMain.handle("updates:download", async () => {
    try {
      await downloadUpdate();
      return true;
    } catch {
      return false;
    }
  });

  ipcMain.handle("updates:install", async () => {
    try {
      installUpdate();
      return true;
    } catch {
      return false;
    }
  });

  // Screenshot capture
  ipcMain.handle("screenshot:capture", async () => {
    try {
      // Get all displays
      const displays = screen.getAllDisplays();
      const primaryDisplay = screen.getPrimaryDisplay();

      // Get desktop capturer sources
      const sources = await desktopCapturer.getSources({
        types: ["screen"],
        thumbnailSize: {
          width: primaryDisplay.size.width,
          height: primaryDisplay.size.height,
        },
      });

      if (sources.length === 0) {
        return { success: false, error: "No screen sources available" };
      }

      // Get the primary screen source
      const primarySource =
        sources.find((s) => s.display_id === primaryDisplay.id.toString()) ||
        sources[0];

      // Get the thumbnail as a data URL
      const thumbnail = primarySource.thumbnail;
      const dataUrl = thumbnail.toDataURL();

      return {
        success: true,
        dataUrl,
        size: {
          width: thumbnail.getSize().width,
          height: thumbnail.getSize().height,
        },
      };
    } catch (error) {
      console.error("Screenshot capture failed:", error);
      return {
        success: false,
        error:
          error instanceof Error ? error.message : "Screenshot capture failed",
      };
    }
  });

  // Detect which agent CLIs are on PATH, so the terminal can show an install
  // hint instead of silently failing with "command not found" on a downloaded
  // user's machine. Non-blocking: each probe has a short timeout and they run
  // in parallel. PATH is augmented with common install dirs because a
  // GUI-launched app does not inherit the login shell's full PATH.
  ipcMain.handle("agents:detect", async () => {
    const bins = ["claude", "codex", "osa", "ollama"] as const;
    const isWin = process.platform === "win32";
    const extraDirs = isWin
      ? []
      : [
          "/usr/local/bin",
          "/opt/homebrew/bin",
          "/usr/bin",
          "/bin",
          path.join(os.homedir(), ".local", "bin"),
          path.join(os.homedir(), ".npm-global", "bin"),
          path.join(os.homedir(), ".bun", "bin"),
          path.join(os.homedir(), ".deno", "bin"),
        ];
    const mergedPath = [process.env.PATH || "", ...extraDirs]
      .filter(Boolean)
      .join(path.delimiter);
    const env = { ...process.env, PATH: mergedPath };

    const probe = (bin: string): Promise<boolean> =>
      new Promise((resolve) => {
        const cmd = isWin ? `where ${bin}` : `command -v ${bin}`;
        const child = exec(cmd, { timeout: 1500, env }, (err, stdout) => {
          resolve(!err && String(stdout).trim().length > 0);
        });
        child.on("error", () => resolve(false));
      });

    const results = await Promise.all(bins.map((b) => probe(b)));
    const detected: Record<string, boolean> = {};
    bins.forEach((b, i) => {
      detected[b] = results[i];
    });
    return detected;
  });

  console.log("IPC handlers registered");
}
