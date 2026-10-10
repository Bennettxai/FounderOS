<script lang="ts">
	import { goto } from '$app/navigation';
	import { notificationStore, type Notification } from '$lib/stores/notifications';
	import {
		CheckSquare,
		FolderOpen,
		MessageSquare,
		Building2,
		Bell,
		AtSign,
		AlertCircle,
		Clock,
		UserPlus,
		X
	} from 'lucide-svelte';

	interface Props {
		notification: Notification;
		onNavigate?: () => void;
		onDismiss?: (id: string) => void;
	}

	let { notification, onNavigate, onDismiss }: Props = $props();

	let isHovered = $state(false);

	function getIcon(type: string) {
		if (type.startsWith('task') || type.includes('task')) return CheckSquare;
		if (type.startsWith('project') || type.includes('project')) return FolderOpen;
		if (type.includes('mention') || type.includes('comment')) return MessageSquare;
		if (type.includes('workspace') || type.includes('invite')) return UserPlus;
		if (type.includes('overdue') || type.includes('due_soon')) return Clock;
		if (type.includes('error') || type.includes('failed')) return AlertCircle;
		if (type.includes('system')) return Bell;
		return Bell;
	}

	function getIconBgColor(type: string): string {
		if (type.includes('overdue') || type.includes('error') || type.includes('failed')) {
			return 'bg-red-100 dark:bg-red-900/30';
		}
		if (type.includes('due_soon') || type.includes('warning')) {
			return 'bg-amber-100 dark:bg-amber-900/30';
		}
		if (type.includes('completed') || type.includes('success')) {
			return 'bg-green-100 dark:bg-green-900/30';
		}
		if (type.includes('mention') || type.includes('comment')) {
			return 'bg-blue-100 dark:bg-blue-900/30';
		}
		if (type.includes('workspace') || type.includes('invite')) {
			return 'bg-purple-100 dark:bg-purple-900/30';
		}
		if (type.includes('task')) {
			return 'bg-emerald-100 dark:bg-emerald-900/30';
		}
		return 'bg-gray-100 dark:bg-gray-700';
	}

	function getIconColor(type: string): string {
		if (type.includes('overdue') || type.includes('error') || type.includes('failed')) {
			return 'text-red-600 dark:text-red-400';
		}
		if (type.includes('due_soon') || type.includes('warning')) {
			return 'text-amber-600 dark:text-amber-400';
		}
		if (type.includes('completed') || type.includes('success')) {
			return 'text-green-600 dark:text-green-400';
		}
		if (type.includes('mention') || type.includes('comment')) {
			return 'text-blue-600 dark:text-blue-400';
		}
		if (type.includes('workspace') || type.includes('invite')) {
			return 'text-purple-600 dark:text-purple-400';
		}
		if (type.includes('task')) {
			return 'text-emerald-600 dark:text-emerald-400';
		}
		return 'text-gray-500 dark:text-gray-400';
	}

	function getTypePill(type: string): { label: string; class: string } | null {
		if (type.includes('invite')) return { label: 'Invite', class: 'pill-purple' };
		if (type.includes('task_assigned')) return { label: 'Assigned', class: 'pill-emerald' };
		if (type.includes('task_completed')) return { label: 'Completed', class: 'pill-green' };
		if (type.includes('comment') || type.includes('mention')) return { label: 'Mention', class: 'pill-blue' };
		if (type.includes('overdue')) return { label: 'Overdue', class: 'pill-red' };
		if (type.includes('system')) return { label: 'System', class: 'pill-gray' };
		return null;
	}

	function getInitials(name: string): string {
		if (!name) return '?';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
		return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
	}

	function formatTime(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diff = now.getTime() - date.getTime();

		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return 'Just now';
		if (minutes < 60) return `${minutes}m`;
		if (hours < 24) return `${hours}h`;
		if (days < 7) return `${days}d`;
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	async function handleClick() {
		// Mark as read
		if (!notification.is_read) {
			await notificationStore.markAsRead(notification.id);
		}

		// Navigate to entity if available
		if (notification.entity_type && notification.entity_id) {
			const entityPath = notification.entity_type === 'task' ? 'tasks' :
				notification.entity_type === 'project' ? 'projects' :
				notification.entity_type === 'comment' ? 'tasks' : // Comments link to their task
				notification.entity_type;
			
			goto(`/${entityPath}/${notification.entity_id}`);
		}

		onNavigate?.();
	}

	function handleDismiss(e: MouseEvent) {
		e.stopPropagation();
		onDismiss?.(notification.id);
	}

	const IconComponent = $derived(getIcon(notification.type));
	const iconColor = $derived(getIconColor(notification.type));
	const iconBgColor = $derived(getIconBgColor(notification.type));
	const typePill = $derived(getTypePill(notification.type));
</script>

<button
	class="btn-pill btn-pill-ghost notification-item"
	class:unread={!notification.is_read}
	onclick={handleClick}
	onmouseenter={() => isHovered = true}
	onmouseleave={() => isHovered = false}
	type="button"
>
	<!-- Unread indicator bar -->
	{#if !notification.is_read}
		<div class="unread-bar"></div>
	{/if}

	<!-- Icon or Avatar -->
	<div class="flex-shrink-0">
		{#if notification.sender_avatar_url}
			<img
				src={notification.sender_avatar_url}
				alt={notification.sender_name || ''}
				class="w-10 h-10 rounded-full object-cover ring-2 ring-white dark:ring-gray-800"
			/>
		{:else if notification.sender_name}
			<div class="w-10 h-10 rounded-full {iconBgColor} flex items-center justify-center ring-2 ring-white dark:ring-gray-800">
				<span class="text-sm font-semibold {iconColor}">{getInitials(notification.sender_name)}</span>
			</div>
		{:else}
			<div class="w-10 h-10 rounded-full {iconBgColor} flex items-center justify-center ring-2 ring-white dark:ring-gray-800">
				<svelte:component this={IconComponent} class="w-5 h-5 {iconColor}" />
			</div>
		{/if}
	</div>

	<!-- Content -->
	<div class="flex-1 min-w-0">
		<div class="flex items-start justify-between gap-2">
			<p class="notification-title">
				{notification.title}
			</p>
			<div class="flex items-center gap-2 flex-shrink-0">
				{#if typePill}
					<span class="type-pill {typePill.class}">{typePill.label}</span>
				{/if}
				<span class="notification-time">
					{formatTime(notification.created_at)}
				</span>
			</div>
		</div>
		{#if notification.body}
			<p class="notification-body">
				{notification.body}
			</p>
		{/if}
		{#if notification.sender_name}
			<p class="notification-sender">
				{notification.sender_name}
			</p>
		{/if}
	</div>

	<!-- Dismiss button (on hover) -->
	{#if isHovered}
		<button
			class="btn-pill btn-pill-ghost dismiss-btn"
			onclick={handleDismiss}
			title="Dismiss"
			type="button"
		>
			<X class="w-3.5 h-3.5" />
		</button>
	{/if}
</button>

<style>
	.notification-item {
		position: relative;
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px 16px;
		padding-left: 20px;
		width: 100%;
		text-align: left;
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.notification-item:hover {
		background-color: var(--color-bg-secondary, #f9fafb);
	}

	:global(.dark) .notification-item:hover {
		background-color: rgba(255, 255, 255, 0.04);
	}

	.notification-item.unread {
		background-color: rgba(59, 130, 246, 0.04);
	}

	:global(.dark) .notification-item.unread {
		background-color: rgba(59, 130, 246, 0.08);
	}

	.unread-bar {
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		width: 3px;
		background: linear-gradient(180deg, #3b82f6 0%, #2563eb 100%);
		border-radius: 0 2px 2px 0;
	}

	.notification-title {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-text, #111827);
		margin: 0;
		line-height: 1.4;
		overflow: hidden;
		text-overflow: ellipsis;
		display: -webkit-box;
		-webkit-line-clamp: 1;
		-webkit-box-orient: vertical;
	}

	:global(.dark) .notification-title {
		color: #f5f5f7;
	}

	.notification-body {
		font-size: 0.8125rem;
		color: var(--color-text-secondary, #6b7280);
		margin: 4px 0 0 0;
		overflow: hidden;
		text-overflow: ellipsis;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		line-height: 1.4;
	}

	:global(.dark) .notification-body {
		color: #a1a1a6;
	}

	.notification-sender {
		font-size: 0.75rem;
		color: var(--color-text-muted, #9ca3af);
		margin: 4px 0 0 0;
	}

	:global(.dark) .notification-sender {
		color: #6e6e73;
	}

	.notification-time {
		font-size: 0.6875rem;
		font-weight: 500;
		color: var(--color-text-muted, #9ca3af);
		white-space: nowrap;
	}

	:global(.dark) .notification-time {
		color: #6e6e73;
	}

	/* Type Pills */
	.type-pill {
		display: inline-flex;
		padding: 2px 6px;
		font-size: 0.625rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		border-radius: 4px;
		white-space: nowrap;
	}

	.pill-purple {
		background: rgba(147, 51, 234, 0.1);
		color: #9333ea;
	}
	:global(.dark) .pill-purple {
		background: rgba(147, 51, 234, 0.2);
		color: #c084fc;
	}

	.pill-emerald {
		background: rgba(16, 185, 129, 0.1);
		color: #059669;
	}
	:global(.dark) .pill-emerald {
		background: rgba(16, 185, 129, 0.2);
		color: #34d399;
	}

	.pill-green {
		background: rgba(34, 197, 94, 0.1);
		color: #16a34a;
	}
	:global(.dark) .pill-green {
		background: rgba(34, 197, 94, 0.2);
		color: #4ade80;
	}

	.pill-blue {
		background: rgba(59, 130, 246, 0.1);
		color: #2563eb;
	}
	:global(.dark) .pill-blue {
		background: rgba(59, 130, 246, 0.2);
		color: #60a5fa;
	}

	.pill-red {
		background: rgba(239, 68, 68, 0.1);
		color: #dc2626;
	}
	:global(.dark) .pill-red {
		background: rgba(239, 68, 68, 0.2);
		color: #f87171;
	}

	.pill-gray {
		background: rgba(107, 114, 128, 0.1);
		color: #6b7280;
	}
	:global(.dark) .pill-gray {
		background: rgba(107, 114, 128, 0.2);
		color: #9ca3af;
	}

	/* Dismiss button */
	.dismiss-btn {
		position: absolute;
		top: 8px;
		right: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 6px;
		border: none;
		background: var(--color-bg-secondary, #f3f4f6);
		color: var(--color-text-muted, #9ca3af);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.dismiss-btn:hover {
		background: var(--color-bg-tertiary, #e5e7eb);
		color: var(--color-text-secondary, #6b7280);
	}

	:global(.dark) .dismiss-btn {
		background: #3a3a3c;
		color: #6e6e73;
	}

	:global(.dark) .dismiss-btn:hover {
		background: #4a4a4c;
		color: #a1a1a6;
	}
</style>
