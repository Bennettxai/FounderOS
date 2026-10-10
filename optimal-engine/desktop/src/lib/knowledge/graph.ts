/**
 * Builds the unified graph data shape for the /graph route:
 *   workspace nodes  (kind=node,   color=type-based)
 *   signals          (kind=signal, color=genre-based)
 *   entities         (kind=entity, color=entity-type-based)
 *
 * Edges:
 *   node → signal    (signal lives in this node)
 *   signal → entity  (signal mentions this entity)
 *   node → node      (parent → child in the workspace tree)
 *
 * The component uses PixiJS for rendering + d3-force for layout, matching
 * the visual language of the Pages module.
 */

import type { WorkspaceNode } from "$lib/api/workspace";

export type GKind = "node" | "signal" | "entity";

export interface GNode {
  id: string;
  label: string;
  kind: GKind;
  // Drives colour + sizing. For `node` this is the node.kind (org/person/…);
  // for `signal` it's the genre; for `entity` it's the entity type.
  sub: string;
  connections: number;
  // Signal-specific
  node_slug?: string;
}

export interface GEdge {
  source: string;
  target: string;
  relation: "contains" | "mentions" | "parent";
}

export interface GraphData {
  nodes: GNode[];
  edges: GEdge[];
}

export interface OptimalGraphResponse {
  entities: { name: string; type: string; connections: number }[];
  edges: {
    source: string;
    target: string;
    relation: string;
    weight: number;
  }[];
  stats: {
    entity_count: number;
    edge_count: number;
    edge_types: Record<string, number>;
  };
}

export interface NodeFilesResponse {
  files: {
    name: string;
    path: string;
    is_dir: boolean;
    size: number;
    genre: string | null;
    modified_at: string | null;
  }[];
}

/** Pull everything the graph needs in parallel and fold it into one shape. */
export async function loadGraph(): Promise<GraphData> {
  const [workspaceRes, optimalRes] = await Promise.all([
    fetch("/api/workspace").then((r) => r.json()) as Promise<{
      nodes: WorkspaceNode[];
    }>,
    fetch("/api/optimal/graph").then((r) =>
      r.json(),
    ) as Promise<OptimalGraphResponse>,
  ]);

  const workspaceNodes = workspaceRes.nodes;
  const entityData = optimalRes.entities;
  const entityEdges = optimalRes.edges;

  // Fetch per-node signal lists in parallel.
  const signalLists = await Promise.all(
    workspaceNodes.map(async (n) => {
      const res = (await fetch(
        `/api/optimal/nodes/${encodeURIComponent(n.slug)}/files`,
      ).then((r) => r.json())) as NodeFilesResponse;
      return { node: n, files: res.files };
    }),
  );

  const nodes: GNode[] = [];
  const edges: GEdge[] = [];

  // Workspace nodes
  for (const n of workspaceNodes) {
    nodes.push({
      id: `node:${n.slug}`,
      label: n.name,
      kind: "node",
      sub: n.kind,
      connections: n.signal_count,
    });
    if (n.parent_id) {
      // parent_id is the nodes.id, but our node ids here use slugs — walk
      // the list to find the parent slug.
      const parent = workspaceNodes.find((p) => p.id === n.parent_id);
      if (parent) {
        edges.push({
          source: `node:${parent.slug}`,
          target: `node:${n.slug}`,
          relation: "parent",
        });
      }
    }
  }

  // Signals under each node
  for (const { node, files } of signalLists) {
    for (const f of files) {
      const sigId = `signal:${f.path}`;
      nodes.push({
        id: sigId,
        label: f.name,
        kind: "signal",
        sub: f.genre ?? "note",
        connections: 0,
        node_slug: node.slug,
      });
      edges.push({
        source: `node:${node.slug}`,
        target: sigId,
        relation: "contains",
      });
    }
  }

  // Entities (collapse into one node per entity name across the tenant)
  const entityIds = new Set<string>();
  for (const e of entityData) {
    const eid = `entity:${e.name}`;
    if (!entityIds.has(eid)) {
      entityIds.add(eid);
      nodes.push({
        id: eid,
        label: e.name,
        kind: "entity",
        sub: e.type,
        connections: e.connections,
      });
    }
  }

  // Entity-entity co-occurrence edges
  for (const e of entityEdges) {
    edges.push({
      source: `entity:${e.source}`,
      target: `entity:${e.target}`,
      relation: "mentions",
    });
  }

  return { nodes, edges };
}
