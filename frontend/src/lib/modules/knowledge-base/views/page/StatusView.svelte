<script lang="ts">
	interface Props {
		state: 'loading' | 'error';
		message?: string | null;
		onRetry?: () => void;
	}

	let { state, message = null, onRetry }: Props = $props();
</script>

<div class="kb-status" role={state === 'error' ? 'alert' : 'status'}>
	{#if state === 'loading'}
		<div class="kb-status__spinner"></div>
		<p class="kb-status__text">{message ?? 'Loading documents...'}</p>
	{:else}
		<p class="kb-status__text kb-status__text--error">{message ?? 'Something went wrong'}</p>
		<button class="kb-status__btn" aria-label="Retry loading documents" onclick={onRetry}>Retry</button>
	{/if}
</div>

<style>
	.kb-status {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 16px;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.kb-status__spinner {
		width: 28px;
		height: 28px;
		border: 3px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		border-top-color: var(--bos-brand-color, #1e96eb);
		border-radius: 50%;
		animation: kb-spin 0.8s linear infinite;
	}

	@keyframes kb-spin {
		to { transform: rotate(360deg); }
	}

	.kb-status__text {
		font-size: 14px;
		margin: 0;
	}

	.kb-status__text--error {
		color: #ef4444;
	}

	.kb-status__btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 32px;
		padding: 0 14px;
		font-size: 13px;
		font-weight: 500;
		border-radius: 8px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		color: var(--bos-v2-text-primary, #121212);
		cursor: pointer;
	}

	.kb-status__btn:hover {
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
	}
</style>
