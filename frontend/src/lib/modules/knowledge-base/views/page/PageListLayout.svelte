<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		meta?: string | null;
		children: Snippet;
	}

	let { title, meta = null, children }: Props = $props();
</script>

<section class="kb-list">
	<header class="kb-list__header">
		<h1 class="kb-list__title">{title}</h1>
		{#if meta}
			<span class="kb-list__meta">{meta}</span>
		{/if}
	</header>
	{@render children()}
</section>

<style>
	.kb-list {
		flex: 1;
		display: flex;
		flex-direction: column;
		max-width: 900px;
		width: 100%;
		margin: 0 auto;
		padding: 40px 32px;
	}

	.kb-list__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 32px;
		gap: 16px;
	}

	.kb-list__title {
		font-size: 24px;
		font-weight: 600;
		color: var(--bos-v2-text-primary, #121212);
		margin: 0;
	}

	.kb-list__meta {
		color: var(--dt3, rgba(0,0,0,0.45));
		font-size: 13px;
		white-space: nowrap;
	}
</style>
