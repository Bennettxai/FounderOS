<!-- Brand Deals: the GladOS "Deal Journeys" slab (FounderOS v1
     components/brand-deals/DealBoard.tsx) on the kit, fed by the Notion Brand
     Deals Hub through GET /api/founderos/pages/brand-deals. A hatched stepped
     funnel hero with a prompt bar melting out of it (a live filter), meters, a
     step line, a dot matrix, ONE insight card, the deal list and a detail
     drawer.

     Read-only by design: Notion stays the source of truth, so every deal
     deep-links back and nothing here writes. The server precomputes every
     figure; with no view (Notion unreadable) the figures read unknown. -->
<script lang="ts">
	import { CalendarClock, ExternalLink, RefreshCw, Search, Sparkles, X } from '$lib/founderos/icons';
	import { Badge, BigStat, CountUp, DotMatrix, InsightCard, Label, MeterStack, PILL_ACCENT, Slab, SlabCard, SlabTitle, StepLine, chipClass } from '$lib/founderos/kit';
	import CardMenu from './CardMenu.svelte';
	import PipelineChart from './PipelineChart.svelte';
	import {
		FILTER_TO_STAGE_IDX,
		HUE,
		badgeTone,
		dealUsd,
		filterChips,
		filterDeals,
		fmtUsd,
		parseQuery,
		statusTone,
		syncAge,
		timeAgo,
		volumeMeters,
		type BrandDeal,
		type BrandDealsBody,
		type Filter
	} from './view';

	let {
		body,
		onRefresh,
		refreshError = null
	}: { body: BrandDealsBody; onRefresh: () => void; refreshError?: string | null } = $props();

	let query = $state('');
	let stageFilter = $state<Filter>('all');
	let selectedId = $state<string | null>(null);

	const deals = $derived(body.deals);
	const view = $derived(body.view);
	const vol = $derived(view?.volume ?? null);
	const filtered = $derived(filterDeals(deals, { filter: stageFilter, query }));
	const slash = $derived(parseQuery(query).slash);
	const selected = $derived(deals.find((d) => d.id === selectedId) ?? null);
	const hubUrl = $derived(view?.hubUrl ?? null);
	const age = $derived(syncAge(body.syncedAt));
	const plural = (n: number, w: string) => `${n} ${w}${n === 1 ? '' : 's'}`;

	const metaText = $derived(
		[
			vol ? `${fmtUsd(vol.openUsd)} open pipeline` : 'open pipeline unknown',
			plural(deals.length, 'deal'),
			body.mode === 'live' ? `${deals.length} Notion rows${age ? ` · ${age}` : ''}` : body.detail
		].join(' · ')
	);

	const insightHeadline = $derived(
		view ? `${plural(view.due.followUps.length, 'follow-up')} due · ${plural(view.due.deadlines.length, 'deadline')} inside 7 days.` : 'Notion is unreadable, so what is due is unknown.'
	);
	const insightBody = $derived(
		view ? view.needsYouBrands.join(' · ') || 'Nothing due. The pipeline is waiting on brands, not on you.' : body.detail
	);
	const openCount = $derived(view?.openCount ?? null);

	function statusStyle(d: BrandDeal): string {
		const t = statusTone(d);
		const c = t === 'ok' ? 'var(--bn-ok)' : t === 'err' ? 'var(--bn-err)' : t === 'warn' ? 'var(--bn-warn)' : t === 'accent' ? HUE.cobalt : 'var(--bn-text-2)';
		return `background: color-mix(in oklab, ${c} 15%, transparent); color: ${c}`;
	}
	const isS = (d: BrandDeal) => (d.tier ?? '').trim().toUpperCase() === 'S';
	const moneyTitle = (d: BrandDeal) => (d.amountAgreedUsd != null ? 'amount agreed' : d.dealValueUsd != null ? 'deal value' : 'brand budget');
	const money = (n: number | null) => (n != null ? fmtUsd(n) : null);
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && (selectedId = null)} />

{#snippet menu(title: string)}
	<CardMenu {title} {onRefresh} {hubUrl} />
{/snippet}

{#snippet field(k: string, v: string | null)}
	<div class="bn-drawer-field flex items-baseline justify-between gap-4 py-2">
		<span class="bn-dim text-[12px]">{k}</span>
		<span class="text-right text-[12.5px] tabular-nums">{#if v}<span class="bn-text">{v}</span>{:else}<span class="bn-dim">not set</span>{/if}</span>
	</div>
{/snippet}

<Slab>
	<SlabTitle eyebrow="sponsorships · notion brand deals hub" title="Brand Deals">
		{#snippet meta()}
			<!-- full width (v1 DealBoard.tsx): a long seeded/error detail pushes the actions under it -->
			<span data-part="bd-meta">{metaText}</span>
		{/snippet}
		{#snippet right()}
			{#if body.mode === 'live'}
				<Badge tone="ok">live · Notion</Badge>
			{:else if body.mode === 'seeded'}
				<Badge tone="warn">seeded · connect Notion</Badge>
			{:else}
				<Badge tone="err">Notion error</Badge>
			{/if}
			{#if refreshError}
				<Badge tone="err">refresh failed · showing last read</Badge>
			{/if}
			<span class="bn-border bn-muted rounded-full border px-4 py-2 text-[13px]">read-only · refreshes 60s</span>
			{#if hubUrl}
				<a
					href={hubUrl}
					target="_blank"
					rel="noreferrer"
					class={PILL_ACCENT}
				>
					Open in Notion <ExternalLink size={12} strokeWidth={1.8} />
				</a>
			{/if}
			<button type="button" aria-label="Refresh" onclick={onRefresh} class="bn-pressable bn-border bn-muted grid h-10 w-10 place-items-center rounded-full border hover:text-[var(--bn-text)]">
				<RefreshCw size={15} strokeWidth={1.7} />
			</button>
		{/snippet}
	</SlabTitle>

	<!-- Hero row: funnel + volume card -->
	<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={1} title="Pipeline" class="pb-4">
			{#snippet action()}{@render menu('Pipeline')}{/snippet}
			{#if view}
				<PipelineChart stages={view.stages} active={FILTER_TO_STAGE_IDX[stageFilter]} onSelect={(s) => (stageFilter = s.filter)}>
					{#snippet search()}
						<div class="bn-muted mb-2 flex items-center gap-2 px-1 text-[13px]">
							<Sparkles size={14} strokeWidth={1.7} class="bn-accent" />
							What are you looking for?
						</div>
						<label class="bn-border flex items-center gap-2 rounded-[12px] border px-3.5 py-2.5" style="background: var(--bn-bg)">
							<Search size={14} strokeWidth={1.7} class="bn-dim shrink-0" />
							<input
								bind:value={query}
								aria-label="Filter deals"
								placeholder="Filter deals: a brand, a contact, a channel, or /talks /production /paid /declined /s"
								class="bn-text w-full bg-transparent text-[13.5px] outline-none placeholder:text-[var(--bn-text-3)]"
							/>
							{#if slash}
								<span
									data-part="slash"
									class="shrink-0 rounded-md border px-2 py-0.5 font-mono text-[11.5px]"
									style="border-color: color-mix(in oklab, var(--bn-warn) 45%, transparent); background: color-mix(in oklab, var(--bn-warn) 12%, transparent); color: var(--bn-warn)"
									>{slash}</span
								>
							{/if}
							<span class="bn-bd-caret h-[15px] w-[1.5px] shrink-0" style="background: var(--bn-accent)"></span>
						</label>
					{/snippet}
				</PipelineChart>
			{:else}
				<div data-part="pipeline-unknown" class="bn-dim px-6 py-16 text-[12.5px]">Pipeline unknown: {body.detail}</div>
			{/if}
		</SlabCard>

		<SlabCard i={2} title="Deal Volume" class="flex flex-col">
			{#snippet action()}{@render menu('Deal Volume')}{/snippet}
			<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
				<BigStat
					value={vol?.openUsd ?? null}
					kind="usd"
					chips={[
						...(vol && vol.paidUsd > 0 ? [{ tone: 'ok' as const, text: `${fmtUsd(vol.paidUsd)} paid` }] : []),
						...(vol && vol.declinedUsd > 0 ? [{ tone: 'err' as const, text: `${fmtUsd(vol.declinedUsd)} declined` }] : [])
					]}
					caption={vol && openCount != null
						? `open pipeline across ${plural(openCount, 'deal')} · ${vol.quotedDeals} of ${deals.length} priced`
						: 'open pipeline unknown until Notion answers'}
				/>
				<MeterStack meters={volumeMeters(view)} foot={vol ? `open = talks + production · ${vol.counts.tierS} S-tier` : 'open = talks + production'} />
			</div>
		</SlabCard>
	</div>

	<!-- Second row: activity line + sizes + THE insight card -->
	<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={3} title="Deal Activity">
			{#snippet action()}{@render menu('Deal Activity')}{/snippet}
			<div data-part="activity-stat" class="px-6 pt-3">
				<span class="bn-text text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{#if view}<CountUp value={view.editsInWindow} />{:else}<span data-unknown class="bn-dim">unknown</span>{/if}</span>
				<span class="bn-dim ml-2 text-[13px]">Notion edits, last 30 days</span>
			</div>
			<StepLine series={view?.activity ?? []} hue={HUE.activity} empty={view ? 'No edits in this window.' : 'Edit activity unknown.'} />
		</SlabCard>

		<SlabCard i={4} title="Deal Sizes">
			{#snippet action()}{@render menu('Deal Sizes')}{/snippet}
			<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
				<div data-part="sizes-stat">
					<div class="bn-text text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{#if vol}<CountUp value={vol.quotedDeals} />{:else}<span data-unknown class="bn-dim">unknown</span>{/if}</div>
					<div class="bn-dim mt-1 text-[13px]">priced deals</div>
					<div class="bn-border bn-muted mt-4 rounded-full border px-3 py-1 text-[12px]">
						Largest: <span class="font-semibold tabular-nums">{view ? fmtUsd(view.largestUsd) : 'unknown'}</span>
					</div>
				</div>
				{#if view}<DotMatrix cols={view.sizes} hue={HUE.cobalt} />{/if}
			</div>
		</SlabCard>

		<InsightCard
			i={5}
			badge="Needs you this week"
			value={view?.needsYou ?? null}
			headline={insightHeadline}
			body={insightBody}
			frac={view ? view.needsYouFrac : null}
		/>
	</div>

	<!-- Deal list -->
	<SlabCard i={6} title="Deals" sub="{filtered.length} of {deals.length}" class="mt-6">
		{#snippet action()}
			{#each filterChips(view) as [key, label] (key)}
				<button
					type="button"
					data-filter={key}
					aria-pressed={stageFilter === key}
					onclick={() => (stageFilter = key)}
					class={chipClass(stageFilter === key)}
				>
					{label}
				</button>
			{/each}
		{/snippet}
		<div class="bn-border mt-4 border-t" data-part="deal-list">
			{#each filtered as d (d.id)}
				{@const usd = dealUsd(d)}
				<button
					type="button"
					data-deal={d.id}
					onclick={() => (selectedId = d.id)}
					class="bn-deal-row bn-pressable bn-border grid w-full grid-cols-[220px_1fr_auto_auto_auto] items-center gap-5 border-b px-6 py-3.5 text-left last:border-0 max-[1000px]:grid-cols-[1fr_auto]"
				>
					<div class="min-w-0">
						<div class="bn-text truncate text-[13.5px] font-medium">{d.brand}</div>
						<div class="bn-dim truncate font-mono text-[11px]">{d.contactName ?? d.contactEmail ?? d.source ?? 'no contact yet'}</div>
					</div>
					<div class="min-w-0 max-[1000px]:hidden">
						<div class="bn-muted truncate text-[12.5px]">{[d.videoType, d.mainChannel].filter(Boolean).join(' on ') || d.status}</div>
					</div>
					<div class="flex items-center gap-2">
						{#if usd > 0}
							<span class="bn-accent rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]" style="background: var(--bn-accent-soft)" title={moneyTitle(d)}>{fmtUsd(usd)}</span>
						{/if}
						{#if isS(d)}
							<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style="background: color-mix(in oklab, {HUE.violet} 18%, transparent); color: {HUE.violet}">S-tier</span>
						{/if}
						<span data-part="status" class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={statusStyle(d)}>{d.status}</span>
					</div>
					<span class="bn-dim inline-flex w-[72px] items-center gap-1.5 font-mono text-[11px] tabular-nums max-[1000px]:hidden">
						{#if d.followUpDate}<CalendarClock size={12} strokeWidth={1.7} />{d.followUpDate.slice(5)}{/if}
					</span>
					<span class="bn-dim w-[64px] text-right font-mono text-[11px] max-[1000px]:hidden">{timeAgo(d.lastEdited)}</span>
				</button>
			{/each}
			{#if filtered.length === 0}
				<div data-part="list-empty" class="bn-dim px-6 py-8 text-center text-[12.5px]">
					{#if body.mode === 'error'}Notion could not be read, so no deals are shown.{:else if deals.length === 0}The Brand Deals Hub has no deals yet.{:else}Nothing matches that filter.{/if}
				</div>
			{/if}
		</div>
	</SlabCard>
</Slab>

<!-- Detail drawer -->
{#if selected}
	<button type="button" aria-label="Close deal" class="fixed inset-0 z-[60] cursor-default bg-black/45 backdrop-blur-[2px]" onclick={() => (selectedId = null)}></button>
	<aside data-part="drawer" class="bn-border fixed right-0 top-0 z-[70] flex h-full w-[460px] max-w-[92vw] flex-col border-l backdrop-blur" style="background: color-mix(in oklab, var(--bn-bg-2) 95%, transparent)">
		<div class="bn-border flex items-start justify-between gap-4 border-b px-6 py-5">
			<div class="min-w-0">
				<div class="bn-text truncate text-[17px] font-semibold">{selected.brand}</div>
				<div class="bn-dim mt-0.5 truncate font-mono text-[11.5px]">{selected.contactEmail ?? selected.contactName ?? 'no contact on the row'}</div>
				<div class="mt-2.5 flex flex-wrap items-center gap-2">
					{#if selected.tier}<Badge tone={isS(selected) ? 'accent' : 'default'}>tier {selected.tier}</Badge>{/if}
					<Badge tone={badgeTone(selected)}>{selected.status}</Badge>
					{#if selected.paidInFull}<Badge tone="ok">paid in full</Badge>{/if}
					{#if selected.seeded}<Badge tone="warn">example row</Badge>{/if}
				</div>
			</div>
			<button type="button" aria-label="Close" onclick={() => (selectedId = null)} class="bn-pressable bn-border bn-muted grid h-[30px] w-[30px] shrink-0 place-items-center rounded-[5px] border hover:text-[var(--bn-text)]">
				<X size={15} strokeWidth={1.8} />
			</button>
		</div>
		<div class="min-h-0 flex-1 overflow-y-auto px-6 py-5">
			<section class="mb-6">
				<a
					href={selected.notionUrl}
					target="_blank"
					rel="noreferrer"
					class="inline-flex items-center gap-1.5 rounded-[9px] px-3.5 py-1.5 text-[12.5px] font-medium"
					style="background: var(--bn-accent); color: var(--bn-accent-ink)"
				>
					<ExternalLink size={13} strokeWidth={1.8} /> Open in Notion
				</a>
				<p class="bn-dim mt-2.5 text-[11px]">Notion is the source of truth. Edit the deal there; this board rereads it on refresh or within 20 minutes.</p>
			</section>
			<section class="mb-6">
				<Label rule>Money</Label>
				<div class="mt-2">
					{@render field('Amount agreed', money(selected.amountAgreedUsd))}
					{@render field('Deal value', money(selected.dealValueUsd))}
					{@render field('Brand budget', money(selected.budgetUsd))}
					{@render field('Suggested rate', money(selected.suggestedRateUsd))}
				</div>
			</section>
			<section class="mb-6">
				<Label rule>Dates</Label>
				<div class="mt-2">
					{@render field('Follow-up', selected.followUpDate)}
					{@render field('Deadline', selected.deadline)}
					{@render field('Last edited', `${timeAgo(selected.lastEdited)} (${selected.lastEdited.slice(0, 10)})`)}
				</div>
			</section>
			<section>
				<Label rule>Fit</Label>
				<div class="mt-2">
					{@render field('Channel', selected.mainChannel)}
					{@render field('Format', selected.videoType)}
					{@render field('Source', selected.source)}
					{@render field('ICP fit', selected.icpFit)}
				</div>
			</section>
		</div>
	</aside>
{/if}

<style>
	.bn-deal-row:hover {
		background: color-mix(in oklab, var(--bn-text) 4%, transparent);
	}
	.bn-drawer-field {
		border-bottom: 1px solid var(--bn-hairline);
	}
	.bn-drawer-field:last-child {
		border-bottom: 0;
	}
	.bn-bd-caret {
		animation: bn-bd-caret 1.1s step-end infinite;
	}
	@keyframes bn-bd-caret {
		0%,
		45% {
			opacity: 1;
		}
		55%,
		100% {
			opacity: 0;
		}
	}
</style>
