import "./utils/install-safe-console";
import "./utils/dev-user-data";
import {
  app,
  BrowserWindow,
  ipcMain,
  Menu,
  Tray,
  nativeImage,
  protocol,
  net,
  session,
  systemPreferences,
} from "electron";
import path from "path";
import fs from "fs";
import { createMainWindow, getMainWindow } from "./window";
import {
  setupIpcHandlers,
  initializeDatabaseSystem,
  startSync,
  stopSync,
  setEngineStatusProvider,
  resolveBusinessOSHomeDir,
} from "./ipc";
import { scaffoldBusinessOSHome } from "./scaffold";
import { BackendManager } from "./backend/manager";
import { mediaPermission, originOf } from "./permissions/media";
import { EngineManager } from "./engine/manager";
import {
  setupAutoUpdater,
  checkForUpdatesInteractive,
} from "./updater/auto-update";
import { initializePopupSystem, cleanupPopupSystem } from "./popup/chat-popup";
import { initializeMeetingRecorder } from "./audio/meeting-recorder";
import { closeDatabase } from "./database/sqlite";
import { killAllTerminals } from "./terminal/pty-manager";
import { pathToFileURL } from "url";

// Handle Squirrel events for Windows installer (only on Windows)
if (process.platform === "win32") {
  try {
    if (require("electron-squirrel-startup")) {
      app.quit();
    }
  } catch {
    // electron-squirrel-startup not available, ignore on non-Windows
  }
}

// Single instance lock
const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
  app.quit();
} else {
  app.on("second-instance", (_event, argv) => {
    // On Windows/Linux the deep link arrives as a launch argument here.
    const deepLink = argv.find((a) => a.startsWith("businessos://"));
    if (deepLink) {
      void handleDeepLink(deepLink);
      return;
    }
    // Otherwise just focus our window.
    const mainWindow = getMainWindow();
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
  });
}

// Global references
let backendManager: BackendManager | null = null;
let engineManager: EngineManager | null = null;

// Cloud backend the desktop app authenticates against. Must be a *.businessos.dev
// host so the session cookie (Domain=.businessos.dev) attaches. Overridable for
// testing via BUSINESSOS_CLOUD_URL.
const CLOUD_URL =
  process.env.BUSINESSOS_CLOUD_URL || "https://app.businessos.dev";

// Deep links that arrive before the main window exists (cold start) are buffered
// here and replayed once the window is ready.
let pendingDeepLink: string | null = null;

// Install the session token handed back from the system-browser Google OAuth via
// the businessos:// deep link. Cookies set in the external browser never reach
// the Electron session, so the backend returns the token in the deep-link URL and
// we write it into the app's own cookie jar here.
async function installSessionCookie(token: string): Promise<void> {
  const expiry = Math.floor(Date.now() / 1000) + 60 * 60 * 24 * 7; // 7 days

  if (!app.isPackaged) {
    // Dev: the renderer (localhost:5273) and backend (localhost:8001) are the
    // SAME SITE (both "localhost"), so a host-only SameSite=Lax cookie over http
    // is sent on the renderer's cross-port fetches to the backend. The prod
    // .businessos.dev / Secure / SameSite=None cookie below would never reach
    // localhost, which left desktop Google sign-in stuck "loading".
    const port = process.env.BUSINESSOS_BACKEND_PORT || "8001";
    await session.defaultSession.cookies.set({
      url: `http://localhost:${port}`,
      name: "better-auth.session_token",
      value: token,
      path: "/",
      secure: false,
      httpOnly: true,
      sameSite: "lax",
      expirationDate: expiry,
    });
  } else {
    await session.defaultSession.cookies.set({
      url: CLOUD_URL,
      name: "better-auth.session_token",
      value: token,
      domain: ".businessos.dev",
      path: "/",
      secure: true,
      httpOnly: true,
      // app://localhost -> https://app.businessos.dev is cross-site, so the cookie
      // must be SameSite=None (no_restriction) to be sent on the renderer's fetches.
      sameSite: "no_restriction",
      expirationDate: expiry,
    });
  }
  await session.defaultSession.cookies.flushStore();
}

// Route an incoming businessos:// deep link. The OAuth callback carries a session
// token; everything else is treated as an in-app navigation.
async function handleDeepLink(rawUrl: string): Promise<void> {
  try {
    const parsed = new URL(rawUrl);
    const token = parsed.searchParams.get("token");
    const isAuthCallback =
      parsed.hostname === "auth" || parsed.pathname.includes("auth");

    console.log(
      "[auth] deep link received:",
      JSON.stringify({
        host: parsed.hostname,
        path: parsed.pathname,
        hasToken: !!token,
        tokenLen: token ? token.length : 0,
        isAuthCallback,
      }),
    );

    if (isAuthCallback && token) {
      await installSessionCookie(token);
      // Read the cookie back so we can prove it actually landed in the jar.
      try {
        const port = process.env.BUSINESSOS_BACKEND_PORT || "8001";
        const back = await session.defaultSession.cookies.get({
          name: "better-auth.session_token",
        });
        console.log(
          "[auth] cookie jar after install:",
          JSON.stringify(
            back.map((c) => ({
              domain: c.domain,
              path: c.path,
              secure: c.secure,
              sameSite: c.sameSite,
              valueLen: c.value.length,
            })),
          ),
          "expected backend http://localhost:" + port,
        );
      } catch (e) {
        console.error("[auth] cookie read-back failed", e);
      }
      const win = getMainWindow();
      console.log("[auth] reloading renderer, hasWindow:", !!win);
      if (win) {
        win.show();
        win.focus();
        // Clear the logout flag in the renderer's localStorage so the reload
        // lands authenticated instead of being forced back to login by a stale
        // "logged out" marker. Done here (main) so it's deterministic.
        try {
          await win.webContents.executeJavaScript(
            "try{localStorage.removeItem('businessos_logged_out')}catch(e){}",
          );
        } catch (e) {
          console.warn("[auth] could not clear logout flag pre-reload", e);
        }
        const dashboardUrl = app.isPackaged
          ? "app://localhost/dashboard"
          : `http://localhost:${process.env.BUSINESSOS_DEV_PORT || "5273"}/dashboard`;
        await win.loadURL(dashboardUrl);
      }
      return;
    }
    // OAuth integration callback (Google Calendar/Gmail/etc). The consent
    // happens in the external browser, which cannot reach app://localhost, so
    // the backend bounces back through businessos://oauth?connected=<tool>.
    // Load the connectors page (preserving the connected marker) in-app.
    const isOAuthCallback =
      parsed.hostname === "oauth" || parsed.pathname.includes("oauth");
    if (isOAuthCallback) {
      const connected = parsed.searchParams.get("connected");
      const win = getMainWindow();
      if (win) {
        win.show();
        win.focus();
        const connectorsBase = app.isPackaged
          ? "app://localhost/connectors"
          : `http://localhost:${process.env.BUSINESSOS_DEV_PORT || "5273"}/connectors`;
        const target = connected
          ? `${connectorsBase}?connected=${encodeURIComponent(connected)}`
          : connectorsBase;
        console.log("[oauth] integration connected, loading", target);
        await win.loadURL(target);
      }
      return;
    }

    console.warn(
      "[auth] deep link NOT treated as auth callback (missing token or not auth path)",
    );

    // Generic deep link: navigate the renderer to the path.
    const win = getMainWindow();
    if (win) {
      win.webContents.send("navigate", parsed.pathname);
      win.show();
      win.focus();
    }
  } catch (error) {
    console.error("Failed to handle deep link:", rawUrl, error);
  }
}

// App metadata
const isDev = !app.isPackaged;
const appPath = app.getAppPath();
const resourcesPath = isDev
  ? path.join(appPath, "resources")
  : process.resourcesPath;

// Register custom protocol for serving app files
// This allows SvelteKit to work correctly with file:// URLs
protocol.registerSchemesAsPrivileged([
  {
    scheme: "app",
    privileges: {
      standard: true,
      secure: true,
      supportFetchAPI: true,
      corsEnabled: true,
    },
  },
]);

/**
 * Create the native application menu
 */
function createAppMenu(): void {
  const isMac = process.platform === "darwin";

  const template: Electron.MenuItemConstructorOptions[] = [
    // App menu (macOS only)
    ...(isMac
      ? [
          {
            label: app.name,
            submenu: [
              { role: "about" as const },
              { type: "separator" as const },
              {
                label: "Check for Updates...",
                click: () => {
                  void checkForUpdatesInteractive();
                },
              },
              { type: "separator" as const },
              {
                label: "Preferences...",
                accelerator: "CommandOrControl+,",
                click: () => {
                  const mainWindow = getMainWindow();
                  if (mainWindow) {
                    mainWindow.webContents.send("navigate", "/profile");
                  }
                },
              },
              { type: "separator" as const },
              { role: "services" as const },
              { type: "separator" as const },
              { role: "hide" as const },
              { role: "hideOthers" as const },
              { role: "unhide" as const },
              { type: "separator" as const },
              { role: "quit" as const },
            ] as Electron.MenuItemConstructorOptions[],
          },
        ]
      : []),
    // File menu
    {
      label: "File",
      submenu: [
        {
          label: "New Task",
          accelerator: "CommandOrControl+N",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("shortcut", "new-task");
            }
          },
        },
        {
          label: "New Project",
          accelerator: "CommandOrControl+Shift+N",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("shortcut", "new-project");
            }
          },
        },
        { type: "separator" },
        isMac ? { role: "close" } : { role: "quit" },
      ] as Electron.MenuItemConstructorOptions[],
    },
    // Edit menu
    {
      label: "Edit",
      submenu: [
        { role: "undo" },
        { role: "redo" },
        { type: "separator" },
        { role: "cut" },
        { role: "copy" },
        { role: "paste" },
        ...(isMac
          ? [
              { role: "pasteAndMatchStyle" as const },
              { role: "delete" as const },
              { role: "selectAll" as const },
            ]
          : [
              { role: "delete" as const },
              { type: "separator" as const },
              { role: "selectAll" as const },
            ]),
      ] as Electron.MenuItemConstructorOptions[],
    },
    // View menu
    {
      label: "View",
      submenu: [
        { role: "reload" },
        { role: "forceReload" },
        { role: "toggleDevTools" },
        { type: "separator" },
        { role: "resetZoom" },
        { role: "zoomIn" },
        { role: "zoomOut" },
        { type: "separator" },
        { role: "togglefullscreen" },
      ] as Electron.MenuItemConstructorOptions[],
    },
    // Navigate menu
    {
      label: "Navigate",
      submenu: [
        {
          label: "Dashboard",
          accelerator: "CommandOrControl+1",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("navigate", "/dashboard");
            }
          },
        },
        {
          label: "Tasks",
          accelerator: "CommandOrControl+2",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("navigate", "/tasks");
            }
          },
        },
        {
          label: "Calendar",
          accelerator: "CommandOrControl+3",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("navigate", "/calendar");
            }
          },
        },
        {
          label: "Projects",
          accelerator: "CommandOrControl+4",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("navigate", "/projects");
            }
          },
        },
        {
          label: "Chat",
          accelerator: "CommandOrControl+5",
          click: () => {
            const mainWindow = getMainWindow();
            if (mainWindow) {
              mainWindow.webContents.send("navigate", "/chat");
            }
          },
        },
      ],
    },
    // Window menu
    {
      label: "Window",
      submenu: [
        { role: "minimize" },
        { role: "zoom" },
        ...(isMac
          ? [
              { type: "separator" as const },
              { role: "front" as const },
              { type: "separator" as const },
              { role: "window" as const },
            ]
          : [{ role: "close" as const }]),
      ] as Electron.MenuItemConstructorOptions[],
    },
    // Help menu
    {
      role: "help",
      submenu: [
        {
          label: "Check for Updates...",
          click: () => {
            void checkForUpdatesInteractive();
          },
        },
        { type: "separator" },
        {
          label: "Documentation",
          click: async () => {
            const { shell } = require("electron");
            await shell.openExternal("https://businessos.app/docs");
          },
        },
        {
          label: "Report Issue",
          click: async () => {
            const { shell } = require("electron");
            await shell.openExternal(
              "https://github.com/Miosa-osa/businessos-5/issues",
            );
          },
        },
        { type: "separator" },
        {
          label: "About BusinessOS",
          click: () => {
            const { dialog } = require("electron");
            dialog.showMessageBox({
              type: "info",
              title: "About BusinessOS",
              message: "BusinessOS Desktop",
              detail: `Version: ${app.getVersion()}\nElectron: ${process.versions.electron}\nChrome: ${process.versions.chrome}\nNode.js: ${process.versions.node}`,
            });
          },
        },
      ],
    },
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

/**
 * MIME types for media assets served over app:// with byte-range support.
 * Only these extensions take the range-aware fs read path; everything else
 * keeps the existing net.fetch + SPA fallback behavior.
 */
const MEDIA_MIME: Record<string, string> = {
  ".mp4": "video/mp4",
  ".m4v": "video/mp4",
  ".webm": "video/webm",
  ".mov": "video/quicktime",
  ".ogg": "video/ogg",
  ".ogv": "video/ogg",
  ".mp3": "audio/mpeg",
  ".m4a": "audio/mp4",
  ".wav": "audio/wav",
  ".aac": "audio/aac",
  ".oga": "audio/ogg",
  ".flac": "audio/flac",
};

/**
 * Initialize the application
 */
// Camera/microphone for the app's own pages (3D OS gestures, voice): ask macOS
// for each device the page wants, since macOS never prompts on its own here.
function installMediaPermissions(): void {
  const trustedOrigins = [
    originOf(process.env.BUSINESSOS_DEV_URL || "http://localhost:5273"),
    "app://localhost",
  ];
  session.defaultSession.setPermissionRequestHandler(
    (_webContents, permission, callback, details) => {
      const decision = mediaPermission({
        permission,
        origin: originOf(details.requestingUrl ?? ""),
        mediaTypes: ("mediaTypes" in details ? details.mediaTypes : undefined) ?? [],
        trustedOrigins,
      });
      if (decision === null) return callback(true); // Electron's default
      if (!decision.allow) return callback(false);
      if (process.platform !== "darwin") return callback(true);
      Promise.all(
        decision.ask.map((device) =>
          systemPreferences.getMediaAccessStatus(device) === "granted"
            ? true
            : systemPreferences.askForMediaAccess(device),
        ),
      )
        .then((granted) => callback(granted.every(Boolean)))
        .catch(() => callback(false));
    },
  );
}

async function initialize(): Promise<void> {
  console.log("BusinessOS Desktop starting...");
  installMediaPermissions();
  console.log(`Environment: ${isDev ? "development" : "production"}`);
  console.log(`App path: ${appPath}`);
  console.log(`Resources path: ${resourcesPath}`);

  // Initialize local SQLite database and sync engine. Guarded: better-sqlite3 is
  // a native module and can fail to load in a dev Electron (ABI mismatch). If it
  // throws here unguarded, the whole initialize() aborts BEFORE IPC handlers are
  // registered - which left shell:open-external unregistered and broke "Sign in
  // with Google" (it could not open the system browser). A local-DB failure must
  // not take down IPC + the window.
  try {
    initializeDatabaseSystem();
    console.log("Local database initialized");
  } catch (e) {
    console.error("[init] local database init failed (continuing):", e);
  }

  // Register the app:// protocol handler for serving static files
  if (!isDev) {
    const rendererRoot = path.join(__dirname, "../renderer/main_window");
    const indexHtmlUrl = pathToFileURL(
      path.join(rendererRoot, "index.html"),
    ).href;

    protocol.handle("app", async (request) => {
      const url = new URL(request.url);

      // Decode (e.g. %20) and strip the query/hash so only the path is matched.
      const requestPath = decodeURIComponent(url.pathname);

      // A request is for a real asset only when its last segment has a file
      // extension (.js, .css, .png, ...). Everything else - "/", "/admin",
      // "/settings", deep client-side routes - is an SPA navigation and must
      // be served index.html so SvelteKit can client-route to it.
      const lastSegment = requestPath.split("/").pop() ?? "";
      const hasExtension = lastSegment.includes(".");

      if (!hasExtension) {
        return net.fetch(indexHtmlUrl);
      }

      // Resolve the asset inside the renderer root. path.join + the leading
      // "/" guard below prevents "../" traversal from escaping the root.
      const safePath = path
        .normalize(requestPath)
        .replace(/^(\.\.(\/|\\|$))+/, "");
      const fullPath = path.join(rendererRoot, safePath);

      // Media files (video/audio) must be served with byte-range support.
      // net.fetch(file://) does NOT satisfy the browser's <video> Range
      // requests (Range: bytes=...), which surfaces as ERR_UNEXPECTED and a
      // dead <video>. Read the file directly and honor the Range header:
      // return 206 Partial Content with Content-Range when ranged, else 200.
      const ext = path.extname(lastSegment).toLowerCase();
      if (MEDIA_MIME[ext]) {
        try {
          const stat = fs.statSync(fullPath);
          if (stat.isFile()) {
            const total = stat.size;
            const contentType = MEDIA_MIME[ext];
            const rangeHeader = request.headers.get("range");
            const rangeMatch = rangeHeader
              ? /bytes=(\d*)-(\d*)/.exec(rangeHeader)
              : null;

            if (rangeMatch) {
              let start = rangeMatch[1] ? parseInt(rangeMatch[1], 10) : 0;
              let end = rangeMatch[2] ? parseInt(rangeMatch[2], 10) : total - 1;
              if (Number.isNaN(start)) start = 0;
              if (Number.isNaN(end) || end >= total) end = total - 1;

              // Unsatisfiable range → 416 with the resource size.
              if (start > end || start >= total) {
                return new Response(null, {
                  status: 416,
                  headers: { "Content-Range": `bytes */${total}` },
                });
              }

              const length = end - start + 1;
              const buffer = Buffer.alloc(length);
              const fd = fs.openSync(fullPath, "r");
              try {
                fs.readSync(fd, buffer, 0, length, start);
              } finally {
                fs.closeSync(fd);
              }
              return new Response(buffer, {
                status: 206,
                headers: {
                  "Content-Type": contentType,
                  "Content-Length": String(length),
                  "Accept-Ranges": "bytes",
                  "Content-Range": `bytes ${start}-${end}/${total}`,
                },
              });
            }

            // No Range header → full 200 response, still advertising range
            // support so the browser knows it can request byte ranges.
            const buffer = fs.readFileSync(fullPath);
            return new Response(buffer, {
              status: 200,
              headers: {
                "Content-Type": contentType,
                "Content-Length": String(total),
                "Accept-Ranges": "bytes",
              },
            });
          }
        } catch {
          // Fall through to the standard net.fetch + SPA fallback below.
        }
      }

      // If the file genuinely does not exist, fall back to the SPA shell
      // rather than surfacing Chromium's 404 page.
      const response = await net.fetch(pathToFileURL(fullPath).href);
      if (response.ok || response.status === 304) {
        return response;
      }
      return net.fetch(indexHtmlUrl);
    });
    console.log("Registered app:// protocol handler");
  }

  // Construct the backend sidecar object, but do NOT start it yet.
  backendManager = new BackendManager(resourcesPath);

  // Construct the bundled OptimalEngine manager (started below, after the
  // window). Every BusinessOS ships with a built-in engine.
  engineManager = new EngineManager(resourcesPath);
  // Let engine:status report the live manager (running/url/port) to the UI.
  setEngineStatusProvider(engineManager);

  // Register IPC handlers FIRST. shell:open-external (used by "Sign in with
  // Google" to open the system browser) and every other handler must be
  // registered regardless of whether the backend sidecar starts. Previously
  // these ran AFTER `await backendManager.start()`, so a slow/failed dev backend
  // left shell:open-external unregistered and the browser never opened.
  setupIpcHandlers(backendManager);

  // Create the application menu
  createAppMenu();

  // Create the main window (so the UI is up even if the backend is slow).
  await createMainWindow();

  // Start the bundled OptimalEngine BEFORE the Go backend, so the backend can be
  // pointed at it via OPTIMAL_ENGINE_URL. Non-fatal: if it can't start, the app
  // continues (cloud mode / external engine). Only meaningful when packaged (in
  // dev the developer runs their own engine).
  if (app.isPackaged) {
    try {
      const engineOk = await engineManager.start();
      if (engineOk && engineManager.getUrl()) {
        process.env.OPTIMAL_ENGINE_URL = engineManager.getUrl();
        console.log(`Bundled engine ready at ${engineManager.getUrl()}`);
      } else {
        console.warn("Bundled engine not available (app continues without it)");
      }
    } catch (error) {
      console.error("Bundled engine failed to start (non-fatal):", error);
    }
  }

  // Seed the user's BusinessOS home folder with agent-onboarding docs + a live
  // status file, so when they open the terminal there any AI agent (Claude,
  // Codex, ...) immediately understands BusinessOS, its engine, and the cloud.
  // Non-fatal.
  try {
    const home = await resolveBusinessOSHomeDir();
    scaffoldBusinessOSHome(home, {
      engineUrl:
        engineManager?.getUrl() || process.env.OPTIMAL_ENGINE_URL || "",
      cloudUrl: CLOUD_URL,
      version: app.getVersion(),
      dataDir: path.join(app.getPath("userData"), "optimal-engine"),
    });
  } catch (error) {
    console.warn("BusinessOS home scaffold skipped:", error);
  }

  // Start the local Go backend sidecar LAST. It is OPTIONAL: the app defaults to
  // cloud mode (app.businessos.dev), so the UI works fully even if the sidecar
  // never starts. The sidecar only powers local-first / offline mode. A failure
  // here must NEVER crash the app - previously a throw in packaged mode took the
  // whole app down on any machine where the sidecar couldn't bind or run,
  // which is what broke fresh DMG installs.
  try {
    await backendManager.start();
    console.log("Local backend sidecar started");
  } catch (error) {
    console.error(
      "Local backend sidecar failed (app continues in cloud mode):",
      error,
    );
  }

  // Initialize popup chat system (includes tray and global shortcuts)
  initializePopupSystem();

  // Initialize meeting recorder
  initializeMeetingRecorder();
  console.log("Meeting recorder initialized");

  // Set up auto-updater (production only)
  if (!isDev) {
    setupAutoUpdater();
  }

  // Start sync engine
  startSync();
  console.log("Sync engine started");

  // Replay a deep link that arrived during cold start (e.g. OAuth callback that
  // launched the app), plus any link passed as a launch argument on Win/Linux.
  const argvLink = process.argv.find((a) => a.startsWith("businessos://"));
  const coldLink = pendingDeepLink || argvLink;
  if (coldLink) {
    pendingDeepLink = null;
    void handleDeepLink(coldLink);
  }
}

// App lifecycle events
app.whenReady().then(initialize).catch(console.error);

// Periodically flush cookies to disk so the session survives even an abrupt
// termination (so "Remember me" actually keeps you signed in across restarts).
app.whenReady().then(() => {
  setInterval(() => {
    session.defaultSession.cookies.flushStore().catch(() => {});
  }, 30_000);
});

app.on("window-all-closed", () => {
  // On macOS, keep the app running in the background (tray)
  if (process.platform !== "darwin") {
    app.quit();
  }
});

app.on("activate", () => {
  // On macOS, re-create window when dock icon is clicked
  if (BrowserWindow.getAllWindows().length === 0) {
    createMainWindow();
  } else {
    const mainWindow = getMainWindow();
    if (mainWindow) {
      mainWindow.show();
      mainWindow.focus();
    }
  }
});

app.on("before-quit", async () => {
  console.log("BusinessOS Desktop shutting down...");

  // Persist cookies (incl. session token) to disk so login survives restarts.
  try {
    await session.defaultSession.cookies.flushStore();
  } catch (e) {
    console.error("cookie flush failed", e);
  }

  // Stop sync engine
  stopSync();

  // Kill any local terminal PTYs
  killAllTerminals();

  // Cleanup popup system (shortcuts, tray)
  cleanupPopupSystem();

  // Close SQLite database
  closeDatabase();

  // Stop the Go backend
  if (backendManager) {
    await backendManager.stop();
  }

  // Stop the bundled OptimalEngine
  if (engineManager) {
    await engineManager.stop();
  }
});

// Handle deep links (businessos://) — macOS delivers them here.
app.on("open-url", (event, url) => {
  event.preventDefault();
  console.log("Deep link received:", url);

  // If the window isn't up yet (cold start launched by the link), buffer it and
  // replay once initialize() has created the window.
  if (!getMainWindow()) {
    pendingDeepLink = url;
    return;
  }
  void handleDeepLink(url);
});

// Register deep link protocol
if (process.defaultApp) {
  if (process.argv.length >= 2) {
    app.setAsDefaultProtocolClient("businessos", process.execPath, [
      path.resolve(process.argv[1]),
    ]);
  }
} else {
  app.setAsDefaultProtocolClient("businessos");
}
