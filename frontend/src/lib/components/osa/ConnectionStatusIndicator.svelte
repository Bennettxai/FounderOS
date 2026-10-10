<script lang="ts">
	import { Loader2, RefreshCw, Clock } from 'lucide-svelte';

	interface Props {
		status: 'connected' | 'reconnecting' | 'failed' | 'timeout';
		reconnectAttempt?: number;
		maxReconnectAttempts?: number;
		onRetry?: () => void;
		compact?: boolean;
	}

	let {
		status,
		reconnectAttempt = 0,
		maxReconnectAttempts = 5,
		onRetry,
		compact = false
	}: Props = $props();

	let isRetrying = $state(false);
	let retryResetTimeout: ReturnType<typeof setTimeout> | null = null;

	// Reset isRetrying when status changes away from 'failed'
	$effect(() => {
		if (status !== 'failed') {
			isRetrying = false;
			if (retryResetTimeout) {
				clearTimeout(retryResetTimeout);
				retryResetTimeout = null;
			}
		}
	});

	function handleRetryClick() {
		if (isRetrying || !onRetry) return;
		isRetrying = true;
		onRetry();

		// Auto-reset after 3 seconds if status hasn't changed
		retryResetTimeout = setTimeout(() => {
			isRetrying = false;
			retryResetTimeout = null;
		}, 3000);
	}
</script>

{#if status === 'connected'}
	<!-- Connected: green dot + "Live" -->
	<div class="flex items-center gap-2 text-green-500 text-sm">
		<span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
		<span>Live</span>
	</div>
{:else if status === 'reconnecting'}
	<!-- Reconnecting: amber spinner + attempt counter -->
	<div class="flex items-center gap-2 text-yellow-500 text-sm">
		<Loader2 class="w-4 h-4 animate-spin" />
		<span>
			Reconnecting...
			{#if reconnectAttempt > 0}
				<span class="text-gray-400">({reconnectAttempt}/{maxReconnectAttempts})</span>
			{/if}
		</span>
	</div>
{:else if status === 'failed'}
	<!-- Failed: red dot + "Disconnected" + optional retry button -->
	{#if compact}
		<div class="flex items-center gap-2 text-sm">
			<span class="w-2 h-2 bg-red-500 rounded-full"></span>
			<span class="text-red-500">Disconnected</span>
			{#if onRetry}
				<button
					onclick={handleRetryClick}
					disabled={isRetrying}
					aria-label="Retry connection"
					class="btn-pill btn-pill-danger btn-pill-xs flex items-center gap-1"
				>
					<RefreshCw class="w-3 h-3 {isRetrying ? 'animate-spin' : ''}" />
					{isRetrying ? 'Retrying...' : 'Retry'}
				</button>
			{/if}
		</div>
	{:else}
		<div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 flex items-start gap-3">
			<span class="w-2.5 h-2.5 bg-red-500 rounded-full flex-shrink-0 mt-1.5"></span>
			<div class="flex-1">
				<p class="text-sm font-medium text-red-800 dark:text-red-200">
					Disconnected
				</p>
				<p class="text-xs text-red-700 dark:text-red-300 mt-1">
					Connection to the generation server was lost.
				</p>
				{#if onRetry}
					<button
						onclick={handleRetryClick}
						disabled={isRetrying}
						aria-label="Retry connection"
						class="btn-pill btn-pill-secondary btn-pill-xs mt-2 flex items-center gap-1.5"
					>
						<RefreshCw class="w-3.5 h-3.5 {isRetrying ? 'animate-spin' : ''}" />
						{isRetrying ? 'Retrying...' : 'Retry Connection'}
					</button>
				{/if}
			</div>
		</div>
	{/if}
{:else if status === 'timeout'}
	<!-- Timeout: amber warning + "Still working..." -->
	{#if compact}
		<div class="flex items-center gap-2 text-yellow-500 text-sm">
			<Clock class="w-4 h-4" />
			<span>Still processing...</span>
		</div>
	{:else}
		<div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3 flex items-start gap-3">
			<Clock class="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
			<div>
				<p class="text-sm font-medium text-yellow-800 dark:text-yellow-200">
					Still working...
				</p>
				<p class="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
					The AI agents may need extra time for complex operations. This is normal for larger applications.
				</p>
			</div>
		</div>
	{/if}
{/if}
