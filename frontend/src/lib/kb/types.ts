// Knowledge Base types. Markdown files on disk are the source of truth (Plane 1);
// these shapes mirror what the backend /api/knowledge endpoints return.

export interface KBTreeNode {
  name: string;
  path: string; // workspace-relative, forward-slash
  type: "dir" | "file";
  title?: string;
  children?: KBTreeNode[];
  indexPath?: string; // for dirs: README that acts as the folder's own page
}

export interface KBFile {
  workspace: string;
  path: string;
  content: string;
  modified?: string;
}

// Forward-looking model (Notion-style) the Docs view will grow into.
export type KBTab =
  | "docs"
  | "nodes"
  | "sources"
  | "decisions"
  | "sops"
  | "packages"
  | "agent";
