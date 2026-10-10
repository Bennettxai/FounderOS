/**
 * ChatStreamManager.ts
 * Encapsulates the SSE streaming logic for the chat page.
 * This class is a pure service — it holds no Svelte reactive state.
 * All state mutations are delegated back to the parent via callbacks.
 */

import {
  getActiveWorkspaceHeaders,
  getApiBaseUrl,
  getCSRFToken,
  initCSRF,
} from "$lib/api/base";
import { parseChatSSEStream, isSSEStream } from "$lib/utils/chatSSEParser";
import type {
  ToolCallEvent,
  SignalClassifiedEvent,
} from "$lib/utils/chatSSEParser";

// ── Shared types ────────────────────────────────────────────────────────────

export interface UsageData {
  input_tokens: number;
  output_tokens: number;
  thinking_tokens: number;
  total_tokens: number;
  duration_ms: number;
  tps: number;
  provider: string;
  model: string;
  estimated_cost: number;
}

export interface StreamingToolCall {
  toolName: string;
  toolCallId: string;
  params: Record<string, unknown>;
  status: "running" | "completed" | "error";
  result?: string;
}

export interface ArtifactPayload {
  title: string;
  type: string;
  content: string;
}

/** Parameters passed to ChatStreamManager.send() */
export interface SendParams {
  message: string;
  model: string;
  conversationId: string | null;
  projectId: string;
  workspaceId: string | null;
  contextIds: string[];
  command?: string;
  temperature: number;
  maxTokens: number;
  topP: number;
  focusMode: string | null;
  focusOptions: Record<string, string>;
  agentId?: string;
  memoryIds: string[];
  documentIds: string[];
  nodeContext?: string;
  useCOT: boolean;
}

/** Callbacks the parent registers to receive state updates. */
export interface StreamCallbacks {
  // Message management
  addUserMessage: (id: string, content: string) => void;
  addAssistantPlaceholder: (id: string) => void;
  updateAssistantContent: (
    id: string,
    content: string,
    artifacts?: ArtifactPayload[],
    usage?: UsageData,
  ) => void;

  // Conversation
  onConversationCreated: (id: string, title: string) => void;

  // Streaming metadata
  onThinkingChunk: (chunk: string) => void;
  onThinkingStart: (preservedLines: string) => void;
  onThinkingEnd: () => void;
  onToolCall: (toolCall: StreamingToolCall) => void;
  onToolResult: (
    toolCallId: string,
    status: "completed" | "error",
    result?: string,
  ) => void;
  onSignalClassified: (
    mode: SignalClassifiedEvent["mode"],
    confidence?: number,
    genre?: SignalClassifiedEvent["genre"],
    docType?: string,
    weight?: number,
  ) => void;

  // Artifact streaming
  onArtifactStart: (title: string, artifactType: string) => void;
  onArtifactComplete: (artifact: ArtifactPayload) => void;
  onArtifactContentUpdate: (content: string) => void;

  // Streaming lifecycle
  onStreamStart: (abortController: AbortController) => void;
  onStreamEnd: () => void;
  onStreamError: (message: string) => void;

  // Post-stream
  onArtifactFinalize: (artifact: ArtifactPayload) => void;
  onModelWarmedUp: (model: string) => void;
}

// ── Manager class ────────────────────────────────────────────────────────────

export class ChatStreamManager {
  private abortController: AbortController | null = null;
  private _userAborted = false;

  stop() {
    this._userAborted = true;
    this.abortController?.abort();
  }

  async send(params: SendParams, callbacks: StreamCallbacks): Promise<void> {
    this._userAborted = false;
    this.abortController = new AbortController();
    callbacks.onStreamStart(this.abortController);

    // Timeout for getting FIRST response (headers). Once streaming starts this is cleared.
    const connectionTimeoutId = setTimeout(() => {
      this.abortController?.abort();
    }, 30_000);

    const userMsgId = crypto.randomUUID();
    const assistantMsgId = crypto.randomUUID();

    callbacks.addUserMessage(userMsgId, params.message);
    callbacks.addAssistantPlaceholder(assistantMsgId);

    // Artifact tracking (local, not reactive)
    let artifactStarted = false;
    let artifactCompleted = false;
    let displayContent = "";
    let inArtifactBlock = false;
    let fullContent = "";

    try {
      const requestBody = this._buildRequestBody(params);
      const headers = await this._buildHeaders();

      const response = await fetch(`${getApiBaseUrl()}/chat/message`, {
        credentials: "include",
        method: "POST",
        headers,
        body: JSON.stringify(requestBody),
        signal: this.abortController.signal,
      });

      clearTimeout(connectionTimeoutId); // Got headers — streaming started, clear timeout

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // Extract conversation ID from response headers
      const newConvId = response.headers.get("X-Conversation-Id");
      if (newConvId && newConvId !== params.conversationId) {
        const title =
          params.message.slice(0, 50) +
          (params.message.length > 50 ? "..." : "");
        callbacks.onConversationCreated(newConvId, title);
      }

      // ── Stream processing ────────────────────────────────────────────────
      const rawReader = response.body?.getReader();
      if (!rawReader) {
        throw new Error("Response body is not readable");
      }

      const firstRead = await rawReader.read();
      const isSSE =
        !firstRead.done &&
        isSSEStream(new TextDecoder().decode(firstRead.value));

      const updateMessage = () => {
        if (artifactStarted) {
          const inProgressArtifact =
            !artifactCompleted && inArtifactBlock
              ? [
                  {
                    title: "Creating artifact...",
                    type: "document",
                    content: "__generating__",
                  },
                ]
              : undefined;
          callbacks.updateAssistantContent(
            assistantMsgId,
            displayContent,
            inProgressArtifact,
          );
        } else {
          callbacks.updateAssistantContent(assistantMsgId, fullContent);
        }
      };

      if (isSSE && !firstRead.done) {
        // ── Typed SSE path ─────────────────────────────────────────────────
        const firstChunk = firstRead.value!;
        const replayStream = new ReadableStream<Uint8Array>({
          async start(controller) {
            controller.enqueue(firstChunk);
            while (true) {
              const { done, value } = await rawReader.read();
              if (done) {
                controller.close();
                break;
              }
              controller.enqueue(value);
            }
          },
          cancel() {
            rawReader.cancel();
          },
        });
        const typedReader =
          replayStream.getReader() as ReadableStreamDefaultReader<Uint8Array>;

        for await (const event of parseChatSSEStream(typedReader)) {
          switch (event.type) {
            case "token": {
              const tok = event.content;
              if (tok && !tok.includes("Chain of Thought Summary")) {
                fullContent += tok;
              }
              break;
            }

            case "thinking_start": {
              callbacks.onThinkingStart("");
              break;
            }

            case "thinking_chunk":
              if (event.content) {
                callbacks.onThinkingChunk(event.content);
              }
              break;

            case "thinking_end":
              callbacks.onThinkingEnd();
              break;

            case "tool_call": {
              const tc = event as ToolCallEvent;
              callbacks.onToolCall({
                toolName: tc.toolName,
                toolCallId: tc.toolCallId,
                params: tc.params,
                status:
                  tc.status === "calling"
                    ? "running"
                    : tc.status === "success"
                      ? "completed"
                      : "error",
              });
              break;
            }

            case "tool_result": {
              const tr = event;
              const status =
                tr.status === "error" ? "error" : ("completed" as const);
              callbacks.onToolResult(tr.toolCallId, status, tr.content);
              break;
            }

            case "signal_classified": {
              const sc = event as SignalClassifiedEvent;
              callbacks.onSignalClassified(
                sc.mode,
                sc.confidence,
                sc.genre,
                sc.docType,
                sc.weight,
              );
              break;
            }

            case "artifact_start": {
              if (!artifactStarted) {
                artifactStarted = true;
                inArtifactBlock = true;
                displayContent = fullContent;
                const title = event.title ?? "";
                const artifactType = event.artifactType ?? "document";
                callbacks.onArtifactStart(title, artifactType);
              }
              break;
            }

            case "artifact_complete": {
              const art = event;
              if (art.title && art.artifactType && art.content) {
                const processedContent = art.content
                  .replace(/\\n/g, "\n")
                  .replace(/\\"/g, '"')
                  .replace(/\\\\/g, "\\");
                const artifact: ArtifactPayload = {
                  title: art.title,
                  type: art.artifactType,
                  content: processedContent,
                };
                inArtifactBlock = false;
                artifactCompleted = true;
                callbacks.onArtifactComplete(artifact);
              }
              break;
            }

            case "error": {
              const errorMsg = event.message || "Unknown error";
              console.error("[SSE] Stream error event:", errorMsg);
              // Surface error to user if no content was generated
              if (!fullContent) {
                fullContent = `Sorry, an error occurred: ${errorMsg}`;
              }
              break;
            }

            case "done":
              break;
          }

          // Legacy ```artifact block tracking
          if (!artifactStarted && fullContent.includes("```artifact")) {
            artifactStarted = true;
            inArtifactBlock = true;
            displayContent = fullContent.split("```artifact")[0];
            callbacks.onArtifactStart("", "document");
          }

          if (inArtifactBlock && !artifactCompleted) {
            const afterStart = fullContent.slice(
              fullContent.indexOf("```artifact"),
            );
            const backtickMatches = afterStart.match(/```/g);
            if (backtickMatches && backtickMatches.length >= 2) {
              inArtifactBlock = false;
              artifactCompleted = true;
              const afterArtifact = fullContent.slice(
                fullContent.indexOf("```artifact"),
              );
              const closingIdx = afterArtifact.indexOf(
                "```",
                afterArtifact.indexOf("\n"),
              );
              const afterClosing = afterArtifact.slice(closingIdx + 3);
              displayContent = fullContent.split("```artifact")[0].trim();
              if (afterClosing.trim())
                displayContent += "\n\n" + afterClosing.trim();
              try {
                const artifactMatch = fullContent.match(
                  /```artifact\s*\n([\s\S]*?)\n```/,
                );
                if (artifactMatch) {
                  const artifactData = JSON.parse(artifactMatch[1].trim());
                  if (
                    artifactData.title &&
                    artifactData.type &&
                    artifactData.content
                  ) {
                    const processedContent = artifactData.content
                      .replace(/\\n/g, "\n")
                      .replace(/\\"/g, '"')
                      .replace(/\\\\/g, "\\");
                    callbacks.onArtifactComplete({
                      title: artifactData.title,
                      type: artifactData.type,
                      content: processedContent,
                    });
                  }
                }
              } catch {
                // Malformed artifact JSON — ignore
              }
            }
          }

          if (artifactStarted && !artifactCompleted) {
            const contentMatch = fullContent.match(
              /"content":\s*"([\s\S]*?)(?:"\s*}|$)/,
            );
            if (contentMatch) {
              const generatingContent = contentMatch[1]
                .replace(/\\n/g, "\n")
                .replace(/\\"/g, '"')
                .replace(/\\\\/g, "\\");
              callbacks.onArtifactContentUpdate(generatingContent);
            }
          }

          updateMessage();
        }
      } else {
        // ── Plain-text fallback path ───────────────────────────────────────
        const decoder = new TextDecoder();
        if (!firstRead.done) {
          fullContent += decoder.decode(firstRead.value, { stream: true });
        }
        while (true) {
          const { done, value } = await rawReader.read();
          if (done) break;
          fullContent += decoder.decode(value, { stream: true });
          callbacks.updateAssistantContent(assistantMsgId, fullContent);
        }
      }

      // ── Post-stream: parse usage comment ─────────────────────────────────
      const usageRegex = /<!--USAGE:(\{[^}]+\})-->/;
      const usageMatch = fullContent.match(usageRegex);
      let usageData: UsageData | undefined;
      if (usageMatch) {
        try {
          usageData = JSON.parse(usageMatch[1]);
          fullContent = fullContent.replace(usageRegex, "").trim();
          displayContent = displayContent.replace(usageRegex, "").trim();
        } catch (e) {
          console.error("Failed to parse usage data:", e);
        }
      }

      const finalContent = artifactStarted ? displayContent : fullContent;
      callbacks.updateAssistantContent(
        assistantMsgId,
        finalContent,
        undefined,
        usageData,
      );

      // ── Artifact finalization ─────────────────────────────────────────────
      if (fullContent.includes("```artifact")) {
        const artifactMatch = fullContent.match(
          /```artifact\s*\n([\s\S]*?)\n```/,
        );
        if (artifactMatch) {
          try {
            const artifactData = JSON.parse(artifactMatch[1].trim());
            if (
              artifactData.title &&
              artifactData.type &&
              artifactData.content
            ) {
              callbacks.onArtifactFinalize({
                title: artifactData.title,
                type: artifactData.type,
                content: artifactData.content
                  .replace(/\\n/g, "\n")
                  .replace(/\\"/g, '"')
                  .replace(/\\\\/g, "\\"),
              });
            }
          } catch {
            // Ignore
          }
        }
      }

      callbacks.onModelWarmedUp(params.model);
    } catch (error: any) {
      clearTimeout(connectionTimeoutId);
      if (error.name === "AbortError") {
        if (this._userAborted) {
          // User-initiated abort — no action needed
        } else {
          // Timeout abort — backend didn't respond within 30 seconds
          callbacks.onStreamError(
            "Response timed out. The server may be busy — please try again.",
          );
          callbacks.updateAssistantContent(assistantMsgId, "");
        }
      } else {
        const errorMsg =
          error instanceof Error
            ? error.message
            : "An error occurred while processing your request";
        console.error("[ChatStreamManager] Stream error:", errorMsg, error);
        callbacks.onStreamError(errorMsg);
        callbacks.updateAssistantContent(
          assistantMsgId,
          `Sorry, there was an error: ${errorMsg || "Unknown error"}. Please try again.`,
        );
      }
    } finally {
      callbacks.onStreamEnd();
      this.abortController = null;
    }
  }

  // ── Private helpers ────────────────────────────────────────────────────────

  private _buildRequestBody(params: SendParams): Record<string, unknown> {
    const body: Record<string, unknown> = {
      message: params.message,
      model: params.model,
      conversation_id: params.conversationId,
      project_id: params.projectId,
      workspace_id: params.workspaceId,
      context_id: params.contextIds.length > 0 ? params.contextIds[0] : null,
      context_ids: params.contextIds.length > 0 ? params.contextIds : undefined,
      command: params.command,
      temperature: params.temperature,
      max_tokens: params.maxTokens,
      top_p: params.topP,
      focus_mode: params.focusMode,
      focus_options:
        Object.keys(params.focusOptions).length > 0
          ? params.focusOptions
          : undefined,
      structured_output: true,
      agent_id: params.agentId || undefined,
      memory_ids: params.memoryIds.length > 0 ? params.memoryIds : undefined,
    };

    if (params.documentIds.length > 0) {
      body.document_ids = params.documentIds;
    }

    if (params.nodeContext) {
      body.node_context = params.nodeContext;
    }

    if (params.useCOT) {
      body.use_cot = true;
    }

    return body;
  }

  private async _buildHeaders(): Promise<Record<string, string>> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...getActiveWorkspaceHeaders(),
    };

    let csrfToken = getCSRFToken();
    if (!csrfToken) {
      await initCSRF();
      csrfToken = getCSRFToken();
    }

    if (csrfToken) {
      headers["X-CSRF-Token"] = csrfToken;
    }

    return headers;
  }
}
