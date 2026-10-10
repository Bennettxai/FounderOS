<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';

	interface Props {
		isElectron: boolean;
		accessibilityGranted: boolean;
		onRequestAccessibility: () => Promise<void>;
	}

	let { isElectron, accessibilityGranted, onRequestAccessibility }: Props = $props();

	let shortcuts = $state({
		spotlight: '⌘+Space',
		quickChat: '⌘+Shift+Space',
		voiceInput: '⌘+D',
	});

	let recordingKey = $state<string | null>(null);

	// Convert Electron accelerator to display format
	function formatAcceleratorForDisplay(accelerator: string): string {
		if (!accelerator) return '';
		return accelerator
			.replace('CommandOrControl', '⌘')
			.replace('Command', '⌘')
			.replace('Control', '⌃')
			.replace('Shift', '⇧')
			.replace('Alt', '⌥')
			.replace('Option', '⌥')
			.replace(/\+/g, '');
	}

	// Convert display format back to Electron accelerator
	function formatDisplayToAccelerator(display: string): string {
		if (!display) return '';
		let result = display
			.replace('⌘', 'CommandOrControl+')
			.replace('⌃', 'Control+')
			.replace('⇧', 'Shift+')
			.replace('⌥', 'Alt+');
		if (result.endsWith('+')) {
			result = result.slice(0, -1);
		}
		return result;
	}

	function startRecording(key: string) {
		recordingKey = key;
		if (browser) {
			window.addEventListener('keydown', handleRecordKeyDown);
		}
	}

	function stopRecording() {
		recordingKey = null;
		if (browser) {
			window.removeEventListener('keydown', handleRecordKeyDown);
		}
	}

	async function handleRecordKeyDown(event: KeyboardEvent) {
		event.preventDefault();
		event.stopPropagation();

		if (['Control', 'Shift', 'Alt', 'Meta', 'Command'].includes(event.key)) {
			return;
		}

		const parts: string[] = [];

		if (event.metaKey || event.ctrlKey) parts.push('CommandOrControl');
		if (event.shiftKey) parts.push('Shift');
		if (event.altKey) parts.push('Alt');

		let key = event.key.toUpperCase();
		if (key === ' ') key = 'Space';
		if (key === 'ESCAPE') key = 'Escape';
		if (key === 'BACKSPACE') key = 'Backspace';
		if (key === 'TAB') key = 'Tab';
		if (key === 'ENTER') key = 'Enter';
		if (key === '`') key = '`';

		parts.push(key);

		const accelerator = parts.join('+');
		const displayFormat = formatAcceleratorForDisplay(accelerator);

		if (recordingKey) {
			(shortcuts as any)[recordingKey] = displayFormat;

			if (isElectron && browser) {
				try {
					const electron = window as any;
					await electron.electron?.shortcuts?.set(recordingKey, accelerator);
				} catch (e) {
					console.error('Failed to save shortcut:', e);
				}
			}
		}

		stopRecording();
	}

	async function resetShortcuts() {
		if (isElectron && browser) {
			try {
				const electron = (window as any).electron;
				const result = await electron?.shortcuts?.reset();
				if (result?.shortcuts) {
					shortcuts = {
						spotlight: formatAcceleratorForDisplay(result.shortcuts.spotlight),
						quickChat: formatAcceleratorForDisplay(result.shortcuts.quickChat),
						voiceInput: formatAcceleratorForDisplay(result.shortcuts.voiceInput),
					};
				}
			} catch (e) {
				console.error('Failed to reset shortcuts:', e);
			}
		}
	}

	onMount(async () => {
		const electron = (window as any).electron;
		if (browser && electron) {
			try {
				const savedShortcuts = await electron.shortcuts?.get();
				if (savedShortcuts) {
					shortcuts = {
						spotlight: formatAcceleratorForDisplay(savedShortcuts.spotlight),
						quickChat: formatAcceleratorForDisplay(savedShortcuts.quickChat),
						voiceInput: formatAcceleratorForDisplay(savedShortcuts.voiceInput),
					};
				}
			} catch (e) {
				console.error('Failed to load shortcuts:', e);
			}
		}
	});
</script>

<!-- Accessibility Permission Banner -->
{#if isElectron && !accessibilityGranted}
	<div class="accessibility-banner">
		<div class="banner-icon">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
			</svg>
		</div>
		<div class="banner-content">
			<div class="banner-title">Accessibility Permission Required</div>
			<div class="banner-desc">Global shortcuts need accessibility access to work from anywhere on your Mac.</div>
		</div>
		<button class="banner-btn" onclick={onRequestAccessibility}>
			Grant Access
		</button>
	</div>
{/if}

<!-- Keyboard Shortcuts -->
<div class="section">
	<label class="section-title">Global Shortcuts</label>
	<p class="section-subtitle">Click on a shortcut to change it. Press your desired key combination.</p>
	<div class="shortcuts-list">
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Spotlight Search</div>
				<div class="shortcut-desc">Quick search and app launcher</div>
			</div>
			<button
				class="shortcut-key-btn"
				class:recording={recordingKey === 'spotlight'}
				onclick={() => recordingKey === 'spotlight' ? stopRecording() : startRecording('spotlight')}
			>
				{#if recordingKey === 'spotlight'}
					<span class="recording-text">Press keys...</span>
				{:else}
					{shortcuts.spotlight}
				{/if}
			</button>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Quick Chat Popup</div>
				<div class="shortcut-desc">Open AI chat from anywhere</div>
			</div>
			<button
				class="shortcut-key-btn"
				class:recording={recordingKey === 'quickChat'}
				onclick={() => recordingKey === 'quickChat' ? stopRecording() : startRecording('quickChat')}
			>
				{#if recordingKey === 'quickChat'}
					<span class="recording-text">Press keys...</span>
				{:else}
					{shortcuts.quickChat}
				{/if}
			</button>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Voice Input</div>
				<div class="shortcut-desc">Start/stop voice recording</div>
			</div>
			<button
				class="shortcut-key-btn"
				class:recording={recordingKey === 'voiceInput'}
				onclick={() => recordingKey === 'voiceInput' ? stopRecording() : startRecording('voiceInput')}
			>
				{#if recordingKey === 'voiceInput'}
					<span class="recording-text">Press keys...</span>
				{:else}
					{shortcuts.voiceInput}
				{/if}
			</button>
		</div>
	</div>
</div>

<div class="section">
	<label class="section-title">Window Management</label>
	<p class="section-subtitle">System shortcuts (not customizable)</p>
	<div class="shortcuts-list">
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Close Window</div>
				<div class="shortcut-desc">Close the active window</div>
			</div>
			<div class="shortcut-key">⌘W</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Minimize Window</div>
				<div class="shortcut-desc">Minimize to dock</div>
			</div>
			<div class="shortcut-key">⌘M</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Maximize/Restore</div>
				<div class="shortcut-desc">Toggle window fullscreen</div>
			</div>
			<div class="shortcut-key">⌘⇧F</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Cycle Windows</div>
				<div class="shortcut-desc">Switch between open windows</div>
			</div>
			<div class="shortcut-key">⌘`</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Snap Left</div>
				<div class="shortcut-desc">Snap window to left half</div>
			</div>
			<div class="shortcut-key">⌃⌥←</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Snap Right</div>
				<div class="shortcut-desc">Snap window to right half</div>
			</div>
			<div class="shortcut-key">⌃⌥→</div>
		</div>
	</div>
</div>

<div class="section">
	<label class="section-title">Quick Actions</label>
	<p class="section-subtitle">Fast access to common tasks</p>
	<div class="shortcuts-list">
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">New Task</div>
				<div class="shortcut-desc">Create a new task quickly</div>
			</div>
			<div class="shortcut-key">⌘⇧T</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">New Project</div>
				<div class="shortcut-desc">Start a new project</div>
			</div>
			<div class="shortcut-key">⌘⇧P</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">New Note</div>
				<div class="shortcut-desc">Create a quick note</div>
			</div>
			<div class="shortcut-key">⌘⇧N</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Toggle Terminal</div>
				<div class="shortcut-desc">Open/close terminal</div>
			</div>
			<div class="shortcut-key">⌘⇧`</div>
		</div>
	</div>
</div>

<div class="section">
	<label class="section-title">Navigation</label>
	<p class="section-subtitle">Move around the workspace</p>
	<div class="shortcuts-list">
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Go to Dashboard</div>
				<div class="shortcut-desc">Open dashboard view</div>
			</div>
			<div class="shortcut-key">⌘1</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Go to Chat</div>
				<div class="shortcut-desc">Open AI chat</div>
			</div>
			<div class="shortcut-key">⌘2</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Go to Tasks</div>
				<div class="shortcut-desc">Open tasks view</div>
			</div>
			<div class="shortcut-key">⌘3</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Go to Calendar</div>
				<div class="shortcut-desc">Open calendar view</div>
			</div>
			<div class="shortcut-key">⌘4</div>
		</div>
		<div class="shortcut-row">
			<div class="shortcut-info">
				<div class="shortcut-name">Go to Projects</div>
				<div class="shortcut-desc">Open projects view</div>
			</div>
			<div class="shortcut-key">⌘5</div>
		</div>
	</div>
</div>

{#if isElectron}
	<div class="section">
		<button class="reset-shortcuts-btn" onclick={resetShortcuts}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
				<path d="M3 3v5h5"/>
			</svg>
			Reset to Default Shortcuts
		</button>
	</div>
{/if}

<div class="section">
	<div class="shortcut-note">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="12" cy="12" r="10"/>
			<path d="M12 16v-4M12 8h.01"/>
		</svg>
		<span>
			{#if isElectron}
				Global shortcuts work system-wide when BusinessOS Desktop is running. Some shortcuts may conflict with macOS defaults (like ⌘+Space for Spotlight).
			{:else}
				Global shortcuts are only available in the desktop app. Download BusinessOS Desktop to use shortcuts anywhere on your Mac.
			{/if}
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

	/* Accessibility banner */
	.accessibility-banner {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 16px;
		background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
		border: 1px solid #f59e0b;
		border-radius: 10px;
		margin-bottom: 16px;
	}

	.banner-icon {
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #f59e0b;
		border-radius: 8px;
		flex-shrink: 0;
	}

	.banner-icon svg {
		width: 18px;
		height: 18px;
		color: white;
	}

	.banner-content {
		flex: 1;
	}

	.banner-title {
		font-size: 13px;
		font-weight: 600;
		color: #92400e;
	}

	.banner-desc {
		font-size: 11px;
		color: #b45309;
		margin-top: 2px;
	}

	.banner-btn {
		padding: 8px 16px;
		background: #f59e0b;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}

	.banner-btn:hover {
		background: #d97706;
	}

	.shortcuts-list {
		display: flex;
		flex-direction: column;
		background: white;
		border-radius: 8px;
		overflow: hidden;
		border: 1px solid #e5e5e5;
	}

	.shortcut-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid #f0f0f0;
	}

	.shortcut-row:last-child {
		border-bottom: none;
	}

	.shortcut-info {
		flex: 1;
	}

	.shortcut-name {
		font-size: 13px;
		font-weight: 500;
		color: #333;
	}

	.shortcut-desc {
		font-size: 11px;
		color: #999;
		margin-top: 2px;
	}

	.shortcut-key {
		font-family: ui-monospace, 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		font-size: 12px;
		font-weight: 500;
		color: #555;
		background: #f5f5f5;
		border: 1px solid #e0e0e0;
		border-radius: 6px;
		padding: 6px 10px;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.shortcut-key-btn {
		font-family: ui-monospace, 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		font-size: 12px;
		font-weight: 500;
		color: #555;
		background: #f5f5f5;
		border: 1px solid #e0e0e0;
		border-radius: 6px;
		padding: 6px 12px;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
		cursor: pointer;
		transition: all 0.15s;
		min-width: 80px;
		text-align: center;
	}

	.shortcut-key-btn:hover {
		background: #eee;
		border-color: #ccc;
	}

	.shortcut-key-btn.recording {
		background: #3b82f6;
		border-color: #2563eb;
		color: white;
		animation: pulse-recording 1.5s infinite;
	}

	@keyframes pulse-recording {
		0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
		50% { box-shadow: 0 0 0 8px rgba(59, 130, 246, 0); }
	}

	.recording-text {
		font-family: inherit;
		font-style: italic;
		font-size: 11px;
	}

	.reset-shortcuts-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		width: 100%;
		padding: 10px 16px;
		background: #f5f5f5;
		border: 1px solid #e0e0e0;
		border-radius: 8px;
		color: #666;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}

	.reset-shortcuts-btn:hover {
		background: #eee;
		color: #333;
	}

	.reset-shortcuts-btn svg {
		width: 16px;
		height: 16px;
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
