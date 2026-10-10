import { BrowserWindow, app, shell } from "electron";
import path from "path";

// Global reference to main window
let mainWindow: BrowserWindow | null = null;
let embeddedWindowHandlersInstalled = false;
let isQuitting = false;

app.on("before-quit", () => {
  isQuitting = true;
});

// Window state defaults
const DEFAULT_WIDTH = 1400;
const DEFAULT_HEIGHT = 900;
const MIN_WIDTH = 800;
const MIN_HEIGHT = 600;

function getAppUrlPrefix(isDev: boolean): string {
  return isDev
    ? process.env.BUSINESSOS_DEV_URL || "http://localhost:5273"
    : "app://localhost/";
}

function isBusinessOSUrl(url: string, isDev: boolean): boolean {
  const appUrl = getAppUrlPrefix(isDev);
  return (
    url.startsWith(appUrl) ||
    url.startsWith("app://localhost/") ||
    url.startsWith("file://")
  );
}

function isExternalWebUrl(url: string): boolean {
  return url.startsWith("http://") || url.startsWith("https://");
}

function installEmbeddedWindowHandlers(): void {
  if (embeddedWindowHandlersInstalled) return;
  embeddedWindowHandlersInstalled = true;

  app.on("web-contents-created", (_event, contents) => {
    if (contents.getType() !== "webview") return;

    contents.setWindowOpenHandler(({ url }) => {
      if (url.startsWith("http://") || url.startsWith("https://")) {
        contents.loadURL(url).catch((err) => {
          console.error("[window] failed to load embedded popup URL", url, err);
        });
      }

      return { action: "deny" };
    });
  });
}

/**
 * Get the main window instance
 */
export function getMainWindow(): BrowserWindow | null {
  return mainWindow;
}

/**
 * Create the main application window
 */
export async function createMainWindow(): Promise<BrowserWindow> {
  const isDev = !app.isPackaged;
  installEmbeddedWindowHandlers();

  mainWindow = new BrowserWindow({
    width: DEFAULT_WIDTH,
    height: DEFAULT_HEIGHT,
    minWidth: MIN_WIDTH,
    minHeight: MIN_HEIGHT,
    title: "BusinessOS",
    // Linux windows carry no icon unless set explicitly (macOS gets it from
    // the bundle, Windows from the exe) — without this the dock/alt-tab
    // show a generic placeholder.
    ...(process.platform === "linux"
      ? { icon: path.join(process.resourcesPath, "icons", "icon.png") }
      : {}),
    show: false, // Don't show until ready
    backgroundColor: "#0a0a0a",
    titleBarStyle: process.platform === "darwin" ? "hiddenInset" : "default",
    // Align the macOS controls with the native title text. The previous y=18
    // placed their top edge near the titlebar's vertical center, making the
    // controls look detached from the title and reducing the usable drag area.
    trafficLightPosition: { x: 20, y: 5 },
    webPreferences: {
      preload: path.join(__dirname, "../preload/index.js"),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false,
      webSecurity: true,
      webviewTag: true,
      allowRunningInsecureContent: false,
    },
  });

  // Load the app
  if (isDev) {
    // In development, load from the BusinessOS local dev server.
    const devUrl = process.env.BUSINESSOS_DEV_URL || "http://localhost:5273";
    console.log(`Loading from ${devUrl}`);
    try {
      await mainWindow.loadURL(devUrl);
    } catch (err) {
      console.error("Failed to load URL, retrying...", err);
      // Retry once after 2 seconds
      await new Promise((r) => setTimeout(r, 2000));
      await mainWindow.loadURL(devUrl);
    }
  } else {
    // In production, use the custom app:// protocol to serve files
    // This allows SvelteKit's router to work correctly
    console.log("Loading production app via app:// protocol");
    await mainWindow.loadURL("app://localhost/");
  }

  // Show window when ready
  mainWindow.once("ready-to-show", () => {
    mainWindow?.show();
    // DevTools can be opened manually via View menu (Cmd+Option+I)
  });

  // Handle external links
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (isBusinessOSUrl(url, isDev)) {
      mainWindow?.loadURL(url).catch((err) => {
        console.error("[window] failed to load internal popup URL", url, err);
      });
    } else if (isExternalWebUrl(url)) {
      shell.openExternal(url).catch((err) => {
        console.error("[window] failed to open external popup URL", url, err);
      });
    }

    return { action: "deny" };
  });

  // Prevent navigation away from the app
  mainWindow.webContents.on("will-navigate", (event, url) => {
    if (!isBusinessOSUrl(url, isDev)) {
      event.preventDefault();
      if (isExternalWebUrl(url)) {
        shell.openExternal(url).catch((err) => {
          console.error(
            "[window] failed to open external navigation URL",
            url,
            err,
          );
        });
      }
      console.warn(
        "[window] opened external navigation outside BusinessOS",
        url,
      );
    }
  });

  // Handle window close
  mainWindow.on("close", (event) => {
    // On macOS, hide the window instead of quitting
    if (process.platform === "darwin" && !isQuitting) {
      event.preventDefault();
      mainWindow?.hide();
    }
  });

  // Clean up reference when window is closed
  mainWindow.on("closed", () => {
    mainWindow = null;
  });

  // Remember window state
  mainWindow.on("resize", saveWindowState);
  mainWindow.on("move", saveWindowState);

  // Restore window state
  restoreWindowState();

  return mainWindow;
}

/**
 * Save window state to localStorage (via IPC or electron-store)
 */
function saveWindowState(): void {
  if (!mainWindow) return;

  const bounds = mainWindow.getBounds();
  const state = {
    x: bounds.x,
    y: bounds.y,
    width: bounds.width,
    height: bounds.height,
    isMaximized: mainWindow.isMaximized(),
    isFullScreen: mainWindow.isFullScreen(),
  };

  // Store state (could use electron-store for persistence)
  // For now, we'll send it to the renderer to store in localStorage
  mainWindow.webContents.send("window:save-state", state);
}

/**
 * Restore window state from storage
 */
function restoreWindowState(): void {
  // Request stored state from renderer
  // The renderer will respond via IPC if it has stored state
}

/**
 * Focus or create the main window
 */
export function focusMainWindow(): void {
  if (mainWindow) {
    if (mainWindow.isMinimized()) {
      mainWindow.restore();
    }
    mainWindow.focus();
  } else {
    createMainWindow();
  }
}

/**
 * Send a message to the main window
 */
export function sendToMainWindow(channel: string, ...args: unknown[]): void {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send(channel, ...args);
  }
}
