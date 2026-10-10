import {
  documentsStore,
  activeDocumentStore,
  treeStore,
} from "../stores/documents";
import type {
  Document,
  DocumentMeta,
  DocumentType,
  Block,
  BlockType,
  RichText,
  PropertyValue,
} from "../entities/types";
import * as contextsApi from "$lib/api/contexts";
import type {
  Context,
  ContextListItem,
  Block as ContextBlock,
} from "$lib/api/contexts/types";
import { getApiBaseUrl, getCSRFToken } from "$lib/api/base";
import type { OptimalNode, FileEntry } from "$lib/stores/optimal";
import {
  extractRichTextPlainText,
  parseInlineMarkdownToRichText,
  richTextToMarkdown,
} from "../utils/rich-text";

/**
 * Document Service
 * Bridges the knowledge-base module with the existing contexts API.
 * Maps Context ↔ Document for compatibility.
 */

// ============================================================================
// Type Mapping Helpers
// ============================================================================

/**
 * Map ContextType to DocumentType
 */
function mapContextTypeToDocType(type: string): DocumentType {
  switch (type) {
    case "document":
      return "document";
    case "custom":
      return "document";
    case "person":
    case "business":
    case "project":
      return "database"; // Context profiles are like databases
    default:
      return "document";
  }
}

/**
 * Map ContextListItem to DocumentMeta
 */
function mapContextToDocMeta(ctx: ContextListItem): DocumentMeta {
  return {
    id: ctx.id,
    parent_id: ctx.parent_id,
    type: mapContextTypeToDocType(ctx.type),
    title: ctx.name,
    icon: ctx.icon ? { type: "emoji", value: ctx.icon } : null,
    is_favorite: ctx.properties?.is_favorite === true,
    is_archived: ctx.is_archived,
    updated_at: ctx.updated_at,
    children_count: 0, // Will be calculated from tree structure
  };
}

/**
 * Map full Context to Document
 */
function mapContextToDocument(ctx: Context): Document {
  return {
    id: ctx.id,
    workspace_id: "", // Not provided by API
    parent_id: ctx.parent_id,
    type: mapContextTypeToDocType(ctx.type),
    title: ctx.name,
    icon: ctx.icon ? { type: "emoji", value: ctx.icon } : null,
    cover: ctx.cover_image || null,
    content: ctx.blocks ? mapContextBlocksToDocBlocks(ctx.blocks) : [],
    properties: (ctx.structured_data || {}) as Record<string, PropertyValue>,
    is_template: ctx.is_template ?? false,
    is_favorite: ctx.properties?.is_favorite === true,
    is_archived: ctx.is_archived,
    is_public: ctx.is_public ?? false,
    share_id: ctx.share_id ?? null,
    created_at: ctx.created_at,
    updated_at: ctx.updated_at,
    created_by: "", // Not provided by API
    last_edited_by: "", // Not provided by API
  };
}

/**
 * Map Context blocks to Document blocks
 */
function mapContextBlocksToDocBlocks(blocks: ContextBlock[]): Block[] {
  return blocks.map((block) => ({
    id: block.id,
    type: block.type as BlockType,
    content: block.content ? parseBlockContent(block.content) : [],
    properties: block.properties || {},
    children: block.children ? mapContextBlocksToDocBlocks(block.children) : [],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }));
}

/**
 * Parse block content string to RichText array
 */
function parseBlockContent(content: string | null): RichText[] {
  if (!content) return [];
  return parseInlineMarkdownToRichText(content);
}

/**
 * Map Document blocks to Context blocks for saving
 */
function mapDocBlocksToContextBlocks(blocks: Block[]): ContextBlock[] {
  return blocks.map((block) => ({
    id: block.id,
    type: block.type,
    content: extractPlainText(block.content),
    properties: block.properties as Record<string, unknown>,
    children: block.children
      ? mapDocBlocksToContextBlocks(block.children)
      : undefined,
  })) as ContextBlock[];
}

/**
 * Extract plain text from RichText array
 */
function extractPlainText(content: RichText[] | string | null): string | null {
  const text = extractRichTextPlainText(content);
  return text || null;
}

// ============================================================================
// OptimalOS Filesystem Adapter
// ============================================================================

/**
 * Build fetch headers matching the pattern used by the optimal store.
 */
function buildOptimalHeaders(
  includeContentType = false,
): Record<string, string> {
  const headers: Record<string, string> = {};
  const csrf = getCSRFToken();
  if (csrf) headers["X-CSRF-Token"] = csrf;
  if (includeContentType) headers["Content-Type"] = "application/json";
  return headers;
}

/**
 * Determine whether a document id is an OptimalOS filesystem path.
 * File paths contain a "/" or end in ".md".
 */
function isOptimalPath(id: string): boolean {
  return id.includes("/") || id.endsWith(".md");
}

/**
 * Split a full path like "04-ai-masters/community/file.md" into
 * { slug: "04-ai-masters", relativePath: "community/file.md" }.
 */
function splitOptimalPath(fullPath: string): {
  slug: string;
  relativePath: string;
} {
  const slash = fullPath.indexOf("/");
  if (slash === -1) {
    // Treat as a top-level slug with no sub-path
    return { slug: fullPath, relativePath: "" };
  }
  return {
    slug: fullPath.slice(0, slash),
    relativePath: fullPath.slice(slash + 1),
  };
}

/**
 * Encode a relative path for use in a URL wildcard segment.
 * Encodes each segment individually, then joins with "/".
 */
function encodeRelativePath(relativePath: string): string {
  return relativePath
    .split("/")
    .map((seg) => encodeURIComponent(seg))
    .join("/");
}

/**
 * Convert a file's base name (without extension) to a readable title.
 * e.g. "ed-free-call-topics" → "Ed Free Call Topics"
 */
function fileNameToTitle(name: string): string {
  return name
    .replace(/\.md$/i, "")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

/**
 * Walk a FileEntry tree and collect all non-directory entries as DocumentMeta,
 * using full paths relative to the OptimalOS root (slug/...path).
 */
function flattenFileTree(
  entries: FileEntry[],
  nodeSlug: string,
  parentId: string | null,
  now: string,
): DocumentMeta[] {
  const result: DocumentMeta[] = [];

  for (const entry of entries) {
    const fullPath = entry.path.startsWith(nodeSlug + "/")
      ? entry.path
      : `${nodeSlug}/${entry.path}`;

    if (entry.is_dir) {
      // Directories become "folder" type documents
      const dirMeta: DocumentMeta = {
        id: fullPath,
        parent_id: parentId,
        type: "folder",
        title: fileNameToTitle(entry.name),
        icon: null,
        is_favorite: false,
        is_archived: false,
        updated_at: now,
        children_count: entry.children ? entry.children.length : 0,
      };
      result.push(dirMeta);

      if (entry.children && entry.children.length > 0) {
        result.push(
          ...flattenFileTree(entry.children, nodeSlug, fullPath, now),
        );
      }
    } else if (entry.name.endsWith(".md")) {
      result.push({
        id: fullPath,
        parent_id: parentId,
        type: "document",
        title: fileNameToTitle(entry.name),
        icon: null,
        is_favorite: false,
        is_archived: false,
        updated_at: now,
        children_count: 0,
      });
    }
  }

  return result;
}

/**
 * Convert a markdown string into an array of Blocks the editor can render.
 *
 * Strategy:
 *   - ATX headings (#, ##, ###) → heading_1/2/3 blocks
 *   - Fenced code blocks → code blocks
 *   - Blank-line-separated paragraphs → paragraph blocks
 *   - Lines starting with "- " or "* " → bulleted_list
 *   - Lines starting with a digit and ". " → numbered_list
 *   - Lines starting with "- [ ]" / "- [x]" → to_do
 *   - Lines starting with "> " → quote
 *   - "---" / "***" / "___" → divider
 *
 * This is intentionally simple — the editor can re-parse richer content later.
 */
function markdownToBlocks(markdown: string): Block[] {
  const now = new Date().toISOString();
  const blocks: Block[] = [];

  const makeBlock = (
    type: BlockType,
    text: string,
    extra?: Record<string, unknown>,
  ): Block => ({
    id: crypto.randomUUID(),
    type,
    content: parseInlineMarkdownToRichText(text),
    properties: extra ?? {},
    children: [],
    created_at: now,
    updated_at: now,
  });

  // Pre-process: collapse fenced code blocks into single logical lines
  // We process the raw text line by line with a small state machine.
  const lines = markdown.split("\n");
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Fenced code block
    if (line.startsWith("```")) {
      const lang = line.slice(3).trim() || "plaintext";
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // consume closing ```
      blocks.push({
        ...makeBlock("code", codeLines.join("\n")),
        properties: { language: lang },
      });
      continue;
    }

    // ATX headings
    const h3 = line.match(/^### (.+)/);
    if (h3) {
      blocks.push(makeBlock("heading_3", h3[1].trim()));
      i++;
      continue;
    }

    const h2 = line.match(/^## (.+)/);
    if (h2) {
      blocks.push(makeBlock("heading_2", h2[1].trim()));
      i++;
      continue;
    }

    const h1 = line.match(/^# (.+)/);
    if (h1) {
      blocks.push(makeBlock("heading_1", h1[1].trim()));
      i++;
      continue;
    }

    // Divider
    if (/^(---+|\*\*\*+|___+)\s*$/.test(line)) {
      blocks.push(makeBlock("divider", ""));
      i++;
      continue;
    }

    // Blockquote
    const quote = line.match(/^> (.+)/);
    if (quote) {
      blocks.push(makeBlock("quote", quote[1].trim()));
      i++;
      continue;
    }

    // To-do
    const todoChecked = line.match(/^- \[x\] (.+)/i);
    if (todoChecked) {
      blocks.push({
        ...makeBlock("to_do", todoChecked[1].trim()),
        properties: { checked: true },
      });
      i++;
      continue;
    }
    const todoUnchecked = line.match(/^- \[ \] (.+)/);
    if (todoUnchecked) {
      blocks.push({
        ...makeBlock("to_do", todoUnchecked[1].trim()),
        properties: { checked: false },
      });
      i++;
      continue;
    }

    // Bulleted list
    const bullet = line.match(/^[-*] (.+)/);
    if (bullet) {
      blocks.push(makeBlock("bulleted_list", bullet[1].trim()));
      i++;
      continue;
    }

    // Numbered list
    const numbered = line.match(/^\d+\. (.+)/);
    if (numbered) {
      blocks.push(makeBlock("numbered_list", numbered[1].trim()));
      i++;
      continue;
    }

    // Blank line — skip
    if (line.trim() === "") {
      i++;
      continue;
    }

    // Default: paragraph
    blocks.push(makeBlock("paragraph", line.trim()));
    i++;
  }

  return blocks.length > 0 ? blocks : [makeBlock("paragraph", "")];
}

/**
 * Convert Document blocks back to a markdown string for saving to the filesystem.
 * Re-uses the existing blocksToMarkdown helper defined later in this file,
 * but we need a forward-compatible version here.
 */
function blocksToMarkdownInternal(blocks: Block[]): string {
  return blocks
    .map((block) => {
      const text =
        block.type === "code"
          ? extractRichTextPlainText(block.content)
          : richTextToMarkdown(block.content);

      switch (block.type) {
        case "heading_1":
          return `# ${text}`;
        case "heading_2":
          return `## ${text}`;
        case "heading_3":
          return `### ${text}`;
        case "bulleted_list":
          return `- ${text}`;
        case "numbered_list":
          return `1. ${text}`;
        case "to_do": {
          const checked = block.properties?.checked ? "x" : " ";
          return `- [${checked}] ${text}`;
        }
        case "quote":
          return `> ${text}`;
        case "code": {
          const lang =
            (block.properties?.language as string | undefined) ?? "plaintext";
          return `\`\`\`${lang}\n${text}\n\`\`\``;
        }
        case "divider":
          return "---";
        default:
          return text;
      }
    })
    .join("\n\n");
}

// ============================================================================
// OptimalOS Filesystem API calls
// ============================================================================

/**
 * Fetch all OptimalOS nodes and their file trees, mapping each .md file to a
 * DocumentMeta. Nodes become top-level folder entries; files/subdirectories
 * become their children.
 *
 * Returns null if the endpoint is unreachable (allows graceful fallback).
 */
export async function fetchOptimalDocuments(): Promise<DocumentMeta[] | null> {
  const base = getApiBaseUrl();
  const headers = buildOptimalHeaders();
  const now = new Date().toISOString();

  let nodes: OptimalNode[];
  try {
    const url = `${base}/optimal/nodes`;
    const res = await fetch(url, {
      method: "GET",
      headers,
      credentials: "include",
      signal: AbortSignal.timeout(8000),
    });
    if (!res.ok) return null;
    const data: { nodes?: OptimalNode[] } = await res.json();
    nodes = Array.isArray(data.nodes) ? data.nodes : [];
  } catch (err) {
    console.error("[OptimalOS] Fetch failed:", err);
    return null;
  }

  if (nodes.length === 0) return [];

  const allMetas: DocumentMeta[] = [];

  // Fetch file trees for all nodes in parallel
  await Promise.all(
    nodes.map(async (node) => {
      // Top-level node entry (acts as a folder)
      const nodeMeta: DocumentMeta = {
        id: node.slug,
        parent_id: null,
        type: "folder",
        title: node.name,
        icon: null,
        is_favorite: false,
        is_archived: false,
        updated_at: now,
        children_count: 0,
      };
      allMetas.push(nodeMeta);

      try {
        const res = await fetch(
          `${base}/optimal/nodes/${encodeURIComponent(node.slug)}/files`,
          {
            method: "GET",
            headers,
            credentials: "include",
            signal: AbortSignal.timeout(8000),
          },
        );
        if (!res.ok) return;

        const data: { files?: FileEntry[] } = await res.json();
        const files = Array.isArray(data.files) ? data.files : [];

        const children = flattenFileTree(files, node.slug, node.slug, now);
        nodeMeta.children_count = children.filter(
          (c) => c.parent_id === node.slug,
        ).length;
        allMetas.push(...children);
      } catch {
        // Node file tree unavailable — keep the node stub in the list
      }
    }),
  );

  return allMetas;
}

/**
 * Fetch a single .md file from the OptimalOS filesystem and return it as a
 * Document with content blocks parsed from markdown.
 *
 * @param filePath  Full path e.g. "04-ai-masters/community/ed-free-call-topics.md"
 */
export async function fetchOptimalDocument(
  filePath: string,
): Promise<Document | null> {
  const { slug, relativePath } = splitOptimalPath(filePath);
  if (!relativePath) return null;
  // Folders cannot be opened as documents — only .md files
  if (!relativePath.endsWith(".md")) return null;

  const base = getApiBaseUrl();
  const encoded = encodeRelativePath(relativePath);

  try {
    const res = await fetch(
      `${base}/optimal/nodes/${encodeURIComponent(slug)}/file/${encoded}`,
      {
        method: "GET",
        headers: buildOptimalHeaders(),
        credentials: "include",
        signal: AbortSignal.timeout(8000),
      },
    );
    if (!res.ok) return null;

    const data: { content?: string; path?: string } = await res.json();
    const content = data.content ?? "";
    const now = new Date().toISOString();

    return {
      id: filePath,
      workspace_id: "",
      parent_id: slug,
      type: "document",
      title: fileNameToTitle(relativePath.split("/").pop() ?? relativePath),
      icon: null,
      cover: null,
      content: markdownToBlocks(content),
      properties: {},
      is_template: false,
      is_favorite: false,
      is_archived: false,
      is_public: false,
      share_id: null,
      created_at: now,
      updated_at: now,
      created_by: "",
      last_edited_by: "",
    };
  } catch {
    return null;
  }
}

/**
 * Save an edited Document back to the OptimalOS filesystem as markdown.
 *
 * @param filePath  Full path e.g. "04-ai-masters/community/ed-free-call-topics.md"
 * @param doc       The Document whose content blocks should be serialised
 */
export async function saveOptimalDocument(
  filePath: string,
  doc: Document,
): Promise<boolean> {
  const { slug, relativePath } = splitOptimalPath(filePath);
  if (!relativePath) return false;

  const base = getApiBaseUrl();
  const encoded = encodeRelativePath(relativePath);
  const markdown = `# ${doc.title}\n\n${blocksToMarkdownInternal(doc.content)}`;

  try {
    const res = await fetch(
      `${base}/optimal/nodes/${encodeURIComponent(slug)}/file/${encoded}`,
      {
        method: "PUT",
        headers: buildOptimalHeaders(true),
        credentials: "include",
        signal: AbortSignal.timeout(10000),
        body: JSON.stringify({ content: markdown }),
      },
    );
    return res.ok;
  } catch {
    return false;
  }
}

// ============================================================================
// Document CRUD Operations
// ============================================================================

export async function fetchDocuments(
  parentId?: string | null,
): Promise<DocumentMeta[]> {
  documentsStore.setLoading(true);
  try {
    // Try OptimalOS filesystem first (only at the root level — no parentId filter)
    if (!parentId) {
      const optimalDocs = await fetchOptimalDocuments();
      if (optimalDocs !== null) {
        // OptimalOS is configured — use it even if empty (don't fall through to contexts API)
        documentsStore.setDocumentMetas(optimalDocs);
        return optimalDocs;
      }
    }

    // Fallback: contexts API (only when OptimalOS is not configured / returned null)
    try {
      const contexts = await contextsApi.getContexts({
        parentId: parentId || undefined,
      });
      const documents = contexts.map(mapContextToDocMeta);
      documentsStore.setDocumentMetas(documents);
      return documents;
    } catch {
      // Contexts API also failed — return empty list
      documentsStore.setDocumentMetas([]);
      return [];
    }
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to fetch documents";
    documentsStore.setError(message);
    throw error;
  } finally {
    documentsStore.setLoading(false);
  }
}

export async function fetchDocument(id: string): Promise<Document> {
  activeDocumentStore.setLoading(true);
  try {
    const context = await contextsApi.getContext(id);
    const document = mapContextToDocument(context);
    documentsStore.setDocument(document);
    return document;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to fetch document";
    activeDocumentStore.setError(message);
    throw error;
  } finally {
    activeDocumentStore.setLoading(false);
  }
}

export async function createDocument(params: {
  title: string;
  type?: DocumentType;
  parent_id?: string | null;
  icon?: string | null;
  content?: Block[];
}): Promise<Document> {
  try {
    const context = await contextsApi.createContext({
      name: params.title || "Untitled",
      type: "document",
      parent_id: params.parent_id || undefined,
      icon: typeof params.icon === "string" ? params.icon : undefined,
      blocks: params.content
        ? mapDocBlocksToContextBlocks(params.content)
        : [{ id: crypto.randomUUID(), type: "paragraph", content: "" }],
    });

    const document = mapContextToDocument(context);
    documentsStore.setDocument(document);

    // Expand parent if exists
    if (params.parent_id) {
      treeStore.setExpanded(params.parent_id, true);
    }

    return document;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to create document";
    documentsStore.setError(message);
    throw error;
  }
}

export async function updateDocument(
  id: string,
  updates: Partial<Document>,
): Promise<Document> {
  activeDocumentStore.setSaving(true);
  try {
    // Map document updates to context updates
    const contextUpdates: Parameters<typeof contextsApi.updateContext>[1] = {};

    if (updates.title !== undefined) contextUpdates.name = updates.title;
    if (updates.icon !== undefined) {
      contextUpdates.icon =
        typeof updates.icon === "object" && updates.icon?.value
          ? updates.icon.value
          : (updates.icon as string | undefined);
    }
    if (updates.cover !== undefined)
      contextUpdates.cover_image = updates.cover || undefined;
    if (updates.parent_id !== undefined)
      contextUpdates.parent_id = updates.parent_id;
    if (updates.is_archived !== undefined)
      contextUpdates.is_archived = updates.is_archived;
    if (updates.is_template !== undefined)
      contextUpdates.is_template = updates.is_template;
    if (updates.content !== undefined) {
      contextUpdates.blocks = mapDocBlocksToContextBlocks(updates.content);
    }
    if (updates.is_favorite !== undefined) {
      contextUpdates.properties = { is_favorite: updates.is_favorite };
    }

    const context = await contextsApi.updateContext(id, contextUpdates);
    const document = mapContextToDocument(context);
    documentsStore.updateDocument(id, document);
    activeDocumentStore.setLastSaved(new Date().toISOString());
    return document;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to update document";
    activeDocumentStore.setError(message);
    throw error;
  } finally {
    activeDocumentStore.setSaving(false);
  }
}

export async function deleteDocument(
  id: string,
  permanent = false,
): Promise<void> {
  try {
    if (permanent) {
      await contextsApi.deleteContext(id);
      documentsStore.removeDocument(id);
    } else {
      await contextsApi.archiveContext(id);
      documentsStore.updateDocument(id, { is_archived: true });
    }
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to delete document";
    documentsStore.setError(message);
    throw error;
  }
}

export async function restoreDocument(id: string): Promise<void> {
  try {
    await contextsApi.unarchiveContext(id);
    documentsStore.updateDocument(id, { is_archived: false });
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to restore document";
    documentsStore.setError(message);
    throw error;
  }
}

export async function moveDocument(
  id: string,
  newParentId: string | null,
): Promise<void> {
  try {
    await contextsApi.updateContext(id, { parent_id: newParentId });
    documentsStore.updateDocument(id, { parent_id: newParentId });

    // Expand new parent
    if (newParentId) {
      treeStore.setExpanded(newParentId, true);
    }
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to move document";
    documentsStore.setError(message);
    throw error;
  }
}

export async function duplicateDocument(id: string): Promise<Document> {
  try {
    const context = await contextsApi.duplicateContext(id);
    const document = mapContextToDocument(context);
    documentsStore.setDocument(document);
    return document;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to duplicate document";
    documentsStore.setError(message);
    throw error;
  }
}

export async function toggleFavorite(id: string): Promise<void> {
  const doc = documentsStore.getDocument(id);
  if (!doc) return;

  await updateDocument(id, { is_favorite: !doc.is_favorite });
}

// ============================================================================
// Block Operations
// ============================================================================

export function createEmptyParagraph(): Block {
  return {
    id: crypto.randomUUID(),
    type: "paragraph",
    content: [],
    properties: {},
    children: [],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
}

export function createBlock(type: BlockType, content?: string): Block {
  const richText: RichText[] = content
    ? [
        {
          type: "text",
          text: { content, link: null },
          annotations: {
            bold: false,
            italic: false,
            strikethrough: false,
            underline: false,
            code: false,
            color: "default",
          },
          plain_text: content,
          href: null,
        },
      ]
    : [];

  // Initialize type-specific properties
  let properties: Record<string, unknown> = {};

  switch (type) {
    case "to_do":
      properties = { checked: false };
      break;
    case "toggle":
      properties = { expanded: false };
      break;
    case "callout":
      properties = { icon: "Lightbulb" };
      break;
    case "code":
      properties = { language: "plaintext" };
      break;
    case "table":
      properties = {
        tableData: {
          rows: [
            ["", "", ""],
            ["", "", ""],
          ],
          headerRow: true,
        },
      };
      break;
    case "image":
      properties = { url: null, caption: "" };
      break;
    case "bookmark":
      properties = { url: null, title: "", description: "" };
      break;
    default:
      properties = {};
  }

  return {
    id: crypto.randomUUID(),
    type,
    content: richText,
    properties,
    children: [],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
}

// ============================================================================
// Search Operations
// ============================================================================

export async function searchDocuments(query: string): Promise<DocumentMeta[]> {
  if (!query.trim()) return [];

  try {
    const contexts = await contextsApi.getContexts({ search: query });
    return contexts.map(mapContextToDocMeta);
  } catch (error) {
    console.error("Search error:", error);
    return [];
  }
}

// ============================================================================
// Navigation Helpers
// ============================================================================

export function openDocument(id: string) {
  activeDocumentStore.setActiveDocument(id);
}

export function closeDocument() {
  activeDocumentStore.setActiveDocument(null);
}

export async function openAndFetchDocument(id: string): Promise<Document> {
  activeDocumentStore.setActiveDocument(id);

  // Route filesystem paths to the OptimalOS adapter
  if (isOptimalPath(id)) {
    const doc = await fetchOptimalDocument(id);
    if (doc) {
      documentsStore.setDocument(doc);
      return doc;
    }
    // Optimal path but no document (folder or missing) — return empty placeholder
    // Don't fall through to contexts API which will error
    const now = new Date().toISOString();
    const placeholder: Document = {
      id,
      workspace_id: "",
      parent_id: null,
      type: "document",
      title: id.split("/").pop()?.replace(".md", "").replace(/-/g, " ") ?? id,
      icon: null,
      cover: null,
      content: [],
      properties: {},
      is_template: false,
      is_favorite: false,
      is_archived: false,
      is_public: false,
      share_id: null,
      created_at: now,
      updated_at: now,
      created_by: "",
      last_edited_by: "",
    };
    documentsStore.setDocument(placeholder);
    return placeholder;
  }

  return await fetchDocument(id);
}

// ============================================================================
// Profile Operations
// ============================================================================

export type ProfileType = "person" | "business" | "project";

export async function fetchProfiles(
  type?: ProfileType,
): Promise<DocumentMeta[]> {
  documentsStore.setLoading(true);
  try {
    const contexts = await contextsApi.getContexts({
      type: type || undefined,
    });

    // Filter by profile types if no specific type given
    const profileTypes: ProfileType[] = ["person", "business", "project"];
    const filtered = type
      ? contexts
      : contexts.filter((ctx) =>
          profileTypes.includes(ctx.type as ProfileType),
        );

    const documents = filtered.map(mapContextToDocMeta);
    return documents;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to fetch profiles";
    documentsStore.setError(message);
    throw error;
  } finally {
    documentsStore.setLoading(false);
  }
}

export async function createProfile(params: {
  name: string;
  type: ProfileType;
  description?: string;
  icon?: string | null;
  properties?: Record<string, unknown>;
}): Promise<Document> {
  try {
    const context = await contextsApi.createContext({
      name: params.name || "Untitled",
      type: params.type,
      icon: typeof params.icon === "string" ? params.icon : undefined,
      properties: params.properties,
      blocks: params.description
        ? [
            {
              id: crypto.randomUUID(),
              type: "paragraph",
              content: params.description,
            },
          ]
        : [{ id: crypto.randomUUID(), type: "paragraph", content: "" }],
    });

    const document = mapContextToDocument(context);
    documentsStore.setDocument(document);
    return document;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Failed to create profile";
    documentsStore.setError(message);
    throw error;
  }
}

// Default icons for profile types (SVG icon names from PageIconPicker)
export const defaultProfileIcons: Record<ProfileType, string> = {
  person: "User",
  business: "Building",
  project: "FolderKanban",
};

// ============================================================================
// Share Operations
// ============================================================================

export async function enableSharing(
  id: string,
): Promise<{ share_id: string; share_url: string }> {
  const result = await contextsApi.enableContextSharing(id);
  // Re-fetch to update local state with share fields
  await fetchDocument(id);
  return result;
}

export async function disableSharing(id: string): Promise<void> {
  await contextsApi.disableContextSharing(id);
  await fetchDocument(id);
}

// ============================================================================
// Export Operations
// ============================================================================

function blocksToMarkdown(blocks: Block[]): string {
  return blocks
    .map((block) => {
      const text =
        block.type === "code"
          ? extractRichTextPlainText(block.content)
          : richTextToMarkdown(block.content);

      switch (block.type) {
        case "heading_1":
          return `# ${text}`;
        case "heading_2":
          return `## ${text}`;
        case "heading_3":
          return `### ${text}`;
        case "bulleted_list":
          return `- ${text}`;
        case "numbered_list":
          return `1. ${text}`;
        case "to_do": {
          const checked = block.properties?.checked ? "x" : " ";
          return `- [${checked}] ${text}`;
        }
        case "quote":
          return `> ${text}`;
        case "code":
          return `\`\`\`\n${text}\n\`\`\``;
        case "divider":
          return "---";
        default:
          return text;
      }
    })
    .join("\n\n");
}

export function exportDocumentAsMarkdown(doc: Document): void {
  const title = doc.title || "Untitled";
  const md = `# ${title}\n\n${blocksToMarkdown(doc.content)}`;
  downloadFile(`${title}.md`, md, "text/markdown");
}

export function exportDocumentAsJSON(doc: Document): void {
  const title = doc.title || "Untitled";
  const data = {
    title: doc.title,
    icon: doc.icon,
    cover: doc.cover,
    content: doc.content,
    properties: doc.properties,
    created_at: doc.created_at,
    updated_at: doc.updated_at,
  };
  downloadFile(
    `${title}.json`,
    JSON.stringify(data, null, 2),
    "application/json",
  );
}

function downloadFile(filename: string, content: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
