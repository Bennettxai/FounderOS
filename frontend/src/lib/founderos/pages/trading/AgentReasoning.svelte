<!-- The reasoning card body (FounderOS v1 AgentReasoning.tsx): the run's notes,
     verdict chips from the verdicts present in the run, and the per-ticker
     rows. Both regions scroll in their own right inside the pinned panel. -->
<script lang="ts">
	import { ToggleChip } from '$lib/founderos/kit';
	import { sortVerdicts } from './view';
	import type { TradeAnalysis } from './types';

	let { analysis }: { analysis: TradeAnalysis | null } = $props();
	let filter = $state('all');

	const verdicts = $derived(analysis ? sortVerdicts(analysis.rows.map((r) => r.verdict)) : []);
	const rows = $derived(analysis ? analysis.rows.filter((r) => filter === 'all' || r.verdict === filter) : []);
	const tone = (v: string) => (v === 'signal' ? 'var(--bn-ok)' : v === 'dropped' ? 'var(--bn-text-3)' : 'var(--bn-text-2)');
</script>

{#if !analysis}
	<div class="bn-dim px-6 py-10 text-center text-[12.5px] leading-relaxed">No analysis pushed yet. The agent writes here every run, whether or not it trades.</div>
{:else}
	<div class="mt-3 flex min-h-0 flex-1 flex-col">
		{#if analysis.notes}
			<p class="bn-muted mx-6 max-h-[132px] shrink-0 overflow-y-auto rounded-[10px] border px-3.5 py-3 text-[11.5px] leading-relaxed" style="border-color: var(--bn-border); background: var(--bn-bg)">
				{analysis.notes}
			</p>
		{/if}
		{#if verdicts.length > 0}
			<div class="flex shrink-0 flex-wrap gap-1.5 px-6 py-3">
				<ToggleChip on={filter === 'all'} onclick={() => (filter = 'all')}>all</ToggleChip>
				{#each verdicts as v (v)}
					<ToggleChip on={filter === v} onclick={() => (filter = v)}>{v}</ToggleChip>
				{/each}
			</div>
		{/if}
		<ul data-part="reasoning-list" class="max-h-[260px] min-h-[120px] flex-1 overflow-y-auto border-t" style="border-color: var(--bn-border)">
			{#each rows as r (r.ticker)}
				<li class="bn-enter border-b px-6 py-2.5 last:border-b-0" style="border-color: var(--bn-hairline)">
					<div class="flex min-w-0 items-baseline gap-2">
						<span class="font-mono text-[12.5px] font-bold">{r.ticker}</span>
						<span class="bn-dim font-mono text-[10px] tabular-nums">{r.score === null ? 'unscored' : `score ${r.score}`}</span>
						<span class="flex-1"></span>
						<span class="shrink-0 font-mono text-[9px] uppercase tracking-wider" style="color: {tone(r.verdict)}">{r.verdict}</span>
					</div>
					{#if r.reason}<div class="bn-dim mt-0.5 line-clamp-3 text-[11.5px] leading-snug">{r.reason}</div>{/if}
				</li>
			{:else}
				<li class="bn-dim px-6 py-6 text-center font-mono text-[11px]">
					{analysis.rows.length === 0 ? 'The run recorded no per-ticker detail.' : `No ${filter} rows in this run.`}
				</li>
			{/each}
		</ul>
	</div>
{/if}
