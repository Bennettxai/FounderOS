<script lang="ts">
	import { AlertCircle, RefreshCw } from 'lucide-svelte';

	interface Props {
		error?: Error | string | null;
		onRetry?: () => void;
		children?: import('svelte').Snippet;
	}

	let { error = null, onRetry, children }: Props = $props();

	const errorMessage = $derived(() => {
		if (!error) return '';
		if (typeof error === 'string') return error;
		return error.message || 'An unexpected error occurred';
	});
</script>

{#if error}
	<div class="bos-error-boundary">
		<div class="bos-error-boundary__card">
			<div class="bos-error-boundary__layout">
				<div class="bos-error-boundary__icon-wrap">
					<AlertCircle class="bos-error-boundary__icon" />
				</div>
				<div class="bos-error-boundary__body">
					<h3 class="bos-error-boundary__title">
						Something went wrong
					</h3>
					<p class="bos-error-boundary__message">
						{errorMessage()}
					</p>
					{#if onRetry}
						<button
							onclick={onRetry}
							class="btn-rounded btn-rounded-danger btn-rounded-sm"
							style="display: inline-flex; align-items: center; gap: 8px;"
						>
							<RefreshCw style="width: 16px; height: 16px;" />
							Try Again
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>
{:else}
	{@render children?.()}
{/if}

<style>
	.bos-error-boundary {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 400px;
		padding: 32px;
	}

	.bos-error-boundary__card {
		max-width: 448px;
		width: 100%;
		background-color: rgba(235, 67, 53, 0.05);
		border: 1px solid rgba(235, 67, 53, 0.2);
		border-radius: 8px;
		padding: 24px;
	}

	.bos-error-boundary__layout {
		display: flex;
		align-items: flex-start;
		gap: 16px;
	}

	.bos-error-boundary__icon-wrap {
		flex-shrink: 0;
	}

	:global(.bos-error-boundary__icon) {
		width: 24px;
		height: 24px;
		color: #eb4335;
	}

	.bos-error-boundary__body {
		flex: 1;
	}

	.bos-error-boundary__title {
		font-size: 18px;
		font-weight: 600;
		color: var(--dt);
		margin-bottom: 8px;
	}

	.bos-error-boundary__message {
		font-size: 13px;
		color: var(--dt2);
		margin-bottom: 16px;
	}

	:global(.dark) .bos-error-boundary__card {
		background-color: rgba(235, 67, 53, 0.08);
		border-color: rgba(235, 67, 53, 0.25);
	}
</style>
