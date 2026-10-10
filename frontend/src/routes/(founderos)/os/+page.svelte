<!-- The operator console (spec 6.1, FounderOS v1 app/page.tsx). Every number is
     composed by GET /api/founderos/pages/console from live readings: the
     Connections board, the Optimal Engine topology and its health checks
     (G-Brain is retired; the engine's score stands where the brain's stood),
     the agent roster and runs in Postgres, the comms lanes and Stripe.
     Unknown reads unknown, never 0. Order and sections follow prod: pulse row,
     Needs you beside Operating volume, the activity/mix/insight row, then
     talk-to-the-OS beside Done today. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, CountUp, DotMatrix, InsightCard, Kbd, MeterStack, Slab, SlabCard, SlabTitle, StepLine, type StatChip } from '$lib/founderos/kit';
	import ConnectorBars from '$lib/founderos/pages/console/ConnectorBars.svelte';
	import DoneToday from '$lib/founderos/pages/console/DoneToday.svelte';
	import HealthMeter from '$lib/founderos/pages/console/HealthMeter.svelte';
	import Interject from '$lib/founderos/pages/console/Interject.svelte';
	import NeedsYou from '$lib/founderos/pages/console/NeedsYou.svelte';
	import SparkBars from '$lib/founderos/pages/console/SparkBars.svelte';
	import StatTile from '$lib/founderos/pages/console/StatTile.svelte';
	import { TONE_COLOR, brainTile, greeting, operatorName, usd, type ConsoleView } from '$lib/founderos/pages/console/console';

	let v = $state<ConsoleView | null>(null);
	let failure = $state<string | null>(null);
	let hello = $state(greeting());
	const who = $derived(operatorName(page.data?.user as { name?: unknown; email?: unknown } | undefined));

	onMount(() => {
		hello = greeting();
		founderosFetch<ConsoleView>('/pages/console')
			.then((body) => (v = body))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});

	const volumeChips = $derived.by<StatChip[]>(() => {
		if (!v) return [];
		const out: StatChip[] = [];
		if (v.volume.failedToday > 0) out.push({ tone: 'err', text: `${v.volume.failedToday} failed` });
		if (v.chargedTodayCents != null && v.chargedTodayCents > 0) out.push({ tone: 'ok', text: `${usd(v.chargedTodayCents)} charged` });
		return out;
	});

	// Prod's "G-Brain health · 90 / 100 · warnings", worn by the Optimal Engine's score.
	const brain = $derived(v ? brainTile(v.brain) : null);

	// prod's data-ramp tokens (globals.css) on the port's theme: --ramp-1 is the
	// theme's brain-2, --send-activity is brain-2 mixed 70% into the text.
	const RAMP_1 = 'var(--bn-brain-2)';
	const SEND_ACTIVITY = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';

	const laneTrouble = $derived(v ? v.sources.filter((s) => s.state === 'error' || s.state === 'stale') : []);
	const errorLines = $derived(v ? Object.entries(v.errors) : []);
</script>

<Slab>
	<SlabTitle eyebrow="command center" title={`${hello}, ${who}`}>
		{#snippet meta()}
			{#if v}
				<!-- Honest state-of-the-world line: what needs you, straight from live data -->
				<span class="flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px]">
					{#each v.hero as s, i (i)}
						<span class="flex items-center gap-2">
							{#if i > 0}<span style:color="var(--bn-border-strong)">·</span>{/if}
							<span style:color={TONE_COLOR[s.tone]}>{s.text}</span>
						</span>
					{/each}
				</span>
			{:else if failure}
				<span style:color="var(--bn-err)">console unreachable: {failure}</span>
			{:else}
				<span>reading the OS…</span>
			{/if}
		{/snippet}
		{#snippet right()}<Kbd>⌘K</Kbd>{/snippet}
	</SlabTitle>

	{#if failure && !v}
		<p class="bn-dim font-mono text-[12px]">Nothing below is drawn: the console endpoint did not answer, and zeros would be a lie.</p>
	{:else if !v}
		<div class="bn-dim font-mono text-[11px]" data-part="loading">checking connectors, engines, agents and comms…</div>
	{:else}
		{@const view = v}
		<!-- Pulse row -->
		<section class="bn-rise mb-6 grid grid-cols-4 gap-3 max-[1100px]:grid-cols-2" style="--rise-i: 1">
			<StatTile href="/os/integrations" label="Systems" unit={`/ ${view.systems.total} connected`}>
				{#snippet value()}<CountUp value={view.systems.connected} />{/snippet}
				{#snippet foot()}<ConnectorBars states={view.systems.bars} />{/snippet}
			</StatTile>
			<StatTile href="/os/agents" label="Agents live" unit={view.agents.total != null ? `/ ${view.agents.total} roster` : 'roster unknown'}>
				{#snippet value()}{#if view.agents.active != null}<CountUp value={view.agents.active} />{:else}—{/if}{/snippet}
				{#snippet foot()}{#if view.agents.spark}<SparkBars data={view.agents.spark} />{/if}{/snippet}
			</StatTile>
			<StatTile href="/os/comms" label="Communications" unit="inbound · 24h">
				{#snippet value()}<CountUp value={view.comms.inbound} />{/snippet}
				{#snippet foot()}<SparkBars data={view.comms.spark} />{/snippet}
			</StatTile>
			<StatTile href="/os/brain" label="Optimal Engine health" unit={brain?.unit ?? ''} accent>
				{#snippet value()}{#if brain?.value != null}<CountUp value={brain.value} />{:else}—{/if}{/snippet}
				{#snippet foot()}<HealthMeter value={brain?.value ?? null} />{/snippet}
			</StatTile>
		</section>

		<!-- Hero row, Brand Deals' shape: the queue that needs you + the volume card -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<!-- The volume card alone sets this row's height (self-start); side by
			     side, the queue is lifted out of flow to fill exactly that box and
			     scrolls inside it, so the two bottoms line up. Stacked, it flows. -->
			<section data-part="needs-you-slot" class="needs-slot bn-rise relative min-w-0" style="--rise-i: 2">
				<NeedsYou />
			</section>
			<SlabCard i={3} title="Operating volume" class="flex flex-col self-start">
				{#snippet action()}
					<a href="/os/agents" class="bn-linky bn-dim rounded-full border px-3 py-1 font-mono text-[11px]" style:border-color="var(--bn-border)">runs →</a>
				{/snippet}
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={view.volume.runsToday} chips={volumeChips} caption={`agent runs today across ${view.volume.agentsToday} agent${view.volume.agentsToday === 1 ? '' : 's'}`} />
					<MeterStack
						meters={view.volume.meters}
						foot={`${view.systems.connected} systems · ${view.agents.active ?? '—'} agents · brain ${brain?.value ?? '—'}/100`}
					/>
				</div>
			</SlabCard>
		</div>

		<!-- Second row: activity line, inbound mix dots, THE gradient card -->
		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={4} title="Agent activity">
				<div class="px-6 pt-3">
					<BigStat size={30} value={view.agents.spark ? view.activityTotal : null} caption="agent runs, last 14 days" />
				</div>
				<StepLine series={view.activity} hue={SEND_ACTIVITY} empty="No agent runs in the last 14 days." />
			</SlabCard>

			<SlabCard i={5} title="Inbound mix">
				<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
					<div>
						<BigStat size={30} value={view.feedCount} caption="latest messages in the feed" />
						<a href="/os/comms" class="bn-muted bn-hover-text mt-4 inline-block rounded-full border px-3 py-1 text-[12px]" style:border-color="var(--bn-border)">
							<span class="font-semibold tabular-nums">{view.comms.inbound}</span> in the last 24h →
						</a>
						{#each laneTrouble as s (s.source)}
							<div class="mt-2 font-mono text-[10.5px]" style:color={s.state === 'error' ? 'var(--bn-err)' : 'var(--bn-warn)'}>
								{s.source}
								{s.state === 'error' ? 'unreachable' : 'stale'}: {s.detail}
							</div>
						{/each}
					</div>
					<DotMatrix cols={view.mix} hue={RAMP_1} />
				</div>
			</SlabCard>

			<InsightCard
				i={6}
				badge="Needs you now"
				value={view.attention.count}
				headline={view.attention.headline}
				body={`${view.doneCount} thing${view.doneCount === 1 ? '' : 's'} already done today.`}
				frac={view.attention.frac}
			/>
		</div>

		<!-- Talk to the OS, and what it finished -->
		<div class="mt-6 grid grid-cols-2 items-start gap-6 max-[1100px]:grid-cols-1">
			<section class="bn-rise min-w-0" style="--rise-i: 7"><Interject /></section>
			<section class="bn-rise min-w-0" style="--rise-i: 8"><DoneToday items={view.done} count={view.doneCount} /></section>
		</div>

		{#if errorLines.length}
			<ul class="mt-4 flex flex-col gap-1 font-mono text-[10.5px]" style:color="var(--bn-warn)">
				{#each errorLines as [source, msg] (source)}
					<li>{source} unreadable: {msg}</li>
				{/each}
			</ul>
		{/if}
	{/if}
</Slab>

<style>
	.bn-hover-text:hover {
		color: var(--bn-text);
	}
	@media (min-width: 1201px) {
		.needs-slot > :global([data-part='card']) {
			position: absolute;
			inset: 0;
		}
	}
</style>
