<script lang="ts">
	/**
	 * ToolCallCard Component
	 *
	 * Renders inline in chat messages for tool/skill invocations.
	 * Shows tool name, collapsible params, spinner while running,
	 * and result content on completion.
	 */

	interface Props {
		toolName: string;
		toolCallId: string;
		params: Record<string, unknown>;
		status: 'running' | 'completed' | 'error';
		result?: string;
	}

	let { toolName, toolCallId, params, status, result }: Props = $props();

	// Starts expanded while running; collapses on completion or error
	let isExpanded = $state(status === 'running');
	let paramsExpanded = $state(status === 'running');
	let resultExpanded = $state(true);

	// When status transitions from running → completed/error, collapse params
	$effect(() => {
		if (status !== 'running') {
			paramsExpanded = false;
		}
	});

	const formattedParams = $derived(JSON.stringify(params, null, 2));

	const statusConfig = $derived.by(() => {
		switch (status) {
			case 'running':
				return {
					borderColor: 'border-blue-200',
					bgColor: 'bg-blue-50/50',
					headerHover: 'hover:bg-blue-100/50',
					iconColor: 'text-blue-600',
					badgeBg: 'bg-blue-100',
					badgeText: 'text-blue-700',
					badgeLabel: 'Running',
					scrollbarThumb: 'rgb(96 165 250)',
					scrollbarTrack: 'rgb(239 246 255)'
				};
			case 'completed':
				return {
					borderColor: 'border-green-200',
					bgColor: 'bg-green-50/50',
					headerHover: 'hover:bg-green-100/50',
					iconColor: 'text-green-600',
					badgeBg: 'bg-green-100',
					badgeText: 'text-green-700',
					badgeLabel: 'Done',
					scrollbarThumb: 'rgb(74 222 128)',
					scrollbarTrack: 'rgb(240 253 244)'
				};
			case 'error':
				return {
					borderColor: 'border-red-200',
					bgColor: 'bg-red-50/50',
					headerHover: 'hover:bg-red-100/50',
					iconColor: 'text-red-600',
					badgeBg: 'bg-red-100',
					badgeText: 'text-red-700',
					badgeLabel: 'Error',
					scrollbarThumb: 'rgb(252 165 165)',
					scrollbarTrack: 'rgb(254 242 242)'
				};
		}
	});

	function toggleExpanded() {
		isExpanded = !isExpanded;
	}

	function toggleParams(e: MouseEvent | KeyboardEvent) {
		if (e instanceof KeyboardEvent && e.key !== 'Enter' && e.key !== ' ') return;
		e.preventDefault();
		paramsExpanded = !paramsExpanded;
	}
</script>

<div
	class="border {statusConfig.borderColor} rounded-xl overflow-hidden {statusConfig.bgColor} shadow-sm"
	id="tool-call-{toolCallId}"
>
	<!-- Header -->
	<button
		onclick={toggleExpanded}
		class="btn-pill btn-pill-ghost btn-pill-sm w-full flex items-center gap-2 text-left {statusConfig.headerHover} transition-colors"
		aria-expanded={isExpanded}
		aria-label="{isExpanded ? 'Collapse' : 'Expand'} tool call: {toolName}"
	>
		<!-- Chevron -->
		<svg
			class="w-4 h-4 {statusConfig.iconColor} transition-transform duration-200 {isExpanded ? 'rotate-90' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
		</svg>

		<!-- Status Icon -->
		{#if status === 'running'}
			<!-- Spinner -->
			<svg
				class="w-4 h-4 {statusConfig.iconColor} animate-spin"
				fill="none"
				viewBox="0 0 24 24"
				aria-hidden="true"
			>
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				/>
			</svg>
		{:else if status === 'completed'}
			<!-- Checkmark -->
			<svg
				class="w-4 h-4 {statusConfig.iconColor}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				aria-hidden="true"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
			</svg>
		{:else}
			<!-- X mark -->
			<svg
				class="w-4 h-4 {statusConfig.iconColor}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				aria-hidden="true"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
		{/if}

		<!-- Tool Name -->
		<span class="text-sm font-medium {statusConfig.iconColor} font-mono">{toolName}</span>

		<!-- Status Badge -->
		<span
			class="ml-auto text-xs font-medium px-2 py-0.5 rounded {statusConfig.badgeBg} {statusConfig.badgeText}"
		>
			{statusConfig.badgeLabel}
		</span>
	</button>

	<!-- Expanded Content -->
	{#if isExpanded}
		<div class="border-t {statusConfig.borderColor}">
			<!-- Params Section -->
			<div class="px-3 pt-2 pb-1">
				<button
					onclick={toggleParams}
					onkeydown={toggleParams}
					class="btn-pill btn-pill-ghost btn-pill-xs flex items-center gap-1.5 {statusConfig.iconColor} opacity-70 hover:opacity-100 transition-opacity mb-1"
					aria-expanded={paramsExpanded}
					aria-label="{paramsExpanded ? 'Collapse' : 'Expand'} parameters"
				>
					<svg
						class="w-3 h-3 transition-transform duration-150 {paramsExpanded ? 'rotate-90' : ''}"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
					Parameters
				</button>

				{#if paramsExpanded}
					<pre
						class="text-xs font-mono leading-relaxed whitespace-pre-wrap break-all max-h-48 overflow-y-auto rounded-lg p-2 bg-black/5 {statusConfig.iconColor} opacity-80 params-scroll"
					>{formattedParams}</pre>
				{/if}
			</div>

			<!-- Result Section -->
			{#if result !== undefined && result !== ''}
				<div class="px-3 pb-2 pt-1 border-t {statusConfig.borderColor}">
					<button
						onclick={() => (resultExpanded = !resultExpanded)}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								e.preventDefault();
								resultExpanded = !resultExpanded;
							}
						}}
						class="flex items-center gap-1.5 text-xs font-medium {statusConfig.iconColor} opacity-70 hover:opacity-100 transition-opacity mb-1"
						aria-expanded={resultExpanded}
						aria-label="{resultExpanded ? 'Collapse' : 'Expand'} {status === 'error' ? 'error' : 'result'}"
					>
						<svg
							class="w-3 h-3 transition-transform duration-150 {resultExpanded ? 'rotate-90' : ''}"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
						{status === 'error' ? 'Error' : 'Result'}
					</button>
					{#if resultExpanded}
						<pre
							class="text-xs font-mono leading-relaxed whitespace-pre-wrap break-all max-h-64 overflow-y-auto rounded-lg p-2 bg-black/5 {statusConfig.iconColor} opacity-80 result-scroll"
						>{result}</pre>
					{/if}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.params-scroll {
		scrollbar-width: thin;
		scrollbar-color: rgb(148 163 184) rgb(241 245 249);
	}

	.params-scroll::-webkit-scrollbar {
		width: 6px;
	}

	.params-scroll::-webkit-scrollbar-track {
		background: rgb(241 245 249);
		border-radius: 3px;
	}

	.params-scroll::-webkit-scrollbar-thumb {
		background: rgb(148 163 184);
		border-radius: 3px;
	}

	.params-scroll::-webkit-scrollbar-thumb:hover {
		background: rgb(100 116 139);
	}

	.result-scroll {
		scrollbar-width: thin;
		scrollbar-color: rgb(148 163 184) rgb(241 245 249);
	}

	.result-scroll::-webkit-scrollbar {
		width: 6px;
	}

	.result-scroll::-webkit-scrollbar-track {
		background: rgb(241 245 249);
		border-radius: 3px;
	}

	.result-scroll::-webkit-scrollbar-thumb {
		background: rgb(148 163 184);
		border-radius: 3px;
	}

	.result-scroll::-webkit-scrollbar-thumb:hover {
		background: rgb(100 116 139);
	}
</style>
