<!-- The full expenditure statement (FounderOS v1 ExpenditureReport.tsx):
     fullscreen, month to month, every card lane, every subscription incl. the
     cancelled ones. Pure client math over the ledger rows; Escape closes,
     arrows step months. -->
<script lang="ts">
	import { Minimize2, ChevronLeft, ChevronRight } from '$lib/founderos/icons';
	import { Badge, Label, SectionHead } from '$lib/founderos/kit';
	import './fin.css';
	import {
		CARD_LANES,
		cardLabel,
		cardTotals,
		categoryTotals,
		detectSubscriptions,
		monthName,
		monthlyTotals,
		spendTotalCents,
		topMerchants,
		usd,
		type SpendRow
	} from './spend-report';

	let {
		rows,
		months,
		initialMonth,
		onclose
	}: { rows: SpendRow[]; months: string[]; initialMonth: string | null; onclose: () => void } = $props();

	const latest = $derived(months.length > 0 ? months[months.length - 1] : null);
	// svelte-ignore state_referenced_locally
	let month = $state<string | null>(initialMonth ?? (months.length > 0 ? months[months.length - 1] : null));
	const index = $derived(month ? months.indexOf(month) : -1);

	function step(delta: number) {
		if (index < 0) return;
		const next = months[index + delta];
		if (next) month = next;
	}

	function onKey(e: KeyboardEvent) {
		const tag = (e.target as HTMLElement | null)?.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA') return;
		if (e.key === 'Escape') onclose();
		else if (e.key === 'ArrowLeft') step(-1);
		else if (e.key === 'ArrowRight') step(1);
	}

	const dayLabel = (date: string) => new Date(`${date}T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });

	const timeline = $derived(monthlyTotals(rows));
	const subs = $derived(latest ? detectSubscriptions(rows, latest) : []);
	const lanes = $derived(cardTotals(rows, month));
	const cats = $derived(categoryTotals(rows, month));
	const merchants = $derived(topMerchants(rows, month, 40));
	const charges = $derived(rows.filter((r) => r.direction === 'out' && (!month || r.date.slice(0, 7) === month)).sort((a, b) => b.amountCents - a.amountCents));
	const monthTotal = $derived(spendTotalCents(rows, month));
	const peak = $derived(Math.max(...timeline.map((t) => t.totalCents), 1));
	const laneMax = $derived(Math.max(...lanes.map((l) => l.totalCents), 1));
	const catMax = $derived(Math.max(...cats.map((c) => c.totalCents), 1));
	const activeSubs = $derived(subs.filter((s) => s.status === 'active'));
	const subsMonthly = $derived(activeSubs.reduce((sum, s) => sum + s.monthlyCents, 0));
</script>

<svelte:window onkeydown={onKey} />

<div class="fin-overlay fixed inset-0 z-[80] overflow-y-auto" role="dialog" aria-modal="true" aria-label="Full expenditure statement">
	<div class="mx-auto max-w-[1180px] px-6 py-6">
		<div class="bn-border mb-5 flex items-start justify-between gap-4 border-b pb-4">
			<div>
				<div class="bn-dim mb-1 font-mono text-[9.5px] uppercase tracking-[0.32em]">// full expenditure statement</div>
				<h2 class="bn-text text-[25px] font-bold uppercase tracking-[0.06em]">{month ? monthName(month) : 'All time'}</h2>
				<p class="bn-dim mt-1 font-mono text-[10.5px]">
					{usd(monthTotal)} out · {charges.length} charges · {months.length} month{months.length === 1 ? '' : 's'} on file
				</p>
			</div>
			<div class="flex items-center gap-2">
				<button type="button" class="fin-icon-btn bn-pressable" data-lens="c" onclick={() => step(-1)} disabled={index <= 0} aria-label="Previous month in statement">
					<ChevronLeft size={14} strokeWidth={1.8} />
				</button>
				<button type="button" class="fin-icon-btn bn-pressable" data-lens="c" onclick={() => step(1)} disabled={index < 0 || index >= months.length - 1} aria-label="Next month in statement">
					<ChevronRight size={14} strokeWidth={1.8} />
				</button>
				<button type="button" class="fin-range bn-pressable font-mono text-[10px] uppercase tracking-[0.1em]" data-lens="c" data-on={month === null} onclick={() => (month = null)}>all time</button>
				<button type="button" class="fin-icon-btn bn-pressable" data-lens="c" onclick={onclose} aria-label="Close expenditure statement">
					<Minimize2 size={14} strokeWidth={1.8} />
				</button>
			</div>
		</div>

		<section class="mb-5">
			<SectionHead label="Month to month" count={`${timeline.length} months`} />
			<div class="fin-panel px-4 py-4">
				{#if timeline.length === 0}
					<p class="bn-dim py-3 text-center font-mono text-[10.5px]">No statements uploaded yet.</p>
				{:else}
					<div class="flex items-end gap-2 overflow-x-auto pb-1">
						{#each timeline as t (t.month)}
							<button type="button" onclick={() => (month = t.month)} class="bn-pressable flex w-[64px] shrink-0 flex-col items-center gap-1.5" title={`${monthName(t.month)} · ${usd(t.totalCents)}`}>
								<span class="font-mono text-[9.5px] {t.month === month ? 'bn-text' : 'bn-dim'}">{usd(t.totalCents)}</span>
								<span class="flex h-[112px] w-full items-end">
									<span class="fin-bar w-full" style="height: {Math.max(2, (t.totalCents / peak) * 100)}%; opacity: {t.month === month ? 1 : 0.3}"></span>
								</span>
								<span class="font-mono text-[9.5px] uppercase tracking-[0.1em] {t.month === month ? 'bn-accent' : 'bn-dim'}">{monthName(t.month)}</span>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		</section>

		<section class="mb-5">
			<SectionHead label="By card · three lanes" count={month ? monthName(month) : 'all time'} />
			<div class="grid gap-3.5 lg:grid-cols-3">
				{#each lanes as l (l.card)}
					<div class="fin-panel px-4 py-3">
						<div class="flex items-baseline justify-between gap-2">
							<Label>{cardLabel(l.card)}</Label>
							<span class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.1em]">{monthTotal > 0 ? `${Math.round((l.totalCents / monthTotal) * 100)}%` : '—'}</span>
						</div>
						<div class="bn-text mt-1.5 font-mono text-[19px] font-semibold leading-none tracking-[-0.02em]">{usd(l.totalCents)}</div>
						<div class="fin-track mt-2.5 h-1.5 overflow-hidden">
							<div class="fin-bar h-full opacity-60" style="width: {(l.totalCents / laneMax) * 100}%"></div>
						</div>
						<p class="bn-dim mt-2 font-mono text-[10px]">{CARD_LANES.find((c) => c.id === l.card)?.blurb ?? ''}</p>
					</div>
				{/each}
			</div>
		</section>

		<section class="mb-5">
			<SectionHead label="Subscriptions · recurring charges" count={`${activeSubs.length} active · ${usd(subsMonthly)}/mo`} />
			<div class="fin-panel">
				{#if subs.length === 0}
					<p class="bn-dim px-4 py-6 text-center font-mono text-[10.5px]">
						Nothing recurring yet: a merchant needs charges in two or more months before it counts as a subscription.
					</p>
				{:else}
					<table class="w-full border-collapse">
						<thead>
							<tr class="bn-border bn-dim border-b text-left font-mono text-[9.5px] uppercase tracking-[0.16em]">
								<th class="px-4 py-2 font-medium">Merchant</th>
								<th class="px-4 py-2 font-medium">Card</th>
								<th class="px-4 py-2 text-right font-medium">Per month</th>
								<th class="px-4 py-2 font-medium">First seen</th>
								<th class="px-4 py-2 font-medium">Last charge</th>
								<th class="px-4 py-2 font-medium">Status</th>
							</tr>
						</thead>
						<tbody>
							{#each subs as s (`${s.card}|${s.merchant}`)}
								<tr class="fin-row">
									<td class="bn-text px-4 py-2 font-mono text-[11px]">{s.merchant}<span class="bn-dim ml-2">{s.category}</span></td>
									<td class="bn-muted px-4 py-2 font-mono text-[10.5px]">{cardLabel(s.card)}</td>
									<td class="bn-text px-4 py-2 text-right font-mono text-[11px]">{usd(s.monthlyCents, true)}</td>
									<td class="bn-dim px-4 py-2 font-mono text-[10.5px]">{monthName(s.firstMonth)}</td>
									<td class="bn-dim px-4 py-2 font-mono text-[10.5px]">{monthName(s.lastMonth)}</td>
									<td class="px-4 py-2">
										<Badge tone={s.status === 'active' ? 'ok' : 'default'}>{s.status === 'active' ? 'active' : `cancelled ${monthName(s.lastMonth)}`}</Badge>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>
		</section>

		<section class="mb-5 grid gap-3.5 lg:grid-cols-[1fr_1.4fr]">
			<div>
				<SectionHead label="By category" count={usd(monthTotal)} />
				<div class="fin-panel p-4">
					{#if cats.length === 0}
						<p class="bn-dim py-3 text-center font-mono text-[10.5px]">Nothing this month.</p>
					{:else}
						<div class="flex flex-col gap-2.5">
							{#each cats as c (c.category)}
								<div>
									<div class="mb-1 flex items-baseline justify-between gap-2 font-mono text-[11px]">
										<span class="bn-muted">{c.category}</span>
										<span class="bn-text">{usd(c.totalCents)}</span>
									</div>
									<div class="fin-track h-1.5 overflow-hidden">
										<div class="fin-bar h-full opacity-60" style="width: {(c.totalCents / catMax) * 100}%"></div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
			<div>
				<SectionHead label="Top merchants" count={`${merchants.length} of ${charges.length} charges`} />
				<div class="fin-panel max-h-[420px] overflow-y-auto">
					{#if merchants.length === 0}
						<p class="bn-dim px-4 py-6 text-center font-mono text-[10.5px]">Nothing this month.</p>
					{:else}
						{#each merchants as m (`${m.card}|${m.merchant}`)}
							<div class="fin-row flex items-baseline justify-between gap-3 px-4 py-2">
								<span class="bn-text min-w-0 truncate font-mono text-[11px]">{m.merchant}</span>
								<span class="bn-dim shrink-0 font-mono text-[10px]">{cardLabel(m.card)} · {m.count}×</span>
								<span class="bn-text shrink-0 font-mono text-[11px]">{usd(m.amountCents, true)}</span>
							</div>
						{/each}
					{/if}
				</div>
			</div>
		</section>

		<section class="mb-8">
			<SectionHead label="Every charge" count={`${charges.length} rows`} />
			<div class="fin-panel max-h-[520px] overflow-y-auto">
				{#if charges.length === 0}
					<p class="bn-dim px-4 py-6 text-center font-mono text-[10.5px]">No charges for this period. Upload a statement from the finances page.</p>
				{:else}
					{#each charges as r, i (`${r.date}|${r.description}|${r.amountCents}|${i}`)}
						<div class="fin-row flex items-baseline gap-3 px-4 py-2">
							<span class="bn-dim w-[54px] shrink-0 font-mono text-[10px]">{dayLabel(r.date)}</span>
							<span class="bn-text min-w-0 flex-1 truncate font-mono text-[11px]">{r.description}</span>
							<span class="bn-dim shrink-0 font-mono text-[10px]">{r.category}</span>
							<span class="bn-dim w-[92px] shrink-0 font-mono text-[10px]">{cardLabel(r.card)}</span>
							<span class="bn-text w-[84px] shrink-0 text-right font-mono text-[11px]">{usd(r.amountCents, true)}</span>
						</div>
					{/each}
				{/if}
			</div>
		</section>
	</div>
</div>
