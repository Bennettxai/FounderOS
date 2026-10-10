<script lang="ts">
	import type { MemoryListItem } from '$lib/api/memory';

	interface Props {
		memories: MemoryListItem[];
	}

	let { memories }: Props = $props();

	const stats = $derived.by(() => {
		const total = memories.length;
		const pinned = memories.filter(m => m.is_pinned).length;
		const avgImportance = memories.length > 0
			? memories.reduce((sum, m) => sum + m.importance_score, 0) / memories.length
			: 0;

		const byType = memories.reduce((acc, m) => {
			acc[m.memory_type] = (acc[m.memory_type] || 0) + 1;
			return acc;
		}, {} as Record<string, number>);

		const mostCommonType = Object.entries(byType)
			.sort(([, a], [, b]) => b - a)[0]?.[0] || 'none';

		const totalAccess = memories.reduce((sum, m) => sum + (m.access_count || 0), 0);

		return { total, pinned, avgImportance, byType, mostCommonType, totalAccess };
	});
</script>

<div class="memory-stats">
	<div class="stat-card">
		<div class="stat-icon stat-icon--blue">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="18" height="18">
				<path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-value">{stats.total}</div>
			<div class="stat-label">Total Memories</div>
		</div>
	</div>

	<div class="stat-card">
		<div class="stat-icon stat-icon--purple">
			<svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="18" height="18">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9" />
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-value">{stats.pinned}</div>
			<div class="stat-label">Pinned</div>
		</div>
	</div>

	<div class="stat-card">
		<div class="stat-icon stat-icon--green">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="18" height="18">
				<path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" />
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-value">{(stats.avgImportance * 100).toFixed(0)}%</div>
			<div class="stat-label">Avg Importance</div>
		</div>
	</div>

	<div class="stat-card">
		<div class="stat-icon stat-icon--amber">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="18" height="18">
				<path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-value">{stats.totalAccess}</div>
			<div class="stat-label">Total Views</div>
		</div>
	</div>
</div>

<style>
	.memory-stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 8px;
		padding: 12px;
		border-bottom: 1px solid var(--color-border);
	}

	:global(.dark) .memory-stats {
		border-bottom-color: rgba(255, 255, 255, 0.06);
	}

	.stat-card {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 8px;
	}

	:global(.dark) .stat-card {
		background: var(--dbg2);
		border-color: rgba(255, 255, 255, 0.08);
	}

	.stat-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border-radius: 8px;
		flex-shrink: 0;
	}

	.stat-icon--blue { background: color-mix(in srgb, var(--bos-status-info, #3b82f6) 10%, transparent); color: var(--bos-status-info, #3b82f6); }
	.stat-icon--purple { background: color-mix(in srgb, #8b5cf6 10%, transparent); color: #8b5cf6; }
	.stat-icon--green { background: color-mix(in srgb, var(--bos-status-success, #22c55e) 10%, transparent); color: var(--bos-status-success, #22c55e); }
	.stat-icon--amber { background: color-mix(in srgb, var(--bos-priority-medium, #f59e0b) 10%, transparent); color: var(--bos-priority-medium, #f59e0b); }

	:global(.dark) .stat-icon--blue { background: color-mix(in srgb, var(--bos-status-info, #3b82f6) 15%, transparent); color: var(--bos-status-info, #60a5fa); }
	:global(.dark) .stat-icon--purple { background: color-mix(in srgb, #8b5cf6 15%, transparent); color: #a78bfa; }
	:global(.dark) .stat-icon--green { background: color-mix(in srgb, var(--bos-status-success, #22c55e) 15%, transparent); color: var(--bos-status-success, #4ade80); }
	:global(.dark) .stat-icon--amber { background: color-mix(in srgb, var(--bos-priority-medium, #f59e0b) 15%, transparent); color: var(--bos-priority-medium, #fbbf24); }

	.stat-content {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.stat-value {
		font-size: 18px;
		font-weight: 700;
		color: var(--color-text);
		line-height: 1;
	}

	:global(.dark) .stat-value {
		color: var(--dt);
	}

	.stat-label {
		font-size: 10px;
		font-weight: 500;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.3px;
	}

	:global(.dark) .stat-label {
		color: var(--dt3);
	}

	@media (max-width: 600px) {
		.memory-stats {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>
