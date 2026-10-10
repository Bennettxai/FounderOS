/**
 * Local terminal transport for the Electron desktop app.
 *
 * Talks to the node-pty PTY running in the Electron main process over IPC, and
 * exposes the same surface as the WebSocket TerminalService
 * ({ isConnected, sendInput, resize, disconnect, connect }) so it drops straight
 * into TerminalShell's local branch. This is the "Local" mode — a real shell on
 * the user's own machine, no backend / cloud / auth involved.
 */

interface ElectronTerminalApi {
  create(opts?: {
    cols?: number;
    rows?: number;
    cwd?: string;
    shell?: string;
    env?: Record<string, string>;
  }): Promise<{
    id?: string;
    shell?: string;
    cwd?: string;
    warning?: string;
    error?: string;
  }>;
  input(id: string, data: string): void;
  resize(id: string, cols: number, rows: number): void;
  kill(id: string): Promise<unknown>;
  reconnect(
    id: string,
  ): Promise<{ alive: boolean; shell?: string; buffer?: string }>;
  onData(cb: (p: { id: string; data: string }) => void): () => void;
  onExit(cb: (p: { id: string; exitCode: number }) => void): () => void;
}

export interface ElectronTerminalHandlers {
  onData: (data: string) => void;
  onConnect: (sessionId: string) => void;
  onDisconnect: () => void;
  onError: (error: string) => void;
}

export interface ElectronTerminalConfig {
  cols?: number;
  rows?: number;
  cwd?: string;
  shell?: string;
  env?: Record<string, string>;
  // If set, try to RECONNECT to this existing PTY session (surviving a reload)
  // before spawning a fresh shell. Enables terminal persistence across refresh.
  resumeId?: string;
}

function getApi(): ElectronTerminalApi | null {
  if (typeof window === "undefined") return null;
  const electron = (
    window as unknown as { electron?: { terminal?: ElectronTerminalApi } }
  ).electron;
  return electron?.terminal ?? null;
}

/** True when the local PTY bridge is available (running inside the desktop app). */
export function isElectronTerminalAvailable(): boolean {
  return getApi() !== null;
}

/**
 * Build a TerminalService-compatible object backed by a local node-pty session.
 */
export function createElectronTerminal(
  handlers: ElectronTerminalHandlers,
  config?: ElectronTerminalConfig,
) {
  const api = getApi();
  let id: string | null = null;
  let connected = false;
  let offData: (() => void) | null = null;
  let offExit: (() => void) | null = null;

  // Buffer PTY output that arrives before create() resolves with our id, so we
  // never drop the initial shell prompt. Keyed by session id; once our id is
  // known we flush ours and discard the rest (those belong to other panes).
  const preIdBuffer = new Map<string, string[]>();

  return {
    isConnected: () => connected && id !== null,
    sendInput: (data: string) => {
      if (id) api?.input(id, data);
    },
    resize: (cols: number, rows: number) => {
      if (id) api?.resize(id, cols, rows);
    },
    // Get the live session id (so the pane can persist it for reconnect).
    getSessionId: () => id,
    // Detach WITHOUT killing the PTY - used on unmount/refresh so the shell keeps
    // running in the main process and can be reconnected. This is the whole point
    // of terminal persistence: a reload must not terminate the session.
    detach: () => {
      offData?.();
      offExit?.();
      connected = false;
    },
    // Explicitly close and KILL the PTY - used only when the user closes the tab.
    disconnect: () => {
      if (id) void api?.kill(id);
      offData?.();
      offExit?.();
      connected = false;
      id = null;
    },
    connect: async () => {
      if (!api) {
        handlers.onError("Local terminal is only available in the desktop app");
        return;
      }

      offData = api.onData(({ id: dataId, data }) => {
        if (id === null) {
          const arr = preIdBuffer.get(dataId) ?? [];
          arr.push(data);
          preIdBuffer.set(dataId, arr);
        } else if (dataId === id) {
          handlers.onData(data);
        }
      });
      offExit = api.onExit(({ id: exitId }) => {
        if (exitId === id) {
          connected = false;
          handlers.onDisconnect();
        }
      });

      // RECONNECT path: if this pane had a session before the reload and it is
      // still alive in the main process, rebind to it and replay the scrollback
      // instead of spawning a fresh shell. The running shell (and any running
      // command) is preserved.
      if (config?.resumeId) {
        try {
          const r = await api.reconnect(config.resumeId);
          if (r.alive) {
            id = config.resumeId;
            connected = true;
            if (r.buffer) handlers.onData(r.buffer);
            // Flush any output that streamed in between the reconnect snapshot
            // and this resolving, so a command producing output mid-reconnect
            // does not lose that slice (mirrors the fresh-create flush below).
            const midReconnect = preIdBuffer.get(id);
            if (midReconnect) midReconnect.forEach((d) => handlers.onData(d));
            preIdBuffer.clear();
            handlers.onConnect(id);
            return;
          }
        } catch {
          // fall through to a fresh shell
        }
      }

      const res = await api.create({
        cols: config?.cols,
        rows: config?.rows,
        cwd: config?.cwd,
        shell: config?.shell,
        env: config?.env,
      });

      if (res.error || !res.id) {
        handlers.onError(res.error || "Failed to start local shell");
        offData?.();
        offExit?.();
        return;
      }

      id = res.id;
      connected = true;
      if (res.warning) {
        handlers.onData(`\r\n\x1b[33m[${res.warning}]\x1b[0m\r\n`);
      }

      // Flush any output buffered before our id was known.
      const buffered = preIdBuffer.get(id);
      if (buffered) buffered.forEach((d) => handlers.onData(d));
      preIdBuffer.clear();

      handlers.onConnect(id);
    },
  };
}
