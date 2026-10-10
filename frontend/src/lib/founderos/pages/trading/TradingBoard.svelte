<!-- /os/trading: the Robinhood accounts and the Phantom wallet as one slab
     (FounderOS v1 components/trading/TradingBoard.tsx, followed by its code, not
     the older CLAUDE.md prose). Monitor-only: the broker numbers arrive by push
     from the Markets Agent feed, and the page re-reads the bridge every 60s and
     on demand. Layout: a hero row of the sleeve's value line beside the
     Accounts card; a second row of reasoning, positions and the one gradient
     card (the agent's next move); then open orders, the trade log, limits. -->
<script lang="ts">
	import { Bot, RefreshCw, X } from '$lib/founderos/icons';
	import { Badge, BigStat, DotMatrix, InsightCard, Label, MeterStack, Slab, SlabCard, SlabTitle, ToggleChip, chipClass } from '$lib/founderos/kit';
	import ActionTag from './ActionTag.svelte';
	import './trading.css';
	import AgentReasoning from './AgentReasoning.svelte';
	import AgentTradeChart from './AgentTradeChart.svelte';
	import CardHead from './CardHead.svelte';
	import TradingLimits from './TradingLimits.svelte';
	import { AGENTIC_ID, type TradingPayload } from './types';
	import { activityCounts, clock, filterActivity, placedByAgent, pnl, statusTone, timeAgo, usd, usd0, type ActivityFilter } from './view';

	let { data, busy = false, onRefresh }: { data: TradingPayload; busy?: boolean; onRefresh: () => void } = $props();

	let view = $state<string>(AGENTIC_ID);
	let logFilter = $state<ActivityFilter>('all');
	let selectedId = $state<string | null>(null);

	const now = $derived(Date.parse(data.at));
	const fresh = $derived(data.view.freshness);
	const agent = $derived(data.view.agent);
	const vol = $derived(data.view.volume);
	const agentAccount = $derived(data.accounts.find((a) => a.accountId === AGENTIC_ID) ?? null);
	const viewAccount = $derived(data.accounts.find((a) => a.accountId === view) ?? agentAccount);
	const viewId = $derived(viewAccount?.accountId ?? AGENTIC_ID);
	const viewTrades = $derived(data.activity.filter((a) => a.accountId === viewId));
	const invested = $derived(data.positions.reduce((s, p) => s + p.marketValueUsd, 0));
	const counts = $derived(activityCounts(data.activity));
	const rows = $derived(filterActivity(data.activity, logFilter));
	const selected = $derived(data.activity.find((a) => a.id === selectedId) ?? null);
	// An unpriced wallet stays out of the total and the title says so, rather
	// than claiming "+ wallet" over an understated sum.
	const walletPriced = $derived(data.phantom?.usdValue != null);
	const total = $derived(data.accounts.reduce((s, a) => s + a.accountValueUsd, 0) + (data.phantom?.usdValue ?? 0));
	// Unknown until the sleeve has been fed: no ticks, never a fabricated 0%.
	const deployedFrac = $derived(agentAccount ? agent.deployedUsd / Math.max(agentAccount.accountValueUsd, 1) : null);

	const toneColor = (t: string) => (t === 'ok' ? 'var(--bn-ok)' : t === 'err' ? 'var(--bn-err)' : 'var(--bn-text-2)');

	const LOG_FILTERS: ActivityFilter[] = ['all', 'agent', 'you', 'rejected'];
	const LOG_LABEL: Record<ActivityFilter, string> = { all: 'All', agent: 'Agent', you: 'You', rejected: 'Rejected' };

	const insightHeadline = $derived(
		data.openOrders.length > 0
			? `order${data.openOrders.length === 1 ? '' : 's'} working at the broker.`
			: data.analysis
				? `signal${data.analysis.signals === 1 ? '' : 's'} on the last run, ${timeAgo(data.analysis.at, now)} ago.`
				: 'runs recorded. The agent has not reported in.'
	);
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && (selectedId = null)} />

<Slab>
	<SlabTitle eyebrow="markets · robinhood + phantom · agent-fed" title="Trading">
		{#snippet meta()}
			{usd(total)} across {data.accounts.length} account{data.accounts.length === 1 ? '' : 's'}{data.phantom ? (walletPriced ? ' + wallet' : ' · wallet value unknown') : ''} · {fresh.label}{fresh.state ===
			'stale'
				? ' · the feed has not pushed in a while'
				: ''}
		{/snippet}
		{#snippet right()}
			{#if fresh.state === 'live'}
				<Badge tone="ok">live · robinhood</Badge>
			{:else if fresh.state === 'stale'}
				<Badge tone="warn">stale · {fresh.label.replace('synced ', '')}</Badge>
			{:else if fresh.state === 'seeded'}
				<Badge tone="warn">seeded · awaiting the feed</Badge>
			{:else}
				<Badge tone="err">no feed</Badge>
			{/if}
			<span class="bn-muted bn-border rounded-full border px-4 py-2 text-[13px]">monitor-only · refreshes 60s</span>
			<button
				type="button"
				onclick={onRefresh}
				aria-label="Refresh feed"
				title="Refresh feed"
				data-lens="c"
				class="bn-pressable bn-muted bn-border tb-hover grid h-10 w-10 place-items-center rounded-full border"
			>
				<RefreshCw size={15} strokeWidth={1.7} class={busy ? 'animate-spin' : ''} />
			</button>
		{/snippet}
	</SlabTitle>

	{#if data.accounts.length === 0}
		<SlabCard i={1} class="px-6 py-12 text-center">
			<div class="bn-text text-[19px] font-semibold">No account data yet</div>
			<div class="bn-dim mx-auto mt-2 max-w-md text-[12.5px] leading-relaxed">{data.status.detail}</div>
		</SlabCard>
	{:else}
		<!-- Hero row: the value line + the accounts card -->
		<div data-part="hero-row" class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<div data-part="sleeve" class="flex min-w-0">
				<SlabCard i={1} class="flex w-full flex-col pb-5">
					<CardHead
						title={view === AGENTIC_ID ? 'Agent · the sleeve' : `${viewAccount?.accountLabel ?? 'Account'} · value`}
						meta={view === AGENTIC_ID
							? agent.hasActed && agent.lastTradeAt
								? `${agent.tradeCount} trade${agent.tradeCount === 1 ? '' : 's'} · last ${timeAgo(agent.lastTradeAt, now)} ago`
								: 'no trades placed yet'
							: 'read-only to agents'}
						{onRefresh}
					/>
					<div class="mt-3 flex flex-wrap items-center gap-1.5 px-6">
						{#each data.accounts as a (a.accountId)}
							<ToggleChip on={viewId === a.accountId} onclick={() => (view = a.accountId)}>{a.accountLabel}</ToggleChip>
						{/each}
					</div>
					<div class="px-6 pt-4">
						<AgentTradeChart history={data.history[viewId] ?? []} trades={viewTrades} />
					</div>
					<div data-part="sleeve-stats" class="mx-6 mt-4 grid grid-cols-2 gap-px overflow-hidden rounded-[10px] border sm:grid-cols-4" style="border-color: var(--bn-border); background: var(--bn-border)">
						{#each [{ label: 'Deployed', value: usd(agent.deployedUsd), color: '' }, { label: 'Idle cash', value: usd(agent.idleCashUsd), color: '' }, { label: 'Unrealized', value: pnl(agent.unrealizedPnlUsd).text, color: toneColor(pnl(agent.unrealizedPnlUsd).tone) }, { label: 'Agent trades', value: String(agent.tradeCount), color: '' }] as s (s.label)}
							<div class="min-w-0 px-4 py-3" style="background: var(--bn-surface)">
								<div class="bn-dim truncate font-mono text-[9px] uppercase tracking-[0.2em]">{s.label}</div>
								<div class="mt-1 truncate font-mono text-[16px] font-bold tabular-nums" style={s.color ? `color: ${s.color}` : ''}>{s.value}</div>
							</div>
						{/each}
					</div>
					{#if !agent.hasActed}
						<div class="bn-dim mx-6 mt-3 text-[12px] leading-relaxed">
							{agentAccount
								? `The sleeve is funded with ${usd(agentAccount.cashUsd)} and sitting in cash. Nothing here moves until the agent places a trade against it.`
								: 'No agentic account has been fed yet.'}
						</div>
					{/if}
				</SlabCard>
			</div>

			<div data-part="accounts" class="flex min-w-0">
				<SlabCard i={2} class="flex w-full flex-col">
					<CardHead title="Accounts" meta={data.status.detail} {onRefresh} />
					<!-- Deal Volume's shape: count-up headline, dot chips, one meter per
					     account plus the wallet, each its share of everything held -->
					<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
						<BigStat value={vol.headline} kind="usdCents" chips={vol.chips} caption={vol.caption} />
						<MeterStack meters={vol.meters} foot={vol.foot} />
					</div>
				</SlabCard>
			</div>
		</div>

		<!-- Second row: reasoning + positions + THE gradient card -->
		<div data-part="second-row" class="mt-6 grid grid-cols-[1.15fr_1fr_.85fr] gap-6 max-[1200px]:grid-cols-1">
			<div data-part="reasoning" class="flex min-w-0">
				<SlabCard i={3} class="flex min-h-0 w-full flex-col">
					<CardHead
						title="Agent · reasoning"
						meta={data.analysis
							? `${data.analysis.examined} examined · ${data.analysis.signals} signal${data.analysis.signals === 1 ? '' : 's'} · ${timeAgo(data.analysis.at, now)} ago`
							: 'no run recorded'}
						{onRefresh}
					/>
					<AgentReasoning analysis={data.analysis} />
				</SlabCard>
			</div>

			<div data-part="positions" class="flex min-w-0">
				<SlabCard i={4} class="flex min-h-0 w-full flex-col">
					<CardHead title="Positions" meta="{usd0(invested)} invested" {onRefresh} />
					<div class="flex items-end justify-between gap-4 px-6 pt-3">
						<BigStat size={30} value={data.positions.length} caption="open position{data.positions.length === 1 ? '' : 's'}" />
						<DotMatrix cols={data.view.sizes} hue="color-mix(in oklab, var(--bn-text) 60%, transparent)" />
					</div>
					<ul class="mt-4 max-h-[236px] flex-1 overflow-y-auto border-t" style="border-color: var(--bn-border)">
						{#each data.positions as p (`${p.accountId}-${p.symbol}`)}
							{@const u = pnl(p.unrealizedPnlUsd)}
							<li data-lens="r" class="bn-pressable is-row grid min-w-0 grid-cols-[minmax(0,1fr)_auto_auto] items-baseline gap-x-3 border-b px-6 py-2.5 last:border-0" style="border-color: var(--bn-hairline)">
								<div class="min-w-0">
									<span class="font-mono text-[12.5px] font-bold">{p.symbol}</span><span class="bn-dim ml-2 font-mono text-[9.5px] uppercase tracking-[0.14em]">{p.accountId}</span>
									<div class="bn-dim truncate font-mono text-[10.5px] tabular-nums">{p.quantity} @ {usd(p.avgCostUsd)}</div>
								</div>
								<span class="font-mono text-[12px] tabular-nums">{usd(p.marketValueUsd)}</span>
								<span class="w-[72px] text-right font-mono text-[11px] tabular-nums" style="color: {toneColor(u.tone)}">{u.text}</span>
							</li>
						{:else}
							<li class="bn-dim px-6 py-7 text-center font-mono text-[11.5px]">No open positions.</li>
						{/each}
					</ul>
				</SlabCard>
			</div>

			<!-- the ONE gradient insight card: the agent's next move -->
			<InsightCard
				i={6}
				badge="Agent · next move"
				value={data.openOrders.length > 0 ? data.openOrders.length : (data.analysis?.signals ?? 0)}
				headline={insightHeadline}
				frac={deployedFrac}
			>
				{#snippet body()}
					<span class="line-clamp-4">
						{data.analysis?.notes ||
							'Index core: QQQ and SPY, bought on dips and topped up monthly, with buys stopping at the kill-switch floor. Every order is checked in code before it reaches the broker.'}
					</span>
					<span class="mt-1.5 block font-mono text-[10px]"
						>{deployedFrac === null ? 'sleeve not fed yet' : `${Math.round(deployedFrac * 100)}% of the sleeve deployed`}</span
					>
				{/snippet}
			</InsightCard>
		</div>

		<!-- Open orders: what has not happened yet -->
		<div data-part="open-orders" class="mt-6">
			<SlabCard i={7}>
				<CardHead title="Open orders" meta={data.openOrders.length === 0 ? 'nothing working' : `${data.openOrders.length} working`} {onRefresh} />
				{#if data.openOrders.length === 0}
					<div class="bn-dim px-6 py-7 text-center font-mono text-[11.5px]">No live orders at the broker.</div>
				{:else}
					<ul class="mt-4 border-t" style="border-color: var(--bn-border)">
						{#each data.openOrders as o (o.id)}
							<li data-lens="r" class="bn-pressable is-row flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1 border-b px-6 py-3 last:border-0" style="border-color: var(--bn-hairline)">
								<ActionTag action={o.side} />
								<span class="font-mono text-[12.5px] font-bold">{o.symbol}</span>
								<span class="bn-muted font-mono text-[11px] tabular-nums"
									>{o.dollarAmountUsd !== null ? usd(o.dollarAmountUsd) : `${o.quantity}`}{o.limitPriceUsd !== null ? ` limit ${usd(o.limitPriceUsd)}` : ''}{o.filledQuantity >
									0
										? ` · ${o.filledQuantity}/${o.quantity} filled`
										: ''}</span
								>
								<Badge tone="warn" ghost>{o.state}</Badge>
								<!-- Robinhood stamps placed_agent 'agentic' on anything the agent submitted -->
								<Badge tone={o.placedAgent === 'agentic' ? 'ok' : 'default'} ghost>{o.placedAgent === 'agentic' ? 'agent' : 'you'}</Badge>
								<span class="flex-1"></span>
								<span class="bn-dim font-mono text-[10px]">{o.type} · {timeAgo(o.createdAt, now)} ago</span>
							</li>
						{/each}
					</ul>
				{/if}
			</SlabCard>
		</div>

		<!-- Trade log: filterable by who and outcome; the full rationale is in the drawer -->
		<div data-part="trade-log" class="mt-6">
			<SlabCard i={8}>
				<div class="flex flex-wrap items-center justify-between gap-4 px-6 pt-5">
					<h2 class="bn-text text-[19px] font-semibold tracking-[-0.01em]">
						Trade log <span class="bn-dim ml-1 font-mono text-[12px] font-normal tabular-nums">{rows.length} of {data.activity.length}</span>
					</h2>
					<div class="flex flex-wrap items-center gap-2">
						{#each LOG_FILTERS as key (key)}
							<button type="button" data-lens="c" class={chipClass(logFilter === key)} onclick={() => (logFilter = key)}>{LOG_LABEL[key]} {counts[key]}</button>
						{/each}
					</div>
				</div>
				<div class="mt-4 border-t" style="border-color: var(--bn-border)">
					{#each rows as a (a.id)}
						<button
							type="button"
							data-lens="r"
							onclick={() => (selectedId = a.id)}
							class="bn-pressable is-row grid w-full min-w-0 grid-cols-[52px_minmax(0,1fr)_auto] items-start gap-x-4 border-b px-6 py-3.5 text-left last:border-0 max-[900px]:grid-cols-[52px_minmax(0,1fr)]"
							style="border-color: var(--bn-hairline)"
						>
							<ActionTag action={a.action} />
							<div class="min-w-0">
								<div class="flex min-w-0 flex-wrap items-baseline gap-x-2 font-mono text-[12.5px]">
									<span class="font-bold">{a.symbol}</span>
									<span class="bn-muted tabular-nums">{a.quantity} @ {usd(a.priceUsd)}</span>
									<Badge tone={statusTone(a.status)} ghost>{a.status}</Badge>
								</div>
								<div class="bn-dim mt-1 line-clamp-2 text-[11.5px] leading-snug">{a.rationale || 'no rationale recorded'}</div>
							</div>
							<div class="bn-dim shrink-0 text-right font-mono text-[10px] max-[900px]:hidden">
								<div class="truncate">{a.agent} · <span class="uppercase tracking-wider">{a.accountId}</span></div>
								<div>{timeAgo(a.at, now)} ago</div>
							</div>
						</button>
					{:else}
						<div class="bn-dim px-6 py-8 text-center text-[12.5px]">{data.activity.length === 0 ? 'No trades logged yet.' : 'Nothing matches that filter.'}</div>
					{/each}
				</div>
			</SlabCard>
		</div>
	{/if}

	<!-- Settings, not state: shown even with no account data -->
	<div class="mt-6">
		<TradingLimits view={data.limits} />
	</div>
	{#if fresh.state === 'seeded'}
		<div class="bn-dim mt-4 text-center font-mono text-[10.5px]">Example rows. The first real push from the Markets Agent feed replaces them.</div>
	{/if}
</Slab>

{#if selected}
	<div class="fixed inset-0 z-[60] bg-black/45 backdrop-blur-[2px]" role="presentation" onclick={() => (selectedId = null)}></div>
	<!-- prod's glass: bg-os-bg2/95 is never generated, so the drawer is the
	     blur alone over the dimmed page -->
	<aside class="fixed right-0 top-0 z-[70] flex h-full w-[460px] max-w-[92vw] flex-col border-l backdrop-blur" style="border-color: var(--bn-border)">
		<div class="flex items-start justify-between gap-4 border-b px-6 py-5" style="border-color: var(--bn-border)">
			<div class="min-w-0">
				<div class="bn-text flex items-center gap-2 text-[17px] font-semibold">
					<ActionTag action={selected.action} />
					{selected.symbol}
				</div>
				<div class="bn-dim mt-0.5 truncate font-mono text-[11.5px]">{selected.quantity} @ {usd(selected.priceUsd)} · {clock(selected.at)}</div>
				<div class="mt-2.5 flex flex-wrap items-center gap-2">
					<Badge tone={statusTone(selected.status)}>{selected.status}</Badge>
					<Badge tone={placedByAgent(selected) ? 'ok' : 'default'} ghost>{selected.agent}</Badge>
					<Badge ghost>{selected.accountId}</Badge>
				</div>
			</div>
			<button type="button" onclick={() => (selectedId = null)} aria-label="Close" data-lens="c" class="bn-pressable bn-muted bn-border tb-hover grid h-[30px] w-[30px] shrink-0 place-items-center rounded-[5px] border">
				<X size={15} strokeWidth={1.8} />
			</button>
		</div>
		<div class="min-h-0 flex-1 overflow-y-auto px-6 py-5">
			<section class="mb-6">
				<Label rule>Rationale</Label>
				<p class="bn-text mt-2 whitespace-pre-wrap text-[12.5px] leading-relaxed">{selected.rationale || 'No rationale was recorded with this trade.'}</p>
			</section>
			<section class="mb-6">
				<Label rule>Order</Label>
				<div class="mt-2">
					{#each [['Side', selected.action], ['Quantity', String(selected.quantity)], ['Price', usd(selected.priceUsd)], ['Notional', usd(selected.quantity * selected.priceUsd)], ['Outcome', selected.status]] as [k, v] (k)}
						<div class="flex items-baseline justify-between gap-4 border-b py-2 last:border-0" style="border-color: var(--bn-hairline)">
							<span class="bn-dim text-[12px]">{k}</span><span class="bn-text text-right text-[12.5px] tabular-nums">{v}</span>
						</div>
					{/each}
				</div>
			</section>
			<section>
				<Label rule>Who and where</Label>
				<div class="mt-2">
					{#each [['Placed by', selected.agent], ['Account', selected.accountId], ['When', `${timeAgo(selected.at, now)} ago (${selected.at.slice(0, 10)})`]] as [k, v] (k)}
						<div class="flex items-baseline justify-between gap-4 border-b py-2 last:border-0" style="border-color: var(--bn-hairline)">
							<span class="bn-dim text-[12px]">{k}</span><span class="bn-text text-right text-[12.5px] tabular-nums">{v}</span>
						</div>
					{/each}
				</div>
				<p class="bn-dim mt-3 flex items-start gap-2 text-[11px] leading-relaxed">
					<Bot size={12} class="mt-0.5 shrink-0" /> Rejected rows carry the exact broker or guardrail reason in the rationale; nothing is retried silently.
				</p>
			</section>
		</div>
	</aside>
{/if}
