<script lang="ts">
	export interface DelegatedTask {
		id: string;
		title: string;
		status: 'pending' | 'working' | 'done' | 'failed';
		conversationId: string;
		conversationTitle: string;
		agent: string;
		createdAt: string;
		progress?: number;
	}

	interface Props {
		tasks: DelegatedTask[];
		onTaskClick?: (task: DelegatedTask) => void;
	}

	let { tasks = [], onTaskClick }: Props = $props();

	function getStatusColor(status: string): string {
		switch (status) {
			case 'pending': return '#f59e0b';
			case 'working': return '#3b82f6';
			case 'done': return '#22c55e';
			case 'failed': return '#ef4444';
			default: return '#6b7280';
		}
	}

	function getStatusLabel(status: string): string {
		switch (status) {
			case 'pending': return 'Pending';
			case 'working': return 'Working';
			case 'done': return 'Done';
			case 'failed': return 'Failed';
			default: return status;
		}
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();

		if (diff < 60000) return 'Just now';
		if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
		if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
		return date.toLocaleDateString();
	}

	let activeTasks = $derived(tasks.filter(t => t.status !== 'done'));
	let completedTasks = $derived(tasks.filter(t => t.status === 'done'));
</script>

<div class="progress-panel">
	<div class="panel-header">
		<h3 class="panel-title">Progress</h3>
		{#if tasks.length > 0}
			<span class="task-count">{activeTasks.length} active</span>
		{/if}
	</div>

	<div class="panel-content">
		{#if tasks.length === 0}
			<div class="empty-state">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="empty-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
				</svg>
				<p class="empty-text">Progress will show up here once you delegate a task</p>
			</div>
		{:else}
			<!-- Active Tasks -->
			{#if activeTasks.length > 0}
				<div class="task-section">
					<div class="section-label">Active</div>
					{#each activeTasks as task (task.id)}
						<button
							class="btn-pill btn-pill-ghost task-item"
							onclick={() => onTaskClick?.(task)}
						>
							<div class="task-status-indicator" style="background-color: {getStatusColor(task.status)}">
								{#if task.status === 'working'}
									<div class="pulse-ring"></div>
								{/if}
							</div>
							<div class="task-info">
								<span class="task-title">{task.title}</span>
								<span class="task-meta">
									{task.agent} · {formatTime(task.createdAt)}
								</span>
							</div>
							<span class="task-status-badge" style="color: {getStatusColor(task.status)}">
								{getStatusLabel(task.status)}
							</span>
						</button>
					{/each}
				</div>
			{/if}

			<!-- Completed Tasks -->
			{#if completedTasks.length > 0}
				<div class="task-section">
					<div class="section-label">Completed</div>
					{#each completedTasks.slice(0, 5) as task (task.id)}
						<button
							class="btn-pill btn-pill-ghost task-item completed"
							onclick={() => onTaskClick?.(task)}
						>
							<div class="task-status-indicator" style="background-color: {getStatusColor(task.status)}">
								<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" width="10" height="10">
									<path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 0 1 .143 1.052l-8 10.5a.75.75 0 0 1-1.127.075l-4.5-4.5a.75.75 0 0 1 1.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 0 1 1.05-.143Z" clip-rule="evenodd" />
								</svg>
							</div>
							<div class="task-info">
								<span class="task-title">{task.title}</span>
								<span class="task-meta">{task.conversationTitle}</span>
							</div>
						</button>
					{/each}
				</div>
			{/if}
		{/if}
	</div>
</div>

<style>
	.progress-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--color-border);
	}

	:global(.dark) .panel-header {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	.panel-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0;
	}

	:global(.dark) .panel-title {
		color: #f5f5f7;
	}

	.task-count {
		font-size: 12px;
		color: var(--color-text-muted);
		background: var(--color-bg-tertiary);
		padding: 2px 8px;
		border-radius: 10px;
	}

	:global(.dark) .task-count {
		background: #3a3a3c;
		color: #a1a1a6;
	}

	.panel-content {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 32px 16px;
		text-align: center;
	}

	.empty-icon {
		width: 40px;
		height: 40px;
		color: var(--color-text-muted);
		margin-bottom: 12px;
	}

	:global(.dark) .empty-icon {
		color: #6e6e73;
	}

	.empty-text {
		font-size: 13px;
		color: var(--color-text-muted);
		line-height: 1.5;
		margin: 0;
	}

	:global(.dark) .empty-text {
		color: #6e6e73;
	}

	.task-section {
		margin-bottom: 16px;
	}

	.section-label {
		font-size: 11px;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.5px;
		padding: 8px 8px 4px;
	}

	:global(.dark) .section-label {
		color: #6e6e73;
	}

	.task-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 10px 8px;
		background: transparent;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s ease;
	}

	.task-item:hover {
		background: var(--color-bg-secondary);
	}

	:global(.dark) .task-item:hover {
		background: #3a3a3c;
	}

	.task-item.completed {
		opacity: 0.7;
	}

	.task-status-indicator {
		position: relative;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.pulse-ring {
		position: absolute;
		inset: -3px;
		border-radius: 50%;
		border: 2px solid currentColor;
		opacity: 0.3;
		animation: pulse 1.5s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { transform: scale(1); opacity: 0.3; }
		50% { transform: scale(1.2); opacity: 0; }
	}

	.task-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.task-title {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .task-title {
		color: #f5f5f7;
	}

	.task-meta {
		font-size: 11px;
		color: var(--color-text-muted);
	}

	:global(.dark) .task-meta {
		color: #6e6e73;
	}

	.task-status-badge {
		font-size: 11px;
		font-weight: 500;
		flex-shrink: 0;
	}
</style>
