<script lang="ts">
	import { Loader2 } from 'lucide-svelte';

	interface Props {
		size?: 'sm' | 'md' | 'lg';
		message?: string;
		fullscreen?: boolean;
	}

	let { size = 'md', message, fullscreen = false }: Props = $props();
</script>

<div class="bos-loading-spinner" class:bos-loading-spinner--fullscreen={fullscreen}>
	<Loader2 class="bos-loading-spinner__icon bos-loading-spinner__icon--{size}" />
	{#if message}
		<span class="bos-loading-spinner__message">{message}</span>
	{/if}
</div>

<style>
	.bos-loading-spinner {
		display: inline-flex;
		align-items: center;
		gap: 8px;
	}

	.bos-loading-spinner--fullscreen {
		display: flex;
		justify-content: center;
		min-height: 400px;
	}

	:global(.bos-loading-spinner__icon) {
		animation: spin 1s linear infinite;
		color: var(--color-primary, #6366f1);
	}

	:global(.bos-loading-spinner__icon--sm) { width: 16px; height: 16px; }
	:global(.bos-loading-spinner__icon--md) { width: 32px; height: 32px; }
	:global(.bos-loading-spinner__icon--lg) { width: 48px; height: 48px; }

	.bos-loading-spinner__message {
		font-size: 13px;
		color: var(--dt3);
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
