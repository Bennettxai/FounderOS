<!-- /os/tasks (FounderOS v1 app/tasks/page.tsx): agent work in the Brand Deals
     slab. Every number in the hero and second row is the volume payload
     (internal/founderos/pages/tasks Volume, v1 lib/tasks-volume) over the same
     rows the three sections below render: the cron strip, the live board queue
     and the local kanban. Those sections keep every action and take the stagger
     from i={6}. -->
<script lang="ts">
	import BigStat from '$lib/founderos/kit/BigStat.svelte';
	import Chip from '$lib/founderos/kit/Chip.svelte';
	import DotMatrix from '$lib/founderos/kit/DotMatrix.svelte';
	import InsightCard from '$lib/founderos/kit/InsightCard.svelte';
	import MeterStack from '$lib/founderos/kit/MeterStack.svelte';
	import Slab from '$lib/founderos/kit/Slab.svelte';
	import SlabCard from '$lib/founderos/kit/SlabCard.svelte';
	import SlabTitle from '$lib/founderos/kit/SlabTitle.svelte';
	import StepLine from '$lib/founderos/kit/StepLine.svelte';
	import { PILL } from '$lib/founderos/kit/slab-classes';
	import BoardQueue from './BoardQueue.svelte';
	import TaskCronStrip from './TaskCronStrip.svelte';
	import TaskKanban from './TaskKanban.svelte';
	import { busiestDay, WINDOW_DAYS } from './tasks';
	import type { TasksView } from './types';

	let { view }: { view: TasksView } = $props();

	// v1's data-ramp tokens on the port's theme (same mapping as AgentsVolumePanel)
	const RAMP_1 = 'var(--bn-brain-2)';
	const SEND_ACTIVITY = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';

	const v = $derived(view.volume);
	const boardUrl = $derived(view.board.url);
	const busiest = $derived(busiestDay(v.cron.rhythm));
</script>

<Slab>
	<SlabTitle
		eyebrow="agent work"
		title="Tasks"
		meta={`${v.touchesInWindow} items moved in ${WINDOW_DAYS} days · ${v.cron.runsInWindow} cron runs · ${v.board.blocked} blocked`}
	>
		{#snippet right()}
			<Chip tone={boardUrl ? 'ok' : undefined}>{boardUrl ? 'board live · paperclip' : 'board offline · paperclip'}</Chip>
			<a href="/os/workflows" class={PILL}>Workflows</a>
			<a href="/os/agents" class={PILL}>Agents</a>
		{/snippet}
	</SlabTitle>

	<!-- Hero row, Brand Deals' shape: the work line + the volume card -->
	<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={1} title="Task Activity" sub={`last ${WINDOW_DAYS} days`} class="pb-2">
			<div class="px-6 pb-2 pt-3">
				<BigStat
					size={30}
					value={v.touchesInWindow}
					chips={v.board.blocked > 0 ? [{ tone: 'err', text: `${v.board.blocked} blocked` }] : []}
					caption="kanban tasks and board issues, on the day each last moved"
				/>
			</div>
			<StepLine series={v.series} hue={SEND_ACTIVITY} unit=" moved" empty={`Nothing moved in the last ${WINDOW_DAYS} days.`} />
		</SlabCard>

		<SlabCard i={2} title="Task Volume" class="flex flex-col">
			<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
				<BigStat value={v.headline} chips={v.chips} caption={v.caption} />
				<MeterStack meters={v.meters} foot={v.foot} empty="no tasks yet, hand the company work below" />
			</div>
		</SlabCard>
	</div>

	<!-- Second row: cron rhythm, who holds the work, THE gradient card -->
	<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={3} title="Cron Runs" sub="by weekday">
			<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
				<BigStat
					size={30}
					value={v.cron.runsInWindow}
					chips={v.cron.failedInWindow > 0 ? [{ tone: 'err', text: `${v.cron.failedInWindow} failed` }] : []}
					caption={busiest ? `busiest ${busiest.label} · ${busiest.count} runs` : `no runs in ${WINDOW_DAYS} days`}
				/>
				<DotMatrix cols={v.cron.rhythm} hue={SEND_ACTIVITY} />
			</div>
		</SlabCard>

		<SlabCard i={4} title="Owners" sub="unfinished kanban work">
			<!-- stacked: agent names are long column labels -->
			<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
				<BigStat
					size={30}
					value={v.busiestOwner?.count ?? 0}
					caption={v.busiestOwner ? `most on ${v.busiestOwner.name}` : 'nothing open on the kanban'}
				/>
				{#if v.owners.length > 0}<DotMatrix cols={v.owners} hue={RAMP_1} />{/if}
			</div>
		</SlabCard>

		<InsightCard i={5} badge="Needs you" value={v.insight.value} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac} />
	</div>

	<!-- The three queues: what fires on a timer, the live board, the local kanban -->
	<TaskCronStrip i={6} jobs={view.jobs} />
	<BoardQueue i={7} initialIssues={view.issues} {boardUrl} />
	<TaskKanban i={8} initialTasks={view.tasks} agentNames={view.agentNames} />
</Slab>
