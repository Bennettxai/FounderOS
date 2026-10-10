/**
 * SSE Client - Robust EventSource wrapper with auto-reconnection
 *
 * @example
 * const client = new SSEClient('/api/stream', handleMessage, handleError);
 * client.connect();
 * // Later: client.disconnect();
 */

import type { ProgressEvent } from "$lib/types/agent";

export interface SSEClientOptions {
  maxReconnectAttempts?: number;
  baseReconnectDelay?: number;
  debug?: boolean;
}

const DEFAULT_OPTIONS: Required<SSEClientOptions> = {
  maxReconnectAttempts: 5,
  baseReconnectDelay: 1000,
  debug: false,
};

export class SSEClient<T = ProgressEvent> {
  private eventSource: EventSource | null = null;
  private reconnectAttempts = 0;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private options: Required<SSEClientOptions>;
  private isManualDisconnect = false;

  constructor(
    private url: string,
    private onMessage: (data: T) => void,
    private onError?: (error: Error) => void,
    private onOpen?: () => void,
    options: SSEClientOptions = {},
  ) {
    this.options = { ...DEFAULT_OPTIONS, ...options };
  }

  connect(): void {
    if (this.eventSource) {
      this.log("Already connected, ignoring connect() call");
      return;
    }

    this.isManualDisconnect = false;

    try {
      this.log(`Connecting to ${this.url}`);
      this.eventSource = new EventSource(this.url);

      this.eventSource.onopen = () => {
        this.log("Connection opened");
        this.reconnectAttempts = 0;
        this.onOpen?.();
      };

      this.eventSource.onmessage = (event) => {
        try {
          const data: T = JSON.parse(event.data);
          this.log("Message received:", data);
          this.onMessage(data);
        } catch (err) {
          const parseError = new Error(
            `Failed to parse SSE message: ${err instanceof Error ? err.message : "Unknown error"}`,
          );
          this.log("Parse error:", parseError);
          this.onError?.(parseError);
        }
      };

      this.eventSource.onerror = (event) => {
        this.log("Connection error:", event);

        if (this.eventSource) {
          this.eventSource.close();
          this.eventSource = null;
        }

        if (!this.isManualDisconnect) {
          if (this.reconnectAttempts < this.options.maxReconnectAttempts) {
            this.scheduleReconnect();
          } else {
            const error = new Error(
              `SSE connection failed after ${this.options.maxReconnectAttempts} attempts`,
            );
            this.log("Max reconnection attempts reached");
            this.onError?.(error);
          }
        }
      };
    } catch (err) {
      const error = new Error(
        `Failed to establish SSE connection: ${err instanceof Error ? err.message : "Unknown error"}`,
      );
      this.log("Connection error:", error);
      this.onError?.(error);
    }
  }

  /** Closes connection and prevents auto-reconnection. Call in onDestroy. */
  disconnect(): void {
    this.log("Disconnecting...");
    this.isManualDisconnect = true;

    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    this.reconnectAttempts = 0;
    this.log("Disconnected");
  }

  isConnected(): boolean {
    return (
      this.eventSource !== null &&
      this.eventSource.readyState === EventSource.OPEN
    );
  }

  getReadyState(): number {
    return this.eventSource?.readyState ?? EventSource.CLOSED;
  }

  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }

  /** Exponential backoff: 1s, 2s, 3s, 4s, 5s */
  private scheduleReconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
    }

    this.reconnectAttempts++;
    const delay = this.options.baseReconnectDelay * this.reconnectAttempts;

    this.log(
      `Scheduling reconnection attempt ${this.reconnectAttempts}/${this.options.maxReconnectAttempts} in ${delay}ms`,
    );

    this.reconnectTimeout = setTimeout(() => {
      this.log(`Reconnection attempt ${this.reconnectAttempts}`);
      this.connect();
    }, delay);
  }

  private log(...args: unknown[]): void {
    if (this.options.debug) {
      if (import.meta.env.DEV) console.log("[SSEClient]", ...args);
    }
  }
}

/** Factory for agent progress SSE client */
export function createAgentProgressClient(
  queueItemId: string,
  onMessage: (event: ProgressEvent) => void,
  onError?: (error: Error) => void,
  onOpen?: () => void,
  options?: SSEClientOptions,
): SSEClient<ProgressEvent> {
  const url = `/api/osa/module-instances/generate/${queueItemId}/stream`;
  return new SSEClient<ProgressEvent>(url, onMessage, onError, onOpen, {
    debug: true,
    ...options,
  });
}
