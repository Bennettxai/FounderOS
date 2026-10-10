<!-- /os/social — FounderOS v1 app/social/page.tsx: the Accounts card (every
     channel + the email list) beside Audience Volume, Posting Activity /
     Platform Mix / the Needs-reply card, the stat strip (series popouts + the
     Instagram DM inbox), the audience + posting-consistency chart with the
     share donut, recent posts and the publish composer.
     Data: GET /api/founderos/pages/social. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Mail } from '$lib/founderos/icons';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import CountUp from '$lib/founderos/kit/CountUp.svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { operatorName } from '$lib/founderos/operator';
	import { PILL, PILL_ACCENT } from '$lib/founderos/kit/slab-classes';
	import '$lib/founderos/pages/pc/pc.css';
	import '$lib/founderos/pages/social/social.css';
	import AudienceChart from '$lib/founderos/pages/social/AudienceChart.svelte';
	import PlatformIcon from '$lib/founderos/pages/social/PlatformIcon.svelte';
	import PostComposer from '$lib/founderos/pages/social/PostComposer.svelte';
	import SharePie from '$lib/founderos/pages/social/SharePie.svelte';
	import StatStrip from '$lib/founderos/pages/social/StatStrip.svelte';
	import {
		agoFrom,
		averageLikeToView,
		formatFollowers,
		formatPct,
		formatRatioPct,
		HUE_RAMP1,
		HUE_SEND,
		likeToViewRatio,
		platformLabel,
		SAMPLE_POSTS
	} from '$lib/founderos/pages/social/lib';
	import type { SocialPage } from '$lib/founderos/pages/social/types';

	let data = $state<SocialPage | null>(null);
	let error = $state<string | null>(null);
	let nowMs = $state(Date.now());

	onMount(() => {
		founderosFetch<SocialPage>('/pages/social')
			.then((d) => {
				data = d;
				nowMs = Date.now();
			})
			.catch((e: unknown) => (error = e instanceof Error ? e.message : String(e)));
	});

	const v = $derived(data?.volume);
	const total = $derived(data?.audienceTotal ?? 0);
	const share = (n: number | null) => (total > 0 && n != null ? (n / total) * 100 : 0);
	const topPlatform = $derived(v && v.mix.length > 0 ? v.mix.reduce((best, m) => (m.count > best.count ? m : best), v.mix[0]) : null);
	const syncChip = $derived.by(() => {
		const s = data?.sync.source;
		if (s === 'zernio-live') return { tone: 'ok' as const, text: 'zernio live' };
		if (s === 'zernio-config') return { tone: 'warn' as const, text: 'zernio config · live api down' };
		return { tone: 'err' as const, text: 'zernio offline' };
	});
	const livePosts = $derived(data?.recentPosts ?? []);
	const recentLive = $derived(livePosts.length > 0);
	// v1 keeps channel order in the donut: every platform, then the email list.
	const pieItems = $derived(
		data
			? [
					...data.platforms.map((p) => ({ key: p.platform, label: platformLabel(p.platform), value: p.followers })),
					{ key: 'email', label: 'Email list', value: data.emailList.subscribers }
				]
			: []
	);
	const operator = $derived(operatorName(page.data?.user as { name?: unknown; email?: unknown } | undefined));
	const newsletterName = $derived(operator === 'there' ? 'Newsletter' : `${operator}'s Newsletter`);
	const growthTone = (n: number | null) => (n == null ? 'bn-dim' : n >= 0 ? 'soc-ok' : 'soc-err');
</script>

{#snippet recency(rank: number, of: number)}
	<!-- one dot per post in the set; lit count = how recent (all lit = newest) -->
	<div class="mt-1.5 flex items-center gap-1" title={`#${rank + 1} most recent of ${of}`} data-part="recency">
		{#each Array.from({ length: of }, (_, d) => d) as d (d)}
			<span
				class="h-1.5 w-1.5 rounded-full"
				style="background: {d < of - rank ? 'var(--bn-accent)' : 'var(--bn-surface-3)'}; opacity: {d < of - rank ? 0.45 + 0.55 * ((of - rank) / of) : 1}"
			></span>
		{/each}
	</div>
{/snippet}

{#if error}
	<Slab>
		<SlabTitle eyebrow="audience" title="Social" />
		<p class="soc-err font-mono text-[12px]" data-part="error">Social is unavailable: {error}</p>
	</Slab>
{:else if !data || !v}
	<Slab>
		<SlabTitle eyebrow="audience" title="Social" meta="loading…" />
		<div class="bn-dim font-mono text-[11px]" data-part="loading">Reading accounts, snapshots and Zernio…</div>
	</Slab>
{:else}
	<Slab class="soc-root">
		<SlabTitle
			eyebrow="audience"
			title="Social"
			meta={`${formatFollowers(total)} reach · ${data.postsKnown ? v.postsInWindow : '?'} posts in 30 days · ${data.queued} queued · ${data.dmThreads.length} DM threads`}
		>
			{#snippet right()}
				<Chip tone={syncChip.tone}>{syncChip.text}</Chip>
				<a href="/os/social/beehiiv" class={PILL}>Beehiiv</a>
				<a href="/os/agents" class={PILL_ACCENT}>Social agent</a>
			{/snippet}
		</SlabTitle>

		<!-- Hero row: every account + the volume card; click through for detail. -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Accounts" sub={`${formatFollowers(total)} total`}>
				<div class="grid grid-cols-2 gap-3 px-6 pb-6 pt-4 sm:grid-cols-3" data-part="accounts">
					{#each data.platforms as p, i (p.platform)}
						<a
							href={`/os/social/${p.platform}`}
							title={`${share(p.followers).toFixed(0)}% of reach`}
							data-lens="r"
							class="soc-tile soc-tile-bg bn-pressable is-row bn-rise block rounded-[10px] px-4 py-4"
							style="--rise-i: {i}"
							data-platform={p.platform}
						>
							<div class="flex items-center gap-2">
								<span class="bn-text shrink-0"><PlatformIcon platform={p.platform} /></span>
								<span class="bn-dim truncate font-mono text-[10px] uppercase tracking-[0.1em]">{platformLabel(p.platform)}</span>
								<span class="ml-auto shrink-0 font-mono text-[10px] {growthTone(p.growth.d7)}" title="7-day growth">{formatPct(p.growth.d7)}</span>
							</div>
							<div class="bn-text mt-3 font-mono text-[26px] font-semibold leading-none tracking-[-0.02em]">
								{#if p.followers == null}<span data-unknown>—</span>{:else}<CountUp value={p.followers} kind="followers" />{/if}
							</div>
							<div class="bn-dim mt-1.5 truncate font-mono text-[9.5px]">{p.handle}</div>
							<div class="soc-track mt-3 h-1 overflow-hidden"><div class="soc-fill h-full" style="width: {share(p.followers)}%"></div></div>
						</a>
					{/each}
					<a
						href="/os/social/beehiiv"
						title={`${share(data.emailList.subscribers).toFixed(0)}% of reach · open Beehiiv analytics`}
						data-lens="r"
						class="soc-tile soc-tile-bg bn-pressable is-row bn-rise block rounded-[10px] px-4 py-4"
						style="--rise-i: {data.platforms.length}"
						data-platform="email"
					>
						<div class="flex items-center gap-2">
							<Mail size={16} class="bn-accent shrink-0" />
							<span class="bn-dim truncate font-mono text-[10px] uppercase tracking-[0.1em]">Email list</span>
							<span class="ml-auto shrink-0 font-mono text-[10px] {growthTone(data.emailList.growth.d7)}" title="7-day growth">{formatPct(data.emailList.growth.d7)}</span>
						</div>
						<div class="bn-text mt-3 font-mono text-[26px] font-semibold leading-none tracking-[-0.02em]">
							{#if data.emailList.subscribers == null}<span data-unknown>—</span>{:else}<CountUp value={data.emailList.subscribers} kind="followers" />{/if}
						</div>
						<div class="bn-dim mt-1.5 truncate font-mono text-[9.5px]">Beehiiv · {newsletterName}</div>
						<div class="soc-track mt-3 h-1 overflow-hidden"><div class="soc-fill h-full" style="width: {share(data.emailList.subscribers)}%"></div></div>
					</a>
				</div>
			</SlabCard>

			<SlabCard i={2} title="Audience Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={v.headline} kind="followers" chips={v.chips} caption={v.caption} />
					<MeterStack meters={v.meters} foot={v.foot} empty="no channel is reporting followers yet" />
				</div>
			</SlabCard>
		</div>

		<!-- Second row: posting line, platform mix, THE gradient card -->
		<div class="mb-6 mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Posting Activity" sub="last 30 days">
				<div class="px-6 pt-3">
					<BigStat
						size={30}
						value={v.postsInWindow}
						display={data.postsKnown ? undefined : '—'}
						caption={data.postsKnown ? 'posts out through Zernio' : 'Zernio not answering · posting history unknown'}
					/>
				</div>
				<StepLine series={v.series} hue={HUE_SEND} unit=" posts" empty={data.postsKnown ? 'No posts in the last 30 days.' : 'Posting history unavailable right now.'} />
			</SlabCard>

			<SlabCard i={4} title="Platform Mix" sub="posts per platform">
				<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						display={topPlatform && topPlatform.count > 0 ? topPlatform.label : data.postsKnown ? 'none' : '—'}
						caption={topPlatform && topPlatform.count > 0
							? `most posted · ${topPlatform.count} posts`
							: data.postsKnown
								? 'nothing posted in 30 days'
								: 'posting history unknown'}
					/>
					<DotMatrix cols={v.mix} hue={HUE_RAMP1} />
				</div>
			</SlabCard>

			<InsightCard i={5} badge="Needs reply" value={v.insight.value} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac} />
		</div>

		<StatStrip
			growthLeader={data.growthLeader}
			audienceTotal={total}
			audienceGrowth={data.audienceGrowth}
			totalDms={data.totalDms}
			dmGrowth={data.dmGrowth}
			platformsCount={data.platforms.length}
			dmThreads={data.dmThreads}
			{nowMs}
		/>

		<div class="mb-6">
			<AudienceChart audience={data.audiencePoints} postDays={data.postDays} today={data.today}>
				{#snippet aside()}
					<SharePie items={pieItems} {total} donutMax={172} />
				{/snippet}
			</AudienceChart>
		</div>

		<!-- Recent posts: box row, newest first; the dot strip grades recency. -->
		<SlabCard
			i={8}
			title="Recent posts"
			sub={recentLive ? `${livePosts.length} live · zernio` : `${formatRatioPct(averageLikeToView(SAMPLE_POSTS))} avg L/V · sample`}
		>
			<div class="grid gap-3 px-6 pb-6 pt-4 sm:grid-cols-2 xl:grid-cols-5" data-part="recent-posts">
				{#if recentLive}
					{#each livePosts as p, i (`${p.url}-${i}`)}
						<div data-lens="r" class="soc-tile soc-tile-bg bn-pressable is-row flex flex-col rounded-[10px] px-3.5 py-3">
							<div class="flex items-center justify-between gap-2">
								<span class="bn-accent truncate font-mono text-[10px] uppercase tracking-[0.1em]">{platformLabel(p.platform)}</span>
								<span class="bn-dim shrink-0 font-mono text-[10px]">{agoFrom(p.publishedAt, nowMs)}</span>
							</div>
							{@render recency(i, livePosts.length)}
							<div class="bn-text mt-2 line-clamp-3 text-[12px]">{p.caption.split('\n')[0]}</div>
							<div class="bn-dim mt-auto flex items-center gap-1.5 pt-2 font-mono text-[10px]">
								<span class={p.status === 'success' ? 'soc-ok' : 'soc-warn'}>{p.status}</span>
								{#if p.url}
									<a href={p.url} target="_blank" rel="noreferrer" class="soc-view ml-auto rounded-[5px] px-1.5 py-0.5">view →</a>
								{/if}
							</div>
						</div>
					{/each}
				{:else}
					{#each SAMPLE_POSTS as p, i (p.caption)}
						<div data-lens="r" class="soc-tile soc-tile-bg bn-pressable is-row flex flex-col rounded-[10px] px-3.5 py-3">
							<div class="flex items-center justify-between gap-2">
								<span class="bn-accent truncate font-mono text-[10px] uppercase tracking-[0.1em]">{p.tag}</span>
								<span class="bn-dim shrink-0 font-mono text-[10px]">{p.ago}</span>
							</div>
							{@render recency(i, SAMPLE_POSTS.length)}
							<div class="bn-text mt-2 line-clamp-3 text-[12px]">{p.caption}</div>
							<div class="bn-dim mt-auto flex items-center gap-1.5 pt-2 font-mono text-[10px]">
								<span>{formatFollowers(p.views)} {p.kind}</span>
								<span aria-hidden="true">·</span>
								<span>{formatFollowers(p.likes)} likes</span>
								<span class="soc-view ml-auto rounded-[5px] px-1.5 py-0.5" title="like-to-view ratio">{formatRatioPct(likeToViewRatio(p.likes, p.views))}</span>
							</div>
						</div>
					{/each}
				{/if}
			</div>
		</SlabCard>

		<!-- Publish: compose a post that queues for the Social agent -->
		<SlabCard i={9} title="Publish" sub={`${data.queued} queued`} class="mt-6">
			{#snippet action()}
				<a href="/os/agents" class={PILL}>Social agent</a>
			{/snippet}
			<div class="px-6 pb-6 pt-4">
				<PostComposer initialPosts={data.posts} />
			</div>
		</SlabCard>
	</Slab>
{/if}
