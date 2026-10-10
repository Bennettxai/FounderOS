<!-- The Markets Agent's limits, as the agent will run under them (FounderOS v1
     TradingLimits.tsx): a number box per limit, the autopilot switch and SAVE.
     This backend has no limits write route yet, so all of it is held
     read-only, and the autopilot switch (which arms real orders) stays in its
     stored state, off by default. The stored row is shown clamped, as the
     agent reads it. -->
<script lang="ts">
	import { SlidersHorizontal, TriangleAlert } from '$lib/founderos/icons';
	import type { TradingLimits } from './types';
	import './trading.css';

	let { view }: { view: { limits: TradingLimits; clamped: string[]; source: 'stored' | 'default'; updatedAt: string | null } } = $props();

	const FIELDS: { key: keyof Omit<TradingLimits, 'autopilot'>; label: string; unit: '$' | '%' | ''; hint: string }[] = [
		{ key: 'maxDeployedCapitalUsd', label: 'Autonomy budget', unit: '$', hint: 'most capital deployed at once without a human' },
		{ key: 'maxNotionalPerTradeUsd', label: 'Max per trade', unit: '$', hint: 'largest single position the agent may open' },
		{ key: 'maxPositionPctOfSleeve', label: 'Max position', unit: '%', hint: 'of sleeve equity in any one name' },
		{ key: 'maxRiskPctPerTrade', label: 'Risk per trade', unit: '%', hint: 'of equity lost if the stop is hit' },
		{ key: 'maxConcurrentPositions', label: 'Max positions', unit: '', hint: 'concurrent open names' },
		{ key: 'maxTradesPerDay', label: 'Trades per day', unit: '', hint: 'throttle, not a strategy rule' },
		{ key: 'minSleeveValueUsd', label: 'Kill-switch floor', unit: '$', hint: 'buys stop if the sleeve falls through this' }
	];
	const HELD = 'read-only: this backend does not edit the agent limits yet';
</script>

<section data-part="limits" class="bn-rise overflow-hidden rounded-[8px] border" style="--rise-i: 9; border-color: var(--bn-border); background: var(--bn-surface)">
	<header class="flex items-center justify-between border-b px-4 py-3" style="border-color: var(--bn-border)">
		<div class="flex items-center gap-2">
			<SlidersHorizontal size={13} class="bn-dim" />
			<span class="bn-text font-mono text-[10px] font-bold uppercase tracking-[0.26em]">Agent limits</span>
		</div>
		<span class="bn-dim font-mono text-[9px] uppercase tracking-[0.2em]">{view.source === 'stored' ? 'stored in OS' : 'code defaults'}</span>
	</header>
	<div class="grid gap-px sm:grid-cols-2" style="background: var(--bn-border)">
		{#each FIELDS as f (f.key)}
			<label class="flex items-center gap-3 px-4 py-3" style="background: var(--bn-surface)">
				<span class="min-w-0 flex-1">
					<span class="bn-muted block font-mono text-[10px] uppercase tracking-[0.18em]">{f.label}</span>
					<span class="bn-dim block truncate font-mono text-[9px]">{f.hint}</span>
				</span>
				<span class="flex items-center gap-1">
					{#if f.unit === '$'}<span class="bn-dim font-mono text-[11px]">$</span>{/if}
					<input
						type="number"
						readonly
						aria-readonly="true"
						title={HELD}
						value={view.limits[f.key]}
						class="tb-limit bn-text w-20 rounded-[6px] border px-2 py-1 text-right font-mono text-[12px]"
					/>
					{#if f.unit === '%'}<span class="bn-dim font-mono text-[11px]">%</span>{/if}
				</span>
			</label>
		{/each}
	</div>

	<!-- The one switch that arms real orders: shown in its stored state, never flippable here. -->
	<div class="flex flex-wrap items-center justify-between gap-3 border-t px-4 py-3" style="border-color: var(--bn-border)">
		<div class="min-w-0">
			<div class="bn-text font-mono text-[10px] uppercase tracking-[0.18em]">Autopilot</div>
			<div class="bn-dim font-mono text-[9.5px] leading-relaxed">
				{view.limits.autopilot
					? 'ON: the daily strategy run places real orders in the agentic sleeve, inside every limit above.'
					: 'OFF: the daily run proposes and logs every order as a dry run; nothing reaches the broker.'}
			</div>
		</div>
		<button
			type="button"
			role="switch"
			aria-checked={view.limits.autopilot}
			aria-label="Autopilot"
			title={HELD}
			disabled
			class="relative h-6 w-11 shrink-0 cursor-not-allowed rounded-full border"
			style={view.limits.autopilot
				? 'border-color: var(--bn-ok); background: color-mix(in oklab, var(--bn-ok) 25%, transparent)'
				: 'border-color: var(--bn-border); background: var(--bn-bg)'}
		>
			<span
				class="absolute top-[3px] h-4 w-4 rounded-full"
				style="left: {view.limits.autopilot ? '24px' : '3px'}; background: {view.limits.autopilot ? 'var(--bn-ok)' : 'var(--bn-text-3)'}"
			></span>
		</button>
	</div>

	<footer class="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-3" style="border-color: var(--bn-border)">
		<div class="font-mono text-[10px]">
			{#if view.clamped.length > 0}
				<span class="flex items-center gap-1" style="color: var(--bn-warn)"><TriangleAlert size={11} /> held at the code ceiling: {view.clamped.join(', ')}</span>
			{/if}
		</div>
		<button type="button" disabled title={HELD} class="bn-dim rounded-[6px] border px-3 py-1 font-mono text-[10px] uppercase tracking-[0.2em]" style="border-color: var(--bn-border)">save</button>
	</footer>
</section>
