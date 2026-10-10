<!-- /os/content/lead-magnets: FounderOS v1 app/content/lead-magnets/page.tsx in
     the Brand Deals slab. Numbers above the table come from the backend's
     port of lib/lead-magnet-volume over ALL rows; ?status= only narrows the
     table (GET /api/founderos/pages/content/lead-magnets). -->
<script lang="ts">
	import { ArrowLeft, Megaphone } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle, StepLine, chipClass } from '$lib/founderos/kit';
	import '$lib/founderos/pages/pc/pc.css';
	import LeadMagnetTable from './LeadMagnetTable.svelte';
	import NewLeadMagnet from './NewLeadMagnet.svelte';
	import { HUE_RAMP1, HUE_RAMP3, HUE_SEND } from './hues';
	import type { LeadMagnetsPage } from './types';

	let { status = null }: { status?: string | null } = $props();

	const FILTERS = ['all', 'live', 'draft', 'paused', 'archived'] as const;

	let data = $state<LeadMagnetsPage | null>(null);
	let error = $state<string | null>(null);

	async function load(s: string | null) {
		try {
			data = await founderosFetch<LeadMagnetsPage>(`/pages/content/lead-magnets${s ? `?status=${encodeURIComponent(s)}` : ''}`);
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	$effect(() => {
		void load(status);
	});

	const reload = () => void load(status);
	const v = $derived(data?.volume);
	const active = $derived(data?.filter ?? 'all');
	const topDest = $derived(v?.destinations[0]);
	const topCapture = $derived(v ? v.captures.reduce((best, c) => (c.count > best.count ? c : best), v.captures[0]) : undefined);
	const countFor = (f: (typeof FILTERS)[number]) => (!v ? 0 : f === 'all' ? v.total : v.counts[f]);
</script>

<Slab>
	<SlabTitle
		eyebrow="content engine"
		title="Lead Magnets"
		meta={v ? `${v.total} pages · ${v.counts.live} live · ${v.foot}` : error ? 'unavailable' : 'loading…'}
	>
		{#snippet right()}
			{#if v}<Chip tone="ok">{v.counts.live} live</Chip>{/if}
			<a href="/os/content" data-lens="c" class={PILL}><ArrowLeft size={14} /> Content</a>
		{/snippet}
	</SlabTitle>

	{#if error && !data}
		<div class="pc-panel rounded-[10px] px-6 py-5 font-mono text-[12px]" style="color: var(--bn-err)">Lead magnets unavailable: {error}</div>
	{:else if !data || !v}
		<div class="pc-panel bn-dim rounded-[10px] px-6 py-5 font-mono text-[12px]">loading the register…</div>
	{:else}
		<!-- Hero row: pages shipped + the volume card -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Pages Shipped" sub={`last ${data.weeks} weeks`} class="pb-2">
				<div class="px-6 pb-2 pt-3">
					<BigStat
						size={30}
						value={v.total}
						chips={v.shippedInWindow > 0 ? [{ tone: 'accent', text: `${v.shippedInWindow} new in ${data.weeks} weeks` }] : []}
						caption="landing pages on record, week by week"
					/>
				</div>
				<StepLine series={v.series} hue={HUE_SEND} unit=" pages" empty="No landing pages on record yet." />
			</SlabCard>

			<SlabCard i={2} title="Magnet Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={v.headline} unit="live" chips={v.chips} caption={v.caption} />
					<MeterStack meters={v.meters} foot={v.foot} empty="no landing pages to measure yet" />
				</div>
			</SlabCard>
		</div>

		<!-- Second row: what they capture, where the leads land, the gradient card -->
		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Captures" sub="what each page asks for">
				<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						display={topCapture && topCapture.count > 0 ? topCapture.label : 'none'}
						caption={topCapture && topCapture.count > 0 ? `most pages · ${topCapture.count} of ${v.total}` : 'no pages yet'}
					/>
					<DotMatrix cols={v.captures} hue={HUE_RAMP1} />
				</div>
			</SlabCard>

			<SlabCard i={4} title="Where Leads Land" sub={`${v.destinations.length} destinations`}>
				<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						display={topDest ? topDest.label : 'none'}
						caption={topDest ? `top destination · ${topDest.count} page${topDest.count === 1 ? '' : 's'}` : 'no destinations yet'}
					/>
					{#if v.destinations.length > 0}<DotMatrix cols={v.destinations} hue={HUE_RAMP3} />{/if}
				</div>
			</SlabCard>

			<InsightCard
				i={5}
				badge="Since last launch"
				value={v.insight.value}
				display={v.insight.display}
				headline={v.insight.headline}
				body={v.insight.body}
				frac={v.insight.frac}
			>
				{#snippet icon()}<Megaphone size={13} strokeWidth={1.7} />{/snippet}
			</InsightCard>
		</div>

		<!-- The register: filter pills, the new-page form, the database table -->
		<SlabCard i={6} title="All pages" sub={`${data.rows.length} of ${v.total}`} class="mt-6">
			{#snippet action()}
				{#each FILTERS as f (f)}
					<a
						href={f === 'all' ? '/os/content/lead-magnets' : `/os/content/lead-magnets?status=${f}`}
						data-lens="c"
						class={chipClass(active === f)}
						aria-current={active === f ? 'page' : undefined}>{f.charAt(0).toUpperCase() + f.slice(1)} {countFor(f)}</a
					>
				{/each}
			{/snippet}
			<p class="bn-dim px-6 pt-2 text-[12.5px] leading-relaxed">
				Every landing page we ship, with the live link on each row. Open it, or copy it straight to whoever asked.
			</p>
			<div class="px-6 pt-4">
				<NewLeadMagnet onCreated={reload} />
			</div>
			{#if error}
				<p class="px-6 pb-2 font-mono text-[10.5px]" style="color: var(--bn-err)">Refresh failed: {error}</p>
			{/if}
			<div class="mt-2">
				{#if data.rows.length === 0 && v.total > 0}
					<div class="pc-hair bn-dim border-t px-6 py-8 text-center text-[12.5px]">Nothing matches that filter.</div>
				{:else}
					<LeadMagnetTable rows={data.rows} showCopy manage onChanged={reload} />
				{/if}
			</div>
		</SlabCard>
	{/if}
</Slab>
