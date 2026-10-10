<script lang="ts">
	interface RecorderState {
		isRecording: boolean;
		liveTranscript: string;
		waveformBars: number[];
		recordingTimeDisplay: string;
		toggleRecording: () => void;
		stopRecording: () => void;
		cancelRecording: () => void;
	}

	interface Props {
		inputValue: string;
		isLoading: boolean;
		isCapturingScreenshot: boolean;
		recorder: RecorderState;
		onKeyDown: (e: KeyboardEvent) => void;
		onSubmit: () => void;
		onCaptureScreenshot: () => void;
		inputElement?: HTMLTextAreaElement;
	}

	let {
		inputValue = $bindable(),
		isLoading,
		isCapturingScreenshot,
		recorder,
		onKeyDown,
		onSubmit,
		onCaptureScreenshot,
		inputElement = $bindable(),
	}: Props = $props();
</script>

<div class="input-area">
	{#if recorder.isRecording}
		<!-- Recording UI with waveform -->
		<div class="recording-ui">
			<!-- Live transcript -->
			{#if recorder.liveTranscript}
				<div class="live-transcript">{recorder.liveTranscript}</div>
			{:else}
				<div class="live-transcript placeholder">Listening...</div>
			{/if}
			<!-- Waveform bar -->
			<div class="waveform-bar">
				<button class="cancel-btn" onclick={recorder.cancelRecording} title="Cancel" aria-label="Cancel recording">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
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
						<polyline points="20 6 9 17 4 12"/>
					</svg>
				</button>
			</div>
		</div>
	{:else}
		<button
			class="screenshot-btn"
			onclick={onCaptureScreenshot}
			disabled={isCapturingScreenshot}
			title="Capture screenshot (Cmd+Shift+S)"
			aria-label="Capture screenshot"
		>
			{#if isCapturingScreenshot}
				<div class="mini-spinner"></div>
			{:else}
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
					<circle cx="8.5" cy="8.5" r="1.5"/>
					<polyline points="21 15 16 10 5 21"/>
				</svg>
			{/if}
		</button>
		<button
			class="mic-btn"
			onclick={recorder.toggleRecording}
			title="Voice input (Cmd+D)"
			aria-label="Toggle voice recording"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
				<path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
				<line x1="12" y1="19" x2="12" y2="23"/>
				<line x1="8" y1="23" x2="16" y2="23"/>
			</svg>
		</button>
		<textarea
			bind:this={inputElement}
			bind:value={inputValue}
			onkeydown={onKeyDown}
			placeholder="Ask anything..."
			rows="1"
		></textarea>
		<button
			class="send-btn"
			onclick={onSubmit}
			disabled={!inputValue.trim() || isLoading}
			aria-label="Send message"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="22" y1="2" x2="11" y2="13"/>
				<polygon points="22 2 15 22 11 13 2 9 22 2"/>
			</svg>
		</button>
	{/if}
</div>

<style>
	.input-area {
		display: flex;
		align-items: flex-end;
		gap: 8px;
		padding: 12px 16px;
		border-top: 1px solid rgba(0, 0, 0, 0.08);
		background: rgba(249, 250, 251, 0.9);
	}

	.mic-btn {
		width: 40px;
		height: 40px;
		border: none;
		background: #f3f4f6;
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #666;
		transition: all 0.15s;
		flex-shrink: 0;
	}

	.mic-btn:hover {
		background: #e5e7eb;
		color: #111;
	}

	.mic-btn svg {
		width: 20px;
		height: 20px;
	}

	textarea {
		flex: 1;
		padding: 10px 14px;
		border: 1px solid rgba(0, 0, 0, 0.1);
		border-radius: 20px;
		font-size: 14px;
		font-family: inherit;
		resize: none;
		outline: none;
		background: white;
		max-height: 120px;
		line-height: 1.4;
	}

	textarea:focus {
		border-color: #111;
	}

	textarea:disabled {
		background: #f9fafb;
		color: #999;
	}

	.send-btn {
		width: 40px;
		height: 40px;
		border: none;
		background: #111;
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		transition: all 0.15s;
		flex-shrink: 0;
	}

	.send-btn:hover:not(:disabled) {
		background: #333;
	}

	.send-btn:disabled {
		background: #d1d5db;
		cursor: not-allowed;
	}

	.send-btn svg {
		width: 18px;
		height: 18px;
	}

	/* Recording UI */
	.recording-ui {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.live-transcript {
		font-size: 13px;
		color: #111;
		min-height: 20px;
		animation: pulse 2s infinite;
	}

	.live-transcript.placeholder {
		color: #999;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.6; }
	}

	.waveform-bar {
		display: flex;
		align-items: center;
		gap: 8px;
		background: #1f2937;
		border-radius: 24px;
		padding: 6px 12px;
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
	}

	.duration {
		font-size: 12px;
		font-family: monospace;
		color: white;
		min-width: 32px;
		text-align: right;
	}

	.cancel-btn, .confirm-btn {
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

	.cancel-btn svg, .confirm-btn svg {
		width: 16px;
		height: 16px;
	}

	/* Screenshot button */
	.screenshot-btn {
		width: 36px;
		height: 36px;
		border: none;
		background: #f3f4f6;
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #666;
		transition: all 0.15s;
		flex-shrink: 0;
	}

	.screenshot-btn:hover:not(:disabled) {
		background: #e5e7eb;
		color: #111;
	}

	.screenshot-btn:disabled {
		cursor: not-allowed;
		opacity: 0.6;
	}

	.screenshot-btn svg {
		width: 18px;
		height: 18px;
	}

	.mini-spinner {
		width: 16px;
		height: 16px;
		border: 2px solid #e5e7eb;
		border-top-color: #111;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Dark mode */
	:global(.dark) .input-area {
		background: rgba(44, 44, 46, 0.9);
		border-top-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) textarea {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		color: #f5f5f7;
	}

	:global(.dark) textarea:focus {
		border-color: #0A84FF;
	}

	:global(.dark) textarea::placeholder {
		color: #6e6e73;
	}

	:global(.dark) .send-btn {
		background: #0A84FF;
	}

	:global(.dark) .send-btn:disabled {
		background: #3a3a3c;
		color: #6e6e73;
	}

	:global(.dark) .mic-btn,
	:global(.dark) .screenshot-btn {
		background: #3a3a3c;
		color: #a1a1a6;
	}

	:global(.dark) .mic-btn:hover,
	:global(.dark) .screenshot-btn:hover:not(:disabled) {
		background: #48484a;
		color: #f5f5f7;
	}
</style>
