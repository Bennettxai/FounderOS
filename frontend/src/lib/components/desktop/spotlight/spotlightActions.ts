/**
 * spotlightActions.ts
 * Action handlers: opening apps, file processing, chat sending.
 */

import { windowStore } from "$lib/stores/windowStore";
import { getActiveWorkspaceHeaders, getApiBaseUrl } from "$lib/api/base";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AttachedFile {
  id: string;
  name: string;
  type: string;
  size: number;
  preview?: string;
  file: File;
}

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

export interface SendMessageOptions {
  message: string;
  model?: string;
  projectId: string;
  conversationId?: string;
}

export interface SendMessageResult {
  conversationId: string | null;
  error?: string;
}

// ---------------------------------------------------------------------------
// App navigation
// ---------------------------------------------------------------------------

export function openApp(appId: string, onClose: () => void): void {
  windowStore.openWindow(appId);
  onClose();
}

export function openInChat(
  conversationId: string | null,
  messages: ChatMessage[],
  projectId: string | null,
  onClose: () => void,
): void {
  if (conversationId || messages.length > 0) {
    const chatData = {
      conversationId,
      messages: messages.map((m) => ({ role: m.role, content: m.content })),
      projectId,
    };
    sessionStorage.setItem("spotlightChatTransfer", JSON.stringify(chatData));
  }
  windowStore.openWindow("chat");
  onClose();
}

// ---------------------------------------------------------------------------
// File utilities
// ---------------------------------------------------------------------------

export function getFileIcon(type: string): string {
  if (type.startsWith("image/")) return "🖼️";
  if (type.startsWith("video/")) return "🎬";
  if (type.startsWith("audio/")) return "🎵";
  if (type.includes("pdf")) return "📄";
  if (type.includes("word") || type.includes("document")) return "📝";
  if (type.includes("sheet") || type.includes("excel")) return "📊";
  if (type.includes("zip") || type.includes("archive")) return "📦";
  return "📎";
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

const MAX_FILE_BYTES = 10 * 1024 * 1024; // 10 MB

export function processFiles(
  files: File[],
  existing: AttachedFile[],
  onUpdate: (updated: AttachedFile[]) => void,
): void {
  // Snapshot of existing — shared across all async image loads in this batch
  const batch = [...existing];
  let syncAdded = false;

  for (const file of files) {
    if (file.size > MAX_FILE_BYTES) continue;

    const entry: AttachedFile = {
      id: crypto.randomUUID(),
      name: file.name,
      type: file.type,
      size: file.size,
      file,
    };

    if (file.type.startsWith("image/")) {
      // Image files: generate preview asynchronously, then notify
      const reader = new FileReader();
      reader.onload = () => {
        entry.preview = reader.result as string;
        batch.push(entry);
        onUpdate([...batch]);
      };
      reader.readAsDataURL(file);
    } else {
      // Non-image files: add synchronously
      batch.push(entry);
      syncAdded = true;
    }
  }

  // Notify once for all synchronous (non-image) additions
  if (syncAdded) {
    onUpdate([...batch]);
  }
}

// ---------------------------------------------------------------------------
// Chat
// ---------------------------------------------------------------------------

export async function sendChatMessage(
  opts: SendMessageOptions,
  onChunk: (fullContent: string) => void,
): Promise<SendMessageResult> {
  try {
    const response = await fetch(`${getApiBaseUrl()}/chat/message`, {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...getActiveWorkspaceHeaders(),
      },
      body: JSON.stringify({
        message: opts.message,
        model: opts.model || undefined,
        project_id: opts.projectId,
        conversation_id: opts.conversationId || undefined,
      }),
    });

    const newConvId = response.headers.get("X-Conversation-Id");

    if (response.ok) {
      const reader = response.body?.getReader();
      const decoder = new TextDecoder();
      let fullContent = "";

      if (reader) {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          fullContent += decoder.decode(value, { stream: true });
          onChunk(fullContent);
        }
      }

      return { conversationId: newConvId };
    }

    return {
      conversationId: newConvId,
      error: "Sorry, I encountered an error. Please try again.",
    };
  } catch {
    return {
      conversationId: null,
      error: "Connection error. Please check your network.",
    };
  }
}
