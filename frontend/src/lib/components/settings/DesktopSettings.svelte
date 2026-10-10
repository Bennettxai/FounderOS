<script lang="ts">
	import { onMount } from 'svelte';
	import { updateStore } from '$lib/stores/updateStore';

	let accessibilityGranted = $state(false);
	let shortcuts = $state<{ quickChat: string; spotlight: string; voiceInput: string }>({
		quickChat: 'CommandOrControl+Shift+Space',
		spotlight: 'CommandOrControl+Space',
		voiceInput: 'CommandOrControl+D',
	});
	let isCheckingPermissions = $state(false);

	let diagnosticsState = $state<'idle' | 'copying' | 'copied' | 'error'>('idle');
	let diagnosticsError = $state('');
	let showReleaseNotes = $state(false);

	$effect(() => {
		loadDesktopSettings();
	});

	onMount(() => {
		updateStore.init();
	});

	function formatReleaseNotes(notes: unknown): string {
		if (!notes) return 'No release notes were attached to this update.';
		if (typeof notes === 'string') return notes.trim() || 'No release notes were attached to this update.';
		return JSON.stringify(notes, null, 2);
	}

	function formatBytes(value?: number): string {
		if (!value || value < 1) return '';
		const units = ['B', 'KB', 'MB', 'GB'];
		let size = value;
		let index = 0;
		while (size >= 1024 && index < units.length - 1) {
			size /= 1024;
			index += 1;
		}
		return `${size.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
	}

	function formatDate(value: string | null): string {
		if (!value) return '';
		const date = new Date(value);
		if (Number.isNaN(date.getTime())) return '';
		return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	async function copyDiagnostics() {
		diagnosticsState = 'copying';
		diagnosticsError = '';
		try {
			const electron = (window as any).electron;
			const report = await electron?.diagnostics?.collect?.();
			if (!report) throw new Error('Diagnostics are only available in the desktop app.');
			await navigator.clipboard.writeText(JSON.stringify(report, null, 2));
			diagnosticsState = 'copied';
			setTimeout(() => {
				if (diagnosticsState === 'copied') diagnosticsState = 'idle';
			}, 2500);
		} catch (e) {
			diagnosticsState = 'error';
			diagnosticsError = e instanceof Error ? e.message : 'Failed to copy diagnostics';
		}
	}

	async function loadDesktopSettings() {
		try {
			const electron = (window as any).electron;
			if (electron?.shortcuts) {
				const result = await electron.shortcuts.checkAccessibility();
				accessibilityGranted = result?.granted ?? false;

				const shortcutsResult = await electron.shortcuts.get();
				if (shortcutsResult) {
					shortcuts = shortcutsResult;
				}
			}
		} catch (error) {
			console.error('Error loading desktop settings:', error);
		}
	}

	async function checkAccessibility() {
		isCheckingPermissions = true;
		try {
			const electron = (window as any).electron;
			if (electron?.shortcuts) {
				const result = await electron.shortcuts.checkAccessibility();
				accessibilityGranted = result?.granted ?? false;
			}
		} catch (error) {
			console.error('Error checking accessibility:', error);
		}
		isCheckingPermissions = false;
	}

	async function requestAccessibility() {
		try {
			const electron = (window as any).electron;
			if (electron?.shortcuts) {
				await electron.shortcuts.requestAccessibility();
				setTimeout(checkAccessibility, 1000);
			}
		} catch (error) {
			console.error('Error requesting accessibility:', error);
		}
	}

	async function openSystemPreferences(pane: string) {
		try {
			const electron = (window as any).electron;
			if (electron?.shell) {
				const urls: Record<string, string> = {
					accessibility: 'x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility',
					screenRecording: 'x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture',
					microphone: 'x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone',
				};
				await electron.shell.openExternal(urls[pane] || 'x-apple.systempreferences:');
			}
		} catch (error) {
			console.error('Error opening system preferences:', error);
		}
	}

	async function resetShortcuts() {
		try {
			const electron = (window as any).electron;
			if (electron?.shortcuts) {
				const result = await electron.shortcuts.reset();
				if (result?.shortcuts) {
					shortcuts = result.shortcuts;
				}
			}
		} catch (error) {
			console.error('Error resetting shortcuts:', error);
		}
	}

	function formatShortcut(shortcut: string): string {
		return shortcut
			.replace('CommandOrControl', '⌘')
			.replace('Command', '⌘')
			.replace('Control', '⌃')
			.replace('Shift', '⇧')
			.replace('Alt', '⌥')
			.replace('Option', '⌥')
			.replace(/\+/g, ' ');
	}
</script>

<div class="space-y-6">
	<!-- App Updates -->
	<div class="card">
		<div class="update-head">
			<div>
				<p class="update-eyebrow">Desktop maintenance</p>
				<h2 class="text-lg font-medium st-title mb-1">App Updates</h2>
				<p class="text-sm st-sub">
					{#if $updateStore.isDesktop}
						BusinessOS checks the desktop release feed and installs updates from the packaged app.
					{:else}
						Update checks run inside the installed BusinessOS desktop app.
					{/if}
				</p>
			</div>
			<div class="update-status-pill" class:status-live={$updateStore.isDesktop} class:status-web={!$updateStore.isDesktop}>
				<span class="status-dot"></span>
				{$updateStore.isDesktop ? ($updateStore.isPackaged ? 'Desktop app' : 'Dev desktop') : 'Web session'}
			</div>
		</div>

		<div class="update-grid">
			<div class="update-fact">
				<span>Current</span>
				<strong>{$updateStore.currentVersion || 'Unknown'}</strong>
			</div>
			<div class="update-fact">
				<span>Channel</span>
				<strong>{$updateStore.channel}</strong>
			</div>
			<div class="update-fact">
				<span>Latest</span>
				<strong>{$updateStore.latestVersion || ($updateStore.state === 'none' ? 'Current' : 'Not checked')}</strong>
			</div>
			<div class="update-fact">
				<span>Last checked</span>
				<strong>{$updateStore.checkedAt ? formatDate($updateStore.checkedAt) : 'Never'}</strong>
			</div>
		</div>

		{#if $updateStore.state === 'available'}
			<div class="update-callout">
				<div>
					<p class="callout-title">BusinessOS {$updateStore.latestVersion} is available.</p>
					<p class="callout-copy">
						{#if $updateStore.releaseDate}
							Released {formatDate($updateStore.releaseDate)}.
						{:else}
							Review the release notes, then download the update when you are ready.
						{/if}
					</p>
				</div>
				<div class="callout-actions">
					<button type="button" onclick={() => (showReleaseNotes = true)} class="btn btn-secondary text-sm st-btn-secondary">
						Release notes
					</button>
					<button type="button" onclick={() => updateStore.download()} class="btn-pill btn-pill-ghost btn-pill-sm btn btn-primary">
						Download
					</button>
				</div>
			</div>
		{:else if $updateStore.state === 'unsupported'}
			<div class="update-callout update-callout-required">
				<div>
					<p class="callout-title">This desktop build is no longer supported.</p>
					<p class="callout-copy">
						Install a newer BusinessOS release before continuing. Minimum supported version:
						{$updateStore.minimumSupportedVersion || 'latest'}.
					</p>
				</div>
				<button type="button" onclick={() => updateStore.check()} class="btn-pill btn-pill-ghost btn-pill-sm btn btn-primary">
					Check for update
				</button>
			</div>
		{:else if $updateStore.state === 'downloading'}
			<div class="update-progress">
				<div class="progress-head">
					<span>Downloading update</span>
					<strong>{Math.round($updateStore.progress?.percent ?? 0)}%</strong>
				</div>
				<div class="progress-track">
					<div class="progress-fill" style={`width: ${Math.max(0, Math.min(100, $updateStore.progress?.percent ?? 0))}%`}></div>
				</div>
				<p>
					{#if $updateStore.progress?.transferred && $updateStore.progress?.total}
						{formatBytes($updateStore.progress.transferred)} of {formatBytes($updateStore.progress.total)}
					{:else}
						Preparing download.
					{/if}
				</p>
			</div>
		{:else if $updateStore.state === 'downloaded'}
			<div class="update-callout update-callout-ready">
				<div>
					<p class="callout-title">Update ready to install.</p>
					<p class="callout-copy">Restart BusinessOS to finish installing version {$updateStore.latestVersion || 'the latest release'}.</p>
				</div>
				<button type="button" onclick={() => updateStore.install()} class="btn-pill btn-pill-ghost btn-pill-sm btn btn-primary">
					Install and restart
				</button>
			</div>
		{:else if $updateStore.state === 'error'}
			<div class="update-error">
				<strong>Update check failed.</strong>
				<span>{$updateStore.error || 'The update server could not be reached.'}</span>
			</div>
		{:else if $updateStore.state === 'none'}
			<div class="update-muted">
				{$updateStore.message || 'BusinessOS is up to date.'}
			</div>
		{/if}

		<div class="flex items-center gap-3 flex-wrap update-actions">
			<button
				onclick={() => updateStore.check()}
				disabled={$updateStore.state === 'checking' || $updateStore.state === 'downloading'}
				class="btn btn-secondary text-sm st-btn-secondary"
			>
				{$updateStore.state === 'checking' ? 'Checking...' : 'Check for updates'}
			</button>
			{#if $updateStore.releaseNotes}
				<button type="button" onclick={() => (showReleaseNotes = true)} class="btn btn-secondary text-sm st-btn-secondary">
					View release notes
				</button>
			{/if}
		</div>
	</div>

	<!-- System Permissions -->
	<div class="card">
		<h2 class="text-lg font-medium st-title mb-4">System Permissions</h2>
		<p class="text-sm st-muted mb-6">
			BusinessOS requires certain system permissions for features like global shortcuts, screenshot capture, and voice input.
		</p>

		<div class="space-y-4">
			<!-- Accessibility -->
			<div class="flex items-center justify-between p-4 rounded-lg st-perm-card">
				<div class="flex items-center gap-4">
					<div
						class="w-10 h-10 rounded-lg flex items-center justify-center"
						style="{accessibilityGranted ? 'background: var(--bos-status-success-bg)' : 'background: var(--bos-status-warning-bg)'}"
					>
						<svg
							class="w-5 h-5"
							style="{accessibilityGranted ? 'color: var(--bos-status-success)' : 'color: var(--bos-status-warning)'}"
							fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
						</svg>
					</div>
					<div>
						<p class="font-medium st-title">Accessibility</p>
						<p class="text-sm st-muted">
							{accessibilityGranted ? 'Global shortcuts enabled' : 'Required for global keyboard shortcuts'}
						</p>
					</div>
				</div>
				<div class="flex items-center gap-2">
					{#if accessibilityGranted}
						<span class="flex items-center gap-1.5 text-sm" style="color: var(--bos-status-success)">
							<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
								<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
							</svg>
							Enabled
						</span>
					{:else}
						<button onclick={requestAccessibility} class="btn-pill btn-pill-ghost btn-pill-sm btn btn-primary">Enable</button>
					{/if}
					<button
						onclick={() => openSystemPreferences('accessibility')}
						class="btn btn-secondary text-sm st-btn-secondary"
					>
						Open Settings
					</button>
				</div>
			</div>

			<!-- Screen Recording -->
			<div class="flex items-center justify-between p-4 rounded-lg st-perm-card">
				<div class="flex items-center gap-4">
					<div class="w-10 h-10 rounded-lg st-icon-bg flex items-center justify-center">
						<svg class="w-5 h-5 st-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
					</div>
					<div>
						<p class="font-medium st-title">Screen Recording</p>
						<p class="text-sm st-muted">Required for screenshot capture</p>
					</div>
				</div>
				<button
					onclick={() => openSystemPreferences('screenRecording')}
					class="btn btn-secondary text-sm st-btn-secondary"
				>
					Open Settings
				</button>
			</div>

			<!-- Microphone -->
			<div class="flex items-center justify-between p-4 rounded-lg st-perm-card">
				<div class="flex items-center gap-4">
					<div class="w-10 h-10 rounded-lg st-icon-bg flex items-center justify-center">
						<svg class="w-5 h-5 st-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
						</svg>
					</div>
					<div>
						<p class="font-medium st-title">Microphone</p>
						<p class="text-sm st-muted">Required for voice input and meeting recording</p>
					</div>
				</div>
				<button
					onclick={() => openSystemPreferences('microphone')}
					class="btn btn-secondary text-sm st-btn-secondary"
				>
					Open Settings
				</button>
			</div>
		</div>
	</div>

	<!-- Diagnostics -->
	<div class="card">
		<h2 class="text-lg font-medium st-title mb-1">Support Diagnostics</h2>
		<p class="text-sm st-sub mb-4">
			Copy a local report for app boot, backend, Optimal Engine, terminal, and workspace routing.
		</p>
		<div class="flex items-center gap-3 flex-wrap">
			<button
				onclick={copyDiagnostics}
				disabled={diagnosticsState === 'copying'}
				class="btn btn-secondary text-sm st-btn-secondary"
			>
				{diagnosticsState === 'copying' ? 'Copying…' : 'Copy diagnostics'}
			</button>
			{#if diagnosticsState === 'copied'}
				<span class="text-sm" style="color: var(--bos-status-success)">Copied.</span>
			{:else if diagnosticsState === 'error'}
				<span class="text-sm" style="color: #e11d48;">{diagnosticsError}</span>
			{/if}
		</div>
	</div>

	<!-- Keyboard Shortcuts -->
	<div class="card">
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-lg font-medium st-title">Keyboard Shortcuts</h2>
			<button
				onclick={resetShortcuts}
				class="btn-pill btn-pill-ghost btn-pill-sm"
			>
				Reset to defaults
			</button>
		</div>
		<p class="text-sm st-muted mb-6">
			Global shortcuts work system-wide, even when the app is in the background.
		</p>

		<div class="space-y-3">
			<div class="flex items-center justify-between p-3 rounded-lg st-shortcut-row">
				<div>
					<p class="font-medium st-title text-sm">Quick Chat</p>
					<p class="text-xs st-muted">Open chat popup from anywhere</p>
				</div>
				<div class="flex items-center gap-2 font-mono text-sm st-kbd px-3 py-1.5 rounded">
					{formatShortcut(shortcuts.quickChat)}
				</div>
			</div>

			<div class="flex items-center justify-between p-3 rounded-lg st-shortcut-row">
				<div>
					<p class="font-medium st-title text-sm">Voice Input</p>
					<p class="text-xs st-muted">Start voice dictation</p>
				</div>
				<div class="flex items-center gap-2 font-mono text-sm st-kbd px-3 py-1.5 rounded">
					{formatShortcut(shortcuts.voiceInput)}
				</div>
			</div>
		</div>

		{#if !accessibilityGranted}
			<div class="mt-4 p-3 rounded-lg" style="background: var(--bos-status-warning-bg); border: 1px solid var(--bos-status-warning)">
				<p class="text-sm" style="color: var(--bos-status-warning)">
					Enable Accessibility permission above to use global shortcuts.
				</p>
			</div>
		{/if}
	</div>
</div>

{#if showReleaseNotes}
	<div class="notes-backdrop">
		<button type="button" class="notes-scrim" onclick={() => (showReleaseNotes = false)} aria-label="Close release notes"></button>
		<div class="notes-modal" role="dialog" aria-modal="true" aria-label="Release notes" tabindex="-1">
			<div class="notes-head">
				<div>
					<p class="update-eyebrow">Release notes</p>
					<h2>BusinessOS {$updateStore.latestVersion || 'update'}</h2>
				</div>
				<button type="button" class="notes-close" onclick={() => (showReleaseNotes = false)} aria-label="Close release notes">×</button>
			</div>
			<pre>{formatReleaseNotes($updateStore.releaseNotes)}</pre>
			<div class="notes-actions">
				<button type="button" class="btn btn-secondary text-sm st-btn-secondary" onclick={() => (showReleaseNotes = false)}>
					Close
				</button>
				{#if $updateStore.state === 'available'}
					<button type="button" class="btn-pill btn-pill-ghost btn-pill-sm btn btn-primary" onclick={() => updateStore.download()}>
						Download update
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.st-title { color: var(--dt); }
	.st-muted { color: var(--dt3); }
	.st-icon  { color: var(--dt3); }
	.st-perm-card { border: 1px solid var(--dbd); }
	.st-icon-bg { background: var(--dbg3); }
	.st-btn-secondary {
		background: var(--dbg2);
		color: var(--dt);
		border-color: var(--dbd);
	}
	.st-shortcut-row { background: var(--dbg2); }
	.st-kbd {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		color: var(--dt);
	}
	.update-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
	.update-eyebrow { margin: 0 0 4px; font-size: 0.68rem; font-weight: 760; letter-spacing: 0.08em; text-transform: uppercase; color: var(--dt3); }
	.update-status-pill { display: inline-flex; align-items: center; gap: 7px; padding: 6px 10px; border-radius: 999px; border: 1px solid var(--dbd); font-size: 0.73rem; font-weight: 650; color: var(--dt2); white-space: nowrap; }
	.status-dot { width: 7px; height: 7px; border-radius: 999px; background: var(--dt3); }
	.status-live .status-dot { background: var(--bos-status-success); }
	.status-web .status-dot { background: var(--dt4); }
	.update-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 16px; }
	.update-fact { min-width: 0; padding: 12px; border: 1px solid var(--dbd); border-radius: 10px; background: var(--dbg2); }
	.update-fact span { display: block; margin-bottom: 5px; color: var(--dt3); font-size: 0.7rem; font-weight: 620; text-transform: uppercase; letter-spacing: 0.05em; }
	.update-fact strong { display: block; overflow: hidden; color: var(--dt); font-size: 0.88rem; font-weight: 680; text-overflow: ellipsis; white-space: nowrap; }
	.update-callout, .update-progress, .update-error, .update-muted { margin-bottom: 14px; padding: 14px; border: 1px solid var(--dbd); border-radius: 12px; background: var(--dbg2); }
	.update-callout { display: flex; align-items: center; justify-content: space-between; gap: 14px; border-color: color-mix(in srgb, var(--bos-status-success) 35%, var(--dbd)); }
	.update-callout-ready { border-color: color-mix(in srgb, #0ea5e9 35%, var(--dbd)); }
	.update-callout-required { border-color: color-mix(in srgb, #e11d48 35%, var(--dbd)); background: color-mix(in srgb, #e11d48 7%, var(--dbg2)); }
	.callout-title { margin: 0 0 4px; color: var(--dt); font-size: 0.92rem; font-weight: 720; }
	.callout-copy { margin: 0; color: var(--dt3); font-size: 0.8rem; line-height: 1.45; }
	.callout-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
	.progress-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 9px; color: var(--dt); font-size: 0.82rem; font-weight: 650; }
	.progress-track { height: 7px; overflow: hidden; border-radius: 999px; background: var(--dbg); border: 1px solid var(--dbd); }
	.progress-fill { height: 100%; border-radius: inherit; background: var(--dt); transition: width 0.18s ease; }
	.update-progress p { margin: 8px 0 0; color: var(--dt3); font-size: 0.76rem; }
	.update-error { color: #e11d48; background: color-mix(in srgb, #e11d48 8%, var(--dbg2)); border-color: color-mix(in srgb, #e11d48 26%, var(--dbd)); }
	.update-error strong, .update-error span { display: block; font-size: 0.82rem; }
	.update-error span { margin-top: 3px; color: color-mix(in srgb, #e11d48 78%, var(--dt)); }
	.update-muted { color: var(--dt3); font-size: 0.82rem; }
	.update-actions { padding-top: 2px; }
	.notes-backdrop { position: fixed; inset: 0; z-index: 1000; display: grid; place-items: center; padding: 24px; background: rgba(0, 0, 0, 0.38); }
	.notes-scrim { position: absolute; inset: 0; border: 0; background: transparent; cursor: default; }
	.notes-modal { position: relative; width: min(680px, 100%); max-height: min(720px, calc(100vh - 48px)); overflow: hidden; display: flex; flex-direction: column; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; box-shadow: 0 24px 70px rgba(0, 0, 0, 0.36); }
	.notes-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 20px 22px 14px; border-bottom: 1px solid var(--dbd); }
	.notes-head h2 { margin: 0; color: var(--dt); font-size: 1.05rem; font-weight: 740; }
	.notes-close { width: 30px; height: 30px; border: 1px solid var(--dbd); border-radius: 8px; background: var(--dbg2); color: var(--dt2); font-size: 1.2rem; line-height: 1; cursor: pointer; }
	.notes-modal pre { flex: 1; min-height: 180px; max-height: 440px; overflow: auto; margin: 0; padding: 18px 22px; white-space: pre-wrap; color: var(--dt2); font-size: 0.82rem; line-height: 1.55; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
	.notes-actions { display: flex; justify-content: flex-end; gap: 10px; padding: 14px 22px 20px; border-top: 1px solid var(--dbd); }
	@media (max-width: 760px) {
		.update-head, .update-callout { align-items: stretch; flex-direction: column; }
		.update-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
		.callout-actions { justify-content: flex-start; flex-wrap: wrap; }
	}
</style>
