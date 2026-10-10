import { ipcMain, type WebContents } from "electron";
import type * as PtyModule from "node-pty";
import os from "os";
import { randomUUID } from "crypto";
import fs from "fs";
import {
  buildPtyEnv,
  loginShellArgs,
  resolveDefaultShell,
  resolveShellCandidates,
} from "./env";

/**
 * node-pty is a NATIVE module. It can fail to load at runtime for reasons that
 * have nothing to do with our code: an arm64 build running on an Intel Mac
 * under Rosetta, a mismatched Electron ABI, or a missing/broken native
 * dependency. A static top-level `import` would THROW during module load, which
 * takes down setupTerminalHandlers entirely (its caller wraps registration in
 * try/catch), so `terminal:create` never gets registered and the terminal UI
 * silently shows nothing. Instead we load node-pty lazily and REMEMBER the
 * failure, so we can still register handlers that report the problem loudly and
 * clearly to the renderer.
 */
let pty: typeof PtyModule | null = null;
let ptyLoadError: string | null = null;

const PTY_UNAVAILABLE_REASON =
  "The local terminal native module could not load on this system (arch mismatch or missing dependency)";

function loadPty(): typeof PtyModule | null {
  if (pty || ptyLoadError) return pty;
  try {
    // Lazy require so a native-load failure is catchable instead of crashing
    // the whole module at import time.
    pty = require("node-pty") as typeof PtyModule;
  } catch (error) {
    ptyLoadError = PTY_UNAVAILABLE_REASON;
    console.error(
      `[terminal] node-pty failed to load on arch=${process.arch}:`,
      error,
    );
  }
  return pty;
}

/**
 * Local terminal: spawns a real PTY (the user's shell) in the Electron main
 * process and bridges it to xterm.js in the renderer over IPC. This is the
 * "Local" terminal mode — it runs on the user's actual machine, with no backend,
 * no cloud, and no auth (the main process already IS the local machine).
 *
 * SESSION PERSISTENCE: PTYs live in the main process and SURVIVE a renderer
 * refresh/reload. Each session buffers its recent output, so when the renderer
 * reloads it can `terminal:reconnect` by id, rebind to the live shell, and
 * replay the scrollback — the shell keeps running (and any running command keeps
 * going). A session only dies when the user closes the tab (terminal:kill) or on
 * app quit (killAllTerminals). Previously onData captured the ORIGINAL
 * WebContents, so after a refresh the shell's output went to a destroyed window
 * and the terminal looked reset.
 *
 * Sandbox / Production (MIOSA) modes are handled separately over WebSockets and
 * are out of scope here.
 */

interface Session {
  proc: PtyModule.IPty;
  wc: WebContents | null; // the CURRENT renderer bound to this session
  buffer: string[]; // recent output, replayed on reconnect
  bytes: number;
  shell: string;
  exited: boolean;
  exitCode: number | null;
}

const sessions = new Map<string, Session>();

// Cap the per-session scrollback we retain for replay (~512KB).
const MAX_BUFFER_BYTES = 512 * 1024;

function pushBuffer(s: Session, data: string): void {
  s.buffer.push(data);
  s.bytes += data.length;
  while (s.bytes > MAX_BUFFER_BYTES && s.buffer.length > 1) {
    s.bytes -= s.buffer.shift()!.length;
  }
}

export function setupTerminalHandlers(): void {
  // Probe node-pty up front so any native-load failure is logged at startup
  // (with the process arch) rather than only on first terminal:create.
  loadPty();

  // Report whether the local terminal can run on this machine. The renderer
  // calls this before/alongside terminal:create so it can show a clear message
  // instead of a blank pane when node-pty is unavailable.
  ipcMain.handle("terminal:available", () => {
    const mod = loadPty();
    return mod
      ? { available: true }
      : { available: false, reason: ptyLoadError ?? PTY_UNAVAILABLE_REASON };
  });

  // Create a PTY. Returns the session id the renderer uses to address it.
  ipcMain.handle(
    "terminal:create",
    (
      event,
      opts: {
        cols?: number;
        rows?: number;
        cwd?: string;
        shell?: string;
        env?: Record<string, string>;
      } = {},
    ) => {
      const ptyMod = loadPty();
      if (!ptyMod) {
        // node-pty could not load: fail LOUDLY with a clear reason instead of
        // throwing (which the renderer would only see as a rejected invoke).
        return { error: ptyLoadError ?? PTY_UNAVAILABLE_REASON };
      }

      const id = randomUUID();
      const requestedShell = opts.shell || resolveDefaultShell();
      const shells = resolveShellCandidates(requestedShell);
      const cwd = opts.cwd || os.homedir();
      let proc: PtyModule.IPty;
      let shell = requestedShell;
      let spawnWarning: string | undefined;
      const errors: string[] = [];
      try {
        fs.mkdirSync(cwd, { recursive: true });
        for (const candidate of shells) {
          try {
            proc = ptyMod.spawn(candidate, loginShellArgs(candidate), {
              name: "xterm-256color",
              cols: opts.cols || 80,
              rows: opts.rows || 24,
              cwd,
              env: buildPtyEnv(candidate, opts.env),
            });
            shell = candidate;
            if (requestedShell !== "auto" && candidate !== requestedShell) {
              spawnWarning = `Could not start ${requestedShell}; opened ${candidate} instead.`;
            }
            break;
          } catch (error) {
            errors.push(`${candidate}: ${String(error)}`);
          }
        }
      } catch (error) {
        return {
          error: `Failed to prepare local shell in ${cwd}: ${String(error)}`,
        };
      }

      if (!proc!) {
        return {
          error: `Failed to start local shell in ${cwd}. Tried: ${errors.join(" | ")}`,
        };
      }

      const session: Session = {
        proc,
        wc: event.sender,
        buffer: [],
        bytes: 0,
        shell,
        exited: false,
        exitCode: null,
      };
      sessions.set(id, session);

      // Always send to the session's CURRENT wc (rebound on reconnect), and keep
      // a scrollback buffer for replay.
      proc.onData((data) => {
        pushBuffer(session, data);
        const wc = session.wc;
        if (wc && !wc.isDestroyed()) wc.send("terminal:data", { id, data });
      });
      proc.onExit(({ exitCode }) => {
        session.exited = true;
        session.exitCode = exitCode;
        const wc = session.wc;
        if (wc && !wc.isDestroyed()) wc.send("terminal:exit", { id, exitCode });
        // Keep the entry briefly so a reconnect can learn it exited, then drop.
        setTimeout(() => sessions.delete(id), 5000);
      });

      return { id, shell, cwd, warning: spawnWarning };
    },
  );

  // Reconnect an existing session to the (possibly new) renderer after a reload.
  // Rebinds the session's wc and returns the buffered scrollback to replay.
  ipcMain.handle("terminal:reconnect", (event, payload: { id: string }) => {
    const s = sessions.get(payload.id);
    if (!s || s.exited) {
      return { alive: false };
    }
    s.wc = event.sender; // route future output to the new renderer
    return { alive: true, shell: s.shell, buffer: s.buffer.join("") };
  });

  // List currently-alive session ids (so the renderer knows what to reconnect).
  ipcMain.handle("terminal:list", () => {
    return {
      ids: Array.from(sessions.entries())
        .filter(([, s]) => !s.exited)
        .map(([id]) => id),
    };
  });

  // Stdin from the renderer → PTY. Fire-and-forget for low latency.
  ipcMain.on("terminal:input", (_e, payload: { id: string; data: string }) => {
    sessions.get(payload.id)?.proc.write(payload.data);
  });

  // Resize from xterm fit → PTY.
  ipcMain.on(
    "terminal:resize",
    (_e, payload: { id: string; cols: number; rows: number }) => {
      try {
        sessions.get(payload.id)?.proc.resize(payload.cols, payload.rows);
      } catch {
        /* resize can race with exit; ignore */
      }
    },
  );

  // Kill a session — ONLY when the user closes the tab. A refresh must never
  // call this (it should reconnect instead).
  ipcMain.handle("terminal:kill", (_e, payload: { id: string }) => {
    const s = sessions.get(payload.id);
    if (s) {
      try {
        s.proc.kill();
      } catch {
        /* already gone */
      }
      sessions.delete(payload.id);
    }
    return { success: true };
  });
}

// Kill every PTY (called on app quit ONLY, never on renderer reload).
export function killAllTerminals(): void {
  for (const s of sessions.values()) {
    try {
      s.proc.kill();
    } catch {
      /* ignore */
    }
  }
  sessions.clear();
}
