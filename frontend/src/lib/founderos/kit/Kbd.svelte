<!-- Keycap (FounderOS v1 terminal.tsx Kbd); size "sm" is the palette's compact
     16px key (globals.css .kbd). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	let { size = 'md', children }: { size?: 'md' | 'sm'; children?: Snippet } = $props();
</script>

{#if size === 'sm'}
	<kbd class="bn-kbd bn-kbd-sm font-mono">{@render children?.()}</kbd>
{:else}
	<kbd class="bn-kbd px-1.5 py-0.5 font-mono text-[10px]">{@render children?.()}</kbd>
{/if}
