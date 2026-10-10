<!--
	Live Captions Overlay

	Displays real-time voice transcriptions and command recognition feedback
	for the 3D Desktop active listening system.

	Features:
	- Real-time caption display
	- Command recognition feedback
	- Auto-fade after silence
	- Minimal, unobtrusive design
-->

<script lang="ts">
	import { fade, fly } from 'svelte/transition';
	import type { VoiceCommand } from '$lib/services/voiceCommands';

	interface Props {
		/** User's spoken message */
		userMessage?: string;

		/** OSA's response message */
		osaMessage?: string;

		/** Recognized voice command (if any) */
		command?: VoiceCommand | null;

		/** Whether actively listening */
		isListening: boolean;

		/** Whether OSA is speaking */
		isSpeaking?: boolean;
	}

	let { userMessage = '', osaMessage = '', command = null, isListening = false, isSpeaking = false }: Props = $props();

	// Simplified - messages are controlled by parent component timing

	// Get command feedback message
	function getCommandFeedback(cmd: VoiceCommand): string {
		if (!cmd) return '';

		switch (cmd.type) {
			case 'enter_edit_mode':
				return '✓ Entering edit mode';
			case 'exit_edit_mode':
				return '✓ Exiting edit mode';
			case 'save_layout':
				return `✓ Saving layout: ${cmd.name}`;
			case 'load_layout':
				return `✓ Loading layout: ${cmd.name}`;
			case 'focus_module':
				return `✓ Opening ${cmd.module}`;
			case 'close_module':
				return `✓ Closing ${cmd.module}`;
			case 'unfocus':
				return '✓ Unfocusing window';
			case 'switch_view':
				return `✓ Switching to ${cmd.view} view`;
			case 'toggle_auto_rotate':
				return '✓ Toggling auto-rotate';
			case 'zoom_in':
				return '✓ Zooming camera in';
			case 'zoom_out':
				return '✓ Zooming camera out';
			case 'reset_zoom':
				return '✓ Resetting zoom';
			case 'expand_orb':
				return '✓ Expanding orb';
			case 'contract_orb':
				return '✓ Contracting orb';
			case 'resize_window':
				return `✓ Making window ${cmd.direction}`;
			case 'next_window':
				return '✓ Next window';
			case 'previous_window':
				return '✓ Previous window';
			case 'help':
				return '✓ Showing help';
			case 'unknown':
				return `✗ Unknown command: "${cmd.text}"`;
			default:
				return '';
		}
	}

</script>

<div class="live-captions-container">
	<!-- Listening indicator -->
	{#if isListening}
		<div class="listening-indicator" transition:fade={{ duration: 200 }}>
			<div class="pulse"></div>
			<span>Listening...</span>
		</div>
	{/if}

	<!-- OSA Speaking indicator -->
	{#if isSpeaking}
		<div class="speaking-indicator" transition:fade={{ duration: 200 }}>
			<div class="sound-wave">
				<div class="bar"></div>
				<div class="bar"></div>
				<div class="bar"></div>
				<div class="bar"></div>
			</div>
			<span>OSA Speaking...</span>
		</div>
	{/if}

	<!-- User message -->
	{#if userMessage}
		<div class="user-message" transition:fly={{ y: 20, duration: 300 }}>
			<div class="message-label">You:</div>
			<div class="message-text">{userMessage}</div>
		</div>
	{/if}

	<!-- OSA message -->
	{#if osaMessage}
		<div class="osa-message" transition:fly={{ y: 20, duration: 300 }}>
			<div class="message-label">OSA:</div>
			<div class="message-text">{osaMessage}</div>
		</div>
	{/if}

	<!-- Command feedback -->
	{#if command}
		<div class="command-feedback" transition:fly={{ y: 20, duration: 300 }}>
			{getCommandFeedback(command)}
		</div>
	{/if}
</div>

<style>
	.live-captions-container {
		position: fixed;
		bottom: 120px; /* Above the dock */
		left: 50%;
		transform: translateX(-50%);
		z-index: 500;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		pointer-events: none; /* Don't interfere with clicks */
	}

	/* ===== LISTENING INDICATOR ===== */
	.listening-indicator {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: rgba(59, 130, 246, 0.9); /* Blue */
		backdrop-filter: blur(12px);
		border-radius: 20px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
	}

	.pulse {
		width: 8px;
		height: 8px;
		background: white;
		border-radius: 50%;
		animation: pulse 1.5s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
			transform: scale(1);
		}
		50% {
			opacity: 0.5;
			transform: scale(1.2);
		}
	}

	/* ===== OSA SPEAKING INDICATOR ===== */
	.speaking-indicator {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: rgba(168, 85, 247, 0.9); /* Purple for OSA */
		backdrop-filter: blur(12px);
		border-radius: 20px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		box-shadow: 0 4px 12px rgba(168, 85, 247, 0.3);
	}

	.sound-wave {
		display: flex;
		align-items: center;
		gap: 3px;
		height: 16px;
	}

	.sound-wave .bar {
		width: 3px;
		background: white;
		border-radius: 2px;
		animation: soundWave 0.8s ease-in-out infinite;
	}

	.sound-wave .bar:nth-child(1) {
		animation-delay: 0s;
	}

	.sound-wave .bar:nth-child(2) {
		animation-delay: 0.2s;
	}

	.sound-wave .bar:nth-child(3) {
		animation-delay: 0.4s;
	}

	.sound-wave .bar:nth-child(4) {
		animation-delay: 0.6s;
	}

	@keyframes soundWave {
		0%,
		100% {
			height: 4px;
		}
		50% {
			height: 16px;
		}
	}

	/* ===== USER MESSAGE ===== */
	.user-message {
		max-width: 900px; /* Wider for better readability */
		max-height: calc(100vh - 200px); /* INCREASED: Use most of viewport height (was 400px) */
		overflow-y: auto; /* Enable scrolling for very long messages */
		padding: 12px 20px;
		background: rgba(59, 130, 246, 0.9); /* Blue */
		backdrop-filter: blur(12px);
		border-radius: 12px;
		box-shadow: 0 8px 24px rgba(59, 130, 246, 0.3);
	}

	/* ===== OSA MESSAGE ===== */
	.osa-message {
		max-width: 900px; /* Wider for better readability */
		max-height: calc(100vh - 200px); /* INCREASED: Use most of viewport height (was 400px) */
		overflow-y: auto; /* Enable scrolling for very long messages */
		padding: 12px 20px;
		background: rgba(168, 85, 247, 0.9); /* Purple */
		backdrop-filter: blur(12px);
		border-radius: 12px;
		box-shadow: 0 8px 24px rgba(168, 85, 247, 0.3);
	}

	.message-label {
		color: rgba(255, 255, 255, 0.8);
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin-bottom: 4px;
	}

	.message-text {
		color: white;
		font-size: 15px;
		line-height: 1.5;
		word-wrap: break-word; /* Break long words if needed */
		white-space: pre-wrap; /* Preserve line breaks */
		overflow-wrap: break-word; /* Modern browsers */
	}

	/* ===== COMMAND FEEDBACK ===== */
	.command-feedback {
		padding: 10px 18px;
		background: rgba(34, 197, 94, 0.9); /* Green for success */
		backdrop-filter: blur(12px);
		border-radius: 20px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		box-shadow: 0 4px 12px rgba(34, 197, 94, 0.3);
		animation: commandPulse 0.5s ease-out;
	}

	/* Unknown command gets red background */
	.command-feedback:has(:global(.unknown)) {
		background: rgba(239, 68, 68, 0.9); /* Red */
		box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3);
	}

	@keyframes commandPulse {
		0% {
			transform: scale(0.95);
			opacity: 0;
		}
		50% {
			transform: scale(1.05);
		}
		100% {
			transform: scale(1);
			opacity: 1;
		}
	}

	/* ===== DARK MODE STYLES ===== */
	:global(.dark) .captions {
		background: rgba(30, 30, 30, 0.95);
	}

	:global(.dark) .listening-indicator {
		background: rgba(59, 130, 246, 0.85);
	}

	:global(.dark) .speaking-indicator {
		background: rgba(168, 85, 247, 0.85);
	}
</style>
