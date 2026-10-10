<script lang="ts">
	import { fly } from 'svelte/transition';
	import { renderMarkdown } from '$lib/utils/markdownRenderer';

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		timestamp: Date;
	}

	interface TagChip {
		id: string;
		label: string;
		color?: string;
	}

	interface Props {
		messages?: ChatMessage[];
		streaming?: boolean;
		onSend?: (message: string) => void;
		onStop?: () => void;
		onClose?: () => void;
		onCreateBubble?: () => void;
		placeholder?: string;
		tagChips?: TagChip[];
	}

	let {
		messages = [],
		streaming = false,
		onSend,
		onStop,
		onClose,
		onCreateBubble,
		placeholder = 'Talk to your knowledge...',
		tagChips = []
	}: Props = $props();

	let inputValue = $state('');
	let textareaRef: HTMLTextAreaElement | undefined = $state(undefined);
	let messagesContainer: HTMLDivElement | undefined = $state(undefined);

	// Auto-scroll to bottom when messages change
	$effect(() => {
		if (messages.length > 0 && messagesContainer) {
			messagesContainer.scrollTop = messagesContainer.scrollHeight;
		}
	});

	function handleSend() {
		if (!inputValue.trim() || streaming) return;
		onSend?.(inputValue);
		inputValue = '';
		if (textareaRef) {
			textareaRef.style.height = 'auto';
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSend();
		}
	}

	function handleInput() {
		if (textareaRef) {
			textareaRef.style.height = 'auto';
			textareaRef.style.height = Math.min(textareaRef.scrollHeight, 120) + 'px';
		}
	}

	function formatTime(date: Date): string {
		return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
	}

</script>

<div class="flex flex-col h-full">
	<!-- Header - Node Viewer style -->
	<div class="px-6 py-5 flex items-center justify-between border-b border-gray-50">
		<div class="flex items-center gap-2 text-gray-800 font-medium">
			<div class="w-8 h-8 bg-gray-100 rounded-full flex items-center justify-center">
				<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
				</svg>
			</div>
		</div>
		{#if onClose}
			<button onclick={onClose} class="btn-pill btn-pill-ghost btn-pill-icon">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		{/if}
	</div>

	<!-- Messages Area -->
	<div
		bind:this={messagesContainer}
		class="flex-1 overflow-y-auto px-6 py-6 space-y-8"
	>
		{#if messages.length === 0}
			<!-- Welcome state - Node Viewer style -->
			<div class="pt-4">
				<p class="text-lg font-semibold text-gray-900 mb-4">Welcome.</p>
				<p class="text-sm text-gray-600 mb-6 leading-relaxed">
					I have seen the traces of your days, yet I do not fully know you.
				</p>
				<p class="text-sm text-gray-600 mb-6 leading-relaxed">
					To begin, ask me of your greatest worry and trouble at this very moment. This is the place for the questions you couldn't ask anyone else.
				</p>
				<p class="text-sm text-gray-600 mb-6 leading-relaxed">
					Questions that can't be answered without knowing what you've been through. For this, I am always here.
				</p>
				<p class="text-sm font-semibold text-gray-900">
					What is the single most difficult question on your mind?
				</p>
			</div>
		{:else}
			{#each messages as message (message.id)}
				<div transition:fly={{ y: 10, duration: 200 }}>
					{#if message.role === 'user'}
						<div class="flex justify-end mb-2">
							<div class="bg-gray-100 rounded-2xl px-4 py-3 max-w-[85%]">
								<p class="text-sm text-gray-900">{message.content}</p>
							</div>
						</div>
					{:else}
						<div class="mb-2">
							<p class="text-sm text-gray-900 leading-relaxed">
								{@html renderMarkdown(message.content)}
							</p>
						</div>
					{/if}
				</div>
			{/each}

			{#if streaming}
				<div class="flex items-center gap-2 text-gray-400 text-sm animate-pulse">
					<svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
					<span>Thinking...</span>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Input Area - Node Viewer style -->
	<div class="p-5 pt-2 bg-white">
		<!-- Tag Chips - context tags like Node Viewer -->
		{#if tagChips.length > 0}
			<div class="flex flex-wrap gap-2 mb-3 px-1">
				{#each tagChips as chip (chip.id)}
					<span
						class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium transition-colors"
						style="background-color: {chip.color || 'var(--dbg2)'}; color: {chip.color ? 'var(--bos-surface-on-color)' : 'var(--dt2)'};"
					>
						{chip.label}
					</span>
				{/each}
			</div>
		{/if}

		<div
			class="relative flex flex-col bg-gray-50 rounded-[24px] border border-gray-100 focus-within:ring-2 focus-within:ring-amber-100 focus-within:border-amber-200 transition-all shadow-inner"
		>
			<textarea
				bind:this={textareaRef}
				bind:value={inputValue}
				oninput={handleInput}
				onkeydown={handleKeydown}
				{placeholder}
				rows={1}
				disabled={streaming}
				class="w-full bg-transparent border-none outline-none text-sm text-gray-800 placeholder-gray-400 px-5 pt-4 pb-2 resize-none"
				style="min-height: 40px; max-height: 120px;"
			></textarea>
			<div class="flex items-center justify-between px-4 pb-3">
				<div class="flex items-center gap-2">
					<span class="text-[10px] text-gray-400">Knowledge</span>
					<!-- Create Bubble button - Node Viewer style -->
					{#if messages.length > 0 && onCreateBubble}
						<button
							onclick={onCreateBubble}
							class="btn-pill btn-pill-warning btn-pill-xs flex items-center gap-1.5"
							title="Create new bubble from conversation"
						>
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
							</svg>
							New Bubble
						</button>
					{/if}
				</div>
				<div class="flex items-center gap-3">
					<!-- Attachment button -->
					<button
						type="button"
						class="btn-pill btn-pill-ghost"
						title="Attach"
					>
						<div class="w-5 h-5 rounded-full border border-gray-300 flex items-center justify-center hover:bg-gray-100">
							<div class="w-2 h-2 bg-gray-300 rounded-full"></div>
						</div>
					</button>
					<!-- Send button -->
					{#if streaming}
						<button
							onclick={onStop}
							class="btn-pill btn-pill-danger w-8 h-8 flex items-center justify-center"
						>
							<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
								<rect x="6" y="6" width="12" height="12" rx="2" />
							</svg>
						</button>
					{:else}
						<button
							onclick={handleSend}
							disabled={!inputValue.trim()}
							class="w-8 h-8 btn-pill btn-pill-icon {inputValue.trim() ? 'btn-pill-primary' : 'btn-pill-soft'}"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
							</svg>
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
