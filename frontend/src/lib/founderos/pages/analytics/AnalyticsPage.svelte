<!-- /os/analytics (spec 6.7): FounderOS v1 app/analytics/page.tsx. The run log
     as the hero chart, reach as the Volume card, runs by agent, the weekday
     rhythm, the credentials insight, live operating-metric tiles and the
     audience by platform, in v1's order. Every number is a real read
     (GET /pages/analytics) or reads unknown. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { Instagram, Linkedin, Music2, Youtube } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, PILL_ACCENT, Slab, SlabCard, SlabTitle, Spark } from '$lib/founderos/kit';
	import '../funnel/tokens.css';
	import RunVolumeCard from './RunVolumeCard.svelte';
	import { formatFollowers, formatPct, tileValue } from './format';
	import type { AnalyticsBody } from './types';

	let data = $state<AnalyticsBody | null>(null);
	let failure = $state<string | null>(null);

	onMount(() => {
		founderosFetch<AnalyticsBody>('/pages/analytics')
			.then((b) => (data = b))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});

	const ICONS: Record<string, typeof Instagram> = { instagram: Instagram, tiktok: Music2, youtube: Youtube, linkedin: Linkedin };
	const busiest = $derived(data ? data.volume.rhythm.reduce((b, c) => (c.count > b.count ? c : b), data.volume.rhythm[0]) : null);
	const vol = $derived(data?.volume);
	const growthStyle = (d7: number | null) =>
		d7 === null
			? 'background: color-mix(in oklab, var(--bn-text) 8%, transparent); color: var(--bn-text-2)'
			: d7 >= 0
				? 'background: color-mix(in oklab, var(--bn-ok) 16%, transparent); color: var(--bn-ok)'
				: 'background: color-mix(in oklab, var(--bn-err) 16%, transparent); color: var(--bn-err)';
	const errorList = $derived(data ? Object.entries(data.errors) : []);
</script>

<!-- prod body is `antialiased` (app/globals.css); /os has no global equivalent yet, so the page root carries it. -->
<div class="bn-fn antialiased" data-testid="analytics-root">
	<Slab>
		<SlabTitle
			eyebrow="operating metrics · connectors · agent runs"
			title="Analytics"
			meta={data && vol
				? `${vol.headline} reach · ${data.live.length} live · ${data.pending.length} pending · ${data.runs30d.toLocaleString('en-US')} runs in 30 days`
				: failure
					? 'analytics unavailable'
					: 'reading connectors…'}
		>
			{#snippet right()}
				{#if data}<span class="bn-muted rounded-full border px-4 py-2 text-[13px]" style="border-color: var(--bn-border)">{data.live.length} live · {data.pending.length} pending</span>{/if}
				<a href="/os/social" data-lens="c" class={PILL}>Open Social</a>
				<a href="/os/integrations" class={PILL_ACCENT}>Wire connectors</a>
			{/snippet}
		</SlabTitle>

		{#if failure && !data}
			<SlabCard title="Analytics unavailable" i={1}>
				<p role="alert" class="px-6 pb-6 pt-3 text-[13px]" style="color: var(--bn-err)">Could not read analytics: {failure}</p>
			</SlabCard>
		{:else if !data || !vol}
			<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1" data-testid="analytics-loading">
				<div class="h-[360px] animate-pulse rounded-[12px] border" style="border-color: var(--bn-border); background: var(--bn-surface)"></div>
				<div class="h-[360px] animate-pulse rounded-[12px] border" style="border-color: var(--bn-border); background: var(--bn-surface)"></div>
			</div>
		{:else}
			{#if errorList.length > 0}
				<p role="alert" class="mb-3 font-mono text-[11px]" style="color: var(--bn-warn)">
					could not read: {errorList.map(([k, v]) => `${k} (${v})`).join(' · ')}
				</p>
			{/if}

			<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
				<RunVolumeCard data={data.runVolume} known={data.runsKnown} i={1} />
				<SlabCard title="Reach Volume" i={2} class="flex flex-col">
					<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
						{#if data.audience.known}
							<BigStat value={vol.reach} chips={vol.chips} caption={vol.caption} />
						{:else}
							<BigStat value={null} caption="audience snapshots could not be read" />
						{/if}
						<MeterStack meters={vol.meters} foot={vol.foot} empty="no audience snapshots yet" />
					</div>
				</SlabCard>
			</div>

			<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
				<SlabCard title="Runs by Agent" sub="{vol.runs.total.toLocaleString('en-US')} runs" i={3} class="flex flex-col">
					<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
						<BigStat
							size={30}
							display={vol.runs.okPct === null ? 'none yet' : `${vol.runs.okPct}%`}
							chips={vol.runs.total ? [{ tone: 'ok', text: `${vol.runs.ok} ok` }, ...(vol.runs.failed ? [{ tone: 'err' as const, text: `${vol.runs.failed} failed` }] : [])] : []}
							caption="of runs succeeded in the last 30 days"
						/>
						<MeterStack meters={vol.runs.meters} foot="{vol.runs.agents} agents in the log" empty="no agent runs recorded yet" />
					</div>
				</SlabCard>
				<SlabCard title="Run Rhythm" sub="last 30 days" i={4}>
					<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
						<div class="shrink-0">
							<BigStat size={30} value={vol.rhythmTotal} />
							<div class="bn-dim mt-1 whitespace-nowrap text-[13px]">runs by weekday</div>
							{#if vol.rhythmTotal > 0 && busiest}
								<div class="bn-muted mt-4 w-fit whitespace-nowrap rounded-full border px-3 py-1 text-[12px]" style="border-color: var(--bn-border)">Busiest: <span class="font-semibold tabular-nums">{busiest.label}</span></div>
							{/if}
						</div>
						<DotMatrix cols={vol.rhythm} hue="var(--send-activity)" />
					</div>
				</SlabCard>
				<InsightCard i={5} badge="Awaiting credentials" value={vol.insight.value} headline={vol.insight.headline} body={vol.insight.body} frac={vol.insight.frac} />
			</div>

			<SlabCard title="Operating Metrics" sub="{data.live.length} live · {data.pending.length} pending" i={6} class="mt-6">
				{#snippet action()}<a href="/os/integrations" data-lens="c" class={PILL}>wire connectors → flip to live</a>{/snippet}
				<div class="px-6 pb-6 pt-4" data-testid="operating-metrics">
					{#if data.live.length > 0}
						<div class="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4 min-[2200px]:grid-cols-6">
							{#each data.live as tile (tile.id)}
								{@const v = tileValue(tile.value, tile.unit)}
								<div data-lens="r" class="bn-pressable is-row flex min-w-0 flex-col gap-3 rounded-[12px] border px-5 py-4" style="border-color: var(--bn-border)" data-testid="metric-tile">
									<div class="flex items-center justify-between gap-2">
										<span class="bn-muted truncate text-[13px]">{tile.label}</span>
										{#if tile.delta !== 0}
											<Chip tone={tile.delta > 0 ? 'ok' : 'err'}>{tile.delta > 0 ? '+' : ''}{tile.delta}{tile.deltaPct ? '%' : ''}</Chip>
										{/if}
									</div>
									<div class="bn-text flex items-baseline gap-2 text-[30px] font-semibold leading-none tracking-[-0.03em] tabular-nums">
										{v.main}{#if v.small}<small class="bn-dim text-[13px] font-normal tracking-normal">{v.small}</small>{/if}
									</div>
									<div class="flex items-end justify-between gap-2">
										<span class="shrink-0" title={tile.sparkReal ? 'captured history' : 'placeholder shape until two days of history exist'} style={tile.sparkReal ? '' : 'opacity: 0.45'}><Spark data={tile.spark} w={96} h={26} /></span>
										<span class="bn-dim min-w-0 truncate font-mono text-[10.5px]" title={tile.source}>{tile.source}</span>
									</div>
								</div>
							{/each}
						</div>
					{:else}
						<p class="bn-dim py-4 text-[12.5px]">No connector has handed back a live number yet.</p>
					{/if}
					<div class="mt-5 flex flex-wrap items-center gap-2 border-t pt-4" style="border-color: var(--bn-border)">
						<span class="bn-dim mr-1 text-[13px]">Awaiting credentials</span>
						{#if data.pending.length === 0}
							<Chip tone="ok">all connectors live</Chip>
						{:else}
							{#each data.pending as m (m.id)}
								<span title={m.source}><Chip tone="warn">{m.label} · {m.source}</Chip></span>
							{/each}
						{/if}
					</div>
				</div>
			</SlabCard>

			<SlabCard title="Audience" sub="{formatFollowers(data.audience.known ? data.audience.totalFollowers : null)} followers" i={9} class="mt-6">
				{#snippet action()}<a href="/os/social" data-lens="c" class={PILL}>Open Social</a>{/snippet}
				<div class="mt-4 border-t" style="border-color: var(--bn-border)" data-testid="audience">
					{#each data.audience.platforms as p (p.platform)}
						{@const Icon = ICONS[p.platform]}
						{@const maxBar = p.bars ? Math.max(...p.bars, 1) : 1}
						<a href="/os/social/{p.platform}" data-lens="r" class="bn-pressable is-row fn-row group grid w-full grid-cols-[minmax(180px,1fr)_auto_auto_auto] items-center gap-5 border-b px-6 py-3.5 last:border-0 max-[800px]:grid-cols-[1fr_auto]" style="border-color: var(--bn-border)">
							<span class="flex min-w-0 items-center gap-3">
								<span data-part="platform-icon" class="fn-platform-icon grid h-9 w-9 shrink-0 place-items-center rounded-full border" style="border-color: var(--bn-border)">
									{#if Icon}<Icon class="h-4 w-4" />{:else}<svg viewBox="0 0 24 24" fill="currentColor" class="h-4 w-4" aria-hidden="true"><path d="M14.234 10.162 22.977 0h-2.072l-7.591 8.824L7.251 0H.258l9.168 13.343L.258 24H2.33l8.016-9.318L16.749 24h6.993zm-2.837 3.299-.929-1.329L3.076 1.56h3.182l5.965 8.532.929 1.329 7.754 11.09h-3.182z" /></svg>{/if}
								</span>
								<span class="min-w-0">
									<span class="bn-text block truncate text-[13.5px] font-medium">{p.label}</span>
									<span class="bn-dim block truncate font-mono text-[11px]">{p.handle}</span>
								</span>
							</span>
							<span class="max-[800px]:hidden">
								{#if p.bars}
									<div class="flex h-[30px] items-end gap-[3px]" title="last {p.bars.length} follower snapshots">
										{#each p.bars as b, k (k)}<div class="w-1.5 rounded-[1px]" style="background: var(--bn-accent); height: {Math.max(8, (b / maxBar) * 100)}%; opacity: {0.35 + (k / p.bars.length) * 0.65}"></div>{/each}
									</div>
								{:else}
									<span class="bn-dim font-mono text-[10.5px]" title="fewer than two snapshots">no history yet</span>
								{/if}
							</span>
							<span class="bn-text text-right text-[20px] font-semibold tabular-nums tracking-[-0.02em]">{formatFollowers(p.followers)}</span>
							<span class="flex items-center gap-2 max-[800px]:hidden">
								<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]" style="background: var(--bn-accent-soft); color: var(--bn-accent)">{p.share === null ? '—' : `${p.share.toFixed(0)}%`} of reach</span>
								<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tabular-nums tracking-[0.08em]" style={growthStyle(p.d7)}>7d {formatPct(p.d7)}</span>
							</span>
						</a>
					{/each}
					{#if data.audience.platforms.length === 0}
						<div class="bn-dim px-6 py-8 text-center text-[12.5px]">{data.audience.known ? 'No platforms synced yet.' : 'Audience snapshots could not be read.'}</div>
					{/if}
				</div>
			</SlabCard>
		{/if}
	</Slab>
</div>
