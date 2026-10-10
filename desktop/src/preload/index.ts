import { contextBridge, ipcRenderer } from "electron";

/**
 * Expose a limited API to the renderer process via contextBridge
 * This maintains security by not exposing full Node.js/Electron APIs
 */

// Database operation result type
interface DbResult<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

// Type definitions for the exposed API
export interface ElectronAPI {
  // App info
  getVersion: () => Promise<string>;
  getPlatform: () => Promise<{
    platform: string;
    arch: string;
    isPackaged: boolean;
  }>;

  // Backend
  backend: {
    getStatus: () => Promise<{
      running: boolean;
      port: number;
      url: string;
    }>;
    getUrl: () => Promise<string>;
    restart: () => Promise<boolean>;
  };

  // Diagnostics
  diagnostics: {
    collect: () => Promise<any>;
  };

  // Network
  network: {
    getStatus: () => Promise<{ online: boolean }>;
  };

  // Sync
  sync: {
    getStatus: () => Promise<{
      status: string;
      lastSync: string | null;
      pendingChanges: number;
    }>;
    trigger: () => Promise<DbResult>;
    getPending: () => Promise<DbResult<{ table: string; count: number }[]>>;
    setWorkspace: (workspaceId: string | null) => Promise<DbResult>;
  };

  // Local terminal (node-pty PTY on the user's machine)
  terminal: {
    create: (opts?: {
      cols?: number;
      rows?: number;
      cwd?: string;
      shell?: string;
      env?: Record<string, string>;
    }) => Promise<{
      id?: string;
      shell?: string;
      cwd?: string;
      warning?: string;
      error?: string;
    }>;
    available: () => Promise<{ available: boolean; reason?: string }>;
    reconnect: (
      id: string,
    ) => Promise<{ alive: boolean; shell?: string; buffer?: string }>;
    list: () => Promise<{ ids: string[] }>;
    input: (id: string, data: string) => void;
    resize: (id: string, cols: number, rows: number) => void;
    kill: (id: string) => Promise<DbResult>;
    onData: (cb: (payload: { id: string; data: string }) => void) => () => void;
    onExit: (
      cb: (payload: { id: string; exitCode: number }) => void,
    ) => () => void;
  };

  // Database - Local SQLite with Cloud Sync
  db: {
    getAll: <T = any>(
      table: string,
      where?: Record<string, any>,
    ) => Promise<DbResult<T[]>>;
    getById: <T = any>(
      table: string,
      id: string,
    ) => Promise<DbResult<T | null>>;
    create: <T = any>(
      table: string,
      data: Record<string, any>,
    ) => Promise<DbResult<T>>;
    update: <T = any>(
      table: string,
      id: string,
      data: Record<string, any>,
    ) => Promise<DbResult<T>>;
    delete: (table: string, id: string) => Promise<DbResult>;
    query: <T = any>(sql: string, params?: any[]) => Promise<DbResult<T[]>>;
    contexts: {
      getWithChildren: (parentId?: string) => Promise<DbResult<any[]>>;
    };
    conversations: {
      getWithMessages: (conversationId: string) => Promise<DbResult<any>>;
    };
    tasks: {
      getByStatus: (status?: string) => Promise<DbResult<any[]>>;
    };
    projects: {
      getWithTasks: (projectId: string) => Promise<DbResult<any>>;
    };
    calendar: {
      getByRange: (
        startDate: string,
        endDate: string,
      ) => Promise<DbResult<any[]>>;
    };
    dailyLog: {
      getToday: (userId: string) => Promise<DbResult<any | null>>;
    };
    clients: {
      getWithDeals: (clientId: string) => Promise<DbResult<any>>;
    };
    settings: {
      get: (userId: string) => Promise<DbResult<any | null>>;
      upsert: (
        userId: string,
        settings: Record<string, any>,
      ) => Promise<DbResult>;
    };
  };

  // Updates
  updates: {
    getInfo: () => Promise<{
      currentVersion: string;
      channel: string;
      isPackaged: boolean;
      owner: string;
      repo: string;
      minimumSupportedVersion: string | null;
      supported: boolean;
    }>;
    check: () => Promise<{
      ok: boolean;
      available: boolean;
      version: string | null;
      currentVersion: string;
      releaseDate?: string;
      releaseNotes?: unknown;
      error?: string;
      message?: string;
    }>;
    download: () => Promise<boolean>;
    install: () => Promise<boolean>;
  };

  // Shell
  shell: {
    openExternal: (url: string) => Promise<void>;
    openPath: (path: string) => Promise<void>;
  };

  // Auth
  auth: {
    clearSession: () => Promise<{ ok: boolean; error?: string }>;
  };

  // Dialog
  dialog: {
    showOpen: (
      options: any,
    ) => Promise<{ canceled: boolean; filePaths: string[] }>;
    showSave: (
      options: any,
    ) => Promise<{ canceled: boolean; filePath?: string }>;
    showMessage: (options: any) => Promise<{ response: number }>;
  };

  // Window
  window: {
    getState: () => Promise<any>;
    setState: (state: any) => void;
  };

  // Event listeners
  on: (channel: string, callback: (...args: any[]) => void) => () => void;
  once: (channel: string, callback: (...args: any[]) => void) => void;
}

// Allowed channels for renderer-to-main communication
const ALLOWED_INVOKE_CHANNELS = [
  "app:get-version",
  "app:get-platform",
  "app:get-path",
  "app:get-businessos-home",
  "app:set-businessos-home",
  "backend:get-status",
  "backend:get-url",
  "backend:restart",
  "diagnostics:collect",
  "network:get-status",
  "sync:get-status",
  "sync:trigger",
  "sync:getStatus",
  "sync:getPending",
  "sync:setWorkspace",
  "terminal:create",
  "terminal:available",
  "terminal:reconnect",
  "terminal:list",
  "terminal:kill",
  "agents:detect",
  "engine:setModelConfig",
  "engine:getModelConfig",
  "engine:status",
  "engine:start",
  "engine:stop",
  "engine:reveal-data",
  "updates:check",
  "updates:get-info",
  "updates:download",
  "updates:install",
  "shell:open-external",
  "shell:open-path",
  "auth:clear-session",
  "dialog:show-open",
  "dialog:show-save",
  "dialog:show-message",
  "window:get-state",
  // Database operations
  "db:getAll",
  "db:getById",
  "db:create",
  "db:update",
  "db:delete",
  "db:query",
  // Domain-specific database operations
  "db:contexts:getWithChildren",
  "db:conversations:getWithMessages",
  "db:tasks:getByStatus",
  "db:projects:getWithTasks",
  "db:calendar:getByRange",
  "db:dailyLog:getToday",
  "db:clients:getWithDeals",
  "db:settings:get",
  "db:settings:upsert",
  // Meeting recorder
  "meeting:get-sources",
  "meeting:start",
  "meeting:stop",
  "meeting:pause",
  "meeting:get-active",
  "meeting:get-sessions",
  "meeting:save-audio-chunk",
  "meeting:get-recording-path",
  // Popup
  "popup:get-size",
  // Shortcuts
  "shortcuts:get",
  "shortcuts:set",
  "shortcuts:reset",
  "shortcuts:check-accessibility",
  "shortcuts:request-accessibility",
  // Screenshot
  "screenshot:capture",
];

// Allowed channels for main-to-renderer communication
const ALLOWED_RECEIVE_CHANNELS = [
  "navigate",
  "shortcut",
  "sync:trigger",
  "sync:status",
  "update:checking",
  "update:available",
  "update:not-available",
  "update:download-progress",
  "update:downloaded",
  "update:error",
  "window:save-state",
  // Meeting recorder events
  "meeting:started",
  "meeting:stopped",
  "meeting:state-change",
  "meeting:saved",
  // Popup events
  "popup:focus-input",
  "popup:start-meeting-recording",
  "popup:start-voice-recording",
  "popup:size-changed",
];

// Expose the API to the renderer
contextBridge.exposeInMainWorld("electron", {
  // App info
  getVersion: () => ipcRenderer.invoke("app:get-version"),
  getPlatform: () => ipcRenderer.invoke("app:get-platform"),
  getPath: (name: string) => ipcRenderer.invoke("app:get-path", name),
  // BusinessOS home dir - terminal cwd + agent working directory.
  getBusinessOSHome: () => ipcRenderer.invoke("app:get-businessos-home"),
  setBusinessOSHome: (dir: string) =>
    ipcRenderer.invoke("app:set-businessos-home", dir),

  // Backend
  backend: {
    getStatus: () => ipcRenderer.invoke("backend:get-status"),
    getUrl: () => ipcRenderer.invoke("backend:get-url"),
    restart: () => ipcRenderer.invoke("backend:restart"),
  },

  diagnostics: {
    collect: () => ipcRenderer.invoke("diagnostics:collect"),
  },

  // Network
  network: {
    getStatus: () => ipcRenderer.invoke("network:get-status"),
  },

  // Optimal Engine (reached from the user's machine, so localhost engines work)
  engine: {
    test: (baseUrl: string, apiKey?: string) =>
      ipcRenderer.invoke("engine:test", baseUrl, apiKey),
    workspaces: (baseUrl: string, apiKey?: string) =>
      ipcRenderer.invoke("engine:workspaces", baseUrl, apiKey),
    createWorkspace: (
      baseUrl: string,
      apiKey: string | undefined,
      workspace: { slug: string; name: string; description?: string },
    ) => ipcRenderer.invoke("engine:createWorkspace", baseUrl, apiKey, workspace),
    setConfig: (
      workspaceId: string,
      cfg: {
        base_url: string;
        api_key: string;
        workspace: string;
        enabled: boolean;
      },
    ) => ipcRenderer.invoke("engine:setConfig", workspaceId, cfg),
    memory: (
      workspaceId: string,
      payload: {
        content: string;
        citation?: string;
        metadata?: Record<string, unknown>;
      },
    ) => ipcRenderer.invoke("engine:memory", workspaceId, payload),
    setModelConfig: (cfg: {
      provider: "anthropic" | "openai" | "ollama" | "";
      apiKey?: string;
      ollamaUrl?: string;
    }) => ipcRenderer.invoke("engine:setModelConfig", cfg),
    getModelConfig: () => ipcRenderer.invoke("engine:getModelConfig"),
    status: () => ipcRenderer.invoke("engine:status"),
    start: () => ipcRenderer.invoke("engine:start"),
    stop: () => ipcRenderer.invoke("engine:stop"),
    revealData: () => ipcRenderer.invoke("engine:reveal-data"),
  },

  // Agent CLI detection (which of claude/codex/osa/ollama are installed).
  agents: {
    detect: () => ipcRenderer.invoke("agents:detect"),
  },

  // Auth - clear the persisted session cookie from the Electron partition so
  // logout actually logs out (the cookie is flushed to disk and would otherwise
  // survive a reload and re-authenticate the user).
  auth: {
    clearSession: () => ipcRenderer.invoke("auth:clear-session"),
  },

  // Sync
  sync: {
    getStatus: () => ipcRenderer.invoke("sync:getStatus"),
    trigger: () => ipcRenderer.invoke("sync:trigger"),
    getPending: () => ipcRenderer.invoke("sync:getPending"),
    setWorkspace: (workspaceId: string | null) =>
      ipcRenderer.invoke("sync:setWorkspace", workspaceId),
  },

  // Local terminal (node-pty PTY on the user's machine)
  terminal: {
    create: (opts?: {
      cols?: number;
      rows?: number;
      cwd?: string;
      shell?: string;
      env?: Record<string, string>;
    }) => ipcRenderer.invoke("terminal:create", opts ?? {}),
    available: () => ipcRenderer.invoke("terminal:available"),
    reconnect: (id: string) => ipcRenderer.invoke("terminal:reconnect", { id }),
    list: () => ipcRenderer.invoke("terminal:list"),
    input: (id: string, data: string) =>
      ipcRenderer.send("terminal:input", { id, data }),
    resize: (id: string, cols: number, rows: number) =>
      ipcRenderer.send("terminal:resize", { id, cols, rows }),
    kill: (id: string) => ipcRenderer.invoke("terminal:kill", { id }),
    onData: (cb: (payload: { id: string; data: string }) => void) => {
      const listener = (_e: unknown, payload: { id: string; data: string }) =>
        cb(payload);
      ipcRenderer.on("terminal:data", listener);
      return () => ipcRenderer.removeListener("terminal:data", listener);
    },
    onExit: (cb: (payload: { id: string; exitCode: number }) => void) => {
      const listener = (
        _e: unknown,
        payload: { id: string; exitCode: number },
      ) => cb(payload);
      ipcRenderer.on("terminal:exit", listener);
      return () => ipcRenderer.removeListener("terminal:exit", listener);
    },
  },

  // Database - Local SQLite with Cloud Sync
  db: {
    // Generic CRUD operations
    getAll: (table: string, where?: Record<string, any>) =>
      ipcRenderer.invoke("db:getAll", table, where),
    getById: (table: string, id: string) =>
      ipcRenderer.invoke("db:getById", table, id),
    create: (table: string, data: Record<string, any>) =>
      ipcRenderer.invoke("db:create", table, data),
    update: (table: string, id: string, data: Record<string, any>) =>
      ipcRenderer.invoke("db:update", table, id, data),
    delete: (table: string, id: string) =>
      ipcRenderer.invoke("db:delete", table, id),
    query: (sql: string, params?: any[]) =>
      ipcRenderer.invoke("db:query", sql, params),

    // Domain-specific helpers
    contexts: {
      getWithChildren: (parentId?: string) =>
        ipcRenderer.invoke("db:contexts:getWithChildren", parentId),
    },
    conversations: {
      getWithMessages: (conversationId: string) =>
        ipcRenderer.invoke("db:conversations:getWithMessages", conversationId),
    },
    tasks: {
      getByStatus: (status?: string) =>
        ipcRenderer.invoke("db:tasks:getByStatus", status),
    },
    projects: {
      getWithTasks: (projectId: string) =>
        ipcRenderer.invoke("db:projects:getWithTasks", projectId),
    },
    calendar: {
      getByRange: (startDate: string, endDate: string) =>
        ipcRenderer.invoke("db:calendar:getByRange", startDate, endDate),
    },
    dailyLog: {
      getToday: (userId: string) =>
        ipcRenderer.invoke("db:dailyLog:getToday", userId),
    },
    clients: {
      getWithDeals: (clientId: string) =>
        ipcRenderer.invoke("db:clients:getWithDeals", clientId),
    },
    settings: {
      get: (userId: string) => ipcRenderer.invoke("db:settings:get", userId),
      upsert: (userId: string, settings: Record<string, any>) =>
        ipcRenderer.invoke("db:settings:upsert", userId, settings),
    },
  },

  // Updates
  updates: {
    getInfo: () => ipcRenderer.invoke("updates:get-info"),
    check: () => ipcRenderer.invoke("updates:check"),
    download: () => ipcRenderer.invoke("updates:download"),
    install: () => ipcRenderer.invoke("updates:install"),
  },

  // Shell
  shell: {
    openExternal: (url: string) =>
      ipcRenderer.invoke("shell:open-external", url),
    openPath: (path: string) => ipcRenderer.invoke("shell:open-path", path),
  },

  // Dialog
  dialog: {
    showOpen: (options: any) => ipcRenderer.invoke("dialog:show-open", options),
    showSave: (options: any) => ipcRenderer.invoke("dialog:show-save", options),
    showMessage: (options: any) =>
      ipcRenderer.invoke("dialog:show-message", options),
  },

  // Window
  window: {
    getState: () => ipcRenderer.invoke("window:get-state"),
    setState: (state: any) => ipcRenderer.send("window:set-state", state),
  },

  // Meeting recorder
  meeting: {
    getSources: () => ipcRenderer.invoke("meeting:get-sources"),
    start: (options: { title?: string; calendarEventId?: string }) =>
      ipcRenderer.invoke("meeting:start", options),
    stop: () => ipcRenderer.invoke("meeting:stop"),
    pause: () => ipcRenderer.invoke("meeting:pause"),
    getActive: () => ipcRenderer.invoke("meeting:get-active"),
    getSessions: () => ipcRenderer.invoke("meeting:get-sessions"),
    saveAudioChunk: (data: {
      sessionId: string;
      chunk: ArrayBuffer;
      isLast: boolean;
    }) => ipcRenderer.invoke("meeting:save-audio-chunk", data),
    getRecordingPath: (sessionId: string) =>
      ipcRenderer.invoke("meeting:get-recording-path", sessionId),
  },

  // Popup communication
  popup: {
    hide: () => ipcRenderer.send("popup:hide"),
    openMain: () => ipcRenderer.send("popup:open-main"),
    setSize: (size: "small" | "medium" | "large" | "full") =>
      ipcRenderer.send("popup:set-size", size),
    getSize: () => ipcRenderer.invoke("popup:get-size"),
    expandToFull: () => ipcRenderer.send("popup:expand-to-full"),
  },

  // Shortcuts management
  shortcuts: {
    get: () => ipcRenderer.invoke("shortcuts:get"),
    set: (key: string, accelerator: string) =>
      ipcRenderer.invoke("shortcuts:set", key, accelerator),
    reset: () => ipcRenderer.invoke("shortcuts:reset"),
    checkAccessibility: () =>
      ipcRenderer.invoke("shortcuts:check-accessibility"),
    requestAccessibility: () =>
      ipcRenderer.invoke("shortcuts:request-accessibility"),
  },

  // Screenshot capture
  screenshot: {
    capture: () => ipcRenderer.invoke("screenshot:capture"),
  },

  // Legacy send method (for backwards compatibility)
  send: (channel: string, ...args: any[]) => {
    const allowedSendChannels = [
      "popup:hide",
      "popup:open-main",
      "popup:set-size",
      "popup:expand-to-full",
    ];
    if (allowedSendChannels.includes(channel)) {
      ipcRenderer.send(channel, ...args);
    }
  },

  // Event listeners
  on: (channel: string, callback: (...args: any[]) => void) => {
    if (!ALLOWED_RECEIVE_CHANNELS.includes(channel)) {
      console.warn(`Attempted to listen to unauthorized channel: ${channel}`);
      return () => {};
    }

    const subscription = (_event: Electron.IpcRendererEvent, ...args: any[]) =>
      callback(...args);
    ipcRenderer.on(channel, subscription);

    // Return unsubscribe function
    return () => {
      ipcRenderer.removeListener(channel, subscription);
    };
  },

  once: (channel: string, callback: (...args: any[]) => void) => {
    if (!ALLOWED_RECEIVE_CHANNELS.includes(channel)) {
      console.warn(`Attempted to listen to unauthorized channel: ${channel}`);
      return;
    }

    ipcRenderer.once(channel, (_event, ...args) => callback(...args));
  },
} as ElectronAPI);

// Log that preload script has loaded
console.log("BusinessOS preload script loaded");
