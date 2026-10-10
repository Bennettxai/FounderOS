<script lang="ts">
	import { onMount } from 'svelte';
	import { notificationStore, type Notification } from '$lib/stores/notifications';
	import { NotificationItem } from '$lib/components/notifications';
	import { 
		Bell, 
		BellOff, 
		Check, 
		CheckCheck, 
		Trash2, 
		Filter,
		Loader2,
		Inbox
	} from 'lucide-svelte';

	type FilterType = 'all' | 'unread' | 'mentions' | 'tasks' | 'invites';

	// Destructure stores from notificationStore (same pattern as NotificationDropdown)
	const { notifications, unreadCount } = notificationStore;

	let filter = $state<FilterType>('all');
	let selectedIds = $state<Set<string>>(new Set());
	let isSelectionMode = $state(false);
	let isLoading = $state(true);

	onMount(async () => {
		try {
			await notificationStore.fetchNotifications();
		} finally {
			isLoading = false;
		}
	});

	// Filtered notifications - use the store value with $
	const filteredNotifications = $derived.by(() => {
		const allNotifications = $notifications;
		let filtered = allNotifications;
		
		switch (filter) {
			case 'unread':
				filtered = filtered.filter(n => !n.is_read);
				break;
			case 'mentions':
				filtered = filtered.filter(n => n.type.includes('mention') || n.type.includes('comment'));
				break;
			case 'tasks':
				filtered = filtered.filter(n => n.type.includes('task'));
				break;
			case 'invites':
				filtered = filtered.filter(n => n.type.includes('invite') || n.type.includes('workspace'));
				break;
		}
		
		return filtered;
	});

	// Group notifications by date
	const groupedNotifications = $derived.by(() => {
		const groups: { label: string; notifications: Notification[] }[] = [];
		const now = new Date();
		const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000);
		const weekAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);

		const todayItems: Notification[] = [];
		const yesterdayItems: Notification[] = [];
		const thisWeekItems: Notification[] = [];
		const earlierItems: Notification[] = [];

		for (const n of filteredNotifications) {
			const date = new Date(n.created_at);
			if (date >= today) {
				todayItems.push(n);
			} else if (date >= yesterday) {
				yesterdayItems.push(n);
			} else if (date >= weekAgo) {
				thisWeekItems.push(n);
			} else {
				earlierItems.push(n);
			}
		}

		if (todayItems.length) groups.push({ label: 'Today', notifications: todayItems });
		if (yesterdayItems.length) groups.push({ label: 'Yesterday', notifications: yesterdayItems });
		if (thisWeekItems.length) groups.push({ label: 'This Week', notifications: thisWeekItems });
		if (earlierItems.length) groups.push({ label: 'Earlier', notifications: earlierItems });

		return groups;
	});

	async function markAllAsRead() {
		await notificationStore.markAllAsRead();
	}

	async function markSelectedAsRead() {
		if (selectedIds.size > 0) {
			await notificationStore.markMultipleAsRead(Array.from(selectedIds));
			selectedIds = new Set();
			isSelectionMode = false;
		}
	}

	async function deleteSelected() {
		for (const id of selectedIds) {
			await notificationStore.deleteNotification(id);
		}
		selectedIds = new Set();
		isSelectionMode = false;
	}

	function toggleSelection(id: string) {
		const newSet = new Set(selectedIds);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		selectedIds = newSet;
	}

	function selectAll() {
		selectedIds = new Set(filteredNotifications.map(n => n.id));
	}

	function deselectAll() {
		selectedIds = new Set();
	}

	async function handleDismiss(id: string) {
		await notificationStore.deleteNotification(id);
	}

	const filterOptions: { value: FilterType; label: string }[] = [
		{ value: 'all', label: 'All' },
		{ value: 'unread', label: 'Unread' },
		{ value: 'mentions', label: 'Mentions' },
		{ value: 'tasks', label: 'Tasks' },
		{ value: 'invites', label: 'Invites' },
	];
</script>

<div class="notifications-page">
	<!-- Header -->
	<div class="page-header">
		<div class="header-left">
			<h1>Notifications</h1>
			{#if $unreadCount > 0}
				<span class="unread-badge">{$unreadCount} unread</span>
			{/if}
		</div>
		<div class="header-actions">
			{#if isSelectionMode}
				<button class="btn-pill btn-pill-ghost action-btn" onclick={deselectAll}>
					Cancel
				</button>
				<button class="btn-pill btn-pill-ghost action-btn" onclick={selectAll}>
					Select All
				</button>
				<button 
					class="btn-pill btn-pill-ghost action-btn primary" 
					onclick={markSelectedAsRead}
					disabled={selectedIds.size === 0}
				>
					<Check class="w-4 h-4" />
					Mark Read ({selectedIds.size})
				</button>
				<button 
					class="btn-pill btn-pill-ghost action-btn danger" 
					onclick={deleteSelected}
					disabled={selectedIds.size === 0}
				>
					<Trash2 class="w-4 h-4" />
					Delete
				</button>
			{:else}
				<button 
					class="btn-pill btn-pill-ghost action-btn" 
					onclick={() => isSelectionMode = true}
				>
					Select
				</button>
				<button 
					class="btn-pill btn-pill-ghost action-btn" 
					onclick={markAllAsRead}
					disabled={$unreadCount === 0}
				>
					<CheckCheck class="w-4 h-4" />
					Mark All Read
				</button>
			{/if}
		</div>
	</div>

	<!-- Filters -->
	<div class="filters">
		{#each filterOptions as option}
			<button
				class="filter-btn"
				class:active={filter === option.value}
				onclick={() => filter = option.value}
			>
				{option.label}
				{#if option.value === 'unread' && $unreadCount > 0}
					<span class="filter-count">{$unreadCount}</span>
				{/if}
			</button>
		{/each}
	</div>

	<!-- Content -->
	<div class="notifications-content">
		{#if isLoading}
			<div class="loading-state">
				<Loader2 class="w-8 h-8 animate-spin" />
				<p>Loading notifications...</p>
			</div>
		{:else if filteredNotifications.length === 0}
			<div class="empty-state">
				{#if filter === 'all'}
					<div class="empty-icon">
						<Inbox class="w-12 h-12" />
					</div>
					<h2>No notifications yet</h2>
					<p>When you receive notifications, they'll appear here.</p>
				{:else if filter === 'unread'}
					<div class="empty-icon success">
						<CheckCheck class="w-12 h-12" />
					</div>
					<h2>All caught up!</h2>
					<p>You've read all your notifications.</p>
				{:else}
					<div class="empty-icon">
						<BellOff class="w-12 h-12" />
					</div>
					<h2>No {filter} notifications</h2>
					<p>Try a different filter or check back later.</p>
				{/if}
			</div>
		{:else}
			{#each groupedNotifications as group}
				<div class="notification-group">
					<h3 class="group-label">{group.label}</h3>
					<div class="group-items">
						{#each group.notifications as notification (notification.id)}
							<div class="notification-row" class:selected={selectedIds.has(notification.id)}>
								{#if isSelectionMode}
									<label class="checkbox-wrapper">
										<input 
											type="checkbox" 
											checked={selectedIds.has(notification.id)}
											onchange={() => toggleSelection(notification.id)}
										/>
										<span class="checkbox"></span>
									</label>
								{/if}
								<div class="notification-item-wrapper">
									<NotificationItem 
										{notification} 
										onDismiss={handleDismiss}
									/>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<style>
	.notifications-page {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--color-bg-primary, white);
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1.5rem 2rem;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.page-header h1 {
		font-size: 1.5rem;
		font-weight: 600;
		color: var(--color-text-primary, #111827);
		margin: 0;
	}

	.unread-badge {
		padding: 0.25rem 0.75rem;
		background: #3b82f6;
		color: white;
		font-size: 0.75rem;
		font-weight: 500;
		border-radius: 9999px;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.action-btn {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.5rem 0.875rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-text-secondary, #6b7280);
		background: transparent;
		border: 1px solid var(--color-border, #e5e7eb);
		border-radius: 0.5rem;
		cursor: pointer;
		transition: all 0.15s;
	}

	.action-btn:hover:not(:disabled) {
		background: var(--color-bg-secondary, #f9fafb);
		color: var(--color-text-primary, #111827);
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.action-btn.primary {
		background: #3b82f6;
		color: white;
		border-color: #3b82f6;
	}

	.action-btn.primary:hover:not(:disabled) {
		background: #2563eb;
	}

	.action-btn.danger {
		color: #dc2626;
		border-color: #fecaca;
	}

	.action-btn.danger:hover:not(:disabled) {
		background: #fef2f2;
	}

	.filters {
		display: flex;
		gap: 0.5rem;
		padding: 1rem 2rem;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
		background: var(--color-bg-secondary, #f9fafb);
	}

	.filter-btn {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.5rem 1rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-text-secondary, #6b7280);
		background: transparent;
		border: none;
		border-radius: 9999px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.filter-btn:hover {
		color: var(--color-text-primary, #111827);
		background: var(--color-bg-primary, white);
	}

	.filter-btn.active {
		color: #3b82f6;
		background: #eff6ff;
	}

	.filter-count {
		padding: 0.125rem 0.5rem;
		font-size: 0.75rem;
		background: #3b82f6;
		color: white;
		border-radius: 9999px;
	}

	.notifications-content {
		flex: 1;
		overflow-y: auto;
	}

	.loading-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 2rem;
		text-align: center;
		color: var(--color-text-secondary, #6b7280);
	}

	.empty-icon {
		width: 5rem;
		height: 5rem;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--color-bg-secondary, #f3f4f6);
		border-radius: 50%;
		margin-bottom: 1.5rem;
		color: var(--color-text-tertiary, #9ca3af);
	}

	.empty-icon.success {
		background: #dcfce7;
		color: #16a34a;
	}

	.empty-state h2 {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--color-text-primary, #111827);
		margin: 0 0 0.5rem;
	}

	.empty-state p {
		font-size: 0.875rem;
		margin: 0;
	}

	.notification-group {
		border-bottom: 1px solid var(--color-border, #e5e7eb);
	}

	.group-label {
		position: sticky;
		top: 0;
		padding: 0.75rem 2rem;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--color-text-tertiary, #9ca3af);
		background: var(--color-bg-secondary, #f9fafb);
		border-bottom: 1px solid var(--color-border, #e5e7eb);
		margin: 0;
		z-index: 10;
	}

	.group-items {
		background: var(--color-bg-primary, white);
	}

	.notification-row {
		display: flex;
		align-items: stretch;
		border-bottom: 1px solid var(--color-border-light, #f3f4f6);
	}

	.notification-row:last-child {
		border-bottom: none;
	}

	.notification-row.selected {
		background: #eff6ff;
	}

	.checkbox-wrapper {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0 0.5rem 0 1.5rem;
		cursor: pointer;
	}

	.checkbox-wrapper input {
		display: none;
	}

	.checkbox {
		width: 1.25rem;
		height: 1.25rem;
		border: 2px solid var(--color-border, #d1d5db);
		border-radius: 0.25rem;
		transition: all 0.15s;
	}

	.checkbox-wrapper input:checked + .checkbox {
		background: #3b82f6;
		border-color: #3b82f6;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='white'%3E%3Cpath d='M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z'/%3E%3C/svg%3E");
		background-size: 80%;
		background-position: center;
		background-repeat: no-repeat;
	}

	.notification-item-wrapper {
		flex: 1;
		min-width: 0;
	}

	/* Dark mode */
	:global(.dark) .notifications-page {
		background: var(--color-bg-primary, #111827);
	}

	:global(.dark) .page-header,
	:global(.dark) .filters,
	:global(.dark) .notification-group {
		border-color: var(--color-border, #374151);
	}

	:global(.dark) .group-label {
		background: var(--color-bg-secondary, #1f2937);
	}

	:global(.dark) .group-items {
		background: var(--color-bg-primary, #111827);
	}

	:global(.dark) .empty-icon {
		background: var(--color-bg-secondary, #1f2937);
	}

	:global(.dark) .notification-row.selected {
		background: rgba(59, 130, 246, 0.1);
	}
</style>
