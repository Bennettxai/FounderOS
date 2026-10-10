<!-- Trakyo channel funnels as a Brand Deals slab card (FounderOS v1
     components/FunnelChannelAnalytics.tsx), shared by /os/analytics and
     /os/funnel: first-touch revenue over three hand-off meters, channel
     chips, content rows, the compact IG VSL and the link studio. -->
<script lang="ts">
	import { RefreshCw } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, MeterStack, PILL, SlabCard } from '$lib/founderos/kit';
	import TrakyoLinkStudio from './TrakyoLinkStudio.svelte';
	import VslAnalytics from './VslAnalytics.svelte';
	import type { ChannelAnalyticsBody, VslSnapshot } from './types';
	import { trakyoVolume } from './volume';

	let {
		vslSnapshots = [],
		igVideo = null,
		vslError = null,
		showVsl = true,
		i = 6
	}: { vslSnapshots?: VslSnapshot[]; igVideo?: string | null; vslError?: string | null; showVsl?: boolean; i?: number } = $props();

	let period = $state<'7d' | '30d' | '90d'>('30d');
	let refresh = $state(0);
	let data = $state<ChannelAnalyticsBody | null>(null);
	let error = $state('');
	let loading = $state(true);
	let selected = $state('instagram_bio');
	let attribution = $state<'first_touch' | 'last_touch'>('first_touch');

	$effect(() => {
		const p = period;
		void refresh;
		let alive = true;
		loading = true;
		error = '';
		data = null;
		founderosFetch<ChannelAnalyticsBody>(`/pages/funnel/analytics?period=${p}`)
			.then((v) => {
				if (alive) data = v;
			})
			.catch(() => {
				if (alive) error = 'Unable to load Trakyo analytics. Try refreshing.';
			})
			.finally(() => {
				if (alive) loading = false;
			});
		return () => {
			alive = false;
		};
	});

	const channel = $derived(data?.channels.find((c) => c.id === selected));
	const funnel = $derived(data?.funnel.data ?? null);
	const vol = $derived(funnel ? trakyoVolume(funnel) : null);
	const count = (v: number) => v.toLocaleString('en-US');
	const money = (v: number) => v.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 });
	// prod's select control and channel box (FunnelChannelAnalytics.tsx `control`).
	const CONTROL = 'fn-control rounded-full border px-3.5 py-2 text-[12.5px]';
	const COLS = ['Video / content', 'Raw video views*', 'Clicks', 'Visits', 'Forms', 'Bookings', 'Revenue'];
</script>



<section aria-label="Trakyo channel analytics" aria-busy={loading} class="mt-6" data-testid="channel-funnels">
	<SlabCard {i} title="Channel Funnels" sub="Trakyo · all businesses · independent of the lead graph filters">
		{#snippet action()}
			<label class="sr-only" for="trakyo-period">Analytics period</label>
			<select id="trakyo-period" class={CONTROL} bind:value={period}>
				<option value="7d">Last 7 days</option><option value="30d">Last 30 days</option><option value="90d">Last 90 days</option>
			</select>
			<button data-lens="c" class={PILL} disabled={loading} onclick={() => (refresh += 1)}><RefreshCw size={13} strokeWidth={1.7} /> Refresh</button>
		{/snippet}
		<div class="px-6 pb-6 pt-4">
			{#if loading}<p role="status" class="bn-dim py-8 text-[13px]">Loading channel and video analytics...</p>{/if}
			{#if error}<p role="alert" class="text-[13px]">{error}</p>{/if}
			{#if data}
				{#if data.content.state === 'not_configured'}
					<p class="py-4 text-[13px]">Connect Trakyo with a server-side API key to see channel and video performance.</p>
				{/if}
				{#each [data.funnel.message, data.content.message].filter(Boolean) as message, k (k)}
					<p role="status" class="mb-3 text-[12.5px]" style="color: var(--bn-warn)">{message}</p>
				{/each}
				{#if funnel && vol}
					<div class="grid gap-8 border-b pb-6 lg:grid-cols-[1fr_1.15fr]" style="border-color: var(--bn-border)">
						<div class="min-w-0">
							<BigStat display={vol.headline} chips={vol.chips} caption={vol.caption} />
							<div class="mt-6 grid grid-cols-3 gap-x-6 gap-y-4 sm:grid-cols-5">
								{#each Object.entries({ 'Tracked clicks': funnel.counts.clicks, 'Page visits': funnel.counts.visits, Forms: funnel.counts.form_submissions, Bookings: funnel.counts.bookings, Closes: funnel.counts.closes }) as [label, value] (label)}
									<div class="min-w-0"><p class="bn-dim truncate text-[12px]">{label}</p><p class="bn-text mt-1 text-[22px] font-semibold tabular-nums tracking-[-0.02em]">{count(value)}</p></div>
								{/each}
							</div>
						</div>
						<div class="flex min-w-0 flex-col"><MeterStack meters={vol.meters} foot={vol.foot} /></div>
					</div>
				{/if}
				{#if data.content.state === 'ready' || data.content.state === 'partial'}
					<div class="pt-5">
						<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
							<p class="bn-dim text-[13px]">Choose a channel to inspect its videos and tracked content.</p>
							<label class="bn-dim flex items-center gap-2 text-[12.5px]">
								Content revenue
								<select aria-label="Content revenue attribution" class={CONTROL} bind:value={attribution}><option value="first_touch">First touch</option><option value="last_touch">Last touch</option></select>
							</label>
						</div>
						<!-- One box per channel funnel (2026-09-24: brighter, obviously
						     clickable, less noisy). Channels with tracked content are boxes;
						     empty ones collapse into one quiet line, still selectable. -->
						<nav aria-label="Choose analytics channel" class="flex flex-wrap gap-2">
							{#each data.channels.filter((c) => c.items.length > 0) as c (c.id)}
								{@const on = c.id === selected}
								<button data-lens="c" type="button" class="bn-pressable fn-channel group flex min-w-[128px] items-center justify-between gap-3 rounded-[10px] border px-3 py-2 text-left" data-on={on ? '' : undefined} aria-pressed={on} onclick={() => (selected = c.id)}>
									<span class="min-w-0">
										<span class="block truncate text-[13px] font-semibold" style="color: {on ? 'var(--bn-accent)' : 'var(--bn-text)'}">{c.label}</span>
										<span class="bn-muted block truncate font-mono text-[10.5px] tabular-nums">{c.items.length} item{c.items.length === 1 ? '' : 's'} · {count(c.clicks)} clicks</span>
									</span>
									<span aria-hidden="true" class="shrink-0 text-[15px] leading-none" style="color: {on ? 'var(--bn-accent)' : 'var(--bn-text-2)'}">›</span>
								</button>
							{/each}
						</nav>
						{#if data.channels.some((c) => c.items.length === 0)}
							<p class="bn-dim mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[10.5px]">
								<span class="uppercase tracking-[0.12em]">no tracked content</span>
								{#each data.channels.filter((c) => c.items.length === 0) as c (c.id)}
									<button type="button" aria-pressed={c.id === selected} onclick={() => (selected = c.id)} data-lens="c" class="bn-pressable fn-quiet-channel underline-offset-2 hover:underline" data-on={c.id === selected ? '' : undefined}>{c.label}</button>
								{/each}
							</p>
						{/if}
						{#if channel && channel.items.length > 0}
							<div class="mt-6 flex flex-wrap gap-x-9 gap-y-3">
								{#each [[count(channel.clicks), 'clicks'], [count(channel.visits), 'visits'], [count(channel.forms), 'forms'], [count(channel.bookings), 'bookings'], [money(channel.revenue[attribution]), `${attribution === 'first_touch' ? 'first' : 'last'}-touch revenue`]] as [value, label] (label)}
									<p class="flex items-baseline gap-2"><strong class="bn-text text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{value}</strong><span class="bn-dim text-[13px]">{label}</span></p>
								{/each}
							</div>
						{/if}
						{#if channel}
							<div class="mt-6">
								<h3 class="bn-text text-[15px] font-semibold">{channel.label} <span class="bn-dim ml-1 font-mono text-[12px] font-normal tabular-nums">{channel.items.length} content items</span></h3>
								{#if channel.items.length === 0}
									<p class="bn-dim py-6 text-[12.5px]">No content attributed to {channel.label} in this response. Use a matching source name in Trakyo; unknown sources are not counted as word of mouth.</p>
								{:else}
									<div class="-mx-6 mt-3 overflow-x-auto border-t" style="border-color: var(--bn-border)">
										<table class="w-full text-left text-[12.5px]">
											<thead class="bn-dim font-mono text-[10.5px] uppercase tracking-[0.1em]"><tr>
												{#each COLS as label, k (label)}<th class="whitespace-nowrap py-3 font-normal {k === 0 ? 'pl-6 pr-3' : k === 6 ? 'pl-3 pr-6 text-right' : 'px-3'}">{label}</th>{/each}
											</tr></thead>
											<tbody>
												{#each channel.items as item, k (`${item.id ?? item.name}-${k}`)}
													<tr class="fn-row border-t" style="border-color: var(--bn-border)">
														<td class="min-w-[220px] max-w-[400px] py-3.5 pl-6 pr-3"><p class="bn-text truncate text-[13.5px] font-medium">{item.name}</p><p class="bn-dim mt-0.5 truncate font-mono text-[11px]">{item.source} · {item.type}{item.published_at ? ` · Published ${item.published_at.slice(0, 10)}` : ''}</p></td>
														{#each [item.type === 'youtube' && item.views !== null ? count(item.views) : 'N/A', count(item.clicks), count(item.visits), count(item.form_submissions), count(item.bookings)] as value, idx (idx)}
															<td class="bn-muted whitespace-nowrap px-3 py-3.5 font-mono tabular-nums">{value}</td>
														{/each}
														<td class="whitespace-nowrap py-3.5 pl-3 pr-6 text-right"><span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums tracking-[0.04em]" style="background: var(--bn-accent-soft); color: var(--bn-accent)">{money(Number(item.revenue[attribution]))}</span></td>
													</tr>
												{/each}
											</tbody>
										</table>
									</div>
								{/if}
							</div>
						{/if}
						<p class="bn-dim mt-4 text-[11.5px] leading-relaxed">* Raw YouTube views are lifetime/platform totals, not views during the selected period. N/A means unavailable. Clicks, visits, forms, bookings and revenue use the selected period. First-touch and last-touch revenue are alternative attribution models, never added together. Channel totals cover returned content, not necessarily all account events.</p>
					</div>
				{/if}
				{#if showVsl && (selected === 'instagram' || selected === 'instagram_bio')}
					<VslAnalytics snapshots={vslSnapshots} {igVideo} error={vslError} compact />
				{/if}
				<TrakyoLinkStudio content={data.content.rows} onCreated={() => (refresh += 1)} />
				<p class="bn-dim mt-4 font-mono text-[10.5px]">Checked {new Date(data.fetchedAt).toLocaleString()} · Source: Trakyo API</p>
			{/if}
		</div>
	</SlabCard>
</section>
