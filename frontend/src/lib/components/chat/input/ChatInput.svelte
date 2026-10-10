<script lang="ts">
	import { onMount } from 'svelte';
	import { DropdownMenu } from 'bits-ui';
	import { fly, fade } from 'svelte/transition';
	import DocumentUploadModal from '../modals/DocumentUploadModal.svelte';
	import { getCustomAgents, type CustomAgent } from '$lib/api/ai';

	interface Props {
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		streaming?: boolean;
		contextName?: string;
		modelName?: string;
		onSend?: (message: string) => void;
		onStop?: () => void;
		onAttach?: () => void;
	}

	let {
		value = $bindable(''),
		placeholder = 'Type your message...',
		disabled = false,
		streaming = false,
		contextName = 'Default',
		modelName = 'Local LLM',
		onSend,
		onStop,
		onAttach
	}: Props = $props();

	let textareaRef: HTMLTextAreaElement | undefined = $state(undefined);

	// Agent autocomplete state
	let agents = $state<CustomAgent[]>([]);
	let showAgentDropdown = $state(false);
	let filteredAgents = $state<CustomAgent[]>([]);
	let selectedAgentIndex = $state(0);
	let mentionStart = $state(-1);

	// Load custom agents on mount
	onMount(async () => {
		try {
			const response = await getCustomAgents();
			agents = response.agents;
		} catch (error) {
			console.error('Failed to load custom agents:', error);
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		// Handle agent autocomplete navigation
		if (showAgentDropdown) {
			if (e.key === 'ArrowDown') {
				e.preventDefault();
				selectedAgentIndex = (selectedAgentIndex + 1) % filteredAgents.length;
				return;
			}
			if (e.key === 'ArrowUp') {
				e.preventDefault();
				selectedAgentIndex = selectedAgentIndex === 0 ? filteredAgents.length - 1 : selectedAgentIndex - 1;
				return;
			}
			if (e.key === 'Enter' && filteredAgents.length > 0) {
				e.preventDefault();
				insertAgent(filteredAgents[selectedAgentIndex]);
				return;
			}
			if (e.key === 'Escape') {
				e.preventDefault();
				showAgentDropdown = false;
				return;
			}
		}

		// Normal send
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSend();
		}
	}

	function handleSend() {
		if (!value.trim() || disabled || streaming) return;
		onSend?.(value);
		value = '';
		// Reset textarea height
		if (textareaRef) {
			textareaRef.style.height = 'auto';
		}
	}

	function handleInput() {
		// Auto-resize textarea
		if (textareaRef) {
			textareaRef.style.height = 'auto';
			textareaRef.style.height = Math.min(textareaRef.scrollHeight, 160) + 'px';
		}

		// Check for @ mention trigger
		checkForMention();
	}

	function checkForMention() {
		if (!textareaRef) return;

		const cursorPos = textareaRef.selectionStart;
		const textBeforeCursor = value.substring(0, cursorPos);

		// Find the last @ character before cursor
		const lastAtIndex = textBeforeCursor.lastIndexOf('@');

		if (lastAtIndex === -1) {
			showAgentDropdown = false;
			return;
		}

		// Check if there's a space between @ and cursor (if so, don't show dropdown)
		const textAfterAt = textBeforeCursor.substring(lastAtIndex + 1);
		if (textAfterAt.includes(' ')) {
			showAgentDropdown = false;
			return;
		}

		// Check if @ is at start or preceded by whitespace
		const charBeforeAt = lastAtIndex > 0 ? value[lastAtIndex - 1] : ' ';
		if (charBeforeAt !== ' ' && charBeforeAt !== '\n' && lastAtIndex !== 0) {
			showAgentDropdown = false;
			return;
		}

		// Show dropdown and filter agents
		mentionStart = lastAtIndex;
		const searchTerm = textAfterAt.toLowerCase();
		filteredAgents = agents.filter(agent =>
			agent.name.toLowerCase().includes(searchTerm) ||
			agent.display_name.toLowerCase().includes(searchTerm)
		);

		selectedAgentIndex = 0;
		showAgentDropdown = filteredAgents.length > 0;
	}

	function insertAgent(agent: CustomAgent) {
		if (!textareaRef || mentionStart === -1) return;

		const cursorPos = textareaRef.selectionStart;
		const beforeMention = value.substring(0, mentionStart);
		const afterCursor = value.substring(cursorPos);

		value = beforeMention + '@' + agent.name + ' ' + afterCursor;
		showAgentDropdown = false;

		// Set cursor position after inserted mention
		const newCursorPos = mentionStart + agent.name.length + 2; // +2 for @ and space
		setTimeout(() => {
			if (textareaRef) {
				textareaRef.focus();
				textareaRef.setSelectionRange(newCursorPos, newCursorPos);
			}
		}, 0);
	}

	let showUploadModal = $state(false);

	function handleUploadComplete(doc: any) {
		showUploadModal = false;
		if (import.meta.env.DEV) console.log('Document uploaded:', doc);
	}

	async function handlePasteFromClipboard() {
		try {
			const items = await navigator.clipboard.read();
			for (const item of items) {
				if (item.types.find(t => t.startsWith('image/'))) {
					showUploadModal = true;
					return;
				}
				if (item.types.includes('text/plain')) {
					const blob = await item.getType('text/plain');
					const text = await blob.text();
					value += text;
					handleInput();
					return;
				}
			}
		} catch {
			try {
				const text = await navigator.clipboard.readText();
				if (text) {
					value += text;
					handleInput();
				}
			} catch {
				if (import.meta.env.DEV) console.error('Clipboard access denied');
			}
		}
	}
</script>

<div class="border-t border-gray-100 bg-white">
	<div class="max-w-4xl mx-auto p-4">
		<!-- Input Container -->
		<div class="flex items-end gap-3">
			<!-- Attachment Button -->
			<DropdownMenu.Root>
				<DropdownMenu.Trigger
					class="flex-shrink-0 w-10 h-10 flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-xl transition-colors"
					disabled={streaming}
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
				</DropdownMenu.Trigger>
				<DropdownMenu.Portal>
					<DropdownMenu.Content
						class="z-50 min-w-[180px] bg-white border border-gray-200 rounded-xl shadow-lg p-1 animate-in fade-in-0 zoom-in-95"
						sideOffset={8}
					>
						<DropdownMenu.Item
							class="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
							onclick={() => showUploadModal = true}
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
							</svg>
							Upload file
						</DropdownMenu.Item>
						<DropdownMenu.Item
							class="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
							onclick={() => showUploadModal = true}
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
							</svg>
							Upload image
						</DropdownMenu.Item>
						<DropdownMenu.Item
							class="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
							onclick={handlePasteFromClipboard}
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
							</svg>
							Paste from clipboard
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Portal>
			</DropdownMenu.Root>

			<!-- Textarea -->
			<div class="flex-1 relative">
				<textarea
					bind:this={textareaRef}
					bind:value
					oninput={handleInput}
					onkeydown={handleKeydown}
					{placeholder}
					rows={1}
					disabled={disabled || streaming}
					class="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-[15px] placeholder-gray-400 resize-none focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent transition-all disabled:opacity-50 disabled:cursor-not-allowed"
					style="min-height: 48px; max-height: 160px;"
				></textarea>

				<!-- Agent Autocomplete Dropdown -->
				{#if showAgentDropdown && filteredAgents.length > 0}
					<div
						class="chat-input-agent-dropdown absolute bottom-full left-0 mb-2 w-80 bg-white border border-gray-200 rounded-xl shadow-lg overflow-hidden z-50"
						transition:fly={{ y: 10, duration: 150 }}
					>
						<div class="px-3 py-2 bg-gray-50 border-b border-gray-200">
							<div class="flex items-center gap-2 text-xs text-gray-600">
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
								</svg>
								<span>Select agent ({filteredAgents.length})</span>
							</div>
						</div>
						<div class="max-h-60 overflow-y-auto">
							{#each filteredAgents as agent, index}
								<button
									type="button"
									class="btn-pill btn-pill-ghost btn-pill-sm w-full flex items-start gap-3 text-left"
									class:bg-gray-100={index === selectedAgentIndex}
									onclick={() => insertAgent(agent)}
								>
									{#if agent.avatar}
										<img src={agent.avatar} alt={agent.display_name} class="w-8 h-8 rounded-full flex-shrink-0" />
									{:else}
										<div class="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center text-white text-sm font-medium flex-shrink-0">
											{agent.display_name.charAt(0).toUpperCase()}
										</div>
									{/if}
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<span class="font-medium text-sm text-gray-900">{agent.display_name}</span>
											<span class="text-xs text-gray-400">@{agent.name}</span>
										</div>
										{#if agent.description}
											<p class="text-xs text-gray-500 mt-0.5 line-clamp-2">{agent.description}</p>
										{/if}
									</div>
								</button>
							{/each}
						</div>
						<div class="px-3 py-2 bg-gray-50 border-t border-gray-200 flex items-center gap-4 text-xs text-gray-500">
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 bg-white border border-gray-300 rounded text-gray-600">↑↓</kbd> Navigate
							</span>
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 bg-white border border-gray-300 rounded text-gray-600">Enter</kbd> Select
							</span>
							<span class="flex items-center gap-1">
								<kbd class="px-1 py-0.5 bg-white border border-gray-300 rounded text-gray-600">Esc</kbd> Close
							</span>
						</div>
					</div>
				{/if}
			</div>

			<!-- Voice Button (optional) -->
			<!-- <button
				class="btn-pill btn-pill-ghost flex-shrink-0 w-10 h-10 flex items-center justify-center"
				disabled={streaming}
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
				</svg>
			</button> -->

			<!-- Send / Stop Button -->
			{#if streaming}
				<button
					onclick={onStop}
					class="btn-pill btn-pill-icon btn-pill-danger"
					aria-label="Stop streaming"
				>
					<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
						<rect x="6" y="6" width="12" height="12" rx="2" />
					</svg>
				</button>
			{:else}
				<button
					onclick={handleSend}
					disabled={!value.trim() || disabled}
					class="btn-pill btn-pill-icon btn-pill-primary"
					aria-label="Send message"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
					</svg>
				</button>
			{/if}
		</div>

		<!-- Status Bar -->
		<div class="flex items-center justify-between mt-2 px-1 text-xs text-gray-400">
			<div class="flex items-center gap-3">
				<span class="flex items-center gap-1">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
					</svg>
					{contextName}
				</span>
				<span class="text-gray-300">|</span>
				<span class="flex items-center gap-1">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
					</svg>
					{modelName}
				</span>
			</div>
			<span class="hidden sm:block">
				<kbd class="px-1.5 py-0.5 bg-gray-100 rounded text-gray-500">Enter</kbd> to send
			</span>
		</div>
	</div>
</div>

<DocumentUploadModal 
	open={showUploadModal} 
	onClose={() => showUploadModal = false} 
	onUploadComplete={handleUploadComplete}
/>

<style>
	/* ── Mobile: ChatInput ─────────────────────────────────────────── */

	@media (max-width: 767px) {
		/* Agent dropdown: clamp width to viewport so it never bleeds off-screen */
		:global(.chat-input-agent-dropdown) {
			width: min(320px, calc(100vw - 32px)) !important;
			left: 0 !important;
			right: auto !important;
		}
	}
</style>
