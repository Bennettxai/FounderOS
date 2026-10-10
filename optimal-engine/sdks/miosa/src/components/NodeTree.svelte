<script lang="ts">
  export interface TreeNode {
    slug: string;
    label: string;
    signalCount?: number;
    children?: TreeNode[];
  }

  interface Props {
    nodes: TreeNode[];
    depth?: number;
    onselect?: (slug: string) => void;
  }

  let { nodes, depth = 0, onselect }: Props = $props();

  let expanded = $state<Set<string>>(new Set());

  function toggle(slug: string) {
    const next = new Set(expanded);
    if (next.has(slug)) next.delete(slug);
    else next.add(slug);
    expanded = next;
  }

  function handleSelect(e: MouseEvent | KeyboardEvent, node: TreeNode) {
    e.stopPropagation();
    if (node.children?.length) toggle(node.slug);
    onselect?.(node.slug);
  }
</script>

<div class="oe-tree-children" style={depth === 0 ? 'padding-left: 0' : undefined}>
  {#each nodes as node (node.slug)}
    {@const hasChildren = !!node.children?.length}
    {@const isOpen = expanded.has(node.slug)}

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="oe-tree-node"
      role="button"
      tabindex="0"
      aria-expanded={hasChildren ? isOpen : undefined}
      aria-label="{node.label}{node.signalCount !== undefined ? `, ${node.signalCount} signals` : ''}"
      onclick={(e) => handleSelect(e, node)}
      onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSelect(e, node); } }}
    >
      <span class="oe-tree-node__chevron {isOpen ? 'oe-tree-node__chevron--open' : ''}">
        {#if hasChildren}
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <path d="M3 2l4 3-4 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        {:else}
          <svg width="6" height="6" viewBox="0 0 6 6" fill="currentColor" aria-hidden="true">
            <circle cx="3" cy="3" r="2"/>
          </svg>
        {/if}
      </span>

      <span class="oe-tree-node__label">{node.label}</span>

      {#if node.signalCount !== undefined}
        <span class="oe-tree-node__count">{node.signalCount}</span>
      {/if}
    </div>

    {#if hasChildren && isOpen}
      <svelte:self nodes={node.children!} depth={depth + 1} {onselect} />
    {/if}
  {/each}
</div>
