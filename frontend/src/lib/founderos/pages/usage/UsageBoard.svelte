<!--
  The live token-burn board (FounderOS v1 components/UsageBoard.tsx). Polls
  GET /api/founderos/pages/usage every 10s. One card per plan the operator pays for
  (Claude, ChatGPT/Codex, Ollama); clicking a card opens where its tokens
  went. The hero, bottom row and insight come from usageVolume per poll.
  Layout is v1's: the hero row (a 7-day burn step line + a Burn Volume card
  of sweeping meters), a second row (models, the official limit gauges, one
  insight card), then the plan cards.
  House rule: a "% of limit" bar only renders an OFFICIAL gauge, or an
  estimate derived from one that says "est.". A plan no machine pushed reads
  unknown, never zero.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, DotMatrix, InsightCard, MeterStack, SlabCard, StepLine } from '$lib/founderos/kit';
	import OllamaCard from './OllamaCard.svelte';
	import OllamaDetail from './OllamaDetail.svelte';
	import PlanCard from './PlanCard.svelte';
	import PlanDetail from './PlanDetail.svelte';
	import { ACTIVITY_HUE, RAMP_1, age, fmtTokens, usageVolume, type UsageBody } from './usage';
	import { POLL_MS, chipClass } from './ui';

	type PlanKey = 'claude' | 'codex' | 'ollama';
	const PLAN_KEYS: Array<[PlanKey, string]> = [
		['claude', 'Claude'],
		['codex', 'ChatGPT · Codex'],
		['ollama', 'Ollama']
	];

	let board = $state<UsageBody | null>(null);
	let error = $state<string | null>(null);
	let fetchedAt = $state(0);
	let open = $state<PlanKey | null>('claude');
	let clock = $state(Date.now());

	async function load() {
		try {
			board = await founderosFetch<UsageBody>('/pages/usage');
			error = null;
			fetchedAt = Date.now();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
		clock = Date.now();
	}

	onMount(() => {
		void load();
		const t = setInterval(() => void load(), POLL_MS);
		return () => clearInterval(t);
	});

	const now = $derived(fetchedAt || clock);
	const vol = $derived(board ? usageVolume({ claude: board.claude, codex: board.codex, ollama: board.ollama, now }) : null);
	const toggle = (k: PlanKey) => (open = open === k ? null : k);
	// prod counts the Claude/Codex plans that answered, plus Ollama's lane
	const providers = $derived(board ? [board.claude, board.codex].filter(Boolean).length + 1 : 0);
	const errorLine = $derived(board?.errors ? Object.entries(board.errors).map(([k, v]) => `${k}: ${v}`).join(' · ') : null);
</script>

{#if !board || !vol}
	<div class="bn-dim bn-border bn-surface rounded-[12px] border px-6 py-8 text-[13px]">
		{error ? `usage read failed: ${error}` : 'reading local transcripts…'}
	</div>
{:else}
	<div>
		<!-- Hero row: the week's burn line + the Deal Volume card worn by token burn -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Token Burn" sub="Claude + Codex · last 7 days">
				<div class="px-6 pt-3">
					<span class="bn-text text-[30px] font-semibold tabular-nums tracking-[-0.03em]">{vol.weekTokens === null ? '—' : fmtTokens(vol.weekTokens)}</span><span
						class="bn-dim ml-2 text-[13px]">tokens burned, last 7 days</span
					>
				</div>
				<div data-part="burn-line">
					<StepLine
						series={vol.series}
						hue={ACTIVITY_HUE}
						unit="k tokens"
						empty={vol.weekTokens === null ? 'No seat pushed yet: burn unknown.' : 'No burn in this window.'}
					/>
				</div>
				<div class="bn-dim px-6 pb-5 font-mono text-[11px]">{vol.meta}</div>
			</SlabCard>

			<SlabCard i={2} title="Burn Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={vol.headlineTokens} kind="tokens" unit="tokens" chips={vol.chips} caption={vol.caption} />
					<MeterStack meters={vol.meters} foot={vol.foot} />
				</div>
			</SlabCard>
		</div>

		<!-- Second row: models, the official limits, THE insight card -->
		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Burn by Model">
				<!-- stacked, not side by side: model names are too long to share a row -->
				<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
					<div class="min-w-0">
						{#if vol.topModel}
							<BigStat value={Math.round(vol.topModel.share * 100)} kind="pct" size={30} caption="of the week on the top model" />
							<div data-part="top-model" class="bn-muted bn-border mt-4 w-fit max-w-full truncate rounded-full border px-3 py-1 text-[12px]">
								Top: <span class="font-semibold">{vol.topModel.label}</span>
							</div>
						{:else}
							<BigStat display="—" size={30} caption={vol.weekTokens === null ? 'no seat pushed yet' : 'no model burn recorded this week'} />
						{/if}
					</div>
					<DotMatrix cols={vol.models} hue={RAMP_1} />
				</div>
			</SlabCard>

			<SlabCard i={4} title="Plan Limits" sub="official gauges only">
				<div class="px-6 pb-6">
					<MeterStack meters={vol.limits} empty="No provider reported an official gauge. Burn is measured from transcripts." baseDelay={700} />
				</div>
			</SlabCard>

			<InsightCard
				i={5}
				badge="Top burner · 7 days"
				value={vol.insight.value}
				kind="pct"
				display={vol.insight.value === null ? '—' : undefined}
				headline={vol.insight.headline}
				body={vol.insight.body}
				frac={vol.insight.frac}
			/>
		</div>

		<!-- The plans: click a card to open where its tokens went -->
		<SlabCard i={6} class="mt-6" title="Plans" sub={`${providers} provider${providers === 1 ? "" : "s"}`}>
			{#snippet action()}
				{#each PLAN_KEYS as [k, label] (k)}
					<button type="button" onclick={() => toggle(k)} class={chipClass(open === k)}>{label}</button>
				{/each}
			{/snippet}
			<div class="space-y-4 px-6 pb-5 pt-4">
				<div class="grid gap-4 lg:grid-cols-3">
					<PlanCard title="Claude" plan={board.claude} selected={open === 'claude'} onselect={() => toggle('claude')} {now} />
					<PlanCard title="ChatGPT · Codex" plan={board.codex} selected={open === 'codex'} onselect={() => toggle('codex')} {now} />
					<OllamaCard lane={board.ollama} selected={open === 'ollama'} onselect={() => toggle('ollama')} />
				</div>

				{#if open === 'claude' && board.claude}<PlanDetail plan={board.claude} {now} />{/if}
				{#if open === 'codex' && board.codex}<PlanDetail plan={board.codex} {now} />{/if}
				{#if open === 'ollama' && board.ollama}<OllamaDetail lane={board.ollama} {now} />{/if}

				<div class="bn-dim bn-border flex flex-wrap items-center justify-between gap-2 border-t pt-3 font-mono text-[10.5px]">
					<span>refreshes every {POLL_MS / 1000}s · local file parsing only · no paid calls</span>
					<span>{error ? `stale — last poll failed (${error})` : `updated ${age(board.generatedAt, clock)}`}</span>
				</div>
				{#if errorLine}
					<div class="font-mono text-[10.5px] text-[color:var(--bn-warn)]">unread sources · {errorLine}</div>
				{/if}
			</div>
		</SlabCard>
	</div>
{/if}
