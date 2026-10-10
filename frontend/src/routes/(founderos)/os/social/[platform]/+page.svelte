<!-- /os/social/[platform] — FounderOS v1 app/social/[platform]/page.tsx: the
     follower history bars and growth badges, the Follower Volume card, the
     growth windows, daily gains and the 30-day change card, all from
     GET /api/founderos/pages/social/:platform. -->
<script lang="ts">
	import { ArrowLeft, ExternalLink } from '$lib/founderos/icons';
	import { BigStat, InsightCard, MeterStack, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
	import { PILL, PILL_ACCENT } from '$lib/founderos/kit/slab-classes';
	import '$lib/founderos/pages/pc/pc.css';
	import '$lib/founderos/pages/social/social.css';
	import FollowerBarChart from '$lib/founderos/pages/social/FollowerBarChart.svelte';
	import GrowthBadge from '$lib/founderos/pages/social/GrowthBadge.svelte';
	import { formatFollowers, formatPct, HUE_OK, platformLabel } from '$lib/founderos/pages/social/lib';
	import type { PlatformPage } from '$lib/founderos/pages/social/types';

	const WINDOW_DAYS = 30;

	let { data: route }: { data: { platform: string } } = $props();

	let data = $state<PlatformPage | null>(null);
	let error = $state<string | null>(null);
	let missing = $state(false);

	$effect(() => {
		const platform = route.platform;
		data = null;
		error = null;
		missing = false;
		founderosFetch<PlatformPage>(`/pages/social/${encodeURIComponent(platform)}`)
			.then((d) => (data = d))
			.catch((e: unknown) => {
				if (e instanceof FounderosApiError && e.status === 404) missing = true;
				else error = e instanceof Error ? e.message : String(e);
			});
	});

	const newest = $derived(data && data.snapshots.length > 0 ? data.snapshots[data.snapshots.length - 1] : null);
	const windows = $derived(
		data
			? ([
					['7 days', data.growth.d7],
					['30 days', data.growth.d30],
					['60 days', data.growth.d60],
					['all time', data.growth.allTime]
				] as const)
			: []
	);
</script>

{#snippet back()}
	<a href="/os/social" class={PILL}><ArrowLeft size={14} /> All platforms</a>
{/snippet}

{#if missing}
	<Slab>
		<SlabTitle eyebrow="audience" title={platformLabel(route.platform)}>{#snippet right()}{@render back()}{/snippet}</SlabTitle>
		<p class="bn-dim font-mono text-[12px]" data-part="not-found">{route.platform} is not a tracked account.</p>
	</Slab>
{:else if error}
	<Slab>
		<SlabTitle eyebrow="audience" title={platformLabel(route.platform)}>{#snippet right()}{@render back()}{/snippet}</SlabTitle>
		<p class="soc-err font-mono text-[12px]" data-part="error">This platform is unavailable: {error}</p>
	</Slab>
{:else if !data}
	<Slab>
		<SlabTitle eyebrow="audience" title={platformLabel(route.platform)} meta="loading…" />
		<div class="bn-dim font-mono text-[11px]" data-part="loading">Reading snapshots…</div>
	</Slab>
{:else}
	{@const v = data.volume}
	<Slab>
		<SlabTitle
			eyebrow={`audience · ${data.account.handle}`}
			title={data.label}
			meta={`${formatFollowers(data.followers)} followers · ${formatPct(data.growth.d7)} 7d · ${data.snapshots.length} snapshots`}
		>
			{#snippet right()}
				{@render back()}
				{#if data?.account.url}
					<a href={data.account.url} target="_blank" rel="noreferrer" data-lens="c" class={PILL_ACCENT}>Open profile <ExternalLink size={14} /></a>
				{/if}
			{/snippet}
		</SlabTitle>

		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Follower History" sub={`${data.snapshots.length} snapshots`}>
				{#snippet action()}
					<GrowthBadge label="7d" value={data!.growth.d7} />
					<GrowthBadge label="30d" value={data!.growth.d30} />
					<GrowthBadge label="60d" value={data!.growth.d60} />
					<GrowthBadge label="all" value={data!.growth.allTime} />
				{/snippet}
				<div class="px-6 pb-5 pt-2">
					<FollowerBarChart series={data.snapshots.map((s) => ({ date: s.capturedAt, followers: s.followers }))} />
					{#if newest}
						<div class="bn-dim mt-2 flex flex-wrap items-center gap-3 font-mono text-[10.5px]">
							<span class="flex items-center gap-1.5"><span class="inline-block h-[3px] w-3 rounded-full" style="background: var(--bn-ok)"></span> gained vs prev</span>
							<span class="flex items-center gap-1.5"><span class="inline-block h-[3px] w-3 rounded-full" style="background: var(--bn-err)"></span> dipped</span>
							<span class="ml-auto">latest {newest.capturedAt} · {newest.source}</span>
						</div>
					{/if}
				</div>
			</SlabCard>

			<SlabCard i={2} title="Follower Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={v.headline} display={v.headline == null ? '—' : undefined} kind="followers" chips={v.chips} caption={v.caption} />
					<MeterStack meters={v.meters} foot={v.foot} empty="no snapshots recorded for this platform yet" />
				</div>
			</SlabCard>
		</div>

		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Growth Windows" sub="vs the snapshot that far back">
				<div class="grid grid-cols-2 gap-x-6 gap-y-5 px-6 pb-6 pt-4" data-part="windows">
					{#each windows as [label, value] (label)}
						<BigStat
							size={30}
							display={formatPct(value)}
							chips={value == null ? [] : value === 0 ? [{ text: 'flat' }] : [{ tone: value > 0 ? 'ok' : 'err', text: value > 0 ? 'up' : 'down' }]}
							caption={value == null ? `${label} · not enough history` : label}
						/>
					{/each}
				</div>
			</SlabCard>

			<SlabCard i={4} title="Daily Gains" sub={`last ${WINDOW_DAYS} days`}>
				<div class="px-6 pt-3">
					<BigStat size={30} value={v.gainedDays} caption={v.intervals > 0 ? `days with a gain, of ${v.intervals} tracked` : 'no day-over-day history yet'} />
				</div>
				<StepLine series={v.series} hue={HUE_OK} unit=" gained" empty={`No follower gains in the last ${WINDOW_DAYS} days.`} />
			</SlabCard>

			<InsightCard
				i={5}
				badge={`${WINDOW_DAYS}-day change`}
				value={v.insight.value}
				display={v.insight.display}
				headline={v.insight.headline}
				body={v.insight.body}
				frac={v.insight.frac}
			/>
		</div>
	</Slab>
{/if}
