<!-- Combined audience line + per-platform posting bars on one range toggle
     and one hover cursor (components/AudienceConsistency.tsx), with the
     audience-share pie riding the right of the card. Posting history that
     Zernio did not answer for reads unknown, not "no posts". -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Maximize2, X } from '$lib/founderos/icons';
	import {
		bucketPostingByDay,
		colorFor,
		dateAxis,
		forwardFill,
		platformLabel,
		platformsPresent,
		PLATFORM_ORDER
	} from './lib';
	import type { PostDay, SeriesValue } from './types';

	type ChartRange = 30 | 90 | 365;
	const RANGES: ChartRange[] = [30, 90, 365];
	const RANGE_LABEL: Record<number, string> = { 30: '30d', 90: '90d', 365: '1y' };

	let {
		audience,
		postDays,
		today,
		aside
	}: { audience: SeriesValue[]; postDays: PostDay[] | null; today: string; aside?: Snippet } = $props();

	let range = $state<ChartRange>(90);
	let expanded = $state(false);
	let hover = $state<number | null>(null);

	const known = $derived(postDays !== null);
	const model = $derived.by(() => {
		const posts = postDays ?? [];
		const dates = [...audience.map((a) => a.date), ...posts.map((p) => p.date)].sort();
		const earliest = dates[0] ?? today;
		const d = new Date(`${today}T00:00:00Z`);
		d.setUTCDate(d.getUTCDate() - (range - 1));
		const want = d.toISOString().slice(0, 10);
		const start = want < earliest ? earliest : want;
		const axis = dateAxis(start, today);
		const audienceVals = forwardFill(audience, axis);
		const inRange = posts.filter((p) => p.date >= start && p.date <= today);
		const days = bucketPostingByDay(inRange, axis);
		const platforms = platformsPresent(inRange, PLATFORM_ORDER);
		const vals = audienceVals.filter((v): v is number => v != null);
		const net = vals.length > 1 ? vals[vals.length - 1] - vals[0] : 0;
		return { axis, audienceVals, days, platforms, net };
	});

	const W = 760;
	const HA = 200;
	const HP = 120;
	const PAD = 12;

	const geo = $derived.by(() => {
		const { axis, audienceVals, days } = model;
		const n = axis.length;
		const xAt = (i: number) => (n > 1 ? (i / (n - 1)) * W : W / 2);
		const vals = audienceVals.filter((v): v is number => v != null);
		const aMax = vals.length ? Math.max(...vals) : 1;
		const aMin = vals.length ? Math.min(...vals) : 0;
		const yA = (v: number) => HA - PAD - ((v - aMin) / Math.max(1, aMax - aMin)) * (HA - 2 * PAD);
		const pts = audienceVals.map((v, i) => (v == null ? null : ([xAt(i), yA(v)] as [number, number]))).filter((p): p is [number, number] => p != null);
		const line = pts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ');
		const dayMax = Math.max(1, ...days.map((d) => d.total));
		const slot = n > 0 ? W / n : W;
		const barW = Math.max(1.5, Math.min(16, slot * 0.7));
		const bars = days.map((d, i) => {
			let y = HP;
			const stack = model.platforms
				.filter((p) => (d.counts[p] ?? 0) > 0)
				.map((p) => {
					const h = ((d.counts[p] ?? 0) / dayMax) * (HP - 8);
					y -= h;
					return { p, y, h: Math.max(1, h - 0.6) };
				});
			return { date: d.date, x: xAt(i) - barW / 2, total: d.total, stack };
		});
		return { n, xAt, yA, pts, line, bars, barW };
	});

	const totals = $derived.by(() => {
		const t: Record<string, number> = {};
		for (const d of model.days) for (const p of model.platforms) t[p] = (t[p] ?? 0) + (d.counts[p] ?? 0);
		return t;
	});
	const postTotal = $derived(model.days.reduce((s, d) => s + d.total, 0));

	function fmtDay(d: string): string {
		return new Date(`${d}T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
	}

	function onMove(e: MouseEvent) {
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const n = geo.n;
		if (n === 0 || rect.width === 0) return;
		hover = Math.min(n - 1, Math.max(0, Math.round(((e.clientX - rect.left) / rect.width) * (n - 1))));
	}

	$effect(() => {
		if (!expanded) return;
		const onKey = (e: KeyboardEvent) => e.key === 'Escape' && (expanded = false);
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});
</script>

{#snippet chips()}
	<div class="flex gap-1">
		{#each RANGES as r (r)}
			<button type="button" class="soc-chip px-2 py-0.5 font-mono text-[10px] uppercase tracking-[0.08em]" data-on={range === r} onclick={() => (range = r)}>
				{RANGE_LABEL[r]}
			</button>
		{/each}
	</div>
{/snippet}

{#snippet net()}
	<span class="font-mono text-[11px] {model.net >= 0 ? 'soc-ok' : 'soc-err'}">
		{model.net >= 0 ? '▲ +' : '▼ '}{model.net.toLocaleString('en-US')} net
	</span>
{/snippet}

{#snippet pair(audH: string, postH: string)}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="relative" onmousemove={onMove} onmouseleave={() => (hover = null)}>
		<div class={audH}>
			<svg viewBox="0 0 {W} {HA}" preserveAspectRatio="none" class="block h-full w-full">
				{#each [0.33, 0.66] as g (g)}
					<line x1="0" x2={W} y1={HA - PAD - g * (HA - 2 * PAD)} y2={HA - PAD - g * (HA - 2 * PAD)} stroke="var(--bn-border)" stroke-dasharray="2 4" />
				{/each}
				{#if geo.pts.length > 1}
					<path d="{geo.line} L{geo.pts[geo.pts.length - 1][0]},{HA} L{geo.pts[0][0]},{HA} Z" fill="var(--bn-accent)" fill-opacity="0.1" />
					<path d={geo.line} fill="none" stroke="var(--bn-accent)" stroke-width="2" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
				{/if}
				{#if hover != null}
					<line x1={geo.xAt(hover)} x2={geo.xAt(hover)} y1="0" y2={HA} stroke="var(--bn-accent)" stroke-opacity="0.5" />
				{/if}
			</svg>
		</div>
		<div class="bn-dim mt-1.5 flex justify-between font-mono text-[9.5px]">
			<span>{model.axis[0] ? fmtDay(model.axis[0]) : ''}</span><span>today</span>
		</div>
		<div class="my-3.5 h-px" style="background: var(--bn-border)"></div>
		<div class="mb-2 flex items-center justify-between gap-3">
			<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.2em]">Posting consistency</span>
			<span class="bn-muted font-mono text-[11px]" data-part="posting-total">
				{known ? `${postTotal} posts · ${model.platforms.length} platforms` : 'posting history unknown · Zernio not answering'}
			</span>
		</div>
		<div class={postH}>
			<svg viewBox="0 0 {W} {HP}" preserveAspectRatio="none" class="block h-full w-full">
				{#each [0.25, 0.5, 0.75] as g (g)}
					<line x1="0" x2={W} y1={g * HP} y2={g * HP} stroke="var(--bn-border)" stroke-dasharray="2 4" />
				{/each}
				{#if known}
					{#each geo.bars as b, i (b.date)}
						{#if b.total === 0}
							<rect x={b.x} y={HP - 2} width={geo.barW} height="2" fill="var(--bn-accent)" opacity="0.14" />
						{:else}
							<g opacity={hover == null || hover === i ? 1 : 0.4}>
								{#each b.stack as s (s.p)}
									<rect x={b.x} y={s.y} width={geo.barW} height={s.h} fill={colorFor(s.p)} />
								{/each}
							</g>
						{/if}
					{/each}
				{/if}
				{#if hover != null}
					<line x1={geo.xAt(hover)} x2={geo.xAt(hover)} y1="0" y2={HP} stroke="var(--bn-accent)" stroke-opacity="0.5" />
				{/if}
			</svg>
		</div>
		{#if hover != null && model.axis[hover]}
			{@const day = model.days[hover]}
			{@const entries = model.platforms.filter((p) => (day?.counts[p] ?? 0) > 0)}
			<div class="pointer-events-none absolute top-0 z-10 -translate-x-1/2" style="left: {Math.min(88, Math.max(12, geo.n > 1 ? (hover / (geo.n - 1)) * 100 : 50))}%">
				<div class="soc-modal min-w-[150px] px-2.5 py-2 font-mono">
					<div class="bn-dim mb-1 flex items-center justify-between gap-3 text-[10px]">
						<span>{fmtDay(model.axis[hover])}</span>
						<span class="bn-muted">{model.audienceVals[hover] != null ? model.audienceVals[hover]!.toLocaleString('en-US') : '—'} aud</span>
					</div>
					{#if !known}
						<div class="bn-dim text-[10px]">posts unknown</div>
					{:else if entries.length === 0}
						<div class="bn-dim text-[10px]">no posts</div>
					{:else}
						{#each entries as p (p)}
							<div class="bn-text flex items-center gap-1.5 text-[10.5px]">
								<span class="h-2 w-2 shrink-0" style="background: {colorFor(p)}"></span>
								<span class="flex-1">{platformLabel(p)}</span>
								{#if (day.counts[p] ?? 0) > 1}<span class="bn-dim">×{day.counts[p]}</span>{/if}
							</div>
						{/each}
					{/if}
				</div>
			</div>
		{/if}
	</div>
	{#if model.platforms.length > 0}
		<div class="mt-3 flex flex-wrap gap-x-4 gap-y-1.5">
			{#each model.platforms as p (p)}
				<div class="flex items-center gap-1.5 font-mono text-[10.5px]">
					<span class="h-2 w-2" style="background: {colorFor(p)}"></span>
					<span class="bn-muted">{platformLabel(p)}</span>
					<span class="bn-dim">{totals[p] ?? 0}</span>
				</div>
			{/each}
		</div>
	{/if}
{/snippet}

<div class="bn-card bn-rise px-5 py-[18px]" style="--rise-i: 7" data-part="audience-chart">
	<div class={aside ? 'grid gap-6 lg:grid-cols-[1.55fr_1fr]' : ''}>
		<div class="flex min-w-0 flex-col">
			<div class="mb-2 flex items-center justify-between gap-3">
				<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.2em]">Audience · {RANGE_LABEL[range]}</span>
				<div class="flex items-center gap-2.5">
					{@render net()}
					{@render chips()}
					<button type="button" class="soc-chip flex h-[22px] w-[22px] items-center justify-center" aria-label="Expand to fullscreen" title="Fullscreen" onclick={() => (expanded = true)}>
						<Maximize2 size={12} />
					</button>
				</div>
			</div>
			{@render pair('h-[140px]', 'h-[84px]')}
		</div>
		{#if aside}
			<div class="flex min-w-0 flex-col justify-center border-t pt-4 lg:border-l lg:border-t-0 lg:pl-6 lg:pt-0" style="border-color: var(--bn-border)">
				{@render aside()}
			</div>
		{/if}
	</div>
</div>

{#if expanded}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="soc-modal-back fixed inset-0 z-[120] flex items-center justify-center p-4 sm:p-6" onclick={() => (expanded = false)}>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div class="soc-modal flex max-h-[94vh] w-[min(1240px,96vw)] flex-col overflow-hidden" onclick={(e) => e.stopPropagation()}>
			<div class="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-3.5" style="border-color: var(--bn-border)">
				<div>
					<div class="bn-dim font-mono text-[10px] uppercase tracking-[0.2em]">social</div>
					<div class="bn-text mt-0.5 text-[15px] font-semibold">Combined audience &amp; posting consistency</div>
				</div>
				<div class="flex items-center gap-3">
					{@render chips()}
					<button type="button" class="soc-chip flex h-7 w-7 items-center justify-center" aria-label="Close" onclick={() => (expanded = false)}><X size={16} /></button>
				</div>
			</div>
			<div class="flex-1 overflow-y-auto px-5 py-4">
				<div class="mb-2 flex items-center justify-between">
					<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.2em]">Audience · {RANGE_LABEL[range]}</span>
					{@render net()}
				</div>
				{@render pair('h-[clamp(240px,40vh,420px)]', 'h-[clamp(160px,26vh,300px)]')}
			</div>
		</div>
	</div>
{/if}
