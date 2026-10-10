<!-- Summary strip (FounderOS v1 components/SocialStatStrip.tsx): Total reach, Audience
     growth and Total DMs tiles with range chips (click → the series popout),
     and the Needs-reply tile that opens the Instagram DM inbox. -->
<script lang="ts">
	import { ArrowUpRight, MessageSquare, TrendingDown, TrendingUp, X } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import DmInbox from './DmInbox.svelte';
	import { growthOf, inRange, pctTone, RANGE_KEY, RANGE_LABEL, RANGES, formatPct, type Range } from './lib';
	import type { DmThread, Growth, LabelledSeries } from './types';

	let {
		audienceTotal,
		audienceGrowth,
		totalDms,
		dmGrowth,
		platformsCount,
		dmThreads,
		nowMs,
		growthLeader
	}: {
		audienceTotal: number;
		audienceGrowth: Growth;
		totalDms: number;
		dmGrowth: Growth;
		platformsCount: number;
		dmThreads: DmThread[];
		nowMs: number;
		growthLeader: string | null;
	} = $props();

	let audRange = $state<Range>(30);
	let dmRange = $state<Range>(30);
	let popout = $state<'audience' | 'dms' | 'inbox' | null>(null);

	const audPct = $derived(audienceGrowth[RANGE_KEY[String(audRange) as keyof typeof RANGE_KEY]]);
	const dmPct = $derived(dmGrowth[RANGE_KEY[String(dmRange) as keyof typeof RANGE_KEY]]);
	const waiting = $derived(dmThreads.filter((t) => t.unreplied));
	const oldest = $derived.by(() => {
		const ms = waiting.reduce((m, t) => Math.min(m, Date.parse(t.last.ts)), Infinity);
		if (!Number.isFinite(ms) || nowMs <= ms) return null;
		const m = Math.floor((nowMs - ms) / 60_000);
		if (m < 60) return `${Math.max(m, 1)}m`;
		const h = Math.floor(m / 60);
		return h < 24 ? `${h}h` : `${Math.floor(h / 24)}d`;
	});
	const fmtNum = (n: number | null) => (n == null ? '—' : n.toLocaleString('en-US'));

	// the series popout
	let series = $state<LabelledSeries[] | null>(null);
	let seriesError = $state<string | null>(null);
	let popRange = $state<Range>(30);
	let active = $state<Set<string>>(new Set());

	function open(metric: 'audience' | 'dms') {
		popout = metric;
		series = null;
		seriesError = null;
		popRange = 30;
		active = new Set(metric === 'audience' ? ['all'] : ['total']);
		founderosFetch<{ series: LabelledSeries[] }>(`/pages/social/series?metric=${metric}`)
			.then((b) => (series = b.series ?? []))
			.catch((e: unknown) => {
				seriesError = e instanceof Error ? e.message : String(e);
				series = [];
			});
	}

	$effect(() => {
		if (!popout) return;
		const onKey = (e: KeyboardEvent) => e.key === 'Escape' && (popout = null);
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});

	const shown = $derived((series ?? []).filter((s) => active.has(s.key)));
	const chart = $derived.by(() => {
		const W = 820;
		const H = 280;
		const pad = { l: 8, r: 8, t: 14, b: 18 };
		const ranged = shown.map((s) => ({ ...s, points: inRange(s.points, popRange) }));
		const all = ranged.flatMap((s) => s.points);
		if (all.length === 0) return null;
		const dates = [...new Set(all.map((p) => p.date))].sort();
		const xi = new Map(dates.map((d, i) => [d, i]));
		const xMax = Math.max(1, dates.length - 1);
		let lo = Math.min(...all.map((p) => p.value));
		let hi = Math.max(...all.map((p) => p.value));
		if (lo === hi) {
			lo -= 1;
			hi += 1;
		}
		const x = (d: string) => pad.l + (xi.get(d)! / xMax) * (W - pad.l - pad.r);
		const y = (v: number) => pad.t + (1 - (v - lo) / (hi - lo)) * (H - pad.t - pad.b);
		return {
			W,
			H,
			grid: [0.25, 0.5, 0.75].map((g) => pad.t + g * (H - pad.t - pad.b)),
			lines: ranged
				.filter((s) => s.points.length > 0)
				.map((s) => ({ key: s.key, color: s.color, pts: s.points.map((p) => `${x(p.date)},${y(p.value)}`).join(' '), end: [x(s.points.at(-1)!.date), y(s.points.at(-1)!.value)] }))
		};
	});

	function toggle(key: string) {
		const next = new Set(active);
		if (next.has(key)) next.delete(key);
		else next.add(key);
		active = next;
	}
</script>

{#snippet rangeChips(value: Range, set: (r: Range) => void)}
	<div class="flex gap-1">
		{#each RANGES as r (String(r))}
			<button
				type="button"
				class="soc-chip px-1.5 py-0.5 font-mono text-[9.5px] uppercase tracking-[0.08em]"
				data-on={value === r}
				onclick={(e) => {
					e.stopPropagation();
					set(r);
				}}>{RANGE_LABEL[String(r)]}</button
			>
		{/each}
	</div>
{/snippet}

<div class="mb-6 grid grid-cols-2 gap-3 xl:grid-cols-4" data-part="stat-strip">
	<div class="bn-card flex flex-col gap-1 px-3 py-2">
		<span class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.14em]">Total reach</span>
		<div class="flex items-baseline justify-between gap-2">
			<span class="bn-text font-mono text-[20px] font-semibold leading-none tracking-[-0.02em]">{fmtNum(audienceTotal)}</span>
			<span class="bn-dim min-w-0 truncate font-mono text-[9.5px]">{platformsCount} platforms + email</span>
		</div>
	</div>

	<div role="button" tabindex="0" data-lens="r" class="bn-card soc-tile bn-pressable is-row group flex cursor-pointer flex-col gap-1 px-3 py-2 text-left" data-part="tile-audience"
		onclick={() => open('audience')} onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && open('audience')}>
		<div class="flex items-center justify-between">
			<span class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.14em]">Audience growth</span>
			<ArrowUpRight size={12} class="bn-dim opacity-0 transition-opacity group-hover:opacity-100" />
		</div>
		<div class="flex items-baseline justify-between gap-2">
			<span class="font-mono text-[16px] font-semibold leading-none {pctTone(audPct)}">{formatPct(audPct)}</span>
			<span class="bn-dim min-w-0 truncate font-mono text-[9.5px]">{growthLeader ? `${growthLeader} leads · 7d` : `${fmtNum(audienceTotal)} total audience`}</span>
		</div>
		{@render rangeChips(audRange, (r) => (audRange = r))}
	</div>

	<div role="button" tabindex="0" data-lens="r" class="bn-card soc-tile bn-pressable is-row group flex cursor-pointer flex-col gap-1 px-3 py-2 text-left" data-part="tile-dms"
		onclick={() => open('dms')} onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && open('dms')}>
		<div class="flex items-center justify-between">
			<span class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.14em]">Total DMs</span>
			<ArrowUpRight size={12} class="bn-dim opacity-0 transition-opacity group-hover:opacity-100" />
		</div>
		<div class="flex items-baseline justify-between gap-2">
			<span class="bn-text font-mono text-[16px] font-semibold leading-none">{fmtNum(totalDms)}</span>
			<span class="bn-dim flex min-w-0 items-center gap-1 truncate font-mono text-[9.5px]">
				<span class={pctTone(dmPct)}>{formatPct(dmPct)}</span>
				{#if dmPct != null}
					{#if dmPct >= 0}<TrendingUp size={12} class="soc-ok" />{:else}<TrendingDown size={12} class="soc-err" />{/if}
				{/if}
				<span class="truncate">· {RANGE_LABEL[String(dmRange)]} · Instagram · ManyChat</span>
			</span>
		</div>
		{@render rangeChips(dmRange, (r) => (dmRange = r))}
	</div>

	<div role="button" tabindex="0" data-lens="r" class="bn-card soc-tile bn-pressable is-row group flex cursor-pointer flex-col gap-1 px-3 py-2 text-left" data-part="tile-inbox"
		onclick={() => (popout = 'inbox')} onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (popout = 'inbox')}>
		<div class="flex items-center justify-between">
			<span class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.14em]">Needs reply</span>
			<ArrowUpRight size={12} class="bn-dim opacity-0 transition-opacity group-hover:opacity-100" />
		</div>
		<div class="flex items-baseline justify-between gap-2">
			<span class="font-mono text-[16px] font-semibold leading-none {waiting.length > 0 ? 'soc-warn' : 'bn-text'}">
				{waiting.length > 0 ? `${waiting.length} to reply` : `${dmThreads.length} threads`}
			</span>
			<span class="bn-dim min-w-0 truncate font-mono text-[9.5px]">{waiting.length > 0 && oldest ? `oldest ${oldest}` : `${dmThreads.length} conversations`}</span>
		</div>
		<span class="bn-dim flex items-center gap-1.5 font-mono text-[9.5px]"><MessageSquare size={12} /> open inbox · reply here</span>
	</div>
</div>

{#if popout}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="soc-modal-back fixed inset-0 z-[70] flex items-center justify-center p-3 sm:p-6" onclick={() => (popout = null)}>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div class="soc-modal max-h-[92vh] w-full overflow-y-auto {popout === 'inbox' ? 'max-w-6xl' : 'max-w-3xl'}" onclick={(e) => e.stopPropagation()} data-part="popout">
			<div class="flex items-center justify-between border-b px-5 py-3.5" style="border-color: var(--bn-border)">
				<div class="flex items-center gap-3">
					{#if popout === 'inbox'}<MessageSquare size={16} class="bn-accent" />{/if}
					<h2 class="bn-text text-sm font-bold">{popout === 'audience' ? 'Audience growth' : popout === 'dms' ? 'Total DMs' : 'Instagram DMs'}</h2>
					{#if popout !== 'inbox'}{@render rangeChips(popRange, (r) => (popRange = r))}{/if}
				</div>
				<button type="button" class="soc-chip grid h-7 w-7 place-items-center" aria-label="Close" onclick={() => (popout = null)}><X size={14} /></button>
			</div>
			<div class="p-5">
				{#if popout === 'inbox'}
					<DmInbox threads={dmThreads} {nowMs} />
				{:else if series === null}
					<div class="bn-dim py-20 text-center font-mono text-xs">loading history…</div>
				{:else}
					{#if seriesError}<p class="soc-err mb-3 font-mono text-[11px]">history unavailable: {seriesError}</p>{/if}
					{#if popout === 'audience'}
						<div class="mb-4 flex flex-wrap gap-1.5">
							{#each series as s (s.key)}
								<button type="button" class="soc-chip flex items-center gap-1.5 px-2 py-1 font-mono text-[10px]" data-on={active.has(s.key)} onclick={() => toggle(s.key)}>
									<span class="h-2 w-2" style="background: {active.has(s.key) ? s.color : 'var(--bn-text-3)'}"></span>{s.label}
								</button>
							{/each}
						</div>
					{/if}
					{#if chart}
						<svg viewBox="0 0 {chart.W} {chart.H}" preserveAspectRatio="none" class="h-64 w-full">
							{#each chart.grid as gy (gy)}<line x1="8" x2={chart.W - 8} y1={gy} y2={gy} stroke="var(--bn-border)" />{/each}
							{#each chart.lines as l (l.key)}
								<polyline points={l.pts} fill="none" stroke={l.color} stroke-width="2" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
								<circle cx={l.end[0]} cy={l.end[1]} r="3" fill={l.color} />
							{/each}
						</svg>
					{:else}
						<div class="bn-dim py-16 text-center font-mono text-xs">no history in range</div>
					{/if}
					<div class="mt-4 flex flex-col gap-1.5">
						{#each shown as s (s.key)}
							{@const pts = inRange(s.points, popRange)}
							{@const g = growthOf(pts)}
							<div class="flex items-center gap-2.5 text-[12px]">
								<span class="h-2.5 w-2.5 shrink-0" style="background: {s.color}"></span>
								<span class="bn-text min-w-0 flex-1 truncate">{s.label}</span>
								<span class="bn-muted font-mono">{fmtNum(pts.at(-1)?.value ?? null)}</span>
								<span class="w-20 text-right font-mono font-semibold {pctTone(g)}">{formatPct(g)}</span>
							</div>
						{/each}
						{#if shown.length === 0}<div class="bn-dim font-mono text-[11px]">select a series to plot</div>{/if}
					</div>
					<p class="bn-dim mt-4 font-mono text-[10px]">
						{RANGE_LABEL[String(popRange)]} window · {popout === 'dms' ? 'DM totals are seeded dummy until a source is wired' : 'email tracks the real Beehiiv subscriber count for the connected newsletter'}
					</p>
				{/if}
			</div>
		</div>
	</div>
{/if}
