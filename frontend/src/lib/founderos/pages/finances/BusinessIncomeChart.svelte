<!-- Per-business bank-statement money (FounderOS v1 BusinessIncomeChart.tsx):
     money in / out / net with range chips, a total over the window plus
     monthly bars. Statements are monthly, so 30d ≈ one month. -->
<script lang="ts">
	import { ToggleChip } from '$lib/founderos/kit';
	import './fin.css';
	import type { BusinessSeries, IncomeRange, MonthPoint } from './spend-report';
	import { usd } from './spend-report';

	type Metric = 'in' | 'out' | 'net';
	const RANGES: { label: string; value: IncomeRange }[] = [
		{ label: '3 mo', value: 3 },
		{ label: '6 mo', value: 6 },
		{ label: 'All', value: 'all' }
	];
	const METRICS: { label: string; value: Metric }[] = [
		{ label: 'Money in', value: 'in' },
		{ label: 'Money out', value: 'out' },
		{ label: 'Net after out', value: 'net' }
	];

	let { series }: { series: BusinessSeries } = $props();

	let range = $state<IncomeRange>('all');
	let metric = $state<Metric>('in');
	let hovered = $state<string | null>(null);

	const cents = (m: MonthPoint, k: Metric) => (k === 'in' ? m.creditsCents : k === 'out' ? m.debitsCents : m.netCents);
	const signed = (c: number) => `${c < 0 ? '−' : '+'}${usd(Math.abs(c))}`;
	const fmtMonth = (m: string) => new Date(`${m}-01T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', timeZone: 'UTC' });

	const shown = $derived(range === 'all' ? series.months : series.months.slice(-range));
	const totalIn = $derived(shown.reduce((s, m) => s + m.creditsCents, 0));
	const totalOut = $derived(shown.reduce((s, m) => s + m.debitsCents, 0));
	const net = $derived(shown.reduce((s, m) => s + m.netCents, 0));
	const total = $derived(metric === 'in' ? totalIn : metric === 'out' ? totalOut : net);
	const max = $derived(Math.max(1, ...shown.map((m) => Math.abs(cents(m, metric)))));
	const headTone = $derived(metric === 'in' ? 'ok' : metric === 'out' ? 'err' : net >= 0 ? 'ok' : 'err');
</script>

<div class="fin-panel p-5">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<div class="bn-text text-sm font-semibold">{series.business}</div>
			<div class="bn-dim mt-0.5 font-mono text-[10px] uppercase tracking-[0.14em]">
				{metric === 'in' ? 'income · bank deposits' : metric === 'out' ? 'outflow · bank debits' : 'net · after money out'}
			</div>
		</div>
		<div class="flex shrink-0 flex-col items-end gap-1.5">
			<div class="flex items-center gap-1">
				{#each METRICS as m (m.value)}
					<ToggleChip on={metric === m.value} onclick={() => (metric = m.value)}>{m.label}</ToggleChip>
				{/each}
			</div>
			<div class="flex items-center gap-1">
				{#each RANGES as r (r.label)}
					<ToggleChip on={range === r.value} onclick={() => (range = r.value)}>{r.label}</ToggleChip>
				{/each}
			</div>
		</div>
	</div>

	<div class="mt-3 flex items-baseline gap-2">
		<span class="font-mono text-[26px] font-semibold tracking-[-0.02em]" style="color: var(--bn-{headTone})">
			{metric === 'net' && net < 0 ? `−${usd(Math.abs(net))}` : usd(total)}
		</span>
		{#if metric === 'in'}
			<span class="font-mono text-[11px]" style="color: var(--bn-{net >= 0 ? 'ok' : 'err'})">{signed(net)} net</span>
		{:else if metric === 'out'}
			<span class="bn-dim font-mono text-[11px]">of {usd(totalIn)} in</span>
		{:else}
			<span class="bn-dim font-mono text-[11px]">{usd(totalIn)} in − {usd(totalOut)} out</span>
		{/if}
	</div>

	{#if shown.length === 0}
		<div class="bn-dim mt-4 font-mono text-[11px]">no statements in range</div>
	{:else}
		<div class="mt-4 flex items-end gap-1.5" style="height: 104px">
			{#each shown as m (m.month)}
				{@const c = cents(m, metric)}
				{@const h = Math.max(2, (Math.abs(c) / max) * 76)}
				{@const hue = metric === 'in' ? 'var(--bn-accent)' : metric === 'out' ? 'var(--bn-err)' : c >= 0 ? 'var(--bn-ok)' : 'var(--bn-err)'}
				<div class="flex flex-1 flex-col items-center gap-1" role="presentation" onmouseenter={() => (hovered = m.month)} onmouseleave={() => (hovered = null)}>
					<div class="relative flex w-full items-end justify-center" style="height: 84px">
						{#if hovered === m.month}
							<span class="fin-tip bn-text pointer-events-none absolute left-1/2 z-10 -translate-x-1/2 whitespace-nowrap px-1.5 py-0.5 font-mono text-[10px] font-semibold" style="bottom: {h + 6}px">
								{metric === 'net' ? signed(c) : usd(c)}
							</span>
						{/if}
						<div class="w-full max-w-[28px] rounded-[5px]" style="height: {h}px; background: {hue}; opacity: {hovered === m.month ? 1 : hovered ? 0.45 : 0.8}"></div>
					</div>
					<span class="state-fade font-mono text-[9px] {hovered === m.month ? 'bn-text' : 'bn-dim'}">{fmtMonth(m.month)}</span>
				</div>
			{/each}
		</div>
	{/if}
</div>
