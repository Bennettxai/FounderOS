<!--
	ResponseStream.svelte
	Displays the OSA conversation history with streaming support.
	ChatGPT-style message bubbles, avatars, timestamps.
	User messages right-aligned, OSA messages left with avatar.
	Inline SkillDecisionCards for EXECUTE mode skill proposals.
-->
<script lang="ts">
	import { renderMarkdown } from '$lib/utils/markdownRenderer';
	import { osaStore } from '$lib/stores/osa';
	import ModeIndicator from './ModeIndicator.svelte';
	import SkillDecisionCard from './SkillDecisionCard.svelte';
	import { Package } from 'lucide-svelte';
	import { goto } from '$app/navigation';

	interface Props {
		maxHeight?: string;
	}

	let { maxHeight = '300px' }: Props = $props();

	let conversation = $derived($osaStore.conversation);
	let isStreaming = $derived($osaStore.isStreaming);
	let streamingContent = $derived($osaStore.streamingContent);
	let activeModel = $derived($osaStore.activeModel);
	let scrollContainer: HTMLDivElement | undefined = $state(undefined);
	let streamStartTime: number | null = $state(null);
	let elapsedSeconds = $state(0);
	let timerInterval: ReturnType<typeof setInterval> | null = $state(null);

	// Live timer during streaming
	$effect(() => {
		if (isStreaming) {
			if (!streamStartTime) streamStartTime = Date.now();
			timerInterval = setInterval(() => {
				elapsedSeconds = Math.round((Date.now() - (streamStartTime ?? Date.now())) / 1000);
			}, 100);
		} else {
			if (timerInterval) clearInterval(timerInterval);
			timerInterval = null;
			streamStartTime = null;
			elapsedSeconds = 0;
		}
		return () => { if (timerInterval) clearInterval(timerInterval); };
	});

	// Auto-scroll to bottom on new content
	$effect(() => {
		// Track dependencies
		conversation.length;
		streamingContent;
		if (scrollContainer) {
			scrollContainer.scrollTop = scrollContainer.scrollHeight;
		}
	});

	function formatDuration(ms: number): string {
		if (ms < 1000) return `${ms}ms`;
		const s = ms / 1000;
		if (s < 60) return `${s.toFixed(1)}s`;
		const m = Math.floor(s / 60);
		const rem = Math.round(s % 60);
		return `${m}m ${rem}s`;
	}

	/** Relative timestamp: "just now", "2m ago", "1h ago" */
	function formatRelativeTime(date: Date): string {
		const now = Date.now();
		const diff = now - date.getTime();
		const seconds = Math.floor(diff / 1000);
		if (seconds < 60) return 'just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}

	async function handleSkillApprove(execution: import('$lib/types/skills').SkillExecution) {
		// TODO: Wire to backend skill execution endpoint
		if (import.meta.env.DEV) console.log('[OSA] Skill approved:', execution.skill.name);
	}

	async function handleSkillReject(execution: import('$lib/types/skills').SkillExecution) {
		// TODO: Wire to backend skill rejection endpoint
		if (import.meta.env.DEV) console.log('[OSA] Skill rejected:', execution.skill.name);
	}
</script>

<div
	bind:this={scrollContainer}
	class="osa-response-stream overflow-y-auto"
	style:max-height={maxHeight}
	role="log"
	aria-label="OSA conversation"
	aria-live="polite"
	aria-relevant="additions"
>
	{#each conversation as message (message.id)}
		{#if message.role === 'user'}
			<!-- User message — right-aligned bubble -->
			<div class="user-message mb-3 flex flex-col items-end" role="listitem">
				<div class="message-bubble message-bubble-user ml-auto text-sm">
					{message.content}
				</div>
				<span class="mt-1 text-[10px] text-gray-400">{formatRelativeTime(message.timestamp)}</span>
			</div>
		{:else}
			<!-- OSA message — left-aligned with avatar -->
			<div class="osa-message mb-3" role="listitem">
				<div class="flex items-start gap-2">
					<!-- Avatar -->
					<div class="mt-1 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600">
						<span class="text-[10px] font-bold text-gray-600 dark:text-gray-300">O</span>
					</div>
					<div class="flex flex-col">
						<ModeIndicator mode={message.mode} confidence={message.confidence} compact />
						<div class="message-bubble message-bubble-assistant mt-1 text-sm">
							{@html renderMarkdown(message.content, { simple: true })}
						</div>
						{#if message.module_id && message.mode === 'BUILD'}
							<button
								class="btn-pill btn-pill-success btn-pill-xs mt-2 inline-flex items-center gap-1.5"
								onclick={() => goto('/settings/modules')}
								aria-label="View created module in settings"
							>
								<Package class="w-3.5 h-3.5" />
								Module created — View in Settings
							</button>
						{/if}
						<div class="mt-1 flex items-center gap-2 text-[10px] text-gray-400">
							<span>{formatRelativeTime(message.timestamp)}</span>
							{#if message.durationMs}
								<span class="opacity-60">·</span>
								<span>{formatDuration(message.durationMs)}</span>
							{/if}
							{#if message.model}
								<span class="opacity-60">·</span>
								<span class="font-mono opacity-70">{message.model}</span>
							{/if}
						</div>
					</div>
				</div>

				<!-- Inline Skill Decision Card (EXECUTE mode) -->
				{#if message.skill_execution}
					<div class="mt-2">
						<SkillDecisionCard
							execution={message.skill_execution}
							onApprove={handleSkillApprove}
							onReject={handleSkillReject}
						/>
					</div>
				{/if}
			</div>
		{/if}
	{/each}

	{#if isStreaming}
		<!-- Streaming bubble -->
		<div class="osa-message streaming mb-3" aria-live="polite" aria-atomic="false">
			<div class="flex items-start gap-2">
				<!-- Avatar -->
				<div class="mt-1 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600">
					<span class="text-[10px] font-bold text-gray-600 dark:text-gray-300">O</span>
				</div>
				<div class="flex flex-col">
					{#if activeModel}
						<div class="mb-1 flex items-center gap-1 text-[10px] text-gray-400">
							<span class="font-mono opacity-70">{activeModel}</span>
							{#if elapsedSeconds > 0}
								<span class="opacity-60">·</span>
								<span>{elapsedSeconds}s</span>
							{/if}
						</div>
					{:else if elapsedSeconds > 0}
						<div class="mb-1 text-[10px] text-gray-400">{elapsedSeconds}s</div>
					{/if}
					<div class="message-bubble message-bubble-assistant text-sm">
						{#if streamingContent}
							{@html renderMarkdown(streamingContent, { simple: true })}
						{/if}
						<span class="streaming-cursor inline-block animate-pulse text-gray-400" aria-hidden="true">|</span>
					</div>
				</div>
			</div>
		</div>
	{/if}

	{#if conversation.length === 0 && !isStreaming}
		<div class="flex items-center justify-center py-6 text-xs text-gray-400 dark:text-gray-500">
			Ask OSA anything...
		</div>
	{/if}
</div>

<style>
</style>
