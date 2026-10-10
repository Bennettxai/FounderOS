<!-- [[wiki-link]] styled chip, optionally clickable (FounderOS v1 KnowledgeDetail WikiLink). -->
<script lang="ts">
	let { label, mcp = false, onclick }: { label: string; mcp?: boolean; onclick?: () => void } = $props();
</script>

{#snippet inner()}
	<span class="bn-dim">[[</span>{label}<span class="bn-dim">]]</span>
	{#if mcp}<span class="kg-mcp rounded-[5px] border px-1 text-[8px] uppercase tracking-wide">mcp</span>{/if}
{/snippet}

{#if onclick}
	<button type="button" {onclick} class="bn-pressable kg-accent kg-h-o70 inline-flex items-center gap-1 font-mono text-[10.5px] transition-opacity">{@render inner()}</button>
{:else}
	<span class="kg-accent inline-flex items-center gap-1 font-mono text-[10.5px]">{@render inner()}</span>
{/if}

<style>
	.kg-mcp {
		border-color: var(--bn-accent-line);
		color: var(--bn-accent);
	}
</style>
