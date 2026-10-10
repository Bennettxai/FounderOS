import {
  BrowserWindow,
  globalShortcut,
  screen,
  app,
  ipcMain,
  Tray,
  Menu,
  nativeImage,
  systemPreferences,
  dialog,
} from "electron";
import path from "path";
import { getMainWindow } from "../window";
import Store from "electron-store";

// Popup window instance
let popupWindow: BrowserWindow | null = null;
let tray: Tray | null = null;

// Settings store for shortcuts
const store = new Store({
  name: "shortcuts",
  defaults: {
    shortcuts: {
      quickChat: "CommandOrControl+Shift+Space",
      spotlight: "CommandOrControl+Space",
      voiceInput: "CommandOrControl+D",
    },
    accessibilityPrompted: false,
  },
});

// Popup window settings
const POPUP_SIZES = {
  small: { width: 420, height: 500 },
  medium: { width: 520, height: 700 },
  large: { width: 800, height: 900 },
  full: { width: 1200, height: 850 },
};

type PopupSize = keyof typeof POPUP_SIZES;
let currentSize: PopupSize = "small";

// Get shortcuts from store
function getShortcuts() {
  return store.get("shortcuts") as {
    quickChat: string;
    spotlight: string;
    voiceInput: string;
  };
}

/**
 * Create the popup chat window (hidden by default)
 */
export function createPopupWindow(): BrowserWindow {
  const isDev = !app.isPackaged;

  // Get the display where the cursor is
  const cursorPoint = screen.getCursorScreenPoint();
  const display = screen.getDisplayNearestPoint(cursorPoint);

  const size = POPUP_SIZES[currentSize];
  // Position in center-top of the current display
  const x = Math.round(
    display.bounds.x + (display.bounds.width - size.width) / 2,
  );
  const y = display.bounds.y + 80; // 80px from top

  popupWindow = new BrowserWindow({
    width: size.width,
    height: size.height,
    minWidth: 380,
    minHeight: 400,
    x,
    y,
    show: false,
    frame: false,
    transparent: true,
    resizable: true, // Allow manual resize
    movable: true,
    minimizable: false,
    maximizable: false,
    closable: true,
    alwaysOnTop: true,
    skipTaskbar: true,
    hasShadow: true,
    vibrancy: process.platform === "darwin" ? "popover" : undefined,
    visualEffectState: "active",
    roundedCorners: true,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false,
      preload: path.join(__dirname, "../preload/index.js"),
    },
  });

  // Load the popup chat page
  if (isDev) {
    const devUrl = process.env.BUSINESSOS_DEV_URL || "http://localhost:5273";
    popupWindow.loadURL(`${devUrl}/popup-chat`);
  } else {
    const indexPath = path.join(__dirname, "../renderer/popup-chat.html");
    popupWindow.loadFile(indexPath);
  }

  // Hide instead of close
  popupWindow.on("close", (event) => {
    event.preventDefault();
    popupWindow?.hide();
  });

  // Hide on blur (click outside)
  popupWindow.on("blur", () => {
    // Small delay to allow for click events
    setTimeout(() => {
      if (popupWindow && !popupWindow.isFocused()) {
        popupWindow.hide();
      }
    }, 100);
  });

  popupWindow.on("closed", () => {
    popupWindow = null;
  });

  return popupWindow;
}

/**
 * Toggle popup window visibility
 */
export function togglePopup(): void {
  if (!popupWindow) {
    createPopupWindow();
  }

  if (popupWindow?.isVisible()) {
    popupWindow.hide();
  } else {
    // Reposition to current cursor screen
    const cursorPoint = screen.getCursorScreenPoint();
    const display = screen.getDisplayNearestPoint(cursorPoint);
    const size = POPUP_SIZES[currentSize];
    const x = Math.round(
      display.bounds.x + (display.bounds.width - size.width) / 2,
    );
    const y = display.bounds.y + 80;

    popupWindow?.setPosition(x, y);
    popupWindow?.show();
    popupWindow?.focus();

    // Tell the popup to focus the input
    popupWindow?.webContents.send("popup:focus-input");
  }
}

/**
 * Set popup size mode
 */
export function setPopupSize(size: PopupSize): void {
  currentSize = size;
  if (popupWindow) {
    const dimensions = POPUP_SIZES[size];

    // Get current position and center the resize
    const bounds = popupWindow.getBounds();
    const cursorPoint = screen.getCursorScreenPoint();
    const display = screen.getDisplayNearestPoint(cursorPoint);

    const newX = Math.round(
      display.bounds.x + (display.bounds.width - dimensions.width) / 2,
    );
    const newY =
      size === "full" ? display.bounds.y + 40 : display.bounds.y + 80;

    popupWindow.setBounds(
      {
        x: newX,
        y: newY,
        width: dimensions.width,
        height: dimensions.height,
      },
      true,
    ); // animate

    // Notify renderer of size change
    popupWindow.webContents.send("popup:size-changed", size);
  }
}

/**
 * Get current popup size
 */
export function getPopupSize(): PopupSize {
  return currentSize;
}

/**
 * Show popup window
 */
export function showPopup(): void {
  if (!popupWindow) {
    createPopupWindow();
  }

  if (!popupWindow?.isVisible()) {
    togglePopup();
  }
}

/**
 * Hide popup window
 */
export function hidePopup(): void {
  popupWindow?.hide();
}

/**
 * Check and request accessibility permissions (macOS)
 */
export async function checkAccessibilityPermissions(): Promise<boolean> {
  if (process.platform !== "darwin") {
    return true; // Not needed on Windows/Linux
  }

  const isTrusted = systemPreferences.isTrustedAccessibilityClient(false);

  if (!isTrusted) {
    const hasPrompted = store.get("accessibilityPrompted") as boolean;

    if (!hasPrompted) {
      const result = await dialog.showMessageBox({
        type: "info",
        title: "Accessibility Permission Required",
        message:
          "BusinessOS needs accessibility permissions to use global keyboard shortcuts.",
        detail:
          'This allows you to trigger Quick Chat (⌘+Shift+Space) and other shortcuts from anywhere on your Mac.\n\nClick "Open Settings" to grant permission in System Preferences → Privacy & Security → Accessibility.',
        buttons: ["Open Settings", "Later"],
        defaultId: 0,
        cancelId: 1,
      });

      store.set("accessibilityPrompted", true);

      if (result.response === 0) {
        // Open System Preferences to Accessibility
        systemPreferences.isTrustedAccessibilityClient(true); // This prompts the system dialog
      }
    }

    return false;
  }

  return true;
}

/**
 * Register global shortcuts
 */
export async function registerGlobalShortcuts(): Promise<void> {
  // Check accessibility permissions first (macOS)
  const hasPermission = await checkAccessibilityPermissions();

  if (!hasPermission && process.platform === "darwin") {
    console.warn(
      "Accessibility permission not granted. Global shortcuts may not work.",
    );
    // Still try to register - they'll work once permission is granted
  }

  // Unregister all existing shortcuts first
  globalShortcut.unregisterAll();

  const shortcuts = getShortcuts();

  // Main popup toggle shortcut (Cmd+Shift+Space by default)
  if (shortcuts.quickChat) {
    const registered = globalShortcut.register(shortcuts.quickChat, () => {
      console.log("Quick Chat shortcut triggered:", shortcuts.quickChat);
      togglePopup();
    });

    if (registered) {
      console.log(`Quick Chat shortcut registered: ${shortcuts.quickChat}`);
    } else {
      console.error(
        `Failed to register Quick Chat shortcut: ${shortcuts.quickChat}`,
      );
    }
  }

  // Spotlight-style shortcut (Cmd+Space by default)
  // Note: This may conflict with macOS Spotlight if not disabled in System Preferences
  if (shortcuts.spotlight && shortcuts.spotlight !== shortcuts.quickChat) {
    const spotlightRegistered = globalShortcut.register(
      shortcuts.spotlight,
      () => {
        console.log("Spotlight shortcut triggered:", shortcuts.spotlight);
        togglePopup();
      },
    );

    if (spotlightRegistered) {
      console.log(`Spotlight shortcut registered: ${shortcuts.spotlight}`);
    } else {
      console.warn(
        `Could not register ${shortcuts.spotlight} - may be in use by system Spotlight.`,
      );
    }
  }

  // Voice input shortcut (Cmd+D by default) - triggers recording in popup
  if (shortcuts.voiceInput) {
    const voiceRegistered = globalShortcut.register(
      shortcuts.voiceInput,
      () => {
        console.log("Voice shortcut triggered:", shortcuts.voiceInput);
        // Show popup and start recording
        if (!popupWindow?.isVisible()) {
          togglePopup();
        }
        // Tell popup to start voice recording
        setTimeout(() => {
          popupWindow?.webContents.send("popup:start-voice-recording");
        }, 200);
      },
    );

    if (voiceRegistered) {
      console.log(`Voice shortcut registered: ${shortcuts.voiceInput}`);
    } else {
      console.warn(
        `Could not register voice shortcut: ${shortcuts.voiceInput}`,
      );
    }
  }

  // Alternative shortcut (Option+Space on macOS) - always register this
  if (process.platform === "darwin") {
    globalShortcut.register("Alt+Space", () => {
      togglePopup();
    });
  }
}

/**
 * Unregister all global shortcuts
 */
export function unregisterGlobalShortcuts(): void {
  globalShortcut.unregisterAll();
  console.log("Global shortcuts unregistered");
}

/**
 * Create system tray icon
 */
export function createTray(): Tray {
  // Create a simple tray icon (you can replace with actual icon)
  const iconPath =
    process.platform === "darwin"
      ? path.join(__dirname, "../../resources/icons/tray-icon.png")
      : path.join(__dirname, "../../resources/icons/tray-icon.png");

  // Create a template image for macOS (16x16 or 22x22)
  let icon: Electron.NativeImage;
  try {
    icon = nativeImage.createFromPath(iconPath);
    if (process.platform === "darwin") {
      icon = icon.resize({ width: 18, height: 18 });
      icon.setTemplateImage(true);
    }
  } catch {
    // Fallback: create a simple colored icon
    icon = nativeImage.createEmpty();
  }

  tray = new Tray(icon);
  tray.setToolTip("BusinessOS");

  const shortcuts = getShortcuts();
  const contextMenu = Menu.buildFromTemplate([
    {
      label: "Quick Chat",
      accelerator: shortcuts.quickChat,
      click: () => togglePopup(),
    },
    {
      label: "Start Meeting Recording",
      click: () => {
        // Send to popup to start recording
        showPopup();
        setTimeout(() => {
          popupWindow?.webContents.send("popup:start-meeting-recording");
        }, 300);
      },
    },
    { type: "separator" },
    {
      label: "Open BusinessOS",
      click: () => {
        const mainWindow = getMainWindow();
        if (mainWindow) {
          mainWindow.show();
          mainWindow.focus();
        }
      },
    },
    { type: "separator" },
    {
      label: "Quit",
      accelerator: "CommandOrControl+Q",
      click: () => {
        app.quit();
      },
    },
  ]);

  tray.setContextMenu(contextMenu);

  // Click on tray icon toggles popup
  tray.on("click", () => {
    togglePopup();
  });

  console.log("System tray created");
  return tray;
}

/**
 * Set up IPC handlers for popup
 */
export function setupPopupIPC(): void {
  // Hide popup
  ipcMain.on("popup:hide", () => {
    hidePopup();
  });

  // Send message to main chat
  ipcMain.on("popup:send-to-main", (_event, message: string) => {
    const mainWindow = getMainWindow();
    if (mainWindow) {
      mainWindow.webContents.send("chat:message", message);
      mainWindow.show();
      mainWindow.focus();
    }
  });

  // Open main window
  ipcMain.on("popup:open-main", () => {
    const mainWindow = getMainWindow();
    if (mainWindow) {
      mainWindow.show();
      mainWindow.focus();
    }
    hidePopup();
  });

  // Set popup size
  ipcMain.on(
    "popup:set-size",
    (_event, size: "small" | "medium" | "large" | "full") => {
      setPopupSize(size);
    },
  );

  // Get current popup size
  ipcMain.handle("popup:get-size", () => {
    return getPopupSize();
  });

  // Expand popup and open main app (after sending message)
  ipcMain.on("popup:expand-to-full", () => {
    const mainWindow = getMainWindow();
    if (mainWindow) {
      mainWindow.show();
      mainWindow.focus();
      hidePopup();
    }
  });

  // Get all shortcuts
  ipcMain.handle("shortcuts:get", () => {
    return getShortcuts();
  });

  // Update a single shortcut
  ipcMain.handle(
    "shortcuts:set",
    async (_event, key: string, accelerator: string) => {
      const shortcuts = getShortcuts();
      (shortcuts as any)[key] = accelerator;
      store.set("shortcuts", shortcuts);

      // Re-register all shortcuts with new configuration
      await registerGlobalShortcuts();

      return { success: true, shortcuts };
    },
  );

  // Reset shortcuts to defaults
  ipcMain.handle("shortcuts:reset", async () => {
    store.set("shortcuts", {
      quickChat: "CommandOrControl+Shift+Space",
      spotlight: "CommandOrControl+Space",
      voiceInput: "CommandOrControl+D",
    });

    await registerGlobalShortcuts();

    return { success: true, shortcuts: getShortcuts() };
  });

  // Check accessibility permissions
  ipcMain.handle("shortcuts:check-accessibility", async () => {
    if (process.platform !== "darwin") {
      return { granted: true };
    }
    const isTrusted = systemPreferences.isTrustedAccessibilityClient(false);
    return { granted: isTrusted };
  });

  // Request accessibility permissions (opens system dialog)
  ipcMain.handle("shortcuts:request-accessibility", async () => {
    if (process.platform !== "darwin") {
      return { granted: true };
    }
    // This will open the system preferences dialog
    systemPreferences.isTrustedAccessibilityClient(true);
    return { requested: true };
  });

  console.log("Popup IPC handlers registered");
}

/**
 * Get popup window instance
 */
export function getPopupWindow(): BrowserWindow | null {
  return popupWindow;
}

/**
 * Initialize popup system
 */
export function initializePopupSystem(): void {
  createPopupWindow();
  registerGlobalShortcuts();
  createTray();
  setupPopupIPC();
  console.log("Popup system initialized");
}

/**
 * Cleanup popup system
 */
export function cleanupPopupSystem(): void {
  unregisterGlobalShortcuts();
  if (tray) {
    tray.destroy();
    tray = null;
  }
  if (popupWindow) {
    popupWindow.destroy();
    popupWindow = null;
  }
  console.log("Popup system cleaned up");
}
