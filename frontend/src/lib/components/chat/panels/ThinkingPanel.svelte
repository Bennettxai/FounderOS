<script lang="ts">
	/**
	 * ThinkingPanel Component
	 *
	 * Displays extended thinking traces with collapsible UI.
	 * Shows detailed reasoning steps, metadata, and streaming status.
	 */
	import type { ThinkingTrace as APIThinkingTrace, ThinkingStep as APIThinkingStep } from '$lib/api/thinking/types';

	// Local simplified interface for component usage
	export interface ThinkingStep {
		type: 'explore' | 'analyze' | 'synthesize' | 'conclude' | 'verify' | 'fallback' | 'understand' | 'plan' | 'reason' | 'evaluate';
		content: string;
		duration?: number;
	}

	export interface ThinkingTrace {
		id?: string;
		content?: string;
		thinking_content?: string; // API field
		steps?: ThinkingStep[];
		metadata?: {
			tokenCount?: number;
			duration?: number;
			model?: string;
		};
		thinking_tokens?: number; // API field
		model_used?: string; // API field
		duration_ms?: number; // API field
		timestamp?: number;
	}

	interface Props {
		trace: ThinkingTrace | APIThinkingTrace;
		isStreaming?: boolean;
		isExpanded?: boolean;
	}

	let { trace, isStreaming = false, isExpanded = $bindable(false) }: Props = $props();

	// Helper to get content from either format
	const getContent = $derived.by(() => {
		if ('content' in trace && trace.content) return trace.content;
		if ('thinking_content' in trace && trace.thinking_content) return trace.thinking_content;
		return null;
	});

	// Helper to get steps from trace
	const getSteps = $derived.by(() => {
		if ('steps' in trace && trace.steps) return trace.steps;
		return null;
	});

	// Toggle expanded state
	function toggle() {
		isExpanded = !isExpanded;
	}

	// Format duration for display
	function formatDuration(ms: number | undefined): string {
		if (!ms) return '';
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}

	// Get badge color for step type
	function getStepBadgeColor(type: string): string {
		const colors: Record<string, string> = {
			explore: 'bg-blue-100 text-blue-700 border-blue-200',
			analyze: 'bg-purple-100 text-purple-700 border-purple-200',
			synthesize: 'bg-green-100 text-green-700 border-green-200',
			conclude: 'bg-amber-100 text-amber-700 border-amber-200',
			verify: 'bg-teal-100 text-teal-700 border-teal-200',
			fallback: 'bg-gray-100 text-gray-700 border-gray-200'
		};
		return colors[type] || colors.fallback;
	}

	// Estimate token count (rough approximation: 4 chars per token)
	$effect(() => {
		const content = getContent;
		if (!trace.metadata?.tokenCount && content) {
			const estimated = Math.ceil(content.length / 4);
			if (!trace.metadata) {
				(trace as any).metadata = {};
			}
			trace.metadata!.tokenCount = estimated;
		}
	});
</script>

<div class="border border-amber-200 rounded-xl overflow-hidden bg-amber-50/50 shadow-sm">
	<!-- Header Button -->
	<button
		onclick={toggle}
		class="btn-pill btn-pill-ghost btn-pill-sm w-full flex items-center gap-2 text-left"
		aria-expanded={isExpanded}
		aria-label={isExpanded ? 'Collapse thinking' : 'Expand thinking'}
	>
		<!-- Chevron Icon -->
		<svg
			class="w-4 h-4 text-amber-600 transition-transform duration-200 {isExpanded ? 'rotate-90' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
		</svg>

		<!-- Light Bulb Icon -->
		<svg
			class="w-4 h-4 text-amber-600 {isStreaming ? 'animate-pulse' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
			/>
		</svg>

		<!-- Label -->
		<span class="text-sm font-medium text-amber-700">Thinking</span>

		<!-- Streaming Indicator -->
		{#if isStreaming}
			<span class="ml-2 w-2 h-2 bg-amber-500 rounded-full animate-pulse"></span>
		{/if}

		<!-- Metadata -->
		<div class="ml-auto flex items-center gap-3 text-xs text-amber-600">
			{#if trace.metadata?.tokenCount}
				<span>{trace.metadata.tokenCount} tokens</span>
			{/if}
			{#if trace.metadata?.duration && typeof trace.metadata.duration === 'number'}
				<span>{formatDuration(trace.metadata.duration)}</span>
			{/if}
			{#if trace.metadata?.model}
				<span class="font-mono">{trace.metadata.model}</span>
			{/if}
		</div>
	</button>

	<!-- Expanded Content -->
	{#if isExpanded}
		<div class="border-t border-amber-200 bg-amber-50/30">
			<!-- Steps Display -->
			{#if getSteps && getSteps.length > 0}
				<div class="px-4 py-3 space-y-3 max-h-72 overflow-y-auto">
					{#each getSteps as step, index}
						<div class="space-y-1">
							<!-- Step Header -->
							<div class="flex items-center gap-2">
								<span class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded border {getStepBadgeColor(step.type)}">
									{step.type}
								</span>
								{#if step.duration}
									<span class="text-xs text-amber-500">
										{formatDuration(step.duration)}
									</span>
								{/if}
							</div>

							<!-- Step Content -->
							<div class="text-sm text-amber-800/90 whitespace-pre-wrap font-mono text-xs leading-relaxed pl-2 border-l-2 border-amber-200">
								{step.content}
							</div>
						</div>
					{/each}

					<!-- Streaming Cursor (appears after last step when streaming) -->
					{#if isStreaming}
						<span class="inline-block w-1.5 h-4 bg-amber-500 animate-pulse ml-0.5"></span>
					{/if}
				</div>
			{:else}
				<!-- Fallback: Display raw content if no steps -->
				<div class="px-4 pb-3 pt-2 text-sm text-amber-800/90 whitespace-pre-wrap font-mono text-xs leading-relaxed max-h-72 overflow-y-auto">
					{getContent || ''}
					{#if isStreaming}
						<span class="inline-block w-1.5 h-4 bg-amber-500 animate-pulse ml-0.5"></span>
					{/if}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	/* Ensure smooth transitions */
	button {
		transition: background-color 150ms ease;
	}

	/* Custom scrollbar for overflow content */
	.overflow-y-auto {
		scrollbar-width: thin;
		scrollbar-color: rgb(251 191 36) rgb(254 243 199);
	}

	.overflow-y-auto::-webkit-scrollbar {
		width: 8px;
	}

	.overflow-y-auto::-webkit-scrollbar-track {
		background: rgb(254 243 199);
		border-radius: 4px;
	}

	.overflow-y-auto::-webkit-scrollbar-thumb {
		background: rgb(251 191 36);
		border-radius: 4px;
	}

	.overflow-y-auto::-webkit-scrollbar-thumb:hover {
		background: rgb(245 158 11);
	}
</style>
