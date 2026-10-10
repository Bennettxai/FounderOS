<!--
	OsaOrb.svelte
	Shared OSA video orb — used on both regular desktop and 3D desktop.

	- Always playing, never static — looks like a continuous animation
	- Idle: full video loops naturally (no fade, no crossfade)
	- Active (listening/speaking): loops last ~0.75s with blue glow
	- Draggable anywhere, position saved to localStorage
	- Live captions: shows user speech (blue) and OSA responses (purple)
-->

<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import { browser } from '$app/environment';
	import { voiceTranscription } from '$lib/services/voiceTranscriptionService';

	interface CaptionMessage {
		id: number;
		sender: 'user' | 'osa';
		text: string;
	}

	interface Props {
		isListening?: boolean;
		isSpeaking?: boolean;
		onToggleListening?: (() => void) | null;
		/** Current transcript text (external mode) */
		transcript?: string;
		/** OSA response text (external mode) */
		osaMessage?: string;
	}

	let {
		isListening: externalListening = false,
		isSpeaking = false,
		onToggleListening = null,
		transcript: externalTranscript = '',
		osaMessage: externalOsaMessage = ''
	}: Props = $props();

	// Internal voice state (used when no external handler is provided)
	let internalListening = $state(false);
	let internalTranscript = $state('');
	let transcriptFadeTimer: ReturnType<typeof setTimeout> | null = null;

	// Use external state if handler provided, otherwise internal
	let isListening = $derived(onToggleListening ? externalListening : internalListening);
	let currentTranscript = $derived(onToggleListening ? externalTranscript : internalTranscript);

	// Conversation captions
	let captions = $state<CaptionMessage[]>([]);
	let captionIdCounter = 0;

	function addCaption(sender: 'user' | 'osa', text: string) {
		if (!text.trim()) return;
		const id = ++captionIdCounter;
		captions = [...captions.slice(-4), { id, sender, text: text.trim() }];
		// Auto-remove after 8s
		setTimeout(() => {
			captions = captions.filter(c => c.id !== id);
		}, 8000);
	}

	// Track external OSA messages
	$effect(() => {
		if (externalOsaMessage) {
			addCaption('osa', externalOsaMessage);
		}
	});

	// Guard against rapid clicks — cooldown-based
	let lastToggleTime = 0;
	const TOGGLE_COOLDOWN = 600; // ms

	// Self-contained voice toggle (when no external handler)
	async function selfToggleListening() {
		const now = Date.now();
		if (now - lastToggleTime < TOGGLE_COOLDOWN) return;
		lastToggleTime = now;

		if (internalListening) {
			voiceTranscription.stop();
			internalListening = false;
			// Fade transcript after 3s
			if (transcriptFadeTimer) clearTimeout(transcriptFadeTimer);
			transcriptFadeTimer = setTimeout(() => { internalTranscript = ''; }, 3000);
		} else {
			internalTranscript = '';
			const started = await voiceTranscription.start(
				(text, isFinal) => {
					internalTranscript = text;
					if (isFinal && text.trim()) {
						addCaption('user', text);
					}
					if (import.meta.env.DEV) console.log(`[OSA Voice] ${isFinal ? 'FINAL' : 'interim'}: ${text}`);
				},
				() => {
					// Voice service stopped unexpectedly (network error, etc.)
					internalListening = false;
				}
			);
			if (started) {
				internalListening = true;
			}
		}
	}

	function handleToggle() {
		if (onToggleListening) {
			onToggleListening();
		} else {
			selfToggleListening();
		}
	}

	let video: HTMLVideoElement | null = $state(null);

	// If the orb video fails to load (e.g. asset missing or protocol can't
	// serve it), degrade gracefully to a static decorative fallback instead
	// of surfacing a broken <video> element. The orb is purely decorative.
	let videoFailed = $state(false);

	function handleVideoError() {
		videoFailed = true;
		stopSegmentLoop();
	}

	// Single video, rAF-driven segment loop for the last 0.75s
	const VIDEO_DURATION = 11.9;
	const LOOP_START = VIDEO_DURATION - 0.75;
	const LOOP_RESET = VIDEO_DURATION - 0.05; // reset 50ms before end
	let rafId: number | null = null;

	// Drag state
	let isDragging = $state(false);
	let dragOffset = $state({ x: 0, y: 0 });
	let hasMoved = $state(false);
	let startPos = $state({ x: 0, y: 0 });
	let position = $state({ x: 0, y: 0 });
	let useCustomPosition = $state(false);

	// Derived: is active
	let isActive = $derived(isListening || isSpeaking);

	// Load saved position
	$effect(() => {
		if (browser) {
			const saved = localStorage.getItem('osaOrbPosition');
			if (saved) {
				try {
					position = JSON.parse(saved);
					useCustomPosition = true;
				} catch {
					// ignore
				}
			}
		}
	});

	function seekVideo(t: number) {
		if (!video) return;
		if ('fastSeek' in video && typeof video.fastSeek === 'function') {
			video.fastSeek(t);
		} else {
			video.currentTime = t;
		}
	}

	// Track whether segment loop should run (non-reactive, for use in rAF)
	let segmentLoopActive = false;

	function startSegmentLoop() {
		stopSegmentLoop();
		segmentLoopActive = true;
		if (video) {
			seekVideo(LOOP_START);
			video.loop = false;
			video.play().catch(() => {});
		}
		function tick() {
			if (!video || !segmentLoopActive) return;
			if (video.currentTime >= LOOP_RESET) {
				seekVideo(LOOP_START);
			}
			rafId = requestAnimationFrame(tick);
		}
		rafId = requestAnimationFrame(tick);
	}

	function stopSegmentLoop() {
		segmentLoopActive = false;
		if (rafId !== null) {
			cancelAnimationFrame(rafId);
			rafId = null;
		}
	}

	// Handle state transitions
	$effect(() => {
		const v = video;
		if (!v) return;

		if (isActive) {
			startSegmentLoop();
		} else {
			stopSegmentLoop();
			v.loop = true;
			if (v.paused) v.play().catch(() => {});
		}

		return () => stopSegmentLoop();
	});

	// Auto-play on load
	function handleCanPlay() {
		if (!video || !video.paused) return;
		video.play().catch(() => {});
	}

	// Drag handlers
	function savePosition() {
		if (browser && useCustomPosition) {
			localStorage.setItem('osaOrbPosition', JSON.stringify(position));
		}
	}

	function handleDragStart(e: MouseEvent) {
		if ((e.target as HTMLElement).closest('.reset-position')) return;
		isDragging = true;
		hasMoved = false;
		startPos = { x: e.clientX, y: e.clientY };
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		dragOffset = { x: e.clientX - rect.left, y: e.clientY - rect.top };
		e.preventDefault();
	}

	function handleDragMove(e: MouseEvent) {
		if (!isDragging) return;
		if (Math.abs(e.clientX - startPos.x) > 5 || Math.abs(e.clientY - startPos.y) > 5) {
			hasMoved = true;
		}
		if (!hasMoved) return;

		position = {
			x: Math.max(0, Math.min(e.clientX - dragOffset.x, window.innerWidth - 170)),
			y: Math.max(0, Math.min(e.clientY - dragOffset.y, window.innerHeight - 170))
		};
		useCustomPosition = true;
	}

	function handleDragEnd() {
		if (!isDragging) return;
		isDragging = false;
		if (hasMoved) {
			savePosition();
		} else {
			handleToggle();
		}
		hasMoved = false;
	}

	function resetPosition() {
		useCustomPosition = false;
		position = { x: 0, y: 0 };
		if (browser) localStorage.removeItem('osaOrbPosition');
	}
</script>

<svelte:window onmousemove={handleDragMove} onmouseup={handleDragEnd} />

<div
	class="osa-orb"
	class:dragging={isDragging}
	class:custom-position={useCustomPosition}
	style={useCustomPosition ? `left: ${position.x}px; top: ${position.y}px;` : ''}
	onmousedown={handleDragStart}
>
	{#if useCustomPosition}
		<button class="reset-position" onclick={resetPosition} title="Reset position">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
				<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
				<path d="M3 3v5h5" />
			</svg>
		</button>
	{/if}

	<button
		class="orb-button"
		class:listening={isListening}
		class:speaking={isSpeaking}
		title={isListening ? 'Stop listening' : 'Tap to speak'}
	>
		<div class="orb-video-wrap">
			{#if videoFailed}
				<!-- Static decorative fallback when the video can't load -->
				<div class="orb-fallback" aria-hidden="true"></div>
			{:else}
				<!-- svelte-ignore a11y-media-has-caption -->
				<video
					bind:this={video}
					src="/OSAFinalNOBG.mp4"
					class="orb-video"
					preload="auto"
					autoplay
					muted
					playsinline
					loop
					oncanplay={handleCanPlay}
					onerror={handleVideoError}
				></video>
			{/if}
		</div>
	</button>

	<!-- Live captions — conversation bubbles near orb -->
	{#if captions.length > 0 || currentTranscript}
		<div class="captions-panel">
			{#each captions as msg (msg.id)}
				<div
					class="caption-bubble"
					class:caption-user={msg.sender === 'user'}
					class:caption-osa={msg.sender === 'osa'}
					transition:fly={{ y: 16, duration: 250 }}
				>
					<span class="caption-label">{msg.sender === 'user' ? 'You' : 'OSA'}</span>
					<span class="caption-text">{msg.text}</span>
				</div>
			{/each}
			{#if currentTranscript}
				<div class="caption-bubble caption-user caption-interim" transition:fade={{ duration: 150 }}>
					<span class="caption-label">You</span>
					<span class="caption-text">{currentTranscript}<span class="typing-dot">...</span></span>
				</div>
			{/if}
		</div>
	{/if}

	<div class="status-label">
		{#if isListening}
			<span class="status-listening">Listening...</span>
		{:else if isSpeaking}
			<span class="status-speaking">OSA speaking...</span>
		{:else}
			<span class="status-idle">OSA</span>
		{/if}
	</div>
</div>

<style>
	.osa-orb {
		position: fixed;
		bottom: 100px;
		right: 30px;
		z-index: 9999;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		cursor: grab;
		pointer-events: auto;
	}

	.osa-orb.custom-position {
		bottom: auto;
		right: auto;
	}

	.osa-orb.dragging {
		cursor: grabbing;
		user-select: none;
	}

	/* Reset button */
	.reset-position {
		position: absolute;
		top: -5px;
		right: -5px;
		width: 22px;
		height: 22px;
		border: none;
		border-radius: 50%;
		background: rgba(0, 0, 0, 0.6);
		color: #a1a1aa;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		opacity: 0;
		transition: all 0.2s ease;
		z-index: 10;
	}

	.osa-orb:hover .reset-position {
		opacity: 0.8;
	}

	.reset-position:hover {
		opacity: 1 !important;
		color: white;
		background: rgba(239, 68, 68, 0.8);
		transform: scale(1.1);
	}

	/* Orb button — explicit resets to prevent global style leaks */
	.orb-button {
		position: relative;
		width: 72px;
		height: 72px;
		padding: 0;
		border: none;
		background: transparent;
		backdrop-filter: none;
		-webkit-backdrop-filter: none;
		border-radius: 0;
		cursor: grab;
		transition: transform 0.2s ease;
		outline: none;
	}

	.osa-orb.dragging .orb-button {
		cursor: grabbing;
		pointer-events: none;
	}

	.orb-button:hover {
		transform: scale(1.06);
	}

	.orb-button:active {
		transform: scale(0.94);
	}

	/* Video wrapper — circle clip hides the black MP4 background */
	.orb-video-wrap {
		position: relative;
		width: 100%;
		height: 100%;
		border-radius: 50%;
		overflow: hidden;
		/* Chromium lets the scaled <video> layer escape the radius clip. */
		clip-path: circle(50%);
		isolation: isolate;
	}

	.orb-video {
		width: 100%;
		height: 100%;
		object-fit: cover;
		transform: scale(1.63);
		filter: drop-shadow(0 8px 20px rgba(0, 0, 0, 0.45))
		        drop-shadow(0 4px 8px rgba(0, 0, 0, 0.3))
		        drop-shadow(0 16px 40px rgba(0, 0, 0, 0.2));
	}

	/* Static decorative fallback shown if the video fails to load */
	.orb-fallback {
		width: 100%;
		height: 100%;
		border-radius: 50%;
		background:
			radial-gradient(circle at 35% 30%, rgba(96, 165, 250, 0.95), rgba(59, 130, 246, 0.85) 45%, rgba(37, 99, 235, 0.9) 100%);
		box-shadow: 0 8px 20px rgba(0, 0, 0, 0.45),
		            0 0 18px rgba(59, 130, 246, 0.45);
	}

	/* === LISTENING STATE: blue glow + breathing on wrap (not video, to preserve scale crop) === */
	.orb-button.listening .orb-video {
		filter: drop-shadow(0 0 14px rgba(59, 130, 246, 0.55))
		        drop-shadow(0 0 28px rgba(59, 130, 246, 0.3))
		        drop-shadow(0 10px 24px rgba(0, 0, 0, 0.35));
	}

	.orb-button.listening .orb-video-wrap {
		animation: listen-breathe 3s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	/* === SPEAKING STATE: brighter blue glow + organic float on wrap === */
	.orb-button.speaking .orb-video {
		filter: drop-shadow(0 0 18px rgba(59, 130, 246, 0.65))
		        drop-shadow(0 0 36px rgba(59, 130, 246, 0.35))
		        drop-shadow(0 12px 28px rgba(0, 0, 0, 0.3))
		        brightness(1.05);
	}

	.orb-button.speaking .orb-video-wrap {
		animation: speak-float 4s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes listen-breathe {
		0%, 100% { transform: scale(1); }
		50% { transform: scale(1.04); }
	}

	@keyframes speak-float {
		0%   { transform: scale(1) translateY(0); }
		25%  { transform: scale(1.03) translateY(-2px); }
		50%  { transform: scale(1.02) translateY(-3px); }
		75%  { transform: scale(1.04) translateY(-1px); }
		100% { transform: scale(1) translateY(0); }
	}

	/* Status label — frosted glass pill with readable dark text */
	.status-label {
		padding: 4px 14px;
		background: rgba(255, 255, 255, 0.55);
		backdrop-filter: blur(20px) saturate(1.4);
		-webkit-backdrop-filter: blur(20px) saturate(1.4);
		border-radius: 20px;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.02em;
		box-shadow: 0 1px 8px rgba(0, 0, 0, 0.1), inset 0 0.5px 0 rgba(255, 255, 255, 0.5);
		white-space: nowrap;
		border: 1px solid rgba(255, 255, 255, 0.4);
	}

	.status-listening {
		color: #1d4ed8;
		animation: text-pulse 2s ease-in-out infinite;
	}

	.status-speaking {
		color: #1d4ed8;
		animation: text-pulse 1.5s ease-in-out infinite;
	}

	.status-idle {
		color: #0f172a;
	}

	@keyframes text-pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.7; }
	}

	/* Dark mode — flip to light text on dark glass */
	:global(.dark) .status-label {
		background: rgba(255, 255, 255, 0.08);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 2px 12px rgba(0, 0, 0, 0.3), inset 0 0.5px 0 rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .status-idle {
		color: rgba(255, 255, 255, 0.7);
	}

	:global(.dark) .status-listening,
	:global(.dark) .status-speaking {
		color: #60a5fa;
	}

	/* ===== CAPTIONS PANEL ===== */
	.captions-panel {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 6px;
		max-width: 280px;
		pointer-events: none;
	}

	.caption-bubble {
		padding: 6px 12px;
		border-radius: 12px;
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		max-width: 100%;
		word-wrap: break-word;
		overflow-wrap: break-word;
	}

	.caption-user {
		background: rgba(59, 130, 246, 0.88);
		box-shadow: 0 2px 10px rgba(59, 130, 246, 0.3);
	}

	.caption-osa {
		background: rgba(168, 85, 247, 0.88);
		box-shadow: 0 2px 10px rgba(168, 85, 247, 0.3);
	}

	.caption-interim {
		opacity: 0.75;
	}

	.caption-label {
		display: block;
		color: rgba(255, 255, 255, 0.7);
		font-size: 10px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin-bottom: 2px;
	}

	.caption-text {
		color: white;
		font-size: 13px;
		line-height: 1.35;
		font-weight: 500;
	}

	.typing-dot {
		animation: blink-dots 1s steps(3, end) infinite;
	}

	@keyframes blink-dots {
		0% { opacity: 0.3; }
		50% { opacity: 1; }
		100% { opacity: 0.3; }
	}

	:global(.dark) .caption-user {
		background: rgba(59, 130, 246, 0.8);
	}

	:global(.dark) .caption-osa {
		background: rgba(168, 85, 247, 0.8);
	}

	/* Responsive */
	@media (max-width: 768px) {
		.osa-orb {
			bottom: 80px;
			right: 20px;
		}

		.orb-button {
			width: 60px;
			height: 60px;
		}
	}
</style>
