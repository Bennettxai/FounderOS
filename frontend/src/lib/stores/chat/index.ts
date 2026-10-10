/**
 * Chat Stores — Barrel Export
 * 6 domain stores extracted from chat/+page.svelte
 */

export { chatUIStore } from "./chatUIStore.svelte";
export { chatModelStore } from "./chatModelStore.svelte";
export { chatContextStore } from "./chatContextStore.svelte";
export { chatAgentStore } from "./chatAgentStore.svelte";
export { chatArtifactStore } from "./chatArtifactStore.svelte";
export { chatConversationStore } from "./chatConversationStore.svelte";

// Re-export all types
export type {
  ChatMessage,
  AttachedFile,
  FocusModeFile,
  SlashCommand,
  AgentPreset,
  SidebarConversation,
  ProjectItem,
  GeneratedTask,
  CombinedArtifact,
  TeamMember,
  UsageData,
  StreamingToolCall,
  ArtifactPayload,
  SendParams,
  StreamCallbacks,
} from "./types";
