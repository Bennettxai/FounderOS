<!-- The canvas's fullscreen toggle (FounderOS v1 FunnelSpace / FunnelRadial:
     "the space is built to be filmed"). Fullscreens the given root element;
     the label follows the document's fullscreen state. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { Maximize2, Minimize2 } from '$lib/founderos/icons';

	let { target }: { target: () => HTMLElement | undefined } = $props();
	let isFullscreen = $state(false);

	onMount(() => {
		const onChange = () => (isFullscreen = Boolean(document.fullscreenElement));
		document.addEventListener('fullscreenchange', onChange);
		return () => document.removeEventListener('fullscreenchange', onChange);
	});
</script>

<button
	type="button"
	onclick={() => {
		if (document.fullscreenElement) void document.exitFullscreen?.();
		else void target()?.requestFullscreen?.();
	}}
	aria-label={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
	data-lens="c"
	class="fn-fullscreen bn-pressable absolute right-1 top-1 z-20 rounded-[var(--bn-r-chip)] border p-1.5"
>
	{#if isFullscreen}<Minimize2 size={14} />{:else}<Maximize2 size={14} />{/if}
</button>
