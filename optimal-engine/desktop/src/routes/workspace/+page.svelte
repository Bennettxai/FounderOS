<script lang="ts">
  import { onMount } from 'svelte';
  import {
    listWorkspace,
    getNodeFiles,
    getSignal,
    toTree,
    type WorkspaceNode,
    type SignalDetail
  } from '$lib/api/workspace';
  import { activeWorkspaceId } from '$lib/stores/workspace';

  type TreeNode = WorkspaceNode & { children: TreeNode[] };
  type FileEntry = { name: string; path: string; genre: string | null; modified_at: string | null };

  let tree = $state<TreeNode[]>([]);
  let expanded = $state<Set<string>>(new Set());
  let selectedNode = $state<WorkspaceNode | null>(null);
  let nodeFiles = $state<FileEntry[]>([]);
  let selectedSignal = $state<SignalDetail | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadTree(workspace: string | null) {
    try {
      error = null;
      selectedNode = null;
      nodeFiles = [];
      selectedSignal = null;
      expanded = new Set();
      const nodes = await listWorkspace(workspace);
      tree = toTree(nodes) as TreeNode[];
    } catch (e) {
      error = (e as Error).message;
    }
  }

  onMount(() => loadTree($activeWorkspaceId));

  // Re-fetch tree when workspace switches
  $effect(() => {
    void loadTree($activeWorkspaceId);
  });

  function toggle(id: string) {
    const next = new Set(expanded);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    expanded = next;
  }

  async function selectNode(n: WorkspaceNode) {
    selectedNode = n;
    selectedSignal = null;
    loading = true;
    try {
      nodeFiles = await getNodeFiles(n.slug);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function selectSignalByPath(path: string) {
    loading = true;
    try {
      // path is the context URI or id — extract id from optimal://contexts/<id> or use as-is
      const id = path.startsWith('optimal://') ? path.split('/').pop()! : path;
      selectedSignal = await getSignal(id);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  function chunkCountAt(scale: string): number {
    if (!selectedSignal) return 0;
    return selectedSignal.chunks.filter((c) => c.scale === scale).length;
  }

  function entityGroups(): Record<string, string[]> {
    if (!selectedSignal) return {};
    const out: Record<string, string[]> = {};
    for (const e of selectedSignal.entities) {
      (out[e.type] ||= []).push(e.name);
    }
    return out;
  }
</script>

<div class="ws">
  <!-- Left: node tree -->
  <aside class="ws__tree">
    <header>Workspace</header>
    {#if error}
      <pre class="error">{error}</pre>
    {/if}
    {#if tree.length === 0}
      <p class="muted">No nodes yet. Run <code>mix optimal.seed</code>.</p>
    {:else}
      <ul>
        {#each tree as root}
          {@render treeItem(root, 0)}
        {/each}
      </ul>
    {/if}
  </aside>

  {#snippet treeItem(node: TreeNode, depth: number)}
    <li>
      <button
        class="ws__node"
        class:ws__node--active={selectedNode?.id === node.id}
        style="padding-left: {depth * 1.25 + 0.5}rem"
        onclick={() => selectNode(node)}
      >
        {#if node.children.length > 0}
          <span
            class="chev"
            class:chev--open={expanded.has(node.id)}
            onclick={(e) => {
              e.stopPropagation();
              toggle(node.id);
            }}
            role="button"
            tabindex="0"
            onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && toggle(node.id)}
          >
            ▸
          </span>
        {:else}
          <span class="chev chev--leaf">·</span>
        {/if}
        <span class="ws__node-name">{node.name}</span>
        <span class="ws__node-count">{node.signal_count}</span>
        {#if node.style === 'external'}
          <span class="ws__pill">external</span>
        {/if}
      </button>
      {#if expanded.has(node.id) && node.children.length > 0}
        <ul>
          {#each node.children as child}
            {@render treeItem(child, depth + 1)}
          {/each}
        </ul>
      {/if}
    </li>
  {/snippet}

  <!-- Middle: signal list -->
  <section class="ws__list">
    <header>
      {#if selectedNode}
        <strong>{selectedNode.name}</strong>
        <span class="muted">· {selectedNode.kind} · {nodeFiles.length} signals</span>
      {:else}
        <span class="muted">Pick a node →</span>
      {/if}
    </header>
    {#if nodeFiles.length === 0 && selectedNode}
      <p class="muted">This node has no signals yet.</p>
    {/if}
    <ul>
      {#each nodeFiles as f}
        <li>
          <button
            class="ws__signal"
            class:ws__signal--active={selectedSignal?.id === f.path.split('/').pop()}
            onclick={() => selectSignalByPath(f.path)}
          >
            <span class="ws__signal-name">{f.name}</span>
            {#if f.genre}
              <span class="ws__genre">{f.genre}</span>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  </section>

  <!-- Right: signal granularity detail -->
  <section class="ws__detail">
    {#if !selectedSignal}
      <div class="ws__hint">
        <p>A <strong>signal</strong> is one data point. Pick one to see:</p>
        <ul>
          <li>S=(M, G, T, F, W) classification dimensions</li>
          <li>4-scale chunk hierarchy (document → section → paragraph → sentence)</li>
          <li>Extracted entities by type</li>
          <li>Cluster membership + wiki citations</li>
          <li>Data architecture binding</li>
        </ul>
      </div>
    {:else}
      <header class="ws__detail-header">
        <h3>{selectedSignal.title}</h3>
        <span class="ws__genre">{selectedSignal.genre}</span>
        {#if selectedSignal.architecture_id}
          <span class="ws__arch">arch: {selectedSignal.architecture_id}</span>
        {/if}
      </header>

      <div class="ws__block">
        <h4>Signal dimensions S=(M, G, T, F, W)</h4>
        <div class="ws__dims">
          <span class="ws__dim"><small>mode</small>{selectedSignal.signal_dimensions.mode}</span>
          <span class="ws__dim"><small>genre</small>{selectedSignal.signal_dimensions.genre}</span>
          <span class="ws__dim"><small>type</small>{selectedSignal.signal_dimensions.type}</span>
          <span class="ws__dim"><small>format</small>{selectedSignal.signal_dimensions.format}</span>
          <span class="ws__dim"><small>structure</small>{selectedSignal.signal_dimensions.structure}</span>
        </div>
        <p class="ws__sn">S/N ratio: <strong>{selectedSignal.sn_ratio}</strong></p>
      </div>

      <div class="ws__block">
        <h4>4-scale chunk hierarchy</h4>
        <div class="ws__scales">
          {#each ['document', 'section', 'paragraph', 'sentence'] as scale}
            <div class="ws__scale">
              <strong>{chunkCountAt(scale)}</strong>
              <small>{scale}</small>
            </div>
          {/each}
        </div>
      </div>

      <div class="ws__block">
        <h4>Entities ({selectedSignal.entities.length})</h4>
        {#each Object.entries(entityGroups()) as [type, names]}
          <div class="ws__entity-group">
            <span class="ws__entity-type">{type}</span>
            {#each names as name}
              <span class="ws__entity">{name}</span>
            {/each}
          </div>
        {/each}
      </div>

      {#if selectedSignal.classification}
        <div class="ws__block">
          <h4>Classification</h4>
          <p class="muted">
            confidence: {selectedSignal.classification.confidence} ·
            S/N: {selectedSignal.classification.sn_ratio}
          </p>
        </div>
      {/if}

      {#if selectedSignal.intent}
        <div class="ws__block">
          <h4>Intent</h4>
          <p><strong>{selectedSignal.intent.intent}</strong> · confidence {selectedSignal.intent.confidence}</p>
        </div>
      {/if}

      {#if selectedSignal.clusters.length > 0}
        <div class="ws__block">
          <h4>Cluster membership</h4>
          {#each selectedSignal.clusters as c}
            <p>{c.theme} <span class="muted">· weight {c.weight}</span></p>
          {/each}
        </div>
      {/if}

      {#if selectedSignal.citations.length > 0}
        <div class="ws__block">
          <h4>Wiki citations</h4>
          {#each selectedSignal.citations as c}
            <p><code>{c.wiki_slug}</code> · {c.audience}</p>
          {/each}
        </div>
      {/if}

      <div class="ws__block">
        <h4>Content</h4>
        <pre class="ws__content">{selectedSignal.content}</pre>
      </div>
    {/if}
  </section>
</div>

<style>
  .ws {
    display: grid;
    grid-template-columns: 280px 340px 1fr;
    height: calc(100vh - 64px);
    background: #0b0d10;
    color: #ddd;
  }
  aside,
  section {
    overflow: auto;
    border-right: 1px solid #1a1e25;
  }
  header {
    padding: 0.75rem 1rem;
    font-size: 0.85rem;
    color: #888;
    border-bottom: 1px solid #1a1e25;
    position: sticky;
    top: 0;
    background: #0b0d10;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .ws__tree ul ul {
    margin-left: 0;
  }
  .ws__node {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    width: 100%;
    padding: 0.35rem 0.5rem;
    background: none;
    border: none;
    color: inherit;
    text-align: left;
    font-size: 0.85rem;
    cursor: pointer;
  }
  .ws__node:hover,
  .ws__node--active {
    background: #161a21;
  }
  .ws__node-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ws__node-count {
    font-size: 0.7rem;
    color: #666;
    padding: 1px 6px;
    background: #111;
    border-radius: 999px;
  }
  .ws__pill {
    font-size: 0.6rem;
    padding: 1px 6px;
    background: #332;
    color: #fc6;
    border-radius: 999px;
  }
  .chev {
    display: inline-block;
    width: 12px;
    transition: transform 0.15s;
    color: #666;
    font-size: 0.7rem;
  }
  .chev--open {
    transform: rotate(90deg);
  }
  .chev--leaf {
    color: #333;
  }
  .ws__signal {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: none;
    border: none;
    color: inherit;
    text-align: left;
    font-size: 0.85rem;
    cursor: pointer;
    border-bottom: 1px solid #111;
  }
  .ws__signal:hover,
  .ws__signal--active {
    background: #161a21;
  }
  .ws__signal-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
  }
  .ws__genre {
    font-size: 0.7rem;
    padding: 2px 8px;
    background: #143;
    color: #6f8;
    border-radius: 999px;
  }
  .ws__arch {
    font-size: 0.7rem;
    padding: 2px 8px;
    background: #223;
    color: #8af;
    border-radius: 999px;
    margin-left: 0.5rem;
  }
  .ws__detail {
    padding: 1rem 1.25rem;
  }
  .ws__detail-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }
  .ws__detail-header h3 {
    margin: 0;
    flex: 1;
    font-size: 1rem;
  }
  .ws__block {
    margin-bottom: 1.25rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid #1a1e25;
  }
  .ws__block h4 {
    margin: 0 0 0.5rem 0;
    color: #888;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 600;
  }
  .ws__dims {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .ws__dim {
    display: flex;
    flex-direction: column;
    padding: 0.5rem 0.75rem;
    background: #111;
    border: 1px solid #1a1e25;
    border-radius: 6px;
    font-size: 0.8rem;
  }
  .ws__dim small {
    color: #666;
    font-size: 0.65rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 2px;
  }
  .ws__sn {
    margin-top: 0.5rem;
    font-size: 0.8rem;
    color: #888;
  }
  .ws__scales {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.5rem;
  }
  .ws__scale {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0.75rem;
    background: #111;
    border: 1px solid #1a1e25;
    border-radius: 6px;
  }
  .ws__scale strong {
    font-size: 1.3rem;
    color: #8af;
  }
  .ws__scale small {
    color: #666;
    font-size: 0.7rem;
    text-transform: uppercase;
  }
  .ws__entity-group {
    margin-bottom: 0.5rem;
  }
  .ws__entity-type {
    display: inline-block;
    padding: 1px 8px;
    margin-right: 0.4rem;
    background: #223;
    color: #8af;
    border-radius: 999px;
    font-size: 0.7rem;
    text-transform: uppercase;
  }
  .ws__entity {
    display: inline-block;
    margin-right: 0.5rem;
    color: #bbb;
    font-size: 0.85rem;
  }
  .ws__content {
    background: #0e1116;
    border: 1px solid #1a1e25;
    border-radius: 4px;
    padding: 0.75rem;
    white-space: pre-wrap;
    font-size: 0.8rem;
    line-height: 1.5;
    max-height: 240px;
    overflow: auto;
  }
  .ws__hint {
    padding: 2rem;
    color: #888;
  }
  .ws__hint ul {
    padding-left: 1.25rem;
  }
  .ws__hint li {
    padding: 0.15rem 0;
  }
  .muted {
    color: #888;
    font-size: 0.85rem;
  }
  .error {
    color: #f88;
    padding: 0.5rem;
  }
  code {
    font-family: 'SF Mono', Menlo, monospace;
    background: #0e1116;
    padding: 0 4px;
    border-radius: 3px;
  }
</style>
