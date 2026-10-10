<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    forceSimulation,
    forceLink,
    forceManyBody,
    forceCenter,
    forceCollide
  } from 'd3-force';
  import type { Memory, MemoryRelation, MemoryRelationType } from '$lib/api';

  // ── Props ────────────────────────────────────────────────────────────────
  interface Props {
    memories: Memory[];
    relations: MemoryRelation[];
    selectedId?: string | null;
    onSelect?: (id: string) => void;
  }
  let { memories, relations, selectedId = null, onSelect }: Props = $props();

  // ── Relation colour map (matches CSS vars in app.css) ────────────────────
  // These must be hex because SVG stroke uses literal color values.
  // We read from the computed style so they respond to light/dark theme.
  const RELATION_COLORS: Record<MemoryRelationType, string> = {
    updates:     'var(--accent)',
    extends:     'var(--good)',
    derives:     'var(--purple)',
    contradicts: 'var(--bad)',
    cites:       'var(--text-muted)',
  };

  const RELATION_DASHED: Record<MemoryRelationType, boolean> = {
    updates:     false,
    extends:     false,
    derives:     false,
    contradicts: false,
    cites:       true,
  };

  // ── Sim node/link types ──────────────────────────────────────────────────
  interface SimNode {
    id: string;
    memory: Memory;
    x: number;
    y: number;
    vx: number;
    vy: number;
    fx: number | null;
    fy: number | null;
    r: number;
  }

  interface SimLink {
    source: SimNode | string;
    target: SimNode | string;
    relation: MemoryRelationType;
  }

  // ── SVG state ────────────────────────────────────────────────────────────
  let svgEl = $state<SVGSVGElement | null>(null);
  let width = $state(600);
  let height = $state(500);

  // Rendered node/link data (updated each sim tick via assignment)
  let nodes = $state<SimNode[]>([]);
  let links = $state<SimLink[]>([]);

  // Pan + zoom
  let pan = $state({ x: 0, y: 0 });
  let zoom = $state(1);

  // Drag state
  let dragging: SimNode | null = null;
  let dragStart = { mx: 0, my: 0, px: 0, py: 0 };
  let wasDragged = false;

  // Pan drag
  let panning = false;
  let panStart = { mx: 0, my: 0, px: 0, py: 0 };

  // Tooltip
  let tooltip = $state<{ x: number; y: number; memory: Memory } | null>(null);

  // sim handle — typed loosely so forceSimulation<SimNode> assignment is compatible
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let sim: any = null;
  let ro: ResizeObserver | null = null;

  // ── Node sizing ──────────────────────────────────────────────────────────
  function nodeRadius(m: Memory): number {
    const base = 12;
    const vBonus = Math.min((m.version - 1) * 3, 14);
    return base + vBonus;
  }

  // ── Build simulation ─────────────────────────────────────────────────────
  function buildSim() {
    sim?.stop();

    const simNodes: SimNode[] = memories.map((m, i) => ({
      id: m.id,
      memory: m,
      x: width / 2 + (Math.random() - 0.5) * 200,
      y: height / 2 + (Math.random() - 0.5) * 200,
      vx: 0,
      vy: 0,
      fx: null,
      fy: null,
      r: nodeRadius(m),
    }));

    const byId = new Map(simNodes.map((n) => [n.id, n]));

    const simLinks: SimLink[] = relations
      .filter((r) => byId.has(r.source_memory_id) && byId.has(r.target_memory_id))
      .map((r) => ({
        source: r.source_memory_id,
        target: r.target_memory_id,
        relation: r.relation,
      }));

    sim = forceSimulation<SimNode>(simNodes)
      .alphaDecay(0.025)
      .alphaMin(0.001)
      .velocityDecay(0.38)
      .force(
        'link',
        forceLink<SimNode, SimLink>(simLinks)
          .id((d) => d.id)
          .distance(90)
          .strength(0.4),
      )
      .force('charge', forceManyBody<SimNode>().strength(-220).distanceMax(350))
      .force('center', forceCenter<SimNode>(width / 2, height / 2).strength(0.05))
      .force('collide', forceCollide<SimNode>().radius((d) => d.r + 6).iterations(2))
      .on('tick', () => {
        // Force reactive re-render by reassigning the arrays.
        // d3-force mutates objects in-place; we shallow-copy to trigger $state.
        nodes = [...simNodes];
        links = [...simLinks];
      });
  }

  // ── Rebuild whenever memories/relations change ───────────────────────────
  $effect(() => {
    // Capture reactive deps
    const _m = memories;
    const _r = relations;
    buildSim();
    return () => sim?.stop();
  });

  // ── Resize observer ──────────────────────────────────────────────────────
  onMount(() => {
    if (!svgEl) return;
    ro = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) return;
      width = entry.contentRect.width;
      height = entry.contentRect.height;
      sim?.force('center', forceCenter<SimNode>(width / 2, height / 2).strength(0.05));
      sim?.alpha(0.3).restart();
    });
    ro.observe(svgEl);
    width = svgEl.clientWidth;
    height = svgEl.clientHeight;
  });

  onDestroy(() => {
    sim?.stop();
    ro?.disconnect();
  });

  // ── Helpers ──────────────────────────────────────────────────────────────
  function resolvedNode(n: SimNode | string): SimNode | null {
    if (typeof n === 'string') return nodes.find((x) => x.id === n) ?? null;
    return n;
  }

  function snip(text: string, len = 80): string {
    return text.length > len ? text.slice(0, len) + '…' : text;
  }

  function fmtDate(iso: string): string {
    return new Date(iso).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      year: '2-digit',
    });
  }

  // ── SVG interaction ──────────────────────────────────────────────────────
  function svgCoords(ev: MouseEvent): { x: number; y: number } {
    if (!svgEl) return { x: 0, y: 0 };
    const rect = svgEl.getBoundingClientRect();
    return {
      x: (ev.clientX - rect.left - pan.x) / zoom,
      y: (ev.clientY - rect.top - pan.y) / zoom,
    };
  }

  function onNodePointerDown(ev: PointerEvent, node: SimNode) {
    ev.stopPropagation();
    dragging = node;
    wasDragged = false;
    dragStart = { mx: ev.clientX, my: ev.clientY, px: node.x, py: node.y };
    node.fx = node.x;
    node.fy = node.y;
    sim?.alphaTarget(0.15).restart();
    (ev.currentTarget as Element).setPointerCapture(ev.pointerId);
  }

  function onSvgPointerDown(ev: PointerEvent) {
    if (ev.button !== 0) return;
    panning = true;
    panStart = { mx: ev.clientX, my: ev.clientY, px: pan.x, py: pan.y };
  }

  function onPointerMove(ev: PointerEvent) {
    if (dragging) {
      const dx = ev.clientX - dragStart.mx;
      const dy = ev.clientY - dragStart.my;
      if (Math.abs(dx) + Math.abs(dy) > 3) wasDragged = true;
      dragging.fx = dragStart.px + dx / zoom;
      dragging.fy = dragStart.py + dy / zoom;
      // Update tooltip position while dragging
      tooltip = null;
      return;
    }
    if (panning) {
      pan = {
        x: panStart.px + (ev.clientX - panStart.mx),
        y: panStart.py + (ev.clientY - panStart.my),
      };
    }
  }

  function onPointerUp(ev: PointerEvent) {
    if (dragging) {
      if (!wasDragged) {
        onSelect?.(dragging.id);
      }
      dragging.fx = null;
      dragging.fy = null;
      dragging = null;
      sim?.alphaTarget(0);
    }
    panning = false;
  }

  function onWheel(ev: WheelEvent) {
    ev.preventDefault();
    const delta = ev.deltaY > 0 ? 0.9 : 1.1;
    const newZoom = Math.min(Math.max(zoom * delta, 0.2), 4);
    // Zoom toward pointer
    if (!svgEl) { zoom = newZoom; return; }
    const rect = svgEl.getBoundingClientRect();
    const mx = ev.clientX - rect.left;
    const my = ev.clientY - rect.top;
    pan = {
      x: mx - (mx - pan.x) * (newZoom / zoom),
      y: my - (my - pan.y) * (newZoom / zoom),
    };
    zoom = newZoom;
  }

  function onNodeEnter(ev: MouseEvent, node: SimNode) {
    if (!svgEl) return;
    const rect = svgEl.getBoundingClientRect();
    tooltip = {
      x: ev.clientX - rect.left + 12,
      y: ev.clientY - rect.top + 12,
      memory: node.memory,
    };
  }

  function onNodeLeave() {
    tooltip = null;
  }

  function resetView() {
    pan = { x: 0, y: 0 };
    zoom = 1;
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="mg">
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <svg
    bind:this={svgEl}
    class="mg__svg"
    role="img"
    aria-label="Memory graph"
    onpointerdown={onSvgPointerDown}
    onpointermove={onPointerMove}
    onpointerup={onPointerUp}
    onpointerleave={onPointerUp}
    onwheel={onWheel}
  >
    <g transform="translate({pan.x},{pan.y}) scale({zoom})">
      <!-- Edges -->
      {#each links as link (typeof link.source === 'string' ? link.source + '-' + link.target : link.source.id + '-' + (typeof link.target === 'string' ? link.target : link.target.id))}
        {@const src = resolvedNode(link.source)}
        {@const tgt = resolvedNode(link.target)}
        {#if src && tgt}
          <line
            class="mg__edge"
            x1={src.x}
            y1={src.y}
            x2={tgt.x}
            y2={tgt.y}
            stroke={RELATION_COLORS[link.relation]}
            stroke-dasharray={RELATION_DASHED[link.relation] ? '5,4' : undefined}
            stroke-width="1.5"
            stroke-opacity="0.7"
          />
          <!-- Arrow marker approximation: a small dot at the target end -->
          <circle
            cx={tgt.x + (src.x - tgt.x) * ((tgt.r + 4) / (Math.hypot(tgt.x - src.x, tgt.y - src.y) || 1))}
            cy={tgt.y + (src.y - tgt.y) * ((tgt.r + 4) / (Math.hypot(tgt.x - src.x, tgt.y - src.y) || 1))}
            r="3"
            fill={RELATION_COLORS[link.relation]}
            opacity="0.8"
          />
        {/if}
      {/each}

      <!-- Nodes -->
      {#each nodes as node (node.id)}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <g
          class="mg__node"
          transform="translate({node.x},{node.y})"
          onpointerdown={(e) => onNodePointerDown(e, node)}
          onpointerenter={(e) => onNodeEnter(e, node)}
          onpointerleave={onNodeLeave}
          role="button"
          tabindex="0"
          aria-label="Memory: {snip(node.memory.content, 40)}"
          onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect?.(node.id); }}}
        >
          <!-- Outer ring for static / selected -->
          {#if node.memory.is_static || selectedId === node.id}
            <circle
              r={node.r + 4}
              fill="none"
              stroke={selectedId === node.id ? 'var(--accent)' : 'var(--amber)'}
              stroke-width={selectedId === node.id ? 2.5 : 1.8}
              opacity={node.memory.is_forgotten ? 0.35 : 1}
            />
          {/if}

          <!-- Main circle -->
          <circle
            r={node.r}
            fill={selectedId === node.id ? 'var(--accent-soft)' : 'var(--bg-elevated-2)'}
            stroke={node.memory.is_forgotten ? 'var(--border)' : 'var(--border)'}
            stroke-width={node.memory.is_forgotten ? 0 : 1.5}
            stroke-dasharray={node.memory.is_forgotten ? '4,3' : undefined}
            opacity={node.memory.is_forgotten ? 0.4 : 1}
          />

          <!-- Version label inside circle -->
          <text
            class="mg__version"
            text-anchor="middle"
            dominant-baseline="central"
            font-size={node.r * 0.7}
            fill={node.memory.is_forgotten ? 'var(--text-subtle)' : 'var(--text)'}
            opacity={node.memory.is_forgotten ? 0.5 : 0.9}
            pointer-events="none"
          >v{node.memory.version}</text>

          <!-- Audience badge beneath -->
          <text
            class="mg__audience"
            y={node.r + 13}
            text-anchor="middle"
            font-size="9"
            fill="var(--text-subtle)"
            opacity={node.memory.is_forgotten ? 0.4 : 0.75}
            pointer-events="none"
          >{node.memory.audience}</text>
        </g>
      {/each}
    </g>
  </svg>

  <!-- Tooltip -->
  {#if tooltip}
    <div class="mg__tooltip" style="left:{tooltip.x}px;top:{tooltip.y}px">
      <div class="mg__tooltip-content">{snip(tooltip.memory.content)}</div>
      <div class="mg__tooltip-meta">
        <span>v{tooltip.memory.version}</span>
        <span>·</span>
        <span>{tooltip.memory.audience}</span>
        {#if tooltip.memory.is_static}<span>· static</span>{/if}
        {#if tooltip.memory.is_forgotten}<span>· forgotten</span>{/if}
        <span>·</span>
        <span>{fmtDate(tooltip.memory.created_at)}</span>
      </div>
    </div>
  {/if}

  <!-- Controls -->
  <div class="mg__controls">
    <button class="mg__ctrl-btn" onclick={resetView} title="Reset view" aria-label="Reset view">⟳</button>
  </div>

  <!-- Legend -->
  <div class="mg__legend">
    {#each Object.entries(RELATION_COLORS) as [rel, color]}
      <div class="mg__legend-item">
        <svg width="20" height="10" aria-hidden="true">
          <line
            x1="0" y1="5" x2="20" y2="5"
            stroke={color}
            stroke-width="1.8"
            stroke-dasharray={RELATION_DASHED[rel as MemoryRelationType] ? '4,3' : undefined}
          />
        </svg>
        <span>{rel}</span>
      </div>
    {/each}
    <div class="mg__legend-item">
      <svg width="14" height="14" aria-hidden="true">
        <circle cx="7" cy="7" r="6" fill="none" stroke="var(--amber)" stroke-width="1.5" />
      </svg>
      <span>static</span>
    </div>
    <div class="mg__legend-item">
      <svg width="14" height="14" aria-hidden="true">
        <circle cx="7" cy="7" r="6" fill="none" stroke="var(--border)" stroke-width="1.5" stroke-dasharray="3,2" opacity="0.5" />
      </svg>
      <span>forgotten</span>
    </div>
  </div>

  <!-- Empty state -->
  {#if memories.length === 0}
    <div class="mg__empty">
      <div class="mg__empty-icon">◈</div>
      <p>No memories to graph yet.</p>
    </div>
  {/if}
</div>

<style>
  .mg {
    position: relative;
    width: 100%;
    height: 100%;
    min-height: 400px;
    overflow: hidden;
    background: var(--bg);
    border-radius: var(--r-md);
  }

  .mg__svg {
    display: block;
    width: 100%;
    height: 100%;
    cursor: grab;
    user-select: none;
  }
  .mg__svg:active { cursor: grabbing; }

  .mg__node {
    cursor: pointer;
  }
  .mg__node:hover circle:not(:first-child) {
    filter: brightness(1.15);
  }

  .mg__version {
    font-family: var(--font-mono);
    font-weight: 600;
    letter-spacing: -0.02em;
  }

  .mg__audience {
    font-family: var(--font-sans);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 600;
  }

  .mg__tooltip {
    position: absolute;
    pointer-events: none;
    z-index: 20;
    max-width: 280px;
    padding: 0.5rem 0.7rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    box-shadow: 0 8px 24px -6px rgba(0,0,0,0.5);
    font-size: 0.78rem;
  }

  .mg__tooltip-content {
    color: var(--text);
    line-height: 1.45;
    margin-bottom: 0.3rem;
  }

  .mg__tooltip-meta {
    display: flex;
    gap: 0.3rem;
    align-items: center;
    color: var(--text-subtle);
    font-size: 0.68rem;
    font-family: var(--font-mono);
  }

  .mg__controls {
    position: absolute;
    bottom: 0.75rem;
    right: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .mg__ctrl-btn {
    width: 32px;
    height: 32px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    color: var(--text-muted);
    cursor: pointer;
    font-size: 1rem;
    transition: background 0.12s, color 0.12s;
  }
  .mg__ctrl-btn:hover {
    background: var(--bg-elevated-2);
    color: var(--text);
  }

  .mg__legend {
    position: absolute;
    top: 0.75rem;
    left: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.55rem 0.75rem;
    background: var(--glass-bg);
    backdrop-filter: blur(8px);
    border: 1px solid var(--glass-border);
    border-radius: var(--r-md);
    pointer-events: none;
  }

  .mg__legend-item {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.68rem;
    color: var(--text-muted);
    text-transform: lowercase;
    font-weight: 500;
  }

  .mg__empty {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    pointer-events: none;
  }

  .mg__empty-icon {
    font-size: 2rem;
    color: var(--text-subtle);
  }

  .mg__empty p {
    color: var(--text-muted);
    font-size: 0.9rem;
    margin: 0;
  }
</style>
