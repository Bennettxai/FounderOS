<script lang="ts">
  import PagesGraph from '$lib/knowledge/PagesGraph.svelte';
  import type { GNode } from '$lib/knowledge/graph';

  let selected = $state<GNode | null>(null);

  function onSelect(n: GNode) {
    selected = n;
  }
</script>

<div class="graph-page">
  <PagesGraph {onSelect} />
</div>

<style>
  .graph-page {
    position: relative;
    width: 100%;
    height: 100%;
  }
</style>
