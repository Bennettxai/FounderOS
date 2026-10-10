<script lang="ts">
	interface Props {
		variant?: 'text' | 'card' | 'avatar' | 'button';
		count?: number;
		class?: string;
	}

	let { variant = 'text', count = 1, class: className = '' }: Props = $props();

	const items = $derived(Array.from({ length: count }, (_, i) => i));
</script>

<div class="bos-skeleton-loader {className}">
	{#each items as item (item)}
		{#if variant === 'text'}
			<div class="bos-skeleton-loader__bar"></div>
		{:else if variant === 'card'}
			<div class="bos-skeleton-loader__card">
				<div class="bos-skeleton-loader__bar bos-skeleton-loader__bar--w75"></div>
				<div class="bos-skeleton-loader__bar bos-skeleton-loader__bar--w50"></div>
				<div class="bos-skeleton-loader__bar bos-skeleton-loader__bar--w83"></div>
			</div>
		{:else if variant === 'avatar'}
			<div class="bos-skeleton-loader__avatar"></div>
		{:else if variant === 'button'}
			<div class="bos-skeleton-loader__button"></div>
		{/if}
	{/each}
</div>

<style>
	.bos-skeleton-loader {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.bos-skeleton-loader__bar {
		height: 16px;
		border-radius: 4px;
		background-color: var(--dbg2);
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	.bos-skeleton-loader__bar--w75 { width: 75%; }
	.bos-skeleton-loader__bar--w50 { width: 50%; }
	.bos-skeleton-loader__bar--w83 { width: 83.333%; }

	.bos-skeleton-loader__card {
		border: 1px solid var(--dbd2);
		border-radius: 8px;
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.bos-skeleton-loader__avatar {
		height: 40px;
		width: 40px;
		border-radius: 9999px;
		background-color: var(--dbg2);
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	.bos-skeleton-loader__button {
		height: 40px;
		width: 96px;
		border-radius: 6px;
		background-color: var(--dbg2);
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}
</style>
