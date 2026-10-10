/**
 * Chat UI Store
 * Manages all UI-layer state for the chat page: panel visibility, resize
 * handles, greeting/typewriter animation, and the inline project picker.
 *
 * Follows the singleton factory pattern from cameraStore.svelte.ts.
 * Uses Svelte 5 $state runes for fine-grained reactivity.
 *
 * State that must be mutated from outside the store (page + child components)
 * exposes both a getter and a setter. Pure-internal resize bookkeeping only
 * needs a getter because mutation goes through the resize methods.
 */

import type { DelegatedTask } from "$lib/components/chat/panels/ProgressPanel.svelte";
import type { ActiveResource } from "$lib/components/chat/panels/ContextPanel.svelte";

const LOCAL_STORAGE_KEY = "chat_panel_state";

function createChatUIStore() {
  // ── Sidebar ────────────────────────────────────────────────────────────────
  let chatSidebarOpen = $state(false);
  let searchQuery = $state("");

  // ── Right panel (Progress / Context / Artifacts) ──────────────────────────
  let rightPanelOpen = $state(false);
  let rightPanelTab = $state<"progress" | "context" | "artifacts" | "recall">(
    "progress",
  );
  let rightPanelWidth = $state(320);

  // Right panel resize bookkeeping (internal — only mutated by resize methods)
  let isResizingRightPanel = $state(false);
  let rightPanelResizeStartX = $state(0);
  let rightPanelResizeStartWidth = $state(0);

  // ── Artifact panel (left split-view) ──────────────────────────────────────
  let artifactPanelWidth = $state(400);

  // Artifact panel resize bookkeeping (internal — only mutated by resize methods)
  let isResizing = $state(false);
  let resizeStartX = $state(0);
  let resizeStartWidth = $state(0);

  // ── Data lists ────────────────────────────────────────────────────────────
  let delegatedTasks = $state<DelegatedTask[]>([]);
  let activeResources = $state<ActiveResource[]>([]);

  // ── Inline project picker ─────────────────────────────────────────────────
  let showInlineProjectPicker = $state(false);

  // ── Greeting / typewriter ─────────────────────────────────────────────────
  let userName = $state("there");
  let currentSuggestionIndex = $state(0);
  let displayedSuggestion = $state("");
  let isTyping = $state(true);
  let typewriterPaused = $state(false);

  // ── Internal helpers ───────────────────────────────────────────────────────

  function getTimeBasedGreeting(): string {
    const hour = new Date().getHours();
    if (hour >= 0 && hour < 5) {
      return `Up late, ${userName}?`;
    } else if (hour >= 5 && hour < 12) {
      return `Good morning, ${userName}`;
    } else if (hour >= 12 && hour < 17) {
      return `Good afternoon, ${userName}`;
    } else if (hour >= 17 && hour < 21) {
      return `Good evening, ${userName}`;
    } else {
      return `Working late, ${userName}?`;
    }
  }

  // ── Resize callbacks (captured so removeEventListener works correctly) ─────

  function handleResize(e: MouseEvent) {
    if (!isResizing) return;
    const delta = resizeStartX - e.clientX;
    artifactPanelWidth = Math.min(Math.max(resizeStartWidth + delta, 300), 800);
  }

  function stopResize() {
    isResizing = false;
    document.removeEventListener("mousemove", handleResize);
    document.removeEventListener("mouseup", stopResize);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  }

  function handleRightPanelResize(e: MouseEvent) {
    if (!isResizingRightPanel) return;
    const delta = rightPanelResizeStartX - e.clientX;
    rightPanelWidth = Math.min(
      Math.max(rightPanelResizeStartWidth + delta, 280),
      500,
    );
  }

  function stopRightPanelResize() {
    isResizingRightPanel = false;
    document.removeEventListener("mousemove", handleRightPanelResize);
    document.removeEventListener("mouseup", stopRightPanelResize);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  }

  // ── Public interface ───────────────────────────────────────────────────────

  return {
    // Sidebar
    get chatSidebarOpen() {
      return chatSidebarOpen;
    },
    set chatSidebarOpen(v: boolean) {
      chatSidebarOpen = v;
    },

    get searchQuery() {
      return searchQuery;
    },
    set searchQuery(v: string) {
      searchQuery = v;
    },

    // Right panel
    get rightPanelOpen() {
      return rightPanelOpen;
    },
    set rightPanelOpen(v: boolean) {
      rightPanelOpen = v;
    },

    get rightPanelTab() {
      return rightPanelTab;
    },
    set rightPanelTab(v: "progress" | "context" | "artifacts" | "recall") {
      rightPanelTab = v;
    },

    get rightPanelWidth() {
      return rightPanelWidth;
    },
    set rightPanelWidth(v: number) {
      rightPanelWidth = v;
    },

    get isResizingRightPanel() {
      return isResizingRightPanel;
    },
    get rightPanelResizeStartX() {
      return rightPanelResizeStartX;
    },
    get rightPanelResizeStartWidth() {
      return rightPanelResizeStartWidth;
    },

    // Artifact panel
    get artifactPanelWidth() {
      return artifactPanelWidth;
    },
    set artifactPanelWidth(v: number) {
      artifactPanelWidth = v;
    },

    get isResizing() {
      return isResizing;
    },
    get resizeStartX() {
      return resizeStartX;
    },
    get resizeStartWidth() {
      return resizeStartWidth;
    },

    // Data lists
    get delegatedTasks() {
      return delegatedTasks;
    },
    set delegatedTasks(v: DelegatedTask[]) {
      delegatedTasks = v;
    },

    get activeResources() {
      return activeResources;
    },
    set activeResources(v: ActiveResource[]) {
      activeResources = v;
    },

    // Inline project picker
    get showInlineProjectPicker() {
      return showInlineProjectPicker;
    },
    set showInlineProjectPicker(v: boolean) {
      showInlineProjectPicker = v;
    },

    // Greeting / typewriter
    get userName() {
      return userName;
    },
    set userName(v: string) {
      userName = v;
    },

    get currentSuggestionIndex() {
      return currentSuggestionIndex;
    },
    set currentSuggestionIndex(v: number) {
      currentSuggestionIndex = v;
    },

    get displayedSuggestion() {
      return displayedSuggestion;
    },
    set displayedSuggestion(v: string) {
      displayedSuggestion = v;
    },

    get isTyping() {
      return isTyping;
    },
    set isTyping(v: boolean) {
      isTyping = v;
    },

    get typewriterPaused() {
      return typewriterPaused;
    },
    set typewriterPaused(v: boolean) {
      typewriterPaused = v;
    },

    // Derived
    get personalizedGreeting() {
      return getTimeBasedGreeting();
    },

    // ── Methods ───────────────────────────────────────────────────────────

    /**
     * Load persisted panel state from localStorage.
     * Call this once on mount — guards against SSR with typeof window check.
     */
    initFromLocalStorage() {
      if (typeof window === "undefined") return;
      try {
        const raw = localStorage.getItem(LOCAL_STORAGE_KEY);
        if (!raw) return;
        const parsed = JSON.parse(raw) as Record<string, unknown>;
        if (typeof parsed.rightPanelOpen === "boolean") {
          rightPanelOpen = parsed.rightPanelOpen;
        }
        if (
          parsed.rightPanelTab === "progress" ||
          parsed.rightPanelTab === "context" ||
          parsed.rightPanelTab === "artifacts" ||
          parsed.rightPanelTab === "recall"
        ) {
          rightPanelTab = parsed.rightPanelTab;
        }
        if (
          typeof parsed.rightPanelWidth === "number" &&
          parsed.rightPanelWidth >= 280 &&
          parsed.rightPanelWidth <= 500
        ) {
          rightPanelWidth = parsed.rightPanelWidth;
        }
      } catch {
        // Silently ignore malformed localStorage data
      }
    },

    /**
     * Persist current panel state to localStorage.
     * Call inside a $effect that tracks rightPanelOpen, rightPanelTab,
     * and rightPanelWidth so saves happen reactively on change.
     */
    saveToLocalStorage() {
      if (typeof window === "undefined") return;
      localStorage.setItem(
        LOCAL_STORAGE_KEY,
        JSON.stringify({ rightPanelOpen, rightPanelTab, rightPanelWidth }),
      );
    },

    /** Toggle the conversation list sidebar. */
    toggleSidebar() {
      chatSidebarOpen = !chatSidebarOpen;
    },

    /** Switch right panel to the artifacts tab and ensure it is visible. */
    openArtifactPanel() {
      rightPanelOpen = true;
      rightPanelTab = "artifacts";
    },

    // ── Artifact panel resize ─────────────────────────────────────────────

    /**
     * Begin a drag-resize on the left artifact split-panel.
     * Attach to the resize handle's mousedown event.
     */
    startResize(e: MouseEvent) {
      isResizing = true;
      resizeStartX = e.clientX;
      resizeStartWidth = artifactPanelWidth;
      document.addEventListener("mousemove", handleResize);
      document.addEventListener("mouseup", stopResize);
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },

    /**
     * Update artifactPanelWidth during an active drag.
     * Exposed for callers that manage their own event listeners,
     * but the internal listener registered by startResize is preferred.
     */
    handleResize,

    /**
     * End an artifact panel resize and clean up document listeners.
     * Exposed for callers that manage their own event listeners.
     */
    stopResize,

    // ── Right panel resize ────────────────────────────────────────────────

    /**
     * Begin a drag-resize on the right panel.
     * Attach to the resize handle's mousedown event.
     */
    startRightPanelResize(e: MouseEvent) {
      isResizingRightPanel = true;
      rightPanelResizeStartX = e.clientX;
      rightPanelResizeStartWidth = rightPanelWidth;
      document.addEventListener("mousemove", handleRightPanelResize);
      document.addEventListener("mouseup", stopRightPanelResize);
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },

    /**
     * Update rightPanelWidth during an active drag (clamped 280–500 px).
     * Exposed for callers that manage their own event listeners.
     */
    handleRightPanelResize,

    /**
     * End a right panel resize and clean up document listeners.
     * Exposed for callers that manage their own event listeners.
     */
    stopRightPanelResize,
  };
}

export const chatUIStore = createChatUIStore();
