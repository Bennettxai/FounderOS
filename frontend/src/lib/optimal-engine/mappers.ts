import type { OptimalNode } from "$lib/stores/optimal";

export interface SDKTreeNode {
  slug: string;
  label: string;
  signalCount?: number;
  children?: SDKTreeNode[];
}

export function toSDKTreeNodes(nodes: OptimalNode[]): SDKTreeNode[] {
  return nodes.map((n) => ({
    slug: n.slug,
    label: n.name || n.slug,
    signalCount: n.signal_count,
  }));
}

export interface SDKGraphNode {
  id: string;
  name: string;
  type: string;
  connections: number;
}

export function toSDKGraphNodes(nodes: OptimalNode[]): {
  nodes: SDKGraphNode[];
  edges: { source: string; target: string }[];
} {
  const graphNodes = nodes.map((n) => ({
    id: n.slug,
    name: n.name || n.slug,
    type: n.type || "node",
    connections: n.signal_count,
  }));
  return { nodes: graphNodes, edges: [] };
}
