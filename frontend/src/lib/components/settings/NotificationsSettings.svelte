<script lang="ts">
	import { api } from '$lib/api';

	interface Props {
		initialEmailNotifications?: boolean;
		initialDailySummary?: boolean;
	}

	let { initialEmailNotifications = true, initialDailySummary = false }: Props = $props();

	type NotificationCategoryKey = 'tasks' | 'projects' | 'mentions' | 'comments' | 'system' | 'reminders';

	let emailNotifications = $state(true);
	let dailySummary = $state(false);

	$effect.pre(() => {
		emailNotifications = initialEmailNotifications;
		dailySummary = initialDailySummary;
	});
	let pushNotifications = $state(true);
	let browserNotifications = $state(false);
	let notificationSound = $state(true);
	let notificationSoundVolume = $state(50);
	let quietHoursEnabled = $state(false);
	let quietHoursStart = $state('22:00');
	let quietHoursEnd = $state('08:00');
	let notificationCategories = $state<Record<NotificationCategoryKey, boolean>>({
		tasks: true,
		projects: true,
		mentions: true,
		comments: true,
		system: true,
		reminders: true
	});
	let recentNotifications = $state<Array<{id: string; title: string; message: string; time: string; read: boolean; type: string}>>([]);
	let isLoadingNotifications = $state(false);
	let isSaving = $state(false);
	let saveMessage = $state('');

	function getNotificationIcon(type: string): string {
		const icons: Record<string, string> = {
			task: '✓',
			project: '📁',
			mention: '@',
			comment: '💬',
			system: '⚙️',
			reminder: '🔔'
		};
		return icons[type] || '📢';
	}

	async function requestBrowserNotifications() {
		if ('Notification' in window) {
			const permission = await Notification.requestPermission();
			browserNotifications = permission === 'granted';
		}
	}

	async function handleSave() {
		isSaving = true;
		saveMessage = '';
		try {
			const settings = await api.getSettings();
			await api.updateSettings({
				default_model: settings?.default_model ?? null,
				theme: settings?.theme ?? 'light',
				email_notifications: emailNotifications,
				daily_summary: dailySummary,
				share_analytics: settings?.share_analytics ?? true,
			});
			saveMessage = 'Notification settings saved!';
			setTimeout(() => (saveMessage = ''), 2000);
		} catch (error) {
			console.error('Error saving notification settings:', error);
			saveMessage = 'Error saving settings';
		} finally {
			isSaving = false;
		}
	}
</script>

<div class="space-y-4">
	<!-- Notification Delivery -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Notification Delivery</h2>
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Push notifications</p>
					<p class="text-sm st-muted">Receive desktop/mobile push notifications</p>
				</div>
				<button
					aria-label="Toggle push notifications"
					onclick={() => (pushNotifications = !pushNotifications)}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {pushNotifications
						? 'st-toggle-on'
						: 'st-toggle-off'}"
				>
					<span class="inline-block h-4 w-4 transform rounded-full transition-transform {pushNotifications
						? 'translate-x-6 st-toggle-knob'
						: 'translate-x-1 st-toggle-knob'}"></span>
				</button>
			</div>

			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Browser notifications</p>
					<p class="text-sm st-muted">Show notifications in your browser</p>
				</div>
				<div class="flex items-center gap-2">
					{#if !browserNotifications}
						<button
							onclick={requestBrowserNotifications}
							class="btn-pill btn-pill-ghost btn-pill-sm hover:underline"
						>
							Enable
						</button>
					{/if}
					<button
						aria-label="Toggle browser notifications"
						onclick={() => browserNotifications && (browserNotifications = !browserNotifications)}
						class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {browserNotifications
							? 'st-toggle-on'
							: 'st-toggle-off'}"
					>
						<span class="inline-block h-4 w-4 transform rounded-full transition-transform {browserNotifications
							? 'translate-x-6 st-toggle-knob'
							: 'translate-x-1 st-toggle-knob'}"></span>
					</button>
				</div>
			</div>

			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Email notifications</p>
					<p class="text-sm st-muted">Receive important updates via email</p>
				</div>
				<button
					aria-label="Toggle email notifications"
					onclick={() => (emailNotifications = !emailNotifications)}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {emailNotifications
						? 'st-toggle-on'
						: 'st-toggle-off'}"
				>
					<span class="inline-block h-4 w-4 transform rounded-full transition-transform {emailNotifications
						? 'translate-x-6 st-toggle-knob'
						: 'translate-x-1 st-toggle-knob'}"></span>
				</button>
			</div>

			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Daily summary</p>
					<p class="text-sm st-muted">Get a daily recap of your activity</p>
				</div>
				<button
					aria-label="Toggle daily summary"
					onclick={() => (dailySummary = !dailySummary)}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {dailySummary
						? 'st-toggle-on'
						: 'st-toggle-off'}"
				>
					<span class="inline-block h-4 w-4 transform rounded-full transition-transform {dailySummary
						? 'translate-x-6 st-toggle-knob'
						: 'translate-x-1 st-toggle-knob'}"></span>
				</button>
			</div>
		</div>
	</div>

	<!-- Sound Settings -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Sound Settings</h2>
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Notification sounds</p>
					<p class="text-sm st-muted">Play a sound when notifications arrive</p>
				</div>
				<button
					aria-label="Toggle notification sounds"
					onclick={() => (notificationSound = !notificationSound)}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {notificationSound
						? 'st-toggle-on'
						: 'st-toggle-off'}"
				>
					<span class="inline-block h-4 w-4 transform rounded-full transition-transform {notificationSound
						? 'translate-x-6 st-toggle-knob'
						: 'translate-x-1 st-toggle-knob'}"></span>
				</button>
			</div>

			{#if notificationSound}
				<div>
					<label for="sound-volume" class="block text-sm font-medium st-label mb-2">
						Sound volume: {notificationSoundVolume}%
					</label>
					<input
						id="sound-volume"
						type="range"
						min="0"
						max="100"
						bind:value={notificationSoundVolume}
						class="w-full h-2 st-range rounded-lg appearance-none cursor-pointer"
					/>
				</div>
			{/if}
		</div>
	</div>

	<!-- Quiet Hours -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Quiet Hours (Do Not Disturb)</h2>
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Enable quiet hours</p>
					<p class="text-sm st-muted">Pause notifications during specific times</p>
				</div>
				<button
					aria-label="Toggle quiet hours"
					onclick={() => (quietHoursEnabled = !quietHoursEnabled)}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {quietHoursEnabled
						? 'st-toggle-on'
						: 'st-toggle-off'}"
				>
					<span class="inline-block h-4 w-4 transform rounded-full transition-transform {quietHoursEnabled
						? 'translate-x-6 st-toggle-knob'
						: 'translate-x-1 st-toggle-knob'}"></span>
				</button>
			</div>

			{#if quietHoursEnabled}
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="quiet-start" class="block text-sm font-medium st-label mb-1">Start time</label>
						<input
							id="quiet-start"
							type="time"
							bind:value={quietHoursStart}
							class="w-full px-3 py-2 st-input rounded-lg"
						/>
					</div>
					<div>
						<label for="quiet-end" class="block text-sm font-medium st-label mb-1">End time</label>
						<input
							id="quiet-end"
							type="time"
							bind:value={quietHoursEnd}
							class="w-full px-3 py-2 st-input rounded-lg"
						/>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Notification Categories -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Notification Categories</h2>
		<p class="text-sm st-muted mb-4">Choose which types of notifications to receive</p>
		<div class="space-y-3">
			{#each Object.entries(notificationCategories) as [category, enabled] (category)}
				{@const categoryKey = category as NotificationCategoryKey}
				<div class="flex items-center justify-between py-2 st-cat-divider last:border-0">
					<div class="flex items-center gap-3">
						<span class="text-lg">{getNotificationIcon(category)}</span>
						<div>
							<p class="font-medium st-title capitalize">{category}</p>
							<p class="text-xs st-muted">
								{#if category === 'tasks'}
									Task assignments, due dates, and status changes
								{:else if category === 'projects'}
									Project updates and milestones
								{:else if category === 'mentions'}
									When someone mentions you
								{:else if category === 'comments'}
									New comments on your items
								{:else if category === 'system'}
									System updates and maintenance
								{:else if category === 'reminders'}
									Scheduled reminders
								{/if}
							</p>
						</div>
					</div>
					<button
						aria-label="Toggle {category} notifications"
						onclick={() => (notificationCategories[categoryKey] = !notificationCategories[categoryKey])}
						class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {enabled
							? 'st-toggle-on'
							: 'st-toggle-off'}"
					>
						<span class="inline-block h-4 w-4 transform rounded-full transition-transform {enabled
							? 'translate-x-6 st-toggle-knob'
							: 'translate-x-1 st-toggle-knob'}"></span>
					</button>
				</div>
			{/each}
		</div>
	</div>

	<!-- Recent Notifications -->
	<div class="card">
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-xs font-semibold uppercase tracking-wide st-label">Recent Notifications</h2>
			<a href="/inbox" class="text-sm hover:underline" style="color: var(--bos-status-info)">View all</a>
		</div>
		{#if isLoadingNotifications}
			<div class="flex items-center justify-center py-8">
				<svg class="animate-spin h-6 w-6 st-icon" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			</div>
		{:else if recentNotifications.length === 0}
			<div class="text-center py-8 st-muted">
				<p class="text-2xl mb-2">🔔</p>
				<p>No recent notifications</p>
			</div>
		{:else}
			<div class="space-y-2">
				{#each recentNotifications as notification}
					<div
					class="flex items-start gap-3 p-3 rounded-lg {notification.read ? 'st-notif-read' : ''}"
					style="{notification.read ? '' : 'background: var(--bos-status-info-bg)'}"
				>
						<span class="text-lg">{getNotificationIcon(notification.type)}</span>
						<div class="flex-1 min-w-0">
							<p class="font-medium st-title text-sm">{notification.title}</p>
							<p class="text-xs st-muted truncate">{notification.message}</p>
							<p class="text-xs st-icon mt-1">{notification.time}</p>
						</div>
						{#if !notification.read}
							<span class="w-2 h-2 rounded-full flex-shrink-0 mt-2" style="background: var(--bos-status-info)"></span>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Save -->
	<div class="mt-6 flex items-center justify-between">
		<p class="text-sm st-muted">
			{saveMessage || 'Changes are saved when you click Save'}
		</p>
		<button
			onclick={handleSave}
			disabled={isSaving}
			class="btn-pill btn-pill-ghost btn btn-primary"
		>
			{#if isSaving}
				<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
				Saving...
			{:else}
				Save Changes
			{/if}
		</button>
	</div>
</div>

<style>
	.st-title { color: var(--dt); }
	.st-muted { color: var(--dt3); }
	.st-label { color: var(--dt2); }
	.st-icon  { color: var(--dt4); }
	.st-toggle-on  { background: var(--dt); }
	.st-toggle-off { background: var(--dbg3); }
	.st-toggle-knob { background: var(--dbg); }
	.st-range { background: var(--dbg3); }
	.st-input {
		border: 1px solid var(--dbd);
		background: var(--dbg);
		color: var(--dt);
	}
	.st-cat-divider { border-bottom: 1px solid var(--dbd2, #f0f0f0); }
	.st-notif-read {
		background: var(--bos-settings-card-bg);
		border: 1px solid var(--bos-settings-card-border);
		border-radius: var(--bos-settings-card-radius);
	}
</style>
