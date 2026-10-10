<!-- The AdPilot command deck (FounderOS v1 components/adpilot/AdPilotDeck.tsx):
     campaigns chip, the globe with glowing stat floaters, the campaign rail,
     then Spend, Delivery, ROAS, Results and Audience. One selection drives
     everything; each selection's aggregates arrive precomputed (deck.views).
     Campaigns arrive only from the ad-account sync. With none, the deck says
     "No ad account connected" and the floaters sum the empty campaign set
     to 0, exactly as production does. -->
<script lang="ts">
	import { ArrowDownRight, ArrowUpRight, Search } from '$lib/founderos/icons';
	import Globe from './Globe.svelte';
	import { deliveryShade, money, money2, num, pct, railOrder, times, windowDelta } from './logic';
	import type { Campaign, Deck } from './types';

	let { deck }: { deck: Deck } = $props();

	const RAIL_FILTER_THRESHOLD = 10;
	const SEGMENTS = 18;

	let selected = $state('all');
	let filter = $state('');

	const campaigns = $derived(deck.campaigns);
	const empty = $derived(campaigns.length === 0 || !deck.views);
	const view = $derived(deck.views ? (deck.views[selected] ?? deck.views.all) : null);
	const active = $derived(selected === 'all' ? campaigns : campaigns.filter((c) => c.id === selected));
	const ordered = $derived(railOrder(campaigns, filter));
	const maxSpend = $derived(Math.max(...campaigns.map((c) => c.spend), 1));
	const deliveryRows = $derived(campaigns.filter((c) => (c.daily?.length ?? 0) > 0));
	const dates = $derived(deliveryRows[0]?.daily?.map((d) => d.date) ?? []);
	const peakAll = $derived(
		deliveryRows.flatMap((c) => c.daily ?? []).reduce((best, d) => (d.spend > best.spend ? d : best), { date: '', spend: 0 })
	);

	const R = 56;
	const C = 2 * Math.PI * R;
	const ROAS_CAP = 4;
	const roas = $derived(view?.metrics.roas ?? null);
	const ringFrac = $derived(roas === null ? 0 : Math.min(roas / ROAS_CAP, 1));

	const results = $derived(
		view
			? ([
					['Leads', num(view.metrics.leads)],
					['Bookings', num(view.metrics.bookings)],
					['Cost / lead', money2(view.metrics.cpl)],
					['Cost / booking', money2(view.metrics.costPerBooking)],
					['Cost / result', money2(view.metrics.costPerResult)],
					['CTR', pct(view.metrics.ctr)]
				] as [string, string][])
			: []
	);

	const deltas = $derived(
		view
			? [
					{ label: 'spend', delta: windowDelta(view.daily, (d) => d.spend) },
					{ label: 'leads', delta: windowDelta(view.daily, (d) => d.leads) }
				]
			: []
	);

	// Production sums the active campaign set (nf.format(metrics.leads)), so
	// an empty set reads 0 beside the "No ad account connected" chip.
	const floatValue = (pick: (m: NonNullable<typeof view>['metrics']) => number): string =>
		view ? num(pick(view.metrics)) : '0';

	const isDimmed = (c: Campaign) => selected !== 'all' && !active.some((a) => a.id === c.id);
</script>

<div data-part="deck">
	<div class="mb-6 flex justify-center">
		<div class="ap-chip flex items-center gap-2 px-5 py-2">
			<span class="h-2 w-2 rounded-full" style="background: {deck.liveCount > 0 ? 'var(--bn-accent)' : 'var(--bn-text-3)'}"></span>
			<span class="bn-text text-[13px]">
				{empty ? 'No ad account connected' : `${deck.liveCount} active campaign${deck.liveCount === 1 ? '' : 's'}`}
			</span>
		</div>
	</div>

	<div class="relative">
		<div
			aria-hidden="true"
			class="pointer-events-none absolute left-1/2 top-1/2 aspect-square w-[820px] max-w-none -translate-x-1/2 -translate-y-1/2 rounded-full"
			style="background: radial-gradient(circle, rgba(255,47,68,0.18), rgba(255,47,68,0.06) 46%, transparent 68%)"
		></div>
		<div class="pointer-events-none absolute right-0 top-10 z-10 flex w-48 flex-col gap-3">
			{#each [['Total leads · 30 days', floatValue((m) => m.leads)], ['Booked calls', floatValue((m) => m.bookings)]] as [label, value] (label)}
				<div class="ap-frost px-4 py-3" data-part="stat-float">
					<div class="bn-dim relative text-[11.5px]">{label}</div>
					<div class="bn-text relative mt-0.5 text-[24px] font-light tabular-nums">{value}</div>
				</div>
			{/each}
		</div>
		<Globe geo={view?.geo ?? []} height={560} />
	</div>

	{#if empty || !view}
		<div class="ap-panel mx-auto mt-4 max-w-xl px-4 py-6 text-center" data-part="no-account">
			<p class="bn-muted text-[13px]">No ad account connected.</p>
			<p class="bn-dim mt-1 text-[12px]">
				Campaigns arrive automatically once a Meta ad account is connected: nothing here is added by hand.
			</p>
		</div>
	{:else}
		<!-- campaign rail -->
		<div class="ap-panel mt-6 flex items-center gap-2 p-2" data-part="rail">
			{#snippet pill(id: string, label: string, live: boolean | null)}
				<button
					type="button"
					onclick={() => (selected = id)}
					aria-pressed={selected === id}
					class="bn-pressable flex shrink-0 snap-start items-center gap-2 rounded-full border px-3.5 py-2 text-[12.5px] {selected === id
						? 'bn-text border-[var(--bn-accent-line)] bg-[var(--bn-accent-soft)]'
						: 'bn-dim ap-hover-muted border-[rgba(255,69,87,0.14)]'}"
				>
					{#if live !== null}
						<span class="h-1.5 w-1.5 rounded-full" style="background: {live ? 'var(--bn-accent)' : 'var(--bn-text-3)'}"></span>
					{/if}
					<span class="max-w-[220px] truncate">{label}</span>
				</button>
			{/snippet}
			{@render pill('all', 'All campaigns', null)}
			<div class="h-6 w-px shrink-0 bg-[rgba(255,69,87,0.16)]"></div>
			<div
				class="ap-rail flex min-w-0 flex-1 snap-x gap-2 overflow-x-auto"
				style="scrollbar-width: none; mask-image: linear-gradient(90deg, #000 92%, transparent); -webkit-mask-image: linear-gradient(90deg, #000 92%, transparent)"
			>
				{#each ordered as c (c.id)}
					{@render pill(c.id, c.name, c.status === 'active')}
				{/each}
			</div>
			{#if campaigns.length > RAIL_FILTER_THRESHOLD}
				<div class="flex shrink-0 items-center gap-1.5 pl-1">
					<Search class="bn-dim h-3.5 w-3.5" strokeWidth={1.7} />
					<input bind:value={filter} placeholder="filter" class="bn-text ap-ph w-24 bg-transparent text-[12px] outline-none" />
				</div>
			{/if}
		</div>

		<div class="mt-5 grid gap-4 lg:grid-cols-12">
			<!-- hero spend -->
			<div class="lg:col-span-4">
				<div class="ap-panel ap-glow h-full p-5" data-part="spend">
					<div class="bn-dim text-[12.5px]">Ad spend · 30 days</div>
					<div class="bn-text mt-1 text-[36px] font-light leading-none tabular-nums">{money(view.metrics.spend)}</div>
					<div class="mt-2.5 flex flex-wrap gap-1.5">
						{#each deltas as d (d.label)}
							{#if d.delta !== null}
								<span class="flex items-center gap-1 rounded-full border border-[rgba(255,69,87,0.18)] bg-black/30 px-2.5 py-1">
									{#if d.delta >= 0}
										<ArrowUpRight class="h-3 w-3" style="color: var(--bn-accent)" strokeWidth={1.8} />
									{:else}
										<ArrowDownRight class="bn-dim h-3 w-3" strokeWidth={1.8} />
									{/if}
									<span class="bn-muted text-[11px] tabular-nums">{d.delta >= 0 ? '+' : ''}{(d.delta * 100).toFixed(0)}% {d.label} · 7d</span>
								</span>
							{/if}
						{/each}
					</div>
					<div class="mt-5 flex flex-col gap-3">
						{#each campaigns as c (c.id)}
							<div class={isDimmed(c) ? 'opacity-35' : ''}>
								<div class="flex items-baseline justify-between gap-2">
									<span class="bn-muted truncate text-[12.5px]">{c.name}</span>
									<span class="bn-text text-[12px] tabular-nums">{money(c.spend)}</span>
								</div>
								<div class="mt-1.5 h-[7px] overflow-hidden rounded-full bg-black/40">
									<div class="ap-hatch h-full rounded-full" style="width: {(c.spend / maxSpend) * 100}%"></div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- delivery matrix -->
			<div class="lg:col-span-8">
				{#if deliveryRows.length === 0}
					<div class="ap-panel flex h-full items-center justify-center p-5" data-part="delivery">
						<span class="bn-dim text-[12px]">Daily delivery arrives with the ad-account sync.</span>
					</div>
				{:else}
					<div class="ap-panel h-full p-5" data-part="delivery">
						<div class="flex items-baseline justify-between">
							<div class="bn-dim text-[12.5px]">Delivery · spend per day</div>
							<div class="bn-dim text-[11.5px] tabular-nums">{dates[0]?.slice(5)}: {dates[dates.length - 1]?.slice(5)}</div>
						</div>
						<div class="mt-4 flex flex-col gap-3.5">
							{#each deliveryRows as c (c.id)}
								{@const rowPeak = Math.max(...(c.daily ?? []).map((d) => d.spend), 1)}
								<div class={selected !== 'all' && selected !== c.id ? 'opacity-30' : ''}>
									<div class="mb-1.5 flex items-baseline justify-between">
										<span class="bn-muted truncate text-[12.5px]">{c.name}</span>
										<span class="bn-dim text-[11px] tabular-nums">peak {money(rowPeak)}/d</span>
									</div>
									<div class="flex justify-between gap-[3px]">
										{#each c.daily ?? [] as d (d.date)}
											<span
												title="{d.date} · ${num(Math.round(d.spend))} · {d.leads} leads"
												class="aspect-square w-full max-w-[13px] rounded-full"
												style="background: {deliveryShade(d.spend / rowPeak)}"
											></span>
										{/each}
									</div>
								</div>
							{/each}
						</div>
						<div class="mt-4 flex items-center justify-between border-t border-[rgba(255,69,87,0.12)] pt-3">
							<span class="bn-dim text-[11.5px]">Peak day {peakAll.date.slice(5)} · {money(peakAll.spend)}</span>
							<span class="bn-dim flex items-center gap-1.5 text-[11px]">
								quiet
								{#each [0.15, 0.45, 0.7, 1] as t (t)}
									<span class="h-[9px] w-[9px] rounded-full" style="background: {deliveryShade(t)}"></span>
								{/each}
								heavy
							</span>
						</div>
					</div>
				{/if}
			</div>

			<!-- ROAS ring -->
			<div class="lg:col-span-3">
				<div class="ap-panel flex h-full flex-col items-center justify-center p-5" data-part="roas">
					<div class="bn-dim self-start text-[12.5px]">ROAS · revenue vs spend</div>
					<div class="relative mt-1">
						<svg width="150" height="150" viewBox="0 0 150 150" aria-hidden="true">
							<circle cx="75" cy="75" r={R} fill="none" stroke="rgba(255,255,255,0.07)" stroke-width="10" />
							<circle
								cx="75"
								cy="75"
								r={R}
								fill="none"
								stroke="#ff4557"
								stroke-width="10"
								stroke-linecap="round"
								stroke-dasharray="{(ringFrac * C).toFixed(1)} {C.toFixed(1)}"
								transform="rotate(-90 75 75)"
							/>
							<line x1="75" y1="11" x2="75" y2="21" stroke="#93a19d" stroke-width="2" transform="rotate({(1 / ROAS_CAP) * 360} 75 75)" />
						</svg>
						<div class="absolute inset-0 flex flex-col items-center justify-center">
							<span class="text-[27px] font-light tabular-nums" style="color: {roas !== null && roas >= 1 ? 'var(--bn-accent)' : 'var(--bn-text)'}">{times(roas)}</span>
							<span class="bn-dim text-[10.5px]">break-even 1.00x</span>
						</div>
					</div>
					<div class="bn-dim mt-1.5 flex gap-4 text-[11.5px] tabular-nums">
						<span>rev {money(view.metrics.revenue)}</span>
						<span>spend {money(view.metrics.spend)}</span>
					</div>
				</div>
			</div>

			<!-- results -->
			<div class="lg:col-span-4">
				<div class="ap-panel h-full p-5" data-part="results">
					<div class="bn-dim text-[12.5px]">Results</div>
					<div class="mt-3 grid grid-cols-2 gap-x-4 gap-y-4">
						{#each results as [label, value] (label)}
							<div class="flex items-center gap-2.5">
								<span class="h-8 w-[3px] shrink-0 rounded-full opacity-70" style="background: var(--bn-accent)"></span>
								<div class="min-w-0">
									<div class="bn-dim truncate text-[11.5px]">{label}</div>
									<div class="bn-text text-[20px] font-light tabular-nums">{value}</div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- audience -->
			<div class="lg:col-span-5">
				<div class="ap-panel h-full p-5" data-part="audience">
					<div class="bn-dim text-[12.5px]">Audience</div>
					<div class="mt-3 grid gap-x-6 gap-y-4 sm:grid-cols-2">
						{#each [['Age', view.audience.age], ['Gender', view.audience.gender], ['Placements', view.audience.placements]] as [title, data] (title)}
							<div class={title === 'Age' ? 'sm:row-span-2' : ''}>
								<div class="bn-dim text-[11px]">{title}</div>
								<div class="mt-1.5 flex flex-col gap-1.5">
									{#each Object.entries(data as Record<string, number>) as [key, value] (key)}
										{@const filled = Math.round((value / 100) * SEGMENTS)}
										<div class="flex items-center gap-2">
											<span class="bn-muted w-20 shrink-0 truncate text-[12px]">{key}</span>
											<div class="flex flex-1 gap-[2.5px]">
												{#each { length: SEGMENTS } as _, i (i)}
													<span class="h-[10px] flex-1 rounded-[2px]" style="background: {i < filled ? 'rgba(255,69,87,0.85)' : 'rgba(255,255,255,0.06)'}"></span>
												{/each}
											</div>
											<span class="bn-dim w-9 shrink-0 text-right text-[11px] tabular-nums">{value.toFixed(0)}%</span>
										</div>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
