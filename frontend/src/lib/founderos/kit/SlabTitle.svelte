<!-- Eyebrow · 46px title · mono meta · actions (FounderOS v1 slab.tsx SlabTitle). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import Render from './Render.svelte';

	let { eyebrow, title, meta, right }: { eyebrow?: string; title: string; meta?: string | Snippet; right?: Snippet } = $props();
</script>

<div class="bn-rise mb-7 flex flex-wrap items-end justify-between gap-5" style="--rise-i: 0">
	<div class="min-w-0">
		{#if eyebrow}
			<div class="bn-dim mb-2 font-mono text-[11px] uppercase tracking-[0.32em]">{`// ${eyebrow}`}</div>
		{/if}
		<h1 class="bn-text text-[46px] font-semibold leading-none tracking-[-0.035em] max-[600px]:text-[34px]">{title}</h1>
		{#if meta}
			<div class="bn-dim mt-3 font-mono text-[11px]"><Render value={meta} /></div>
		{/if}
	</div>
	{#if right}
		<div class="flex flex-wrap items-center gap-2.5">{@render right()}</div>
	{/if}
</div>
