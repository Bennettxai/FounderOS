<!-- Where it goes (FounderOS v1 MonthlyExpenses.tsx): the pie is per month,
     not a frozen latest-month snapshot. Stepping months redraws it from the
     ledger rows the page handed down, and the full expenditure statement
     opens over the page. With no statements uploaded it falls back to the
     declared set fees, honestly labelled. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { untrack } from 'svelte';
	import { ChevronLeft, ChevronRight, Maximize2 } from '$lib/founderos/icons';
	import { VolumeMeter, chipClass } from '$lib/founderos/kit';
	import './fin.css';
	import ExpenditureReport from './ExpenditureReport.svelte';
	import SharePie from './SharePie.svelte';
	import { cardLabel, cardTotals, categoryTotals, monthAfterRefresh, monthName, RAMP_4, spendTotalCents, usd, type SpendRow } from './spend-report';

	let {
		rows,
		months,
		fallback,
		focus = null,
		children
	}: {
		rows: SpendRow[];
		/** months with card spend, ascending */
		months: string[];
		/** declared set fees in cents, used only when nothing has been uploaded */
		fallback: { category: string; totalCents: number }[];
		/** the newest month an upload just covered: the view moves there */
		focus?: string | null;
		/** the statement uploader, rendered under the chart */
		children?: Snippet;
	} = $props();
	const live = $derived(rows.length > 0);
	// svelte-ignore state_referenced_locally
	let month = $state<string | null>(monthAfterRefresh(null, [], months));
	// svelte-ignore state_referenced_locally
	let seen: string[] = months;
	let open = $state(false);

	// The ledger refreshed under us (an upload): land on a month that arrived,
	// else hold the one being read.
	$effect.pre(() => {
		const ms = months;
		if (seen.length !== ms.length || seen.some((m, i) => m !== ms[i])) {
			month = monthAfterRefresh(untrack(() => month), seen, ms);
			seen = ms;
		}
	});
	// An upload names the month it covered, even one already on file.
	$effect.pre(() => {
		if (focus && months.includes(focus)) month = focus;
	});

	const index = $derived(month ? months.indexOf(month) : -1);
	function step(delta: number) {
		const next = months[index + delta];
		if (next) month = next;
	}

	const cats = $derived(live ? categoryTotals(rows, month) : fallback);
	const lanes = $derived(cardTotals(rows, month));
	// Last six months, plus the selected one when a back-dated statement lands
	// outside that window: the chosen chip is always on screen.
	const chips = $derived.by(() => {
		const tail = months.slice(-6);
		return month && !tail.includes(month) ? [month, ...tail] : tail;
	});
	const total = $derived(live ? spendTotalCents(rows, month) : fallback.reduce((s, c) => s + c.totalCents, 0));
	const period = $derived(live ? (month ? monthName(month) : 'all time') : 'per month');
</script>

<section class="flex h-full flex-col">
	<!-- the slab card head: 19px title, mono total, the way in to the full statement -->
	<div class="flex flex-wrap items-center justify-between gap-3 px-6 pt-5">
		<div class="flex min-w-0 items-baseline gap-2">
			<h2 class="bn-text text-[19px] font-semibold tracking-[-0.01em]">Where it goes</h2>
			<span data-testid="expenses-total" class="bn-dim font-mono text-[12px] tabular-nums">{live ? `${usd(total)} · ${period}` : `${usd(total)} /mo`}</span>
		</div>
		<button type="button" data-lens="c" class="bn-pressable bn-pill inline-flex items-center gap-1.5 rounded-full border px-4 py-2 text-[13px] disabled:opacity-40" onclick={() => (open = true)} disabled={!live}>
			<Maximize2 size={12} strokeWidth={1.8} />
			View full expenditure statement
		</button>
	</div>

	<!-- month switcher: filter pills, the chosen month solid -->
	<div class="mt-4 flex flex-wrap items-center gap-2 px-6">
		<button type="button" class="fin-step bn-pressable rounded-[6px] border p-1.5" onclick={() => step(-1)} disabled={!live || index <= 0} aria-label="Previous month">
			<ChevronLeft size={14} strokeWidth={1.8} />
		</button>
		{#each chips as m (m)}
			<button type="button" data-lens="c" class={chipClass(m === month)} onclick={() => (month = m)}>{monthName(m)}</button>
		{/each}
		<button type="button" class="fin-step bn-pressable rounded-[6px] border p-1.5" onclick={() => step(1)} disabled={!live || index < 0 || index >= months.length - 1} aria-label="Next month">
			<ChevronRight size={14} strokeWidth={1.8} />
		</button>
		{#if !live}
			<span class="font-mono text-[10.5px] uppercase tracking-[0.1em]" style="color: var(--bn-warn)">set fees · upload a card statement for real months</span>
		{/if}
	</div>

	<div class="mt-5 grid flex-1 items-stretch gap-5 px-6 pb-6 md:grid-cols-2 2xl:grid-cols-[1.15fr_1fr_0.85fr]">
		<!-- where the money goes: share per category, for the chosen month -->
		<SharePie
			items={cats.map((c) => ({ key: c.category, label: c.category, value: c.totalCents }))}
			{total}
			centerLabel={period}
			format={(cents) => usd(cents)}
			donutPx={190}
			ariaLabel="Monthly expenses by category"
		/>

		<!-- each category as a Deal Volume bar: its real share of the month's spend.
		     Keyed by month so stepping months sweeps them out again. -->
		<div class="flex flex-col justify-between">
			<div class="flex flex-col gap-4">
				{#if cats.length === 0}
					<p class="bn-dim py-3 text-center font-mono text-[10.5px]">Nothing spent this month.</p>
				{:else}
					{#each cats as c, i (`${month ?? 'fees'}-${c.category}`)}
						<VolumeMeter label={c.category} frac={total > 0 ? c.totalCents / total : 0} display={usd(c.totalCents)} hue={RAMP_4} delay={300 + i * 120} />
					{/each}
				{/if}
			</div>

			<!-- the three card lanes, so the split is visible without opening the report -->
			{#if live}
				<div class="bn-border mt-4 flex flex-wrap gap-x-4 gap-y-1 border-t pt-3">
					{#each lanes as l (l.card)}
						<span class="bn-dim font-mono text-[10.5px]">{cardLabel(l.card)} <span class="bn-muted">{usd(l.totalCents)}</span></span>
					{/each}
				</div>
			{/if}
		</div>

		{#if children}<div class="md:col-span-2 2xl:col-span-1">{@render children()}</div>{/if}
	</div>
</section>

{#if open}
	<ExpenditureReport {rows} {months} initialMonth={month} onclose={() => (open = false)} />
{/if}
