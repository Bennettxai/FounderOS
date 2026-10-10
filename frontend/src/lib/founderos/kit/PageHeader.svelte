<!-- Page title: 25px/700 uppercase, .06em; the eyebrow reads as a `//` comment.
     No descriptions under titles (FounderOS v1 PageHeader). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	let {
		eyebrow,
		title,
		caret = false,
		right,
		rightWide = false
	}: {
		eyebrow?: string;
		title: string;
		caret?: boolean;
		right?: Snippet;
		/** Let `right` stretch across the header whitespace instead of sitting compact at the edge. */
		rightWide?: boolean;
	} = $props();
</script>

<header class="bn-rise mb-6 flex justify-between gap-4 {rightWide ? 'items-start' : 'items-end'}">
	<div class="min-w-0 shrink-0">
		{#if eyebrow}
			<div class="bn-eyebrow bn-dim mb-2 flex items-center gap-2 font-mono text-[9.5px] uppercase tracking-[0.32em]">{eyebrow}</div>
		{/if}
		<h1 class="bn-text text-[25px] font-bold uppercase leading-[1.1] tracking-[0.06em]" class:bn-caret={caret}>{title}</h1>
	</div>
	{#if right}
		<div data-part="right" class={rightWide ? 'flex min-w-0 flex-1 items-start' : 'flex shrink-0 items-center gap-2'}>
			{@render right()}
		</div>
	{/if}
</header>
