<script lang="ts">
	import ToolCallCard from './ToolCallCard.svelte';
	import SignalBadge from './SignalBadge.svelte';
	import { MessageActions } from '$lib/components/ai-elements';
	import {
		extractThinking,
		renderMarkdown,
		getArtifactIcon,
		getArtifactColor,
		parseMessageContent,
	} from '$lib/utils/chatHelpers';
	import { isCloudModelId } from '$lib/ai/modelCatalog';
	import type { SignalClassifiedEvent } from '$lib/utils/chatSSEParser';

	// ── Interfaces (mirror +page.svelte) ─────────────────────────────────────

	interface UsageData {
		input_tokens: number;
		output_tokens: number;
		thinking_tokens: number;
		total_tokens: number;
		duration_ms: number;
		tps: number;
		provider: string;
		model: string;
		estimated_cost: number;
	}

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		artifacts?: { title: string; type: string; content: string }[];
		usage?: UsageData;
	}

	type StreamingToolCall = {
		toolName: string;
		toolCallId: string;
		params: Record<string, unknown>;
		status: 'running' | 'completed' | 'error';
		result?: string;
	};

	interface GeneratedTask {
		title: string;
		description: string;
		priority: 'low' | 'medium' | 'high';
		assignee_id?: string;
		estimated_hours?: number;
	}

	// ── Props ────────────────────────────────────────────────────────────────

	interface Props {
		messages: ChatMessage[];
		isStreaming: boolean;
		loadingConversation: boolean;
		selectedModel: string;
		currentModelName: string;
		activeProvider: string;
		warmedUpModels: Set<string>;
		hasThinking: boolean;
		currentThinking: string;
		thinkingExpanded: boolean;
		streamingToolCalls: Map<string, StreamingToolCall>;
		streamingSignalMode: {
		mode: SignalClassifiedEvent['mode'];
		confidence?: number;
		genre?: 'DIRECT' | 'INFORM' | 'COMMIT' | 'DECIDE' | 'EXPRESS';
		docType?: string;
		weight?: number;
	} | null;
		artifactCompletedInStream: boolean;
		showInlineTaskCreation: boolean;
		creatingInlineTasks: boolean;
		inlineTasksForArtifact: GeneratedTask[];
		availableTeamMembers: { id: string; name: string; role: string }[];
		copiedMessageId: string | null;
		conversationId: string | null;
		showUsageInChat: boolean;
		onThinkingToggle: () => void;
		onViewArtifact: (artifact: { title: string; type: string; content: string }) => void;
		onGenerateTasks: (artifact: { title: string; type: string; content: string }) => void;
		onSaveToProfile: (artifact: { title: string; type: string; content: string }) => void;
		onCopyMessage: (content: string, id: string) => void;
		onConfirmInlineTasks: () => void;
		onDismissInlineTasks: () => void;
		onUpdateInlineTaskAssignee: (index: number, assigneeId: string) => void;
		onRemoveInlineTask: (index: number) => void;
	}

	let {
		messages,
		isStreaming,
		loadingConversation,
		selectedModel,
		currentModelName,
		activeProvider,
		warmedUpModels,
		hasThinking,
		currentThinking,
		thinkingExpanded,
		streamingToolCalls,
		streamingSignalMode,
		artifactCompletedInStream,
		showInlineTaskCreation,
		creatingInlineTasks,
		inlineTasksForArtifact,
		availableTeamMembers,
		copiedMessageId,
		conversationId,
		showUsageInChat,
		onThinkingToggle,
		onViewArtifact,
		onGenerateTasks,
		onSaveToProfile,
		onCopyMessage,
		onConfirmInlineTasks,
		onDismissInlineTasks,
		onUpdateInlineTaskAssignee,
		onRemoveInlineTask,
	}: Props = $props();

	// Show "Processing..." hint after 5 seconds of waiting for first content
	let showProcessingHint = $state(false);
	let processingHintTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		const lastMsg = messages[messages.length - 1];
		const waitingForContent =
			isStreaming &&
			lastMsg?.role === 'assistant' &&
			!lastMsg?.content &&
			!lastMsg?.artifacts?.length &&
			!hasThinking;

		if (waitingForContent) {
			processingHintTimer = setTimeout(() => {
				showProcessingHint = true;
			}, 5000);
		} else {
			showProcessingHint = false;
			if (processingHintTimer !== null) {
				clearTimeout(processingHintTimer);
				processingHintTimer = null;
			}
		}

		return () => {
			if (processingHintTimer !== null) {
				clearTimeout(processingHintTimer);
				processingHintTimer = null;
			}
		};
	});
</script>

{#if loadingConversation}
	<div class="flex items-center justify-center py-12">
		<div class="flex items-center gap-3 text-gray-500">
			<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
			<span class="text-sm">Loading conversation...</span>
		</div>
	</div>
{/if}

{#each messages as message, i (message.id)}
	{@const isLastMessage = i === messages.length - 1}
	{@const parsedParts = parseMessageContent(message.content)}

	{#if message.role === 'user'}
		<!-- User message - dark bubble on right -->
		<div class="flex justify-end">
			<div class="max-w-[90%] sm:max-w-[80%] bg-gray-900 text-white px-3 sm:px-4 py-2.5 sm:py-3 rounded-2xl rounded-br-md">
				<p class="text-sm sm:text-[15px] leading-relaxed whitespace-pre-wrap break-words">{message.content}</p>
			</div>
		</div>
	{:else if message.role === 'assistant'}
		<!-- Assistant message - left aligned -->
		<div class="max-w-[95%] sm:max-w-[85%]">
			<!-- Signal mode badge -->
			{#if isStreaming && isLastMessage && streamingSignalMode}
				<div class="mb-2">
					<SignalBadge
					mode={streamingSignalMode.mode}
					confidence={streamingSignalMode.confidence}
					genre={streamingSignalMode.genre}
					docType={streamingSignalMode.docType}
					weight={streamingSignalMode.weight}
				/>
				</div>
			{/if}

			<!-- Live Thinking Panel -->
			{#if isStreaming && isLastMessage && hasThinking && currentThinking}
				<div class="mb-3 border border-amber-200 rounded-xl overflow-hidden bg-amber-50/50 shadow-sm">
					<button
						onclick={onThinkingToggle}
						class="btn-pill btn-pill-ghost btn-pill-sm w-full flex items-center gap-2 text-left"
					>
						<svg class="w-4 h-4 text-amber-600 transition-transform duration-200 {thinkingExpanded ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
						<svg class="w-4 h-4 text-amber-600 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
						</svg>
						<span class="text-sm font-medium text-amber-700">Thinking</span>
						<span class="ml-2 w-2 h-2 bg-amber-500 rounded-full animate-pulse"></span>
						<span class="ml-auto text-xs text-amber-500">{Math.ceil(currentThinking.length / 4)} tokens</span>
					</button>
					{#if thinkingExpanded}
						<div class="px-4 pb-3 text-sm text-amber-800/90 whitespace-pre-wrap border-t border-amber-200 bg-amber-50/30 max-h-72 overflow-y-auto font-mono text-xs leading-relaxed">
							{currentThinking}<span class="inline-block w-1.5 h-4 bg-amber-500 animate-pulse ml-0.5"></span>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Streaming tool call cards -->
			{#if isStreaming && isLastMessage && streamingToolCalls.size > 0}
				<div class="mb-3 flex flex-col gap-2">
					{#each [...streamingToolCalls.values()] as tc (tc.toolCallId)}
						<ToolCallCard
							toolName={tc.toolName}
							toolCallId={tc.toolCallId}
							params={tc.params}
							status={tc.status}
							result={tc.result}
						/>
					{/each}
				</div>
			{/if}

			<!-- Loading indicator (waiting for first content) -->
			{#if !message.content && !message.artifacts?.length && isStreaming && isLastMessage && (!hasThinking || !currentThinking)}
				{@const modelId = selectedModel.toLowerCase()}
				{@const isLargeModel = modelId.includes(':30b') || modelId.includes(':32b') || modelId.includes(':70b') || modelId.includes(':72b') || modelId.includes(':235b')}
				{@const isColdStart = (activeProvider === 'ollama_local' || activeProvider === 'ollama_cloud') && !warmedUpModels.has(selectedModel)}
				<div class="flex flex-col gap-1.5 p-3 bg-gray-50 rounded-xl border border-gray-100">
					<div class="flex items-center gap-2 text-sm text-gray-600">
						<svg class="w-4 h-4 animate-spin text-blue-500" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						<span class="font-medium">
							{#if isColdStart}
								Loading {currentModelName}...
							{:else}
								Generating response...
							{/if}
						</span>
					</div>
					{#if isColdStart}
						<div class="text-xs text-gray-500 ml-6 space-y-0.5">
							<p>Loading model into memory for first use</p>
							{#if isLargeModel}
								<p class="text-orange-600">Large model ({currentModelName}) - this may take 30-60 seconds</p>
							{:else}
								<p>This usually takes 5-15 seconds</p>
							{/if}
						</div>
					{:else if isLargeModel}
						<div class="text-xs text-gray-500 ml-6">
							<p class="text-orange-600">Using large model - response may be slower</p>
						</div>
					{/if}
					{#if showProcessingHint}
						<div class="text-xs text-gray-400 ml-6 mt-1">
							<p>Still processing — the server is working on your request...</p>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Message content -->
			{#if message.content || message.artifacts?.length || (!isStreaming && message.role === 'assistant')}
				{@const extracted = extractThinking(message.content || '')}

				<!-- Thinking panel (persisted, post-stream) -->
				{#if extracted.thinking}
					<div class="mb-3 border border-amber-200 rounded-xl overflow-hidden bg-amber-50/50">
						<button
							onclick={onThinkingToggle}
							class="btn-pill btn-pill-ghost btn-pill-sm w-full flex items-center gap-2 text-left"
						>
							<svg class="w-4 h-4 text-amber-600 transition-transform duration-200 {thinkingExpanded ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
							</svg>
							<svg class="w-4 h-4 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
							</svg>
							<span class="text-sm font-medium text-amber-700">Thinking</span>
							<span class="ml-auto text-xs text-amber-500">{Math.ceil(extracted.thinking.length / 4)} tokens</span>
						</button>
						{#if thinkingExpanded}
							<div class="px-4 pb-3 text-sm text-amber-800/90 whitespace-pre-wrap border-t border-amber-200 bg-amber-50/30 max-h-72 overflow-y-auto font-mono text-xs leading-relaxed">
								{extracted.thinking}
							</div>
						{/if}
					</div>
				{/if}

				<!-- Main text content -->
				{#if extracted.mainContent}
					<div class="text-sm sm:text-[15px] leading-relaxed text-gray-800 dark:text-gray-100 prose prose-sm dark:prose-invert max-w-none streaming-content">
						{@html renderMarkdown(extracted.mainContent)}{#if isLastMessage && isStreaming && !artifactCompletedInStream}<span class="streaming-cursor"></span>{/if}
					</div>
				{/if}

				<!-- Artifacts from message.artifacts (new approach) -->
				{#if message.artifacts?.length}
					{#each message.artifacts as artifact}
						{#if artifact.content === '__generating__'}
							<!-- Generating artifact card -->
							<div class="my-3 flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-blue-50 to-purple-50 border border-blue-200 rounded-xl animate-pulse">
								<div class="w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center flex-shrink-0">
									<svg class="w-5 h-5 text-blue-600 animate-spin" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{artifact.title}</p>
									<p class="text-xs text-gray-500 capitalize">{artifact.type} &bull; Creating...</p>
								</div>
								<div class="h-2 w-16 bg-blue-200 rounded-full overflow-hidden">
									<div class="h-full bg-blue-500 rounded-full animate-pulse" style="width: 60%"></div>
								</div>
							</div>
						{:else}
							<!-- Completed artifact card -->
							<div class="my-3">
								<button
									onclick={() => onViewArtifact(artifact)}
									class="flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-blue-50 to-purple-50 border border-blue-200 rounded-t-xl hover:shadow-md hover:border-blue-300 transition-all cursor-pointer w-full text-left group"
								>
									<div class="w-10 h-10 rounded-lg {getArtifactColor(artifact.type)} flex items-center justify-center flex-shrink-0">
										<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getArtifactIcon(artifact.type)} />
										</svg>
									</div>
									<div class="flex-1 min-w-0">
										<p class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{artifact.title}</p>
										<p class="text-xs text-gray-500 capitalize">{artifact.type} &bull; Click to view</p>
									</div>
									<svg class="w-5 h-5 text-gray-400 group-hover:text-blue-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
									</svg>
								</button>
								<!-- Action buttons -->
								<div class="flex items-center gap-2 px-3 py-2 bg-gray-50 border border-t-0 border-gray-200 rounded-b-xl">
									<button
										onclick={() => onGenerateTasks(artifact)}
										class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-green-700 bg-green-50 hover:bg-green-100 rounded-lg transition-colors"
									>
										<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
										</svg>
										Generate Tasks
									</button>
									<button
										onclick={() => onViewArtifact(artifact)}
										class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
									>
										<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
										</svg>
										View
									</button>
									<button
										onclick={() => onSaveToProfile(artifact)}
										class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
									>
										<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
										</svg>
										Save to Profile
									</button>
								</div>
							</div>
						{/if}
					{/each}
				{/if}

				<!-- Fallback: parsed artifacts from content (legacy) -->
				{#if !message.artifacts?.length}
					{#each parsedParts as part}
						{#if part.type === 'artifact' && part.artifact}
							<button
								onclick={() => onViewArtifact(part.artifact!)}
								class="my-3 flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-blue-50 to-purple-50 border border-blue-200 rounded-xl hover:shadow-md hover:border-blue-300 transition-all cursor-pointer w-full text-left group"
							>
								<div class="w-10 h-10 rounded-lg {getArtifactColor(part.artifact.type)} flex items-center justify-center flex-shrink-0">
									<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getArtifactIcon(part.artifact.type)} />
									</svg>
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium text-gray-900 truncate">{part.artifact.title}</p>
									<p class="text-xs text-gray-500 capitalize">{part.artifact.type} &bull; Click to view</p>
								</div>
								<svg class="w-5 h-5 text-gray-400 group-hover:text-blue-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
								</svg>
							</button>
						{:else if part.type === 'text' && part.text && !message.content}
							<p class="text-[15px] leading-relaxed text-gray-800 whitespace-pre-wrap">{part.text}</p>
						{/if}
					{/each}
				{/if}
			{/if}

			<!-- Inline Task Creation (after artifact) -->
			{#if isLastMessage && showInlineTaskCreation}
				<div class="my-4 p-4 bg-gradient-to-br from-green-50 to-emerald-50 border border-green-200 rounded-xl">
					<div class="flex items-center justify-between mb-3">
						<div class="flex items-center gap-2">
							<svg class="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
							</svg>
							<h4 class="font-medium text-gray-900">Create Tasks from Artifact?</h4>
						</div>
						<button
							onclick={onDismissInlineTasks}
							class="btn-pill btn-pill-ghost btn-pill-icon"
							aria-label="Dismiss task creation"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					</div>

					{#if creatingInlineTasks}
						<div class="flex items-center gap-2 py-4 justify-center">
							<svg class="w-5 h-5 animate-spin text-green-600" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							<span class="text-sm text-gray-600">Analyzing artifact and generating tasks...</span>
						</div>
					{:else if inlineTasksForArtifact.length === 0}
						<p class="text-sm text-gray-500 text-center py-3">No actionable tasks found in this artifact.</p>
						<button
							onclick={onDismissInlineTasks}
							class="btn-pill btn-pill-ghost btn-pill-sm w-full mt-2"
						>
							Dismiss
						</button>
					{:else}
						<div class="space-y-2 mb-4 max-h-64 overflow-y-auto">
							{#each inlineTasksForArtifact as task, idx}
								<div class="flex items-start gap-3 p-3 bg-white rounded-lg border border-gray-200">
									<div class="flex-1 min-w-0">
										<p class="text-sm font-medium text-gray-900">{task.title}</p>
										{#if task.description}
											<p class="text-xs text-gray-500 mt-0.5 line-clamp-2">{task.description}</p>
										{/if}
										<div class="flex items-center gap-2 mt-2">
											<span class="px-2 py-0.5 text-xs rounded-full {task.priority === 'high' ? 'bg-red-100 text-red-700' : task.priority === 'medium' ? 'bg-yellow-100 text-yellow-700' : 'bg-gray-100 text-gray-700'}">
												{task.priority}
											</span>
											<select
												value={task.assignee_id || ''}
												onchange={(e) => onUpdateInlineTaskAssignee(idx, (e.target as HTMLSelectElement).value)}
												class="text-xs border border-gray-200 rounded px-2 py-1 bg-white"
											>
												<option value="">Unassigned</option>
												{#each availableTeamMembers as member}
													<option value={member.id}>{member.name} ({member.role})</option>
												{/each}
											</select>
										</div>
									</div>
									<button
										onclick={() => onRemoveInlineTask(idx)}
										class="p-1 text-gray-400 hover:text-red-500 rounded"
										aria-label="Remove task"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
						</div>

						<div class="flex gap-2">
							<button
								onclick={onDismissInlineTasks}
								class="btn-pill btn-pill-ghost btn-pill-sm flex-1"
							>
								Skip
							</button>
							<button
								onclick={onConfirmInlineTasks}
								disabled={creatingInlineTasks}
								class="btn-pill btn-pill-success btn-pill-sm flex-1 flex items-center justify-center gap-2"
							>
								{#if creatingInlineTasks}
									<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
									</svg>
									Creating...
								{:else}
									Create {inlineTasksForArtifact.length} Task{inlineTasksForArtifact.length > 1 ? 's' : ''}
								{/if}
							</button>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Message action bar -->
			{#if (message.content || message.artifacts?.length || parsedParts.length > 0) && (!isStreaming || !isLastMessage || artifactCompletedInStream)}
				<div class="flex items-center gap-2 mt-3">
					<MessageActions
						messageId={message.id}
						conversationId={conversationId || undefined}
						agentType={selectedModel}
						originalContent={message.content}
						onCopy={() => onCopyMessage(message.content, message.id)}
						copied={copiedMessageId === message.id}
					/>

					<!-- Usage stats -->
					{#if message.usage && showUsageInChat}
						<div class="flex items-center gap-3 text-xs text-gray-400 dark:text-gray-500 ml-2 pl-2 border-l border-gray-200 dark:border-gray-700">
							<span class="flex items-center gap-1" title="Tokens per second">
								<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
								</svg>
								{message.usage.tps.toFixed(1)} t/s
							</span>
							<span title="Total tokens">{message.usage.total_tokens} tokens</span>
							{#if message.usage.thinking_tokens > 0}
								<span class="text-purple-400" title="Thinking tokens (COT)">{message.usage.thinking_tokens} thinking</span>
							{/if}
							<span class="text-gray-300 dark:text-gray-600">.</span>
							<span title="Provider">{isCloudModelId(message.usage.model) ? 'Cloud' : (message.usage.provider === 'ollama_local' ? 'Local' : message.usage.provider)}</span>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	{/if}
{/each}

{#if isStreaming && messages[messages.length - 1]?.role === 'user'}
	<!-- Typing indicator -->
	<div class="flex items-center gap-1.5">
		<div class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0ms"></div>
		<div class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 150ms"></div>
		<div class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 300ms"></div>
	</div>
{/if}

<style>
	/* ── Mobile: ChatMessageList ────────────────────────────────────── */

	/* Code blocks inside messages must scroll horizontally within the bubble,
	   never cause the page itself to overflow. */
	:global(.streaming-content pre),
	:global(.prose pre) {
		overflow-x: auto;
		max-width: 100%;
		word-break: normal;
		white-space: pre;
	}

	/* Inline code: allow wrapping on mobile rather than blowing the bubble */
	:global(.streaming-content code:not(pre code)),
	:global(.prose code:not(pre code)) {
		word-break: break-word;
	}

	/* Tables inside messages: horizontal scroll within bubble */
	:global(.streaming-content table),
	:global(.prose table) {
		display: block;
		overflow-x: auto;
		max-width: 100%;
	}

	@media (max-width: 767px) {
		/* User message bubble: ensure no horizontal bleed */
		:global(.chat-user-bubble) {
			max-width: 92% !important;
			word-break: break-word;
			overflow-wrap: break-word;
		}

		/* Assistant message: full width on mobile */
		:global(.chat-assistant-message) {
			max-width: 100% !important;
		}
	}
</style>
