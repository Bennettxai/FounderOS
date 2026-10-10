/**
 * Terminal WebSocket Service
 * Handles bi-directional communication with the terminal backend
 */

import { getBackendUrl } from "$lib/api/base";

export interface TerminalMessage {
  type: "input" | "output" | "resize" | "heartbeat" | "error" | "status";
  session_id?: string;
  data?: string;
  metadata?: Record<string, unknown>;
}

export interface TerminalConfig {
  cols?: number;
  rows?: number;
  shell?: string;
  cwd?: string;
  environmentMode?: string;
}

export type TerminalEventHandler = {
  onData: (data: string) => void;
  onConnect: (sessionId: string, metadata: Record<string, unknown>) => void;
  onDisconnect: () => void;
  onError: (error: string) => void;
};

export class TerminalService {
  private ws: WebSocket | null = null;
  private sessionId: string | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private config: TerminalConfig;
  private destroyed = false;

  constructor(
    private baseUrl: string,
    private handlers: TerminalEventHandler,
    config: TerminalConfig = {},
  ) {
    this.config = {
      cols: config.cols ?? 80,
      rows: config.rows ?? 24,
      shell: config.shell ?? "zsh",
      cwd: config.cwd ?? "",
      environmentMode: config.environmentMode ?? "local",
    };
  }

  /**
   * Connect to the terminal WebSocket
   */
  connect(): void {
    // Build WebSocket URL with query params
    const wsProtocol = this.baseUrl.startsWith("https") ? "wss" : "ws";
    const wsBase = this.baseUrl.replace(/^https?/, wsProtocol);

    const params = new URLSearchParams({
      cols: String(this.config.cols),
      rows: String(this.config.rows),
      shell: this.config.shell || "zsh",
      environment_mode: this.config.environmentMode || "local",
    });

    if (this.config.cwd) {
      params.set("cwd", this.config.cwd);
    }

    // Cloud mode uses /term/ws (MIOSA compute proxy), local uses /api/terminal/ws
    const termPath =
      this.config.environmentMode === "cloud" ? "/term/ws" : "/api/terminal/ws";
    const wsUrl = `${wsBase}${termPath}?${params.toString()}`;

    try {
      this.ws = new WebSocket(wsUrl);
      this.setupEventHandlers();
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
      this.handlers.onError("Failed to connect to terminal");
    }
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.startHeartbeat();
    };

    this.ws.onmessage = (event) => {
      // REDUCED LOGGING: Only log errors, not every message
      try {
        const message: TerminalMessage = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        // If not JSON, treat as raw output
        this.handlers.onData(event.data);
      }
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      this.handlers.onError("Connection error");
    };

    this.ws.onclose = (event) => {
      this.stopHeartbeat();
      this.handlers.onDisconnect();

      if (
        !event.wasClean &&
        this.reconnectAttempts < this.maxReconnectAttempts
      ) {
        this.attemptReconnect();
      }
    };
  }

  private handleMessage(message: TerminalMessage): void {
    // REDUCED LOGGING: Only log errors, not every message

    switch (message.type) {
      case "output":
        if (message.data) {
          this.handlers.onData(message.data);
        }
        break;

      case "status":
        if (message.data === "connected" && message.metadata?.session_id) {
          this.sessionId = message.metadata.session_id as string;
          this.handlers.onConnect(this.sessionId, message.metadata);
        }
        break;

      case "error":
        this.handlers.onError(message.data || "Unknown error");
        break;
    }
  }

  /**
   * Send user input to the terminal
   */
  sendInput(data: string): void {
    if (!this.isConnected()) {
      console.warn("WebSocket not connected");
      return;
    }

    const message: TerminalMessage = {
      type: "input",
      session_id: this.sessionId || undefined,
      data: data,
    };

    this.ws!.send(JSON.stringify(message));
  }

  /**
   * Send terminal resize event
   */
  resize(cols: number, rows: number): void {
    if (!this.isConnected()) return;

    this.config.cols = cols;
    this.config.rows = rows;

    const message: TerminalMessage = {
      type: "resize",
      session_id: this.sessionId || undefined,
      data: JSON.stringify({ cols, rows }),
    };

    this.ws!.send(JSON.stringify(message));
  }

  /**
   * Send heartbeat to keep connection alive
   */
  private sendHeartbeat(): void {
    if (!this.isConnected()) return;

    const message: TerminalMessage = {
      type: "heartbeat",
      session_id: this.sessionId || undefined,
    };

    this.ws!.send(JSON.stringify(message));
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatInterval = setInterval(() => {
      this.sendHeartbeat();
    }, 30000); // Every 30 seconds
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  /**
   * Attempt to reconnect with exponential backoff
   */
  private attemptReconnect(): void {
    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 10000);

    setTimeout(() => {
      if (!this.destroyed) this.connect();
    }, delay);
  }

  /**
   * Disconnect from the terminal
   */
  disconnect(): void {
    this.destroyed = true;
    this.stopHeartbeat();

    if (this.ws) {
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
    }

    this.sessionId = null;
  }

  /**
   * Check if WebSocket is connected
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Get current session ID
   */
  getSessionId(): string | null {
    return this.sessionId;
  }
}

/**
 * Create a terminal service instance with the default API URL.
 * If computerUrl is provided (e.g. "https://myosa.sandbox.miosa.ai"),
 * connects to the cloud computer's terminal instead of the local backend.
 */
export function createTerminalService(
  handlers: TerminalEventHandler,
  config?: TerminalConfig,
  computerUrl?: string,
): TerminalService {
  // Cloud computer mode: connect to the VM's terminal via compute proxy
  if (computerUrl) {
    return new TerminalService(computerUrl, handlers, {
      ...config,
      environmentMode: "cloud",
    });
  }

  // Check for Electron or custom API URL
  const customUrl =
    typeof window !== "undefined"
      ? (window as unknown as { __API_URL__?: string }).__API_URL__
      : undefined;

  // In local dev, use the Vite proxy on the active frontend origin so the
  // terminal websocket shares the same cookie domain as normal auth requests.
  // The local desktop runner uses :5273, while older docs/tests use :5173.
  const isLocalFrontend =
    typeof window !== "undefined" &&
    (window.location.hostname === "localhost" ||
      window.location.hostname === "127.0.0.1") &&
    window.location.port !== "" &&
    window.location.port !== "8801";

  const apiUrl =
    customUrl || (isLocalFrontend ? window.location.origin : getBackendUrl());

  return new TerminalService(apiUrl, handlers, config);
}

/**
 * Cloud terminal session info returned by GET /api/computer/terminal-session.
 */
export interface CloudTerminalSession {
  mode: "cloud" | "local";
  ws_url?: string;
  session_id?: string;
  computer_id?: string;
  slug?: string;
  expires_at?: number;
  message?: string;
}

/**
 * Get a cloud terminal session with a pre-authenticated WebSocket URL.
 * Calls the BusinessOS backend which proxies to MIOSA to create a PTY session.
 * Returns the full wss:// URL with auth token, ready to connect.
 */
export async function getCloudTerminalSession(): Promise<CloudTerminalSession | null> {
  try {
    const res = await fetch(
      `${getBackendUrl()}/api/computer/terminal-session`,
      {
        credentials: "include",
        signal: AbortSignal.timeout(10000),
      },
    );
    if (!res.ok) return null;
    const data: CloudTerminalSession = await res.json();
    return data;
  } catch {
    return null;
  }
}

/** @deprecated Use getCloudTerminalSession instead */
export async function getCloudTerminalUrl(): Promise<string | null> {
  const session = await getCloudTerminalSession();
  if (session?.mode === "cloud" && session.ws_url) {
    // Extract base URL from wss URL
    try {
      const url = new URL(session.ws_url.replace("wss:", "https:"));
      return `${url.protocol}//${url.host}`;
    } catch {
      return null;
    }
  }
  return null;
}
