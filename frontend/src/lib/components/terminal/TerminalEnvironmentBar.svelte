<script lang="ts">
	import type { EnvironmentMode, EnvironmentInfo } from '$lib/stores/terminal/terminalTypes';

	interface Props {
		currentMode: EnvironmentMode;
		environmentInfo: EnvironmentInfo;
		onModeChange: (mode: EnvironmentMode) => void;
		onExitSandbox?: () => void;
		creatingSandbox?: boolean;
	}

	let { currentMode, environmentInfo, onModeChange, onExitSandbox, creatingSandbox = false }: Props = $props();

	let showConfirmDialog = $state(false);
	let pendingMode = $state<EnvironmentMode | null>(null);

	const modes: { id: EnvironmentMode; label: string; color: string; shortLabel: string }[] = [
		{ id: 'sandbox', label: 'Sandbox', color: '#eab308', shortLabel: 'SBX' },
		{ id: 'production', label: 'Production', color: '#ef4444', shortLabel: 'PROD' },
		{ id: 'local', label: 'Local', color: '#d946ef', shortLabel: 'LCL' },
	];

	function handleModeClick(mode: EnvironmentMode) {
		if (mode === currentMode) return;

		// If leaving sandbox with changes, show confirm dialog
		if (currentMode === 'sandbox' && (environmentInfo.changeCount ?? 0) > 0) {
			pendingMode = mode;
			if (mode === 'production') {
				// Trigger sandbox analysis flow
				onExitSandbox?.();
				return;
			}
			showConfirmDialog = true;
			return;
		}

		onModeChange(mode);
	}

	function confirmSwitch() {
		if (pendingMode) {
			onModeChange(pendingMode);
		}
		showConfirmDialog = false;
		pendingMode = null;
	}

	function cancelSwitch() {
		showConfirmDialog = false;
		pendingMode = null;
	}

	const osIcon = $derived.by(() => {
		if (environmentInfo.os === 'darwin') return '\u{F8FF}'; // Apple logo placeholder
		if (environmentInfo.os === 'linux') return '\u{1F427}'; // Penguin
		if (environmentInfo.os === 'windows') return '\u{1F5D4}'; // Windows
		return '';
	});

	const sandboxProviderLabel = $derived.by(() => {
		if (currentMode !== 'sandbox') return '';
		const source = environmentInfo.miosaAccountSource ?? 'local';
		const remoteState = environmentInfo.sandboxRemoteState ?? 'local-ready';
		if (remoteState === 'remote-attached') {
			return source === 'businessos' ? 'MIOSA BOS' : 'MIOSA USER';
		}
		if (source === 'businessos') return 'BOS tenant';
		if (source === 'user') return 'User key';
		return 'Local CLI';
	});
</script>

<div class="env-bar">
	<div class="env-modes">
		{#each modes as mode (mode.id)}
			<button
				class="env-pill"
				class:active={mode.id === currentMode}
				style="--env-color: {mode.color}"
				onclick={() => handleModeClick(mode.id)}
				disabled={creatingSandbox}
				title="{mode.label} mode"
			>
				{mode.shortLabel}
				{#if mode.id === 'sandbox' && (environmentInfo.changeCount ?? 0) > 0}
					<span class="change-badge">{environmentInfo.changeCount}</span>
				{/if}
				{#if mode.id === 'local' && currentMode === 'local' && environmentInfo.os}
					<span class="os-tag">{environmentInfo.os === 'darwin' ? 'macOS' : environmentInfo.os}</span>
				{/if}
			</button>
		{/each}
	</div>

	{#if currentMode === 'sandbox'}
		<div class="env-status">
			<span class="tracking-dot"></span>
			<span class="tracking-text">
				{creatingSandbox
					? 'Creating MIOSA sandbox'
					: (environmentInfo.changeCount ?? 0) > 0
					? `${environmentInfo.changeCount} changes tracked`
					: 'Tracking changes'}
			</span>
			{#if sandboxProviderLabel}
				<span class="sandbox-provider" title={environmentInfo.miosaPreviewUrl ?? environmentInfo.sandboxRemoteState ?? 'Local sandbox'}>
					{sandboxProviderLabel}
				</span>
			{/if}
		</div>
	{/if}
</div>

{#if showConfirmDialog}
	<div class="confirm-overlay" onclick={cancelSwitch} onkeydown={(e) => e.key === 'Escape' && cancelSwitch()} role="dialog" tabindex="-1">
		<div class="confirm-dialog" onclick={(e) => e.stopPropagation()} role="alertdialog" aria-label="Confirm mode switch">
			<p class="confirm-text">You have {environmentInfo.changeCount ?? 0} uncommitted changes in sandbox mode. Switch anyway?</p>
			<div class="confirm-actions">
				<button class="confirm-btn cancel" onclick={cancelSwitch}>Stay in Sandbox</button>
				<button class="confirm-btn proceed" onclick={confirmSwitch}>Switch Anyway</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.env-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 24px;
		padding: 0 8px;
		background: #080808;
		border-bottom: 1px solid #151515;
		flex-shrink: 0;
	}

	.env-modes {
		display: flex;
		gap: 3px;
		align-items: center;
	}

	.env-pill {
		display: flex;
		align-items: center;
		gap: 3px;
		padding: 2px 8px;
		border-radius: 3px;
		border: 1px solid #222;
		background: transparent;
		color: #555;
		font-family: 'SF Mono', monospace;
		font-size: 9px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.12s ease;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.env-pill:hover {
		border-color: var(--env-color, #888);
		color: var(--env-color, #888);
	}

	.env-pill.active {
		background: color-mix(in srgb, var(--env-color) 20%, transparent);
		color: var(--env-color);
		border-color: color-mix(in srgb, var(--env-color) 40%, transparent);
	}

	.env-pill:disabled {
		opacity: 0.55;
		cursor: progress;
	}

	.change-badge {
		background: var(--env-color);
		color: #000;
		font-size: 8px;
		font-weight: 700;
		padding: 0 4px;
		border-radius: 6px;
		min-width: 14px;
		text-align: center;
	}

	.os-tag {
		font-size: 8px;
		color: #666;
		font-weight: 400;
	}

	.env-status {
		display: flex;
		align-items: center;
		gap: 5px;
	}

	.tracking-dot {
		width: 5px;
		height: 5px;
		border-radius: 50%;
		background: #eab308;
		animation: pulse-dot 2s ease-in-out infinite;
	}

	@keyframes pulse-dot {
		0%, 100% { opacity: 0.4; }
		50% { opacity: 1; }
	}

	.tracking-text {
		font-family: 'SF Mono', monospace;
		font-size: 9px;
		color: #555;
	}

	.sandbox-provider {
		margin-left: 4px;
		padding: 1px 5px;
		border: 1px solid #283246;
		border-radius: 3px;
		color: #7dd3fc;
		background: #0ea5e914;
		font-family: 'SF Mono', monospace;
		font-size: 8px;
		font-weight: 700;
		letter-spacing: 0.35px;
		text-transform: uppercase;
	}

	/* Confirm Dialog */
	.confirm-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.confirm-dialog {
		background: #1a1a1a;
		border: 1px solid #333;
		border-radius: 8px;
		padding: 16px 20px;
		max-width: 360px;
		width: 90%;
	}

	.confirm-text {
		color: #ccc;
		font-family: 'SF Mono', monospace;
		font-size: 12px;
		margin: 0 0 12px;
		line-height: 1.5;
	}

	.confirm-actions {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
	}

	.confirm-btn {
		padding: 5px 12px;
		border-radius: 4px;
		border: 1px solid #333;
		font-family: 'SF Mono', monospace;
		font-size: 11px;
		cursor: pointer;
		font-weight: 500;
	}

	.confirm-btn.cancel {
		background: transparent;
		color: #eab308;
		border-color: #eab30855;
	}

	.confirm-btn.cancel:hover {
		background: #eab30820;
	}

	.confirm-btn.proceed {
		background: #333;
		color: #ccc;
		border-color: #444;
	}

	.confirm-btn.proceed:hover {
		background: #444;
	}
</style>
