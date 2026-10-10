<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import type { Tone } from './format';

	let { tone, children }: { tone?: Tone; children?: Snippet } = $props();
</script>

<span class="bn-chip inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 font-mono text-[11.5px] tabular-nums">
	{#if tone}<span class="bn-chip-dot h-1.5 w-1.5 rounded-full" data-tone={tone}></span>{/if}
	{@render children?.()}
</span>
