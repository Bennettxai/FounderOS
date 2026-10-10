/**
 * Editor Store — Generated Apps [id] page
 * Manages all code-workspace state: active tab, file tree, selected file,
 * editor content, editing mode, cursor position, file search, and loading
 * transitions.
 *
 * Uses the Svelte 5 singleton factory pattern with $state runes.
 * $derived values are exposed as plain getters that re-compute reactively.
 */

import type { OSAFile, FileTreeNode, ActiveTab } from "./types";
import { detectLanguage } from "$lib/editor/utils/language-detection";
import { getWorkflowFiles, getFileContent } from "$lib/api/osa/files";
import type MonacoEditor from "$lib/editor/MonacoEditor.svelte";

function buildFileTree(files: OSAFile[]): FileTreeNode[] {
  const root: FileTreeNode[] = [];
  const folderMap = new Map<string, FileTreeNode>();

  for (const file of files) {
    const parts = (file.path || file.name).split("/");
    let currentPath = "";

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const parentPath = currentPath;
      currentPath = currentPath ? `${currentPath}/${part}` : part;

      if (i === parts.length - 1) {
        const node: FileTreeNode = {
          id: file.id,
          name: part,
          path: currentPath,
          type: "file",
          file,
        };
        const parent = folderMap.get(parentPath);
        if (parent) {
          parent.children = parent.children || [];
          parent.children.push(node);
        } else {
          root.push(node);
        }
      } else {
        if (!folderMap.has(currentPath)) {
          const folder: FileTreeNode = {
            id: `folder-${currentPath}`,
            name: part,
            path: currentPath,
            type: "folder",
            children: [],
            expanded: true,
          };
          folderMap.set(currentPath, folder);
          const parent = folderMap.get(parentPath);
          if (parent) {
            parent.children = parent.children || [];
            parent.children.push(folder);
          } else {
            root.push(folder);
          }
        }
      }
    }
  }
  return root;
}

function filterTree(nodes: FileTreeNode[], query: string): FileTreeNode[] {
  if (!query.trim()) return nodes;
  const q = query.toLowerCase();
  return nodes
    .map((node) => {
      if (node.type === "folder") {
        const children = filterTree(node.children || [], query);
        if (children.length > 0) return { ...node, children, expanded: true };
        return null;
      }
      return node.name.toLowerCase().includes(q) ||
        node.path.toLowerCase().includes(q)
        ? node
        : null;
    })
    .filter(Boolean) as FileTreeNode[];
}

function createEditorStore() {
  // ── Tab ──────────────────────────────────────────────────────────────────
  let activeTab = $state<ActiveTab>("preview");

  // ── Files ─────────────────────────────────────────────────────────────────
  let files = $state<OSAFile[]>([]);
  let fileTreeNodes = $state<FileTreeNode[]>([]);
  let selectedFile = $state<OSAFile | null>(null);
  let fileSearchQuery = $state("");

  // ── Editor content ────────────────────────────────────────────────────────
  let editorValue = $state("");
  let originalValue = $state("");
  let isEditing = $state(false);
  let isSaving = $state(false);
  let saveError = $state<string | null>(null);
  let fileLoading = $state(false);
  let isTransitioning = $state(false);

  // ── Cursor position ───────────────────────────────────────────────────────
  let cursorLine = $state(1);
  let cursorColumn = $state(1);

  // ── DOM ref (bound by page, not reactive store state) ────────────────────
  let editorRef = $state<MonacoEditor | undefined>(undefined);

  return {
    // Tab
    get activeTab() {
      return activeTab;
    },
    set activeTab(v: ActiveTab) {
      activeTab = v;
    },

    // Files
    get files() {
      return files;
    },
    set files(v: OSAFile[]) {
      files = v;
    },

    get fileTreeNodes() {
      return fileTreeNodes;
    },
    set fileTreeNodes(v: FileTreeNode[]) {
      fileTreeNodes = v;
    },

    get selectedFile() {
      return selectedFile;
    },
    set selectedFile(v: OSAFile | null) {
      selectedFile = v;
    },

    get fileSearchQuery() {
      return fileSearchQuery;
    },
    set fileSearchQuery(v: string) {
      fileSearchQuery = v;
    },

    // Derived: filtered tree
    get filteredTreeNodes() {
      return filterTree(fileTreeNodes, fileSearchQuery);
    },

    // Editor content
    get editorValue() {
      return editorValue;
    },
    set editorValue(v: string) {
      editorValue = v;
    },

    get originalValue() {
      return originalValue;
    },
    set originalValue(v: string) {
      originalValue = v;
    },

    // Derived: dirty flag
    get isDirty() {
      return editorValue !== originalValue;
    },

    get isEditing() {
      return isEditing;
    },
    set isEditing(v: boolean) {
      isEditing = v;
    },

    get isSaving() {
      return isSaving;
    },
    set isSaving(v: boolean) {
      isSaving = v;
    },

    get saveError() {
      return saveError;
    },
    set saveError(v: string | null) {
      saveError = v;
    },

    get fileLoading() {
      return fileLoading;
    },
    set fileLoading(v: boolean) {
      fileLoading = v;
    },

    get isTransitioning() {
      return isTransitioning;
    },
    set isTransitioning(v: boolean) {
      isTransitioning = v;
    },

    // Cursor
    get cursorLine() {
      return cursorLine;
    },
    set cursorLine(v: number) {
      cursorLine = v;
    },

    get cursorColumn() {
      return cursorColumn;
    },
    set cursorColumn(v: number) {
      cursorColumn = v;
    },

    // Derived: language id
    get languageId() {
      return selectedFile
        ? detectLanguage(selectedFile.path || selectedFile.name)
        : "plaintext";
    },

    // DOM ref
    get editorRef() {
      return editorRef;
    },
    set editorRef(v: MonacoEditor | undefined) {
      editorRef = v;
    },

    // ── Methods ─────────────────────────────────────────────────────────

    /** Load files for a given app id and rebuild the file tree. */
    async loadFiles(appId: string) {
      try {
        const appFiles = await getWorkflowFiles(appId);
        files = appFiles;
        fileTreeNodes = buildFileTree(appFiles);
      } catch {
        files = [];
        fileTreeNodes = [];
      }
    },

    /** Select a file, fetch its content, and animate the transition. */
    async selectFile(file: OSAFile) {
      if (selectedFile?.id === file.id) return;
      isTransitioning = true;
      fileLoading = true;

      await new Promise<void>((resolve) => {
        setTimeout(async () => {
          selectedFile = file;
          isEditing = false;

          try {
            if (file.content) {
              editorValue = file.content;
              originalValue = file.content;
            } else {
              const result = await getFileContent(file.id);
              editorValue = result.content;
              originalValue = result.content;
            }
          } catch {
            editorValue = "// Failed to load file content";
            originalValue = editorValue;
          } finally {
            fileLoading = false;
            requestAnimationFrame(() => {
              isTransitioning = false;
              resolve();
            });
          }
        }, 150);
      });
    },

    /** Toggle edit mode and focus the editor when entering. */
    toggleEdit() {
      isEditing = !isEditing;
      if (isEditing) {
        setTimeout(() => editorRef?.focus(), 50);
      }
    },

    /** Save the current editor value to the backend. */
    async saveFile(appId: string, value: string) {
      if (!selectedFile || isSaving) return;
      isSaving = true;
      saveError = null;

      try {
        const response = await fetch(
          `/api/osa/module-instances/${appId}/files`,
          {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({
              file_id: selectedFile.id,
              content: value,
            }),
          },
        );

        if (!response.ok) {
          if (response.status === 404) {
            console.warn("[Editor] Save endpoint not available yet (404)");
          }
          const errorData = (await response.json().catch(() => ({}))) as {
            message?: string;
          };
          throw new Error(
            errorData.message || `Save failed (${response.status})`,
          );
        }

        originalValue = value;
        isEditing = false;
      } catch (err) {
        saveError = err instanceof Error ? err.message : "Failed to save file";
        console.error("[Editor] Save failed:", err);
      } finally {
        isSaving = false;
      }
    },

    /** Copy current editor value to clipboard. */
    copyToClipboard() {
      navigator.clipboard.writeText(editorValue);
    },

    /** Update cursor position from a Monaco editor position event. */
    updateCursorFromEditor(editor: {
      getPosition(): { lineNumber: number; column: number } | null;
    }) {
      const pos = editor.getPosition();
      if (pos) {
        cursorLine = pos.lineNumber;
        cursorColumn = pos.column;
      }
    },

    /** Dismiss the save error banner. */
    clearSaveError() {
      saveError = null;
    },

    /** Reset all editor state (call on app navigation / destroy). */
    reset() {
      files = [];
      fileTreeNodes = [];
      selectedFile = null;
      editorValue = "";
      originalValue = "";
      isEditing = false;
      isSaving = false;
      saveError = null;
      fileLoading = false;
      isTransitioning = false;
      fileSearchQuery = "";
      cursorLine = 1;
      cursorColumn = 1;
    },
  };
}

export const editorStore = createEditorStore();
