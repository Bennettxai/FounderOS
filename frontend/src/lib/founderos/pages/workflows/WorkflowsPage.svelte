<!-- /os/workflows body (FounderOS v1 app/workflows/page.tsx): the Brand Deals
     slab from the Go workflows volume (Run Activity + Cron Volume, then Run
     Rhythm, Process Load and the Needs-you card), then the clock half
     (scheduled tasks) and the process map. Every number comes from the view
     payload. -->
<script lang="ts">
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import ScheduledTasks from './ScheduledTasks.svelte';
	import WorkflowTree from './WorkflowTree.svelte';
	import type { WorkflowsView } from './types';

	const WINDOW_DAYS = 14;
	/** v1's globals.css --send-activity and --ramp-1, over the port's brain tokens. */
	const SEND_ACTIVITY = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';
	const RAMP_1 = 'var(--bn-brain-2)';

	let { view, onchange }: { view: WorkflowsView; onchange?: () => void } = $props();

	const v = $derived(view.volume);
	const busiest = $derived(v.rhythm.reduce((best, d) => (d.count > best.count ? d : best), v.rhythm[0] ?? { label: '', count: 0 }));
	const wholeHours = $derived(Number.isInteger(v.load.manualHours));
</script>

<Slab>
	<SlabTitle
		eyebrow="scheduled tasks + process map"
		title="Workflows"
		meta={`${view.workflows.length} workflows · ${v.runsInWindow} cron runs in ${WINDOW_DAYS} days · ${v.failedInWindow} failed`}
	>
		{#snippet right()}
			<Chip tone={v.counts.healthy === view.jobs.length ? 'ok' : 'warn'}>{view.jobs.length} crons · {v.counts.healthy} healthy</Chip>
			<a href="/os/tasks" class={PILL}>Tasks</a>
			<a href="/os/agents" class={PILL}>Agents</a>
		{/snippet}
	</SlabTitle>

	<!-- Hero row, Brand Deals' shape: the run line + the volume card -->
	<div data-part="hero-row" class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={1} title="Run Activity" sub={`last ${WINDOW_DAYS} days`} class="pb-2">
			<div class="px-6 pb-2 pt-3">
				<BigStat
					size={30}
					value={v.runsInWindow}
					chips={v.failedInWindow > 0 ? [{ tone: 'err', text: `${v.failedInWindow} failed` }] : []}
					caption="real cron runs, straight off cron_runs"
				/>
			</div>
			<StepLine series={v.series} hue={SEND_ACTIVITY} unit=" runs" empty={`No cron runs in the last ${WINDOW_DAYS} days.`} />
		</SlabCard>

		<SlabCard i={2} title="Cron Volume" class="flex flex-col">
			<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
				<BigStat value={v.headline} chips={v.chips} caption={v.caption} />
				<MeterStack meters={v.meters} foot={v.foot} />
			</div>
		</SlabCard>
	</div>

	<!-- Second row: weekday rhythm, process load, THE gradient card -->
	<div data-part="second-row" class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={3} title="Run Rhythm" sub="by weekday">
			<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
				<BigStat
					size={30}
					display={busiest.count > 0 ? busiest.label : 'none'}
					caption={busiest.count > 0 ? `busiest day · ${busiest.count} runs` : `no runs in ${WINDOW_DAYS} days`}
				/>
				<DotMatrix cols={v.rhythm} hue={SEND_ACTIVITY} />
			</div>
		</SlabCard>

		<SlabCard i={4} title="Process Load" sub="hours per week">
			<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
				<BigStat
					size={30}
					value={wholeHours ? v.load.manualHours : undefined}
					display={wholeHours ? undefined : String(v.load.manualHours)}
					unit="h"
					caption={`by hand · ${v.load.agentHours}h carried by agents`}
				/>
				<DotMatrix cols={v.load.perWorkflow} hue={RAMP_1} />
			</div>
		</SlabCard>

		<InsightCard i={5} badge="Needs you" value={v.insight.value} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac} />
	</div>

	<!-- The clock half: add, run, pause and delete crons in place -->
	<SlabCard i={6} class="mt-6 px-6 pb-1 pt-5">
		<ScheduledTasks jobs={view.jobs} agents={view.agents} {onchange} />
	</SlabCard>

	<!-- The process-map half: the tree, its step drawer and the builder -->
	<SlabCard i={7} title="Process map" sub={`${view.workflows.length} workflows`} class="mt-6">
		<div class="px-6 pb-6 pt-4">
			<WorkflowTree workflows={view.workflows} agentPresence={view.agentPresence} agents={view.agents} runsByOwner={view.runsByOwner} {onchange} />
		</div>
	</SlabCard>
</Slab>
