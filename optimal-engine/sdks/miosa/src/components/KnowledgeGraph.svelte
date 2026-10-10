<script lang="ts">
  import type { EngineContext } from '../engine.js';

  interface GraphNode {
    id: string;
    name: string;
    type: string;
    connections: number;
    x: number;
    y: number;
    vx: number;
    vy: number;
  }

  interface GraphEdge {
    source: string;
    target: string;
    type?: string;
  }

  interface GraphData {
    nodes: GraphNode[];
    edges: GraphEdge[];
  }

  interface NodeSelectPayload {
    id: string;
    name: string;
    type: string;
  }

  interface Props {
    engine: EngineContext;
    onselect?: (node: NodeSelectPayload) => void;
    height?: number;
  }

  let { engine, onselect, height = 400 }: Props = $props();

  let svgEl: SVGSVGElement;
  let containerEl: HTMLDivElement;

  let nodes = $state<GraphNode[]>([]);
  let edges = $state<GraphEdge[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  let hoveredId = $state<string | null>(null);
  let draggingId = $state<string | null>(null);
  let dragOffsetX = $state(0);
  let dragOffsetY = $state(0);

  // Pan/zoom state
  let panX = $state(0);
  let panY = $state(0);
  let zoom = $state(1);
  let isPanning = $state(false);
  let panStartX = $state(0);
  let panStartY = $state(0);
  let panStartPanX = $state(0);
  let panStartPanY = $state(0);

  let width = $state(600);

  $effect(() => {
    if (containerEl) {
      const ro = new ResizeObserver((entries) => {
        width = entries[0]?.contentRect.width ?? 600;
      });
      ro.observe(containerEl);
      return () => ro.disconnect();
    }
  });

  function nodeColor(type: string): string {
    switch (type) {
      case 'person': return 'var(--daccent, #7ea8ff)';
      case 'concept': return '#bb7eff';
      case 'product': return '#fc9e6c';
      case 'organization': return '#5fcfd4';
      default: return 'var(--dt4, #555)';
    }
  }

  function nodeRadius(connections: number): number {
    return Math.max(7, Math.min(18, 7 + connections * 1.5));
  }

  function edgeColor(type?: string): string {
    switch (type) {
      case 'updates': return '#7ea8ff';
      case 'extends': return '#7be3a3';
      case 'derives': return '#bb7eff';
      case 'contradicts': return '#f88';
      default: return 'rgba(255,255,255,0.2)';
    }
  }

  async function loadGraph() {
    loading = true;
    error = null;
    try {
      const ws = engine.getWorkspace();
      const raw = await engine.client.http.get<GraphData>(`/api/graph?workspace=${encodeURIComponent(ws)}`);
      const w = width || 600;
      const h = height;
      // Position nodes randomly in center region
      nodes = (raw.nodes ?? []).map((n: GraphNode) => ({
        ...n,
        connections: n.connections ?? 1,
        x: w / 2 + (Math.random() - 0.5) * 200,
        y: h / 2 + (Math.random() - 0.5) * 200,
        vx: 0,
        vy: 0,
      }));
      edges = raw.edges ?? [];
      runSimulation();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  // Simple spring force simulation — no d3
  function runSimulation() {
    const REPULSION = 2500;
    const ATTRACTION = 0.03;
    const DAMPING = 0.85;
    const CENTER_GRAVITY = 0.012;
    const ITERS = 200;

    const cx = (width || 600) / 2;
    const cy = height / 2;

    let tick = 0;
    let rafId: number;

    function step() {
      if (tick >= ITERS) return;
      tick++;

      const ns = nodes;
      const nodeMap = new Map(ns.map((n) => [n.id, n]));

      for (let i = 0; i < ns.length; i++) {
        let ax = 0;
        let ay = 0;

        // Repulsion from all other nodes
        for (let j = 0; j < ns.length; j++) {
          if (i === j) continue;
          const dx = ns[i].x - ns[j].x;
          const dy = ns[i].y - ns[j].y;
          const dist2 = Math.max(1, dx * dx + dy * dy);
          const force = REPULSION / dist2;
          ax += (dx / Math.sqrt(dist2)) * force;
          ay += (dy / Math.sqrt(dist2)) * force;
        }

        // Center gravity
        ax += (cx - ns[i].x) * CENTER_GRAVITY;
        ay += (cy - ns[i].y) * CENTER_GRAVITY;

        ns[i].vx = (ns[i].vx + ax) * DAMPING;
        ns[i].vy = (ns[i].vy + ay) * DAMPING;
      }

      // Attraction along edges
      for (const edge of edges) {
        const src = nodeMap.get(edge.source);
        const tgt = nodeMap.get(edge.target);
        if (!src || !tgt) continue;
        const dx = tgt.x - src.x;
        const dy = tgt.y - src.y;
        src.vx += dx * ATTRACTION;
        src.vy += dy * ATTRACTION;
        tgt.vx -= dx * ATTRACTION;
        tgt.vy -= dy * ATTRACTION;
      }

      // Integrate positions
      for (const n of ns) {
        if (draggingId === n.id) continue;
        n.x += n.vx;
        n.y += n.vy;
      }

      // Trigger reactivity by reassigning
      nodes = [...ns];

      rafId = requestAnimationFrame(step);
    }

    rafId = requestAnimationFrame(step);
  }

  $effect(() => {
    void loadGraph();
  });

  $effect(() => {
    engine.workspace.subscribe(() => void loadGraph());
  });

  // Drag handling
  function onNodeMousedown(e: MouseEvent, id: string) {
    e.stopPropagation();
    draggingId = id;
    const node = nodes.find((n) => n.id === id);
    if (!node) return;
    const svgPt = toSvgPoint(e.clientX, e.clientY);
    dragOffsetX = svgPt.x - node.x;
    dragOffsetY = svgPt.y - node.y;
  }

  function onNodeTouchstart(e: TouchEvent, id: string) {
    e.stopPropagation();
    draggingId = id;
    const node = nodes.find((n) => n.id === id);
    if (!node) return;
    const t = e.touches[0];
    const svgPt = toSvgPoint(t.clientX, t.clientY);
    dragOffsetX = svgPt.x - node.x;
    dragOffsetY = svgPt.y - node.y;
  }

  function onSvgMousemove(e: MouseEvent) {
    if (draggingId) {
      const svgPt = toSvgPoint(e.clientX, e.clientY);
      nodes = nodes.map((n) =>
        n.id === draggingId
          ? { ...n, x: svgPt.x - dragOffsetX, y: svgPt.y - dragOffsetY, vx: 0, vy: 0 }
          : n,
      );
    } else if (isPanning) {
      panX = panStartPanX + (e.clientX - panStartX);
      panY = panStartPanY + (e.clientY - panStartY);
    }
  }

  function onSvgMouseup() {
    draggingId = null;
    isPanning = false;
  }

  function onSvgMousedown(e: MouseEvent) {
    if ((e.target as SVGElement).closest('.kg-node')) return;
    isPanning = true;
    panStartX = e.clientX;
    panStartY = e.clientY;
    panStartPanX = panX;
    panStartPanY = panY;
  }

  function onWheel(e: WheelEvent) {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    zoom = Math.max(0.2, Math.min(4, zoom * delta));
  }

  function toSvgPoint(cx: number, cy: number): { x: number; y: number } {
    if (!svgEl) return { x: cx, y: cy };
    const rect = svgEl.getBoundingClientRect();
    const x = (cx - rect.left - panX) / zoom;
    const y = (cy - rect.top - panY) / zoom;
    return { x, y };
  }

  const hoveredNode = $derived(hoveredId ? nodes.find((n) => n.id === hoveredId) : null);
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="kg-root" bind:this={containerEl} style="height: {height}px;">
  {#if loading}
    <div class="kg-loading">
      <div class="oe-spinner"></div>
      <span>Loading graph…</span>
    </div>
  {:else if error}
    <div class="kg-error">{error}</div>
  {:else if nodes.length === 0}
    <div class="oe-empty">
      <span class="oe-empty__icon">◎</span>
      <span>No graph data available.</span>
    </div>
  {:else}
    <svg
      bind:this={svgEl}
      class="kg-svg"
      viewBox="0 0 {width} {height}"
      style="width: 100%; height: {height}px; cursor: {isPanning ? 'grabbing' : 'grab'};"
      onmousemove={onSvgMousemove}
      onmouseup={onSvgMouseup}
      onmousedown={onSvgMousedown}
      onwheel={onWheel}
      aria-label="Knowledge graph"
    >
      <g transform="translate({panX},{panY}) scale({zoom})">
        <!-- Edges -->
        <g class="kg-edges">
          {#each edges as edge (`${edge.source}-${edge.target}`)}
            {@const src = nodes.find((n) => n.id === edge.source)}
            {@const tgt = nodes.find((n) => n.id === edge.target)}
            {#if src && tgt}
              <line
                x1={src.x}
                y1={src.y}
                x2={tgt.x}
                y2={tgt.y}
                stroke={edgeColor(edge.type)}
                stroke-width="1.2"
                stroke-opacity="0.3"
              />
            {/if}
          {/each}
        </g>

        <!-- Nodes -->
        <g class="kg-nodes">
          {#each nodes as node (node.id)}
            {@const r = nodeRadius(node.connections)}
            {@const color = nodeColor(node.type)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <g
              class="kg-node"
              transform="translate({node.x},{node.y})"
              onmousedown={(e) => onNodeMousedown(e, node.id)}
              ontouchstart={(e) => onNodeTouchstart(e, node.id)}
              onmouseenter={() => hoveredId = node.id}
              onmouseleave={() => hoveredId = null}
              onclick={() => onselect?.({ id: node.id, name: node.name, type: node.type })}
              role="button"
              tabindex="0"
              aria-label="{node.name} ({node.type})"
              onkeydown={(e) => { if (e.key === 'Enter') onselect?.({ id: node.id, name: node.name, type: node.type }); }}
              style="cursor: pointer;"
            >
              <circle
                r={r}
                fill={color}
                fill-opacity={hoveredId === node.id ? 0.9 : 0.6}
                stroke={color}
                stroke-width={hoveredId === node.id ? 2 : 1}
              />
              {#if r > 9 || hoveredId === node.id}
                <text
                  y={r + 11}
                  text-anchor="middle"
                  font-size="9"
                  fill="var(--dt3, #888)"
                  pointer-events="none"
                >
                  {node.name.length > 14 ? node.name.slice(0, 13) + '…' : node.name}
                </text>
              {/if}
            </g>
          {/each}
        </g>
      </g>
    </svg>

    <!-- Hover tooltip -->
    {#if hoveredNode}
      <div class="kg-tooltip" aria-live="polite">
        <span class="kg-tooltip__name">{hoveredNode.name}</span>
        <span class="kg-tooltip__type">{hoveredNode.type}</span>
        <span class="kg-tooltip__connections">{hoveredNode.connections} connections</span>
      </div>
    {/if}
  {/if}
</div>

<style>
  .kg-root {
    position: relative;
    width: 100%;
    background: var(--dbg, #0d1117);
    border-radius: 8px;
    border: 1px solid var(--dbd, rgba(255,255,255,0.08));
    overflow: hidden;
  }

  .kg-svg {
    display: block;
    user-select: none;
  }

  .kg-loading {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    font-size: 0.82rem;
    color: var(--dt4, #555);
  }

  .kg-error {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.82rem;
    color: #f88;
    padding: 1rem;
    text-align: center;
  }

  .kg-tooltip {
    position: absolute;
    bottom: 10px;
    left: 10px;
    display: flex;
    gap: 0.4rem;
    align-items: center;
    background: var(--dbg, #0d1117);
    border: 1px solid var(--dbd, rgba(255,255,255,0.12));
    border-radius: 6px;
    padding: 0.3rem 0.65rem;
    pointer-events: none;
    z-index: 10;
  }

  .kg-tooltip__name {
    font-size: 0.82rem;
    font-weight: 600;
    color: var(--dt, #f1f1f3);
  }

  .kg-tooltip__type {
    font-size: 0.7rem;
    color: var(--daccent, #7ea8ff);
    background: rgba(126, 168, 255, 0.1);
    padding: 1px 6px;
    border-radius: 999px;
  }

  .kg-tooltip__connections {
    font-size: 0.68rem;
    color: var(--dt4, #555);
  }
</style>
