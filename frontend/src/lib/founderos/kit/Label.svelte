<!-- Mono section label: `LABEL  count ————` -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	let { count, rule = false, children }: { count?: string | number; rule?: boolean; children?: Snippet } = $props();
</script>

<div data-part="label" class="bn-dim flex items-center gap-2 font-mono text-[10px] font-bold uppercase tracking-[0.26em]">
	<span class="whitespace-nowrap">{@render children?.()}</span>
	{#if count != null}<span data-part="count" class="bn-muted">{count}</span>{/if}
	{#if rule}<span data-part="rule" class="bn-rule h-px flex-1"></span>{/if}
</div>

<style>
	.bn-rule {
		background: var(--bn-border);
	}
</style>
