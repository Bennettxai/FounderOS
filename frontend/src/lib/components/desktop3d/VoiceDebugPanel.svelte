<!--
	Voice Debug Panel

	Shows real-time status of voice system for debugging
-->

<script lang="ts">
	import { fade } from 'svelte/transition';

	interface Props {
		isListening: boolean;
		isSpeaking: boolean;
		currentTranscript: string;
		logs: string[];
	}

	let { isListening, isSpeaking, currentTranscript, logs }: Props = $props();

	let expanded = $state(false);
</script>

<button class="btn-pill btn-pill-ghost debug-toggle" onclick={() => expanded = !expanded}>
	🐛 Debug
</button>

{#if expanded}
	<div class="debug-panel" transition:fade={{ duration: 200 }}>
		<div class="debug-header">
			<span>Voice System Debug</span>
			<button onclick={() => expanded = false}>×</button>
		</div>

		<div class="debug-content">
			<!-- Status -->
			<div class="debug-section">
				<div class="debug-title">Status</div>
				<div class="debug-row">
					<span>Listening:</span>
					<span class:active={isListening}>{isListening ? '🎤 YES' : '❌ NO'}</span>
				</div>
				<div class="debug-row">
					<span>Speaking:</span>
					<span class:active={isSpeaking}>{isSpeaking ? '🔊 YES' : '❌ NO'}</span>
				</div>
				<div class="debug-row">
					<span>Transcript:</span>
					<span class="transcript">{currentTranscript || '(empty)'}</span>
				</div>
			</div>

			<!-- Logs -->
			<div class="debug-section">
				<div class="debug-title">Logs (last 10)</div>
				<div class="debug-logs">
					{#each logs.slice(-10).reverse() as log}
						<div class="log-line">{log}</div>
					{/each}
				</div>
			</div>

			<!-- Actions -->
			<div class="debug-section">
				<div class="debug-title">Test Actions</div>
				<button class="btn-pill btn-pill-ghost debug-btn" onclick={() => { if (import.meta.env.DEV) console.log('Testing mic...'); }}>
					Test Microphone
				</button>
				<button class="btn-pill btn-pill-ghost debug-btn" onclick={() => { if (import.meta.env.DEV) console.log('Testing TTS...'); }}>
					Test OSA Voice
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.debug-toggle {
		position: fixed;
		top: 60px;
		right: 200px;
		padding: 6px 12px;
		background: rgba(255, 200, 0, 0.9);
		border: none;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
		z-index: 1000;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
	}

	.debug-toggle:hover {
		background: rgba(255, 200, 0, 1);
	}

	.debug-panel {
		position: fixed;
		top: 100px;
		right: 20px;
		width: 350px;
		max-height: 600px;
		background: rgba(0, 0, 0, 0.95);
		border: 1px solid rgba(255, 200, 0, 0.3);
		border-radius: 12px;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
		z-index: 1000;
		overflow: hidden;
		font-family: 'Courier New', monospace;
		font-size: 12px;
	}

	.debug-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 12px;
		background: rgba(255, 200, 0, 0.2);
		border-bottom: 1px solid rgba(255, 200, 0, 0.3);
		color: #ffc800;
		font-weight: 600;
	}

	.debug-header button {
		background: none;
		border: none;
		color: #ffc800;
		font-size: 20px;
		cursor: pointer;
		padding: 0 4px;
	}

	.debug-content {
		padding: 12px;
		max-height: 540px;
		overflow-y: auto;
	}

	.debug-section {
		margin-bottom: 16px;
	}

	.debug-title {
		color: #ffc800;
		font-weight: 600;
		margin-bottom: 8px;
		text-transform: uppercase;
		font-size: 11px;
		letter-spacing: 0.5px;
	}

	.debug-row {
		display: flex;
		justify-content: space-between;
		padding: 6px 0;
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
		color: #aaa;
	}

	.debug-row span:last-child {
		color: #666;
	}

	.debug-row span.active {
		color: #00ff00;
		font-weight: 600;
	}

	.debug-row .transcript {
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: #fff;
	}

	.debug-logs {
		max-height: 200px;
		overflow-y: auto;
		background: rgba(0, 0, 0, 0.3);
		border-radius: 6px;
		padding: 8px;
	}

	.log-line {
		padding: 4px 0;
		color: #888;
		font-size: 11px;
		border-bottom: 1px solid rgba(255, 255, 255, 0.03);
	}

	.log-line:last-child {
		border-bottom: none;
	}

	.debug-btn {
		display: block;
		width: 100%;
		padding: 8px;
		margin-bottom: 6px;
		background: rgba(255, 200, 0, 0.1);
		border: 1px solid rgba(255, 200, 0, 0.3);
		border-radius: 6px;
		color: #ffc800;
		cursor: pointer;
		font-size: 11px;
		font-weight: 600;
	}

	.debug-btn:hover {
		background: rgba(255, 200, 0, 0.2);
	}

	:global(.dark) .debug-panel {
		background: rgba(20, 20, 20, 0.98);
	}
</style>
