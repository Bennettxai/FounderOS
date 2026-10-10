<!-- The /agents slab hero (FounderOS v1 components/AgentsVolumePanel.tsx): run
     activity + Agent Volume, then runs by seat, the open task lanes and the one
     "Needs you" card. Every number is internal/founderos/pages/agents AgentsVolume
     over the same snapshot the board strip renders, so they never disagree.
     As in prod, an unreachable board reads 0 with its "board unreachable" chip
     and the dead-strip copy; the numbers are the snapshot's, not a guess. -->
<script lang="ts">
	import BigStat from '$lib/founderos/kit/BigStat.svelte';
	import DotMatrix from '$lib/founderos/kit/DotMatrix.svelte';
	import InsightCard from '$lib/founderos/kit/InsightCard.svelte';
	import MeterStack from '$lib/founderos/kit/MeterStack.svelte';
	import SlabCard from '$lib/founderos/kit/SlabCard.svelte';
	import StepLine from '$lib/founderos/kit/StepLine.svelte';
	import type { AgentsVolume } from './types';

	const WINDOW_DAYS = 14;
	let { volume, connected }: { volume: AgentsVolume; connected: boolean } = $props();

	// prod's data-ramp tokens (globals.css), on the port's theme: --ramp-1 is
	// the theme's brain-2, --send-activity brain-2 mixed 70% into the text.
	const RAMP_1 = 'var(--bn-brain-2)';
	const SEND_ACTIVITY = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';

	const inWindow = $derived(volume.window === `${WINDOW_DAYS}d`);
	const busiest = $derived(volume.bySeat[0]);
</script>

<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
	<SlabCard i={2} title="Run Activity" sub={inWindow ? `last ${WINDOW_DAYS} days` : volume.window} class="pb-2">
		<div class="px-6 pb-2 pt-3">
			<BigStat
				size={30}
				value={volume.runsInWindow}
				chips={volume.failedInWindow > 0 ? [{ tone: 'err', text: `${volume.failedInWindow} failed` }] : []}
				caption="heartbeat runs off the live board (its latest 120)"
			/>
		</div>
		<StepLine
			series={volume.series}
			hue={SEND_ACTIVITY}
			unit=" runs"
			empty={connected ? `No heartbeat runs in the last ${WINDOW_DAYS} days.` : 'Board unreachable, no runs to draw.'}
		/>
	</SlabCard>

	<SlabCard i={3} title="Agent Volume" class="flex flex-col">
		<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
			<BigStat value={volume.headline} chips={volume.chips} caption={volume.caption} />
			<MeterStack meters={volume.meters} foot={volume.foot} empty="board unreachable, nothing to measure" />
		</div>
	</SlabCard>
</div>

<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
	<SlabCard i={4} title="Runs by Seat" sub={inWindow ? `${WINDOW_DAYS} days` : volume.window}>
		<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
			<BigStat size={30} display={busiest ? busiest.label : 'none'} caption={busiest ? `busiest seat · ${busiest.count} runs` : `no runs in ${WINDOW_DAYS} days`} />
			<DotMatrix cols={volume.bySeat} hue={RAMP_1} />
		</div>
	</SlabCard>

	<SlabCard i={5} title="Task Lanes" sub="open stages">
		<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
			<BigStat
				size={30}
				value={volume.openTasks}
				caption={volume.openTasks === 1 ? 'open task on the board' : 'open tasks on the board'}
			/>
			<DotMatrix cols={volume.lanes} hue="var(--bn-accent)" />
		</div>
	</SlabCard>

	<InsightCard
		i={6}
		badge="Needs you"
		value={volume.insight.value}
		headline={volume.insight.headline}
		body={volume.insight.body}
		frac={volume.insight.frac}
	/>
</div>
