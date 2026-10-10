<!-- Copy a lead magnet's live URL; on failure it says so rather than pretending. -->
<script lang="ts">
	import { Check, Copy } from '$lib/founderos/icons';

	let { url }: { url: string } = $props();
	let state = $state<'idle' | 'ok' | 'err'>('idle');

	async function copy() {
		try {
			await navigator.clipboard.writeText(url);
			state = 'ok';
		} catch {
			state = 'err';
		}
		setTimeout(() => (state = 'idle'), 1800);
	}
</script>

<button type="button" onclick={copy} title={`Copy ${url}`} data-lens="c"
	class="cl-btn bn-pressable inline-flex items-center gap-1.5 rounded-[5px] border px-2 py-1 font-mono text-[10px]"
>
	{#if state === 'ok'}
		<Check size={12} class="shrink-0" style="color: var(--bn-ok)" /> copied
	{:else if state === 'err'}
		<Copy size={12} class="shrink-0" style="color: var(--bn-err)" /> failed
	{:else}
		<Copy size={12} class="shrink-0" /> copy
	{/if}
</button>

<style>
	.cl-btn {
		border-color: var(--bn-border);
		background: var(--bn-bg);
		color: var(--bn-text-2);
	}
	.cl-btn:hover {
		border-color: var(--bn-text-3);
		color: var(--bn-text);
	}
</style>
