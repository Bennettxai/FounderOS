<!-- VSL performance (FounderOS v1 components/VslAnalytics.tsx): plays in the
     saved period over one meter per video, video rows, and the selected
     video's retention curve beside its rate meters. `compact` is the form
     that sits inside the channel funnels card. The IG association is a
     bridge-data write (POST /pages/analytics/vsl/association). -->
<script lang="ts">
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, MeterStack, PILL, SlabCard } from '$lib/founderos/kit';
	import type { VslSnapshot } from './types';
	import { vslForPeriod, vslPeriods, vslRateMeters, vslVolume } from './volume';

	let {
		snapshots,
		igVideo = null,
		error = null,
		compact = false,
		i = 7
	}: { snapshots: VslSnapshot[]; igVideo?: string | null; error?: string | null; compact?: boolean; i?: number } = $props();

	const uid = $props.id();
	const periods = $derived(vslPeriods(snapshots));
	let period = $state('');
	let selected = $state('');
	let mapped = $state<string | null>(null);
	let saving = $state(false);
	let status = $state('');
	let seeded = false;
	$effect(() => {
		if (seeded) return;
		seeded = true;
		period = periods.at(-1) ?? '';
		selected = igVideo ?? 'FIGSOIxs3cQNosiW';
		mapped = igVideo;
	});

	const rows = $derived(vslForPeriod(snapshots, period).sort((a, b) => b.plays - a.plays));
	const video = $derived(rows.find((r) => r.videoId === selected) ?? rows[0]);
	const options = $derived([...new Map(snapshots.map((r) => [r.videoId, r])).values()]);
	const vol = $derived(vslVolume(rows));
	const count = (n: number) => n.toLocaleString('en-US');
	const percent = (n: number | null) => (n === null ? 'N/A' : `${(n * 100).toFixed(1)}%`);
	const CONTROL = 'fn-control rounded-full border px-3.5 py-2 text-[12.5px]';
	// prod's intro names the count in words ("Five videos, ..."); past twelve it is digits.
	const WORDS = ['No', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten', 'Eleven', 'Twelve'];
	const videosIn = (k: number) => `${WORDS[k] ?? k} video${k === 1 ? '' : 's'}`;

	async function associate() {
		if (!video) return;
		saving = true;
		status = '';
		try {
			await founderosFetch('/pages/analytics/vsl/association', { method: 'POST', json: { videoId: video.videoId } });
			mapped = video.videoId;
			status = 'Saved as the Instagram funnel video.';
		} catch {
			status = 'Could not save the video association. Try again.';
		} finally {
			saving = false;
		}
	}

	const curve = $derived.by(() => {
		if (!video) return null;
		const points = [...video.audience].sort((a, b) => a.second - b.second);
		if (!points.length) return null;
		const max = Math.max(1, ...points.map((p) => p.viewers));
		const duration = video.duration.split(':').reduce((n, part) => n * 60 + Number(part), 0);
		return { max, xy: points.map((p) => `${36 + (p.second / Math.max(1, duration)) * 540},${180 - (p.viewers / max) * 145}`).join(' ') };
	});
	const trendMax = $derived(video ? Math.max(1, ...video.trend.map((p) => p.plays)) : 1);
</script>

{#snippet periodPicker()}
	<label class="bn-dim flex items-center gap-2 text-[12px]" for="{uid}-period">
		Saved video period
		<select id="{uid}-period" class={CONTROL} bind:value={period}>
			{#each periods as p (p)}<option value={p}>{p.replace('/', ' to ')}</option>{/each}
		</select>
	</label>
{/snippet}

{#snippet detail()}
	{#if video}
		<div class="grid gap-8 lg:grid-cols-[1.3fr_1fr]">
			<div class="min-w-0">
				<h3 class="bn-text text-[15px] font-semibold">{video.title} <span class="bn-dim font-mono text-[12px] font-normal">{video.duration}</span></h3>
				<p class="bn-dim mt-1 text-[12px]">Audience remaining at each sampled video position</p>
				{#if curve}
					<svg viewBox="0 0 600 215" class="mt-3 block w-full" role="img" aria-label="{video.title}: audience retention over {video.duration}">
						{#each [0, 0.5, 1] as n (n)}
							<g><line x1="36" x2="576" y1={180 - n * 145} y2={180 - n * 145} stroke="var(--bn-border)" /><text x="30" y={184 - n * 145} text-anchor="end" font-size="10" fill="var(--bn-text-3)">{Math.round(curve.max * n)}</text></g>
						{/each}
						{#key `${video.videoId}-${period}`}<polyline points={curve.xy} fill="none" stroke="var(--bn-accent)" stroke-width="2.5" stroke-linejoin="round" pathLength="1" class="bn-draw" style="animation-duration: 1.6s; animation-delay: 0.4s" />{/key}
						<text x="36" y="204" font-size="10" fill="var(--bn-text-3)">0:00</text><text x="576" y="204" text-anchor="end" font-size="10" fill="var(--bn-text-3)">{video.duration}</text>
					</svg>
				{:else}
					<p class="bn-dim py-12 text-[12px]">No retention curve captured for this period. Select the historical range to see it.</p>
				{/if}
				{#if video.trend.length > 0}
					<div class="mt-3">
						<p class="bn-dim font-mono text-[10.5px] uppercase tracking-[0.1em]">Plays by {video.trendInterval} / saved trend</p>
						<div class="mt-2 flex h-14 items-end gap-1" role="img" aria-label="{video.title}: {video.trendInterval}ly play counts">
							{#each video.trend as point (point.date)}
								<div title="{point.date}: {point.plays} plays" class="min-w-[2px] flex-1 rounded-t-[2px] opacity-60" style="background: var(--bn-accent); height: {Math.max(2, (point.plays / trendMax) * 100)}%"></div>
							{/each}
						</div>
						<p class="bn-dim mt-1 flex justify-between font-mono text-[10.5px]"><span>{video.trend[0].date}</span><span>{video.trend.at(-1)?.date}</span></p>
					</div>
				{/if}
			</div>
			<div class="flex min-w-0 flex-col">
				<BigStat value={video.plays} size={30} caption="plays of this video" />
				{#key `${video.videoId}-${period}`}<MeterStack meters={vslRateMeters(video)} baseDelay={300} foot="saved {video.capturedAt.slice(0, 10)} · {video.dateFrom} to {video.dateTo}" />{/key}
				<p class="bn-dim mt-3 text-[11.5px] leading-relaxed">Refreshing Trakyo does not refresh this video snapshot. No viewer-level attribution join is available.</p>
			</div>
		</div>
	{/if}
{/snippet}

{#snippet caveat()}
	<p class="bn-dim mt-5 text-[11.5px] leading-relaxed">Vidalytics recorded no conversions or revenue in this import; this does not mean there were no sales. Use Trakyo for attributed revenue. Account timezone was not exposed by the video export.</p>
{/snippet}

{#if compact}
	<section aria-label="Instagram VSL performance" class="mt-7 border-t pt-6" style="border-color: var(--bn-border)">
		<header class="mb-5 flex flex-wrap items-start justify-between gap-4">
			<div>
				<p class="bn-dim font-mono text-[11px] uppercase tracking-[0.32em]">// vidalytics · saved import</p>
				<h3 class="bn-text mt-2 text-[19px] font-semibold tracking-[-0.01em]">Inside the Instagram VSL</h3>
				<p class="bn-muted mt-1.5 text-[12.5px]">Video-wide results alongside Instagram attribution. These are not Instagram-only viewers.</p>
			</div>
			{@render periodPicker()}
		</header>
		{#if error}
			<p class="text-[13px]" style="color: var(--bn-warn)">Saved video snapshots unavailable: {error}</p>
		{:else if !video}
			<p class="bn-dim text-[13px]">No video snapshots imported yet.</p>
		{:else}
			<div class="mb-6 flex flex-wrap items-end gap-3">
				<label class="bn-dim flex min-w-0 flex-col gap-1.5 text-[12px]" for="{uid}-video">
					Instagram funnel video
					<select id="{uid}-video" class="{CONTROL} max-w-full" value={video.videoId} onchange={(e) => { selected = e.currentTarget.value; status = ''; }}>
						{#each options as v (v.videoId)}<option value={v.videoId}>{v.title} ({v.duration})</option>{/each}
					</select>
				</label>
				<button data-lens="c" class="{PILL} disabled:opacity-50" disabled={saving || mapped === video.videoId} onclick={associate}>
					{saving ? 'Saving...' : mapped === video.videoId ? 'Assigned to IG' : 'Use this video for IG'}
				</button>
				{#if !mapped}<p class="bn-dim w-full text-[12px]">Preview only. Choose and assign the video used by your Instagram funnel.</p>{/if}
				{#if status}<p role="status" class="bn-text w-full text-[12px]">{status}</p>{/if}
			</div>
			{@render detail()}
			{@render caveat()}
		{/if}
	</section>
{:else}
	<section aria-label="VSL performance" class="mt-6">
		<SlabCard {i} title="VSL Performance" sub="Vidalytics · saved import">
			{#snippet action()}{@render periodPicker()}{/snippet}
			<div class="px-6 pb-6 pt-4">
				{#if error}
					<p class="py-6 text-[13px]" style="color: var(--bn-warn)">Saved video snapshots unavailable: {error}</p>
				{:else if !video}
					<p class="bn-dim py-6 text-[13px]">No video snapshots imported yet.</p>
				{:else}
					<div class="grid gap-8 border-b pb-6 lg:grid-cols-[1fr_1.15fr]" style="border-color: var(--bn-border)">
						<div class="flex min-w-0 flex-col">
							<BigStat value={vol.plays} chips={vol.chips} caption={vol.caption} />
							{#key period}<MeterStack meters={vol.meters} foot="share of plays in this saved period · historical snapshot" />{/key}
						</div>
						<div class="min-w-0">
							<p class="bn-dim mb-2 text-[13px]">{videosIn(rows.length)}, with their watch behavior preserved from Vidalytics. Pick one to inspect.</p>
							<div class="border-t" style="border-color: var(--bn-border)">
								{#each rows as row (row.videoId)}
									{@const on = row.videoId === video.videoId}
									<button data-lens="c" onclick={() => (selected = row.videoId)} aria-pressed={on} data-on={on ? '' : undefined} class="bn-pressable fn-row fn-video grid w-full grid-cols-[1fr_auto] items-center gap-4 border-b px-1 py-3 text-left last:border-0" style="border-color: var(--bn-border)">
										<span class="min-w-0">
											<span class="bn-text block truncate text-[13.5px] {on ? 'font-semibold' : 'font-medium'}">{row.title}</span>
											<span class="bn-dim block truncate font-mono text-[11px]">{row.duration} · {count(row.uniqueViewers)} unique · play rate {percent(row.playRate)} · watched {percent(row.averageWatched)}</span>
										</span>
										<span class="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]" style={on ? 'background: var(--bn-accent-soft); color: var(--bn-accent)' : 'background: color-mix(in oklab, var(--bn-text) 8%, transparent); color: var(--bn-text-2)'}>{count(row.plays)} plays</span>
									</button>
								{/each}
							</div>
						</div>
					</div>
					<div class="pt-6">{@render detail()}</div>
					{@render caveat()}
				{/if}
			</div>
		</SlabCard>
	</section>
{/if}
