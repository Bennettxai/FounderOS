<script lang="ts">
	import FocusAttachments from './FocusAttachments.svelte';
	import FocusCommandDropdown from './FocusCommandDropdown.svelte';
	import FocusToolbar from './FocusToolbar.svelte';

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
		// Input state (bindable)
		inputValue: string;
		attachedFiles: AttachedFile[];
		// Mode badge
		selectedModeName?: string | null;
		onClearMode?: () => void;
		// Submit
		canSubmit: boolean;
		selectedProjectId?: string | null;
		onSubmit: () => void;
		// Commands
		commands?: SlashCommand[];
		activeCommand?: SlashCommand | null;
		onClearCommand?: () => void;
		// Context
		availableContexts?: ContextItem[];
		selectedContextIds?: string[];
		onContextToggle?: (contextId: string) => void;
		// Placeholder
		placeholderText?: string;
		// Input ref for parent focus control
		inputRef?: HTMLTextAreaElement | null;
	}

	let {
		inputValue = $bindable(),
		attachedFiles = $bindable(),
		selectedModeName = null,
		onClearMode,
		canSubmit,
		selectedProjectId = null,
		onSubmit,
		commands = [],
		activeCommand = $bindable(null),
		onClearCommand,
		availableContexts = [],
		selectedContextIds = [],
		onContextToggle,
		placeholderText = 'What would you like to do?',
		inputRef = $bindable(null)
	}: Props = $props();

	let fileInputRef = $state<HTMLInputElement | null>(null);
	let isDragging = $state(false);
	let showCommandSuggestions = $state(false);
	let filteredCommands = $state<SlashCommand[]>([]);
	let commandDropdownIndex = $state(0);

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files) {
			addFiles(Array.from(target.files));
		}
		target.value = '';
	}

	async function addFiles(files: File[]) {
		for (const file of files) {
			const newFile: AttachedFile = {
				id: crypto.randomUUID(),
				name: file.name,
				type: file.type,
				size: file.size
			};
			const reader = new FileReader();
			reader.onload = (e) => {
				newFile.content = e.target?.result as string;
				attachedFiles = [...attachedFiles, newFile];
				if (import.meta.env.DEV) console.log('[FocusInputArea] File added:', file.name, 'size:', file.size, 'type:', file.type);
			};
			reader.onerror = (e) => {
				console.error('[FocusInputArea] Error reading file:', file.name, e);
				attachedFiles = [...attachedFiles, newFile];
			};
			reader.readAsDataURL(file);
		}
	}

	function removeFile(fileId: string) {
		attachedFiles = attachedFiles.filter(f => f.id !== fileId);
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDragging = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		if (e.dataTransfer?.files) {
			addFiles(Array.from(e.dataTransfer.files));
		}
	}

	function selectCommand(cmd: SlashCommand) {
		activeCommand = cmd;
		inputValue = '';
		showCommandSuggestions = false;
		inputRef?.focus();
	}

	function clearActiveCommand() {
		onClearCommand?.();
		inputRef?.focus();
	}

	function handleInput(e: Event) {
		const target = e.target as HTMLTextAreaElement;
		const value = target.value;

		// Auto-resize
		target.style.height = 'auto';
		target.style.height = Math.min(target.scrollHeight, 200) + 'px';

		// Slash command detection
		if (value.startsWith('/') && commands.length > 0) {
			const query = value.slice(1).toLowerCase();
			filteredCommands = query === ''
				? commands.slice(0, 8)
				: commands
					.filter(cmd =>
						cmd.name.toLowerCase().includes(query) ||
						cmd.display_name.toLowerCase().includes(query)
					)
					.slice(0, 8);
			showCommandSuggestions = filteredCommands.length > 0;
			commandDropdownIndex = 0;
		} else {
			showCommandSuggestions = false;
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (showCommandSuggestions && filteredCommands.length > 0) {
			if (e.key === 'ArrowDown') {
				e.preventDefault();
				commandDropdownIndex = (commandDropdownIndex + 1) % filteredCommands.length;
			} else if (e.key === 'ArrowUp') {
				e.preventDefault();
				commandDropdownIndex = commandDropdownIndex <= 0
					? filteredCommands.length - 1
					: commandDropdownIndex - 1;
			} else if (e.key === 'Enter' || e.key === 'Tab') {
				e.preventDefault();
				const cmd = filteredCommands[commandDropdownIndex];
				if (cmd) selectCommand(cmd);
				return;
			} else if (e.key === 'Escape') {
				showCommandSuggestions = false;
				return;
			}
		}

		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			onSubmit();
		}
	}
</script>

<!-- Hidden file input -->
<input
	bind:this={fileInputRef}
	type="file"
	multiple
	accept="image/*,.pdf,.txt,.md,.doc,.docx,.csv,.json"
	onchange={handleFileSelect}
	class="hidden"
/>

<div
	class="input-container"
	class:dragging={isDragging}
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
	role="region"
	aria-label="Message input with file drop"
>
	<!-- Selected mode badge -->
	{#if selectedModeName}
		<div class="selected-mode-badge">
			<span class="mode-name">{selectedModeName}</span>
			<button class="mode-clear" onclick={onClearMode} aria-label="Clear mode">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="12" height="12">
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	<!-- Attached files -->
	<FocusAttachments {attachedFiles} onRemove={removeFile} />

	<!-- Command suggestions + active command chip -->
	<FocusCommandDropdown
		filteredCommands={showCommandSuggestions ? filteredCommands : []}
		{activeCommand}
		{commandDropdownIndex}
		onSelect={selectCommand}
		onHover={(i) => commandDropdownIndex = i}
		onClearActive={clearActiveCommand}
	/>

	<!-- Textarea -->
	<textarea
		bind:this={inputRef}
		bind:value={inputValue}
		class="focus-input"
		placeholder={activeCommand
			? `Describe your ${activeCommand.display_name.toLowerCase()} request...`
			: placeholderText}
		rows="1"
		oninput={handleInput}
		onkeydown={handleKeyDown}
	></textarea>

	<!-- Toolbar: attach + context + submit -->
	<FocusToolbar
		{canSubmit}
		{selectedProjectId}
		{inputValue}
		attachedFilesCount={attachedFiles.length}
		{availableContexts}
		{selectedContextIds}
		onAttach={() => fileInputRef?.click()}
		{onContextToggle}
		{onSubmit}
	/>

	<!-- Drag overlay -->
	{#if isDragging}
		<div class="drag-overlay">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="32" height="32">
				<path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5" />
			</svg>
			<span>Drop files here</span>
		</div>
	{/if}
</div>

<style>
	.hidden {
		display: none;
	}

	.input-container {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 12px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 16px;
		padding: 16px;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
		transition: border-color 0.15s;
	}

	.input-container:focus-within {
		border-color: var(--dt3);
	}

	.input-container.dragging {
		border-color: var(--dt2);
		border-style: dashed;
		background: var(--dbg2);
	}

	/* Mode badge */
	.selected-mode-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		background: var(--dbg2);
		padding: 4px 8px 4px 12px;
		border-radius: 20px;
		align-self: flex-start;
	}

	.mode-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt);
	}

	.mode-clear {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
		border-radius: 50%;
		transition: all 0.15s ease;
	}

	.mode-clear:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	/* Textarea */
	.focus-input {
		width: 100%;
		border: none;
		background: transparent;
		font-size: 15px;
		color: var(--dt);
		resize: none;
		outline: none;
		line-height: 1.5;
		min-height: 24px;
		max-height: 200px;
	}

	.focus-input::placeholder {
		color: var(--dt3);
	}

	/* Drag overlay */
	.drag-overlay {
		position: absolute;
		inset: 0;
		background: var(--dbg2);
		border-radius: 16px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		color: var(--dt2);
		font-size: 14px;
		font-weight: 500;
		pointer-events: none;
	}
</style>
