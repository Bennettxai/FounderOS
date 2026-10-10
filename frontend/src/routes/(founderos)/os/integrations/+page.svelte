<!-- The Connections board, FounderOS v1 app/integrations/page.tsx: the hero
     row (your connected tools + Connection Volume), the second row (by
     category, connector health, the Needs-you card), Popular, Browse by
     category and the API keys. Every number comes from
     GET /api/founderos/pages/connections: the live connector checks, the
     catalog merged onto them and the volume view-model computed server side.
     A stored key is never counted as connected; only a connector that
     answered is. The Optimal Engine tile replaces G-Brain. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle } from '$lib/founderos/kit';
	import ApiKeys from '$lib/founderos/pages/integrations/ApiKeys.svelte';
	import ConnectionCard from '$lib/founderos/pages/integrations/ConnectionCard.svelte';
	import IntegrationBrowser from '$lib/founderos/pages/integrations/IntegrationBrowser.svelte';
	import type { BrowseCategory, CatalogEntry, ConnectionsBoard } from '$lib/founderos/pages/integrations/types';

	const GRID = 'grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4';

	let board = $state<ConnectionsBoard | null>(null);
	let failure = $state<string | null>(null);
	let loading = $state(true);

	async function load() {
		try {
			board = await founderosFetch<ConnectionsBoard>('/pages/connections');
			failure = null;
		} catch (e) {
			failure = e instanceof Error ? e.message : 'unreachable';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load();
	});

	const v = $derived(board?.volume ?? null);
	const catalog = $derived(board?.catalog ?? []);
	const connected = $derived(catalog.filter((c) => c.connected));
	const popular = $derived(catalog.filter((c) => c.popular));
	const bySlug = $derived(new Map(catalog.map((c) => [c.slug, c])));
	const categories = $derived<BrowseCategory[]>(
		(board?.categories ?? [])
			.map((g) => {
				const entries = g.slugs.map((s) => bySlug.get(s)).filter((c): c is CatalogEntry => !!c);
				return { label: g.name, count: entries.length, connected: entries.filter((c) => c.connected).length, entries };
			})
			.filter((c) => c.count > 0)
	);
	const detailById = $derived(new Map((board?.connections ?? []).map((s) => [s.id, s.detail])));
	const guidanceFor = (e: CatalogEntry) => (e.connectorId ? detailById.get(e.connectorId) : undefined);
	const liveChip = $derived(v ? `${v.counts.connected} live${v.counts.error ? ` · ${v.counts.error} erroring` : ''}` : '');
	const meta = $derived(
		board && v
			? `${catalog.length} tools · ${v.counts.total} connector checks · live status, never a stored key alone`
			: loading
				? 'checking connectors…'
				: 'live status unavailable'
	);
	const reload = () => void load();
</script>

{#snippet card(e: CatalogEntry)}
	<ConnectionCard entry={e} guidance={guidanceFor(e)} oauth={board?.oauth[e.slug] ?? null} onchange={reload} />
{/snippet}

<Slab>
	<SlabTitle eyebrow="connections" title="Connections" {meta}>
		{#snippet right()}
			{#if v}<Chip tone={v.counts.error ? 'err' : 'ok'}>{liveChip}</Chip>{/if}
			<a href="#api-keys" data-lens="c" class={PILL}>API keys</a>
		{/snippet}
	</SlabTitle>

	{#if !board || !v}
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Your connected tools">
				<div class="px-6 pb-6 pt-3">
					{#if loading}
						<BigStat size={30} display="…" caption="checking connectors" />
					{:else}
						<BigStat size={30} value={null} caption={`connections unreachable: ${failure}`} />
					{/if}
				</div>
			</SlabCard>
			<SlabCard i={2} title="Connection Volume">
				<div class="px-6 pb-6 pt-3">
					<BigStat value={null} caption={loading ? 'checking connectors' : 'no reading: the board did not answer'} />
				</div>
			</SlabCard>
		</div>
	{:else}
		<!-- Hero row: what is live + the volume card -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<div data-part="connected-tools" class="contents">
				<SlabCard i={1} title="Your connected tools" sub={`${connected.length} of ${catalog.length}`}>
					<div class="px-6 pb-6 pt-3">
						<BigStat size={30} value={connected.length} caption="catalog tools whose connector answered on this load" />
						<div class="bn-rule-top mt-5 pt-5">
							{#if connected.length > 0}
								<div class={GRID}>{#each connected as e (e.slug)}{@render card(e)}{/each}</div>
							{:else}
								<div class="bn-dim text-[12.5px]">Nothing is connected yet. Pick a tool below and connect it.</div>
							{/if}
						</div>
					</div>
				</SlabCard>
			</div>

			<div data-part="volume" class="contents">
				<SlabCard i={2} title="Connection Volume" class="flex flex-col">
					<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
						<BigStat value={v.headline} chips={v.chips} caption={v.caption} />
						<MeterStack meters={v.meters} foot={v.foot} empty="no connector checks ran" />
					</div>
				</SlabCard>
			</div>
		</div>

		<!-- Second row: by category, connector health, the gradient card -->
		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<div data-part="by-category" class="contents">
				<SlabCard i={3} title="By Category" sub="connected tools">
					<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
						<BigStat
							size={30}
							display={v.topCategory ? v.topCategory.name : 'none'}
							caption={v.topCategory ? `most connected · ${v.topCategory.count} live` : 'no category has a live tool'}
						/>
						<DotMatrix cols={v.byCategory} hue="var(--bn-brain-2)" />
					</div>
				</SlabCard>
			</div>

			<div data-part="health" class="contents">
				<SlabCard i={4} title="Connector Health" sub="this load">
					<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
						<BigStat size={30} value={v.counts.total} caption="connector checks, live · unset · error" />
						<DotMatrix cols={v.health} hue="var(--bn-ok)" />
					</div>
				</SlabCard>
			</div>

			<InsightCard
				i={5}
				badge="Needs you"
				value={v.insight.value}
				headline={v.insight.headline}
				body={v.insight.body}
				frac={v.insight.frac}
			/>
		</div>

		<div data-part="popular" class="contents">
			<SlabCard i={6} title="Popular" sub={`${popular.length}`} class="mt-6">
				<div class="px-6 pb-6 pt-4">
					<div class={GRID}>{#each popular as e (e.slug)}{@render card(e)}{/each}</div>
				</div>
			</SlabCard>
		</div>

		<div data-part="browse" class="contents">
			<SlabCard i={7} title="Browse by category" sub={`${categories.length}`} class="mt-6">
				<div class="px-6 pb-6 pt-4">
					<IntegrationBrowser {categories} grid={GRID} guidance={guidanceFor} oauth={board.oauth} onchange={reload} />
				</div>
			</SlabCard>
		</div>

		<SlabCard i={8} class="mt-6">
			<div id="api-keys" class="scroll-mt-24 px-6 pb-6 pt-5">
				<ApiKeys keys={board.keys} onchange={reload} />
			</div>
		</SlabCard>
	{/if}
</Slab>

<style>
	.bn-rule-top {
		border-top: 1px solid var(--bn-border);
	}
</style>
