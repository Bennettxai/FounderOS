<!-- The skill catalog as one filtered card wall (FounderOS v1 components/SkillsGrid.tsx,
     mock 5g, 2026-09-24 a Brand Deals SlabCard): the card is titled "Skills"
     with "n of m" beside it and the All / Claude Code / Operator / Draft chips
     as its action; under the head sit the source note and the text filter,
     then the wall. Cards read eyebrow / status / title / two-line description
     / footer. A card opens its SKILL.md in a reader (fetched from
     /pages/skills/:slug unless the card carries it inline) with a download. -->
<script lang="ts">
	import type { Component } from 'svelte';
	import {
		ClipboardList,
		Clapperboard,
		CodeXml,
		Cog,
		Download,
		FilePenLine,
		Flame,
		Hammer,
		Image,
		Plug,
		SearchCheck,
		Sparkles,
		Target,
		X
	} from '$lib/founderos/icons';
	import { founderosFetch, founderosUrl } from '$lib/founderos/api';
	import { SlabCard, chipClass } from '$lib/founderos/kit';
	import SkillMarkdown from './SkillMarkdown.svelte';
	import { eyebrowOf, FILTERS, filterCards, iconKeyOf, matches, type Filter, type IconKey } from './skills';
	import type { SkillCard } from './types';

	let {
		cards,
		sourceNote,
		i = 6
	}: {
		cards: SkillCard[];
		sourceNote: string;
		i?: number;
	} = $props();

	// lucide-svelte ships legacy class components.
	type Icon = Component<{ class?: string; strokeWidth?: number }>;
	const ICONS: Record<IconKey, Icon> = {
		flame: Flame,
		code: CodeXml,
		image: Image,
		signature: FilePenLine,
		plug: Plug,
		hammer: Hammer,
		spec: ClipboardList,
		review: SearchCheck,
		target: Target,
		content: Clapperboard,
		ops: Cog,
		sparkles: Sparkles
	} as unknown as Record<IconKey, Icon>;
	const XIcon = X as unknown as Icon;
	const DownloadIcon = Download as unknown as Icon;

	const STATUS: Record<string, string> = { live: 'var(--bn-ok)', learning: 'var(--bn-warn)', planned: 'var(--bn-text-3)' };

	let filter = $state<Filter>('All');
	let query = $state('');
	let viewing = $state<SkillCard | null>(null);
	let md = $state<string | null>(null);

	const shown = $derived(filterCards(cards, filter, query));
	const counts = $derived(Object.fromEntries(FILTERS.map((f) => [f, cards.filter((c) => matches(c, f)).length])) as Record<Filter, number>);
	const ViewingIcon = $derived(viewing ? ICONS[iconKeyOf(viewing)] : ICONS.sparkles);

	async function open(card: SkillCard) {
		viewing = card;
		if (card.markdown != null) {
			md = card.markdown;
			return;
		}
		md = null; // loading
		try {
			const body = await founderosFetch<{ markdown?: string }>(`/pages/skills/${encodeURIComponent(card.id)}`);
			if (viewing?.id === card.id) md = body.markdown ?? 'SKILL.md could not be read (empty response).';
		} catch (e) {
			if (viewing?.id === card.id) md = `SKILL.md could not be read (${e instanceof Error ? e.message : 'unreachable'}).`;
		}
	}

	/** Operator skills carry their SKILL.md inline: save it as a file. */
	function downloadInline(card: SkillCard) {
		const blob = new Blob([card.markdown ?? ''], { type: 'text/markdown;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${card.id}-SKILL.md`;
		a.click();
		URL.revokeObjectURL(url);
	}

</script>

<svelte:document onkeydown={(e) => e.key === 'Escape' && (viewing = null)} />

<SlabCard {i} class="mt-6" title="Skills" sub={`${shown.length} of ${cards.length}`}>
	{#snippet action()}
		{#each FILTERS as f (f)}
			<button type="button" onclick={() => (filter = f)} data-lens="c" class={chipClass(filter === f)} aria-pressed={filter === f}
				>{f} {counts[f]}</button
			>
		{/each}
	{/snippet}
	<div class="flex flex-wrap items-center gap-3 px-6 pb-4 pt-4">
		<p class="bn-dim min-w-0 flex-1 font-mono text-[11px]">{sourceNote}</p>
		<input
			bind:value={query}
			placeholder="filter skills"
			class="bn-text h-[34px] w-52 rounded-full border px-4 font-mono text-[12px] bn-border placeholder:text-[var(--bn-text-3)] focus:border-[var(--bn-border-strong)] focus:outline-none"
			style="background: var(--bn-bg)"
		/>
	</div>

	<div class="border-t px-6 pb-6 pt-5 bn-border">
		{#if shown.length === 0}
			<p class="bn-dim py-4 text-center text-[12.5px]">
				no skills match{query.trim() ? ` "${query.trim()}"` : ''} · clear the filter to see all {cards.length}
			</p>
		{:else}
			<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
				{#each shown as c (c.id)}
					{@const CardIcon = ICONS[iconKeyOf(c)]}
					{@const status = c.status ?? 'live'}
					<button
						type="button"
						onclick={() => open(c)}
						title={`${c.name} · open SKILL.md`}
						data-lens="r"
						class="bn-pressable is-row group flex flex-col gap-2 rounded-[10px] border p-4 text-left bn-border"
						style="background: var(--bn-bg)"
					>
						<div class="flex items-center justify-between gap-2">
							<span class="bn-dim truncate font-mono text-[9.5px] uppercase tracking-[0.14em]">{eyebrowOf(c)}</span>
							<span class="bn-dim flex shrink-0 items-center gap-1.5 font-mono text-[9.5px] uppercase tracking-[0.1em]">
								<span data-part="status-dot" class="h-1.5 w-1.5 rounded-full" style={`background: ${STATUS[status]}`}></span>{status}
							</span>
						</div>
						<div class="flex items-center gap-2">
							<span class="bn-accent shrink-0"><CardIcon class="h-4 w-4" strokeWidth={1.8} /></span>
							<span class="bn-text min-w-0 flex-1 truncate text-[13.5px] font-bold">{c.name}</span>
						</div>
						<p class="bn-dim line-clamp-2 min-h-[30px] text-[11px] leading-snug">{c.description}</p>
						<div class="bn-dim truncate border-t pt-2 font-mono text-[9.5px] bn-border">{c.meta}</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</SlabCard>

<!-- The reader sits outside the SlabCard, whose rise/lift transform would
     otherwise become the containing block of position: fixed. -->
{#if viewing}
	{@const card = viewing}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-6 backdrop-blur-sm" onclick={() => (viewing = null)}>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			role="dialog"
			aria-modal="true"
			aria-label={`${card.name} SKILL.md`}
			tabindex="-1"
			class="flex max-h-[82vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border"
			style="border-color: var(--bn-border-strong); background: var(--bn-surface)"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex shrink-0 items-center justify-between border-b px-5 py-3 bn-border">
				<span class="bn-dim flex min-w-0 items-center gap-2 font-mono text-[11px]">
					<span class="bn-accent shrink-0"><ViewingIcon class="h-3.5 w-3.5" /></span>
					<span class="truncate">{card.filePath}</span>
				</span>
				<span class="flex shrink-0 items-center gap-3">
					{#if card.markdown == null}
						<a
							href={founderosUrl(`/pages/skills/${encodeURIComponent(card.id)}?download=1`)}
							title="Download SKILL.md"
							class="bn-pressable bn-dim flex items-center gap-1.5 rounded-md border px-2 py-1 font-mono text-[10px] uppercase tracking-widest bn-border hover:text-[var(--bn-text)]"
							><DownloadIcon class="h-3 w-3" />skill.md</a
						>
					{:else}
						<button
							type="button"
							onclick={() => downloadInline(card)}
							title="Download SKILL.md"
							class="bn-pressable bn-dim flex items-center gap-1.5 rounded-md border px-2 py-1 font-mono text-[10px] uppercase tracking-widest bn-border hover:text-[var(--bn-text)]"
							><DownloadIcon class="h-3 w-3" />skill.md</button
						>
					{/if}
					<button type="button" onclick={() => (viewing = null)} aria-label="Close" class="bn-pressable bn-dim shrink-0 hover:text-[var(--bn-text)]"><XIcon class="h-4 w-4" /></button>
				</span>
			</div>
			<div class="overflow-y-auto px-5 py-4">
				{#if md === null}
					<p class="bn-dim animate-pulse font-mono text-[11px]">loading SKILL.md…</p>
				{:else}
					<SkillMarkdown src={md} />
				{/if}
			</div>
		</div>
	</div>
{/if}
