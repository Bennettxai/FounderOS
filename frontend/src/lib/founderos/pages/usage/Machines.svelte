<!-- "machines reporting": every machine that pushed this plan, stale ones flagged. -->
<script lang="ts">
	import { Dot, Label } from '$lib/founderos/kit';
	import { age, type PlanMachine } from './usage';

	let { machines, now }: { machines: PlanMachine[]; now: number } = $props();
</script>

<div>
	<Label>machines reporting</Label>
	<div class="mt-2 space-y-1.5">
		{#each machines as m (m.id)}
			<div class="flex items-center justify-between text-[12.5px]">
				<span class="flex items-center gap-2"><Dot state={m.stale ? 'warn' : 'ok'} />{m.label}</span>
				<span class="bn-dim font-mono text-[10.5px]"
					>{m.source === 'local' ? 'this box · live' : `pushed ${age(m.capturedAt, now)}${m.stale ? ' · stale' : ''}`}</span
				>
			</div>
		{/each}
		{#if machines.length === 0}<p class="bn-dim text-[12.5px]">no machine reporting</p>{/if}
	</div>
</div>
