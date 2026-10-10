/**
 * Adapter: converts OptimalOS nodes into NodeTree format
 * for the existing 3D graph and building visualization components.
 *
 * Parses `cross_ref` from frontmatter to build edges between nodes.
 */

import type { OptimalNode } from "./optimal";
import type { NodeTree, NodeType, NodeHealth } from "$lib/api/nodes/types";

/** Map OptimalOS node types to BusinessOS NodeType enum */
function mapNodeType(optType: string): NodeType {
  const typeMap: Record<string, NodeType> = {
    person: "business",
    entity: "business",
    "domain:product": "project",
    "operation:program": "operational",
    "operation:research": "operational",
    "operation:network": "operational",
    "unit:community": "operational",
    "domain:media": "project",
    inbox: "learning",
    registry: "operational",
    "cross-cutting": "business",
    "layer:command": "operational",
    "command-center": "operational",
  };
  return typeMap[optType] || "business";
}

/** Extract cross_ref slugs from context_md frontmatter */
function extractCrossRefs(contextMD: string): string[] {
  if (!contextMD.startsWith("---")) return [];

  const endIdx = contextMD.indexOf("\n---", 3);
  if (endIdx < 0) return [];

  const frontmatter = contextMD.slice(3, endIdx);
  const crossRefLine = frontmatter
    .split("\n")
    .find((l) => l.trim().startsWith("cross_ref:"));

  if (!crossRefLine) return [];

  // Parse YAML array: cross_ref: [company, department, project]
  const match = crossRefLine.match(/\[([^\]]*)\]/);
  if (!match) return [];

  return match[1]
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

/** Map a slug fragment to the full node slug (e.g. "miosa" → "02-miosa") */
function resolveSlug(
  fragment: string,
  slugMap: Map<string, string>,
): string | null {
  // Direct match
  if (slugMap.has(fragment)) return slugMap.get(fragment)!;
  // Try with common prefixes stripped
  for (const [key, fullSlug] of slugMap) {
    const namePart = key.replace(/^\d+-/, "");
    if (namePart === fragment) return fullSlug;
  }
  return null;
}

/**
 * Convert OptimalOS nodes into NodeTree[] for the graph components.
 * Root nodes = the 13 numbered folders.
 * Edges = cross_ref relationships from frontmatter.
 */
export function toNodeTrees(optNodes: OptimalNode[]): NodeTree[] {
  // Build slug lookup map: "miosa" → "02-miosa", "02-miosa" → "02-miosa"
  const slugMap = new Map<string, string>();
  for (const n of optNodes) {
    slugMap.set(n.slug, n.slug);
    const namePart = n.slug.replace(/^\d+-/, "");
    if (namePart) slugMap.set(namePart, n.slug);
  }

  const trees: NodeTree[] = optNodes.map((n) => ({
    id: n.slug,
    user_id: "local",
    parent_id: null,
    context_id: null,
    name: n.name,
    type: mapNodeType(n.type),
    health: (n.has_signals ? "healthy" : "needs_attention") as NodeHealth,
    purpose: null,
    current_status: null,
    this_week_focus: null,
    decision_queue: null,
    delegation_ready: null,
    is_active: true,
    is_archived: false,
    sort_order: parseInt(n.slug.split("-")[0]) || 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    children: [] as NodeTree[],
    children_count: n.signal_count,
  }));

  // Build cross_ref edges as parent-child for graph layout
  // We add referenced nodes as "virtual children" so the graph draws edges
  const treeMap = new Map<string, NodeTree>();
  for (const t of trees) treeMap.set(t.id, t);

  for (const n of optNodes) {
    const refs = extractCrossRefs(n.context_md);
    const parentTree = treeMap.get(n.slug);
    if (!parentTree) continue;

    for (const ref of refs) {
      const resolvedSlug = resolveSlug(ref, slugMap);
      if (resolvedSlug && resolvedSlug !== n.slug) {
        // Set parent_id on the referenced node to create a visual edge
        const childTree = treeMap.get(resolvedSlug);
        if (childTree && !childTree.parent_id) {
          childTree.parent_id = n.slug;
        }
      }
    }
  }

  return trees;
}

/** Get edges as pairs of [sourceSlug, targetSlug] for custom rendering */
export function getEdges(optNodes: OptimalNode[]): Array<[string, string]> {
  const slugMap = new Map<string, string>();
  for (const n of optNodes) {
    slugMap.set(n.slug, n.slug);
    const namePart = n.slug.replace(/^\d+-/, "");
    if (namePart) slugMap.set(namePart, n.slug);
  }

  const edges: Array<[string, string]> = [];
  const seen = new Set<string>();

  for (const n of optNodes) {
    const refs = extractCrossRefs(n.context_md);
    for (const ref of refs) {
      const resolved = resolveSlug(ref, slugMap);
      if (resolved && resolved !== n.slug) {
        const key = [n.slug, resolved].sort().join("↔");
        if (!seen.has(key)) {
          seen.add(key);
          edges.push([n.slug, resolved]);
        }
      }
    }
  }

  return edges;
}
