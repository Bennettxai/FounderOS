<script lang="ts">
	import { fade } from 'svelte/transition';
	import type { AttachedFile } from './spotlightActions.ts';
	import type { SlashCommand, Project, AIModel } from './spotlightSearch.ts';
	import type { useVoiceRecorder } from '$lib/hooks/useVoiceRecorder.svelte';
	import SpotlightVoice from './SpotlightVoice.svelte';
	import SpotlightProjectSelector from './SpotlightProjectSelector.svelte';
	import SpotlightModelSelector from './SpotlightModelSelector.svelte';
	import SpotlightFileAttachments from './SpotlightFileAttachments.svelte';
	import SpotlightCommandsDropdown from './SpotlightCommandsDropdown.svelte';

	type VoiceRecorder = ReturnType<typeof useVoiceRecorder>;

	interface Props {
		inputValue: string;
		onInputChange: (value: string) => void;
		inputElement?: HTMLTextAreaElement;
		onKeyDown: (event: KeyboardEvent) => void;

		// Slash commands
		filteredCommands: SlashCommand[];
		showCommandsDropdown: boolean;
		commandDropdownIndex: number;
		onSelectCommand: (cmd: SlashCommand) => void;
		onCommandHover: (index: number) => void;

		// File attachments
		attachedFiles: AttachedFile[];
		fileInputRef?: HTMLInputElement;
		onFileSelect: (event: Event) => void;
		onRemoveFile: (fileId: string) => void;
		isDragging: boolean;
		onDragEnter: (e: DragEvent) => void;
		onDragLeave: (e: DragEvent) => void;
		onDragOver: (e: DragEvent) => void;
		onDrop: (e: DragEvent) => void;

		// Voice
		recorder: VoiceRecorder;

		// Project selector
		projectsList: Project[];
		selectedProjectId: string | null;
		showProjectDropdown: boolean;
		projectDropdownIndex: number;
		onProjectSelect: (id: string) => void;
		onToggleProjectDropdown: () => void;
		onProjectHover: (index: number) => void;
		onCloseAndNavigate: (appId: string) => void;

		// Model selector
		availableModels: AIModel[];
		selectedModel: string;
		activeProvider: string;
		showModelDropdown: boolean;
		currentModelName: string;
		onModelSelect: (id: string) => void;
		onToggleModelDropdown: () => void;

		// Send
		canSend: boolean;
		onSend: () => void;
	}

	let {
		inputValue,
		onInputChange,
		inputElement = $bindable(),
		onKeyDown,
		filteredCommands,
		showCommandsDropdown,
		commandDropdownIndex,
		onSelectCommand,
		onCommandHover,
		attachedFiles,
		fileInputRef = $bindable(),
		onFileSelect,
		onRemoveFile,
		isDragging,
		onDragEnter,
		onDragLeave,
		onDragOver,
		onDrop,
		recorder,
		projectsList,
		selectedProjectId,
		showProjectDropdown,
		projectDropdownIndex,
		onProjectSelect,
		onToggleProjectDropdown,
		onProjectHover,
		onCloseAndNavigate,
		availableModels,
		selectedModel,
		activeProvider,
		showModelDropdown,
		currentModelName,
		onModelSelect,
		onToggleModelDropdown,
		canSend,
		onSend
	}: Props = $props();

	function handleInput(e: Event) {
		const ta = e.target as HTMLTextAreaElement;
		onInputChange(ta.value);
		// Auto-resize
		ta.style.height = 'auto';
		ta.style.height = Math.min(ta.scrollHeight, 120) + 'px';
	}
</script>

<div
	class="input-card"
	class:dragging={isDragging}
	ondragenter={onDragEnter}
	ondragleave={onDragLeave}
	ondragover={onDragOver}
	ondrop={onDrop}
>
	<!-- Hidden file input -->
	<input
		type="file"
		bind:this={fileInputRef}
		onchange={onFileSelect}
		multiple
		accept="image/*,.pdf,.doc,.docx,.txt,.csv,.json,.md"
		style="display: none;"
	/>

	<!-- Drag overlay -->
	{#if isDragging}
		<div class="drag-overlay" transition:fade={{ duration: 100 }}>
			<div class="drag-content">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
					<polyline points="17 8 12 3 7 8" />
					<line x1="12" y1="3" x2="12" y2="15" />
				</svg>
				<span>Drop files here</span>
			</div>
		</div>
	{/if}

	<!-- File attachments preview -->
	<SpotlightFileAttachments {attachedFiles} {onRemoveFile} />

	<!-- Voice recording or textarea -->
	{#if recorder.isRecording}
		<SpotlightVoice {recorder} />
	{:else}
		<!-- Textarea -->
		<textarea
			bind:this={inputElement}
			value={inputValue}
			placeholder="Ask anything... (type / for commands)"
			rows={1}
			onkeydown={onKeyDown}
			oninput={handleInput}
			aria-label="Chat input"
		></textarea>

		<!-- Commands dropdown -->
		<SpotlightCommandsDropdown
			{filteredCommands}
			show={showCommandsDropdown}
			highlightedIndex={commandDropdownIndex}
			onSelect={onSelectCommand}
			onHover={onCommandHover}
		/>
	{/if}

	<!-- Bottom Controls -->
	<div class="controls-row">
		<div class="left-controls">
			<!-- Attachment -->
			<button
				class="icon-btn"
				title="Attach file"
				onclick={() => fileInputRef?.click()}
				aria-label="Attach file"
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path
						d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
					/>
				</svg>
			</button>

			<!-- Project Selector -->
			<SpotlightProjectSelector
				{projectsList}
				{selectedProjectId}
				{showProjectDropdown}
				{projectDropdownIndex}
				{onProjectSelect}
				{onToggleProjectDropdown}
				{onProjectHover}
				{onCloseAndNavigate}
			/>

			<!-- Model Selector -->
			<SpotlightModelSelector
				{availableModels}
				{selectedModel}
				{activeProvider}
				{showModelDropdown}
				{currentModelName}
				{onModelSelect}
				{onToggleModelDropdown}
			/>
		</div>

		<div class="right-controls">
			<div class="hints">
				<span><kbd>⌘D</kbd> Voice</span>
				<span><kbd>↵</kbd> Send</span>
			</div>

			<!-- Voice Button -->
			<button
				class="icon-btn mic"
				class:recording={recorder.isRecording}
				onclick={recorder.toggleRecording}
				title="Voice input"
				aria-label={recorder.isRecording ? 'Stop recording' : 'Start voice input'}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 1a3 3 0 00-3 3v8a3 3 0 006 0V4a3 3 0 00-3-3z" />
					<path d="M19 10v2a7 7 0 01-14 0v-2" />
					<line x1="12" y1="19" x2="12" y2="23" />
					<line x1="8" y1="23" x2="16" y2="23" />
				</svg>
			</button>

			<!-- Send Button -->
			<button
				class="send-btn"
				onclick={onSend}
				disabled={!canSend}
				title={!selectedProjectId ? 'Select a project first' : 'Send message'}
				aria-label="Send message"
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M5 10l7-7m0 0l7 7m-7-7v18" />
				</svg>
			</button>
		</div>
	</div>
</div>

<style>
	/* Input Card */
	.input-card {
		background: white;
		border-radius: 20px;
		padding: 16px;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
		border: 1px solid rgba(0, 0, 0, 0.08);
		position: relative;
		transition: border-color 0.2s;
	}

	.input-card.dragging {
		border-color: #3b82f6;
		border-style: dashed;
	}

	/* Drag overlay */
	.drag-overlay {
		position: absolute;
		inset: 0;
		background: rgba(59, 130, 246, 0.1);
		border-radius: 20px;
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 10;
	}

	.drag-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		color: #3b82f6;
	}

	.drag-content svg {
		width: 32px;
		height: 32px;
	}

	.drag-content span {
		font-size: 14px;
		font-weight: 500;
	}

	textarea {
		width: 100%;
		border: none;
		font-size: 15px;
		font-family: inherit;
		resize: none;
		outline: none;
		background: transparent;
		color: #111;
		line-height: 1.5;
		min-height: 24px;
		max-height: 120px;
	}

	textarea::placeholder {
		color: #999;
	}

	/* Controls Row */
	.controls-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 12px;
	}

	.left-controls,
	.right-controls {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.icon-btn {
		width: 36px;
		height: 36px;
		border: none;
		background: transparent;
		border-radius: 10px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #888;
		transition: all 0.15s;
	}

	.icon-btn:hover {
		background: #f3f3f3;
		color: #333;
	}

	.icon-btn:active {
		transform: scale(0.95);
	}

	.icon-btn.mic.recording {
		background: #ef4444;
		color: white;
		animation: pulse 1.5s infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4);
		}
		50% {
			box-shadow: 0 0 0 8px rgba(239, 68, 68, 0);
		}
	}

	.icon-btn svg {
		width: 18px;
		height: 18px;
	}

	/* Hints */
	.hints {
		display: flex;
		gap: 12px;
		font-size: 11px;
		color: #999;
	}

	.hints kbd {
		background: #f3f3f3;
		padding: 2px 5px;
		border-radius: 4px;
		font-family: inherit;
		font-size: 10px;
	}

	/* Send Button */
	.send-btn {
		width: 36px;
		height: 36px;
		border: none;
		background: #3b82f6;
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		transition: all 0.15s;
	}

	.send-btn:hover:not(:disabled) {
		background: #2563eb;
		transform: scale(1.05);
	}

	.send-btn:active:not(:disabled) {
		transform: scale(0.95);
	}

	.send-btn:disabled {
		background: #e5e5e5;
		color: #bbb;
		cursor: not-allowed;
	}

	.send-btn svg {
		width: 18px;
		height: 18px;
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .input-card {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .input-card.dragging {
		border-color: #0a84ff;
	}

	:global(.dark) textarea {
		color: #f5f5f7;
	}

	:global(.dark) textarea::placeholder {
		color: #6e6e73;
	}

	:global(.dark) .icon-btn {
		color: #a1a1a6;
	}

	:global(.dark) .icon-btn:hover {
		background: #2c2c2e;
		color: #f5f5f7;
	}

	:global(.dark) .hints {
		color: #6e6e73;
	}

	:global(.dark) .hints kbd {
		background: #2c2c2e;
		color: #a1a1a6;
	}

	:global(.dark) .send-btn {
		background: #0a84ff;
	}

	:global(.dark) .send-btn:hover:not(:disabled) {
		background: #0070e0;
	}

	:global(.dark) .send-btn:disabled {
		background: #3a3a3c;
		color: #6e6e73;
	}
</style>
