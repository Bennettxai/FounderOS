<script lang="ts">
	import { browser } from '$app/environment';

	interface Props {
		isElectron: boolean;
		accessibilityGranted: boolean;
		onAccessibilityChange: (granted: boolean) => void;
	}

	let { isElectron, accessibilityGranted, onAccessibilityChange }: Props = $props();

	let isCheckingPermissions = $state(false);

	async function requestAccessibility() {
		if (isElectron && browser) {
			try {
				const electron = (window as any).electron;
				await electron?.shortcuts?.requestAccessibility();
				setTimeout(async () => {
					const result = await electron?.shortcuts?.checkAccessibility();
					onAccessibilityChange(result?.granted ?? false);
				}, 1000);
			} catch (e) {
				console.error('Failed to request accessibility:', e);
			}
		}
	}

	async function openSystemPreferences(pane: string) {
		if (isElectron && browser) {
			try {
				const electron = (window as any).electron;
				const urls: Record<string, string> = {
					accessibility: 'x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility',
					screenRecording: 'x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture',
					microphone: 'x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone',
				};
				await electron?.shell?.openExternal(urls[pane] || 'x-apple.systempreferences:');
			} catch (e) {
				console.error('Failed to open system preferences:', e);
			}
		}
	}

	async function recheckPermissions() {
		if (isElectron && browser) {
			isCheckingPermissions = true;
			try {
				const electron = (window as any).electron;
				const result = await electron?.shortcuts?.checkAccessibility();
				onAccessibilityChange(result?.granted ?? false);
			} catch (e) {
				console.error('Failed to check permissions:', e);
			}
			isCheckingPermissions = false;
		}
	}
</script>

<!-- System Permissions -->
<div class="section">
	<label class="section-title">System Permissions</label>
	<p class="section-subtitle">BusinessOS needs these permissions for global shortcuts, screenshots, and voice input.</p>

	<div class="permissions-list">
		<!-- Accessibility -->
		<div class="permission-row">
			<div class="permission-icon" class:granted={accessibilityGranted}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"/>
				</svg>
			</div>
			<div class="permission-info">
				<div class="permission-name">Accessibility</div>
				<div class="permission-desc">Required for global keyboard shortcuts to work system-wide</div>
			</div>
			<div class="permission-status">
				{#if accessibilityGranted}
					<span class="status-badge granted">
						<svg viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
						</svg>
						Granted
					</span>
				{:else}
					<button class="grant-btn" onclick={requestAccessibility}>
						Grant Access
					</button>
				{/if}
				<button class="settings-btn" onclick={() => openSystemPreferences('accessibility')} title="Open System Settings" aria-label="Open System Settings for Accessibility">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
						<circle cx="12" cy="12" r="3"/>
					</svg>
				</button>
			</div>
		</div>

		<!-- Screen Recording -->
		<div class="permission-row">
			<div class="permission-icon">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
				</svg>
			</div>
			<div class="permission-info">
				<div class="permission-name">Screen Recording</div>
				<div class="permission-desc">Required for capturing screenshots from the popup chat</div>
			</div>
			<div class="permission-status">
				<button class="settings-btn" onclick={() => openSystemPreferences('screenRecording')} title="Open System Settings" aria-label="Open System Settings for Screen Recording">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
						<circle cx="12" cy="12" r="3"/>
					</svg>
				</button>
			</div>
		</div>

		<!-- Microphone -->
		<div class="permission-row">
			<div class="permission-icon">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"/>
				</svg>
			</div>
			<div class="permission-info">
				<div class="permission-name">Microphone</div>
				<div class="permission-desc">Required for voice input and meeting recording features</div>
			</div>
			<div class="permission-status">
				<button class="settings-btn" onclick={() => openSystemPreferences('microphone')} title="Open System Settings" aria-label="Open System Settings for Microphone">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
						<circle cx="12" cy="12" r="3"/>
					</svg>
				</button>
			</div>
		</div>
	</div>

	<button class="recheck-btn" onclick={recheckPermissions} disabled={isCheckingPermissions}>
		{#if isCheckingPermissions}
			<svg class="spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
			</svg>
			Checking...
		{:else}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
			</svg>
			Recheck Permissions
		{/if}
	</button>
</div>

<div class="section">
	<div class="shortcut-note">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="12" cy="12" r="10"/>
			<path d="M12 16v-4M12 8h.01"/>
		</svg>
		<span>
			After granting permissions in System Settings, click "Recheck Permissions" to update the status. You may need to restart BusinessOS for some permissions to take effect.
		</span>
	</div>
</div>

<style>
	.section {
		margin-bottom: 24px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: #333;
		display: block;
		margin-bottom: 12px;
	}

	.section-subtitle {
		font-size: 11px;
		color: #999;
		margin: -4px 0 8px 0;
	}

	.permissions-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin-top: 16px;
	}

	.permission-row {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 16px;
		background: white;
		border: 1px solid #e5e5e5;
		border-radius: 10px;
		transition: border-color 0.15s ease;
	}

	.permission-row:hover {
		border-color: #ccc;
	}

	.permission-icon {
		width: 44px;
		height: 44px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #f5f5f5;
		border-radius: 10px;
		flex-shrink: 0;
		transition: all 0.15s ease;
	}

	.permission-icon.granted {
		background: #d4edda;
	}

	.permission-icon svg {
		width: 22px;
		height: 22px;
		color: #666;
	}

	.permission-icon.granted svg {
		color: #28a745;
	}

	.permission-info {
		flex: 1;
		min-width: 0;
	}

	.permission-name {
		font-size: 14px;
		font-weight: 600;
		color: #333;
	}

	.permission-desc {
		font-size: 12px;
		color: #666;
		margin-top: 3px;
		line-height: 1.4;
	}

	.permission-status {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.grant-btn {
		padding: 8px 16px;
		background: #3b82f6;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.grant-btn:hover {
		background: #2563eb;
	}

	.settings-btn {
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #f5f5f5;
		border: 1px solid #ddd;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.settings-btn:hover {
		background: #eee;
		border-color: #ccc;
	}

	.settings-btn svg {
		width: 18px;
		height: 18px;
		color: #666;
	}

	.status-badge {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 6px 12px;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 600;
	}

	.status-badge.granted {
		background: #d4edda;
		color: #155724;
	}

	.status-badge svg {
		width: 14px;
		height: 14px;
	}

	.recheck-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		width: 100%;
		padding: 12px 16px;
		margin-top: 16px;
		background: white;
		border: 1px solid #ddd;
		border-radius: 8px;
		color: #555;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.recheck-btn:hover:not(:disabled) {
		background: #f5f5f5;
		color: #333;
	}

	.recheck-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.recheck-btn svg {
		width: 16px;
		height: 16px;
	}

	.recheck-btn .spin {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.shortcut-note {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 12px 14px;
		background: #f8f9fa;
		border: 1px solid #e9ecef;
		border-radius: 8px;
		font-size: 12px;
		color: #666;
		line-height: 1.5;
	}

	.shortcut-note svg {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		color: #6c757d;
		margin-top: 1px;
	}
</style>
