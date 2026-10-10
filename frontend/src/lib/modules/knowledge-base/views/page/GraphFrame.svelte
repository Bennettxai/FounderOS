<script lang="ts">
	import { browser } from '$app/environment';

	interface Props {
		src?: string;
	}

	let { src = 'http://localhost:4201/graph' }: Props = $props();
</script>

<section class="kb-graph">
	{#if browser}
		<iframe
			{src}
			title="Knowledge Graph"
			class="kb-graph__iframe"
			allow="clipboard-read; clipboard-write"
		></iframe>
	{/if}
</section>

<style>
	.kb-graph {
		width: 100%;
		height: 100%;
		min-height: 0;
		overflow: hidden;
		background: #0d0d0d;
	}

	.kb-graph__iframe {
		width: 100%;
		height: 100%;
		border: none;
		background: #0d0d0d;
	}
</style>
