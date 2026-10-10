import { app, ipcMain, dialog } from "electron";
import { sendToMainWindow } from "../window";

// Lazy-loaded auto-updater to avoid initialization errors
let autoUpdater: any = null;
// Guards so the native "update ready" dialog is shown at most once per download,
// and an interactive (menu-triggered) check can suppress the auto banners.
let updateDownloadedPromptShown = false;
let interactiveCheckInFlight = false;

const UPDATE_OWNER = process.env.BUSINESSOS_UPDATE_OWNER || "Miosa-osa";
const UPDATE_REPO = process.env.BUSINESSOS_UPDATE_REPO || "businessos-5";
const UPDATE_CHANNEL = process.env.BUSINESSOS_UPDATE_CHANNEL || "stable";
const MIN_SUPPORTED_VERSION =
  process.env.BUSINESSOS_MIN_SUPPORTED_VERSION || "";

export interface UpdateCheckResponse {
  ok: boolean;
  available: boolean;
  version: string | null;
  currentVersion: string;
  releaseDate?: string;
  releaseNotes?: unknown;
  error?: string;
  message?: string;
}

export interface UpdateRuntimeInfo {
  currentVersion: string;
  channel: string;
  isPackaged: boolean;
  owner: string;
  repo: string;
  minimumSupportedVersion: string | null;
  supported: boolean;
}

function versionParts(version: string): number[] {
  return version
    .split(/[.-]/)
    .slice(0, 3)
    .map((part) => Number.parseInt(part, 10))
    .map((part) => (Number.isFinite(part) ? part : 0));
}

function isVersionAtLeast(version: string, minimum: string): boolean {
  if (!minimum) return true;
  const current = versionParts(version);
  const required = versionParts(minimum);
  for (let i = 0; i < Math.max(current.length, required.length); i += 1) {
    const a = current[i] ?? 0;
    const b = required[i] ?? 0;
    if (a > b) return true;
    if (a < b) return false;
  }
  return true;
}

function getAutoUpdater() {
  if (!autoUpdater) {
    // Only import when needed (after app is ready)
    const { autoUpdater: au } = require("electron-updater");
    autoUpdater = au;
    // Auto-download in the background so an update is ready without any user
    // action. The native "restart to install" dialog (update-downloaded) then
    // works even when the user is stuck at the login screen / not signed in,
    // because it is a main-process dialog, not a renderer banner.
    autoUpdater.autoDownload = true;
    autoUpdater.autoInstallOnAppQuit = true;
    autoUpdater.setFeedURL({
      provider: "github",
      owner: UPDATE_OWNER,
      repo: UPDATE_REPO,
      private: process.env.BUSINESSOS_UPDATES_PRIVATE === "true",
    });
    autoUpdater.channel = UPDATE_CHANNEL;
  }
  return autoUpdater;
}

/**
 * Set up the auto-updater with event handlers
 */
export function setupAutoUpdater(): void {
  const updater = getAutoUpdater();

  // Register the version IPC handler so the renderer can display the current
  // version. The OptimalEngine ships bundled inside the app, so it is treated
  // as one unified version (app version === engine version).
  registerVersionHandler();

  // Check for updates on startup (after a delay)
  setTimeout(() => {
    checkForUpdates();
  }, 10000);

  // Check for updates periodically (every 4 hours)
  setInterval(
    () => {
      checkForUpdates();
    },
    4 * 60 * 60 * 1000,
  );

  // Event: Error occurred
  updater.on("error", (error: Error) => {
    console.error("Auto-update error:", error);
    sendToMainWindow("update:error", error.message);
  });

  // Event: Checking for updates
  updater.on("checking-for-update", () => {
    console.log("Checking for updates...");
    sendToMainWindow("update:checking");
  });

  // Event: Update available
  updater.on("update-available", (info: any) => {
    console.log("Update available:", info.version);
    sendToMainWindow("update:available", {
      version: info.version,
      currentVersion: app.getVersion(),
      releaseDate: info.releaseDate,
      releaseNotes: info.releaseNotes,
    });
  });

  // Event: Update not available
  updater.on("update-not-available", () => {
    console.log("No updates available");
    sendToMainWindow("update:not-available");
  });

  // Event: Download progress
  updater.on("download-progress", (progress: any) => {
    console.log(`Download progress: ${progress.percent.toFixed(1)}%`);
    sendToMainWindow("update:download-progress", {
      percent: progress.percent,
      bytesPerSecond: progress.bytesPerSecond,
      transferred: progress.transferred,
      total: progress.total,
    });
  });

  // Event: Update downloaded
  updater.on("update-downloaded", (info: any) => {
    console.log("Update downloaded:", info.version);
    sendToMainWindow("update:downloaded", {
      version: info.version,
    });
    // Native dialog so the prompt appears regardless of login state (a signed-out
    // user stuck at the login screen still gets it — it is not a renderer banner).
    if (updateDownloadedPromptShown) return;
    updateDownloadedPromptShown = true;
    dialog
      .showMessageBox({
        type: "info",
        buttons: ["Restart & Update", "Later"],
        defaultId: 0,
        cancelId: 1,
        title: "Update Ready",
        message: `BusinessOS ${info.version} is ready to install.`,
        detail:
          "The update has been downloaded. Restart now to install it, or it will install automatically the next time you quit BusinessOS.",
      })
      .then(({ response }) => {
        if (response === 0) installUpdate();
      })
      .catch((err) => console.error("update-downloaded dialog failed", err));
  });

  console.log("Auto-updater initialized");
}

/**
 * Check for updates
 */
export async function checkForUpdates(): Promise<UpdateCheckResponse> {
  const currentVersion = app.getVersion();
  if (!app.isPackaged && process.env.BUSINESSOS_FORCE_UPDATE_CHECK !== "true") {
    return {
      ok: true,
      available: false,
      version: null,
      currentVersion,
      message: "Updates are only checked from a packaged app.",
    };
  }

  try {
    const updater = getAutoUpdater();
    const result = await updater.checkForUpdates();
    if (!result) {
      return {
        ok: true,
        available: false,
        version: null,
        currentVersion,
        message: "Updater is not active.",
      };
    }
    const info = result.updateInfo ?? result.versionInfo ?? {};
    return {
      ok: true,
      available: Boolean(result.isUpdateAvailable),
      version: info.version ?? null,
      currentVersion,
      releaseDate: info.releaseDate,
      releaseNotes: info.releaseNotes,
    };
  } catch (error) {
    console.error("Failed to check for updates:", error);
    return {
      ok: false,
      available: false,
      version: null,
      currentVersion,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

/**
 * Menu-triggered "Check for Updates…". Shows native dialogs for every outcome
 * so it works from the app menu even when the user is not signed in. When an
 * update exists, autoDownload pulls it and the update-downloaded dialog offers
 * the restart; here we just confirm the check result.
 */
export async function checkForUpdatesInteractive(): Promise<void> {
  if (interactiveCheckInFlight) return;
  interactiveCheckInFlight = true;
  try {
    const result = await checkForUpdates();
    if (!result.ok) {
      await dialog.showMessageBox({
        type: "error",
        buttons: ["OK"],
        title: "Check for Updates",
        message: "Could not check for updates.",
        detail: result.error || "Please try again later.",
      });
      return;
    }
    if (result.available) {
      await dialog.showMessageBox({
        type: "info",
        buttons: ["OK"],
        title: "Update Available",
        message: `BusinessOS ${result.version} is available.`,
        detail:
          "It is downloading now. You will be prompted to restart and install when it is ready.",
      });
      return;
    }
    await dialog.showMessageBox({
      type: "info",
      buttons: ["OK"],
      title: "Check for Updates",
      message: "You are up to date.",
      detail: `BusinessOS ${result.currentVersion} is the latest version.`,
    });
  } finally {
    interactiveCheckInFlight = false;
  }
}

export function getUpdateRuntimeInfo(): UpdateRuntimeInfo {
  const currentVersion = app.getVersion();
  return {
    currentVersion,
    channel: UPDATE_CHANNEL,
    isPackaged: app.isPackaged,
    owner: UPDATE_OWNER,
    repo: UPDATE_REPO,
    minimumSupportedVersion: MIN_SUPPORTED_VERSION || null,
    supported: isVersionAtLeast(currentVersion, MIN_SUPPORTED_VERSION),
  };
}

/**
 * Download available update
 */
export async function downloadUpdate(): Promise<void> {
  try {
    const updater = getAutoUpdater();
    await updater.downloadUpdate();
  } catch (error) {
    console.error("Failed to download update:", error);
    sendToMainWindow("update:error", "Failed to download update");
  }
}

/**
 * Install downloaded update (quit and install)
 */
export function installUpdate(): void {
  const updater = getAutoUpdater();
  updater.quitAndInstall(false, true);
}

/**
 * Unified version for the app + bundled OptimalEngine.
 * The engine ships inside the app, so both share the app version.
 */
export interface AppVersionInfo {
  app: string;
  engine: string;
}

/**
 * Register the "updates:get-version" IPC handler.
 * Returns the current unified version. Idempotent (safe to call once).
 */
function registerVersionHandler(): void {
  ipcMain.removeHandler("updates:get-version");
  ipcMain.handle("updates:get-version", (): AppVersionInfo => {
    const version = app.getVersion();
    // The OptimalEngine is bundled with the app and versioned together.
    return { app: version, engine: version };
  });
}
