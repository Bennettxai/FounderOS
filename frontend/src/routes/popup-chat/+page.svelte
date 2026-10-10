<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { apiClient } from '$lib/api';
	import { useVoiceRecorder } from '$lib/hooks/useVoiceRecorder.svelte';
	import PopupChatHeader from '$lib/components/popup-chat/PopupChatHeader.svelte';
	import PopupChatInput from '$lib/components/popup-chat/PopupChatInput.svelte';
	import PopupChatMessages from '$lib/components/popup-chat/PopupChatMessages.svelte';
	import PopupMeetingBanners from '$lib/components/popup-chat/PopupMeetingBanners.svelte';

	import {
		type LLMModel,
		type PullProgress,
		cloudModelsByProvider,
		loadModels as loadModelsFn,
		isModelPulled as isModelPulledFn,
		isCloudModel as isCloudModelFn,
		selectModel as selectModelFn,
		type ModelStateSetters,
	} from '$lib/components/popup-chat/popupChatModel';
	import {
		startMeetingRecording as startMeetingRecordingFn,
		stopMeetingRecording as stopMeetingRecordingFn,
		captureScreenshot as captureScreenshotFn,
		type ElectronStateSetters,
	} from '$lib/components/popup-chat/popupChatElectron';

	// State
	let inputValue = $state('');
	let messages = $state<Array<{ role: 'user' | 'assistant'; content: string }>>([]);
	let isLoading = $state(false);
	let isMeetingMode = $state(false);
	let inputElement: HTMLTextAreaElement | undefined = $state(undefined);
	let messagesContainer: HTMLDivElement | undefined = $state(undefined);

	// Model selection
	let availableModels = $state<LLMModel[]>([]);
	let localModels = $state<LLMModel[]>([]);
	let selectedModel = $state('');
	let activeProvider = $state('ollama_local');
	let showModelSelector = $state(false);
	let configuredProviders = $state<string[]>([]);

	// Model pulling state
	let isPulling = $state(false);
	let pullingModel = $state('');
	let pullProgress = $state<PullProgress | null>(null);

	// Helper to build model state setters object
	function getModelSetters(): ModelStateSetters {
		return {
			setAvailableModels: (v) => { availableModels = v; },
			setLocalModels: (v) => { localModels = v; },
			setSelectedModel: (v) => { selectedModel = v; },
			setActiveProvider: (v) => { activeProvider = v; },
			setConfiguredProviders: (v) => { configuredProviders = v; },
			setIsPulling: (v) => { isPulling = v; },
			setPullingModel: (v) => { pullingModel = v; },
			setPullProgress: (v) => { pullProgress = v; },
			setShowModelSelector: (v) => { showModelSelector = v; },
			getLocalModels: () => localModels,
			getAvailableModels: () => availableModels,
			getSelectedModel: () => selectedModel,
			getIsPulling: () => isPulling,
		};
	}

	// Voice recording
	const recorder = useVoiceRecorder({
		barCount: 30,
		maxBarHeight: 20,
		onTranscription: async (text) => {
			inputValue = text;
			await handleSubmit();
		},
		onTranscriptionError: (message) => {
			messages = [...messages, { role: 'assistant', content: `Transcription not available: ${message}` }];
		},
	});

	// Clipboard state
	let copiedMessageId = $state<number | null>(null);

	// Screenshot state
	let pendingScreenshot = $state<string | null>(null);
	let isCapturingScreenshot = $state(false);

	// Size state
	type PopupSize = 'small' | 'medium' | 'large' | 'full';
	let currentSize = $state<PopupSize>('small');

	// Upcoming meetings from calendar
	let upcomingMeeting = $state<{ id: string; title: string; start: string } | null>(null);

	onMount(() => {
		inputElement?.focus();

		if (browser && 'electron' in window) {
			const electron = (window as any).electron;

			electron?.on?.('popup:focus-input', () => {
				inputElement?.focus();
			});

			electron?.on?.('popup:start-meeting-recording', () => {
				startMeetingRecording();
			});

			electron?.on?.('popup:size-changed', (size: PopupSize) => {
				currentSize = size;
			});

			electron?.popup?.getSize?.().then((size: PopupSize) => {
				currentSize = size;
			});
		}

		loadModels();
		checkWhisperStatus();
		loadUpcomingMeeting();

		const handleKeyDown = (e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				hidePopup();
			}
		};
		window.addEventListener('keydown', handleKeyDown);

		return () => {
			window.removeEventListener('keydown', handleKeyDown);
		};
	});

	async function loadModels() { await loadModelsFn(getModelSetters()); }
	function isModelPulled(modelId: string) { return isModelPulledFn(localModels, modelId); }
	function isCloudModel(modelId: string) { return isCloudModelFn(availableModels, modelId); }
	function selectModel(model: LLMModel) { selectModelFn(model, getModelSetters()); }

	let whisperAvailable = false;
	async function checkWhisperStatus() {
		try {
			const response = await apiClient.get('/transcribe/status');
			if (response.ok) {
				const data = await response.json();
				whisperAvailable = data.available;
			}
		} catch {
			whisperAvailable = false;
		}
	}

	async function copyToClipboard(text: string, index: number) {
		try {
			await navigator.clipboard.writeText(text);
			copiedMessageId = index;
			setTimeout(() => copiedMessageId = null, 2000);
		} catch {
			// Copy failed silently
		}
	}

	function hidePopup() {
		if (browser && 'electron' in window) {
			(window as any).electron?.popup?.hide?.() || (window as any).electron?.send?.('popup:hide');
		}
	}

	function openMainWindow() {
		if (browser && 'electron' in window) {
			(window as any).electron?.popup?.openMain?.() || (window as any).electron?.send?.('popup:open-main');
		}
	}

	function setSize(size: PopupSize) {
		if (browser && 'electron' in window) {
			(window as any).electron?.popup?.setSize?.(size);
			currentSize = size;
		}
	}

	async function loadUpcomingMeeting() {
		try {
			const response = await apiClient.get('/calendar/upcoming');
			if (response.ok) {
				const data = await response.json();
				if (data.events && data.events.length > 0) {
					const nextMeeting = data.events[0];
					const meetingStart = new Date(nextMeeting.start_time);
					const now = new Date();
					const diffMinutes = (meetingStart.getTime() - now.getTime()) / (1000 * 60);

					if (diffMinutes <= 30 && diffMinutes > -60) {
						upcomingMeeting = {
							id: nextMeeting.id,
							title: nextMeeting.title,
							start: nextMeeting.start_time
						};
					}
				}
			}
		} catch {
			// Calendar loading is non-fatal
		}
	}

	async function handleSubmit() {
		if (!inputValue.trim() || isLoading) return;

		const userMessage = inputValue.trim();
		inputValue = '';

		messages = [...messages, { role: 'user', content: userMessage }];
		scrollToBottom();

		isLoading = true;

		try {
			const response = await apiClient.post('/api/chat/message', {
				message: userMessage,
				model: selectedModel || undefined,
				context: isMeetingMode ? 'meeting_assistant' : 'quick_chat'
			});

			if (response.ok) {
				const data = await response.json();
				messages = [...messages, { role: 'assistant', content: data.response || data.content }];
			} else {
				const error = await response.json();
				messages = [...messages, { role: 'assistant', content: `Error: ${error.error || 'Unknown error'}` }];
			}
		} catch {
			messages = [...messages, { role: 'assistant', content: 'Connection error. Please check your network.' }];
		} finally {
			isLoading = false;
			scrollToBottom();
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSubmit();
		}
	}

	function scrollToBottom() {
		setTimeout(() => {
			messagesContainer?.scrollTo({
				top: messagesContainer.scrollHeight,
				behavior: 'smooth'
			});
		}, 100);
	}

	// Meeting recording state
	let meetingSession: unknown = null;
	let systemMediaRecorder: MediaRecorder | null = null;
	let systemAudioChunks: Blob[] = [];

	function getElectronCtx(): ElectronStateSetters {
		return {
			setIsMeetingMode: (v) => { isMeetingMode = v; },
			setMessages: (updater) => { messages = updater(messages); },
			getMeetingSession: () => meetingSession,
			setMeetingSession: (v) => { meetingSession = v; },
			getSystemMediaRecorder: () => systemMediaRecorder,
			setSystemMediaRecorder: (v) => { systemMediaRecorder = v; },
			getSystemAudioChunks: () => systemAudioChunks,
			setSystemAudioChunks: (v) => { systemAudioChunks = v; },
			getUpcomingMeeting: () => upcomingMeeting,
			setPendingScreenshot: (v) => { pendingScreenshot = v; },
			setIsCapturingScreenshot: (v) => { isCapturingScreenshot = v; },
			getIsCapturingScreenshot: () => isCapturingScreenshot,
			scrollToBottom,
			hidePopup,
		};
	}

	async function startMeetingRecording() { await startMeetingRecordingFn(getElectronCtx()); }
	async function stopMeetingRecording() { await stopMeetingRecordingFn(getElectronCtx()); }
	async function captureScreenshot() { await captureScreenshotFn(getElectronCtx()); }

	function clearScreenshot() {
		pendingScreenshot = null;
	}
</script>

<div class="popup-container">
	<!-- Header -->
	<PopupChatHeader
		{isMeetingMode}
		{currentSize}
		onSetSize={setSize}
		onOpenMainWindow={openMainWindow}
		onHidePopup={hidePopup}
	/>

	<!-- Banners: pull progress, meeting alerts, meeting controls -->
	<PopupMeetingBanners
		{isPulling}
		{pullingModel}
		{pullProgress}
		{isMeetingMode}
		{upcomingMeeting}
		onStartMeetingRecording={startMeetingRecording}
		onStopMeetingRecording={stopMeetingRecording}
	/>

	<!-- Messages -->
	<div class="messages-container" bind:this={messagesContainer}>
		<PopupChatMessages
			{messages}
			{isLoading}
			{copiedMessageId}
			onCopy={copyToClipboard}
		/>
	</div>

	<!-- Screenshot preview -->
	{#if pendingScreenshot}
		<div class="screenshot-preview">
			<img src={pendingScreenshot} alt="Screenshot preview" />
			<button class="remove-screenshot" onclick={clearScreenshot} title="Remove screenshot" aria-label="Remove screenshot">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
				</svg>
			</button>
		</div>
	{/if}

	<!-- Input area -->
	<PopupChatInput
		bind:inputValue
		bind:inputElement
		{isLoading}
		{isCapturingScreenshot}
		{recorder}
		onKeyDown={handleKeyDown}
		onSubmit={handleSubmit}
		onCaptureScreenshot={captureScreenshot}
	/>
</div>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
		background: transparent;
		overflow: hidden;
	}

	.popup-container {
		width: 100%;
		height: 100vh;
		display: flex;
		flex-direction: column;
		background: rgba(255, 255, 255, 0.95);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border-radius: 12px;
		border: 1px solid rgba(0, 0, 0, 0.1);
		overflow: hidden;
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	}

	/* Messages */
	.messages-container {
		flex: 1;
		overflow-y: auto;
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	/* Screenshot preview */
	.screenshot-preview {
		position: relative;
		margin: 0 16px 8px;
		border-radius: 8px;
		overflow: hidden;
		border: 1px solid rgba(0, 0, 0, 0.1);
		max-height: 120px;
	}

	.screenshot-preview img {
		width: 100%;
		height: auto;
		max-height: 120px;
		object-fit: cover;
		display: block;
	}

	.remove-screenshot {
		position: absolute;
		top: 6px;
		right: 6px;
		width: 24px;
		height: 24px;
		border: none;
		background: rgba(0, 0, 0, 0.6);
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		transition: all 0.15s;
	}

	.remove-screenshot:hover {
		background: rgba(0, 0, 0, 0.8);
	}

	.remove-screenshot svg {
		width: 14px;
		height: 14px;
	}

	/* Dark mode */
	:global(.dark) .popup-container {
		background: rgba(28, 28, 30, 0.95);
		border-color: rgba(255, 255, 255, 0.12);
	}

	:global(.dark) .messages-container {
		background: #1c1c1e;
	}
</style>
