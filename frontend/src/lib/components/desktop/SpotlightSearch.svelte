<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import { windowStore } from '$lib/stores/windowStore';
	import { useVoiceRecorder } from '$lib/hooks/useVoiceRecorder.svelte';
	import { ImageSearchModal } from '$lib/components/search';

	import SpotlightInput from './spotlight/SpotlightInput.svelte';
	import SpotlightResults from './spotlight/SpotlightResults.svelte';

	import {
		filterSearchItems,
		filterSlashCommands,
		initSlashCommands,
		loadProjects as apiLoadProjects,
		loadModels as apiLoadModels,
		type SearchItem,
		type Project,
		type AIModel
	} from './spotlight/spotlightSearch.ts';

	import {
		openApp,
		openInChat as openInChatAction,
		processFiles,
		sendChatMessage,
		type AttachedFile,
		type ChatMessage
	} from './spotlight/spotlightActions.ts';

	// ---------------------------------------------------------------------------
	// Props
	// ---------------------------------------------------------------------------

	interface Props {
		open: boolean;
		onClose: () => void;
	}

	let { open, onClose }: Props = $props();

	// ---------------------------------------------------------------------------
	// Mode
	// ---------------------------------------------------------------------------

	let mode = $state<'search' | 'chat'>('chat');

	// ---------------------------------------------------------------------------
	// Input / keyboard
	// ---------------------------------------------------------------------------

	let inputValue = $state('');
	let selectedIndex = $state(0);
	let inputElement: HTMLTextAreaElement | undefined = $state(undefined);

	// ---------------------------------------------------------------------------
	// Voice
	// ---------------------------------------------------------------------------

	const recorder = useVoiceRecorder({
		barCount: 30,
		maxBarHeight: 20,
		onTranscription: (text) => {
			inputValue = text;
		}
	});

	// ---------------------------------------------------------------------------
	// Chat
	// ---------------------------------------------------------------------------

	let messages = $state<ChatMessage[]>([]);
	let isTyping = $state(false);
	let conversationId = $state<string | null>(null);

	// ---------------------------------------------------------------------------
	// File attachments
	// ---------------------------------------------------------------------------

	let attachedFiles = $state<AttachedFile[]>([]);
	let fileInputRef: HTMLInputElement | undefined = $state(undefined);
	let isDragging = $state(false);

	// ---------------------------------------------------------------------------
	// Slash commands
	// ---------------------------------------------------------------------------

	let showCommandsDropdown = $state(false);
	let commandDropdownIndex = $state(0);

	// ---------------------------------------------------------------------------
	// Projects
	// ---------------------------------------------------------------------------

	let projectsList = $state<Project[]>([]);
	let selectedProjectId = $state<string | null>(null);
	let showProjectDropdown = $state(false);
	let projectDropdownIndex = $state(0);

	// ---------------------------------------------------------------------------
	// Models
	// ---------------------------------------------------------------------------

	let availableModels = $state<AIModel[]>([]);
	let selectedModel = $state('');
	let activeProvider = $state('ollama_local');
	let showModelDropdown = $state(false);

	// ---------------------------------------------------------------------------
	// Image search
	// ---------------------------------------------------------------------------

	let showImageSearch = $state(false);

	// ---------------------------------------------------------------------------
	// Derived
	// ---------------------------------------------------------------------------

	let filteredItems = $derived(filterSearchItems(inputValue, mode));
	let filteredCommands = $derived(filterSlashCommands(inputValue));

	let canSend = $derived(!!(inputValue.trim() && selectedProjectId && !isTyping));

	let currentModelName = $derived(() => {
		if (!selectedModel) return 'Select Model';
		const model = availableModels.find((m) => m.id === selectedModel);
		return model ? model.name : selectedModel.split(':')[0];
	});

	// ---------------------------------------------------------------------------
	// Effects
	// ---------------------------------------------------------------------------

	$effect(() => {
		if (open) {
			initSlashCommands();
			apiLoadProjects().then((projects) => {
				projectsList = projects;
			});
			apiLoadModels().then((result) => {
				activeProvider = result.activeProvider;
				availableModels = result.models;
				if (!selectedModel) {
					selectedModel =
						result.defaultModel || (result.models.length > 0 ? result.models[0].id : '');
				}
			});
			setTimeout(() => inputElement?.focus(), 50);
		}
		if (!open) {
			inputValue = '';
			selectedIndex = 0;
			messages = [];
			attachedFiles = [];
			conversationId = null;
			recorder.cleanup();
			showProjectDropdown = false;
			showModelDropdown = false;
			showCommandsDropdown = false;
		}
	});

	$effect(() => {
		if (inputValue.startsWith('/') && filteredCommands.length > 0) {
			showCommandsDropdown = true;
			commandDropdownIndex = 0;
		} else {
			showCommandsDropdown = false;
		}
	});

	// ---------------------------------------------------------------------------
	// Keyboard navigation
	// ---------------------------------------------------------------------------

	function handleKeyDown(event: KeyboardEvent) {
		// Commands dropdown navigation
		if (showCommandsDropdown && filteredCommands.length > 0) {
			switch (event.key) {
				case 'ArrowDown':
					event.preventDefault();
					commandDropdownIndex = Math.min(commandDropdownIndex + 1, filteredCommands.length - 1);
					return;
				case 'ArrowUp':
					event.preventDefault();
					commandDropdownIndex = Math.max(commandDropdownIndex - 1, 0);
					return;
				case 'Enter':
				case 'Tab':
					event.preventDefault();
					handleSelectCommand(filteredCommands[commandDropdownIndex]);
					return;
				case 'Escape':
					event.preventDefault();
					showCommandsDropdown = false;
					return;
			}
		}

		// Project dropdown navigation
		if (showProjectDropdown) {
			const total = projectsList.length + 1; // +1 for New Project
			switch (event.key) {
				case 'ArrowDown':
					event.preventDefault();
					projectDropdownIndex = Math.min(projectDropdownIndex + 1, total - 1);
					return;
				case 'ArrowUp':
					event.preventDefault();
					projectDropdownIndex = Math.max(projectDropdownIndex - 1, 0);
					return;
				case 'Enter':
					event.preventDefault();
					if (projectDropdownIndex < projectsList.length) {
						selectedProjectId = projectsList[projectDropdownIndex].id;
						showProjectDropdown = false;
					} else {
						showProjectDropdown = false;
						windowStore.openWindow('projects');
						onClose();
					}
					return;
				case 'Escape':
					event.preventDefault();
					showProjectDropdown = false;
					return;
			}
		}

		// Main keys
		switch (event.key) {
			case 'ArrowDown':
				event.preventDefault();
				if (mode === 'search') {
					selectedIndex = Math.min(selectedIndex + 1, filteredItems.length - 1);
				}
				break;
			case 'ArrowUp':
				event.preventDefault();
				if (mode === 'search') {
					selectedIndex = Math.max(selectedIndex - 1, 0);
				}
				break;
			case 'Enter':
				if (event.shiftKey) return;
				event.preventDefault();
				if (mode === 'chat') {
					if (!selectedProjectId && inputValue.trim()) {
						showProjectDropdown = true;
						showModelDropdown = false;
						projectDropdownIndex = 0;
						return;
					}
					handleSendMessage();
				} else if (filteredItems[selectedIndex]) {
					handleSelectItem(filteredItems[selectedIndex]);
				}
				break;
			case 'Escape':
				onClose();
				break;
			case 'Tab':
				event.preventDefault();
				mode = mode === 'search' ? 'chat' : 'search';
				break;
		}
	}

	// ---------------------------------------------------------------------------
	// Action handlers
	// ---------------------------------------------------------------------------

	function handleSelectItem(item: SearchItem) {
		if (item.type === 'app') {
			openApp(item.id, onClose);
		}
	}

	function handleSelectCommand(cmd: { id: string; name: string }) {
		if (cmd.id === 'image') {
			showImageSearch = true;
			inputValue = '';
			showCommandsDropdown = false;
			return;
		}
		inputValue = cmd.name + ' ';
		showCommandsDropdown = false;
		inputElement?.focus();
	}

	async function handleSendMessage() {
		if (!inputValue.trim() || isTyping || !selectedProjectId) return;

		const userMessage = inputValue.trim();
		messages = [...messages, { role: 'user', content: userMessage }];
		inputValue = '';
		isTyping = true;

		const result = await sendChatMessage(
			{
				message: userMessage,
				model: selectedModel || undefined,
				projectId: selectedProjectId,
				conversationId: conversationId || undefined
			},
			(fullContent) => {
				// Streaming chunk — update or append assistant message
				const last = messages[messages.length - 1];
				if (last?.role === 'user') {
					messages = [...messages, { role: 'assistant', content: fullContent }];
				} else {
					messages = messages.map((m, i) =>
						i === messages.length - 1 ? { ...m, content: fullContent } : m
					);
				}
			}
		);

		if (result.conversationId) {
			conversationId = result.conversationId;
		}
		if (result.error) {
			messages = [...messages, { role: 'assistant', content: result.error }];
		}

		isTyping = false;
	}

	function handleOpenInChat() {
		openInChatAction(conversationId, messages, selectedProjectId, onClose);
	}

	function handleBackdropClick(event: MouseEvent) {
		if ((event.target as HTMLElement).classList.contains('spotlight-backdrop')) {
			onClose();
		}
	}

	function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files) return;
		processFiles(Array.from(input.files), attachedFiles, (updated) => {
			attachedFiles = updated;
		});
		if (fileInputRef) fileInputRef.value = '';
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		if (e.dataTransfer?.files) {
			processFiles(Array.from(e.dataTransfer.files), attachedFiles, (updated) => {
				attachedFiles = updated;
			});
		}
	}

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		isDragging = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		if (
			e.clientX < rect.left ||
			e.clientX > rect.right ||
			e.clientY < rect.top ||
			e.clientY > rect.bottom
		) {
			isDragging = false;
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
	}
</script>

{#if open}
	<div
		class="spotlight-backdrop"
		onclick={handleBackdropClick}
		role="dialog"
		aria-modal="true"
		aria-label="Quick Chat"
		transition:fade={{ duration: 150 }}
	>
		<div class="spotlight-container" transition:scale={{ duration: 150, start: 0.95 }}>
			<!-- Messages Area -->
			{#if messages.length > 0}
				<div class="messages-area">
					<button class="expand-chat-btn" onclick={handleOpenInChat} title="Open in Chat" aria-label="Open in Chat">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6" />
							<polyline points="15 3 21 3 21 9" />
							<line x1="10" y1="14" x2="21" y2="3" />
						</svg>
					</button>
					{#each messages as msg}
						<div class="message {msg.role}">
							<div class="message-content">{msg.content}</div>
						</div>
					{/each}
					{#if isTyping}
						<div class="message assistant">
							<div class="message-content typing">
								<span></span><span></span><span></span>
							</div>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Input Card -->
			<SpotlightInput
				{inputValue}
				onInputChange={(v) => (inputValue = v)}
				bind:inputElement
				onKeyDown={handleKeyDown}
				{filteredCommands}
				{showCommandsDropdown}
				{commandDropdownIndex}
				onSelectCommand={handleSelectCommand}
				onCommandHover={(i) => (commandDropdownIndex = i)}
				{attachedFiles}
				bind:fileInputRef
				onFileSelect={handleFileSelect}
				onRemoveFile={(id) => (attachedFiles = attachedFiles.filter((f) => f.id !== id))}
				{isDragging}
				onDragEnter={handleDragEnter}
				onDragLeave={handleDragLeave}
				onDragOver={handleDragOver}
				onDrop={handleDrop}
				{recorder}
				{projectsList}
				{selectedProjectId}
				{showProjectDropdown}
				{projectDropdownIndex}
				onProjectSelect={(id) => {
					selectedProjectId = id;
					showProjectDropdown = false;
				}}
				onToggleProjectDropdown={() => {
					showProjectDropdown = !showProjectDropdown;
					showModelDropdown = false;
				}}
				onProjectHover={(i) => (projectDropdownIndex = i)}
				onCloseAndNavigate={(appId) => {
					windowStore.openWindow(appId);
					onClose();
				}}
				{availableModels}
				{selectedModel}
				{activeProvider}
				{showModelDropdown}
				currentModelName={currentModelName()}
				onModelSelect={(id) => {
					selectedModel = id;
					showModelDropdown = false;
				}}
				onToggleModelDropdown={() => {
					showModelDropdown = !showModelDropdown;
					showProjectDropdown = false;
				}}
				{canSend}
				onSend={handleSendMessage}
			/>

			<!-- Search Results -->
			{#if mode === 'search'}
				<SpotlightResults
					items={filteredItems}
					{selectedIndex}
					onSelect={handleSelectItem}
					onHoverIndex={(i) => (selectedIndex = i)}
				/>
			{/if}

			<!-- Footer -->
			<div class="footer">
				<button
					class="mode-btn"
					class:active={mode === 'search'}
					onclick={() => (mode = 'search')}
					aria-label="Switch to search mode"
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="11" cy="11" r="8" />
						<path d="m21 21-4.35-4.35" />
					</svg>
					Search
				</button>
				<button
					class="mode-btn"
					class:active={mode === 'chat'}
					onclick={() => (mode = 'chat')}
					aria-label="Switch to chat mode"
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path
							d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
						/>
					</svg>
					Chat
				</button>
				<div class="footer-spacer"></div>
				<span class="footer-hint"><kbd>Tab</kbd> Switch &middot; <kbd>Esc</kbd> Close</span>
			</div>
		</div>
	</div>
{/if}

<!-- Image Search Modal -->
<ImageSearchModal bind:show={showImageSearch} />

<style>
	.spotlight-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.3);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 99999;
	}

	.spotlight-container {
		width: 100%;
		max-width: 600px;
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	/* Messages */
	.messages-area {
		position: relative;
		background: white;
		border-radius: 20px 20px 0 0;
		padding: 16px;
		padding-top: 40px;
		max-height: 300px;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 12px;
		border: 1px solid rgba(0, 0, 0, 0.08);
		border-bottom: none;
		scrollbar-width: none;
		-ms-overflow-style: none;
	}

	.messages-area::-webkit-scrollbar {
		display: none;
	}

	.expand-chat-btn {
		position: absolute;
		top: 8px;
		right: 8px;
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: white;
		border: 1px solid #e5e5e5;
		border-radius: 6px;
		color: #888;
		cursor: pointer;
		transition: all 0.15s;
		z-index: 10;
	}

	.expand-chat-btn:hover {
		background: #f5f5f5;
		color: #333;
		border-color: #ccc;
	}

	.expand-chat-btn svg {
		width: 14px;
		height: 14px;
	}

	.message {
		display: flex;
		max-width: 85%;
	}

	.message.user {
		align-self: flex-end;
	}

	.message.assistant {
		align-self: flex-start;
	}

	.message-content {
		padding: 10px 14px;
		border-radius: 16px;
		font-size: 14px;
		line-height: 1.5;
		white-space: pre-wrap;
	}

	.message.user .message-content {
		background: #111;
		color: white;
		border-bottom-right-radius: 4px;
	}

	.message.assistant .message-content {
		background: #f3f4f6;
		color: #111;
		border-bottom-left-radius: 4px;
	}

	.message-content.typing {
		display: flex;
		gap: 4px;
		padding: 14px 18px;
	}

	.message-content.typing span {
		width: 8px;
		height: 8px;
		background: #999;
		border-radius: 50%;
		animation: bounce 1.4s infinite ease-in-out both;
	}

	.message-content.typing span:nth-child(2) {
		animation-delay: 0.2s;
	}
	.message-content.typing span:nth-child(3) {
		animation-delay: 0.4s;
	}

	@keyframes bounce {
		0%,
		80%,
		100% {
			transform: scale(0.8);
		}
		40% {
			transform: scale(1);
		}
	}

	/* Footer */
	.footer {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-top: 8px;
		padding: 0 4px;
	}

	.mode-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		border: none;
		background: rgba(255, 255, 255, 0.8);
		border-radius: 8px;
		cursor: pointer;
		font-size: 13px;
		color: #666;
		transition: all 0.15s;
	}

	.mode-btn:hover {
		background: white;
		color: #333;
	}

	.mode-btn.active {
		background: white;
		color: #111;
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
	}

	.mode-btn svg {
		width: 14px;
		height: 14px;
	}

	.footer-spacer {
		flex: 1;
	}

	.footer-hint {
		font-size: 11px;
		color: #888;
	}

	.footer-hint kbd {
		background: rgba(0, 0, 0, 0.06);
		padding: 2px 5px;
		border-radius: 4px;
		font-family: inherit;
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .spotlight-backdrop {
		background: rgba(0, 0, 0, 0.5);
	}

	:global(.dark) .messages-area {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.12);
	}

	:global(.dark) .expand-chat-btn {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		color: #a1a1a6;
	}

	:global(.dark) .expand-chat-btn:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	:global(.dark) .message.user .message-content {
		background: #0a84ff;
		color: white;
	}

	:global(.dark) .message.assistant .message-content {
		background: #2c2c2e;
		color: #f5f5f7;
	}

	:global(.dark) .mode-btn {
		background: rgba(44, 44, 46, 0.8);
		color: #a1a1a6;
	}

	:global(.dark) .mode-btn:hover {
		background: #2c2c2e;
		color: #f5f5f7;
	}

	:global(.dark) .mode-btn.active {
		background: #2c2c2e;
		color: #f5f5f7;
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
	}

	:global(.dark) .footer-hint {
		color: #6e6e73;
	}

	:global(.dark) .footer-hint kbd {
		background: rgba(255, 255, 255, 0.1);
		color: #a1a1a6;
	}
</style>
