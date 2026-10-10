<!-- /os/social/beehiiv — FounderOS v1 app/social/beehiiv/page.tsx: sends, the
     Newsletter Volume card, open rates, engagement, the best-open card and
     the past newsletters, from GET /api/founderos/pages/social/beehiiv. Seeded
     issues (while Beehiiv returns none) are labelled a preview. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { ArrowLeft, ExternalLink, Mail } from '$lib/founderos/icons';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import { founderosFetch } from '$lib/founderos/api';
	import { PILL, PILL_ACCENT } from '$lib/founderos/kit/slab-classes';
	import '$lib/founderos/pages/pc/pc.css';
	import '$lib/founderos/pages/social/social.css';
	import NewsletterList from '$lib/founderos/pages/social/NewsletterList.svelte';
	import { HUE_RAMP1, HUE_SEND, matrixCols } from '$lib/founderos/pages/social/lib';
	import type { BeehiivPage } from '$lib/founderos/pages/social/types';

	let data = $state<BeehiivPage | null>(null);
	let error = $state<string | null>(null);

	onMount(() => {
		founderosFetch<BeehiivPage>('/pages/social/beehiiv')
			.then((d) => (data = d))
			.catch((e: unknown) => (error = e instanceof Error ? e.message : String(e)));
	});

	const issues = $derived(data?.newsletters.length ?? 0);
	// Prod's copy: "live via Beehiiv API", or the seeded preview without a key.
	// The port adds the one state prod cannot reach: a key that stopped answering.
	const meta = $derived(
		!data
			? 'loading…'
			: data.live
				? data.fresh
					? 'live via Beehiiv API'
					: 'last good Beehiiv reading · API failing'
				: data.seeded
					? 'seeded preview · add BEEHIIV_API_KEY for live'
					: 'no live subscriber count · add BEEHIIV_API_KEY for live'
	);
</script>

{#snippet actions()}
	{#if data}
		<Chip tone={data.seeded ? 'warn' : data.live ? 'ok' : 'err'}>{data.seeded ? 'seeded' : 'beehiiv live'}</Chip>
	{/if}
	<a href="/os/social" class={PILL}><ArrowLeft size={14} /> All platforms</a>
	<a href="https://app.beehiiv.com" target="_blank" rel="noreferrer" data-lens="c" class={PILL_ACCENT}>Open Beehiiv <ExternalLink size={14} /></a>
{/snippet}

<Slab>
	<SlabTitle eyebrow="beehiiv · email list" title="Newsletter" {meta}>
		{#snippet right()}{@render actions()}{/snippet}
	</SlabTitle>

	{#if error}
		<p class="soc-err font-mono text-[12px]" data-part="error">The newsletter is unavailable: {error}</p>
	{:else if !data}
		<div class="bn-dim font-mono text-[11px]" data-part="loading">Reading Beehiiv…</div>
	{:else}
		{@const v = data.volume}
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Sends" sub={`${issues} issue${issues === 1 ? '' : 's'}`} class="pb-2">
				<div class="px-6 pb-2 pt-3">
					<BigStat size={30} value={v.totalRecipients} kind="followers" caption="recipients across every issue, oldest to newest" />
				</div>
				<StepLine series={v.sends} hue={HUE_SEND} unit=" sent" empty="No issues sent yet." />
			</SlabCard>

			<SlabCard i={2} title="Newsletter Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={v.headline} display={v.headline == null ? '—' : undefined} kind="followers" chips={v.chips} caption={v.caption} />
					<MeterStack meters={v.meters} foot={v.foot} empty="no sends to measure yet" />
				</div>
			</SlabCard>
		</div>

		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Open Rate" sub="last 6 issues">
				<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						display={v.avgOpenRate == null ? '—' : `${v.avgOpenRate.toFixed(1)}%`}
						caption={issues > 0 ? `average open rate across ${issues} issue${issues === 1 ? '' : 's'}` : 'no issues sent yet'}
					/>
					{#if v.openRates.length > 0}<DotMatrix cols={matrixCols(v.openRates)} hue={HUE_RAMP1} />{/if}
				</div>
			</SlabCard>

			<SlabCard i={4} title="Engagement" sub="summed across issues">
				<div class="px-6 pb-6 pt-3">
					<BigStat size={30} value={v.totalClicks} unit="clicks" caption={issues > 0 ? 'link clicks across every issue' : 'no clicks recorded yet'} />
					<div class="mt-4 flex flex-wrap gap-2">
						<Chip tone={v.unsubscribes > 0 ? 'warn' : 'ok'}>{v.unsubscribes.toLocaleString('en-US')} unsubscribed</Chip>
						<Chip tone={v.spamReports > 0 ? 'err' : 'ok'}>{v.spamReports.toLocaleString('en-US')} spam reports</Chip>
					</div>
				</div>
			</SlabCard>

			<InsightCard i={5} badge="Best open" display={v.insight.display} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac}>
				{#snippet icon()}<Mail size={13} />{/snippet}
			</InsightCard>
		</div>

		<SlabCard i={6} title="Past newsletters" sub="click any issue to expand its analytics" class="mt-6">
			<div class="px-6 pb-6 pt-4">
				<NewsletterList newsletters={data.newsletters} />
			</div>
		</SlabCard>
	{/if}
</Slab>
