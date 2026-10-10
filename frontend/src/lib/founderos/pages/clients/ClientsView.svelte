<!-- /clients in the Brand Deals slab (FounderOS v1 app/clients/page.tsx):
     hero 2fr/1fr (Request Activity step line + Request Volume), a second row
     (Request Rhythm dot matrix, Roster, the one InsightCard), then the board.
     Every number comes from the backend's clientsVolume view-model. -->
<script lang="ts">
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import ClientWorkspaceBoard from './ClientWorkspaceBoard.svelte';
	import { ACTIVITY_HUE, busiestDay, type ClientsPayload } from './types';

	let { data, onchange }: { data: ClientsPayload; onchange?: () => void | Promise<void> } = $props();

	const v = $derived(data.volume);
	const days = $derived(data.windowDays);
	const busiest = $derived(busiestDay(v.rhythm));
</script>

<Slab>
	<SlabTitle
		eyebrow="client work · draft first"
		title="Clients"
		meta="Client context, marketing requests, and Superset workspaces. Draft first. Review before publishing."
	>
		{#snippet right()}
			<Chip tone={v.counts.needs_attention > 0 ? 'err' : 'ok'}>{data.clients.length} confirmed · {v.headline} requests</Chip>
			<a href="#new-request" data-lens="c" class={PILL}>New request</a>
		{/snippet}
	</SlabTitle>

	<div data-part="hero" class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={1} title="Request Activity" sub="last {days} days" class="pb-2">
			<div class="px-6 pb-2 pt-3">
				<BigStat size={30} value={v.requestsInWindow} caption="briefs saved for confirmed clients" />
			</div>
			<StepLine series={v.series} hue={ACTIVITY_HUE} unit=" requests" empty="No requests in the last {days} days." />
		</SlabCard>

		<SlabCard i={2} title="Request Volume" class="flex flex-col">
			<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
				<BigStat value={v.headline} chips={v.chips} caption={v.caption} />
				<MeterStack meters={v.meters} foot={v.foot} empty="no requests yet, save a brief below" />
			</div>
		</SlabCard>
	</div>

	<div data-part="second-row" class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={3} title="Request Rhythm" sub="by weekday">
			<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
				<BigStat
					size={30}
					display={busiest && busiest.count > 0 ? busiest.label : 'none'}
					caption={busiest && busiest.count > 0 ? `busiest day · ${busiest.count} requests` : `no requests in ${days} days`}
				/>
				<DotMatrix cols={v.rhythm} hue={ACTIVITY_HUE} />
			</div>
		</SlabCard>

		<SlabCard i={4} title="Roster" sub="confirmed by the operator">
			<div class="px-6 pb-6 pt-3">
				<BigStat size={30} value={data.clients.length} unit="confirmed" caption="paying cohort members are not retainers" />
				<div data-part="roster" class="mt-4 flex flex-wrap gap-2">
					{#each data.clients as c (c.id)}
						<Chip tone="accent">{c.name} · {v.perClient.find((p) => p.id === c.id)?.count ?? 0}</Chip>
					{/each}
					<Chip>Slack bridge off</Chip>
				</div>
			</div>
		</SlabCard>

		<InsightCard i={5} badge="Needs you" value={v.insight.value} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac} />
	</div>

	<ClientWorkspaceBoard clients={data.clients} work={data.work} statusOrder={data.statusOrder} launchEnabled={data.launchEnabled} {onchange} />
</Slab>

