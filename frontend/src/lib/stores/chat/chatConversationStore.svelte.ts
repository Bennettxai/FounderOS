/**
 * Chat Conversation Store
 * Manages all conversation-layer state for the chat page: message lists,
 * conversation selection, pagination, archive/pin operations, and streaming.
 *
 * Follows the singleton factory pattern from chatUIStore.svelte.ts.
 * Uses Svelte 5 $state runes for fine-grained reactivity.
 *
 * State that must be mutated from outside the store exposes both a getter
 * and a setter. Pure-internal helpers are kept private to the factory.
 */

import { api, apiClient } from "$lib/api";
import { handleApiCall } from "$lib/utils/api-handler";
import { parseArtifactsFromContent } from "./chatActions";
import type { ChatMessage, SidebarConversation } from "./types";

// ── Private clipboard helper ───────────────────────────────────────────────

async function copyToClipboard(text: string): Promise<void> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return;
    }
  } catch {
    // fall through to textarea fallback
  }
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.top = "0";
  textarea.style.left = "0";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  try {
    document.execCommand("copy");
  } finally {
    document.body.removeChild(textarea);
  }
}

// ── Store factory ──────────────────────────────────────────────────────────

function createChatConversationStore() {
  // ── Core message / conversation state ────────────────────────────────────
  let messages = $state<ChatMessage[]>([]);
  let conversationId = $state<string | null>(null);
  let activeConversationId = $state<string | null>(null);
  let isStreaming = $state(false);
  let abortController = $state<AbortController | null>(null);
  let loadingConversation = $state(false);

  // ── Sidebar / list state ─────────────────────────────────────────────────
  let conversations = $state<SidebarConversation[]>([]);
  let archivedConversations = $state<SidebarConversation[]>([]);
  let showArchivedView = $state(false);
  let conversationsPage = $state(1);
  let conversationsPageSize = $state(10);
  let conversationsTotal = $state(0);

  // ── Input / interaction state ─────────────────────────────────────────────
  let inputValue = $state("");
  let copiedMessageId = $state<string | null>(null);
  let filterTab = $state<"all" | "pinned" | "recent">("all");

  // ── Derived ───────────────────────────────────────────────────────────────
  const hasConversation = $derived(messages.length > 0 || loadingConversation);

  // ── Public interface ───────────────────────────────────────────────────────

  return {
    // ── Getters / setters: messages ────────────────────────────────────────
    get messages() {
      return messages;
    },
    set messages(v: ChatMessage[]) {
      messages = v;
    },

    // ── Getters / setters: conversation identity ───────────────────────────
    get conversationId() {
      return conversationId;
    },
    set conversationId(v: string | null) {
      conversationId = v;
    },

    get activeConversationId() {
      return activeConversationId;
    },
    set activeConversationId(v: string | null) {
      activeConversationId = v;
    },

    // ── Getters / setters: streaming ───────────────────────────────────────
    get isStreaming() {
      return isStreaming;
    },
    set isStreaming(v: boolean) {
      isStreaming = v;
    },

    get abortController() {
      return abortController;
    },
    set abortController(v: AbortController | null) {
      abortController = v;
    },

    // ── Getters / setters: loading ────────────────────────────────────────
    get loadingConversation() {
      return loadingConversation;
    },
    set loadingConversation(v: boolean) {
      loadingConversation = v;
    },

    // ── Getters / setters: sidebar lists ─────────────────────────────────
    get conversations() {
      return conversations;
    },
    set conversations(v: SidebarConversation[]) {
      conversations = v;
    },

    get archivedConversations() {
      return archivedConversations;
    },
    set archivedConversations(v: SidebarConversation[]) {
      archivedConversations = v;
    },

    get showArchivedView() {
      return showArchivedView;
    },
    set showArchivedView(v: boolean) {
      showArchivedView = v;
    },

    // ── Getters / setters: pagination ─────────────────────────────────────
    get conversationsPage() {
      return conversationsPage;
    },
    set conversationsPage(v: number) {
      conversationsPage = v;
    },

    get conversationsPageSize() {
      return conversationsPageSize;
    },
    set conversationsPageSize(v: number) {
      conversationsPageSize = v;
    },

    get conversationsTotal() {
      return conversationsTotal;
    },
    set conversationsTotal(v: number) {
      conversationsTotal = v;
    },

    // ── Getters / setters: input / interaction ────────────────────────────
    get inputValue() {
      return inputValue;
    },
    set inputValue(v: string) {
      inputValue = v;
    },

    get copiedMessageId() {
      return copiedMessageId;
    },
    set copiedMessageId(v: string | null) {
      copiedMessageId = v;
    },

    get filterTab() {
      return filterTab;
    },
    set filterTab(v: "all" | "pinned" | "recent") {
      filterTab = v;
    },

    // ── Derived ────────────────────────────────────────────────────────────
    get hasConversation() {
      return hasConversation;
    },

    // ── Methods ────────────────────────────────────────────────────────────

    /**
     * Load the paginated conversation list from the API.
     * Splits archived and non-archived into separate lists.
     */
    async loadConversations(): Promise<void> {
      const { data } = await handleApiCall(
        () => api.getConversations(conversationsPage, conversationsPageSize),
        { showErrorToast: true, errorMessage: "Failed to load conversations" },
      );

      if (!data) return;

      conversationsTotal = data.total;

      const active: SidebarConversation[] = [];
      const archived: SidebarConversation[] = [];

      for (const c of data.conversations) {
        const sidebar: SidebarConversation = {
          id: c.id,
          title: c.title || "Untitled",
          timestamp: c.updated_at ?? c.created_at,
          preview: c.preview,
          pinned: c.pinned ?? false,
          projectId: c.project_id ?? undefined,
          messageCount: c.message_count,
          conversationType: c.conversation_type,
          isArchived: c.is_archived ?? false,
        };
        if (c.is_archived) {
          archived.push(sidebar);
        } else {
          active.push(sidebar);
        }
      }

      conversations = active;
      archivedConversations = archived;
    },

    /**
     * Navigate to a specific page in the conversation list and reload.
     */
    async handleConversationPageChange(newPage: number): Promise<void> {
      conversationsPage = newPage;
      await this.loadConversations();
    },

    /**
     * Load all messages for a conversation and set it as the active conversation.
     * Assistant message content is parsed for artifact blocks.
     */
    async selectConversation(id: string): Promise<void> {
      loadingConversation = true;
      activeConversationId = id;
      conversationId = id;

      const { data } = await handleApiCall(() => api.getConversation(id), {
        showErrorToast: true,
        errorMessage: "Failed to load conversation",
      });

      loadingConversation = false;

      if (!data) return;

      messages = (data.messages ?? []).map((m) => {
        if (m.role === "assistant") {
          const { cleanContent, artifacts } = parseArtifactsFromContent(
            m.content ?? "",
          );
          const chatMessage: ChatMessage = {
            id: m.id,
            role: "assistant",
            content: cleanContent,
            artifacts: artifacts.length > 0 ? artifacts : undefined,
          };

          // Map usage if available
          if (m.usage) {
            chatMessage.usage = {
              input_tokens: m.usage.prompt_tokens,
              output_tokens: m.usage.completion_tokens,
              thinking_tokens: m.usage.thinking_tokens ?? 0,
              total_tokens: m.usage.total_tokens,
              duration_ms: 0,
              tps: m.usage.tps ?? 0,
              provider: m.usage.provider ?? "",
              model: m.usage.model ?? "",
              estimated_cost: 0,
            };
          }

          return chatMessage;
        }

        return {
          id: m.id,
          role: m.role as "user" | "assistant",
          content: m.content ?? "",
        } satisfies ChatMessage;
      });
    },

    /**
     * Clear the current conversation and return to the empty state.
     * Does not affect the conversations list.
     */
    handleNewChat(): void {
      messages = [];
      conversationId = null;
      activeConversationId = null;
    },

    /**
     * Start a fresh conversation, resetting all transient state including the
     * input field and any streaming in progress.
     */
    startNewConversation(): void {
      messages = [];
      conversationId = null;
      activeConversationId = null;
      inputValue = "";
      isStreaming = false;
      if (abortController) {
        abortController.abort();
        abortController = null;
      }
    },

    /**
     * Archive a conversation by id.
     * Moves it from the active list to the archived list.
     * If it is the current conversation, clears conversation state.
     */
    handleArchiveConversation(id: string): void {
      const idx = conversations.findIndex((c) => c.id === id);
      if (idx === -1) return;

      const [target] = conversations.splice(idx, 1);
      conversations = [...conversations];
      archivedConversations = [
        { ...target, isArchived: true },
        ...archivedConversations,
      ];

      if (activeConversationId === id) {
        messages = [];
        conversationId = null;
        activeConversationId = null;
      }
    },

    /**
     * Unarchive a conversation by id.
     * Moves it from the archived list back to the active list.
     */
    handleUnarchiveConversation(id: string): void {
      const idx = archivedConversations.findIndex((c) => c.id === id);
      if (idx === -1) return;

      const [target] = archivedConversations.splice(idx, 1);
      archivedConversations = [...archivedConversations];
      conversations = [{ ...target, isArchived: false }, ...conversations];
    },

    /**
     * Toggle the pinned flag on a conversation.
     * Updates both the active and archived lists so the toggle works regardless
     * of which view is currently showing.
     */
    handlePinConversation(id: string): void {
      conversations = conversations.map((c) =>
        c.id === id ? { ...c, pinned: !c.pinned } : c,
      );
      archivedConversations = archivedConversations.map((c) =>
        c.id === id ? { ...c, pinned: !c.pinned } : c,
      );
    },

    /**
     * Prompt the user for a new title and update the conversation in the API
     * and in the local lists.
     */
    handleRenameConversation(id: string): void {
      const current =
        conversations.find((c) => c.id === id) ??
        archivedConversations.find((c) => c.id === id);

      const newTitle = window.prompt(
        "Rename conversation",
        current?.title ?? "",
      );
      if (!newTitle || !newTitle.trim()) return;

      const trimmed = newTitle.trim();

      // Optimistic update
      conversations = conversations.map((c) =>
        c.id === id ? { ...c, title: trimmed } : c,
      );
      archivedConversations = archivedConversations.map((c) =>
        c.id === id ? { ...c, title: trimmed } : c,
      );

      // Persist to API — fire and forget, toast on error
      handleApiCall(() => api.updateConversation(id, { title: trimmed }), {
        showErrorToast: true,
        errorMessage: "Failed to rename conversation",
      });
    },

    /** Stub: export a conversation. */
    handleExportConversation(_id: string): void {
      // TODO: implement export (PDF / Markdown download)
    },

    /** Stub: link a conversation to a project. */
    handleLinkProject(_id: string): void {
      // TODO: implement project linking modal
    },

    /**
     * Confirm with the user and permanently delete a conversation.
     * Removes it from both local lists.
     */
    async handleDeleteConversation(id: string): Promise<void> {
      const confirmed = window.confirm(
        "Are you sure you want to delete this conversation? This cannot be undone.",
      );
      if (!confirmed) return;

      const { error } = await handleApiCall(
        () => apiClient.delete(`/chat/conversations/${id}`),
        { showErrorToast: true, errorMessage: "Failed to delete conversation" },
      );

      if (error) return;

      conversations = conversations.filter((c) => c.id !== id);
      archivedConversations = archivedConversations.filter((c) => c.id !== id);

      if (activeConversationId === id) {
        messages = [];
        conversationId = null;
        activeConversationId = null;
      }
    },

    /** Switch the sidebar to the archived conversations view. */
    handleViewArchived(): void {
      showArchivedView = true;
    },

    /** Switch the sidebar back to the active conversations view. */
    handleBackToChats(): void {
      showArchivedView = false;
    },

    /**
     * Append a user message to the message list.
     * Uses crypto.randomUUID() for a collision-free local id.
     */
    addUserMessage(content: string): ChatMessage {
      const msg: ChatMessage = {
        id: crypto.randomUUID(),
        role: "user",
        content,
      };
      messages = [...messages, msg];
      return msg;
    },

    /**
     * Append an empty assistant placeholder to the message list.
     * Returns the generated id so callers can update the message later via
     * `updateAssistantContent`.
     */
    addAssistantPlaceholder(): string {
      const id = crypto.randomUUID();
      const placeholder: ChatMessage = {
        id,
        role: "assistant",
        content: "",
      };
      messages = [...messages, placeholder];
      return id;
    },

    /**
     * Update the content (and optionally artifacts and usage) of an existing
     * assistant message identified by `id`.
     */
    updateAssistantContent(
      id: string,
      content: string,
      artifacts?: { title: string; type: string; content: string }[],
      usage?: ChatMessage["usage"],
    ): void {
      messages = messages.map((m: ChatMessage) => {
        if (m.id !== id) return m;
        return {
          ...m,
          content,
          ...(artifacts !== undefined ? { artifacts } : {}),
          ...(usage !== undefined ? { usage } : {}),
        };
      });
    },

    /**
     * Record a newly created conversation id and title returned from the API.
     * Prepends the conversation to the active sidebar list.
     */
    onConversationCreated(id: string, title: string): void {
      conversationId = id;
      activeConversationId = id;

      const entry: SidebarConversation = {
        id,
        title: title || "New conversation",
        timestamp: new Date().toISOString(),
        pinned: false,
        isArchived: false,
      };

      conversations = [entry, ...conversations];
    },

    /**
     * Copy a message's content to the clipboard and briefly mark the message
     * as copied (2 s) so the UI can show a confirmation state.
     */
    async copyMessage(content: string, id: string): Promise<void> {
      await copyToClipboard(content);
      copiedMessageId = id;
      setTimeout(() => {
        copiedMessageId = null;
      }, 2000);
    },

    /**
     * Abort any in-flight streaming request.
     */
    handleStop(): void {
      if (abortController) {
        abortController.abort();
        abortController = null;
      }
      isStreaming = false;
    },
  };
}

export const chatConversationStore = createChatConversationStore();
