<!-- Inset card: rises on the stagger, lifts 1px on hover (FounderOS v1 slab.tsx SlabCard). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import Render from './Render.svelte';

	let {
		title,
		sub,
		action,
		i = 1,
		class: className = '',
		children
	}: {
		title?: string;
		sub?: string | Snippet;
		action?: Snippet;
		i?: number;
		class?: string;
		children?: Snippet;
	} = $props();
</script>

<div class="bn-rise bn-rise-card bn-card relative min-w-0 {className}" style="--rise-i: {i}">
	{#if title}
		<div class="flex flex-wrap items-center justify-between gap-3 px-6 pt-5">
			<div class="flex min-w-0 items-baseline gap-2">
				<h2 class="bn-text text-[19px] font-semibold tracking-[-0.01em]">{title}</h2>
				{#if sub}<span class="bn-dim font-mono text-[12px] tabular-nums"><Render value={sub} /></span>{/if}
			</div>
			{#if action}<div class="flex flex-wrap items-center gap-2">{@render action()}</div>{/if}
		</div>
	{/if}
	{@render children?.()}
</div>
