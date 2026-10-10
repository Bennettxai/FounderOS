<script lang="ts">
	import { fly, fade } from 'svelte/transition';

	interface Props {
		content: string;
		timestamp: string;
		userName?: string;
		onEdit?: () => void;
		onCopy?: () => void;
		onDelete?: () => void;
	}

	let { content, timestamp, userName, onEdit, onCopy, onDelete }: Props = $props();

	let userInitial = $derived((userName ?? 'U').charAt(0).toUpperCase());

	let showActions = $state(false);

	function formatTime(dateStr: string) {
		return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function handleCopy() {
		navigator.clipboard.writeText(content);
		onCopy?.();
	}
</script>

<div
	class="flex justify-end group"
	onmouseenter={() => showActions = true}
	onmouseleave={() => showActions = false}
	in:fly={{ y: 20, duration: 300 }}
>
	<div class="flex items-end gap-2 max-w-[75%]">
	<div class="relative flex-1">
		{#if showActions && (onEdit || onCopy || onDelete)}
			<div
				class="absolute -top-8 right-0 flex items-center gap-1 user-action-bar rounded-lg shadow-sm px-1 py-0.5"
				in:fade={{ duration: 150 }}
			>
				{#if onEdit}
					<button
						onclick={onEdit}
						class="btn-pill btn-pill-ghost btn-pill-xs btn-pill-icon"
						title="Edit"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
						</svg>
					</button>
				{/if}
				<button
					onclick={handleCopy}
					class="btn-pill btn-pill-ghost btn-pill-xs btn-pill-icon"
					title="Copy"
				>
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
					</svg>
				</button>
				{#if onDelete}
					<button
						onclick={onDelete}
						class="btn-pill btn-pill-ghost btn-pill-xs btn-pill-icon hover:!bg-red-50 hover:!text-red-600"
						title="Delete"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
					</button>
				{/if}
			</div>
		{/if}

		<div class="user-bubble px-4 py-3 user-bubble-radius">
			<p class="text-[15px] leading-relaxed whitespace-pre-wrap">{content}</p>
		</div>
		<div class="flex items-center justify-end gap-1.5 mt-1 px-1 msg-timestamp">
			<span class="text-xs msg-meta-text">{formatTime(timestamp)}</span>
			<svg class="w-3.5 h-3.5 msg-meta-text" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
			</svg>
		</div>
	</div>
	<div class="user-avatar" aria-hidden="true">{userInitial}</div>
	</div>
</div>

<style>
	.user-action-bar {
		background: var(--dbg);
		border: 1px solid var(--dbd);
	}

	.user-bubble {
		background: var(--dt);
		color: var(--dbg);
	}

	.user-bubble-radius {
		border-radius: 1rem 1rem 0.25rem 1rem;
	}

	.msg-meta-text {
		color: var(--dt3);
	}

	.msg-timestamp {
		opacity: 0;
		transition: opacity 0.15s;
	}

	.group:hover .msg-timestamp {
		opacity: 1;
	}

	.user-avatar {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: var(--dt);
		color: var(--dbg);
		font-size: 11px;
		font-weight: 600;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
</style>
