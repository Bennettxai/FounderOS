/**
 * Chat Store Types
 * Shared interfaces extracted from chat/+page.svelte.
 * Types already defined in ChatStreamManager.ts are re-exported from there.
 */

import type {
  ArtifactListItem,
  Artifact,
  Node,
  ContextListItem,
  CustomAgent,
} from "$lib/api";
import type { DelegatedTask } from "$lib/components/chat/panels/ProgressPanel.svelte";
import type { ActiveResource } from "$lib/components/chat/panels/ContextPanel.svelte";

// Re-export types that live in ChatStreamManager
export type {
  UsageData,
  StreamingToolCall,
  ArtifactPayload,
  SendParams,
  StreamCallbacks,
} from "./ChatStreamManager";

// Also re-export imported types for convenience
export type {
  ArtifactListItem,
  Artifact,
  Node,
  ContextListItem,
  CustomAgent,
  DelegatedTask,
  ActiveResource,
};

// ── Chat Messages ───────────────────────────────────────────────────────────

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  artifacts?: { title: string; type: string; content: string }[];
  usage?: {
    input_tokens: number;
    output_tokens: number;
    thinking_tokens: number;
    total_tokens: number;
    duration_ms: number;
    tps: number;
    provider: string;
    model: string;
    estimated_cost: number;
  };
}

// ── File Attachments ────────────────────────────────────────────────────────

export interface AttachedFile {
  id: string;
  name: string;
  type: string;
  size: number;
  content?: string;
  documentId?: string;
  uploading?: boolean;
  uploadError?: string;
}

export interface FocusModeFile {
  id: string;
  name: string;
  type: string;
  size: number;
  content?: string;
}

// ── Slash Commands ──────────────────────────────────────────────────────────

export interface SlashCommand {
  name: string;
  display_name: string;
  description: string;
  icon: string;
  category: string;
}

// ── Agent Presets ───────────────────────────────────────────────────────────

export interface AgentPreset {
  id: string;
  name: string;
  display_name: string;
  description: string | null;
  avatar: string | null;
  category: string | null;
}

// ── Sidebar Conversations ───────────────────────────────────────────────────

export interface SidebarConversation {
  id: string;
  title: string;
  timestamp: string;
  preview?: string;
  pinned?: boolean;
  projectId?: string;
  projectName?: string;
  messageCount?: number;
  conversationType?: "chat" | "focus";
  isArchived?: boolean;
}

// ── Projects ────────────────────────────────────────────────────────────────

export interface ProjectItem {
  id: string;
  name: string;
  description?: string;
}

// ── Task Generation ─────────────────────────────────────────────────────────

export interface GeneratedTask {
  title: string;
  description: string;
  priority: "low" | "medium" | "high";
  assignee_id?: string;
  estimated_hours?: number;
}

// ── Combined Artifacts ──────────────────────────────────────────────────────

export interface CombinedArtifact {
  id: string;
  title: string;
  type: string;
  content?: string;
  summary?: string | null;
  version?: number;
  fromMessage?: boolean;
  messageId?: string;
  context_name?: string | null;
  project_id?: string | null;
  context_id?: string | null;
  conversation_id?: string | null;
  created_at?: string;
  updated_at?: string;
}

// ── Team Members ────────────────────────────────────────────────────────────

export interface TeamMember {
  id: string;
  name: string;
  role: string;
}

// ── Model Options (from chatActions) ────────────────────────────────────────

export type {
  ModelOption,
  ModelCapability,
  ParsedPart,
} from "./chatActions";
