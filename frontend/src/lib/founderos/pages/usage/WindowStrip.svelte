<!-- Burn for the last hour, the 5h session, today and the week (UsageBoard.tsx WindowStrip). -->
<script lang="ts">
	import { WINDOWS, WINDOW_LABEL, fmtTokens, pct, windowTable, type PlanUsage } from './usage';

	let { plan }: { plan: PlanUsage } = $props();

	const rows = $derived(plan.breakdown ? windowTable(plan.breakdown, plan.official?.weekly?.usedPercent ?? null) : null);
</script>

<div class="grid grid-cols-4 gap-2">
	{#each WINDOWS as w (w)}
		{@const r = rows?.find((x) => x.window === w)}
		<div class="bn-border border-l pl-2">
			<div class="bn-text text-[19px] font-semibold tabular-nums tracking-[-0.02em]">{r ? fmtTokens(r.burn) : '—'}</div>
			<div class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.16em]">{WINDOW_LABEL[w]}</div>
			<div class="bn-muted font-mono text-[9.5px]">
				{#if r && r.pctOfWeekly !== null}≈{pct(r.pctOfWeekly)} of wk{w === 'week' ? '' : ' est.'}{:else if r}{pct(r.shareOfWeek * 100)} of wk burn{:else}no lanes{/if}
			</div>
		</div>
	{/each}
</div>
