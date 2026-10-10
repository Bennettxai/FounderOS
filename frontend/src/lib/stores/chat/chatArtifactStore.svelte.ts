/**
 * Chat Artifact Store
 * Svelte 5 runes-based singleton store for all artifact state and operations
 * extracted from chat/+page.svelte.
 */

import { api } from "$lib/api";
import { getActiveWorkspaceHeaders, getApiBaseUrl } from "$lib/api/base";
import type {
  ArtifactListItem,
  Artifact,
  Node,
  ContextListItem,
} from "$lib/api";
import { handleApiCall } from "$lib/utils/api-handler";
import { markdownToBlocks } from "$lib/utils/markdown-blocks";
import type { GeneratedTask, TeamMember } from "./types";

// ── Private helpers ──────────────────────────────────────────────────────────

const ALLOWED_ARTIFACT_TYPES = new Set([
  "proposal",
  "sop",
  "framework",
  "agenda",
  "report",
  "plan",
  "code",
  "document",
  "markdown",
  "other",
] as const);

const ARTIFACT_TYPE_ICONS: Record<string, string> = {
  plan: "📋",
  proposal: "📄",
  framework: "🗺️",
  sop: "📖",
  report: "📊",
};

const ROLE_KEYWORDS: Record<string, string[]> = {
  developer: [
    "code",
    "implement",
    "build",
    "develop",
    "api",
    "frontend",
    "backend",
    "database",
    "bug",
    "fix",
    "feature",
    "technical",
    "integration",
  ],
  designer: [
    "design",
    "ui",
    "ux",
    "mockup",
    "wireframe",
    "visual",
    "layout",
    "style",
    "brand",
  ],
  "project manager": [
    "coordinate",
    "schedule",
    "timeline",
    "milestone",
    "meeting",
    "stakeholder",
    "plan",
    "track",
    "report",
  ],
  ceo: [
    "strategy",
    "vision",
    "decision",
    "executive",
    "leadership",
    "partnership",
    "investor",
  ],
  cto: [
    "architecture",
    "infrastructure",
    "security",
    "scalability",
    "technical strategy",
    "technology",
  ],
  marketing: [
    "marketing",
    "campaign",
    "content",
    "social",
    "seo",
    "advertising",
    "promotion",
    "brand",
  ],
  sales: [
    "sales",
    "client",
    "customer",
    "deal",
    "proposal",
    "pitch",
    "revenue",
    "lead",
  ],
  operations: [
    "operations",
    "process",
    "workflow",
    "efficiency",
    "sop",
    "documentation",
  ],
  qa: ["test", "quality", "qa", "bug", "verification", "validation"],
  devops: [
    "deploy",
    "ci/cd",
    "infrastructure",
    "monitoring",
    "server",
    "cloud",
    "kubernetes",
    "docker",
  ],
};

// ── Store factory ────────────────────────────────────────────────────────────

function createChatArtifactStore() {
  // Core artifact list
  let artifacts = $state<ArtifactListItem[]>([]);
  let selectedArtifact = $state<Artifact | null>(null);
  let loadingArtifacts = $state(false);
  let artifactFilter = $state("all");
  let artifactsPanelOpen = $state(false);
  let artifactsLoadedOnce = $state(false);

  // Ephemeral viewing (artifact from a streamed message, before persistence)
  let viewingArtifactFromMessage = $state<{
    title: string;
    type: string;
    content: string;
  } | null>(null);

  // Streaming generation state
  let generatingArtifact = $state(false);
  let generatingArtifactTitle = $state("");
  let generatingArtifactType = $state("");
  let generatingArtifactContent = $state("");
  let artifactCompletedInStream = $state(false);

  // Save to profile modal
  let showSaveToProfileModal = $state(false);
  let availableProfiles = $state<ContextListItem[]>([]);
  let selectedProfileForSave = $state("");
  let savingArtifactToProfile = $state(false);

  // Save to node modal (legacy)
  let showSaveToNodeModal = $state(false);
  let availableNodes = $state<Node[]>([]);
  let selectedNodeForSave = $state("");

  // Task generation modal
  let showTaskGenerationModal = $state(false);
  let generatingTasks = $state(false);
  let generatedTasks = $state<GeneratedTask[]>([]);
  let selectedProjectForTasks = $state("");
  let taskGenerationArtifact = $state<{
    title: string;
    type: string;
    content: string;
  } | null>(null);
  let availableProjects = $state<{ id: string; name: string }[]>([]);

  // Inline task creation (auto-triggered after actionable artifacts)
  let showInlineTaskCreation = $state(false);
  let inlineTasksForArtifact = $state<GeneratedTask[]>([]);
  let creatingInlineTasks = $state(false);

  // ── Derived ─────────────────────────────────────────────────────────────────

  const isArtifactFocused = $derived(
    viewingArtifactFromMessage !== null ||
      generatingArtifact ||
      selectedArtifact !== null,
  );

  // ── Private helpers ──────────────────────────────────────────────────────────

  function findBestAssignee(
    task: GeneratedTask,
    teamMembers: TeamMember[],
  ): TeamMember | undefined {
    const title = task.title.toLowerCase();
    const desc = (task.description || "").toLowerCase();
    const combined = `${title} ${desc}`;

    let bestMatch: { member: TeamMember; score: number } | null = null;

    for (const member of teamMembers) {
      const memberRole = member.role.toLowerCase();
      let score = 0;

      for (const [role, keywords] of Object.entries(ROLE_KEYWORDS)) {
        if (memberRole.includes(role)) {
          for (const keyword of keywords) {
            if (combined.includes(keyword)) {
              score += 10;
            }
          }
        }
      }

      if (combined.includes(memberRole)) {
        score += 20;
      }

      if (score > 0 && (!bestMatch || score > bestMatch.score)) {
        bestMatch = { member, score };
      }
    }

    return bestMatch?.member;
  }

  // ── Public methods ───────────────────────────────────────────────────────────

  async function loadArtifacts(): Promise<void> {
    loadingArtifacts = true;
    try {
      const filters: { type?: string } = {};
      if (artifactFilter !== "all") filters.type = artifactFilter;
      const result = await api.getArtifacts(filters);
      artifacts = result;
    } catch (error) {
      console.error("[chatArtifactStore] Failed to load artifacts:", error);
      artifacts = [];
    } finally {
      loadingArtifacts = false;
    }
  }

  async function selectArtifact(id: string): Promise<void> {
    try {
      selectedArtifact = await api.getArtifact(id);
    } catch (error) {
      console.error("[chatArtifactStore] Failed to load artifact:", error);
    }
  }

  function closeArtifactDetail(): void {
    selectedArtifact = null;
  }

  function viewArtifactInPanel(artifact: {
    title: string;
    type: string;
    content: string;
  }): void {
    viewingArtifactFromMessage = artifact;
    selectedArtifact = null;
    artifactsPanelOpen = true;
  }

  function updateArtifactContent(content: string): void {
    if (viewingArtifactFromMessage) {
      viewingArtifactFromMessage = { ...viewingArtifactFromMessage, content };
    }
  }

  function updateSelectedArtifactContent(content: string): void {
    if (selectedArtifact) {
      selectedArtifact = { ...selectedArtifact, content };
    }
  }

  // autoSaveArtifact no longer persists to the API — the backend handles
  // persistence in postProcessStream after the stream ends. This function
  // is kept for backward-compatibility with any callers that may still
  // reference it, but it is now a no-op that returns null.
  // Use loadArtifacts() after the stream ends to sync from the database.
  async function autoSaveArtifact(
    _artifactData: { title: string; type: string; content: string },
    _conversationId: string | null,
    _selectedProjectId: string | null,
  ): Promise<Artifact | null> {
    // Intentionally empty: backend saves the artifact; frontend syncs via loadArtifacts().
    return null;
  }

  async function deleteArtifactById(id: string): Promise<void> {
    const { data } = await handleApiCall(() => api.deleteArtifact(id), {
      showErrorToast: true,
      showSuccessToast: true,
      successMessage: "Artifact deleted successfully",
      errorMessage: "Failed to delete artifact",
    });

    if (data !== null) {
      artifacts = artifacts.filter((a) => a.id !== id);
      if (selectedArtifact?.id === id) {
        selectedArtifact = null;
      }
    }
  }

  async function loadAvailableProfiles(): Promise<void> {
    try {
      const contexts = await api.getContexts();
      availableProfiles = contexts.filter((c) => c.type !== "document");
    } catch (error) {
      console.error("[chatArtifactStore] Failed to load profiles:", error);
    }
  }

  function openSaveToProfileModal(): void {
    loadAvailableProfiles();
    showSaveToProfileModal = true;
    selectedProfileForSave = "";
  }

  async function saveArtifactToProfile(): Promise<void> {
    if (!selectedProfileForSave || !viewingArtifactFromMessage) return;

    savingArtifactToProfile = true;

    const blocks = markdownToBlocks(viewingArtifactFromMessage.content);
    const icon = ARTIFACT_TYPE_ICONS[viewingArtifactFromMessage.type] ?? "📑";

    const { data } = await handleApiCall(
      async () =>
        api.createContext({
          name: viewingArtifactFromMessage!.title,
          type: "document",
          blocks,
          parent_id:
            selectedProfileForSave === "loose"
              ? undefined
              : selectedProfileForSave,
          icon,
        }),
      {
        showErrorToast: true,
        showSuccessToast: true,
        errorMessage: "Failed to save artifact to profile",
        successMessage: "Artifact saved to profile successfully",
      },
    );

    if (data) {
      showSaveToProfileModal = false;
      selectedProfileForSave = "";
      viewingArtifactFromMessage = null;
      artifactsPanelOpen = false;
    }

    savingArtifactToProfile = false;
  }

  async function generateTasksFromArtifact(
    artifact: { title: string; type: string; content: string },
    teamMembers: TeamMember[],
  ): Promise<void> {
    taskGenerationArtifact = artifact;
    showTaskGenerationModal = true;
    generatingTasks = true;
    generatedTasks = [];

    try {
      const projects = await api.getProjects();
      availableProjects = projects.map((p) => ({ id: p.id, name: p.name }));
    } catch (error) {
      console.error("[chatArtifactStore] Failed to load projects:", error);
    }

    try {
      const response = await fetch(`${getApiBaseUrl()}/chat/ai/extract-tasks`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...getActiveWorkspaceHeaders(),
        },
        credentials: "include",
        body: JSON.stringify({
          artifact_title: artifact.title,
          artifact_content: artifact.content,
          artifact_type: artifact.type,
          team_members: teamMembers,
        }),
      });

      if (!response.ok) throw new Error("Failed to extract tasks");

      const data = await response.json();
      generatedTasks = data.tasks ?? [];
    } catch (error) {
      console.error("[chatArtifactStore] Failed to generate tasks:", error);
      generatedTasks = [];
    } finally {
      generatingTasks = false;
    }
  }

  async function confirmTaskCreation(): Promise<void> {
    if (!selectedProjectForTasks || generatedTasks.length === 0) return;

    await handleApiCall(
      async () => {
        for (const task of generatedTasks) {
          await api.createTask({
            title: task.title,
            description: task.description,
            project_id: selectedProjectForTasks,
            priority: task.priority,
            assignee_id: task.assignee_id,
          });
        }
        return true;
      },
      {
        showErrorToast: true,
        showSuccessToast: false,
        errorMessage: "Failed to create tasks",
      },
    );

    showTaskGenerationModal = false;
    generatedTasks = [];
    taskGenerationArtifact = null;
  }

  function removeGeneratedTask(index: number): void {
    generatedTasks = generatedTasks.filter((_, i) => i !== index);
  }

  function updateTaskAssignee(index: number, assigneeId: string): void {
    generatedTasks = generatedTasks.map((task, i) =>
      i === index ? { ...task, assignee_id: assigneeId } : task,
    );
  }

  async function triggerInlineTaskCreation(
    artifact: { title: string; type: string; content: string },
    teamMembers: TeamMember[],
  ): Promise<void> {
    showInlineTaskCreation = true;
    creatingInlineTasks = true;
    inlineTasksForArtifact = [];

    try {
      const response = await fetch(`${getApiBaseUrl()}/chat/ai/extract-tasks`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...getActiveWorkspaceHeaders(),
        },
        credentials: "include",
        body: JSON.stringify({
          artifact_title: artifact.title,
          artifact_content: artifact.content,
          artifact_type: artifact.type,
          team_members: teamMembers,
        }),
      });

      if (!response.ok) throw new Error("Failed to extract tasks");

      const data = await response.json();
      const tasks: GeneratedTask[] = data.tasks ?? [];

      inlineTasksForArtifact = tasks.map((task) => {
        const assignee = findBestAssignee(task, teamMembers);
        return { ...task, assignee_id: assignee?.id };
      });
    } catch (error) {
      console.error("[chatArtifactStore] Failed to generate tasks:", error);
      inlineTasksForArtifact = [];
    } finally {
      creatingInlineTasks = false;
    }
  }

  async function confirmInlineTasks(
    selectedProjectId: string | null,
  ): Promise<string> {
    if (inlineTasksForArtifact.length === 0) return "";

    creatingInlineTasks = true;
    let confirmMessage = "";

    const { data } = await handleApiCall(
      async () => {
        for (const task of inlineTasksForArtifact) {
          const taskData: {
            title: string;
            description: string;
            priority: "low" | "medium" | "high";
            assignee_id?: string;
            project_id?: string;
          } = {
            title: task.title,
            description: task.description,
            priority: task.priority,
            assignee_id: task.assignee_id,
          };
          if (selectedProjectId) {
            taskData.project_id = selectedProjectId;
          }
          await api.createTask(taskData);
        }
        return { success: true };
      },
      {
        showErrorToast: true,
        showSuccessToast: false,
        errorMessage: "Failed to create tasks",
      },
    );

    if (data) {
      const count = inlineTasksForArtifact.length;
      confirmMessage = `Created ${count} task${count > 1 ? "s" : ""} from the artifact. You can view them in the Tasks tab or project dashboard.`;
      showInlineTaskCreation = false;
      inlineTasksForArtifact = [];
    }

    creatingInlineTasks = false;
    return confirmMessage;
  }

  function dismissInlineTasks(): void {
    showInlineTaskCreation = false;
    inlineTasksForArtifact = [];
  }

  function updateInlineTaskAssignee(index: number, assigneeId: string): void {
    inlineTasksForArtifact = inlineTasksForArtifact.map((task, i) =>
      i === index ? { ...task, assignee_id: assigneeId } : task,
    );
  }

  function removeInlineTask(index: number): void {
    inlineTasksForArtifact = inlineTasksForArtifact.filter(
      (_, i) => i !== index,
    );
  }

  function resetStreamingArtifactState(): void {
    generatingArtifact = false;
    generatingArtifactTitle = "";
    generatingArtifactType = "";
    generatingArtifactContent = "";
    artifactCompletedInStream = false;
  }

  function onArtifactStart(title: string, type: string): void {
    generatingArtifact = true;
    generatingArtifactTitle = title;
    generatingArtifactType = type;
    generatingArtifactContent = "";
    artifactCompletedInStream = false;
    artifactsPanelOpen = true;
  }

  function onArtifactComplete(artifact: {
    title: string;
    artifactType: string;
    content: string;
  }): void {
    const processedContent = artifact.content
      .replace(/\\n/g, "\n")
      .replace(/\\"/g, '"')
      .replace(/\\\\/g, "\\");

    // Update local UI state immediately so the panel renders the artifact
    // before the backend finishes persisting it. The backend (postProcessStream)
    // is the single source of truth for persistence — no API call is made here.
    viewingArtifactFromMessage = {
      title: artifact.title,
      type: artifact.artifactType,
      content: processedContent,
    };

    artifactsPanelOpen = true;
    generatingArtifact = false;
    artifactCompletedInStream = true;
    generatingArtifactTitle = artifact.title;
    generatingArtifactType = artifact.artifactType;
  }

  // ── Public surface ───────────────────────────────────────────────────────────

  return {
    // ── State getters / setters ──
    get artifacts() {
      return artifacts;
    },
    set artifacts(v: ArtifactListItem[]) {
      artifacts = v;
    },

    get selectedArtifact() {
      return selectedArtifact;
    },
    set selectedArtifact(v: Artifact | null) {
      selectedArtifact = v;
    },

    get loadingArtifacts() {
      return loadingArtifacts;
    },
    set loadingArtifacts(v: boolean) {
      loadingArtifacts = v;
    },

    get artifactFilter() {
      return artifactFilter;
    },
    set artifactFilter(v: string) {
      artifactFilter = v;
    },

    get artifactsPanelOpen() {
      return artifactsPanelOpen;
    },
    set artifactsPanelOpen(v: boolean) {
      artifactsPanelOpen = v;
    },

    get artifactsLoadedOnce() {
      return artifactsLoadedOnce;
    },
    set artifactsLoadedOnce(v: boolean) {
      artifactsLoadedOnce = v;
    },

    get viewingArtifactFromMessage() {
      return viewingArtifactFromMessage;
    },
    set viewingArtifactFromMessage(
      v: { title: string; type: string; content: string } | null,
    ) {
      viewingArtifactFromMessage = v;
    },

    get generatingArtifact() {
      return generatingArtifact;
    },
    set generatingArtifact(v: boolean) {
      generatingArtifact = v;
    },

    get generatingArtifactTitle() {
      return generatingArtifactTitle;
    },
    set generatingArtifactTitle(v: string) {
      generatingArtifactTitle = v;
    },

    get generatingArtifactType() {
      return generatingArtifactType;
    },
    set generatingArtifactType(v: string) {
      generatingArtifactType = v;
    },

    get generatingArtifactContent() {
      return generatingArtifactContent;
    },
    set generatingArtifactContent(v: string) {
      generatingArtifactContent = v;
    },

    get artifactCompletedInStream() {
      return artifactCompletedInStream;
    },
    set artifactCompletedInStream(v: boolean) {
      artifactCompletedInStream = v;
    },

    get showSaveToProfileModal() {
      return showSaveToProfileModal;
    },
    set showSaveToProfileModal(v: boolean) {
      showSaveToProfileModal = v;
    },

    get availableProfiles() {
      return availableProfiles;
    },
    set availableProfiles(v: ContextListItem[]) {
      availableProfiles = v;
    },

    get selectedProfileForSave() {
      return selectedProfileForSave;
    },
    set selectedProfileForSave(v: string) {
      selectedProfileForSave = v;
    },

    get savingArtifactToProfile() {
      return savingArtifactToProfile;
    },
    set savingArtifactToProfile(v: boolean) {
      savingArtifactToProfile = v;
    },

    get showSaveToNodeModal() {
      return showSaveToNodeModal;
    },
    set showSaveToNodeModal(v: boolean) {
      showSaveToNodeModal = v;
    },

    get availableNodes() {
      return availableNodes;
    },
    set availableNodes(v: Node[]) {
      availableNodes = v;
    },

    get selectedNodeForSave() {
      return selectedNodeForSave;
    },
    set selectedNodeForSave(v: string) {
      selectedNodeForSave = v;
    },

    get showTaskGenerationModal() {
      return showTaskGenerationModal;
    },
    set showTaskGenerationModal(v: boolean) {
      showTaskGenerationModal = v;
    },

    get generatingTasks() {
      return generatingTasks;
    },
    set generatingTasks(v: boolean) {
      generatingTasks = v;
    },

    get generatedTasks() {
      return generatedTasks;
    },
    set generatedTasks(v: GeneratedTask[]) {
      generatedTasks = v;
    },

    get selectedProjectForTasks() {
      return selectedProjectForTasks;
    },
    set selectedProjectForTasks(v: string) {
      selectedProjectForTasks = v;
    },

    get taskGenerationArtifact() {
      return taskGenerationArtifact;
    },
    set taskGenerationArtifact(
      v: { title: string; type: string; content: string } | null,
    ) {
      taskGenerationArtifact = v;
    },

    get availableProjects() {
      return availableProjects;
    },
    set availableProjects(v: { id: string; name: string }[]) {
      availableProjects = v;
    },

    get showInlineTaskCreation() {
      return showInlineTaskCreation;
    },
    set showInlineTaskCreation(v: boolean) {
      showInlineTaskCreation = v;
    },

    get inlineTasksForArtifact() {
      return inlineTasksForArtifact;
    },
    set inlineTasksForArtifact(v: GeneratedTask[]) {
      inlineTasksForArtifact = v;
    },

    get creatingInlineTasks() {
      return creatingInlineTasks;
    },
    set creatingInlineTasks(v: boolean) {
      creatingInlineTasks = v;
    },

    // ── Derived (read-only) ──
    get isArtifactFocused() {
      return isArtifactFocused;
    },

    // ── Methods ──
    loadArtifacts,
    selectArtifact,
    closeArtifactDetail,
    viewArtifactInPanel,
    updateArtifactContent,
    updateSelectedArtifactContent,
    autoSaveArtifact,
    deleteArtifactById,
    loadAvailableProfiles,
    openSaveToProfileModal,
    saveArtifactToProfile,
    generateTasksFromArtifact,
    confirmTaskCreation,
    removeGeneratedTask,
    updateTaskAssignee,
    triggerInlineTaskCreation,
    confirmInlineTasks,
    dismissInlineTasks,
    updateInlineTaskAssignee,
    removeInlineTask,
    resetStreamingArtifactState,
    onArtifactStart,
    onArtifactComplete,
  };
}

export const chatArtifactStore = createChatArtifactStore();
