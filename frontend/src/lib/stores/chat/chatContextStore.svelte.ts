/**
 * Chat Context Store
 *
 * Svelte 5 runes-based singleton store managing all context-selection,
 * project-selection, file-attachment, and team-member state for the
 * chat page.  All UI panel visibility flags live here too, keeping the
 * page component thin.
 *
 * Pattern: createXStore() → singleton export
 */

import { api, apiClient } from "$lib/api";
import type { Node, ContextListItem } from "$lib/api";
import { handleApiCall } from "$lib/utils/api-handler";
import type {
  AttachedFile,
  ProjectItem,
  ChatMessage,
  TeamMember,
} from "./types";

// ─── Constants ───────────────────────────────────────────────────────────────

const FILE_SIZE_LIMIT_BYTES = 50 * 1024 * 1024; // 50 MB
const TOKENS_PER_CHAR = 1 / 4; // rough estimate: 4 chars ≈ 1 token
const WORDS_TO_TOKENS = 1.3; // word_count → estimated token factor

// ─── Store Factory ───────────────────────────────────────────────────────────

function createChatContextStore() {
  // ── Core context state ─────────────────────────────────────────────────────
  let availableContexts = $state<ContextListItem[]>([]);
  let selectedContextIds = $state<string[]>([]);
  let loadingContexts = $state(false);

  // ── Node / workspace focus ─────────────────────────────────────────────────
  let activeNode = $state<Node | null>(null);
  let nodeContextPrompt = $state<string | null>(null);

  // ── Project picker ─────────────────────────────────────────────────────────
  let selectedProjectId = $state<string | null>(null);
  let showProjectDropdown = $state(false);
  let projectsList = $state<ProjectItem[]>([]);
  let loadingProjects = $state(false);
  let projectDropdownIndex = $state(0);

  // ── New-project modal ──────────────────────────────────────────────────────
  let showNewProjectModal = $state(false);
  let newProjectName = $state("");
  let creatingProject = $state(false);

  // ── File attachments ───────────────────────────────────────────────────────
  let attachedFiles = $state<AttachedFile[]>([]);

  // ── UI panel visibility ────────────────────────────────────────────────────
  let showPlusMenu = $state(false);
  let showDocumentUploadModal = $state(false);
  let showHybridSearchPanel = $state(false);
  let showContextDropdown = $state(false);
  let showHeaderContextDropdown = $state(false);
  let showNodeDropdown = $state(false);

  // ── Team members ───────────────────────────────────────────────────────────
  let availableTeamMembers = $state<TeamMember[]>([]);

  // ── Derived values ─────────────────────────────────────────────────────────

  const selectedContexts = $derived(
    availableContexts.filter((ctx) => selectedContextIds.includes(ctx.id)),
  );

  const selectedContextsLabel = $derived((): string => {
    if (selectedContextIds.length === 0) return "Select Context";
    if (selectedContextIds.length === 1) {
      return (
        availableContexts.find((c) => c.id === selectedContextIds[0])?.name ??
        "Select Context"
      );
    }
    return `${selectedContextIds.length} contexts`;
  });

  const selectedProject = $derived(
    projectsList.find((p) => p.id === selectedProjectId) ?? null,
  );

  const nodeContextTokens = $derived(
    nodeContextPrompt
      ? Math.ceil(nodeContextPrompt.length * TOKENS_PER_CHAR)
      : 0,
  );

  const contextDocTokens = $derived(
    selectedContexts.reduce((sum, ctx) => {
      const wordCount =
        typeof ctx.word_count === "string"
          ? parseInt(ctx.word_count, 10)
          : (ctx.word_count ?? 0);
      return (
        sum + Math.ceil((isNaN(wordCount) ? 0 : wordCount) * WORDS_TO_TOKENS)
      );
    }, 0),
  );

  // ── Methods ────────────────────────────────────────────────────────────────

  /** Estimate token count from an array of chat messages. */
  function messageTokens(messages: ChatMessage[]): number {
    return messages.reduce((sum, msg) => {
      return sum + Math.ceil(msg.content.length * TOKENS_PER_CHAR);
    }, 0);
  }

  /** Load available context list from the API. */
  async function loadContexts(): Promise<void> {
    loadingContexts = true;
    try {
      const { data } = await handleApiCall(() => api.getContexts(), {
        showErrorToast: true,
        errorMessage: "Failed to load contexts",
      });
      if (data) {
        availableContexts = data;
      }
    } finally {
      loadingContexts = false;
    }
  }

  /** Load project list from the API and map to lightweight ProjectItem[]. */
  async function loadProjects(): Promise<void> {
    loadingProjects = true;
    try {
      const { data } = await handleApiCall(() => api.getProjects(), {
        showErrorToast: true,
        errorMessage: "Failed to load projects",
      });
      if (data) {
        projectsList = data.map((p) => ({
          id: p.id,
          name: p.name,
          description: p.description ?? undefined,
        }));
      }
    } finally {
      loadingProjects = false;
    }
  }

  /** Load team members from the API. */
  async function loadTeamMembers(): Promise<void> {
    const { data } = await handleApiCall(() => api.getTeamMembers(), {
      showErrorToast: true,
      errorMessage: "Failed to load team members",
    });
    if (data) {
      availableTeamMembers = data.map((m) => ({
        id: m.id,
        name: m.name,
        role: m.role,
      }));
    }
  }

  /**
   * Load the currently active node, then build a context prompt string
   * from its fields for injection into chat.
   */
  async function loadActiveNode(): Promise<void> {
    const { data } = await handleApiCall(() => api.getActiveNode(), {
      showErrorToast: false, // 404 is expected when no node is active
    });

    if (data) {
      activeNode = data;
      const parts: string[] = [
        `Active Node: ${data.name}`,
        data.type ? `Type: ${data.type}` : "",
        data.purpose ? `Purpose: ${data.purpose}` : "",
        data.current_status ? `Current Status: ${data.current_status}` : "",
        data.this_week_focus?.length
          ? `This Week Focus: ${data.this_week_focus.join(", ")}`
          : "",
        data.health ? `Health: ${data.health}` : "",
      ].filter(Boolean);
      nodeContextPrompt = parts.join("\n");
    } else {
      activeNode = null;
      nodeContextPrompt = null;
    }
  }

  /**
   * Deactivate the current node on the server and clear local node state.
   */
  async function handleDeactivateNode(): Promise<void> {
    if (!activeNode) return;

    await handleApiCall(() => api.deactivateNode(activeNode!.id), {
      showErrorToast: true,
      errorMessage: "Failed to deactivate node",
    });

    activeNode = null;
    nodeContextPrompt = null;
    showNodeDropdown = false;
  }

  /**
   * Toggle a context ID in the selected set.
   *
   * @param contextId - The ID of the context to toggle.
   * @param selected  - `true` to add, `false` to remove.
   */
  function handleContextToggle(contextId: string, selected: boolean): void {
    if (selected) {
      if (!selectedContextIds.includes(contextId)) {
        selectedContextIds = [...selectedContextIds, contextId];
      }
    } else {
      selectedContextIds = selectedContextIds.filter((id) => id !== contextId);
    }
  }

  /**
   * Create a new project with `newProjectName`, add it to the list,
   * select it, and reset the form.
   */
  async function createProjectQuick(): Promise<void> {
    const trimmedName = newProjectName.trim();
    if (!trimmedName) return;

    creatingProject = true;
    try {
      const { data } = await handleApiCall(
        () => api.createProject({ name: trimmedName }),
        {
          showErrorToast: true,
          showSuccessToast: true,
          successMessage: `Project "${trimmedName}" created`,
          errorMessage: "Failed to create project",
        },
      );

      if (data) {
        const newItem: ProjectItem = {
          id: data.id,
          name: data.name,
          description: data.description ?? undefined,
        };
        projectsList = [...projectsList, newItem];
        selectedProjectId = data.id;
        showNewProjectModal = false;
        newProjectName = "";
      }
    } finally {
      creatingProject = false;
    }
  }

  /**
   * Handle file selection from an `<input type="file">` element.
   *
   * For each selected file:
   * 1. Enforce the 50 MB size limit — skip oversized files with a toast.
   * 2. For image files, read as a data URL for in-memory preview.
   * 3. Add the file to `attachedFiles` immediately with `uploading: true`.
   * 4. Upload via `FormData` POST to `/api/documents`.
   * 5. On success: stamp the entry with `documentId`, clear `uploading`.
   * 6. On error: stamp the entry with `uploadError`, clear `uploading`.
   * 7. Reset the input element value so the same file can be re-selected.
   *
   * @param event           - The native `change` event from the file input.
   * @param selectedProjectId - Optional project ID to associate the upload with.
   */
  async function handleFileSelect(
    event: Event,
    currentSelectedProjectId: string | null,
  ): Promise<void> {
    const input = event.target as HTMLInputElement;
    const files = input.files;
    if (!files || files.length === 0) return;

    const fileArray = Array.from(files);
    const uploadPromises: Promise<void>[] = [];

    for (const file of fileArray) {
      // ── Size guard ──────────────────────────────────────────────────────────
      if (file.size > FILE_SIZE_LIMIT_BYTES) {
        const { toast } = await import("svelte-sonner");
        toast.error(`"${file.name}" exceeds the 50 MB file size limit`);
        continue;
      }

      // ── Unique client-side ID for this entry ────────────────────────────────
      const fileId = `${Date.now()}-${Math.random().toString(36).slice(2)}`;

      // ── Read preview for images ─────────────────────────────────────────────
      let previewContent: string | undefined;
      if (file.type.startsWith("image/")) {
        previewContent = await new Promise<string>((resolve) => {
          const reader = new FileReader();
          reader.onload = (e) => resolve((e.target?.result as string) ?? "");
          reader.onerror = () => resolve("");
          reader.readAsDataURL(file);
        });
      }

      // ── Optimistic insert ───────────────────────────────────────────────────
      const entry: AttachedFile = {
        id: fileId,
        name: file.name,
        type: file.type,
        size: file.size,
        content: previewContent,
        uploading: true,
      };
      attachedFiles = [...attachedFiles, entry];

      // ── Upload promise ──────────────────────────────────────────────────────
      const uploadPromise = (async () => {
        try {
          const formData = new FormData();
          formData.append("file", file);
          formData.append("title", file.name);
          if (currentSelectedProjectId) {
            formData.append("project_id", currentSelectedProjectId);
          }

          const response = await apiClient.postFormData("/documents", formData);

          if (!response.ok) {
            const errorBody = await response
              .json()
              .catch(() => ({ detail: "Upload failed" }));
            const errorMessage =
              errorBody.detail ?? errorBody.message ?? "Upload failed";
            attachedFiles = attachedFiles.map((f) =>
              f.id === fileId
                ? { ...f, uploading: false, uploadError: errorMessage }
                : f,
            );
            return;
          }

          const result = await response.json();
          const documentId: string | undefined =
            result?.id ?? result?.document_id ?? undefined;

          attachedFiles = attachedFiles.map((f) =>
            f.id === fileId ? { ...f, uploading: false, documentId } : f,
          );
        } catch (err) {
          const message = err instanceof Error ? err.message : "Upload failed";
          attachedFiles = attachedFiles.map((f) =>
            f.id === fileId
              ? { ...f, uploading: false, uploadError: message }
              : f,
          );
        }
      })();

      uploadPromises.push(uploadPromise);
    }

    // Wait for all uploads before resetting the input so the user can see
    // immediate feedback, then clear the value to allow re-selection.
    await Promise.allSettled(uploadPromises);
    input.value = "";
  }

  /** Remove a single attached file by its client-side ID. */
  function removeAttachedFile(fileId: string): void {
    attachedFiles = attachedFiles.filter((f) => f.id !== fileId);
  }

  /** Clear all attached files. */
  function clearAttachedFiles(): void {
    attachedFiles = [];
  }

  /**
   * Return the backend document IDs of all successfully uploaded files.
   * Files still uploading or that errored are excluded.
   */
  function getUploadedDocumentIds(): string[] {
    return attachedFiles
      .filter(
        (f): f is AttachedFile & { documentId: string } =>
          typeof f.documentId === "string",
      )
      .map((f) => f.documentId);
  }

  // ── Public surface ─────────────────────────────────────────────────────────

  return {
    // ── State getters / setters ─────────────────────────────────────────────

    get availableContexts() {
      return availableContexts;
    },
    set availableContexts(v: ContextListItem[]) {
      availableContexts = v;
    },

    get selectedContextIds() {
      return selectedContextIds;
    },
    set selectedContextIds(v: string[]) {
      selectedContextIds = v;
    },

    get loadingContexts() {
      return loadingContexts;
    },
    set loadingContexts(v: boolean) {
      loadingContexts = v;
    },

    get activeNode() {
      return activeNode;
    },
    set activeNode(v: Node | null) {
      activeNode = v;
    },

    get nodeContextPrompt() {
      return nodeContextPrompt;
    },
    set nodeContextPrompt(v: string | null) {
      nodeContextPrompt = v;
    },

    get selectedProjectId() {
      return selectedProjectId;
    },
    set selectedProjectId(v: string | null) {
      selectedProjectId = v;
    },

    get showProjectDropdown() {
      return showProjectDropdown;
    },
    set showProjectDropdown(v: boolean) {
      showProjectDropdown = v;
    },

    get projectsList() {
      return projectsList;
    },
    set projectsList(v: ProjectItem[]) {
      projectsList = v;
    },

    get loadingProjects() {
      return loadingProjects;
    },
    set loadingProjects(v: boolean) {
      loadingProjects = v;
    },

    get projectDropdownIndex() {
      return projectDropdownIndex;
    },
    set projectDropdownIndex(v: number) {
      projectDropdownIndex = v;
    },

    get showNewProjectModal() {
      return showNewProjectModal;
    },
    set showNewProjectModal(v: boolean) {
      showNewProjectModal = v;
    },

    get newProjectName() {
      return newProjectName;
    },
    set newProjectName(v: string) {
      newProjectName = v;
    },

    get creatingProject() {
      return creatingProject;
    },
    set creatingProject(v: boolean) {
      creatingProject = v;
    },

    get attachedFiles() {
      return attachedFiles;
    },
    set attachedFiles(v: AttachedFile[]) {
      attachedFiles = v;
    },

    get showPlusMenu() {
      return showPlusMenu;
    },
    set showPlusMenu(v: boolean) {
      showPlusMenu = v;
    },

    get showDocumentUploadModal() {
      return showDocumentUploadModal;
    },
    set showDocumentUploadModal(v: boolean) {
      showDocumentUploadModal = v;
    },

    get showHybridSearchPanel() {
      return showHybridSearchPanel;
    },
    set showHybridSearchPanel(v: boolean) {
      showHybridSearchPanel = v;
    },

    get showContextDropdown() {
      return showContextDropdown;
    },
    set showContextDropdown(v: boolean) {
      showContextDropdown = v;
    },

    get showHeaderContextDropdown() {
      return showHeaderContextDropdown;
    },
    set showHeaderContextDropdown(v: boolean) {
      showHeaderContextDropdown = v;
    },

    get showNodeDropdown() {
      return showNodeDropdown;
    },
    set showNodeDropdown(v: boolean) {
      showNodeDropdown = v;
    },

    get availableTeamMembers() {
      return availableTeamMembers;
    },
    set availableTeamMembers(v: TeamMember[]) {
      availableTeamMembers = v;
    },

    // ── Derived getters ─────────────────────────────────────────────────────

    get selectedContexts() {
      return selectedContexts;
    },

    get selectedContextsLabel() {
      return selectedContextsLabel;
    },

    get selectedProject() {
      return selectedProject;
    },

    get nodeContextTokens() {
      return nodeContextTokens;
    },

    get contextDocTokens() {
      return contextDocTokens;
    },

    // ── Methods ─────────────────────────────────────────────────────────────

    messageTokens,
    loadContexts,
    loadProjects,
    loadTeamMembers,
    loadActiveNode,
    handleDeactivateNode,
    handleContextToggle,
    createProjectQuick,
    handleFileSelect,
    removeAttachedFile,
    clearAttachedFiles,
    getUploadedDocumentIds,
  };
}

export const chatContextStore = createChatContextStore();
