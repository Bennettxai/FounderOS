<!--
  Desktop 3D Permission Prompt

  Beautiful permission request UI that appears when user enters 3D Desktop mode.
  Requests camera and microphone access with clear explanation of features.

  Features:
  - Shows after 2 second delay (let user see 3D Desktop first)
  - Clear explanation of what permissions are used for
  - Privacy assurance (local processing only)
  - Skip option for users who don't want these features
  - Auto-dismisses on successful permission grant
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import {
		desktop3dPermissions,
		cameraPermission,
		microphonePermission
	} from '$lib/services/desktop3dPermissions';

	// Component state
	let showPrompt = $state(false);
	let requesting = $state(false);
	let error = $state('');

	// Show prompt after delay
	onMount(() => {
		// Check if permissions are already granted
		const camPerm = $cameraPermission;
		const micPerm = $microphonePermission;

		if (camPerm === 'granted' && micPerm === 'granted') {
			if (import.meta.env.DEV) console.log('[Permission Prompt] Permissions already granted, skipping prompt');
			return;
		}

		// Show prompt after 2 seconds (let user see 3D Desktop first)
		const timeout = setTimeout(() => {
			// Only show if still in prompt state
			if ($cameraPermission === 'prompt' || $microphonePermission === 'prompt') {
				showPrompt = true;
			}
		}, 2000);

		return () => clearTimeout(timeout);
	});

	/**
	 * Handle permission request
	 */
	async function handleRequestPermissions() {
		requesting = true;
		error = '';

		try {
			const result = await desktop3dPermissions.requestAll();

			if (result.camera && result.microphone) {
				// Both granted - dismiss prompt
				showPrompt = false;
				if (import.meta.env.DEV) console.log('[Permission Prompt] All permissions granted, closing prompt');
			} else if (!result.camera && !result.microphone) {
				// Both denied
				error =
					'Camera and microphone access denied. You can still use 3D Desktop without gesture controls.';
				if (import.meta.env.DEV) console.log('[Permission Prompt] All permissions denied');
			} else {
				// Partial permissions
				const denied = !result.camera ? 'camera' : 'microphone';
				error = `${denied.charAt(0).toUpperCase() + denied.slice(1)} access denied. Some features will be unavailable.`;
				if (import.meta.env.DEV) console.log('[Permission Prompt] Partial permissions granted');
			}
		} catch (err) {
			const errorMsg = err instanceof Error ? err.message : 'Unknown error';
			error = `Failed to request permissions: ${errorMsg}`;
			console.error('[Permission Prompt] Error requesting permissions:', err);
		} finally {
			requesting = false;
		}
	}

	/**
	 * Handle skip button
	 */
	function handleSkip() {
		showPrompt = false;
		if (import.meta.env.DEV) console.log('[Permission Prompt] User skipped permissions');
	}

	/**
	 * Handle retry after error
	 */
	function handleRetry() {
		error = '';
		handleRequestPermissions();
	}
</script>

{#if showPrompt && ($cameraPermission === 'prompt' || $microphonePermission === 'prompt')}
	<div class="prompt-overlay" transition:fade={{ duration: 200 }}>
		<div class="permission-prompt" transition:fly={{ y: 20, duration: 300 }}>
			<div class="prompt-content">
				<!-- Header -->
				<div class="prompt-header">
					<div class="header-icon">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
							/>
						</svg>
					</div>
					<h3>Enable Advanced Controls</h3>
				</div>

				<!-- Description -->
				<p class="prompt-description">3D Desktop can use your camera and microphone for:</p>

				<!-- Feature list -->
				<ul class="feature-list">
					<li>
						<span class="feature-icon">🤚</span>
						<span>Hand tracking and gesture control</span>
					</li>
					<li>
						<span class="feature-icon">🎤</span>
						<span>Voice commands</span>
					</li>
					<li>
						<span class="feature-icon">👏</span>
						<span>Clap and wave gestures</span>
					</li>
					<li>
						<span class="feature-icon">🎯</span>
						<span>Body pointing and presence detection</span>
					</li>
				</ul>

				<!-- Privacy note -->
				<div class="privacy-note">
					<svg class="privacy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/>
					</svg>
					<span
						>All processing happens locally on your device. No video or audio is sent to
						servers.</span
					>
				</div>

				<!-- Error message -->
				{#if error}
					<div class="error-message" transition:fade={{ duration: 200 }}>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span>{error}</span>
					</div>
				{/if}

				<!-- Actions -->
				<div class="prompt-actions">
					<button
						onclick={handleSkip}
						class="btn-pill btn-pill-ghost btn-skip"
						disabled={requesting}
						aria-label="Skip permissions"
					>
						{error ? 'Continue Without' : 'Skip for Now'}
					</button>

					<button
						onclick={error ? handleRetry : handleRequestPermissions}
						class="btn-pill btn-pill-ghost btn-enable"
						disabled={requesting}
						aria-label="Enable camera and microphone"
					>
						{#if requesting}
							<span class="spinner"></span>
							<span>Requesting...</span>
						{:else if error}
							Try Again
						{:else}
							Enable Camera & Mic
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	/* Overlay */
	.prompt-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(8px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
	}

	/* Permission prompt card */
	.permission-prompt {
		background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
		border-radius: 1.5rem;
		max-width: 500px;
		width: 100%;
		box-shadow:
			0 20px 60px rgba(0, 0, 0, 0.5),
			0 0 0 1px rgba(255, 255, 255, 0.1);
		overflow: hidden;
	}

	.prompt-content {
		padding: 2rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	/* Header */
	.prompt-header {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.header-icon {
		width: 3rem;
		height: 3rem;
		background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
		border-radius: 0.75rem;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.header-icon svg {
		width: 1.75rem;
		height: 1.75rem;
		color: white;
	}

	.prompt-header h3 {
		font-size: 1.5rem;
		font-weight: 600;
		margin: 0;
		color: white;
	}

	/* Description */
	.prompt-description {
		font-size: 1rem;
		color: #cbd5e1;
		margin: 0;
		line-height: 1.6;
	}

	/* Feature list */
	.feature-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.feature-list li {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		font-size: 0.95rem;
		color: #e2e8f0;
		line-height: 1.5;
	}

	.feature-icon {
		font-size: 1.25rem;
		flex-shrink: 0;
	}

	/* Privacy note */
	.privacy-note {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 1rem;
		background: rgba(59, 130, 246, 0.1);
		border: 1px solid rgba(59, 130, 246, 0.2);
		border-radius: 0.75rem;
	}

	.privacy-icon {
		width: 1.25rem;
		height: 1.25rem;
		color: #60a5fa;
		flex-shrink: 0;
		margin-top: 0.125rem;
	}

	.privacy-note span {
		font-size: 0.875rem;
		color: #cbd5e1;
		line-height: 1.5;
	}

	/* Error message */
	.error-message {
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 1rem;
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.2);
		border-radius: 0.75rem;
	}

	.error-message svg {
		width: 1.25rem;
		height: 1.25rem;
		color: #f87171;
		flex-shrink: 0;
		margin-top: 0.125rem;
	}

	.error-message span {
		font-size: 0.875rem;
		color: #fca5a5;
		line-height: 1.5;
	}

	/* Actions */
	.prompt-actions {
		display: flex;
		gap: 1rem;
		margin-top: 0.5rem;
	}

	.btn-skip,
	.btn-enable {
		flex: 1;
		padding: 0.875rem 1.5rem;
		border-radius: 0.75rem;
		font-weight: 500;
		font-size: 0.95rem;
		border: none;
		cursor: pointer;
		transition: all 0.2s ease;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
	}

	.btn-skip {
		background: rgba(255, 255, 255, 0.05);
		color: #cbd5e1;
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	.btn-skip:hover:not(:disabled) {
		background: rgba(255, 255, 255, 0.1);
		border-color: rgba(255, 255, 255, 0.2);
	}

	.btn-enable {
		background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
		color: white;
		box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
	}

	.btn-enable:hover:not(:disabled) {
		background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
		box-shadow: 0 6px 16px rgba(59, 130, 246, 0.4);
		transform: translateY(-1px);
	}

	.btn-skip:disabled,
	.btn-enable:disabled {
		opacity: 0.5;
		cursor: not-allowed;
		transform: none;
	}

	/* Spinner */
	.spinner {
		width: 1rem;
		height: 1rem;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* Responsive */
	@media (max-width: 640px) {
		.prompt-content {
			padding: 1.5rem;
			gap: 1.25rem;
		}

		.prompt-header h3 {
			font-size: 1.25rem;
		}

		.prompt-actions {
			flex-direction: column;
		}

		.btn-skip,
		.btn-enable {
			width: 100%;
		}
	}
</style>
