<script lang="ts">
	import { onMount } from 'svelte';
	import FocusCard from './FocusCard.svelte';
	import FocusModeConfig from './FocusModeConfig.svelte';
	import FocusInputArea from './FocusInputArea.svelte';
	import { FOCUS_MODES, getDefaultOptions, type FocusMode } from '../input/focusModes';

	interface SlashCommand {
		name: string;
		display_name: string;
		description: string;
		icon: string;
		category: string;
	}

	interface AttachedFile {
		id: string;
		name: string;
		type: string;
		size: number;
		content?: string;
	}

	interface ContextItem {
		id: string;
		name: string;
		icon?: string;
	}

	interface Props {
		onSubmit: (message: string, focusMode: string | null, focusOptions: Record<string, string>, files?: AttachedFile[]) => void;
		commands?: SlashCommand[];
		onModeChange?: (isFocusMode: boolean) => void;
		selectedProjectId?: string | null;
		onRequestProjectSelect?: () => void;
		availableContexts?: ContextItem[];
		selectedContextIds?: string[];
		onContextToggle?: (contextId: string) => void;
		initialInput?: string;
	}

	let {
		onSubmit,
		commands = [],
		onModeChange,
		selectedProjectId = null,
		onRequestProjectSelect,
		availableContexts = [],
		selectedContextIds = [],
		onContextToggle,
		initialInput = ''
	}: Props = $props();

	// React to initialInput changes (for voice transcripts)
	$effect(() => {
		if (initialInput && initialInput.trim()) {
			inputValue = initialInput;
			setTimeout(() => inputRef?.focus(), 100);
		}
	});

	let selectedFocusId = $state<string | null>(null);
	let focusOptions = $state<Record<string, string>>({});
	let inputValue = $state('');
	let inputRef = $state<HTMLTextAreaElement | null>(null);
	let attachedFiles = $state<AttachedFile[]>([]);
	let activeCommand = $state<SlashCommand | null>(null);

	let canSubmit = $derived(
		!!selectedProjectId && (inputValue.trim().length > 0 || attachedFiles.length > 0 || activeCommand !== null)
	);

	// Typewriter animation state
	const typewriterTexts = [
		'What would you like to do?',
		'Need help with research?',
		'Ready to build something?',
		'Time to write a document?',
		'Want to analyze some data?'
	];
	let typewriterIndex = $state(0);
	let typewriterCharIndex = $state(0);
	let typewriterText = $state('');
	let isDeleting = $state(false);
	let typewriterPaused = $state(false);

	onMount(() => {
		const typeSpeed = 80;
		const deleteSpeed = 40;
		const pauseTime = 2000;

		const tick = () => {
			const currentText = typewriterTexts[typewriterIndex];

			if (typewriterPaused) return;

			if (!isDeleting) {
				if (typewriterCharIndex < currentText.length) {
					typewriterText = currentText.substring(0, typewriterCharIndex + 1);
					typewriterCharIndex++;
					setTimeout(tick, typeSpeed);
				} else {
					typewriterPaused = true;
					setTimeout(() => {
						typewriterPaused = false;
						isDeleting = true;
						tick();
					}, pauseTime);
				}
			} else {
				if (typewriterCharIndex > 0) {
					typewriterText = currentText.substring(0, typewriterCharIndex - 1);
					typewriterCharIndex--;
					setTimeout(tick, deleteSpeed);
				} else {
					isDeleting = false;
					typewriterIndex = (typewriterIndex + 1) % typewriterTexts.length;
					setTimeout(tick, typeSpeed);
				}
			}
		};

		tick();
	});

	let selectedMode = $derived(
		selectedFocusId ? FOCUS_MODES.find(m => m.id === selectedFocusId) : null
	);

	let placeholderText = $derived(
		selectedMode
			? `Describe what you'd like to ${selectedMode.name.toLowerCase()}...`
			: typewriterText || 'What would you like to do?'
	);

	function handleSelectMode(mode: FocusMode) {
		selectedFocusId = mode.id;
		focusOptions = getDefaultOptions(mode);
	}

	function handleDeselectMode() {
		selectedFocusId = null;
		focusOptions = {};
	}

	function handleOptionChange(optionId: string, value: string) {
		focusOptions = { ...focusOptions, [optionId]: value };
	}

	function handleSubmit() {
		if (!inputValue.trim() && attachedFiles.length === 0 && !activeCommand) return;

		if (!selectedProjectId) {
			onRequestProjectSelect?.();
			return;
		}

		let message = inputValue.trim();
		if (activeCommand) {
			message = `/${activeCommand.name} ${message}`.trim();
		}

		onSubmit(message, selectedFocusId, focusOptions, attachedFiles.length > 0 ? attachedFiles : undefined);
		inputValue = '';
		attachedFiles = [];
		activeCommand = null;
	}

	function handleClearCommand() {
		activeCommand = null;
		inputValue = '';
	}
</script>

<div class="focus-mode-selector">
	<!-- Header -->
	<div class="focus-header">
		<h2 class="focus-title">What's your focus?</h2>
		<p class="focus-subtitle">Choose a mode to help me assist you better</p>
	</div>

	<!-- Focus Cards Row -->
	<div class="focus-cards">
		{#each FOCUS_MODES as mode (mode.id)}
			<FocusCard
				{mode}
				isSelected={selectedFocusId === mode.id}
				onSelect={() => handleSelectMode(mode)}
				onDeselect={handleDeselectMode}
			/>
		{/each}
	</div>

	<!-- Options Panel (shown when mode selected and has options) -->
	{#if selectedMode && selectedMode.options.length > 0}
		<FocusModeConfig
			mode={selectedMode}
			options={focusOptions}
			onOptionChange={handleOptionChange}
		/>
	{/if}

	<!-- Input Area -->
	<div class="input-area">
		<FocusInputArea
			bind:inputValue
			bind:attachedFiles
			bind:activeCommand
			bind:inputRef
			selectedModeName={selectedMode?.name ?? null}
			onClearMode={handleDeselectMode}
			{canSubmit}
			{selectedProjectId}
			onSubmit={handleSubmit}
			{commands}
			onClearCommand={handleClearCommand}
			{availableContexts}
			{selectedContextIds}
			{onContextToggle}
			{placeholderText}
		/>

		<!-- Mode Toggle (below input) -->
		{#if onModeChange}
			<div class="mode-toggle-container">
				<button
					class="mode-toggle-btn"
					onclick={() => onModeChange?.(false)}
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
					</svg>
					<span>Switch to Chat mode</span>
				</button>
			</div>
		{/if}
	</div>
</div>

<style>
	.focus-mode-selector {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 48px 24px;
		max-width: 800px;
		margin: 0 auto;
		gap: 32px;
	}

	.focus-header {
		text-align: center;
	}

	.focus-title {
		font-size: 26px;
		font-weight: 700;
		color: var(--dt);
		margin: 0 0 8px 0;
		letter-spacing: -0.02em;
	}

	.focus-subtitle {
		font-size: 15px;
		color: var(--dt3);
		margin: 0;
	}

	.focus-cards {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		justify-content: center;
		align-items: center;
		width: 100%;
	}

	.input-area {
		width: 100%;
		max-width: 640px;
	}

	/* Mode toggle */
	.mode-toggle-container {
		display: flex;
		justify-content: center;
		margin-top: 16px;
	}

	.mode-toggle-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 7px 16px;
		font-size: 13px;
		color: var(--dt3);
		background: transparent;
		border: 1px solid var(--dbd);
		border-radius: 20px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.mode-toggle-btn:hover {
		color: var(--dt);
		background: var(--dbg2);
		border-color: var(--dt3);
	}

	/* Responsive */
	@media (max-width: 640px) {
		.focus-mode-selector {
			padding: 24px 16px;
			gap: 24px;
		}

		.focus-title {
			font-size: 22px;
		}

		.focus-cards {
			gap: 6px;
		}
	}
</style>
