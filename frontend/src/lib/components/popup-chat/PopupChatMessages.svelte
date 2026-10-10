<script lang="ts">
	interface Message {
		role: 'user' | 'assistant';
		content: string;
	}

	interface Props {
		messages: Message[];
		isLoading: boolean;
		copiedMessageId: number | null;
		onCopy: (text: string, index: number) => void;
	}

	let { messages, isLoading, copiedMessageId, onCopy }: Props = $props();
</script>

<div class="messages-list">
	{#if messages.length === 0}
		<div class="empty-state">
			<p>Ask me anything or start a meeting recording</p>
			<p class="shortcut-hint">Press <kbd>Cmd+Shift+Space</kbd> to toggle</p>
		</div>
	{:else}
		{#each messages as message, i}
			<div class="message {message.role}">
				<div class="message-content">
					{message.content}
				</div>
				{#if message.role === 'assistant'}
					<button
						class="copy-btn"
						class:copied={copiedMessageId === i}
						onclick={() => onCopy(message.content, i)}
						title="Copy to clipboard"
						aria-label="Copy message to clipboard"
					>
						{#if copiedMessageId === i}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
						{:else}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
								<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
							</svg>
						{/if}
					</button>
				{/if}
			</div>
		{/each}
		{#if isLoading}
			<div class="message assistant">
				<div class="message-content loading">
					<span class="dot"></span>
					<span class="dot"></span>
					<span class="dot"></span>
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.messages-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		color: #666;
		text-align: center;
		padding: 40px 20px;
	}

	.empty-state p {
		margin: 0;
		font-size: 14px;
	}

	.shortcut-hint {
		margin-top: 12px !important;
		font-size: 12px !important;
		color: #999;
	}

	.shortcut-hint kbd {
		background: #f3f4f6;
		padding: 2px 6px;
		border-radius: 4px;
		font-family: inherit;
		font-size: 11px;
	}

	.message {
		display: flex;
		max-width: 90%;
		position: relative;
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

	.message-content.loading {
		display: flex;
		gap: 4px;
		padding: 14px 18px;
	}

	.dot {
		width: 8px;
		height: 8px;
		background: #999;
		border-radius: 50%;
		animation: bounce 1.4s infinite ease-in-out both;
	}

	.dot:nth-child(1) { animation-delay: -0.32s; }
	.dot:nth-child(2) { animation-delay: -0.16s; }

	@keyframes bounce {
		0%, 80%, 100% { transform: scale(0.8); }
		40% { transform: scale(1); }
	}

	.copy-btn {
		position: absolute;
		bottom: -4px;
		right: 8px;
		width: 24px;
		height: 24px;
		border: none;
		background: #f3f4f6;
		border-radius: 4px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #666;
		opacity: 0;
		transition: all 0.15s;
	}

	.message:hover .copy-btn {
		opacity: 1;
	}

	.copy-btn:hover {
		background: #e5e7eb;
		color: #111;
	}

	.copy-btn.copied {
		color: #22c55e;
		opacity: 1;
	}

	.copy-btn svg {
		width: 14px;
		height: 14px;
	}

	:global(.dark) .message.user .message-content {
		background: #0A84FF;
		color: white;
	}

	:global(.dark) .message.assistant .message-content {
		background: #2c2c2e;
		color: #f5f5f7;
	}

	:global(.dark) .empty-state {
		color: #6e6e73;
	}

	:global(.dark) .copy-btn {
		background: #3a3a3c;
		color: #a1a1a6;
	}

	:global(.dark) .copy-btn:hover {
		background: #48484a;
		color: #f5f5f7;
	}
</style>
