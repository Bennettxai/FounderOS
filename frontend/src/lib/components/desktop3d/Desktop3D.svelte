<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Canvas } from '@threlte/core';
	import { desktop3dStore, openWindows, focusedWindow, type ModuleId } from '$lib/stores/desktop3dStore';
	import Desktop3DScene from './Desktop3DScene.svelte';
	import Desktop3DControls from './Desktop3DControls.svelte';
	import Desktop3DDock from './Desktop3DDock.svelte';
	import MenuBar from '$lib/components/desktop/MenuBar.svelte';
	import LayoutManager from './LayoutManager.svelte';
	import LiveCaptions from './LiveCaptions.svelte';
	import VoiceControlPanel from './VoiceControlPanel.svelte';
	import { desktop3dPermissions } from '$lib/services/desktop3dPermissions';
	import { desktop3dLayoutStore } from '$lib/stores/desktop3dLayoutStore';
	import { voiceTranscription } from '$lib/services/voiceTranscriptionService';
	import { voiceCommandParser, type VoiceCommand } from '$lib/services/voiceCommands';
	import { osaVoiceService } from '$lib/services/osaVoice';
	import { getActiveWorkspaceHeaders, getApiBaseUrl } from '$lib/api/base';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { getDesktopModuleIds } from '$lib/config/workspaceModules';

	// Extracted utilities and hooks
	import { isCompleteSentence } from './utils/sentenceDetection';
	import { getQuickAck } from './utils/voiceResponses';
	import { executeCommandAction } from './utils/commandExecutor';
	import { useGestureControl } from './hooks/useGestureControl.svelte';
	import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts.svelte';

	interface Props {
		onExit?: () => void;
	}

	let { onExit }: Props = $props();

	// Voice command state
	let isListening = $state(false);
	let currentTranscript = $state('');
	let lastCommand = $state<VoiceCommand | null>(null);
	let isSpeaking = $state(false);
	let lastRequestTime = 0;
	const REQUEST_COOLDOWN = 1000; // 1 second cooldown between requests

	// Conversation display
	let userMessage = $state('');
	let osaMessage = $state('');

	// Layout manager state
	let showLayoutManager = $state(false);

	// Conversation persistence
	let conversationId = $state<string | null>(null);
	let conversationHistory: Array<{role: string, content: string}> = $state([]);

	// OrbitControls reference (needed for direct camera manipulation)
	let orbitControlsRef: any = $state(null);
	let unsubscribeWorkspace: (() => void) | null = null;
	let moduleSignature = '';

	// ===== HOOKS =====

	const gesture = useGestureControl({ getOrbitControls: () => orbitControlsRef });

	const { handleKeydown } = useKeyboardShortcuts({ onExit });

	// Build the deps object passed to executeCommandAction
	function makeCommandDeps() {
		return {
			desktop3dStore,
			desktop3dLayoutStore,
			osaVoiceService,
			openLayoutManager: () => { showLayoutManager = true; },
			openWindows,
			desktop3dState: desktop3dStore
		};
	}

	// Initialize store and permissions on mount
	onMount(async () => {
		if (import.meta.env.DEV) console.log('[Desktop3D] Initializing 3D Desktop mode...');
		unsubscribeWorkspace = currentWorkspace.subscribe((workspace) => {
			const modules = [...getDesktopModuleIds(workspace?.settings ?? {}), 'settings'];
			const nextSignature = `${workspace?.id ?? 'none'}:${modules.join(',')}`;
			if (nextSignature === moduleSignature) return;
			moduleSignature = nextSignature;
			desktop3dStore.initialize(modules);
		});

		// Wait for OrbitControls to be ready
		setTimeout(() => {
			if (orbitControlsRef) {
				if (import.meta.env.DEV) console.log('[Desktop3D] OrbitControls ready for gesture control');
			} else {
				console.warn('[Desktop3D] OrbitControls not yet available (might take a moment)');
			}
		}, 2000);

		// Setup OSA voice speaking callback
		osaVoiceService.onSpeakingChange((speaking) => {
			isSpeaking = speaking;
		});

		// Check if media permissions are supported
		if (!desktop3dPermissions.isSupported()) {
			console.warn('[Desktop3D] Media permissions not supported in this environment');
		} else {
			desktop3dPermissions.initialize();
			if (import.meta.env.DEV) console.log('[Desktop3D] Permission service initialized');
		}

		// Initialize layout system (async - wait for it)
		await desktop3dLayoutStore.initialize();
		if (import.meta.env.DEV) console.log('[Desktop3D] Layout system initialized');
	});

	// Cleanup on unmount
	onDestroy(() => {
		if (import.meta.env.DEV) console.log('[Desktop3D] Cleaning up 3D Desktop mode...');

		// Cleanup gesture controller
		gesture.disableGesture();

		// Stop voice transcription if active
		if (isListening) {
			voiceTranscription.stop();
		}

		// CRITICAL: Release camera and microphone streams
		desktop3dPermissions.cleanup();
		unsubscribeWorkspace?.();
		if (import.meta.env.DEV) console.log('[Desktop3D] Cleanup complete');
	});

	// Handle window focus from dock
	function handleDockSelect(module: ModuleId) {
		const win = $openWindows.find(w => w.module === module);
		if (win) {
			desktop3dStore.focusWindow(win.id);
		} else {
			desktop3dStore.openWindow(module);
		}
	}

	// Handle view mode toggle
	function handleToggleView() {
		desktop3dStore.toggleViewMode();
	}

	// Handle exit
	function handleExit() {
		onExit?.();
	}

	// ===== VOICE COMMANDS =====

	async function toggleVoiceCommands() {
		if (isListening) {
			voiceTranscription.stop();
			isListening = false;
			currentTranscript = '';

			const micStream = desktop3dPermissions.getMicrophoneStream();
			if (micStream) {
				micStream.getTracks().forEach(track => track.stop());
				if (import.meta.env.DEV) console.log('[Desktop3D] Microphone turned OFF');
			}
		} else {
			try {
				if (import.meta.env.DEV) console.log('[Desktop3D] Acquiring microphone...');

				const stream = await desktop3dPermissions.acquireMicrophoneStream();

				if (!stream) {
					alert('Microphone access denied or unavailable');
					return;
				}

				if (import.meta.env.DEV) console.log('[Desktop3D] Microphone acquired, starting voice system...');

				const started = await voiceTranscription.start(handleTranscript);
				if (started) {
					isListening = true;
					if (import.meta.env.DEV) console.log('[Desktop3D] Voice system started');
				} else {
					alert('Voice system failed to start');
					stream.getTracks().forEach(track => track.stop());
				}
			} catch (err) {
				console.error('[Desktop3D] Voice activation failed:', err);
				alert('Failed to activate voice: ' + (err as Error).message);
			}
		}
	}

	function handleTranscript(text: string, isFinal: boolean) {
		currentTranscript = text;

		// INTERRUPT: Only interrupt OSA if user says a meaningful phrase (3+ words)
		if (isSpeaking && isFinal && text.trim().split(/\s+/).length >= 3) {
			if (import.meta.env.DEV) console.log('[Voice] User interrupted OSA');
			osaVoiceService.stop();
			isSpeaking = false;
		}

		if (isFinal) {
			if (import.meta.env.DEV) {
				console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
				console.log('[Voice] HEARD:', text);
			}

			userMessage = text;

			// Route everything to AI; AI executes commands via [CMD:xxx] markers
			if (import.meta.env.DEV) console.log('[Voice] Routing to AI agent (no command parsing)');
			handleConversation(text);
		}
	}

	function executeVoiceCommand(command: VoiceCommand) {
		if (command.type === 'unknown') {
			if (import.meta.env.DEV) console.log('[Voice] ROUTING TO AI for conversation');
			handleConversation(command.text);
			return;
		}

		const quickAck = getQuickAck(command.type);
		if (import.meta.env.DEV) console.log('[Voice] SPEAKING ACK:', quickAck);
		osaVoiceService.speak(quickAck);

		try {
			if (import.meta.env.DEV) console.log('[Voice] EXECUTING:', command.type);
			executeCommandAction(command, makeCommandDeps());
			if (import.meta.env.DEV) console.log('[Voice] SUCCESS:', command.type);
		} catch (err) {
			console.error('[Voice] FAILED:', command.type, err);
			osaVoiceService.speak("Sorry, that didn't work");
		}
	}

	// ===== CONVERSATION HANDLER (stays here — tightly coupled to component state) =====

	async function handleConversation(text: string) {
		try {
			// Rate limiting
			const now = Date.now();
			if (now - lastRequestTime < REQUEST_COOLDOWN) {
				if (import.meta.env.DEV) console.log('[Voice] Rate limited, please wait');
				return;
			}
			lastRequestTime = now;

			// Build OSA personality prompt with context
			const currentModule = $focusedWindow?.module || 'none';
			const openModules = $openWindows.map(w => w.module).join(', ') || 'none';
			const viewMode = $desktop3dStore.viewMode;

			// VOICE AGENT: Short, conversational, 3D-Desktop-focused
			const systemPrompt = `You are OSA - a fast, casual voice assistant for BusinessOS 3D Desktop.

STATE: ${viewMode} view | Open: ${openModules || 'none'} | Focus: ${currentModule || 'desktop'}

═══════════════════════════════════════════════════════════════
YOUR STYLE (THIS IS HOW YOU TALK):
═══════════════════════════════════════════════════════════════

▸ SHORT: 1-2 sentences max. You're being SPOKEN out loud.
▸ CASUAL: "got it", "on it", "sure" NOT "certainly", "of course", "I shall"
▸ NO MARKDOWN: No **, ##, lists, bullets. Just plain talk.
▸ ACTIONS: When user wants something done → include [CMD:xxx]

═══════════════════════════════════════════════════════════════
COMMANDS YOU CAN EXECUTE:
═══════════════════════════════════════════════════════════════

MODULES: open/close {dashboard, chat, tasks, projects, team, clients, terminal, settings, help, agents, crm, tables, pages, nodes, daily, knowledge}
WINDOWS: next window, previous window, close all windows, minimize, maximize, unfocus
CAMERA: zoom in/out, reset zoom, rotate left/right, stop rotation, rotate faster/slower
VIEW: switch to grid/orb, expand/contract orb, increase/decrease grid spacing
RESIZE: make wider/narrower/taller/shorter
LAYOUT: enter/exit edit mode, save layout [name], load layout [name]

═══════════════════════════════════════════════════════════════
PERFECT RESPONSES (COPY THIS STYLE EXACTLY):
═══════════════════════════════════════════════════════════════

"Hey OSA"
→ "Hey! What's up?"

"Open the terminal"
→ "On it. [CMD:open terminal]"

"Open chat"
→ "Opening chat. [CMD:open chat]"

"What can we do here?"
→ "I can open modules, control the camera, switch views. What do you need?"

"Go to the next page"
→ "Next one. [CMD:next window]"

"Go to the next window"
→ "Got it. [CMD:next window]"

"Switch to tasks"
→ "Switching to tasks. [CMD:open tasks]"

"Zoom in"
→ "Zooming in. [CMD:zoom in]"

"Close everything"
→ "Closing all. [CMD:close all windows]"

"Make this wider"
→ "Making it wider. [CMD:make wider]"

"Rotate left"
→ "Rotating left. [CMD:rotate left]"

"Switch to grid view"
→ "Switching to grid. [CMD:switch to grid]"

"What's your plan today?"
→ "Whatever you need! Want to open something or change the view?"

"Help"
→ "I can open stuff, zoom around, switch views. What do you need?"

═══════════════════════════════════════════════════════════════
BAD RESPONSES (NEVER DO THIS):
═══════════════════════════════════════════════════════════════

❌ "**Terminal Opened**

You are now in the terminal. What would you like to do in the terminal? Type a command, and I'll resp"
WHY: Way too long, markdown **, no [CMD:xxx], fake simulation

❌ "## Chat Mode Capabilities

In this chat mode, we can:
1. **Discuss topics**: Share thoughts, ask questions..."
WHY: Markdown headers/lists, way too long, no [CMD:xxx]

❌ "Certainly sir, let me pull that up for you right away."
WHY: Too formal ("certainly", "sir"), no [CMD:xxx]

❌ "I can simulate a terminal for you. What would you like to do?"
WHY: Never simulate - use [CMD:xxx] to open REAL things

❌ "Terminal opened! You now have access to a terminal window where you can execute commands."
WHY: Too long, no [CMD:xxx], describing instead of doing

═══════════════════════════════════════════════════════════════
CRITICAL RULES:
═══════════════════════════════════════════════════════════════

1. MAX 1-2 sentences per response
2. When user wants action → include [CMD:xxx]
3. Be casual: "got it", "on it", "sure" NOT "certainly", "of course"
4. NO markdown (**, ##, lists, bullets) EVER
5. Don't describe/explain - just execute with [CMD:xxx]
6. Match the PERFECT RESPONSES style EXACTLY

RESPOND NOW:`;

			// Add user message to history
			conversationHistory.push({
				role: 'user',
				content: text
			});

			// Send conversation history for better context
			const response = await fetch(`${getApiBaseUrl()}/chat/message`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', ...getActiveWorkspaceHeaders() },
				credentials: 'include',
				body: JSON.stringify({
					message: text,
					context: 'voice_desktop_3d',
					stream: true,
					conversation_id: conversationId,
					system_prompt: systemPrompt,
					conversation_history: conversationHistory
				})
			});

			if (!response.ok) {
				osaVoiceService.speak("Sorry, I'm having trouble connecting right now");
				return;
			}

			// Stream the SSE response and speak sentence by sentence
			const reader = response.body?.getReader();
			const decoder = new TextDecoder();
			let fullResponse = '';
			let pendingText = '';
			const sentenceEnders = ['.', '!', '?', '\n'];

			if (reader) {
				let sseBuffer = '';

				while (true) {
					const { done, value } = await reader.read();
					if (done) break;

					const chunk = decoder.decode(value, { stream: true });
					sseBuffer += chunk;
					const lines = sseBuffer.split('\n');
					sseBuffer = lines.pop() || '';

					for (const line of lines) {
						if (line.startsWith('data: ')) {
							try {
								const data = JSON.parse(line.slice(6));

								let tokenContent = '';

								if (data.type === 'token' || data.type === 'content') {
									tokenContent = data.content || data.data || data.text || '';
								} else if (data.content) {
									tokenContent = data.content;
								} else if (data.data) {
									tokenContent = typeof data.data === 'string' ? data.data : '';
								} else if (data.text) {
									tokenContent = data.text;
								}

								if (tokenContent) {
									fullResponse += tokenContent;
									pendingText += tokenContent;

									// Smart sentence detection — uses extracted utility
									const trimmed = pendingText.trim();
									const lastChar = trimmed.slice(-1);

									if (sentenceEnders.includes(lastChar)) {
										if (isCompleteSentence(trimmed)) {
											if (import.meta.env.DEV) console.log('[Voice] SPEAKING:', trimmed);
											osaVoiceService.speak(trimmed);
											pendingText = '';
										}
									}
								}
							} catch (err) {
								console.error('[Voice] Parse error:', err, 'Line:', line);
							}
						}
					}
				}

				// Speak any remaining text
				// CRITICAL FIX: ALWAYS speak remaining — fixes truncation of "OK", "Sure", "Done"
				const remaining = pendingText.trim();
				if (remaining) {
					const endsWithPunctuation = /[.!?,;:]$/.test(remaining);
					const completeSentence = endsWithPunctuation ? remaining : remaining + '.';
					if (import.meta.env.DEV) console.log('[Voice] SPEAKING REMAINING:', completeSentence);
					osaVoiceService.speak(completeSentence);
				}
			}

			// Store assistant response and display
			if (fullResponse.trim()) {
				let aiResponse = fullResponse.trim();

				// VALIDATION: If user requested action but AI didn't include command marker
				const userText = text.toLowerCase();
				const actionKeywords = [
					'open', 'close', 'launch', 'start', 'stop', 'zoom', 'rotate',
					'switch', 'change', 'move', 'reset', 'expand', 'contract',
					'minimize', 'maximize', 'save', 'load', 'enter', 'exit'
				];
				const userRequestedAction = actionKeywords.some(keyword => userText.includes(keyword));

				const cmdMatch = aiResponse.match(/\[CMD:([^\]]+)\]/);

				// CRITICAL FIX: If user requested action but AI omitted command, force parse
				if (userRequestedAction && !cmdMatch) {
					console.warn('[Voice] AI failed to include command marker for action request!');
					console.warn('[Voice] User said:', text);
					console.warn('[Voice] AI said:', aiResponse);

					const userCommand = voiceCommandParser.parse(text);
					if (userCommand.type !== 'unknown') {
						if (import.meta.env.DEV) console.log('[Voice] Auto-fixing: Executing user command directly:', userCommand.type);
						executeCommandAction(userCommand, makeCommandDeps());
						aiResponse = 'On it.';
					} else {
						aiResponse = "Sorry, I didn't catch that. Can you try again?";
					}
				} else if (cmdMatch) {
					const commandStr = cmdMatch[1];
					if (import.meta.env.DEV) console.log('[Voice] AI wants to execute command:', commandStr);

					// Remove command marker before speaking
					aiResponse = aiResponse.replace(/\[CMD:[^\]]+\]/g, '').trim();

					// Parse and execute the command
					const parsedCommand = voiceCommandParser.parse(commandStr);
					if (import.meta.env.DEV) console.log('[Voice] Parsed AI command:', parsedCommand);

					if (parsedCommand.type !== 'unknown') {
						if (import.meta.env.DEV) console.log('[Voice] Executing AI command:', parsedCommand.type);
						executeCommandAction(parsedCommand, makeCommandDeps());
					}
				}

				conversationHistory.push({
					role: 'assistant',
					content: aiResponse
				});

				// Keep conversation history to last 10 messages
				if (conversationHistory.length > 10) {
					conversationHistory = conversationHistory.slice(-10);
				}

				osaMessage = aiResponse;

				// Clear OSA message after time proportional to length
				const displayTime = Math.max(20000, aiResponse.length * 50);
				if (import.meta.env.DEV) console.log(`[Voice] Displaying OSA message for ${(displayTime / 1000).toFixed(1)}s`);

				setTimeout(() => {
					osaMessage = '';
				}, displayTime);

				if (import.meta.env.DEV) console.log('[Voice] OSA responded:', aiResponse);
			} else {
				console.error('[Voice] NO RESPONSE - SSE events:', fullResponse.length);
				osaVoiceService.speak("Hmm, give me a second");
			}

			// Get conversation ID from response header for persistence
			const convId = response.headers.get('X-Conversation-Id');
			if (convId) {
				conversationId = convId;
				if (import.meta.env.DEV) console.log('[Voice] Conversation ID:', conversationId);
			}
		} catch (err) {
			console.error('[Voice] Conversation error:', err);
			osaVoiceService.speak("Sorry, I encountered an error");
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="desktop-3d">
	<!-- Top Navigation (same as normal desktop) -->
	<MenuBar />

	<!-- 3D Canvas -->
	<div class="canvas-container">
		<Canvas>
			<Desktop3DScene
				windows={$openWindows}
				viewMode={$desktop3dStore.viewMode}
				focusedWindowId={$desktop3dStore.focusedWindowId}
				autoRotate={$desktop3dStore.autoRotate}
				sphereRadius={$desktop3dStore.sphereRadius}
				cameraDistance={$desktop3dStore.cameraDistance}
				cameraRotationDelta={$desktop3dStore.cameraRotationDelta}
				gestureDragging={$desktop3dStore.gestureDragging}
				bind:orbitControlsRef={orbitControlsRef}
				onWindowClick={(id) => {
					// Always focus the clicked window (smooth transition via springs)
					desktop3dStore.focusWindow(id);
				}}
				onBackgroundClick={() => {
					if ($desktop3dStore.focusedWindowId) {
						desktop3dStore.unfocusWindow();
					}
				}}
				onResize={(w, h) => desktop3dStore.resizeFocusedWindow(w, h)}
				onZoomOut={() => {
					// User zoomed out while in focus mode - exit focus
					if ($desktop3dStore.focusedWindowId) {
						desktop3dStore.unfocusWindow();
					}
				}}
			/>
		</Canvas>
	</div>

	<!-- UI Controls Overlay -->
	<Desktop3DControls
		viewMode={$desktop3dStore.viewMode}
		autoRotate={$desktop3dStore.autoRotate}
		hasFocusedWindow={!!$desktop3dStore.focusedWindowId}
		onToggleView={handleToggleView}
		onToggleAutoRotate={() => desktop3dStore.toggleAutoRotate()}
		onExit={handleExit}
	/>

	<!-- Bottom Dock -->
	<Desktop3DDock
		windows={$openWindows}
		focusedWindowId={$desktop3dStore.focusedWindowId}
		onSelect={handleDockSelect}
	/>

	<!-- Navigation Arrows (only show when focused) -->
	{#if $focusedWindow}
		<button class="nav-arrow nav-arrow-left" onclick={() => desktop3dStore.focusPrevious()}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M15 18l-6-6 6-6" />
			</svg>
		</button>
		<button class="nav-arrow nav-arrow-right" onclick={() => desktop3dStore.focusNext()}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M9 18l6-6-6-6" />
			</svg>
		</button>
	{/if}

	<!-- Permission Prompt - DISABLED: Permissions now requested lazily when user enables voice/gestures -->
	<!-- <PermissionPrompt /> -->

	<!-- Layout Manager Modal -->
	{#if showLayoutManager}
		<LayoutManager
			show={showLayoutManager}
			onClose={() => (showLayoutManager = false)}
		/>
	{/if}

	<!-- Live Captions (voice command feedback) -->
	<LiveCaptions {userMessage} {osaMessage} command={lastCommand} {isListening} {isSpeaking} />

	<!-- Voice Control Panel (enhanced UI) -->
	<VoiceControlPanel {isListening} {isSpeaking} onToggleListening={toggleVoiceCommands} />

	<!-- Hidden video element for gesture camera (MediaPipe) -->
	<!-- svelte-ignore a11y-media-has-caption -->
	<video
		bind:this={gesture.videoElement}
		style="position: absolute; opacity: 0; pointer-events: none; width: 1px; height: 1px;"
		autoplay
		playsinline
		muted
	></video>

	<!-- Gesture Control Toggle Button -->
	<button
		onclick={gesture.toggleGesture}
		class="gesture-toggle-btn"
		class:active={gesture.gestureEnabled}
		class:loading={gesture.gestureLoading}
		disabled={gesture.gestureLoading}
		title={gesture.gestureLoading ? 'Initializing...' : gesture.gestureEnabled ? 'Disable Gesture Control' : 'Enable Gesture Control'}
	>
		{#if gesture.gestureLoading}
			<!-- Loading spinner -->
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" class="spinner">
				<circle cx="12" cy="12" r="10" stroke-width="3" stroke-dasharray="50" stroke-dashoffset="0" />
			</svg>
			<span class="btn-label">Loading...</span>
		{:else}
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 11.5V14m0-2.5v-6a1.5 1.5 0 113 0m-3 6a1.5 1.5 0 00-3 0v2a7.5 7.5 0 0015 0v-5a1.5 1.5 0 00-3 0m-6-3V11m0-5.5v-1a1.5 1.5 0 013 0v1m0 0V11m0-5.5a1.5 1.5 0 013 0v3m0 0V11" />
			</svg>
			{#if !gesture.gestureEnabled}
				<span class="btn-label">Gestures</span>
			{:else}
				<span class="btn-label">ON</span>
			{/if}
		{/if}
	</button>
</div>

<style>
	.desktop-3d {
		position: fixed;
		inset: 0;
		/* Light mode: white top, gray bottom - floating room effect */
		background: linear-gradient(180deg,
			var(--dbg) 0%,
			var(--dbg) 30%,
			var(--dbg2) 70%,
			var(--dbd) 100%
		);
		overflow: hidden;
	}

	/* Dark mode background - darker gradient */
	:global(.dark) .desktop-3d {
		background: linear-gradient(180deg,
			#1a1a1a 0%,
			#141414 30%,
			#0d0d0d 70%,
			#080808 100%
		);
	}

	.canvas-container {
		position: absolute;
		top: 40px; /* Below MenuBar */
		left: 0;
		right: 0;
		bottom: 0;
	}

	/* Navigation Arrows */
	.nav-arrow {
		position: fixed;
		top: 50%;
		transform: translateY(-50%);
		width: 60px;
		height: 60px;
		background: rgba(255, 255, 255, 0.9);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(0, 0, 0, 0.1);
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 200;
		transition: all 0.2s ease;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	}

	.nav-arrow:hover {
		background: rgba(255, 255, 255, 1);
		transform: translateY(-50%) scale(1.1);
		box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
	}

	.nav-arrow svg {
		width: 28px;
		height: 28px;
		stroke: var(--dt);
	}

	.nav-arrow-left {
		left: 30px;
	}

	.nav-arrow-right {
		right: 30px;
	}

	/* ===== DARK MODE STYLES ===== */
	:global(.dark) .nav-arrow {
		background: rgba(44, 44, 46, 0.9);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .nav-arrow:hover {
		background: rgba(58, 58, 60, 0.95);
		box-shadow: 0 6px 20px rgba(0, 0, 0, 0.5);
	}

	:global(.dark) .nav-arrow svg {
		stroke: var(--dt);
	}

	/* Gesture Control Toggle Button */
	.gesture-toggle-btn {
		position: fixed;
		bottom: 30px;
		right: 30px;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 16px;
		background: rgba(15, 15, 20, 0.95);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 12px;
		color: var(--bos-surface-on-color);
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.3s ease;
		z-index: 999;
		backdrop-filter: blur(10px);
	}

	.gesture-toggle-btn:hover {
		background: rgba(25, 25, 30, 0.98);
		border-color: rgba(255, 255, 255, 0.2);
		transform: translateY(-2px);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
	}

	.gesture-toggle-btn.active {
		background: linear-gradient(135deg, var(--bos-status-success) 0%, var(--bos-status-success) 100%);
		color: var(--dt);
		border-color: var(--bos-status-success);
		box-shadow: 0 0 24px rgba(0, 255, 0, 0.5), 0 8px 24px rgba(0, 0, 0, 0.3);
	}

	.gesture-toggle-btn.active:hover {
		background: linear-gradient(135deg, var(--bos-status-success) 0%, var(--bos-status-success) 100%);
		box-shadow: 0 0 32px rgba(0, 255, 0, 0.7), 0 8px 24px rgba(0, 0, 0, 0.3);
	}

	.gesture-toggle-btn svg {
		width: 24px;
		height: 24px;
		stroke-width: 2;
	}

	.gesture-toggle-btn.active svg {
		stroke: var(--dt);
		animation: wave 2s ease-in-out infinite;
	}

	.gesture-toggle-btn.loading {
		background: rgba(30, 30, 35, 0.95);
		cursor: wait;
		opacity: 0.8;
	}

	.gesture-toggle-btn .spinner {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	@keyframes wave {
		0%,
		100% {
			transform: rotate(0deg);
		}
		10% {
			transform: rotate(14deg);
		}
		20% {
			transform: rotate(-8deg);
		}
		30% {
			transform: rotate(14deg);
		}
		40% {
			transform: rotate(-4deg);
		}
		50% {
			transform: rotate(10deg);
		}
		60% {
			transform: rotate(0deg);
		}
	}

	.btn-label {
		white-space: nowrap;
	}
</style>
