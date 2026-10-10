<!-- The live Paperclip board rendered natively (FounderOS v1 components/BoardLive.tsx):
     seat chips with models and Run buttons, the stats row, the heartbeat run feed
     and the task lanes. The parent polls /pages/board/live every 4s and hands the
     snapshot in. An unreachable board is an honest dead strip, never fake green. -->
<script lang="ts">
	import { ArrowUpRight } from '$lib/founderos/icons';
	import CountUp from '$lib/founderos/kit/CountUp.svelte';
	import Label from '$lib/founderos/kit/Label.svelte';
	import BoardTaskCard from './BoardTaskCard.svelte';
	import SeatChip from './SeatChip.svelte';
	import { ago, boardDecisionFor, groupIssues, LANE_DOT, modelSummary, orderRuns, orderSeats, RUN_OK, runDuration, runGlyph, runningSince, runTone } from './board';
	import type { BoardLive, BoardStats } from './types';

	let { board, stats, boardUrl = null }: { board: Omit<BoardLive, 'volume' | 'stats'>; stats: BoardStats; boardUrl?: string | null } = $props();

	const running = $derived(board.agents.filter((a) => a.status === 'running').length);
	const nameById = $derived(new Map(board.agents.map((a) => [a.id, a.name])));
	const runs = $derived(orderRuns(board.runs).slice(0, 120));
	const lanes = $derived(groupIssues(board.issues));
	const runningNames = $derived(new Set(board.agents.filter((a) => a.status === 'running').map((a) => a.name)));
	const statRow = $derived([
		['Seats', stats.seats],
		['Running now', stats.running],
		['Open tasks', stats.openTasks],
		['Runs · 24h', stats.runs24h],
		['Heartbeats · 24h', stats.heartbeats24h]
	] as const);
	const toneColor = (t: string) => (t === 'muted' ? 'var(--bn-text-2)' : `var(--bn-${t})`);
</script>

<section data-part="board-live" class="bn-board flex h-full min-h-0 flex-col border">
	<div class="bn-board-head shrink-0 border-b px-4 py-2.5">
		<div class="flex flex-wrap items-center gap-x-3 gap-y-1">
			<!-- prod's square LED: green and pulsing while the board answers, red when not -->
			<span data-part="led" class="h-1.5 w-1.5 {board.connected ? 'animate-pulse' : ''}" style:background={board.connected ? 'var(--bn-ok)' : 'var(--bn-err)'}></span>
			<Label>Board live</Label>
			<span class="bn-dim font-mono text-[10px]">{board.connected ? `${board.agents.length} seats · ${running} running` : 'board unreachable'}</span>
			{#if boardUrl}
				<a href={boardUrl} target="_blank" rel="noreferrer" class="bn-dim bn-linky ml-auto flex shrink-0 items-center gap-1 font-mono text-[10px]">
					open board <ArrowUpRight size={12} />
				</a>
			{/if}
		</div>
		{#if board.connected}
			<p class="bn-muted mt-1 font-mono text-[10px] leading-relaxed" title="models holding seats right now">{modelSummary(board.agents)}</p>
			{#if board.error}<p class="mt-1 font-mono text-[10px]" style:color="var(--bn-warn)">partial read: {board.error}</p>{/if}
		{/if}
	</div>

	{#if !board.connected}
		<p class="bn-dim px-4 py-3 font-mono text-[10.5px]" title={board.error ?? undefined}>
			Paperclip is not answering on the tailnet. No fake data: this strip lights up the moment the board responds.
		</p>
	{:else}
		<div class="flex min-h-0 flex-1 flex-col gap-4 p-4">
			<div class="grid shrink-0 grid-cols-5 gap-2 max-[900px]:grid-cols-2">
				{#each statRow as [label, value] (label)}
					<div data-lens="r" class="bn-pressable is-row bn-cell flex min-w-0 flex-col gap-1 border px-2.5 py-2">
						<div class="bn-dim min-h-[1.6em] font-mono text-[9.5px] font-bold uppercase leading-[1.3] tracking-[0.16em]">{label}</div>
						<div class="bn-text font-mono text-[22px] font-semibold leading-none tracking-[-0.02em]"><CountUp {value} /></div>
					</div>
				{/each}
			</div>

			<div class="grid shrink-0 grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
				{#each orderSeats(board.agents) as a (a.id)}
					<SeatChip agent={a} since={runningSince(board.runs, a.id)} />
				{/each}
			</div>

			<div class="grid min-h-0 flex-1 gap-4 lg:grid-cols-[minmax(0,12rem)_minmax(0,1fr)]">
				<div class="flex min-h-0 flex-col">
					<div class="mb-2 shrink-0"><Label>Run feed</Label></div>
					<div class="min-h-0 flex-1 space-y-1 overflow-y-auto overscroll-contain pr-1">
						{#if runs.length === 0}
							<p class="bn-dim font-mono text-[10px]">no heartbeat runs yet, hit a ▸ on a seat</p>
						{/if}
						{#each runs as r (r.id)}
							<div data-lens="r" class="bn-pressable is-row bn-enter bn-ctl flex items-baseline gap-1.5 px-1 py-0.5 font-mono text-[9.5px] leading-snug">
								<span class="font-bold" style:color={toneColor(runTone(r.status))}>{runGlyph(r.status)}</span>
								<span class="bn-text min-w-0 flex-1 truncate">{r.agentName ?? nameById.get(r.agentId) ?? r.agentId.slice(0, 8)}</span>
								{#if !RUN_OK.has(r.status)}<span style:color={toneColor(runTone(r.status))}>{r.status}</span>{/if}
								{#if runDuration(r)}<span class="bn-dim">{runDuration(r)}</span>{/if}
								<span class="bn-dim shrink-0">{ago(r.startedAt)}</span>
							</div>
						{/each}
					</div>
				</div>

				<div class="flex min-h-0 flex-col">
					<div class="mb-2 shrink-0"><Label>Task lanes</Label></div>
					<div class="flex min-h-0 flex-1 gap-3 overflow-x-auto pb-1">
						{#each lanes as lane (lane.status)}
							<div data-part="lane" class="flex min-w-[5.5rem] flex-1 basis-0 flex-col">
								<!-- prod: text-os-dim/60 on an empty lane, an opacity on a var()
								     colour Tailwind v3 never generates, so the head reads in the
								     body text colour; the port draws what prod shows -->
								<div class="mb-1.5 flex shrink-0 items-center gap-1 font-mono text-[9px] uppercase tracking-[0.08em] {lane.issues.length === 0 ? 'bn-text' : 'bn-dim'}">
									<span class="h-1 w-1 shrink-0" style:background={LANE_DOT[lane.status] ?? 'var(--bn-text-3)'}></span>
									<span class="truncate">{lane.status.replace(/_/g, ' ')}</span>
									<span class="ml-auto shrink-0 tabular-nums">{lane.issues.length}</span>
								</div>
								<div class="min-h-0 flex-1 space-y-1.5 overflow-y-auto overscroll-contain pr-1">
									{#if lane.issues.length === 0}
										<div class="bn-text bn-enter bn-ctl border border-dashed px-2 py-1.5 font-mono text-[9.5px]" style:border-color="var(--bn-border)">nothing here</div>
									{/if}
									{#each lane.issues as i (i.id)}
										<BoardTaskCard issue={i} working={!!i.assigneeName && runningNames.has(i.assigneeName)} decision={boardDecisionFor(i, board.decisions)} />
									{/each}
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}
</section>

<style>
	.bn-board {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-panel, 10px);
		background: var(--bn-surface);
	}
	.bn-ctl {
		border-radius: var(--bn-r-ctl, 6px);
	}
	.bn-board-head {
		border-color: var(--bn-border);
	}
	.bn-cell {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-ctl, 6px);
		background: var(--bn-bg);
	}
</style>
