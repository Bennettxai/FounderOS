<!-- /os/funnel (spec 6.8): FounderOS v1 app/funnel/page.tsx. Vantage +
     Launchpad Cohort client journeys, composed on the backend (GET
     /pages/funnel; live lanes when set up, the seeded funnel otherwise) on
     v1's five stages: First touch → Engaged → Nurtured → Opted in → Converted.
     Below the lazy-loaded graph, v1's three cards (Funnel Volume, Lead
     Activity, Needs you today), the Push Now / Save Now rails and the
     Journeys list. The archive tab lists what decayed. -->
<script lang="ts">
	import { safeHref } from '$lib/founderos/safe-href';
	import { onMount } from 'svelte';
	import { replaceState } from '$app/navigation';
	import { founderosFetch } from '$lib/founderos/api';
	import { Badge, BigStat, InsightCard, MeterStack, Slab, SlabCard, SlabTitle, StepLine, chipClass } from '$lib/founderos/kit';
	import FunnelGraph from './FunnelGraph.svelte';
	import './tokens.css';
	import type { CommsItem, FunnelBody, FunnelStage, FunnelVenture, Journey } from './types';
	import { funnelHref, funnelQuery, parseFunnelView, type FunnelView } from './url';
	import { CHANNEL_GLYPHS, usd } from './viz';

	let view = $state<FunnelView>({ venture: null, view: 'live', stage: null, layout: 'flow', lead: null });
	let ready = $state(false);
	let data = $state<FunnelBody | null>(null);
	let failure = $state<string | null>(null);
	let loading = $state(true);

	onMount(() => {
		view = parseFunnelView(window.location.search);
		ready = true;
	});

	// Venture and stage are backend filters: refetch when they change.
	const query = $derived(ready ? funnelQuery(view) : null);
	$effect(() => {
		const q = query;
		if (!q) return;
		let alive = true;
		loading = true;
		failure = null;
		founderosFetch<FunnelBody>(q)
			.then((b) => {
				if (alive) data = b;
			})
			.catch((e: unknown) => {
				if (alive) failure = e instanceof Error ? e.message : 'unreachable';
			})
			.finally(() => {
				if (alive) loading = false;
			});
		return () => {
			alive = false;
		};
	});

	function go(patch: Partial<FunnelView>) {
		view = { ...view, ...patch };
		try {
			replaceState(funnelHref(view), {});
		} catch {
			// the router is not ready (tests): the view state still moved
		}
	}

	const VENTURE_TABS: { id: FunnelVenture | null; label: string }[] = [
		{ id: null, label: 'All clients' },
		{ id: 'vantage', label: 'Vantage' },
		{ id: 'launchpad-cohort', label: 'Launchpad Cohort' }
	];
	// prod lib/ventures.ts brand hues (Vantage logo green, LC site crimson)
	const VENTURE_COLOR: Record<FunnelVenture, string> = { vantage: '#00ffaa', 'launchpad-cohort': '#d9263f' };
	/** prod's rounded-ctl pressable control (venture tabs, Archive). */
	const CTL = 'fn-ctl bn-pressable rounded-[var(--bn-r-ctl)]';
	/** prod PILL_BASE: round status pills. */
	const TAG = 'fn-tag rounded-full';
	const VENTURE_LABEL: Record<FunnelVenture, string> = { vantage: 'Vantage', 'launchpad-cohort': 'Launchpad Cohort' };
	const REL: Record<string, string> = { hot: 'var(--fn-hot)', warm: 'var(--fn-warm)', cold: 'var(--fn-cold)' };

	const summary = $derived(data?.summary);
	const now = $derived(data ? new Date(data.now) : new Date());
	const decayOf = (days: number, status: FunnelStage) =>
		!data || status === 'converted' ? 0 : Math.min(1, Math.max(0, (days - data.constants.decayFadeStart) / (data.constants.decayDays - data.constants.decayFadeStart)));
	const metaOf = (j: Journey) => {
		const last = j.touches.at(-1)?.at ?? j.createdAt;
		const days = Math.max(0, Math.floor((now.getTime() - new Date(`${last}T00:00:00Z`).getTime()) / 86_400_000));
		const canStall = j.status !== 'converted' && j.status !== 'first_touch';
		const state = j.status === 'converted' ? 'converted' : days > (data?.constants.decayDays ?? 90) ? 'decayed' : canStall && days > (data?.constants.stallDays ?? 7) ? 'stalled' : 'active';
		return { days, state };
	};
	const stagePill = (j: Journey) => {
		const s = metaOf(j).state;
		if (s === 'converted') return 'background: color-mix(in oklab, var(--bn-ok) 16%, transparent); color: var(--bn-ok)';
		if (s === 'stalled' || s === 'decayed') return 'background: color-mix(in oklab, var(--bn-err) 16%, transparent); color: var(--bn-err)';
		return 'background: color-mix(in oklab, var(--bn-text) 8%, transparent); color: var(--bn-text-2)';
	};
	const quietStyle = (j: Journey, days: number) => {
		const d = decayOf(days, j.status);
		return d > 0 ? `color: color-mix(in oklab, var(--bn-err) ${Math.round(Math.sqrt(d) * 85)}%, var(--bn-text-2))` : '';
	};
	const laneOf = (j: Journey) => {
		const src = j.touches[0]?.source;
		return src === 'attio' || src === 'ghl' || src === 'typeform' || src === 'fathom' || src === 'calendar' ? 'crm' : j.touches[0]?.channel === 'ads' ? 'ads' : 'organic';
	};
	const agoDays = (ts: string) => {
		const d = Math.max(0, Math.floor((now.getTime() - Date.parse(ts)) / 86_400_000));
		return d === 0 ? 'today' : `${d}d ago`;
	};
	/** Retired CRM history reads its source, not a quiet count (prod archive row). */
	const crmSource = (j: Journey): string | null =>
		j.touches.some((t) => t.source === 'attio') ? 'Attio' : j.touches.some((t) => t.source === 'ghl') ? 'GoHighLevel' : null;
	const leadInSpace = $derived(data && view.lead && data.journeys.some((j) => j.id === view.lead) ? view.lead : null);
	const lastMsgFor = (id: string): CommsItem | null | undefined => (data?.lastMessages ? (data.lastMessages[id]?.message ?? null) : undefined);
</script>

{#snippet valuePill(amount: number | null)}
	<span class="{TAG}" style="background: var(--bn-accent-soft); color: var(--bn-accent); letter-spacing: 0.04em">{amount != null ? usd(amount) : 'value unknown'}</span>
{/snippet}

{#snippet contactActions(j: Journey)}
	{@const digits = j.phone?.replace(/[^\d]/g, '')}
	<span class="flex items-center gap-2.5 font-mono text-[10.5px] uppercase tracking-wide">
		{#if j.email}<a href="mailto:{j.email}" title={j.email} class="bn-muted hover:text-[var(--bn-accent)]">email</a>{/if}
		{#if digits}<a href="https://wa.me/{digits}" target="_blank" rel="noopener noreferrer" title={j.phone ?? undefined} class="bn-muted hover:text-[var(--bn-accent)]">wa</a>{/if}
		{#if j.phone}<a href="sms:{j.phone}" title={j.phone} class="bn-muted hover:text-[var(--bn-accent)]">sms</a>{/if}
		{#if safeHref(j.url)}<a href={safeHref(j.url)} target="_blank" rel="noopener noreferrer" class="bn-dim hover:text-[var(--bn-accent)]">{j.url?.includes('fathom') ? 'call↗' : 'open↗'}</a>{/if}
		{#if !j.email && !j.phone && !j.url}<span class="bn-dim">no contact</span>{/if}
	</span>
{/snippet}

<!-- v1 AttentionRow: clicking it pins that lead's dossier in the canvas. -->
{#snippet attentionRow(j: Journey)}
	{@const m = metaOf(j)}
	<button
		type="button"
		data-lens="r"
		class="fn-attn bn-pressable grid w-full grid-cols-[1fr_auto_auto] items-center gap-4 border-b px-6 py-3.5 text-left last:border-0"
		style="border-color: var(--bn-border)"
		onclick={() => go({ lead: j.id, view: 'live' })}
	>
		<span class="min-w-0">
			<span class="fn-attn-name bn-text block truncate text-[13.5px] font-medium">{j.person ?? j.name}</span>
			<span class="bn-dim block truncate font-mono text-[11px]">{[j.company, `${j.likelihood}% likely`].filter(Boolean).join(' · ')}</span>
		</span>
		<span class="flex items-center gap-2">
			{#if (j.amountUsd ?? 0) > 0}{@render valuePill(j.amountUsd)}{/if}
			<span class={TAG} style={stagePill(j)}>{data?.stageLabels[j.status] ?? j.status}</span>
		</span>
		<span class="w-[44px] text-right font-mono text-[11px] tabular-nums" style={quietStyle(j, m.days) || 'color: var(--bn-text-3)'}>{m.days}d</span>
	</button>
{/snippet}

{#snippet journeyRow(j: Journey)}
	{@const converted = j.status === 'converted'}
	{@const m = metaOf(j)}
	{@const lane = laneOf(j)}
	{@const lastMsg = lastMsgFor(j.id)}
	<div data-lens="r" class="fn-row border-b px-6 py-3.5 last:border-0" style="border-color: var(--bn-border)" data-testid="journey-row">
		<div class="grid grid-cols-[minmax(200px,1fr)_auto_auto_auto_auto] items-center gap-5 max-[1000px]:grid-cols-[1fr_auto]">
			<span class="flex min-w-0 items-center gap-2.5">
				<span data-testid="venture-dot" class="h-2 w-2 shrink-0 rounded-full" style="background: {VENTURE_COLOR[j.venture]}" title={VENTURE_LABEL[j.venture]}></span>
				<span class="min-w-0">
					<span class="bn-text block truncate text-[13.5px] font-medium" title={j.name}>{j.person ?? j.name}</span>
					<span class="bn-dim block truncate font-mono text-[11px]">{[j.company, converted && j.product ? j.product : null].filter(Boolean).join(' · ') || lane}</span>
				</span>
			</span>
			<span class="flex items-center gap-2">
				{#if converted}{@render valuePill(j.amountUsd)}{/if}
				<span class={TAG} style="background: color-mix(in oklab, {REL[j.relationship]} 16%, transparent); color: {REL[j.relationship]}">{j.relationship}</span>
				<span class={TAG} style={stagePill(j)}>{data?.stageLabels[j.status] ?? j.status}</span>
			</span>
			<span class="w-[64px] font-mono text-[11px] tabular-nums max-[1000px]:hidden" style={m.state === 'stalled' && decayOf(m.days, j.status) === 0 ? 'color: var(--bn-err)' : quietStyle(j, m.days) || 'color: var(--bn-text-3)'} title="days since the last touch">{m.days}d quiet</span>
			<span class="bn-muted w-[72px] font-mono text-[11px] tabular-nums max-[1000px]:hidden" title="likelihood to buy">{j.likelihood}% likely</span>
			<span class="max-[1000px]:hidden">{@render contactActions(j)}</span>
		</div>
		<div class="mt-2.5 flex flex-wrap items-center gap-1.5 pl-[18px]">
			<span class="bn-dim mr-1 font-mono text-[10.5px] uppercase tracking-wide" title="entry lane">{lane}</span>
			{#each j.touches as t, i (t.id + i)}
				<span class="flex items-center gap-1.5">
					{#if i > 0}<span class="bn-dim font-mono text-[10px]">→</span>{/if}
					<span class="inline-flex max-w-[260px] items-center gap-1.5 rounded-full border px-2.5 py-1" style="border-color: var(--bn-border)" title="{t.stage} · via {t.source} · {t.at}">
						<span class="bn-accent shrink-0 font-mono text-[10.5px]">{CHANNEL_GLYPHS[t.channel] ?? '·'}</span>
						<span class="bn-muted truncate text-[11.5px]">{t.label}</span>
						<span class="bn-dim shrink-0 font-mono text-[10px]">{t.at.slice(5)}</span>
					</span>
				</span>
			{/each}
			{#if !converted}<span class="bn-dim font-mono text-[10px]">→ …</span>{/if}
			{#if lastMsg !== undefined}
				<span class="ml-auto flex min-w-0 items-center gap-1.5 font-mono text-[10.5px]">
					<span class="bn-dim shrink-0 uppercase tracking-wide">last msg</span>
					{#if lastMsg}
						<span class="bn-muted min-w-0 truncate" title={lastMsg.preview}>via {lastMsg.source} · {agoDays(lastMsg.ts)} · “{lastMsg.preview.slice(0, 60)}”</span>
					{:else}
						<span class="bn-dim">no thread on record</span>
					{/if}
				</span>
			{/if}
		</div>
	</div>
{/snippet}

<div class="bn-fn">
	<Slab>
		<SlabTitle
			eyebrow="client journeys · leads → conversations → sales"
			title="Funnel"
			meta={summary && data
				? `${summary.clients} active client${summary.clients === 1 ? '' : 's'} · ${summary.converted} closed · ${usd(summary.revenueUsd)} revenue · ${data.archived.length} archived`
				: loading
					? 'composing journeys…'
					: 'funnel unavailable'}
		>
			{#snippet right()}
				{#if data}
					<div class="flex flex-wrap items-center gap-2.5" data-testid="funnel-title-badges">
						{#if data.isLive}
							<Badge tone="ok">live · {data.liveLabel}</Badge>
						{:else}
							<Badge tone="warn" ghost>demo data</Badge>
						{/if}
						<span class="bn-muted rounded-full border px-4 py-2 text-[13px] tabular-nums" style="border-color: var(--bn-border)">{data.summary.converted}/{data.summary.clients} converted · {usd(data.summary.revenueUsd)}</span>
					</div>
				{/if}
			{/snippet}
		</SlabTitle>

		{#if failure && !data}
			<SlabCard title="Funnel unavailable" i={1}>
				<p class="px-6 pb-6 pt-3 text-[13px]" style="color: var(--bn-err)" role="alert">Could not compose the funnel: {failure}</p>
			</SlabCard>
		{:else if !data}
			<div class="grid gap-4" data-testid="funnel-loading">
				<div class="w-full animate-pulse rounded-[var(--bn-r-panel)] border" style="aspect-ratio: 1100 / 460; border-color: var(--bn-border); background: var(--bn-surface)"></div>
			</div>
		{:else}
			{#if data.seedError}
				<p role="alert" class="mb-3 font-mono text-[11px]" style="color: var(--bn-warn)">No live lane answered and the seeded funnel could not be read: {data.seedError}</p>
			{/if}

			<!-- one control line: venture filter · synced sources · view toggle -->
			<div class="bn-rise mb-3 flex flex-wrap items-center gap-x-3 gap-y-1.5" style="--rise-i: 1">
				<span class="flex items-center gap-1.5" role="group" aria-label="Venture">
					{#each VENTURE_TABS as tab (tab.label)}
						<button
							data-lens="c"
							class={CTL}
							data-on={(view.venture ?? null) === tab.id ? '' : undefined}
							aria-pressed={(view.venture ?? null) === tab.id}
							title={tab.id && data.isLive ? 'Live split = name heuristic: Vantage when the form or call names it' : undefined}
							onclick={() => go({ venture: tab.id, lead: null })}
						>
							{#if tab.id}<span class="mr-1 inline-block h-1.5 w-1.5 rounded-full align-middle" style="background: {VENTURE_COLOR[tab.id]}"></span>{/if}{tab.label}
						</button>
					{/each}
				</span>
				<span class="h-3 w-px" style="background: var(--bn-border)"></span>
				<span class="flex items-center gap-2.5" title="leads · calls booked · calls held · attribution · paid" data-testid="funnel-sources">
					{#each data.sources as s (s.id)}
						{@const ok = s.state === 'connected'}
						<span
							title={data.laneErrors[s.id] ? `${s.detail} · lane: ${data.laneErrors[s.id]}` : s.detail}
							class="inline-flex items-center gap-1 font-mono text-[9.5px] uppercase tracking-[0.12em]"
							style="color: {ok ? (s.live ? 'var(--bn-ok)' : 'var(--bn-text-2)') : 'var(--bn-text-3)'}"
						>
							{ok ? '✓' : '○'} {s.name}{s.live && s.count != null ? ` ${s.count}` : ''}
						</span>
					{/each}
				</span>
				<span class="ml-auto flex items-center gap-1.5">
					<nav aria-label="Funnel graph view" class="relative isolate grid grid-cols-2 rounded-[var(--bn-r-ctl)] border p-0.5 font-mono text-[10px] uppercase tracking-wide" style="border-color: var(--bn-border)">
						<span
							aria-hidden="true"
							data-testid="layout-slider"
							class="pointer-events-none absolute bottom-0.5 left-0.5 top-0.5 -z-10 w-[calc(50%-2px)] rounded-[calc(var(--bn-r-ctl)-2px)] transition-transform duration-200 motion-reduce:transition-none {view.view === 'archive' ? 'opacity-40' : ''}"
							style="background: var(--bn-accent-soft); transform: translateX({view.layout === 'radial' ? '100%' : '0%'})"
						></span>
						{#each [{ id: 'flow', label: 'Flow' }, { id: 'radial', label: 'Radial' }] as const as opt (opt.id)}
							<button
								data-lens="c"
								class="bn-pressable rounded-[calc(var(--bn-r-ctl)-2px)] px-3 py-1.5 text-center font-mono text-[10px] uppercase tracking-wide"
								style={view.layout === opt.id ? 'color: var(--bn-accent)' : 'color: var(--bn-text-3)'}
								aria-current={view.view === 'live' && view.layout === opt.id ? 'page' : undefined}
								onclick={() => go({ layout: opt.id, view: 'live' })}>{opt.label}</button
							>
						{/each}
					</nav>
					<span class="mx-0.5 h-3 w-px" style="background: var(--bn-border)"></span>
					<button
						data-lens="c"
						class={CTL}
						data-on={view.view === 'archive' ? '' : undefined}
						title="Leads quiet past the decay window rest here"
						onclick={() => go({ view: view.view === 'archive' ? 'live' : 'archive' })}>Archive ({data.archived.length})</button
					>
				</span>
			</div>

			{#if view.view === 'archive'}
				<SlabCard title="Archive" sub="{data.archived.length} quiet past {data.constants.decayDays} days" i={3}>
					<div class="mt-4 border-t" style="border-color: var(--bn-border)" data-testid="funnel-archive">
						{#if data.archived.length === 0}
							<p class="bn-dim px-6 py-8 text-center text-[12.5px]">Nothing decayed · no lead has sat quiet past {data.constants.decayDays} days.</p>
						{:else}
							{#each data.archived as j (j.id)}
								{@const m = metaOf(j)}
								{@const last = j.touches.at(-1)}
								{@const crm = crmSource(j)}
								<div class="grid grid-cols-[minmax(200px,1fr)_auto_auto_auto] items-center gap-5 border-b px-6 py-3.5 last:border-0 max-[1000px]:grid-cols-[1fr_auto]" style="border-color: var(--bn-border)">
									<span class="flex min-w-0 items-center gap-2.5">
										<span data-testid="venture-dot" class="h-2 w-2 shrink-0 rounded-full opacity-60" style="background: {VENTURE_COLOR[j.venture]}"></span>
										<span class="min-w-0">
											<span class="bn-muted block truncate text-[13.5px] font-medium">{j.name}</span>
											<span class="bn-dim block truncate font-mono text-[11px]" title={last?.label}>{last?.label ?? 'no touches'}</span>
										</span>
									</span>
									<span class={TAG} style={stagePill(j)}>{data.stageLabels[j.status] ?? j.status}</span>
									<span class="bn-dim font-mono text-[11px] tabular-nums max-[1000px]:hidden">{crm ? `${crm} · archived` : `${m.days}d quiet · ${j.likelihood}% likely`}</span>
									<span class="w-[56px] text-right max-[1000px]:hidden">
										{#if safeHref(j.url)}<a href={safeHref(j.url)} target="_blank" rel="noopener noreferrer" class="bn-dim font-mono text-[10.5px] uppercase tracking-wide hover:text-[var(--bn-accent)]">{j.url?.includes('fathom') ? 'call ↗' : 'open ↗'}</a>{/if}
									</span>
								</div>
							{/each}
						{/if}
					</div>
				</SlabCard>
			{:else}
				<section class="bn-rise rounded-[var(--bn-r-panel)] border p-2" style="--rise-i: 2; border-color: var(--bn-border); background: var(--bn-surface)" data-testid="funnel-graph">
					<FunnelGraph
						layout={view.layout}
						nodes={data.radial.nodes}
						segments={data.radial.segments}
						summary={data.summary}
						stages={data.stages}
						stageLabels={data.stageLabels}
						constants={data.constants}
						initialLeadId={leadInSpace}
					/>
				</section>

				<!-- Below the graph, v1's Brand Deals rows: volume, activity, THE insight. -->
				<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
					<SlabCard title="Funnel Volume" sub="{data.summary.clients} active" i={4} class="flex flex-col">
						<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
							<BigStat value={data.volume.revenueUsd} kind="usd" chips={data.volume.chips} caption={data.volume.caption} />
							<MeterStack meters={data.volume.meters} foot={data.volume.foot} empty="no leads yet" />
						</div>
					</SlabCard>

					<SlabCard title="Lead Activity" sub="last 30 days" i={5}>
						<div class="px-6 pt-3">
							<BigStat size={30} value={data.volume.touchesInWindow} caption="touches across every journey: forms, bookings, calls, payments" />
						</div>
						<StepLine series={data.volume.series} hue="var(--send-activity)" unit=" touches" empty="No touches in this window." />
					</SlabCard>

					<InsightCard
						i={6}
						badge="Needs you today"
						value={data.volume.insight.value}
						headline={data.volume.insight.headline}
						body={data.volume.insight.body}
						frac={data.volume.insight.frac}
					/>
				</div>

				<!-- What to act on today; every row pins that lead's dossier above. -->
				{#if data.attention.pushNow.length > 0 || data.attention.saveNow.length > 0}
					<div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
						<SlabCard title="Push Now" sub="hot + moving · close them" i={7}>
							<div class="mt-4 border-t" style="border-color: var(--bn-border)">
								{#if data.attention.pushNow.length === 0}
									<p class="bn-dim px-6 py-6 text-center text-[12.5px]">no hot leads in motion right now</p>
								{:else}
									{#each data.attention.pushNow as j (j.id)}{@render attentionRow(j)}{/each}
								{/if}
							</div>
						</SlabCard>
						<SlabCard title="Save Now" sub="fading toward the archive · highest likelihood first" i={8}>
							<div class="mt-4 border-t" style="border-color: var(--bn-border)">
								{#if data.attention.saveNow.length === 0}
									<p class="bn-dim px-6 py-6 text-center text-[12.5px]">nothing fading · every lead is fresh</p>
								{:else}
									{#each data.attention.saveNow as j (j.id)}{@render attentionRow(j)}{/each}
								{/if}
							</div>
						</SlabCard>
					</div>
				{/if}
			{/if}

			<SlabCard title="Journeys" sub="{data.table.length} of {data.journeys.length}" i={10} class="mt-6">
			{#snippet action()}
				<button data-lens="c" class={chipClass(!view.stage)} onclick={() => go({ stage: null })}>All {data?.journeys.length ?? 0}</button>
				{#each data?.stages ?? [] as s, i (s.id)}
					<button data-lens="c" class="{chipClass(view.stage === s.id)} inline-flex items-center gap-1.5" onclick={() => go({ stage: view.stage === s.id ? null : s.id })}>
						<span class="inline-block h-1.5 w-1.5 rounded-full" style="background: var(--fn-s{i})"></span>
						{s.label} {data?.stageCounts[s.id] ?? 0}
					</button>
				{/each}
			{/snippet}

				{#if view.stage && data.commsUnavailable && data.table.length > 0}
					<p class="bn-dim px-6 pt-3 font-mono text-[11px]">comms feed unavailable · last messages hidden</p>
				{/if}
				<div class="mt-4 border-t" style="border-color: var(--bn-border)" data-testid="funnel-journeys">
					{#if data.table.length === 0}
						<p class="bn-dim px-6 py-8 text-center text-[12.5px]">No leads in this segment.</p>
					{:else}
						{#each data.table as j (j.id)}{@render journeyRow(j)}{/each}
					{/if}
				</div>
			</SlabCard>
		{/if}
	</Slab>
</div>
