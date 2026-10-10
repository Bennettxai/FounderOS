<script lang="ts">
	import { Popover } from 'bits-ui';
	import { notificationStore, type Notification } from '$lib/stores/notifications';
	import { CheckCheck, Bell, Loader2, Settings } from 'lucide-svelte';
	import NotificationItem from './NotificationItem.svelte';
	import { fly, fade } from 'svelte/transition';

	const { recentNotifications, unreadCount, isConnected } = notificationStore;

	let open = $state(false);
	let markingAllRead = $state(false);

	// Group notifications by time period
	function groupNotifications(notifications: Notification[]) {
		const now = new Date();
		const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		const yesterday = new Date(today.getTime() - 86400000);
		const weekAgo = new Date(today.getTime() - 7 * 86400000);

		const groups: { label: string; notifications: Notification[] }[] = [];
		
		const todayItems = notifications.filter(n => new Date(n.created_at) >= today);
		const yesterdayItems = notifications.filter(n => {
			const date = new Date(n.created_at);
			return date >= yesterday && date < today;
		});
		const thisWeekItems = notifications.filter(n => {
			const date = new Date(n.created_at);
			return date >= weekAgo && date < yesterday;
		});
		const olderItems = notifications.filter(n => new Date(n.created_at) < weekAgo);

		if (todayItems.length > 0) groups.push({ label: 'Today', notifications: todayItems });
		if (yesterdayItems.length > 0) groups.push({ label: 'Yesterday', notifications: yesterdayItems });
		if (thisWeekItems.length > 0) groups.push({ label: 'This Week', notifications: thisWeekItems });
		if (olderItems.length > 0) groups.push({ label: 'Earlier', notifications: olderItems });

		return groups;
	}

	const groupedNotifications = $derived(groupNotifications($recentNotifications));

	async function handleMarkAllRead() {
		markingAllRead = true;
		await notificationStore.markAllAsRead();
		markingAllRead = false;
	}

	function handleNavigate() {
		open = false;
	}

	async function handleDismiss(id: string) {
		await notificationStore.deleteNotification(id);
	}
</script>

<Popover.Root bind:open>
	<div class="notification-wrapper">
		<Popover.Trigger class="notification-trigger" title="Notifications">
			<Bell class="w-5 h-5" />
		</Popover.Trigger>
		{#if $unreadCount > 0}
			<span class="notification-badge">
				{$unreadCount > 99 ? '99+' : $unreadCount}
			</span>
		{/if}
	</div>

	<Popover.Content
		class="notification-dropdown"
		side="bottom"
		align="end"
		sideOffset={8}
	>
		<!-- Header -->
		<div class="dropdown-header">
			<div class="header-title">
				<h3>Notifications</h3>
				{#if !$isConnected}
					<span class="connection-status" title="Reconnecting...">
						<Loader2 class="w-3 h-3 animate-spin" />
					</span>
				{/if}
			</div>
			<div class="header-actions">
				{#if $unreadCount > 0}
					<button
						class="btn-pill btn-pill-ghost mark-all-btn"
						onclick={handleMarkAllRead}
						disabled={markingAllRead}
						title="Mark all as read"
					>
						{#if markingAllRead}
							<Loader2 class="w-3.5 h-3.5 animate-spin" />
						{:else}
							<CheckCheck class="w-3.5 h-3.5" />
						{/if}
					</button>
				{/if}
				<a href="/settings?tab=notifications" class="settings-btn" title="Notification settings" onclick={() => open = false}>
					<Settings class="w-3.5 h-3.5" />
				</a>
			</div>
		</div>

		<!-- Notification List -->
		<div class="dropdown-content">
			{#if $recentNotifications.length === 0}
				<div class="empty-state" in:fade={{ duration: 200 }}>
					<div class="empty-icon">
						<Bell class="w-8 h-8" />
					</div>
					<p>All caught up!</p>
					<span>No new notifications</span>
				</div>
			{:else}
				{#each groupedNotifications as group, groupIndex}
					<div class="notification-group" in:fly={{ y: 10, duration: 200, delay: groupIndex * 50 }}>
						<div class="group-label">{group.label}</div>
						{#each group.notifications as notification, index (notification.id)}
							<div in:fly={{ x: -10, duration: 200, delay: index * 30 }}>
								<NotificationItem 
									{notification} 
									onNavigate={handleNavigate}
									onDismiss={handleDismiss}
								/>
							</div>
						{/each}
					</div>
				{/each}
			{/if}
		</div>

		<!-- Footer -->
		{#if $recentNotifications.length > 0}
			<div class="dropdown-footer">
				<a href="/notifications" class="view-all-link" onclick={() => open = false}>
					View all notifications
				</a>
			</div>
		{/if}
	</Popover.Content>
</Popover.Root>

<style>
	.notification-wrapper {
		position: relative;
		display: inline-flex;
	}

	.notification-trigger {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 10px;
		background: var(--color-bg-secondary, #f3f4f6);
		border: 1px solid var(--color-border, #e5e7eb);
		color: var(--color-text-secondary, #6b7280);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.notification-trigger:hover {
		background: var(--color-bg-tertiary, #e5e7eb);
		color: var(--color-text, #111827);
		border-color: var(--color-border-hover, #d1d5db);
	}

	:global(.dark) .notification-trigger {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .notification-trigger:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	.notification-badge {
		position: absolute;
		top: -6px;
		right: -6px;
		min-width: 18px;
		height: 18px;
		padding: 0 5px;
		font-size: 11px;
		font-weight: 600;
		line-height: 18px;
		text-align: center;
		color: white;
		background: #ef4444;
		border-radius: 9px;
		border: 2px solid white;
		pointer-events: none;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	:global(.dark) .notification-badge {
		border-color: #1c1c1e;
	}

	:global(.notification-dropdown) {
		width: 380px;
		max-height: 480px;
		background: white;
		border: 1px solid var(--color-border, #e5e7eb);
		border-radius: 12px;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
		overflow: hidden;
		z-index: 9999;
	}

	:global(.dark .notification-dropdown) {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.1);
	}

	.dropdown-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
	}

	:global(.dark) .dropdown-header {
		border-color: rgba(255, 255, 255, 0.1);
	}

	.header-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.header-title h3 {
		font-size: 1rem;
		font-weight: 600;
		color: var(--color-text, #111827);
		margin: 0;
	}

	:global(.dark) .header-title h3 {
		color: #f5f5f7;
	}

	.connection-status {
		color: #f59e0b;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.mark-all-btn,
	.settings-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		color: var(--color-text-secondary, #6b7280);
		background: transparent;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-decoration: none;
	}

	.mark-all-btn:hover:not(:disabled),
	.settings-btn:hover {
		background: var(--color-bg-secondary, #f3f4f6);
		color: var(--color-text, #111827);
	}

	:global(.dark) .mark-all-btn:hover:not(:disabled),
	:global(.dark) .settings-btn:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	.mark-all-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.dropdown-content {
		max-height: 360px;
		overflow-y: auto;
		scroll-behavior: smooth;
	}

	.dropdown-content::-webkit-scrollbar {
		width: 6px;
	}

	.dropdown-content::-webkit-scrollbar-track {
		background: transparent;
	}

	.dropdown-content::-webkit-scrollbar-thumb {
		background: rgba(0, 0, 0, 0.15);
		border-radius: 3px;
	}

	:global(.dark) .dropdown-content::-webkit-scrollbar-thumb {
		background: rgba(255, 255, 255, 0.15);
	}

	.notification-group {
		/* Group container - no border since label handles separation */
	}

	.notification-group:last-child .group-label {
		/* Keep border on last group label too for consistency */
	}

	:global(.dark) .notification-group {
		/* Dark mode handled by group-label */
	}

	.group-label {
		padding: 10px 16px 8px;
		font-size: 0.6875rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--color-text-muted, #9ca3af);
		background: var(--color-bg-secondary, #f9fafb);
		position: sticky;
		top: 0;
		z-index: 2;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
	}

	:global(.dark) .group-label {
		color: #6e6e73;
		background: #252527;
		border-color: rgba(255, 255, 255, 0.06);
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 20px;
		text-align: center;
	}

	.empty-icon {
		width: 56px;
		height: 56px;
		border-radius: 50%;
		background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%);
		display: flex;
		align-items: center;
		justify-content: center;
		color: #9ca3af;
		margin-bottom: 4px;
	}

	:global(.dark) .empty-icon {
		background: linear-gradient(135deg, #2c2c2e 0%, #3a3a3c 100%);
		color: #6e6e73;
	}

	.empty-state p {
		margin: 12px 0 4px;
		font-size: 0.9375rem;
		font-weight: 600;
		color: var(--color-text, #111827);
	}

	:global(.dark) .empty-state p {
		color: #f5f5f7;
	}

	.empty-state span {
		font-size: 0.8125rem;
		color: var(--color-text-muted, #9ca3af);
	}

	:global(.dark) .empty-state span {
		color: #6e6e73;
	}

	.dropdown-footer {
		padding: 12px 16px;
		border-top: 1px solid var(--color-border, #e5e7eb);
		text-align: center;
		background: var(--color-bg-secondary, #f9fafb);
	}

	:global(.dark) .dropdown-footer {
		border-color: rgba(255, 255, 255, 0.1);
		background: #252527;
	}

	.view-all-link {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #3b82f6;
		text-decoration: none;
		transition: color 0.15s ease;
	}

	.view-all-link:hover {
		color: #2563eb;
	}
</style>
