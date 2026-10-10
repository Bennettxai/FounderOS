<!--
  OnboardingConversation.svelte
  Handles the intro and conversation phases of the onboarding flow.
  Shows the PurpleOrb, agent message, chip selector or chat input, and go-back.
-->
<script lang="ts">
	import PurpleOrb from './PurpleOrb.svelte';
	import SequentialTypewriter from './SequentialTypewriter.svelte';
	import type { QuestionType, ChipOption } from './onboardingTypes.ts';

	function registerInputRef(el: HTMLInputElement) {
		onInputRef(el);
		return {
			destroy() { onInputRef(null); }
		};
	}

	interface ConversationConfig {
		inputType: 'chat' | 'chips';
		chips?: ChipOption[];
	}

	interface Props {
		phase: 'intro' | 'conversation';
		currentQuestion: QuestionType;
		currentQuestionConfig: ConversationConfig;
		currentAgentMessage: string;
		isAgentTyping: boolean;
		isResuming: boolean;
		resumeMessage: string;
		canGoBack: boolean;
		inputValue: string;
		isRecording: boolean;
		introLines: string[];
		onIntroComplete: () => void;
		onChipSelect: (chipId: string) => void;
		onFormSubmit: (e: Event) => void;
		onGoBack: () => void;
		onSkip: () => void;
		onContinueResume: () => void;
		onToggleVoice: () => void;
		onInputRef: (el: HTMLInputElement | null) => void;
	}

	let {
		phase,
		currentQuestion,
		currentQuestionConfig,
		currentAgentMessage,
		isAgentTyping,
		isResuming,
		resumeMessage,
		canGoBack,
		inputValue = $bindable(),
		isRecording,
		introLines,
		onIntroComplete,
		onChipSelect,
		onFormSubmit,
		onGoBack,
		onSkip,
		onContinueResume,
		onToggleVoice,
		onInputRef
	}: Props = $props();
</script>

<!-- Skip button (conversation phase only, not resuming) -->
{#if phase === 'conversation' && !isResuming}
	<button class="btn-pill btn-pill-ghost skip-btn" onclick={onSkip} title="Skip questions and go to integrations">
		Skip for now
	</button>
{/if}

<div class="centered-layout">
	<div class="orb-section">
		<PurpleOrb size="lg" isThinking={isAgentTyping} />
	</div>

	<div class="text-section">
		{#if phase === 'intro'}
			<SequentialTypewriter
				lines={introLines}
				speed={30}
				lineDelay={600}
				onComplete={onIntroComplete}
			/>
		{:else}
			{#if isResuming}
				<p class="agent-text resume-message">{resumeMessage}</p>
				<button class="btn-pill btn-pill-ghost continue-resume-btn" onclick={onContinueResume}>
					Continue
				</button>
			{:else if isAgentTyping}
				<div class="agent-text typing" aria-label="Agent is typing">
					<span class="dot"></span>
					<span class="dot"></span>
					<span class="dot"></span>
				</div>
			{:else}
				<p class="agent-text">{currentAgentMessage}</p>
			{/if}
		{/if}
	</div>

	{#if phase === 'conversation' && !isAgentTyping && !isResuming}
		<div class="input-section">
			{#if currentQuestionConfig.inputType === 'chips' && currentQuestionConfig.chips}
				<div class="chips-container">
					{#each currentQuestionConfig.chips as chip (chip.id)}
						<button
							class="btn-pill btn-pill-ghost chip"
							onclick={() => onChipSelect(chip.id)}
						>
							{chip.label}
						</button>
					{/each}
				</div>
			{:else}
				<form class="minimal-input" onsubmit={onFormSubmit}>
					<button
						type="button"
						class="btn-pill btn-pill-ghost voice-btn"
						class:recording={isRecording}
						onclick={onToggleVoice}
						disabled={isAgentTyping}
						aria-label={isRecording ? 'Stop recording' : 'Start voice input'}
					>
						<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/>
							<path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
							<line x1="12" x2="12" y1="19" y2="22"/>
						</svg>
					</button>
					<input
						type="text"
						bind:value={inputValue}
						use:registerInputRef
						placeholder={isRecording ? 'Listening...' : 'Type here...'}
						disabled={isAgentTyping || isRecording}
						autocomplete="off"
					/>
					<button
						type="submit"
						class="btn-pill btn-pill-ghost send-btn"
						disabled={isAgentTyping || !inputValue.trim()}
						aria-label="Send"
					>
						<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="m5 12 7-7 7 7"/>
							<path d="M12 19V5"/>
						</svg>
					</button>
				</form>
			{/if}

			{#if canGoBack}
				<button class="btn-pill btn-pill-ghost go-back-btn" onclick={onGoBack}>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<polyline points="15 18 9 12 15 6"/>
					</svg>
					Go back
				</button>
			{/if}
		</div>
	{/if}
</div>

<style>
	/* Skip button */
	.skip-btn {
		position: fixed;
		top: 20px;
		right: 24px;
		padding: 8px 16px;
		font-size: 14px;
		font-weight: 500;
		color: var(--muted-foreground, #6b7280);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: color 0.2s;
		z-index: 10;
	}

	.skip-btn:hover {
		color: var(--foreground, #1f2937);
	}

	/* Resume/Welcome back styles */
	.resume-message {
		font-size: 16px;
		color: var(--muted-foreground, #6b7280);
		margin-bottom: 8px;
	}

	.continue-resume-btn {
		margin-top: 16px;
		padding: 12px 32px;
		font-size: 15px;
		font-weight: 500;
		color: white;
		background: var(--primary, #6366f1);
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: background 0.2s, transform 0.1s;
	}

	.continue-resume-btn:hover {
		background: var(--primary-dark, #4f46e5);
		transform: translateY(-1px);
	}

	/* Centered layout */
	.centered-layout {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		gap: 32px;
	}

	.orb-section {
		display: flex;
		justify-content: center;
	}

	.text-section {
		text-align: center;
		max-width: 400px;
	}

	.text-section :global(.sequential-typewriter) {
		font-size: 18px;
		line-height: 1.6;
		color: var(--foreground, #1f2937);
	}

	.agent-text {
		font-size: 18px;
		line-height: 1.6;
		color: var(--foreground, #1f2937);
		margin: 0;
	}

	/* Typing dots */
	.agent-text.typing {
		display: flex;
		justify-content: center;
		gap: 6px;
	}

	.dot {
		width: 8px;
		height: 8px;
		background-color: var(--muted-foreground, #9ca3af);
		border-radius: 50%;
		animation: bounce 1.4s infinite ease-in-out both;
	}

	.dot:nth-child(1) { animation-delay: -0.32s; }
	.dot:nth-child(2) { animation-delay: -0.16s; }
	.dot:nth-child(3) { animation-delay: 0s; }

	@keyframes bounce {
		0%, 80%, 100% { transform: scale(0.8); opacity: 0.5; }
		40% { transform: scale(1); opacity: 1; }
	}

	/* Input section */
	.input-section {
		margin-top: 16px;
	}

	/* Go Back button */
	.go-back-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		margin-top: 16px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		color: var(--muted-foreground, #6b7280);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: color 0.2s, transform 0.1s;
	}

	.go-back-btn:hover {
		color: var(--foreground, #1f2937);
		transform: translateX(-2px);
	}

	.go-back-btn svg {
		transition: transform 0.2s;
	}

	.go-back-btn:hover svg {
		transform: translateX(-2px);
	}

	/* Chips */
	.chips-container {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
		justify-content: center;
		max-width: 400px;
	}

	.chip {
		padding: 10px 20px;
		font-size: 14px;
		font-weight: 500;
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 20px;
		background: var(--card, #ffffff);
		color: var(--foreground, #1f2937);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.chip:hover {
		border-color: var(--primary, #6366f1);
		background: var(--primary, #6366f1);
		color: white;
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
	}

	/* Minimal chat input */
	.minimal-input {
		display: flex;
		align-items: center;
		gap: 4px;
		background: var(--card, #f9fafb);
		border: 1px solid var(--border, #e5e7eb);
		border-radius: 24px;
		padding: 4px;
		transition: border-color 0.2s, box-shadow 0.2s;
	}

	.minimal-input:focus-within {
		border-color: var(--primary, #6366f1);
		box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
	}

	.minimal-input input {
		border: none;
		outline: none;
		background: transparent;
		font-size: 15px;
		color: var(--foreground, #1f2937);
		width: 160px;
		padding: 8px 12px;
	}

	.minimal-input input::placeholder {
		color: var(--muted-foreground, #9ca3af);
	}

	/* Voice button */
	.voice-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: none;
		background: transparent;
		color: var(--muted-foreground, #6b7280);
		cursor: pointer;
		transition: all 0.2s;
	}

	.voice-btn:hover:not(:disabled) {
		background: var(--accent, #f3f4f6);
		color: var(--foreground, #1f2937);
	}

	.voice-btn.recording {
		background: #ef4444;
		color: white;
		animation: pulse-recording 1.5s infinite;
	}

	@keyframes pulse-recording {
		0%, 100% { transform: scale(1); }
		50% { transform: scale(1.1); }
	}

	.voice-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* Send button */
	.send-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: none;
		background: var(--primary, #6366f1);
		color: white;
		cursor: pointer;
		transition: opacity 0.2s, transform 0.2s;
	}

	.send-btn:hover:not(:disabled) {
		opacity: 0.9;
		transform: scale(1.05);
	}

	.send-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* Dark mode */
	:global(.dark) .text-section :global(.sequential-typewriter) {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .agent-text {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .minimal-input {
		background: var(--card, #1f2937);
		border-color: var(--border, #374151);
	}

	:global(.dark) .minimal-input input {
		color: var(--foreground, #f9fafb);
	}
</style>
