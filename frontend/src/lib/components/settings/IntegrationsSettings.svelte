<script lang="ts">
	import { api, type GoogleConnectionStatus } from '$lib/api';

	interface Props {
		initialGoogleStatus?: GoogleConnectionStatus | null;
		initialMessage?: string;
	}

	let { initialGoogleStatus = null, initialMessage = '' }: Props = $props();

	let googleStatus = $state<GoogleConnectionStatus | null>(null);
	let isConnectingGoogle = $state(false);
	let isDisconnectingGoogle = $state(false);
	let googleMessage = $state('');

	$effect.pre(() => {
		googleStatus = initialGoogleStatus;
		googleMessage = initialMessage;
	});

	async function connectGoogle() {
		isConnectingGoogle = true;
		googleMessage = '';
		try {
			const response = await api.initiateGoogleAuth();
			window.location.href = response.auth_url;
		} catch (error) {
			console.error('Error initiating Google auth:', error);
			googleMessage = 'Failed to initiate Google authentication';
			isConnectingGoogle = false;
		}
	}

	async function disconnectGoogle() {
		if (!confirm('Are you sure you want to disconnect your Google Calendar? Your synced events will remain in BusinessOS.')) {
			return;
		}
		isDisconnectingGoogle = true;
		googleMessage = '';
		try {
			await api.disconnectGoogle();
			googleStatus = { connected: false };
			googleMessage = 'Google Calendar disconnected';
			setTimeout(() => (googleMessage = ''), 3000);
		} catch (error) {
			console.error('Error disconnecting Google:', error);
			googleMessage = 'Failed to disconnect Google account';
		} finally {
			isDisconnectingGoogle = false;
		}
	}
</script>

<div class="space-y-4">
	{#if googleMessage}
		<div
			class="p-4 rounded-lg"
			style="{googleMessage.includes('Error') || googleMessage.includes('Failed')
				? 'background: var(--bos-status-error-bg); color: var(--bos-status-error);'
				: 'background: var(--bos-status-success-bg); color: var(--bos-status-success);'}"
		>
			{googleMessage}
		</div>
	{/if}

	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Google Calendar</h2>
		<p class="text-sm st-muted mb-4">
			Connect your Google Calendar to sync events, see your schedule, and let the AI help plan your tasks around your existing commitments.
		</p>

		{#if googleStatus?.connected}
			<div class="flex items-center justify-between p-4 rounded-lg" style="background: var(--bos-status-success-bg); border: 1px solid var(--bos-status-success)">
				<div class="flex items-center gap-4">
					<div class="w-12 h-12 rounded-full st-int-icon-bg flex items-center justify-center shadow-sm">
						<svg class="w-6 h-6" viewBox="0 0 24 24">
							<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
							<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
							<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
							<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
						</svg>
					</div>
					<div>
						<p class="font-medium" style="color: var(--bos-status-success)">Connected</p>
						{#if googleStatus.email}
							<p class="text-sm" style="color: var(--bos-status-success)">{googleStatus.email}</p>
						{/if}
						{#if googleStatus.connected_at}
							<p class="text-xs" style="color: var(--bos-status-success); opacity: 0.8">
								Connected {new Date(googleStatus.connected_at).toLocaleDateString()}
							</p>
						{/if}
					</div>
				</div>
				<button
					onclick={disconnectGoogle}
					disabled={isDisconnectingGoogle}
					class="btn-pill btn-pill-soft btn-pill-sm"
				>
					{#if isDisconnectingGoogle}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Disconnecting...
					{:else}
						Disconnect
					{/if}
				</button>
			</div>
		{:else}
			<div class="flex items-center justify-between p-4 rounded-lg st-int-disconnected">
				<div class="flex items-center gap-4">
					<div class="w-12 h-12 rounded-full st-int-icon-bg flex items-center justify-center shadow-sm">
						<svg class="w-6 h-6 st-icon" viewBox="0 0 24 24">
							<path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
							<path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
							<path fill="currentColor" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
							<path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
						</svg>
					</div>
					<div>
						<p class="font-medium st-label">Not connected</p>
						<p class="text-sm st-muted">Connect to sync your calendar</p>
					</div>
				</div>
				<button
					onclick={connectGoogle}
					disabled={isConnectingGoogle}
					class="btn-pill btn-pill-primary btn-pill-sm"
				>
					{#if isConnectingGoogle}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Connecting...
					{:else}
						Connect Google Calendar
					{/if}
				</button>
			</div>
		{/if}

		<div class="mt-4 text-xs st-muted">
			<p class="font-medium mb-1">Permissions requested:</p>
			<ul class="list-disc list-inside space-y-0.5">
				<li>Read your calendar events</li>
				<li>Create and update events (for two-way sync)</li>
				<li>View your email address</li>
			</ul>
		</div>
	</div>

	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">More Integrations</h2>
		<p class="text-sm st-muted mb-4">
			Additional integrations coming soon. Let us know what you'd like to see!
		</p>
		<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
			{#each ['Slack', 'Notion', 'Linear'] as integration}
				<div class="p-4 rounded-lg st-int-coming-soon text-center opacity-50">
					<div class="w-8 h-8 mx-auto mb-2 rounded st-int-placeholder"></div>
					<p class="text-sm st-muted">{integration}</p>
					<span class="text-xs st-icon">Coming soon</span>
				</div>
			{/each}
		</div>
	</div>
</div>

<style>
	.st-title { color: var(--dt); }
	.st-muted { color: var(--dt3); }
	.st-label { color: var(--dt2); }
	.st-icon  { color: var(--dt4); }
	.st-int-disconnected {
		background: var(--bos-settings-card-bg);
		border: 1px solid var(--bos-settings-card-border);
		border-radius: var(--bos-settings-card-radius);
	}
	.st-int-icon-bg {
		background: var(--dbg);
	}
	.st-int-coming-soon {
		border: 1px solid var(--dbd);
	}
	.st-int-placeholder {
		background: var(--dbg3);
	}
</style>
