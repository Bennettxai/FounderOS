<script lang="ts">
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	interface Assignee {
		id: string;
		name: string;
		avatar?: string;
	}

	interface Props {
		id: string;
		title: string;
		priority: Priority;
		projectName?: string;
		projectColor?: string;
		assignee?: Assignee;
		dueDate?: string;
		subtaskCount?: number;
		subtaskCompleted?: number;
		commentCount?: number;
		onClick?: () => void;
	}

	let {
		id,
		title,
		priority,
		projectName,
		projectColor = 'var(--status-todo)',
		assignee,
		dueDate,
		subtaskCount = 0,
		subtaskCompleted = 0,
		commentCount = 0,
		onClick
	}: Props = $props();

	// Priority stripe colors (#59)
	const priorityStripeColor: Record<Priority, string | null> = {
		critical: 'var(--priority-critical)',
		high: 'var(--priority-high)',
		medium: 'var(--priority-medium)',
		low: 'var(--dbd2)'
	};

	const stripeColor = priorityStripeColor[priority] ?? null;

	function formatDueDate(dateStr: string) {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = date.getTime() - now.getTime();
		const days = Math.ceil(diff / (1000 * 60 * 60 * 24));

		if (days < 0) return { text: `${Math.abs(days)}d overdue`, isOverdue: true };
		if (days === 0) return { text: 'Today', isOverdue: false };
		if (days === 1) return { text: 'Tomorrow', isOverdue: false };
		return { text: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }), isOverdue: false };
	}

	const dueDateInfo = $derived(dueDate ? formatDueDate(dueDate) : null);

	const hasHoverContent = $derived(
		(subtaskCount > 0) || (commentCount > 0) || !!assignee
	);
</script>

<button
	onclick={onClick}
	class="board-card group"
	style={stripeColor ? `border-left: 3px solid ${stripeColor};` : 'border-left: 3px solid transparent;'}
>
	<!-- Title (#60 — bold, max 2 lines) -->
	<h4 class="board-card__title">
		{title}
	</h4>

	<!-- Project (#60 — small with colored dot) -->
	{#if projectName}
		<div class="board-card__project">
			<span class="board-card__project-dot" style="background-color: {projectColor};"></span>
			<span class="board-card__project-name">{projectName}</span>
		</div>
	{/if}

	<!-- Due Date (#60 — keep visible) -->
	{#if dueDateInfo}
		<div class="board-card__due" class:board-card__due--overdue={dueDateInfo.isOverdue}>
			<svg class="board-card__due-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
			</svg>
			{dueDateInfo.text}
		</div>
	{/if}

	<!-- Hover-reveal footer (#60 — subtask count + assignee avatar, opacity transition) -->
	{#if hasHoverContent}
		<div class="board-card__hover-footer">
			<!-- Subtasks & Comments -->
			<div class="board-card__meta">
				{#if subtaskCount > 0}
					<span class="board-card__meta-item" aria-label="{subtaskCompleted} of {subtaskCount} subtasks completed">
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
						</svg>
						{subtaskCompleted}/{subtaskCount}
					</span>
				{/if}
				{#if commentCount > 0}
					<span class="board-card__meta-item" aria-label="{commentCount} comments">
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
						</svg>
						{commentCount}
					</span>
				{/if}
			</div>

			<!-- Assignee -->
			{#if assignee}
				{#if assignee.avatar}
					<img
						src={assignee.avatar}
						alt={assignee.name}
						title={assignee.name}
						class="board-card__avatar"
					/>
				{:else}
					<div class="board-card__avatar board-card__avatar--initials" title={assignee.name} aria-label={assignee.name}>
						{assignee.name.charAt(0).toUpperCase()}
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</button>

<style>
	/* #58 — proper card, no button-pill appearance */
	.board-card {
		display: block;
		width: 100%;
		text-align: left;
		background: var(--dbg);
		border: 1px solid var(--dbd2, #ebebeb);
		border-radius: 0.5rem;
		padding: 0.75rem;
		cursor: pointer;
		transition:
			box-shadow 0.15s ease-out,
			border-color 0.15s ease-out;
		/* left border set inline per priority (#59) */
	}
	.board-card:hover {
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
		border-color: var(--dbd, #d4d4d4);
	}
	.board-card:focus-visible {
		outline: 2px solid color-mix(in srgb, var(--status-in-progress) 60%, transparent);
		outline-offset: 2px;
	}

	/* Title — bold, 2-line clamp (#60) */
	.board-card__title {
		font-weight: 600;
		font-size: 0.8125rem;
		line-height: 1.4;
		color: var(--dt, #111);
		margin-bottom: 0.375rem;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	/* Project chip (#60) */
	.board-card__project {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		margin-bottom: 0.375rem;
	}
	.board-card__project-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.board-card__project-name {
		font-size: 0.6875rem;
		color: var(--dt3, #888);
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}

	/* Due date (#60) */
	.board-card__due {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.6875rem;
		color: var(--dt3, #888);
		margin-bottom: 0;
	}
	.board-card__due--overdue {
		color: var(--priority-critical);
		font-weight: 500;
	}
	.board-card__due-icon {
		width: 0.75rem;
		height: 0.75rem;
		flex-shrink: 0;
	}

	/* Hover-reveal footer (#60 — opacity transition, no divider line) */
	.board-card__hover-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 0.5rem;
		opacity: 0;
		transition: opacity 0.15s ease-out;
	}
	.board-card:hover .board-card__hover-footer {
		opacity: 1;
	}
	@media (prefers-reduced-motion: reduce) {
		.board-card__hover-footer {
			opacity: 1;
		}
	}

	.board-card__meta {
		display: flex;
		align-items: center;
		gap: 0.625rem;
	}
	.board-card__meta-item {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.6875rem;
		color: var(--dt3, #888);
	}

	/* Assignee avatar */
	.board-card__avatar {
		width: 1.375rem;
		height: 1.375rem;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.board-card__avatar--initials {
		background: var(--dbg2, #f5f5f5);
		border: 1px solid var(--dbd, #e0e0e0);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.625rem;
		font-weight: 600;
		color: var(--dt2, #555);
	}
</style>
