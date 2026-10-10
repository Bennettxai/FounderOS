<!-- The Ollama plan's detail (UsageBoard.tsx OllamaDetail): requests per window, models, machines. -->
<script lang="ts">
	import { Badge, Label } from '$lib/founderos/kit';
	import Machines from './Machines.svelte';
	import { RAMP_1, RAMP_3, WINDOWS, WINDOW_LABEL, type OllamaBoard } from './usage';
	import { DETAIL, TRACK } from './ui';

	let { lane, now }: { lane: OllamaBoard; now: number } = $props();

	const max = $derived(Math.max(1, ...WINDOWS.map((w) => (lane.requests ? lane.requests[w].chat + lane.requests[w].embed : 0))));
</script>

<div data-part="ollama-detail" class={DETAIL}>
	<div>
		<Label>requests by window · chat vs embeddings</Label>
		<div class="mt-3 space-y-3">
			{#if !lane.requests}
				<p class="bn-dim text-[12.5px]">no reporting machine sent an Ollama server log</p>
			{:else}
				{#each WINDOWS as w (w)}
					{@const r = lane.requests[w]}
					<div class="grid grid-cols-[90px_1fr_120px] items-center gap-3 text-[12.5px]">
						<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.14em]">{WINDOW_LABEL[w]}</span>
						<div class="flex h-2 gap-[2px] overflow-hidden rounded-full" style={TRACK}>
							<div class="h-full" style="width: {(r.chat / max) * 100}%; background: {RAMP_1}" title="{r.chat} chat"></div>
							<div class="h-full" style="width: {(r.embed / max) * 100}%; background: {RAMP_3}" title="{r.embed} embeddings"></div>
						</div>
						<span class="text-right font-mono text-[11px] tabular-nums">{r.chat} chat · {r.embed} emb</span>
					</div>
				{/each}
			{/if}
		</div>
		{#if lane.note}<p class="bn-dim mt-4 font-mono text-[10px] leading-relaxed">{lane.note}</p>{/if}
	</div>
	<div>
		<Label>models</Label>
		<div class="mt-2 space-y-1.5">
			{#each lane.models as m (`${m.host}|${m.name}`)}
				<div class="flex items-center justify-between text-[12.5px]">
					<span class="bn-muted">{m.name}<span class="bn-dim ml-2 font-mono text-[10px]">{m.host.replace(/^Ollama · /, '')}</span></span>
					<Badge tone={m.cloud ? 'warn' : 'default'}>{m.cloud ? 'cloud · bills plan' : 'local · free'}</Badge>
				</div>
			{/each}
			{#if lane.models.length === 0}<p class="bn-dim text-[12.5px]">no models listed</p>{/if}
		</div>
		<div class="mt-5"><Machines machines={lane.machines} {now} /></div>
	</div>
</div>
