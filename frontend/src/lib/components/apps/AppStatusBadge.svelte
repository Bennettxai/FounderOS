<!--
	AppStatusBadge.svelte
	Status indicator badge for apps

	Statuses:
	- active: No badge (default state)
	- draft: Gray badge
	- generating: Blue badge with spinner
	- error: Red badge
	- archived: Gray badge with lower opacity
-->
<script lang="ts">
	import type { AppStatus } from '$lib/types/apps';

	interface Props {
		status: AppStatus;
		class?: string;
	}

	let { status, class: className = '' }: Props = $props();

	const statusConfig: Record<AppStatus, { label: string; classes: string; showSpinner?: boolean }> = {
		active: { label: '', classes: '' }, // No badge for active
		draft: {
			label: 'DRAFT',
			classes: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
		},
		generating: {
			label: 'GENERATING',
			classes: 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400',
			showSpinner: true
		},
		error: {
			label: 'ERROR',
			classes: 'bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-400'
		},
		archived: {
			label: 'ARCHIVED',
			classes: 'bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-500'
		}
	};

	const config = $derived(statusConfig[status]);
</script>

{#if status !== 'active' && config.label}
	<span class="app-status-badge {config.classes} {className}">
		{#if config.showSpinner}
			<svg class="spinner" viewBox="0 0 24 24" fill="none">
				<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" opacity="0.25" />
				<path
					d="M12 2a10 10 0 0 1 10 10"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
				/>
			</svg>
		{/if}
		{config.label}
	</span>
{/if}

<style>
	.app-status-badge {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px;
		font-size: 11px;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.025em;
		border-radius: 4px;
		white-space: nowrap;
	}

	.spinner {
		width: 12px;
		height: 12px;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
