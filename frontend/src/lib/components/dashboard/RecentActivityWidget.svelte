<script lang="ts">
	import { fly } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import WidgetWrapper from './WidgetWrapper.svelte';

	type ActivityType =
		| 'task_completed'
		| 'task_started'
		| 'project_created'
		| 'project_updated'
		| 'conversation'
		| 'team'
		| 'artifact';

	interface Activity {
		id: string;
		type: ActivityType;
		description: string;
		actorName?: string;
		actorAvatar?: string;
		targetId?: string;
		targetType?: string;
		createdAt: string;
	}

	interface Props {
		activities?: Activity[];
		isLoading?: boolean;
		onViewAll?: () => void;
	}

	let { activities = [], isLoading = false, onViewAll }: Props = $props();

	/** Derive the action variant key used for CSS class and color var lookup */
	function actionVariant(type: ActivityType): 'created' | 'updated' | 'commented' | 'completed' {
		if (type === 'task_completed') return 'completed';
		if (type === 'project_created') return 'created';
		if (type === 'project_updated' || type === 'task_started') return 'updated';
		return 'commented';
	}

	function formatRelativeTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(minutes / 60);
		const days = Math.floor(hours / 24);

		if (minutes < 1) return 'Just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		if (days === 1) return 'Yesterday';
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	function handleActivityClick(activity: Activity) {
		if (activity.targetId && activity.targetType) {
			switch (activity.targetType) {
				case 'project':
					goto(`/projects/${activity.targetId}`);
					break;
				case 'task':
					goto(`/tasks?id=${activity.targetId}`);
					break;
				case 'conversation':
					goto(`/chat?id=${activity.targetId}`);
					break;
			}
		}
	}

	const visibleActivities = $derived(activities.slice(0, 5));
</script>

<!-- Activity type icon snippets -->
{#snippet iconTaskCompleted()}
	<svg class="dw-feed-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
	</svg>
{/snippet}

{#snippet iconTaskStarted()}
	<svg class="dw-feed-icon" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path d="M8 5v14l11-7z" />
	</svg>
{/snippet}

{#snippet iconProjectCreated()}
	<svg class="dw-feed-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
	</svg>
{/snippet}

{#snippet iconProjectUpdated()}
	<svg class="dw-feed-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
	</svg>
{/snippet}

{#snippet iconConversation()}
	<svg class="dw-feed-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
	</svg>
{/snippet}

{#snippet iconTeam()}
	<svg class="dw-feed-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z" />
	</svg>
{/snippet}

{#snippet iconArtifact()}
	<svg class="dw-feed-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
	</svg>
{/snippet}

<WidgetWrapper>
	<!-- Header with count badge -->
	<div class="dw-feed-header">
		<div class="widget-header">
			<div class="widget-header-left">
				<span class="widget-icon-inline">
					<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</span>
				<h2 class="widget-title">Recent Activity</h2>
			</div>
		</div>
		{#if activities.length > 0}
			<span class="dw-feed-count-badge">{activities.length}</span>
		{/if}
	</div>

	{#if isLoading}
		<!-- Skeleton -->
		<div class="dw-feed-skeleton" aria-hidden="true">
			{#each [1, 2, 3, 4] as _, i}
				<div class="dw-feed-sk-item {i === 0 ? 'dw-feed-sk-item--first' : ''}">
					<div class="dw-feed-sk dw-feed-sk--avatar"></div>
					<div class="dw-feed-sk-content">
						<div class="dw-feed-sk dw-feed-sk--line dw-feed-sk--wide"></div>
						<div class="dw-feed-sk dw-feed-sk--line dw-feed-sk--narrow"></div>
					</div>
					<div class="dw-feed-sk dw-feed-sk--time"></div>
				</div>
			{/each}
		</div>

	{:else if activities.length === 0}
		<!-- Empty state -->
		<div class="dw-feed-empty">
			<svg class="dw-feed-empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<p class="dw-feed-empty-title">No recent activity</p>
			<button onclick={() => goto('/tasks')} class="dw-feed-empty-action">
				Add a task
			</button>
		</div>
	{:else}
		<!-- Activity feed list — max 5 items, no scroll -->
		<div class="dw-feed-list">
			{#each visibleActivities as activity, index (activity.id)}
				{@const variant = actionVariant(activity.type)}
				<button
					onclick={() => handleActivityClick(activity)}
					class="dw-feed-item"
					aria-label="View {activity.type.replace(/_/g, ' ')} activity"
					in:fly={{ x: -10, duration: 300, delay: index * 30 }}
				>
					<!-- Avatar: image > initials > type icon -->
					{#if activity.actorAvatar}
						<img
							src={activity.actorAvatar}
							alt=""
							class="dw-feed-avatar dw-feed-avatar--img"
						/>
					{:else if activity.actorName}
						<div class="dw-feed-avatar dw-feed-avatar--initials" aria-hidden="true">
							{activity.actorName.charAt(0).toUpperCase()}
						</div>
					{:else}
						<div class="dw-feed-avatar dw-feed-avatar--icon dw-feed-avatar--{variant}" aria-hidden="true">
							{#if activity.type === 'task_completed'}
								{@render iconTaskCompleted()}
							{:else if activity.type === 'task_started'}
								{@render iconTaskStarted()}
							{:else if activity.type === 'project_created'}
								{@render iconProjectCreated()}
							{:else if activity.type === 'project_updated'}
								{@render iconProjectUpdated()}
							{:else if activity.type === 'conversation'}
								{@render iconConversation()}
							{:else if activity.type === 'team'}
								{@render iconTeam()}
							{:else}
								{@render iconArtifact()}
							{/if}
						</div>
					{/if}

					<!-- Description + action badge -->
					<div class="dw-feed-content">
						<p class="dw-feed-text">
							{#if activity.actorName}
								<span class="dw-feed-actor">{activity.actorName}</span>{' '}
							{/if}
							{activity.description}
						</p>
						<span class="dw-feed-action dw-feed-action--{variant}">
							{variant}
						</span>
					</div>

					<!-- Relative timestamp -->
					<time class="dw-feed-time" datetime={activity.createdAt}>
						{formatRelativeTime(activity.createdAt)}
					</time>
				</button>
			{/each}
		</div>

		<!-- View all footer link -->
		{#if onViewAll}
			<button class="dw-feed-viewall" onclick={onViewAll} aria-label="View all activity">
				View all activity
				<svg class="dw-feed-viewall-arrow" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		{/if}
	{/if}
</WidgetWrapper>

<style>
	/* ─── Activity color tokens — CSS vars with hex fallbacks ─── */
	:root {
		--dw-action-created:   var(--color-success, #22c55e);
		--dw-action-updated:   var(--accent-blue, #3b82f6);
		--dw-action-commented: var(--accent-orange, #FF6B35);
		--dw-action-completed: var(--accent-purple, #A855F7);
	}

	/* ─── Header row (wraps WidgetHeader + count badge) ─── */
	.dw-feed-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.widget-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.widget-header-left {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.widget-icon-inline {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.widget-icon-inline :global(svg) {
		width: 1.1rem;
		height: 1.1rem;
		color: var(--bos-text-secondary, var(--dt2));
	}

	.widget-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--bos-text-primary);
		line-height: 1.25;
	}

	.dw-feed-count-badge {
		font-size: 0.7rem;
		font-weight: 700;
		color: var(--dt3);
		background: var(--dbg3);
		border: 1px solid var(--dbd2);
		border-radius: 999px;
		padding: 0.1rem 0.5rem;
		line-height: 1.6;
		white-space: nowrap;
		flex-shrink: 0;
	}

	/* ─── Feed list — column, no scroll ─── */
	.dw-feed-list {
		display: flex;
		flex-direction: column;
	}

	/* ─── Individual feed item ─── */
	.dw-feed-item {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 0.75rem 0;
		border-top: 1px solid var(--dbd2);
		transition: background 0.15s;
		text-align: left;
		background: none;
		border-left: none;
		border-right: none;
		border-bottom: none;
		cursor: pointer;
		width: 100%;
		font-family: inherit;
		color: inherit;
		border-radius: 0;
	}

	/* First item has no top border */
	.dw-feed-item:first-child {
		border-top: none;
		padding-top: 0;
	}

	.dw-feed-item:last-child {
		padding-bottom: 0;
	}

	.dw-feed-item:hover {
		background: var(--dbg2);
	}

	/* ─── Avatar ─── */
	.dw-feed-avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		font-weight: 700;
		flex-shrink: 0;
		object-fit: cover;
	}

	.dw-feed-avatar--img {
		/* inherits border-radius: 50% from base */
	}

	.dw-feed-avatar--initials {
		background: var(--dbg3);
		color: var(--dt2);
		border: 1px solid var(--dbd2);
	}

	/* Icon avatars use action variant colors */
	.dw-feed-avatar--icon {
		background: transparent;
	}

	.dw-feed-avatar--created {
		color: var(--dw-action-created);
		background: color-mix(in srgb, var(--dw-action-created) 12%, transparent);
	}

	.dw-feed-avatar--updated {
		color: var(--dw-action-updated);
		background: color-mix(in srgb, var(--dw-action-updated) 12%, transparent);
	}

	.dw-feed-avatar--commented {
		color: var(--dw-action-commented);
		background: color-mix(in srgb, var(--dw-action-commented) 12%, transparent);
	}

	.dw-feed-avatar--completed {
		color: var(--dw-action-completed);
		background: color-mix(in srgb, var(--dw-action-completed) 12%, transparent);
	}

	/* ─── Feed icon (SVG inside avatar) ─── */
	:global(.dw-feed-icon) {
		width: 1rem;
		height: 1rem;
	}

	/* ─── Content column ─── */
	.dw-feed-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		min-width: 0;
	}

	.dw-feed-text {
		font-size: 0.84rem;
		color: var(--dt);
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.dw-feed-actor {
		font-weight: 600;
		color: var(--dt);
	}

	/* ─── Action type badge ─── */
	.dw-feed-action {
		display: inline-flex;
		align-items: center;
		font-weight: 600;
		font-size: 0.7rem;
		text-transform: capitalize;
		padding: 1px 6px;
		border-radius: 4px;
		width: fit-content;
		line-height: 1.6;
	}

	.dw-feed-action--created {
		color: var(--dw-action-created);
		background: color-mix(in srgb, var(--dw-action-created) 10%, transparent);
	}

	.dw-feed-action--updated {
		color: var(--dw-action-updated);
		background: color-mix(in srgb, var(--dw-action-updated) 10%, transparent);
	}

	.dw-feed-action--commented {
		color: var(--dw-action-commented);
		background: color-mix(in srgb, var(--dw-action-commented) 10%, transparent);
	}

	.dw-feed-action--completed {
		color: var(--dw-action-completed);
		background: color-mix(in srgb, var(--dw-action-completed) 10%, transparent);
	}

	/* ─── Timestamp ─── */
	.dw-feed-time {
		font-size: 0.72rem;
		color: var(--dt4);
		white-space: nowrap;
		flex-shrink: 0;
		padding-top: 0.15rem;
	}

	/* ─── View all footer ─── */
	.dw-feed-viewall {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.25rem;
		width: 100%;
		margin-top: 0.75rem;
		padding: 0.4rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.82rem;
		color: var(--dt2);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
		font-family: inherit;
	}

	.dw-feed-viewall:hover {
		background: var(--dbg2);
		color: var(--dt);
	}

	.dw-feed-viewall-arrow {
		width: 0.75rem;
		height: 0.75rem;
	}

	/* ─── Skeleton ─── */
	.dw-feed-skeleton {
		display: flex;
		flex-direction: column;
	}

	.dw-feed-sk-item {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 0.75rem 0;
		border-top: 1px solid var(--dbd2);
	}

	.dw-feed-sk-item--first {
		border-top: none;
		padding-top: 0;
	}

	.dw-feed-sk-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		min-width: 0;
	}

	@keyframes dw-feed-pulse {
		50% { opacity: 0.5; }
	}

	.dw-feed-sk {
		background: var(--dbg3, color-mix(in srgb, var(--dt) 8%, transparent));
		animation: dw-feed-pulse 1.5s ease-in-out infinite;
		border-radius: 4px;
		flex-shrink: 0;
	}

	.dw-feed-sk--avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
	}

	.dw-feed-sk--line {
		height: 11px;
		border-radius: 4px;
	}

	.dw-feed-sk--wide {
		width: 75%;
	}

	.dw-feed-sk--narrow {
		width: 35%;
		height: 9px;
	}

	.dw-feed-sk--time {
		width: 32px;
		height: 9px;
		border-radius: 3px;
		margin-top: 2px;
	}

	/* ─── Empty state ─── */
	.dw-feed-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 2rem 1rem;
		gap: 0.75rem;
		text-align: center;
	}

	.dw-feed-empty-icon {
		width: 1.5rem;
		height: 1.5rem;
		color: var(--dt3);
		flex-shrink: 0;
	}

	.dw-feed-empty-title {
		font-size: 0.85rem;
		color: var(--dt2);
		margin: 0;
	}

	.dw-feed-empty-action {
		display: inline-flex;
		align-items: center;
		font-size: 0.8rem;
		color: var(--dt2);
		background: transparent;
		border: 1px solid var(--dbd2);
		border-radius: 6px;
		padding: 0.2rem 0.75rem;
		cursor: pointer;
		transition: border-color 0.15s, color 0.15s, background 0.15s;
		height: 28px;
		font-family: inherit;
	}

	.dw-feed-empty-action:hover {
		border-color: var(--dt4);
		color: var(--dt);
		background: var(--dbg2);
	}
</style>
