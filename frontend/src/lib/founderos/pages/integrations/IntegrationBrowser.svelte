<!-- "Browse by category" (FounderOS v1 components/IntegrationBrowser +
     IntegrationCategory): filter pills over the collapsible categories. "All"
     is the accordion with the first category open; a category pill narrows
     the list to that one, opened. -->
<script lang="ts">
	import { chipClass } from '$lib/founderos/kit';
	import ConnectionCard from './ConnectionCard.svelte';
	import type { BrowseCategory, CatalogEntry, OAuthReadiness } from './types';

	let {
		categories,
		grid,
		guidance,
		oauth,
		onchange
	}: {
		categories: BrowseCategory[];
		grid: string;
		guidance: (e: CatalogEntry) => string | undefined;
		oauth: Record<string, OAuthReadiness>;
		onchange?: () => void;
	} = $props();

	let active = $state('all');
	/** Open state per filter view; a fresh view opens its first (or only) category. */
	let opened = $state<Record<string, boolean>>({});

	const shown = $derived(active === 'all' ? categories : categories.filter((c) => c.label === active));
	const total = $derived(categories.reduce((n, c) => n + c.count, 0));
	const key = (label: string) => `${active}:${label}`;
	const isOpen = (label: string, idx: number) => opened[key(label)] ?? (active !== 'all' || idx === 0);

	function pick(next: string) {
		active = next;
		opened = {};
	}
</script>

<div>
	<div class="mb-4 flex flex-wrap gap-2">
		<button type="button" class={chipClass(active === 'all')} aria-pressed={active === 'all'} onclick={() => pick('all')}>All · {total}</button>
		{#each categories as c (c.label)}
			<button
				type="button"
				class={chipClass(active === c.label)}
				aria-pressed={active === c.label}
				title={`${c.connected} of ${c.count} connected`}
				onclick={() => pick(c.label)}>{c.label} · {c.count}</button
			>
		{/each}
	</div>
	<div class="flex flex-col gap-2.5">
		{#each shown as c, idx (key(c.label))}
			{@const open = isOpen(c.label, idx)}
			<div class="bn-cat rounded-[var(--bn-r-panel)]">
				<button
					type="button"
					data-part="category-toggle"
					data-lens="r"
					aria-expanded={open}
					class="bn-pressable flex w-full items-center gap-3 px-4 py-3.5 text-left"
					onclick={() => (opened = { ...opened, [key(c.label)]: !open })}
				>
					<span aria-hidden="true" class="bn-dim bn-chev shrink-0 font-mono text-[11px] leading-none" class:is-open={open}>▸</span>
					<span class="bn-text flex-1 text-[13px] font-semibold">{c.label}</span>
					<span class="bn-count bn-dim grid h-5 min-w-5 place-items-center rounded-full px-1.5 font-mono text-[10px]">{c.count}</span>
				</button>
				{#if open}
					<div class="px-4 pb-4 pt-1">
						<div class={grid}>
							{#each c.entries as e (e.slug)}<ConnectionCard entry={e} guidance={guidance(e)} oauth={oauth[e.slug] ?? null} {onchange} />{/each}
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>
</div>

<style>
	.bn-cat {
		border: 1px solid var(--bn-border);
		background: color-mix(in oklab, var(--bn-surface) 40%, transparent);
	}
	.bn-count {
		border: 1px solid var(--bn-border);
	}
	.bn-chev {
		transition: transform 200ms;
	}
	.bn-chev.is-open {
		transform: rotate(90deg);
	}
</style>
