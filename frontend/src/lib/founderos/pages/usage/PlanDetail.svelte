<!-- Where a plan's tokens went (UsageBoard.tsx PlanDetail): lanes per window, top burners, days, models, machines. -->
<script lang="ts">
	import { Label, VolumeMeter } from '$lib/founderos/kit';
	import LaneBar from './LaneBar.svelte';
	import Machines from './Machines.svelte';
	import {
		ACTIVITY_HUE,
		BURN_SOURCES,
		LANE_HUE,
		SOURCE_LABEL,
		WINDOWS,
		WINDOW_LABEL,
		burnOf,
		fmtTokens,
		laneBurn,
		pct,
		type PlanUsage,
		type UsageWindow
	} from './usage';
	import { DETAIL, TRACK, chipClass } from './ui';

	let { plan, now }: { plan: PlanUsage; now: number } = $props();

	let win = $state<UsageWindow>('day');
	const b = $derived(plan.breakdown ?? null);
	const lanes = $derived(b?.windows[win]);
	const total = $derived(lanes ? laneBurn(lanes) : 0);
	const weekly = $derived(plan.official?.weekly?.usedPercent ?? null);
	const weekTotal = $derived(b ? laneBurn(b.windows.week) : 0);
	const topMax = $derived(Math.max(1, ...(b?.top.map((t) => t.burn) ?? [1])));
	const models = $derived(Object.entries(plan.byModel).sort((a, c) => burnOf(c[1]) - burnOf(a[1])));
	const modelTotal = $derived(models.reduce((s, [, t]) => s + burnOf(t), 0));
	const dayMax = $derived(Math.max(1, ...plan.days.map((d) => burnOf(d))));
	const reRead = $derived(plan.days.reduce((s, d) => s + d.cacheRead, 0));
	const share = $derived(
		weekly !== null && weekTotal > 0
			? `≈${pct((weekly * total) / weekTotal)} of the weekly limit${win === 'week' ? '' : ' (est.)'}`
			: weekTotal > 0
				? `${pct((total / weekTotal) * 100)} of this week's burn`
				: 'no burn'
	);
</script>

<div data-part="plan-detail" class={DETAIL}>
	<div class="space-y-5">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<Label>where it burns</Label>
			<div class="flex flex-wrap gap-2" role="tablist">
				{#each WINDOWS as w (w)}
					<button type="button" role="tab" aria-selected={win === w} onclick={() => (win = w)} class={chipClass(win === w)}>{WINDOW_LABEL[w]}</button>
				{/each}
			</div>
		</div>

		{#if !b || !lanes}
			<p class="bn-dim text-[12.5px]">this reading carries no lane breakdown yet (an older push)</p>
		{:else}
			<div>
				<div class="flex flex-wrap items-baseline justify-between gap-2">
					<span class="bn-text text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{fmtTokens(total)}</span>
					<span class="bn-muted font-mono text-[11px]">{share}</span>
				</div>
				<div class="mt-3"><LaneBar {plan} window={win} height={10} /></div>
				<div class="mt-3 flex flex-wrap gap-x-3 gap-y-1">
					{#each BURN_SOURCES as s (s)}
						<span class="bn-muted flex items-center gap-1.5 text-[11.5px]">
							<span data-part="legend-dot" class="inline-block h-2 w-2 rounded-full" style="background: {LANE_HUE[s]}"></span>{SOURCE_LABEL[s]}
						</span>
					{/each}
				</div>
			</div>

			<div class="bn-border space-y-5 border-t pt-5">
				{#each BURN_SOURCES as s, i (`${win}-${s}`)}
					{@const v = burnOf(lanes[s])}
					{@const sh = total > 0 ? v / total : 0}
					<VolumeMeter label={SOURCE_LABEL[s]} frac={sh} display={`${fmtTokens(v)} · ${pct(sh * 100)}`} hue={LANE_HUE[s]} delay={i * 120} />
				{/each}
			</div>

			<div>
				<Label>top burners · 7 days</Label>
				<div class="mt-3 space-y-2">
					{#if b.top.length === 0}<p class="bn-dim text-[12.5px]">nothing burned this week</p>{/if}
					{#each b.top.slice(0, 12) as t (`${t.source}|${t.label}`)}
						<div class="grid grid-cols-[1fr_120px_60px] items-center gap-3 text-[12.5px]">
							<span class="truncate" title={t.label}>
								<span class="bn-text">{t.label}</span>
								<span class="bn-dim ml-2 font-mono text-[9.5px] uppercase tracking-[0.14em]">{SOURCE_LABEL[t.source]}</span>
							</span>
							<div class="h-1.5 overflow-hidden rounded-full" style={TRACK}>
								<div class="h-full rounded-full" style="width: {(t.burn / topMax) * 100}%; background: {LANE_HUE[t.source]}"></div>
							</div>
							<span class="text-right font-mono text-[11px] tabular-nums">{fmtTokens(t.burn)}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<div class="space-y-5">
		<div>
			<Label>7-day burn · by day</Label>
			<div class="mt-3 flex h-20 items-end gap-1.5">
				{#each plan.days as d, i (d.day)}
					{@const v = burnOf(d)}
					{@const h = v === 0 ? 2 : Math.max(3, Math.round((v / dayMax) * 72))}
					<div class="flex flex-1 flex-col items-center gap-1" title="{d.day}: {fmtTokens(v)} burn · {fmtTokens(d.cacheRead)} context re-read">
						<div
							class="w-full rounded-t-[4px]"
							style="height: {h}px; background: {i === plan.days.length - 1 ? ACTIVITY_HUE : `color-mix(in oklab, ${ACTIVITY_HUE} 35%, transparent)`}"
						></div>
						<span class="bn-dim font-mono text-[9px]">{'SMTWTFS'[new Date(d.day + 'T12:00:00').getDay()]}</span>
					</div>
				{/each}
			</div>
			<p class="bn-dim mt-1.5 font-mono text-[10px]">context re-read (cache reads, not burn): {fmtTokens(reRead)}</p>
		</div>

		{#if models.length > 0}
			<div>
				<Label>models</Label>
				<div class="mt-2 space-y-1.5">
					{#each models.slice(0, 5) as [m, t] (m)}
						<div class="flex items-center justify-between text-[12.5px]">
							<span class="bn-muted">{m.replace(/^claude-/, '')}</span>
							<span class="font-mono text-[11px] tabular-nums"
								>{fmtTokens(burnOf(t))}<span class="bn-dim">{` · ${modelTotal ? Math.round((burnOf(t) / modelTotal) * 100) : 0}%`}</span></span
							>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<Machines machines={plan.machines} {now} />
		<p class="bn-dim font-mono text-[10px] leading-relaxed">
			a machine joins this plan when its founderos-collector pushes its seat (POST /api/founderos/device/push) or push-usage posts to /api/usage/push
		</p>
		{#if plan.note}<p class="bn-dim font-mono text-[10px] leading-relaxed">{plan.note}</p>{/if}
	</div>
</div>
