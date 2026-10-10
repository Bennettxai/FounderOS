<!-- A clickable pulse tile that routes to its detail page (FounderOS v1
     app/page.tsx StatTile, as the .os-slab look renders it): a pressable row
     with the hover lens, a title-case label, the arrow tell visible at rest, a
     34px figure with its unit. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Label from '$lib/founderos/kit/Label.svelte';

	let {
		href,
		label,
		unit,
		value,
		foot,
		accent = false
	}: { href: string; label: string; unit: string; value: Snippet; foot?: Snippet; accent?: boolean } = $props();
</script>

<!-- v1 .os-slab .rounded-tile: 22px padding (14px on phones); the slab's tile
     rule (kit.css) supplies the 10px gap, 12px radius and 34px figure -->
<a {href} data-lens="r" data-part="tile" class="bn-pressable is-row bn-card group flex flex-col gap-2 p-[22px] max-[600px]:p-[14px]">
	<div class="flex items-center justify-between">
		<Label>{label}</Label>
		<!-- visible at rest: the tile's "this opens something" tell -->
		<span aria-hidden="true" class="bn-lens-child bn-dim font-mono text-[11px] leading-none group-hover:text-[var(--bn-text)]">↗</span>
	</div>
	<div
		class="flex flex-wrap items-baseline gap-[7px] font-mono text-[26px] font-semibold tracking-[-0.02em]"
		style:color={accent ? 'var(--bn-accent)' : 'var(--bn-text)'}
	>
		{@render value()}
		<small class="bn-dim whitespace-nowrap text-xs font-normal tracking-normal">{unit}</small>
	</div>
	{@render foot?.()}
</a>
