<!-- /os/content: FounderOS v1 app/content/page.tsx: the Brand Deals slab
     (posting line + Content Volume, then crew runs, lead magnets and the
     since-last-post card), then content intelligence, the content agents (run
     in place) and the Zernio pipeline. Data: GET /api/founderos/pages/content
     (volume = the backend's port of lib/content-volume). -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { ArrowUpRight, BarChart3, Clapperboard, ExternalLink, Megaphone, Play } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Chip, Dot, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import { agoRounded } from '$lib/founderos/pages/pc/ui';
	import '$lib/founderos/pages/pc/pc.css';
	import BacklinkCard from './BacklinkCard.svelte';
	import vantageMark from './vantage-mark.png';
	import ContentAgentCard from './ContentAgentCard.svelte';
	import { HUE_RAMP1, HUE_SEND } from './hues';
	import { statusPillStyle, type ContentPage } from './types';

	// The Vantage content-intelligence system this view backlinks out to.
	const INTEL_URL = 'https://intel.vantage.example';
	const INTEL_ANALYTICS_URL = 'https://intel.vantage.example/my-analytics';

	let data = $state<ContentPage | null>(null);
	let error = $state<string | null>(null);
	let nowMs = $state(Date.now());

	onMount(() => {
		founderosFetch<ContentPage>('/pages/content')
			.then((d) => {
				data = d;
				nowMs = Date.now();
			})
			.catch((e: unknown) => (error = e instanceof Error ? e.message : String(e)));
	});

	const lead = $derived(data?.crew[0] ?? null);
	const workers = $derived(data ? (lead ? data.crew.slice(1) : data.crew) : []);
	const liveMagnets = $derived(data?.leadMagnets.filter((m) => m.status === 'live').length ?? 0);
	const platformLabel = (p: string) => p.charAt(0).toUpperCase() + p.slice(1);
	const v = $derived(data?.volume);
	const busiest = $derived(
		(v?.crewRuns ?? []).reduce((best, c) => (c.count > best.count ? c : best), { label: 'none', count: 0 })
	);
</script>

<Slab>
	<SlabTitle
		eyebrow="content engine"
		title="Content Creation"
		meta={data
			? `${data.crew.length} agents · ${data.leadMagnets.length} lead magnets · ${data.volume.activeDays} active days in ${data.windowDays}`
			: error
				? 'unavailable'
				: 'loading…'}
	>
		{#snippet right()}
			{#if data}<Chip tone="accent">{data.crew.length} agents</Chip>{/if}
			<a href="/os/content/lead-magnets" data-lens="c" class={PILL}>Lead magnets</a>
			<a href="/os/social" data-lens="c" class={PILL}>Social</a>
		{/snippet}
	</SlabTitle>

	{#if error}
		<div class="pc-panel rounded-[10px] px-6 py-5 font-mono text-[12px]" style="color: var(--bn-err)">
			Content page unavailable: {error}
		</div>
	{:else if !data}
		<div class="pc-panel bn-dim rounded-[10px] px-6 py-5 font-mono text-[12px]">loading the content engine…</div>
	{:else}
		<div data-part="volume">
			<!-- Hero row, Brand Deals' shape: the posting line + the volume card -->
			<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
				<SlabCard i={1} title="Posting Activity" sub={`last ${data.windowDays} days`} class="pb-2">
					<div class="px-6 pb-2 pt-3">
						<BigStat
							size={30}
							value={data.volume.postsInWindow}
							display={data.postsKnown ? undefined : '—'}
							chips={data.volume.activeDays > 0 ? [{ tone: 'accent', text: `${data.volume.activeDays} active days` }] : []}
							caption={data.postsKnown ? 'posts out through Zernio, one per cross-post' : 'Zernio not answering · posting history unknown'}
						/>
					</div>
					<StepLine
						series={data.volume.series}
						hue={HUE_SEND}
						unit=" posts"
						empty={data.postsKnown ? `No posts on record in the last ${data.windowDays} days.` : 'Posting history unavailable right now.'}
					/>
				</SlabCard>

				<SlabCard i={2} title="Content Volume" class="flex flex-col">
					<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
						<BigStat value={data.volume.headline} chips={data.volume.chips} caption={data.volume.caption} />
						<MeterStack meters={data.volume.meters} foot={data.volume.foot} empty="no posts, magnets or crew runs to measure yet" />
					</div>
				</SlabCard>
			</div>

			<!-- Second row: crew runs, the lead magnets, THE gradient card -->
			<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
				<SlabCard i={3} title="Crew Runs" sub={`last ${data.windowDays} days`}>
					<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
						<BigStat
							size={30}
							value={data.volume.runsInWindow}
							caption={busiest.count > 0 ? `busiest · ${busiest.label} with ${busiest.count}` : 'no crew runs in this window'}
						/>
						<DotMatrix cols={data.volume.crewRuns} hue={HUE_RAMP1} />
					</div>
				</SlabCard>

				<SlabCard i={4} title="Lead Magnets" sub={`${data.volume.magnets.total} pages`}>
					{#snippet action()}
						<a href="/os/content/lead-magnets" data-lens="c" class={PILL}>Open <ArrowUpRight size={14} /></a>
					{/snippet}
					<div class="px-6 pb-6 pt-3">
						<BigStat
							size={30}
							value={data.volume.magnets.live}
							unit="live"
							caption={data.volume.magnets.total > 0 ? `of ${data.volume.magnets.total} landing pages shipped` : 'no landing pages recorded yet'}
						/>
						<div class="mt-4 flex flex-wrap gap-2">
							{#if data.volume.magnets.draft > 0}<Chip tone="warn">{data.volume.magnets.draft} draft</Chip>{/if}
							{#if data.volume.magnets.paused > 0}<Chip tone="err">{data.volume.magnets.paused} paused</Chip>{/if}
							{#if data.volume.magnets.archived > 0}<Chip>{data.volume.magnets.archived} archived</Chip>{/if}
						</div>
					</div>
				</SlabCard>

				<InsightCard
					i={5}
					badge="Since last post"
					value={data.volume.insight.value}
					display={data.volume.insight.display}
					headline={data.volume.insight.headline}
					body={data.volume.insight.body}
					frac={data.volume.insight.frac}
				/>
			</div>
		</div>

		<!-- Three equal sections: the lead magnet index and both Vantage surfaces -->
		<SlabCard i={6} title="Content intelligence" sub="3 surfaces" class="mt-6">
			<div data-part="intel" class="grid gap-3 px-6 pb-6 pt-4 md:grid-cols-3">
				<BacklinkCard
					href="/os/content/lead-magnets"
					internal
					icon={Megaphone as never}
					title="Lead Magnets"
					sub="Every landing page we ship, with the live link on each row."
					meta={`${data.leadMagnets.length} page${data.leadMagnets.length === 1 ? '' : 's'} · ${liveMagnets} live`}
				/>
				<BacklinkCard
					href={INTEL_URL}
					mark={vantageMark}
					title="Vantage Intel"
					sub="Your content intelligence system — research, hooks, and what's working, feeding the content agent."
				/>
				<BacklinkCard
					href={INTEL_ANALYTICS_URL}
					icon={BarChart3 as never}
					title="My Analytics"
					sub="Per-piece performance and audience analytics from the intelligence system."
				/>
			</div>
		</SlabCard>

		<!-- The content agent + crew -->
		<SlabCard i={7} title="Content agents" sub={`${data.crew.length}`} class="mt-6">
			{#snippet action()}
				<a href="/os/agents" data-lens="c" class={PILL}>Agents <ArrowUpRight size={14} /></a>
			{/snippet}
			<div class="px-6 pb-6 pt-3">
				<p class="bn-dim mb-4 flex items-center gap-1.5 text-[12.5px]">
					<Clapperboard size={14} /> Tied to your social media. Run one here or from Agents.
				</p>
				{#if lead}<ContentAgentCard agent={lead} lead />{/if}
				{#if workers.length > 0}
					<div class="mt-3 grid gap-3 lg:grid-cols-2">
						{#each workers as a (a.id)}<ContentAgentCard agent={a} />{/each}
					</div>
				{/if}
				{#if data.crew.length === 0}
					<p class="bn-dim font-mono text-[11px]">No agents on the Marketing/Growth pillar in the bridge roster.</p>
				{/if}
			</div>
		</SlabCard>

		<!-- Zernio content pipeline: recent published content -->
		<SlabCard i={8} title="Zernio content pipeline" sub={data.posts.length > 0 ? `${data.posts.length} recent` : 'no live pull'} class="mt-6">
			{#snippet action()}
				<a href="/os/social" data-lens="c" class={PILL}>Social <ArrowUpRight size={14} /></a>
			{/snippet}
			<p class="bn-dim px-6 pt-2 font-mono text-[11px]">
				Published across six platforms via Zernio · {data.pipelineActiveDays ?? 0} active days tracked
			</p>
			{#if data.posts.length > 0}
				<ul class="pc-hair mt-4 border-t">
					{#each data.posts as p, i (`${p.url}-${i}`)}
						<li data-lens="r" class="bn-pressable is-row pc-hair flex items-center gap-4 border-b px-6 py-3.5 last:border-0">
							<Play size={14} class="bn-dim shrink-0" />
							<Dot state={p.status === 'published' ? 'ok' : 'available'} />
							<span class="bn-muted w-24 shrink-0 font-mono text-[11px] uppercase tracking-wide">{platformLabel(p.platform)}</span>
							<span class="bn-text min-w-0 flex-1 truncate text-[13.5px]">{p.caption || 'Untitled post'}</span>
							<span class="hidden shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em] sm:inline" style={statusPillStyle(p.status)}>{p.status}</span>
							{#if p.url}
								<a href={p.url} target="_blank" rel="noopener noreferrer" data-lens="c" class="bn-pressable bn-dim shrink-0 rounded-full px-1" aria-label="open post"><ExternalLink size={14} /></a>
							{/if}
							<span class="bn-dim w-10 shrink-0 text-right font-mono text-[11px]">{agoRounded(p.publishedAt, nowMs)}</span>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="pc-panel bn-dim mx-6 mb-6 mt-4 rounded-[12px] border-dashed px-4 py-5 text-center font-mono text-[11.5px]">
					No live Zernio pull right now — recent content shows here once the API responds{data.postsError ? ` (${data.postsError})` : ''}.
				</p>
			{/if}
		</SlabCard>
	{/if}
</Slab>
