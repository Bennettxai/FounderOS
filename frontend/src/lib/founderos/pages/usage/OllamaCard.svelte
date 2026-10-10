<!-- The Ollama plan's card (UsageBoard.tsx OllamaCard). Nobody reporting reads unknown, not "down". -->
<script lang="ts">
	import { Badge, Dot } from '$lib/founderos/kit';
	import { WINDOWS, WINDOW_LABEL, type OllamaBoard } from './usage';

	let { lane, selected, onselect }: { lane: OllamaBoard | null; selected: boolean; onselect: () => void } = $props();

	const cloud = $derived(lane ? lane.models.filter((m) => m.cloud).length : 0);
	import { EMPTY_CARD, planCardClass } from './ui';
</script>

{#if !lane}
	<div data-part="plan-card" class={EMPTY_CARD}>
		<span class="bn-muted text-[15px] font-semibold">Ollama</span>
		<p class="mt-2">no machine has reported Ollama yet</p>
	</div>
{:else}
	<button
		type="button"
		data-part="plan-card"
		onclick={onselect}
		aria-expanded={selected}
		data-lens="r"
		class={planCardClass(selected)}
	>
		<div class="flex items-center justify-between gap-2">
			<div class="flex items-center gap-2">
				<Dot state={lane.state === 'up' ? 'ok' : 'err'} />
				<span class="bn-text text-[15px] font-semibold tracking-[-0.01em]">Ollama</span>
			</div>
			<Badge tone={lane.plan ? 'accent' : 'default'}>{lane.plan ? `${lane.plan} · active` : lane.state === 'up' ? 'plan unknown' : 'server down'}</Badge>
		</div>
		<p class="bn-dim mt-4 text-[12px] leading-relaxed">no official gauge · ollama.com exposes plan % only on its settings page</p>
		<div class="mt-4 grid grid-cols-4 gap-2">
			{#each WINDOWS as w (w)}
				{@const r = lane.requests?.[w]}
				<div class="bn-border border-l pl-2">
					<div class="bn-text text-[19px] font-semibold tabular-nums tracking-[-0.02em]">{r ? r.chat + r.embed : '—'}</div>
					<div class="bn-dim font-mono text-[9.5px] uppercase tracking-[0.16em]">{WINDOW_LABEL[w]}</div>
					<div class="bn-muted font-mono text-[9.5px]">{r ? `${r.chat} chat · ${r.embed} emb` : 'no log'}</div>
				</div>
			{/each}
		</div>
		<div class="bn-dim mt-4 flex items-center justify-between font-mono text-[10px]">
			<span>{lane.models.length} models · {cloud} cloud (bill the plan)</span>
			<span>{selected ? 'breakdown ▾' : 'breakdown ▸'}</span>
		</div>
	</button>
{/if}
