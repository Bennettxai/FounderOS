<script lang="ts">
	import type { useVoiceRecorder } from '$lib/hooks/useVoiceRecorder.svelte';

	type VoiceRecorder = ReturnType<typeof useVoiceRecorder>;

	interface Props {
		recorder: VoiceRecorder;
	}

	let { recorder }: Props = $props();
</script>

{#if recorder.isRecording}
	<div class="recording-area">
		{#if recorder.liveTranscript}
			<div class="live-transcript">{recorder.liveTranscript}</div>
		{:else}
			<div class="live-transcript placeholder">Listening...</div>
		{/if}
		<div class="waveform-bar">
			<button class="cancel-btn" onclick={recorder.cancelRecording} title="Cancel" aria-label="Cancel recording">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
				</svg>
			</button>
			<div class="waveform">
				{#each recorder.waveformBars as height}
					<div class="bar" style="height: {height}px"></div>
				{/each}
			</div>
			<span class="duration">{recorder.recordingTimeDisplay}</span>
			<button class="confirm-btn" onclick={recorder.stopRecording} title="Done" aria-label="Stop recording">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<polyline points="20 6 9 17 4 12" />
				</svg>
			</button>
		</div>
	</div>
{/if}

<style>
	.recording-area {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.live-transcript {
		font-size: 14px;
		color: #111;
		min-height: 24px;
	}

	.live-transcript.placeholder {
		color: #999;
	}

	.waveform-bar {
		display: flex;
		align-items: center;
		gap: 10px;
		background: #1f2937;
		border-radius: 24px;
		padding: 8px 14px;
	}

	.waveform {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 2px;
		height: 24px;
	}

	.waveform .bar {
		width: 2px;
		background: white;
		border-radius: 1px;
		transition: height 0.05s;
	}

	.duration {
		font-size: 12px;
		font-family: monospace;
		color: white;
		min-width: 36px;
		text-align: right;
	}

	.cancel-btn,
	.confirm-btn {
		width: 28px;
		height: 28px;
		border: none;
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.15s;
	}

	.cancel-btn {
		background: transparent;
		color: #9ca3af;
	}

	.cancel-btn:hover {
		color: white;
	}

	.confirm-btn {
		background: white;
		color: #1f2937;
	}

	.confirm-btn:hover {
		background: #e5e7eb;
	}

	.cancel-btn svg,
	.confirm-btn svg {
		width: 16px;
		height: 16px;
	}

	:global(.dark) .live-transcript {
		color: #f5f5f7;
	}

	:global(.dark) .live-transcript.placeholder {
		color: #6e6e73;
	}
</style>
