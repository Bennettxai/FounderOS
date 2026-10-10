<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import Label from './Label.svelte';

	let {
		label,
		count,
		link,
		href,
		right
	}: {
		label: string;
		count?: string | number;
		link?: string;
		href?: string;
		/** Controls that belong to the section, sitting after the hairline. */
		right?: Snippet;
	} = $props();
</script>

<div class="bn-rise mb-3 flex items-baseline justify-between gap-3">
	<div class="min-w-0 flex-1">
		<Label {count} rule>{label}</Label>
	</div>
	{#if link && href}
		<a {href} class="bn-linky bn-dim shrink-0 font-mono text-[11px]">{link} →</a>
	{/if}
	{@render right?.()}
</div>
