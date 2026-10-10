<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import notificationStore, { type Notification } from '$lib/stores/notifications';
	import { Bell, BellOff, Loader2, Check, CheckCheck, Trash2, Circle } from 'lucide-svelte';

	// The notification store exposes individual writable stores we can
	// auto-subscribe to with the $ prefix.
	const { notifications, unreadCount } = notificationStore;

	let loading = $state(true);
	let filter = $state<'all' | 'unread'>('all');

	onMount(async () => {
		try {
			await notificationStore.fetchNotifications(50, 0);
			await notificationStore.fetchUnreadCount();
		} finally {
			loading = false;
		}
	});

	const visible = $derived(
		filter === 'unread' ? $notifications.filter((n) => !n.is_read) : $notifications
	);

	async function markOne(n: Notification) {
		if (n.is_read) return;
		await notificationStore.markAsRead(n.id);
	}
	async function markAll() {
		await notificationStore.markAllAsRead();
	}
	async function remove(n: Notification) {
		await notificationStore.deleteNotification(n.id);
	}

	function timeAgo(iso: string): string {
		if (!iso) return '';
		const secs = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
		if (secs < 60) return 'just now';
		if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
		if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`;
		return `${Math.floor(secs / 86400)}d ago`;
	}
	function priClass(p: string): string {
		return p === 'urgent' || p === 'high' ? 'pri-high' : p === 'low' ? 'pri-low' : 'pri-normal';
	}
</script>

<svelte:head><title>Notifications - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Bell size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Notifications</h1>
			<p class="page-desc">Your recent activity feed across the workspace.</p>
		</div>
		<div class="head-actions">
			{#if $unreadCount > 0}
				<button class="btn btn-ghost" onclick={markAll}><CheckCheck size={15} /> Mark all read</button>
			{/if}
		</div>
	</header>

	<div class="toolbar">
		<div class="tabs">
			<button class="tab {filter === 'all' ? 'active' : ''}" onclick={() => (filter = 'all')}>
				All <span class="tab-count">{$notifications.length}</span>
			</button>
			<button class="tab {filter === 'unread' ? 'active' : ''}" onclick={() => (filter = 'unread')}>
				Unread {#if $unreadCount > 0}<span class="tab-count badge">{$unreadCount}</span>{/if}
			</button>
		</div>
	</div>

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading notifications…</div>
	{:else if visible.length === 0}
		<div class="empty-state">
			{#if filter === 'unread'}
				<CheckCheck size={40} strokeWidth={1.4} class="empty-icon" />
				<p class="empty-title">You're all caught up</p>
				<p class="empty-body">No unread notifications. New activity will show up here in real time.</p>
			{:else}
				<BellOff size={40} strokeWidth={1.4} class="empty-icon" />
				<p class="empty-title">No notifications yet</p>
				<p class="empty-body">
					When something happens in your workspace, it'll appear here.
					You can adjust delivery in <button class="link" onclick={() => goto('/settings')}>Settings</button>.
				</p>
			{/if}
		</div>
	{:else}
		<ul class="feed">
			{#each visible as n (n.id)}
				<li class="item {n.is_read ? '' : 'unread'} {priClass(n.priority)}">
					<div class="dot-col">
						{#if !n.is_read}<Circle size={9} class="unread-dot" fill="currentColor" />{/if}
					</div>
					<div class="item-body">
						<div class="item-top">
							<span class="item-title">{n.title}</span>
							<span class="item-time">{timeAgo(n.created_at)}</span>
						</div>
						{#if n.body}<p class="item-text">{n.body}</p>{/if}
						{#if n.type}<span class="type-tag">{n.type.replace(/_/g, ' ')}</span>{/if}
					</div>
					<div class="item-actions">
						{#if !n.is_read}
							<button class="icon-btn" title="Mark read" onclick={() => markOne(n)}><Check size={13} /></button>
						{/if}
						<button class="icon-btn danger" title="Dismiss" onclick={() => remove(n)}><Trash2 size={13} /></button>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 18px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }
	.head-actions { display: flex; gap: 8px; flex-shrink: 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn-ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }

	.toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
	.tabs { display: flex; gap: 4px; }
	.tab { display: inline-flex; align-items: center; gap: 6px; padding: 7px 12px; border-radius: 8px; border: 1px solid transparent; background: transparent; color: var(--dt3); font-size: 0.83rem; font-weight: 550; cursor: pointer; }
	.tab:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.tab.active { color: var(--dt); border-color: var(--dbd); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.tab-count { font-size: 0.72rem; color: var(--dt3); }
	.tab-count.badge { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); padding: 1px 6px; border-radius: 10px; font-weight: 600; }

	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.feed { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; }
	.item { display: flex; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--dbd); border-left: 3px solid transparent; }
	.item:last-child { border-bottom: none; }
	.item:hover { background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.item.unread { background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.item.pri-high { border-left-color: #ef4444; }
	.dot-col { width: 10px; display: flex; justify-content: center; padding-top: 5px; flex-shrink: 0; color: #6366f1; }
	.item-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
	.item-top { display: flex; align-items: baseline; gap: 8px; }
	.item-title { font-size: 0.88rem; font-weight: 600; color: var(--dt); flex: 1; }
	.item-time { font-size: 0.74rem; color: var(--dt3); flex-shrink: 0; }
	.item-text { font-size: 0.83rem; color: var(--dt2); margin: 0; line-height: 1.45; }
	.type-tag { align-self: flex-start; font-size: 0.66rem; text-transform: capitalize; letter-spacing: 0.02em; padding: 2px 7px; border-radius: 4px; border: 1px solid var(--dbd); color: var(--dt3); }
	.item-actions { display: flex; gap: 4px; align-items: flex-start; opacity: 0; transition: opacity 0.12s; }
	.item:hover .item-actions { opacity: 1; }
	.icon-btn { width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); display: flex; align-items: center; justify-content: center; cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.icon-btn.danger:hover { color: #ef4444; border-color: #ef4444; }

	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 380px; margin: 0; }
	.link { background: none; border: none; padding: 0; color: var(--dt2); font-size: inherit; font-weight: 500; cursor: pointer; text-decoration: underline; text-underline-offset: 2px; }
	.link:hover { color: var(--dt); }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) { .page { padding: 16px 18px; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
