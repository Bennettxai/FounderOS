<!--
	ChatInput.svelte
	Glassmorphism input with circular send/stop button and voice recording.
	Handles submission lifecycle, auto-resize, and microphone transcription.
	Voice recording ported from BusinessOS2 Dock.svelte.
-->
<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { browser } from '$app/environment';
	import { osaStore } from '$lib/stores/osa';
	import { voiceTranscription } from '$lib/services/voiceTranscriptionService';
	import { getActiveWorkspaceHeaders, getApiBaseUrl } from '$lib/api/base';

	import type { Snippet } from 'svelte';

	interface Props {
		compact?: boolean;
		placeholder?: string;
		onfocus?: () => void;
		onmetrics?: (metrics: { charCount: number; lineCount: number }) => void;
		onattach?: () => void;
		actionBarCenter?: Snippet;
	}

	let { compact = false, placeholder, onfocus, onmetrics, onattach, actionBarCenter }: Props = $props();

	let inputValue = $state('');
	let inputElement: HTMLTextAreaElement | undefined = $state(undefined);

	let isStreaming = $derived($osaStore.isStreaming);

	// Voice recording state
	let isRecording = $state(false);
	let isStartingRecording = $state(false);
	let mediaRecorder: MediaRecorder | null = null;
	let mediaStream: MediaStream | null = null;
	let audioChunks: Blob[] = [];
	let recordingDuration = $state(0);
	let recordingInterval: number | null = null;

	// Audio visualization
	let audioContext: AudioContext | null = null;
	let analyser: AnalyserNode | null = null;
	let audioSource: MediaStreamAudioSourceNode | null = null;
	let audioDataArray: Uint8Array | null = null;
	let waveformBars = $state<number[]>(Array(10).fill(2));
	let animationFrameId: number | null = null;

	// Transcription abort controller (fallback server-side)
	let transcriptionAbortController: AbortController | null = null;
	let isTranscribing = $state(false);

	// Web Speech API real-time transcription
	let useWebSpeech = $state(false);
	let interimText = $state('');
	let finalText = $state('');

	/** Holds the live text captured at stop-time — shown while server transcription runs */
	let pendingTranscript = $state('');

	export function focus() {
		inputElement?.focus();
	}

	async function handleSend() {
		const trimmed = inputValue.trim();
		if (!trimmed || isStreaming) return;

		inputValue = '';
		resetHeight();
		await osaStore.sendMessage(trimmed);
		inputElement?.focus();
	}

	function handleStop() {
		osaStore.cancelStream();
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSend();
		} else if (e.key === 'd' && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			toggleRecording();
		} else if (e.key === 'Escape' && isRecording) {
			cancelRecording();
		}
	}

	function handleInput() {
		if (!inputElement) return;
		inputElement.style.height = 'auto';
		const max = compact ? 38 : 200;
		inputElement.style.height = `${Math.min(inputElement.scrollHeight, max)}px`;
	}

	function resetHeight() {
		if (!inputElement) return;
		inputElement.style.height = 'auto';
	}

	/** Append transcribed text to existing input (voice adds to typed text) */
	async function appendToInput(text: string) {
		const existing = inputValue.trim();
		inputValue = existing ? `${existing} ${text}` : text;
		await tick();
		handleInput();
		inputElement?.focus();
	}

	let hasInput = $derived(inputValue.trim().length > 0);

	// Report char/line metrics to parent for width tier calculation
	$effect(() => {
		if (onmetrics) {
			const charCount = inputValue.length;
			const lineCount = (inputValue.match(/\n/g) || []).length + 1;
			onmetrics({ charCount, lineCount });
		}
	});

	// ===== VOICE RECORDING =====

	function formatDuration(seconds: number): string {
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	function cleanupAudioResources() {
		if (animationFrameId) {
			cancelAnimationFrame(animationFrameId);
			animationFrameId = null;
		}
		if (recordingInterval) {
			clearInterval(recordingInterval);
			recordingInterval = null;
		}
		if (mediaRecorder && mediaRecorder.state !== 'inactive') {
			mediaRecorder.stop();
			mediaRecorder = null;
		}
		if (mediaStream) {
			mediaStream.getTracks().forEach(track => track.stop());
			mediaStream = null;
		}
		if (audioSource) {
			audioSource.disconnect();
			audioSource = null;
		}
		if (analyser) {
			analyser.disconnect();
			analyser = null;
		}
		if (audioContext && audioContext.state !== 'closed') {
			audioContext.close().catch(() => {});
			audioContext = null;
		}
		if (transcriptionAbortController) {
			transcriptionAbortController.abort();
			transcriptionAbortController = null;
		}
		if (voiceTranscription.isListening()) {
			voiceTranscription.stop();
		}
		isRecording = false;
		isStartingRecording = false;
		recordingDuration = 0;
		audioChunks = [];
		audioDataArray = null;
		waveformBars = Array(10).fill(2);
		useWebSpeech = false;
		interimText = '';
		finalText = '';
		pendingTranscript = '';
	}

	async function startRecording() {
		if (isStartingRecording || isRecording) return;
		isStartingRecording = true;

		try {
			const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
			mediaStream = stream;

			// Audio analyser for waveform visualization
			audioContext = new AudioContext();
			analyser = audioContext.createAnalyser();
			audioSource = audioContext.createMediaStreamSource(stream);
			audioSource.connect(analyser);
			analyser.fftSize = 64;
			audioDataArray = new Uint8Array(analyser.frequencyBinCount);

			// ALWAYS start MediaRecorder — this is the reliable fallback for
			// server-side transcription (Whisper) when Web Speech API fails
			mediaRecorder = new MediaRecorder(stream);
			audioChunks = [];
			mediaRecorder.ondataavailable = (event) => {
				audioChunks.push(event.data);
			};
			mediaRecorder.start();

			// ALSO try Web Speech API for real-time live preview (best-effort)
			const webSpeechStarted = await voiceTranscription.start(
				(text, isFinal) => {
					if (isFinal) {
						finalText = finalText ? `${finalText} ${text}` : text;
						interimText = '';
					} else {
						interimText = text;
					}
				},
				() => { /* recognition ended — handled by stop */ }
			);
			useWebSpeech = webSpeechStarted;

			isRecording = true;
			recordingDuration = 0;

			recordingInterval = window.setInterval(() => {
				recordingDuration++;
			}, 1000);

			updateWaveform();
		} catch (error) {
			console.error('Failed to start recording:', error);
			cleanupAudioResources();
		} finally {
			isStartingRecording = false;
		}
	}

	async function stopRecording() {
		const wasUsingWebSpeech = useWebSpeech;

		// 1. Cleanup timers and audio resources (keep mediaRecorder + Web Speech alive)
		if (recordingInterval) {
			clearInterval(recordingInterval);
			recordingInterval = null;
		}
		if (animationFrameId) {
			cancelAnimationFrame(animationFrameId);
			animationFrameId = null;
		}
		if (audioSource) { audioSource.disconnect(); audioSource = null; }
		if (analyser) { analyser.disconnect(); analyser = null; }
		if (audioContext && audioContext.state !== 'closed') {
			audioContext.close().catch(() => {});
			audioContext = null;
		}
		waveformBars = Array(10).fill(2);

		// 2. Snapshot live text for display while processing
		pendingTranscript = (finalText + (interimText ? ` ${interimText}` : '')).trim();

		// 3. Switch UI to textarea (but Web Speech callback still active)
		isRecording = false;
		useWebSpeech = false;
		isTranscribing = true;
		await tick();

		// 3. Stop MediaRecorder — onstop callback handles all text capture
		if (mediaRecorder && mediaRecorder.state !== 'inactive') {
			mediaRecorder.onstop = async () => {
				const audioBlob = new Blob(audioChunks, { type: 'audio/webm' });
				if (mediaStream) {
					mediaStream.getTracks().forEach(track => track.stop());
					mediaStream = null;
				}

				// Give Web Speech API ~250ms to flush final results before reading
				if (wasUsingWebSpeech) {
					await new Promise(resolve => setTimeout(resolve, 250));
				}

				// Capture accumulated live text, THEN stop Web Speech
				const liveText = (finalText + (interimText ? ` ${interimText}` : '')).trim();
				if (import.meta.env.DEV) console.log('[OSA Voice] Captured live text:', JSON.stringify(liveText), 'final:', JSON.stringify(finalText), 'interim:', JSON.stringify(interimText));
				if (wasUsingWebSpeech) {
					voiceTranscription.stop();
				}
				finalText = '';
				interimText = '';

				// Fast path: Web Speech gave us text — append to existing input
				if (liveText) {
					const existing = inputValue.trim();
					inputValue = existing ? `${existing} ${liveText}` : liveText;
					pendingTranscript = '';
					isTranscribing = false;
					await tick();
					handleInput();
					inputElement?.focus();
					return;
				}

				// Slow path: try server transcription (Whisper)
				await transcribeAudio(audioBlob);
			};
			mediaRecorder.stop();
		} else {
			// No active recorder — capture whatever Web Speech accumulated
			const liveText = (finalText + (interimText ? ` ${interimText}` : '')).trim();
			if (wasUsingWebSpeech) {
				voiceTranscription.stop();
			}
			finalText = '';
			interimText = '';
			pendingTranscript = '';
			isTranscribing = false;

			if (mediaStream) {
				mediaStream.getTracks().forEach(track => track.stop());
				mediaStream = null;
			}

			if (liveText) {
				const existing = inputValue.trim();
				inputValue = existing ? `${existing} ${liveText}` : liveText;
				await tick();
				handleInput();
				inputElement?.focus();
			}
		}
	}

	function cancelRecording() {
		cleanupAudioResources();
	}

	function toggleRecording() {
		if (isRecording) {
			stopRecording();
		} else {
			startRecording();
		}
	}

	function updateWaveform() {
		if (!isRecording || !analyser || !audioDataArray) return;

		analyser.getByteTimeDomainData(audioDataArray as Uint8Array<ArrayBuffer>);

		const newBars: number[] = [];
		const step = Math.floor(audioDataArray.length / 10);

		for (let i = 0; i < 10; i++) {
			const index = i * step;
			const value = audioDataArray[index];
			const deviation = Math.abs(value - 128);
			const height = Math.max(2, Math.min(18, 2 + (deviation / 128) * 32));
			newBars.push(height);
		}

		waveformBars = newBars;
		animationFrameId = requestAnimationFrame(updateWaveform);
	}

	async function transcribeAudio(audioBlob: Blob) {
		// isTranscribing is already set by stopRecording before this runs
		transcriptionAbortController = new AbortController();
		const timeoutId = setTimeout(() => {
			transcriptionAbortController?.abort();
		}, 15000);

		// Keep a local copy of the pending text to use as fallback
		const fallbackText = pendingTranscript;

		try {
			const formData = new FormData();
			formData.append('audio', audioBlob, 'recording.webm');

			const response = await fetch(`${getApiBaseUrl()}/transcribe`, {
				method: 'POST',
				credentials: 'include',
				headers: getActiveWorkspaceHeaders(),
				body: formData,
				signal: transcriptionAbortController.signal
			});

			if (response.ok) {
				const data = await response.json();
				if (data.text) {
					appendToInput(data.text);
					return;
				}
			} else {
				console.warn('[OSA Voice] Server transcription unavailable (HTTP', response.status, ')');
			}

			// Server returned empty or failed — use pending live text if available
			if (fallbackText) {
				if (import.meta.env.DEV) console.log('[OSA Voice] Using live transcript fallback:', fallbackText);
				appendToInput(fallbackText);
			}
		} catch (error) {
			if (error instanceof Error && error.name !== 'AbortError') {
				console.warn('[OSA Voice] Server transcription failed:', error.message);
			}
			// Use live text fallback on error too
			if (fallbackText) {
				appendToInput(fallbackText);
			}
		} finally {
			clearTimeout(timeoutId);
			transcriptionAbortController = null;
			isTranscribing = false;
			pendingTranscript = '';
		}
	}

	onDestroy(() => {
		cleanupAudioResources();
	});
</script>

{#if isRecording}
	<!-- Recording bar with live transcription -->
	<div class="recording-bar-container">
		<button class="rec-action cancel" onclick={cancelRecording} aria-label="Cancel recording">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
			</svg>
		</button>
		{#if finalText || interimText}
			<div class="recording-transcript">
				{#if finalText}<span class="transcript-final">{finalText}</span>{/if}
				{#if interimText}<span class="transcript-interim">{interimText}</span>{/if}
			</div>
		{:else}
			<div class="recording-waveform">
				{#each waveformBars as height}
					<div class="rec-bar" style="height: {height}px"></div>
				{/each}
			</div>
		{/if}
		<span class="recording-duration">{formatDuration(recordingDuration)}</span>
		<button class="rec-action done" onclick={stopRecording} aria-label="Done recording">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
				<polyline points="20 6 9 17 4 12"/>
			</svg>
		</button>
	</div>
{:else}
	<div class="chat-input">
		{#if isTranscribing}
			<div class="transcribing-indicator">
				<div class="transcribing-dots">
					<span class="dot"></span><span class="dot"></span><span class="dot"></span>
				</div>
				<span class="transcribing-label">{pendingTranscript || 'Transcribing...'}</span>
			</div>
		{/if}
		<!-- Textarea — full width, no buttons next to it -->
		<textarea
			bind:this={inputElement}
			bind:value={inputValue}
			placeholder={isTranscribing ? '' : (placeholder ?? 'Ask OSA...')}
			aria-label="Message OSA"
			aria-multiline={!compact}
			aria-busy={isStreaming || isTranscribing}
			rows={1}
			class="chat-textarea"
			class:compact
			class:hidden-for-transcribing={isTranscribing}
			onkeydown={handleKeyDown}
			oninput={handleInput}
			onfocus={onfocus}
		></textarea>

		<!-- Action bar below textarea: [mode] [attach] [model] [runtime] ... [mic] [send] -->
		<div class="chat-action-bar">
			{#if actionBarCenter}
				<div class="chat-actions-center">
					{@render actionBarCenter()}
				</div>
			{/if}
			<div class="chat-actions-left">
				{#if onattach}
					<button
						class="chat-btn attach"
						onclick={onattach}
						disabled={isStreaming}
						aria-label="Attach file"
						title="Attach file"
					>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="btn-icon">
							<path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
						</svg>
					</button>
				{/if}
			</div>
			<div class="chat-actions-right">
				<!-- Voice button -->
				<button
					class="chat-btn voice"
					class:transcribing={isTranscribing}
					onclick={toggleRecording}
					disabled={isStreaming}
					aria-label="Voice input (⌘D)"
					title="Voice input (⌘D)"
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="btn-icon">
						<path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
						<path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
						<line x1="12" y1="19" x2="12" y2="23"/>
						<line x1="8" y1="23" x2="16" y2="23"/>
					</svg>
				</button>

				{#if isStreaming}
					<button class="chat-btn stop" onclick={handleStop} aria-label="Stop streaming">
						<svg viewBox="0 0 24 24" fill="currentColor" class="btn-icon">
							<rect x="7" y="7" width="10" height="10" rx="2" />
						</svg>
					</button>
				{:else}
					<button
						class="chat-btn send"
						class:active={hasInput}
						disabled={!hasInput}
						onclick={handleSend}
						aria-label="Send message"
					>
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="btn-icon">
							<line x1="12" y1="19" x2="12" y2="5" />
							<polyline points="5 12 12 5 19 12" />
						</svg>
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.chat-input {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: 0;
	}

	.chat-action-bar {
		display: flex;
		align-items: center;
		padding: 0;
		gap: 0;
	}

	.chat-actions-left {
		display: flex;
		align-items: center;
		gap: 1px;
	}

	.chat-actions-center {
		display: flex;
		align-items: center;
		gap: 1px;
	}

	.chat-actions-right {
		display: flex;
		align-items: center;
		gap: 1px;
		margin-left: auto;
	}

	.chat-textarea {
		width: 100%;
		resize: none;
		border: none;
		background: transparent;
		outline: none;
		font-size: 14px;
		line-height: 1.5;
		color: #1c1c1e;
		min-width: 0;
		padding: 4px 2px;
	}

	.chat-textarea::placeholder {
		color: #c7c7cc;
	}

	:global(.dark) .chat-textarea {
		color: #f5f5f7;
	}

	:global(.dark) .chat-textarea::placeholder {
		color: #48484a;
	}

	.chat-textarea.compact {
		max-height: 38px;
	}

	.chat-textarea:not(.compact) {
		max-height: 200px;
		overflow-y: auto;
	}

	/* ===== SEND / STOP / VOICE BUTTON ===== */
	.chat-btn {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		border: none;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		transition: all 0.15s ease;
		padding: 0;
	}

	.chat-btn.send {
		background: #d1d1d6;
		color: white;
	}

	.chat-btn.send.active {
		background: #007aff;
		box-shadow: 0 2px 8px rgba(0, 122, 255, 0.3);
	}

	.chat-btn.send.active:hover {
		background: #0066d6;
	}

	.chat-btn.send:disabled {
		cursor: default;
		opacity: 0.5;
	}

	:global(.dark) .chat-btn.send {
		background: #3a3a3c;
	}

	:global(.dark) .chat-btn.send.active {
		background: #0a84ff;
		box-shadow: 0 2px 8px rgba(10, 132, 255, 0.3);
	}

	.chat-btn.stop {
		background: #ff3b30;
		color: white;
		box-shadow: 0 2px 8px rgba(255, 59, 48, 0.3);
	}

	.chat-btn.stop:hover {
		background: #d32f2f;
	}

	:global(.dark) .chat-btn.stop {
		background: #ff453a;
	}

	/* Voice button */
	.chat-btn.voice {
		background: transparent;
		color: #8e8e93;
	}

	.chat-btn.voice:hover {
		color: #1c1c1e;
		background: rgba(0, 0, 0, 0.05);
	}

	.chat-btn.voice:disabled {
		cursor: default;
		opacity: 0.3;
	}

	.chat-btn.voice.transcribing {
		color: #007aff;
		animation: pulse-voice 1.5s infinite;
	}

	:global(.dark) .chat-btn.voice {
		color: #636366;
	}

	:global(.dark) .chat-btn.voice:hover {
		color: #f5f5f7;
		background: rgba(255, 255, 255, 0.06);
	}

	:global(.dark) .chat-btn.voice.transcribing {
		color: #0a84ff;
	}

	/* Attach button */
	.chat-btn.attach {
		background: transparent;
		color: #8e8e93;
	}

	.chat-btn.attach:hover {
		color: #1c1c1e;
		background: rgba(0, 0, 0, 0.05);
	}

	.chat-btn.attach:disabled {
		cursor: default;
		opacity: 0.3;
	}

	:global(.dark) .chat-btn.attach {
		color: #636366;
	}

	:global(.dark) .chat-btn.attach:hover {
		color: #f5f5f7;
		background: rgba(255, 255, 255, 0.06);
	}

	.btn-icon {
		width: 14px;
		height: 14px;
	}

	/* ===== RECORDING WAVEFORM BAR ===== */
	.recording-bar-container {
		display: flex;
		align-items: center;
		gap: 8px;
		background: rgba(0, 0, 0, 0.05);
		border-radius: 999px;
		padding: 6px 8px;
	}

	:global(.dark) .recording-bar-container {
		background: rgba(255, 255, 255, 0.08);
	}

	.recording-waveform {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 3px;
		height: 20px;
	}

	.rec-bar {
		width: 3px;
		background: #636366;
		border-radius: 2px;
		transition: height 0.06s ease-out;
		min-height: 2px;
	}

	:global(.dark) .rec-bar {
		background: #98989d;
	}

	.recording-transcript {
		flex: 1;
		min-width: 0;
		font-size: 12px;
		color: #1c1c1e;
		line-height: 1.4;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.dark) .recording-transcript {
		color: #f5f5f7;
	}

	.transcript-final {
		color: inherit;
	}

	.transcript-interim {
		color: #8e8e93;
		font-style: italic;
	}

	:global(.dark) .transcript-interim {
		color: #636366;
	}

	.recording-duration {
		font-size: 11px;
		font-family: ui-monospace, 'SF Mono', monospace;
		color: #636366;
		min-width: 28px;
		text-align: right;
		flex-shrink: 0;
	}

	:global(.dark) .recording-duration {
		color: #98989d;
	}

	.rec-action {
		width: 26px;
		height: 26px;
		border: none;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		transition: all 0.15s ease;
		padding: 0;
	}

	.rec-action svg {
		width: 12px;
		height: 12px;
	}

	.rec-action.cancel {
		background: #ef4444;
		color: white;
	}

	.rec-action.cancel:hover {
		background: #dc2626;
	}

	.rec-action.done {
		background: #22c55e;
		color: white;
	}

	.rec-action.done:hover {
		background: #16a34a;
	}

	/* ===== TOOLBAR HINTS ===== */
	.toolbar-hints {
		display: flex;
		align-items: center;
		gap: 4px;
		justify-content: center;
		padding: 4px 0 0;
	}

	.hint-key {
		font-size: 10px;
		font-family: ui-monospace, 'SF Mono', monospace;
		color: #aeaeb2;
		background: rgba(0, 0, 0, 0.04);
		padding: 1px 4px;
		border-radius: 3px;
	}

	.hint-label {
		font-size: 10px;
		color: #aeaeb2;
	}

	.hint-dot {
		font-size: 10px;
		color: #d1d1d6;
		margin: 0 2px;
	}

	:global(.dark) .hint-key {
		color: #636366;
		background: rgba(255, 255, 255, 0.06);
	}

	:global(.dark) .hint-label {
		color: #636366;
	}

	:global(.dark) .hint-dot {
		color: #48484a;
	}

	/* ===== TRANSCRIBING INDICATOR ===== */
	.transcribing-indicator {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 2px 0;
		min-width: 0;
	}

	.transcribing-dots {
		display: flex;
		gap: 3px;
		align-items: center;
	}

	.transcribing-dots .dot {
		width: 5px;
		height: 5px;
		border-radius: 50%;
		background: #007aff;
		animation: dot-bounce 1.2s infinite ease-in-out;
	}

	.transcribing-dots .dot:nth-child(2) { animation-delay: 0.15s; }
	.transcribing-dots .dot:nth-child(3) { animation-delay: 0.3s; }

	:global(.dark) .transcribing-dots .dot {
		background: #0a84ff;
	}

	.transcribing-label {
		font-size: 13px;
		color: #8e8e93;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.dark) .transcribing-label {
		color: #636366;
	}

	.chat-textarea.hidden-for-transcribing {
		position: absolute;
		width: 0;
		height: 0;
		overflow: hidden;
		opacity: 0;
		pointer-events: none;
	}

	@keyframes dot-bounce {
		0%, 80%, 100% { transform: scale(0.6); opacity: 0.4; }
		40% { transform: scale(1); opacity: 1; }
	}

	@keyframes pulse-voice {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}
</style>
