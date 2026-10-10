<!-- /finances in the Brand Deals slab (FounderOS v1 app/finances/page.tsx):
     where the money goes (with the statement uploader) beside Money Volume,
     then spend / charge sizes / the one insight card, income by business (once
     a bank statement is uploaded), processors, recent Stripe income and
     outgoing Wise. A processor with no pull reads "awaiting key", never live. -->
<script lang="ts">
	import { ExternalLink, Send, Upload } from '$lib/founderos/icons';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, PILL_ACCENT, Slab, SlabCard, SlabTitle, StepLine } from '$lib/founderos/kit';
	import './fin.css';
	import BusinessIncomeChart from './BusinessIncomeChart.svelte';
	import MonthlyExpenses from './MonthlyExpenses.svelte';
	import StatementUploader from './StatementUploader.svelte';
	import { keepRange, RAMP_1, RAMP_4 } from './spend-report';
	import type { FinancesPayload } from './types';

	let { data, focus = null, onuploaded }: { data: FinancesPayload; focus?: string | null; onuploaded?: (months: string[]) => void } = $props();

	const usd = (n: number, cents = false) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: cents ? 2 : 0 });

	function ago(unix: number): string {
		const mins = Math.round((Date.now() - unix * 1000) / 60_000);
		if (mins < 60) return `${Math.max(0, mins)}m`;
		const hours = Math.round(mins / 60);
		if (hours < 48) return `${hours}h`;
		return `${Math.round(hours / 24)}d`;
	}

	const inc = $derived(data.income);
	const stripe = $derived(inc.stripe);
	const accounts = $derived(inc.accounts);
	const recent = $derived(stripe.recentCharges ?? []);
	const liveCount = $derived(accounts.filter((a) => a.live).length);
	const incomeMtd = $derived(accounts.reduce((s, a) => s + (a.income ?? 0), 0));
	const incomeUpper = $derived(accounts.reduce((s, a) => s + (a.incomeUpper ?? a.income ?? 0), 0));
	const vol = $derived(data.volume);
	const latestSpend = $derived(data.spend.at(-1) ?? null);
	const wise = $derived(inc.wiseOutgoing);
	// A money band stays on one line ("$0 – $92,989"), in the headline and the meters.
	// Prod's moneyVolume always hands a number: the display carries the honesty
	// ("pull pending", "no statement this month") over a 2% floor, so an older
	// bridge's null fraction draws that floor too instead of "unknown".
	const meters = $derived(vol.meters.map((m) => ({ ...m, frac: m.frac ?? 0, display: keepRange(m.display) })));

	const meta = $derived(
		`${usd(incomeMtd)}${incomeUpper > incomeMtd ? ` – ${usd(incomeUpper)}` : ''} in this month` +
			` · ${usd(data.expenses)} out` +
			(data.expensesLive ? ` (${data.monthLabel} statement)` : ' (set fees)') +
			` · ${liveCount}/${accounts.length} processors live` +
			(stripe.live ? ` · Stripe balance ${usd(stripe.availableUsd, true)}, ${usd(stripe.pendingUsd, true)} pending` : ' · Stripe balance needs a live key')
	);
</script>

<Slab>
	<SlabTitle eyebrow="money · every processor, one view" title="Finances" {meta}>
		{#snippet right()}
			{#if data.netComparable}
				<Chip tone={data.netMonthly >= 0 ? 'ok' : 'err'}>{data.netMonthly >= 0 ? '+' : '−'}{usd(Math.abs(data.netMonthly))} net /mo</Chip>
			{/if}
			<a href="#statements" class={PILL}><Upload size={13} strokeWidth={1.7} /> Upload statement</a>
			<a href="https://dashboard.stripe.com/" target="_blank" rel="noreferrer" class={PILL_ACCENT}>Open Stripe <ExternalLink size={12} strokeWidth={1.8} /></a>
		{/snippet}
	</SlabTitle>

	{#if data.incomeError || data.ledger.error || data.bank.error || inc.wiseError}
		<div class="mb-6 flex flex-wrap gap-2">
			{#if data.incomeError}<Chip tone="err">processors unknown: {data.incomeError}</Chip>{/if}
			{#if data.ledger.error}<Chip tone="err">ledger unreachable: {data.ledger.error}</Chip>{/if}
			{#if data.bank.error}<Chip tone="err">bank statements unreachable: {data.bank.error}</Chip>{/if}
			{#if inc.wiseError}<Chip tone="err">Wise read failed: {inc.wiseError}</Chip>{/if}
		</div>
	{/if}

	<!-- Hero row: where the money goes + the Money Volume card -->
	<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={1} class="flex flex-col">
			<MonthlyExpenses rows={data.ledger.rows} months={data.ledger.months} fallback={data.fallback} {focus}>
				<!-- Statement ingestion: pick a card lane, drop a CSV or PDF -->
				<div id="statements" class="h-full scroll-mt-24">
					<StatementUploader {onuploaded} />
				</div>
			</MonthlyExpenses>
		</SlabCard>

		<SlabCard i={2} title="Money Volume" sub="month to date" class="flex flex-col">
			<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
				<BigStat value={vol.headline} kind="usd" unit={vol.upper != null ? keepRange(`– ${usd(vol.upper)}`) : undefined} chips={vol.chips} caption={vol.caption} />
				<MeterStack {meters} foot={vol.foot} empty="no processors wired yet" />
			</div>
		</SlabCard>
	</div>

	<!-- Second row: spend by month, charge sizes, THE gradient card -->
	<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
		<SlabCard i={3} title="Spend by month" sub={data.expensesLive ? 'card statements' : 'no statements yet'}>
			<div class="px-6 pt-3">
				<BigStat
					size={30}
					value={latestSpend?.count ?? 0}
					kind="usd"
					caption={latestSpend
						? `spent in ${latestSpend.label}, across ${data.spend.length} uploaded month${data.spend.length === 1 ? '' : 's'}`
						: 'upload a card statement to chart spend'}
				/>
			</div>
			<StepLine series={data.spend} hue={RAMP_4} unit=" USD" empty="No card statements uploaded yet." />
		</SlabCard>

		<SlabCard i={4} title="Charge sizes" sub={stripe.live ? 'Stripe · recent' : 'Stripe not live'}>
			<div class="flex items-end justify-between gap-4 px-6 pb-6 pt-3">
				<div>
					<BigStat size={30} value={recent.length} caption={`recent charge${recent.length === 1 ? '' : 's'}`} />
					<div class="bn-border bn-muted mt-4 w-fit rounded-full border px-3 py-1 text-[12px]">
						Largest: <span data-testid="largest-charge" class="font-semibold tabular-nums">{stripe.live && data.largestChargeUsd != null ? usd(data.largestChargeUsd, true) : ' - '}</span>
					</div>
				</div>
				<DotMatrix cols={data.chargeSizes} hue={RAMP_1} />
			</div>
		</SlabCard>

		<InsightCard i={5} badge="Kept this month" display={vol.insight.display} headline={vol.insight.headline} body={vol.insight.body} frac={vol.insight.frac} />
	</div>

	<!-- Income by business: from uploaded bank statements, with range chips -->
	{#if data.bank.series.length > 0}
		<SlabCard i={6} title="Income · by business" sub="bank deposits" class="mt-6">
			<div class="grid gap-4 px-6 pb-6 pt-4 lg:grid-cols-2">
				{#each data.bank.series as s (s.business)}
					<BusinessIncomeChart series={s} />
				{/each}
			</div>
		</SlabCard>
	{/if}

	<SlabCard i={7} title="Income · by processor" sub={`${liveCount}/${accounts.length} live`} class="mt-6">
		<div class="bn-border mt-4 border-t">
			{#each accounts as a (a.id)}
				<div data-testid="processor-row" data-lens="r" class="bn-pressable is-row fin-row grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-5 px-6 py-3.5 max-[700px]:grid-cols-[minmax(0,1fr)_auto]">
					<div class="min-w-0">
						<div class="bn-text truncate text-[13.5px] font-medium">{a.label}</div>
						<div class="bn-dim truncate font-mono text-[11px]">
							{a.processor}{a.unsplittableCustomers > 0
								? ` · ${a.unsplittableCustomers} repeat ${a.unsplittableCustomers === 1 ? 'customer' : 'customers'} · split unavailable`
								: ''}
						</div>
					</div>
					<div class="text-right">
						<div class="bn-text font-mono text-[15px] font-semibold tabular-nums">
							{a.income != null ? usd(a.income) : '—'}{#if a.incomeUpper != null}<span class="bn-dim"> – {usd(a.incomeUpper)}</span>{/if}
						</div>
						<div class="bn-dim font-mono text-[10.5px]">{a.live ? 'this month' : a.configured ? 'pull pending' : 'awaiting key'}</div>
					</div>
					<span class="fin-tag rounded-full max-[700px]:hidden" data-tone={a.live ? 'ok' : a.configured ? 'warn' : 'off'}>{a.live ? 'live' : a.configured ? 'key set' : 'connect →'}</span>
				</div>
			{/each}
		</div>
	</SlabCard>

	<!-- Recent income: real Stripe charges -->
	{#if stripe.live && recent.length > 0}
		<SlabCard i={8} title="Recent income" sub="Stripe · live" class="mt-6">
			<ul class="bn-border mt-4 border-t">
				{#each recent as c, i (`${c.created}-${i}`)}
					<li data-lens="r" class="bn-pressable is-row fin-row flex items-center gap-4 px-5 py-2.5">
						<span class="fin-tag rounded-full shrink-0 !text-[11px] font-semibold tabular-nums !normal-case !tracking-normal" data-tone="ok">+{usd(c.amount / 100, true)}</span>
						<span class="bn-muted min-w-0 flex-1 truncate text-[13px]">{c.description}</span>
						<span class="bn-dim shrink-0 font-mono text-[11px]">{ago(c.created)}</span>
					</li>
				{/each}
			</ul>
		</SlabCard>
	{/if}

	<!-- Outgoing transfers: Wise (hidden entirely until a Wise token lands) -->
	{#if wise}
		<SlabCard i={9} title="Outgoing · Wise" sub={`${wise.length} transfer${wise.length === 1 ? '' : 's'}`} class="mt-6">
			{#if wise.length === 0}
				<div class="bn-dim px-6 py-7 text-center font-mono text-[11.5px]">Wise connected · no recent outgoing transfers</div>
			{:else}
				<ul class="bn-border mt-4 border-t">
					{#each wise as t, i (`${t.created}-${i}`)}
						<li data-lens="r" class="bn-pressable is-row fin-row flex items-center gap-4 px-5 py-2.5">
							<span style="color: var(--bn-err)"><Send size={15} strokeWidth={1.8} /></span>
							<span class="fin-tag rounded-full shrink-0 !text-[11px] font-semibold tabular-nums !normal-case !tracking-normal" data-tone="err"
								>−{(t.amountCents / 100).toLocaleString('en-US', { style: 'currency', currency: t.currency })}</span
							>
							<span class="bn-muted min-w-0 flex-1 truncate text-[13px]">{t.reference ?? t.status}</span>
							<span class="bn-dim shrink-0 font-mono text-[11px]">{t.status}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</SlabCard>
	{/if}
</Slab>
