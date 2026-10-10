<script lang="ts">
	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface Props {
		status: Status | null | undefined;
		showLabel?: boolean;
		size?: 'sm' | 'md';
	}

	let { status, showLabel = true, size = 'sm' }: Props = $props();

	const statusLabels: Record<Status, string> = {
		available: 'Available',
		busy: 'Busy',
		overloaded: 'Overloaded',
		ooo: 'Out of Office'
	};

	const normalizedStatus = $derived(status && statusLabels[status] ? status : 'available');
	const label = $derived(statusLabels[normalizedStatus]);
</script>

{#if showLabel}
	<span class="td-status-badge td-status-badge--{normalizedStatus} {size === 'md' ? 'td-status-badge--md' : ''}">
		<span class="td-status-badge__dot td-status-badge__dot--{normalizedStatus}"></span>
		{label}
	</span>
{:else}
	<span class="td-status-dot td-status-dot--{normalizedStatus}" title={label}></span>
{/if}

<style>
	.td-status-badge {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 22px;
		padding: 0 9px 0 7px;
		border-radius: 9999px;
		font-size: 11px;
		font-weight: 600;
		white-space: nowrap;
	}
	.td-status-badge--md {
		height: 26px;
		font-size: 13px;
		padding: 0 11px 0 9px;
	}
	.td-status-badge__dot {
		width: 6px;
		height: 6px;
		border-radius: 9999px;
		flex-shrink: 0;
	}
	.td-status-dot {
		width: 9px;
		height: 9px;
		border-radius: 9999px;
		border: 2px solid var(--dbg, #fff);
		display: block;
		flex-shrink: 0;
	}

	/* Status colors */
	.td-status-badge--available { background: #f0fdf4; color: #15803d; }
	.td-status-badge__dot--available { background: #22c55e; }
	.td-status-dot--available { background: #22c55e; }

	.td-status-badge--busy { background: #fefce8; color: #a16207; }
	.td-status-badge__dot--busy { background: #f59e0b; }
	.td-status-dot--busy { background: #f59e0b; }

	.td-status-badge--overloaded { background: #fef2f2; color: #b91c1c; }
	.td-status-badge__dot--overloaded { background: #ef4444; }
	.td-status-dot--overloaded { background: #ef4444; }

	.td-status-badge--ooo { background: var(--dbg2, #f5f5f5); color: var(--dt3, #888); }
	.td-status-badge__dot--ooo { background: #9ca3af; }
	.td-status-dot--ooo { background: #9ca3af; }

	:global(.dark) .td-status-dot { border-color: var(--dbg); }
</style>
