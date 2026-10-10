/**
 * chatSSEParser — Typed SSE stream parser for BOS chat endpoints.
 *
 * Wire format (from sse_writer.go):
 *   event: <type>\n
 *   data: <json>\n
 *   \n
 *
 * The JSON payload is a StreamEvent: { type, content?, data? }
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ChatSSEEventType =
  | "token"
  | "blocks"
  | "thinking_chunk"
  | "thinking_start"
  | "thinking_end"
  | "tool_call"
  | "tool_result"
  | "error"
  | "done"
  | "signal_classified"
  | "artifact_start"
  | "artifact_complete"
  | "artifact_error"
  | "delegating";

interface BaseEvent {
  type: ChatSSEEventType;
  timestamp?: string;
}

export interface TokenEvent extends BaseEvent {
  type: "token";
  content: string;
}

export interface BlocksEvent extends BaseEvent {
  type: "blocks";
  data: unknown;
}

export interface ThinkingChunkEvent extends BaseEvent {
  type: "thinking_chunk";
  content: string;
  /** The thinking step label, e.g. "analyzing", "planning" */
  step?: string;
  agent?: string;
}

export interface ThinkingStartEvent extends BaseEvent {
  type: "thinking_start";
}

export interface ThinkingEndEvent extends BaseEvent {
  type: "thinking_end";
}

export interface ToolCallEvent extends BaseEvent {
  type: "tool_call";
  toolName: string;
  toolCallId: string;
  params: Record<string, unknown>;
  status: "calling" | "success" | "error";
}

export interface ToolResultEvent extends BaseEvent {
  type: "tool_result";
  toolCallId: string;
  toolName: string;
  content: string;
  status: "success" | "error";
}

export interface ErrorEvent extends BaseEvent {
  type: "error";
  message: string;
  code?: string;
}

export interface DoneEvent extends BaseEvent {
  type: "done";
}

export interface SignalClassifiedEvent extends BaseEvent {
  type: "signal_classified";
  mode: "BUILD" | "ASSIST" | "ANALYZE" | "EXECUTE" | "MAINTAIN";
  confidence?: number;
  genre?: "DIRECT" | "INFORM" | "COMMIT" | "DECIDE" | "EXPRESS";
  docType?: string;
  weight?: number;
}

export interface ArtifactStartEvent extends BaseEvent {
  type: "artifact_start";
  id?: string;
  artifactType?: string;
  title?: string;
}

export interface ArtifactCompleteEvent extends BaseEvent {
  type: "artifact_complete";
  id?: string;
  artifactType?: string;
  title?: string;
  content?: string;
}

export interface ArtifactErrorEvent extends BaseEvent {
  type: "artifact_error";
  message: string;
}

export interface DelegatingEvent extends BaseEvent {
  type: "delegating";
  agent?: string;
}

export type ChatSSEEvent =
  | TokenEvent
  | BlocksEvent
  | ThinkingChunkEvent
  | ThinkingStartEvent
  | ThinkingEndEvent
  | ToolCallEvent
  | ToolResultEvent
  | ErrorEvent
  | DoneEvent
  | SignalClassifiedEvent
  | ArtifactStartEvent
  | ArtifactCompleteEvent
  | ArtifactErrorEvent
  | DelegatingEvent;

// ---------------------------------------------------------------------------
// Internal raw payload type (matches StreamEvent in events.go)
// ---------------------------------------------------------------------------

interface RawStreamEvent {
  type: string;
  content?: string;
  data?: unknown;
}

interface RawThinkingStep {
  step?: string;
  content?: string;
  agent?: string;
  completed?: boolean;
}

interface RawToolCallEvent {
  tool_call_id?: string;
  tool_name?: string;
  params?: Record<string, unknown> | string;
  parameters?: Record<string, unknown>;
  status?: string;
  result?: string;
}

interface RawArtifact {
  id?: string;
  type?: string;
  artifactType?: string;
  title?: string;
  content?: string;
}

// ---------------------------------------------------------------------------
// Normalise event-type strings from the backend
// ---------------------------------------------------------------------------

/**
 * Map backend event type strings (including SDK dot-notation variants) to our
 * canonical ChatSSEEventType values. Returns null for unknown/skippable types.
 */
function normaliseEventType(raw: string): ChatSSEEventType | null {
  switch (raw) {
    case "token":
    case "chunk":
    case "content":
    case "content_delta":
      return "token";
    case "blocks":
      return "blocks";
    case "thinking_chunk":
    case "thinking":
      return "thinking_chunk";
    case "thinking_start":
      return "thinking_start";
    case "thinking_end":
      return "thinking_end";
    case "tool_call":
      return "tool_call";
    case "tool_result":
      return "tool_result";
    case "error":
      return "error";
    case "done":
    case "end":
    case "complete":
      return "done";
    case "signal_classified":
    case "signal.classified":
      return "signal_classified";
    case "artifact_start":
      return "artifact_start";
    case "artifact_complete":
      return "artifact_complete";
    case "artifact_error":
      return "artifact_error";
    case "delegating":
      return "delegating";
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// Map raw payload → typed ChatSSEEvent
// ---------------------------------------------------------------------------

function mapRawEvent(raw: RawStreamEvent): ChatSSEEvent | null {
  const type = normaliseEventType(raw.type);
  if (type === null) return null;

  switch (type) {
    case "token":
      return { type: "token", content: raw.content ?? "" };

    case "blocks":
      return { type: "blocks", data: raw.data };

    case "thinking_chunk": {
      const thinking = raw.data as RawThinkingStep | undefined;
      return {
        type: "thinking_chunk",
        content: thinking?.content ?? raw.content ?? "",
        step: thinking?.step,
        agent: thinking?.agent,
      };
    }

    case "thinking_start":
      return { type: "thinking_start" };

    case "thinking_end":
      return { type: "thinking_end" };

    case "tool_call": {
      const call = raw.data as RawToolCallEvent | undefined;
      let params: Record<string, unknown> = {};
      if (call?.params && typeof call.params === "object") {
        params = call.params;
      } else if (typeof call?.params === "string") {
        try {
          const parsed = JSON.parse(call.params);
          if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
            params = parsed as Record<string, unknown>;
          }
        } catch {
          params = {};
        }
      } else if (call?.parameters) {
        params = call.parameters;
      }
      return {
        type: "tool_call",
        toolName: call?.tool_name ?? "",
        toolCallId: call?.tool_call_id ?? call?.tool_name ?? "",
        params,
        status:
          call?.status === "completed" || call?.status === "success"
            ? "success"
            : call?.status === "error"
              ? "error"
              : "calling",
      };
    }

    case "tool_result": {
      const result = raw.data as RawToolCallEvent | undefined;
      return {
        type: "tool_result",
        toolName: result?.tool_name ?? "",
        toolCallId: result?.tool_call_id ?? result?.tool_name ?? "",
        content: result?.result ?? raw.content ?? "",
        status: result?.status === "error" ? "error" : "success",
      };
    }

    case "error": {
      // Error message can be in content (preferred) or data (legacy slash commands)
      const errMsg =
        raw.content ??
        (typeof raw.data === "string" ? raw.data : undefined) ??
        "Unknown error";
      return {
        type: "error",
        message: errMsg,
      };
    }

    case "done":
      return { type: "done" };

    case "signal_classified": {
      const signal = raw.data as Record<string, unknown> | undefined;
      return {
        type: "signal_classified",
        mode: (signal?.mode as SignalClassifiedEvent["mode"]) ?? "ASSIST",
        confidence:
          typeof signal?.confidence === "number"
            ? signal.confidence
            : undefined,
        genre:
          typeof signal?.genre === "string"
            ? (signal.genre as SignalClassifiedEvent["genre"])
            : undefined,
        docType:
          typeof signal?.doc_type === "string"
            ? (signal.doc_type as string)
            : undefined,
        weight: typeof signal?.weight === "number" ? signal.weight : undefined,
      };
    }

    case "artifact_start": {
      const artifact = raw.data as RawArtifact | undefined;
      const artifactType = artifact?.artifactType ?? (artifact?.type !== raw.type ? artifact?.type : undefined);
      return {
        type: "artifact_start",
        id: artifact?.id,
        artifactType,
        title: artifact?.title,
      };
    }

    case "artifact_complete": {
      const artifact = raw.data as RawArtifact | undefined;
      const artifactType = artifact?.artifactType ?? (artifact?.type !== raw.type ? artifact?.type : undefined);
      return {
        type: "artifact_complete",
        id: artifact?.id,
        artifactType,
        title: artifact?.title,
        content: artifact?.content,
      };
    }

    case "artifact_error":
      return {
        type: "artifact_error",
        message: raw.content ?? "Artifact error",
      };

    case "delegating": {
      const delegating = raw.data as Record<string, unknown> | undefined;
      return {
        type: "delegating",
        agent:
          typeof delegating?.agent === "string" ? delegating.agent : undefined,
      };
    }
  }
}

// ---------------------------------------------------------------------------
// SSE frame parser (handles the event:/data: line format)
// ---------------------------------------------------------------------------

interface SSEFrame {
  /** The `event:` line value, if present */
  eventName: string | null;
  /** The accumulated `data:` lines joined with newlines */
  data: string;
}

/**
 * Extract complete SSE frames from a text buffer.
 * Returns the parsed frames and the remaining incomplete buffer tail.
 *
 * SSE spec: frames are separated by a blank line (\n\n).
 * Lines starting with `event:` set the event name.
 * Lines starting with `data:` contribute to the data payload.
 * Lines starting with `:` are comments — ignored.
 */
function extractFrames(buffer: string): {
  frames: SSEFrame[];
  remainder: string;
} {
  const frames: SSEFrame[] = [];
  // Split on blank lines (frame boundaries)
  const parts = buffer.split(/\n\n/);
  // Last part may be incomplete — keep it as remainder
  const remainder = parts.pop() ?? "";

  for (const part of parts) {
    if (!part.trim()) continue;

    let eventName: string | null = null;
    const dataLines: string[] = [];

    for (const line of part.split("\n")) {
      if (line.startsWith("event:")) {
        eventName = line.slice("event:".length).trim();
      } else if (line.startsWith("data:")) {
        dataLines.push(line.slice("data:".length).trimStart());
      }
      // Ignore comment lines (`:`) and id/retry lines
    }

    if (dataLines.length > 0) {
      frames.push({ eventName, data: dataLines.join("\n") });
    }
  }

  return { frames, remainder };
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Async generator that consumes a `ReadableStreamDefaultReader<Uint8Array>` and
 * yields typed `ChatSSEEvent` values.
 *
 * Handles:
 * - Chunks split across network boundaries (buffers until frame is complete)
 * - Malformed JSON (silently skips the frame and continues)
 * - `event:` + `data:` SSE format (standard) as well as bare `data:` frames
 * - Network / reader errors (re-throws as `ErrorEvent` then terminates)
 */
export async function* parseChatSSEStream(
  reader: ReadableStreamDefaultReader<Uint8Array>,
): AsyncGenerator<ChatSSEEvent> {
  const decoder = new TextDecoder("utf-8");
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();

      if (done) {
        // Flush any remaining content in the buffer as a final frame attempt
        if (buffer.trim()) {
          const { frames } = extractFrames(buffer + "\n\n");
          for (const frame of frames) {
            const event = parseFrame(frame);
            if (event) yield event;
          }
        }
        break;
      }

      buffer += decoder.decode(value, { stream: true });

      const { frames, remainder } = extractFrames(buffer);
      buffer = remainder;

      for (const frame of frames) {
        const event = parseFrame(frame);
        if (event) {
          yield event;
          if (event.type === "done") return;
        }
      }
    }
  } catch (err) {
    // Surface network-level errors as a typed ErrorEvent
    const message = err instanceof Error ? err.message : "Stream read error";
    yield { type: "error", message };
  }
}

/**
 * Parse a single SSE frame into a typed event, or return null if the frame
 * should be skipped (empty data, malformed JSON, unknown type).
 */
function parseFrame(frame: SSEFrame): ChatSSEEvent | null {
  const raw = frame.data.trim();
  if (!raw || raw === "[DONE]") {
    // Some backends send `data: [DONE]` as a stream terminator
    return raw === "[DONE]" ? { type: "done" } : null;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // Malformed JSON — skip frame silently
    return null;
  }

  if (typeof parsed !== "object" || parsed === null) return null;

  const payload = parsed as Record<string, unknown>;

  // The event name on the SSE frame line takes precedence over the JSON `type`
  // field, but both should agree. Use whichever is available.
  const rawType =
    frame.eventName ??
    (typeof payload["type"] === "string" ? payload["type"] : null);

  if (!rawType) return null;

  const rawEvent: RawStreamEvent = {
    type: rawType,
    content:
      typeof payload["content"] === "string"
        ? payload["content"]
        : typeof payload["delta"] === "string"
          ? payload["delta"]
          : undefined,
    data: payload["data"] ?? payload,
  };

  return mapRawEvent(rawEvent);
}

/**
 * Detect whether the first received chunk looks like an SSE stream.
 * Useful for backward-compatibility when the endpoint may return plain text.
 */
export function isSSEStream(firstChunk: string): boolean {
  const trimmed = firstChunk.trimStart();
  return trimmed.startsWith("data:") || trimmed.startsWith("event:");
}
