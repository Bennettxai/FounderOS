<!-- /os/comms (spec 6.2): FounderOS v1 app/comms/page.tsx on the bridge. One
     GET /api/founderos/pages/comms builds the whole view: the five sources,
     the per-inbox lanes, the Slack client board, the week's meetings, the
     Recordings tab (Plaud + Fathom), the stored morning report and its read
     state, and the Brand Deals-style numbers. Replies are guarded writes.
     Layout is v1's: slab title, Sources grid beside Message Volume, then
     Message Activity / Meetings This Week / Waiting on you, the morning
     report, the Inbox card and the dashed footer note. -->
<script lang="ts">
	import { CalendarDays, Hash, Mail, MessageSquare, Mic } from '$lib/founderos/icons';
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { Badge, BigStat, CountUp, Dot, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import CommsTabs from '$lib/founderos/pages/comms/CommsTabs.svelte';
	import DigestPanel from '$lib/founderos/pages/comms/DigestPanel.svelte';
	import type { CommsPage } from '$lib/founderos/pages/comms/model';
	import '$lib/founderos/pages/pc/pc.css';

	const ICON: Record<string, typeof Mail> = { whatsapp: MessageSquare, email: Mail, slack: Hash, calendar: CalendarDays, plaud: Mic };
	/** FounderOS v1 --send-activity and --ramp-1, over the port's brain tokens. */
	const SEND_ACTIVITY = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';
	const RAMP_1 = 'var(--bn-brain-2)';
	const EYEBROW = 'by source · inboxes · whatsapp · slack · calendar · recorder';

	let page = $state<CommsPage | null>(null);
	let failure = $state<string | null>(null);

	onMount(() => {
		founderosFetch<CommsPage>('/pages/comms')
			.then((p) => (page = p))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});

	const nowMs = $derived(page ? Date.parse(page.now) : Date.now());
	const vol = $derived(page?.volume);
	const connected = $derived(page ? page.sources.filter((s) => s.state === 'connected').length : 0);
	const recErrors = $derived(page?.recordings.errors?.length ?? 0);
</script>

{#if failure}
	<Slab>
		<SlabTitle eyebrow={EYEBROW} title="Comms" meta="unreachable" />
		<p data-part="page-error" class="font-mono text-[12px]" style="color: var(--bn-err)">Comms could not be loaded: {failure}</p>
	</Slab>
{:else if !page || !vol}
	<Slab>
		<SlabTitle eyebrow={EYEBROW} title="Comms" meta="reading inboxes, WhatsApp, Slack, calendar and recorders…" />
		<div data-part="loading" class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Sources" sub="checking…"><div class="h-[200px]"></div></SlabCard>
			<SlabCard i={2} title="Message Volume"><div class="px-6 pb-6 pt-3"><BigStat display="…" caption="counting unread" /></div></SlabCard>
		</div>
	</Slab>
{:else}
	<Slab>
		<SlabTitle eyebrow={EYEBROW} title="Comms" meta={vol.meta}>
			{#snippet right()}
				<Badge tone="accent">{vol.headline} unread</Badge>
				<a href="#inbox" class={PILL}>Open inbox</a>
			{/snippet}
		</SlabTitle>

		{#if page.gaps?.length}
			<p data-part="gaps" class="mb-3 font-mono text-[10.5px]" style="color: var(--bn-warn)">Partly unavailable: {page.gaps.join(' · ')}</p>
		{/if}

		<!-- Hero row: the sources + the Deal Volume card worn by comms numbers -->
		<div data-part="hero-row" class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Sources" sub="{connected}/{page.sources.length} connected" class="pb-6">
				<div data-part="sources-grid" class="mt-4 grid gap-3 px-6 sm:grid-cols-2 xl:grid-cols-3">
					{#each page.sources as source (source.id)}
						{@const Icon = ICON[source.id] ?? Mail}
						{@const ok = source.state === 'connected'}
						{@const label = ok ? 'Connected' : source.state === 'error' ? 'Error' : 'Not configured'}
						<div data-part="source" data-source={source.id} data-lens="r" class="bn-pressable is-row rounded-[12px] border px-4 py-3.5" style="border-color: var(--bn-border); background: var(--bn-bg)">
							<div class="flex items-center gap-[9px]">
								<Icon size={15} strokeWidth={1.7} class="shrink-0 {ok ? 'bn-accent' : 'bn-dim'}" />
								<span class="text-[13.5px] font-medium">{source.name}</span>
								<span class="ml-auto flex items-center gap-2">
									{#if ok}<Dot state="connected" pulse />{/if}
									<Badge tone={ok ? 'ok' : source.state === 'error' ? 'err' : 'default'} ghost={source.state === 'not_configured'}>{label}</Badge>
								</span>
							</div>
							<p class="bn-dim mt-[9px] font-mono text-[10.5px] leading-relaxed">{source.detail}</p>
						</div>
					{/each}
				</div>
			</SlabCard>

			<SlabCard i={2} title="Message Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={vol.headline} chips={vol.chips} caption={vol.caption} />
					<MeterStack meters={vol.meters} foot={vol.foot} />
				</div>
			</SlabCard>
		</div>

		<!-- Second row: message activity, the week's meetings, THE insight card -->
		<div data-part="stat-row" class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Message Activity">
				<div class="px-6 pt-3">
					<!-- no whitespace between the two spans: v1's gap is the ml-2 alone -->
					<span class="text-[30px] font-semibold tabular-nums tracking-[-0.03em]"><CountUp value={vol.seriesTotal} /></span><span
						class="bn-dim ml-2 text-[13px]">threads in view, last 14 days</span
					>
				</div>
				<StepLine series={vol.series} hue={SEND_ACTIVITY} empty="No messages in this window." />
			</SlabCard>

			<SlabCard i={4} title="Meetings This Week">
				<div data-part="meetings" class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
					<div>
						{#if page.calendar.error}
							<BigStat display="—" size={30} caption="calendar unreachable · meetings unknown" />
						{:else}
							<BigStat value={vol.meetingsTotal} size={30} caption="meetings, next 7 days" />
						{/if}
						<div data-part="recordings-chip" class="bn-muted mt-4 rounded-full border px-3 py-1 text-[12px]" style="border-color: var(--bn-border)">
							Recordings: <span class="font-semibold tabular-nums">{recErrors >= 2 ? 'unknown' : page.recordings.recordings.length}</span>
						</div>
						{#if recErrors === 1}
							<div class="mt-1 font-mono text-[10px]" style="color: var(--bn-warn)">{page.recordings.errors?.[0]?.split(':')[0]} unreachable</div>
						{/if}
					</div>
					{#if !page.calendar.error}
						<DotMatrix cols={vol.meetings} hue={RAMP_1} />
					{/if}
				</div>
			</SlabCard>

			<InsightCard i={5} badge="Waiting on you" value={vol.insight.value} headline={vol.insight.headline} body={vol.insight.body} frac={vol.insight.frac} />
		</div>

		<div class="bn-rise mt-6" style="--rise-i: 6">
			<DigestPanel initial={page.digest} initialRead={page.readKeys} {nowMs} />
		</div>

		<!-- Swappable front: the messaging board or the 7-day calendar or the recordings -->
		<SlabCard i={7} title="Inbox" sub="{vol.headline} unread" class="mt-6">
			<div id="inbox" class="scroll-mt-6 px-6 pb-6 pt-4">
				<CommsTabs {page} {nowMs} />
			</div>
		</SlabCard>

		<p
			data-part="footer-note"
			class="bn-rise bn-dim mt-6 rounded-full border border-dashed px-4 py-3 text-center font-mono text-[10.5px]"
			style="--rise-i: 8; border-color: var(--bn-border-strong)"
		>
			Four inboxes (expand to read + reply) and WhatsApp as lanes · Slack per client + every current channel · meetings via CalDAV · recordings from Plaud + Fathom
		</p>
	</Slab>
{/if}
