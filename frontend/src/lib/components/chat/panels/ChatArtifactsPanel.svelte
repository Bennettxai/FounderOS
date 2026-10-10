<script lang="ts">
	import { fly } from 'svelte/transition';
	import ArtifactEditor from '$lib/components/artifacts/ArtifactEditor.svelte';

	interface ArtifactListItem {
		id: string;
		title: string;
		type: string;
		summary?: string | null;
		version?: number;
		context_name?: string | null;
		project_id?: string | null;
		context_id?: string | null;
		conversation_id?: string | null;
		created_at?: string;
		updated_at?: string;
		content?: string;
		fromMessage?: boolean;
		messageId?: string;
	}

	interface ViewingArtifact {
		title: string;
		type: string;
		content: string;
	}

	interface Props {
		allArtifacts: ArtifactListItem[];
		selectedArtifact: { id: string; title: string; type: string; content: string; version: number } | null;
		viewingArtifactFromMessage: ViewingArtifact | null;
		generatingArtifact: boolean;
		generatingArtifactTitle: string;
		generatingArtifactType: string;
		generatingArtifactContent: string;
		loadingArtifacts: boolean;
		artifactFilter: string;
		// Helpers
		getArtifactIcon: (type: string) => string;
		getArtifactColor: (type: string) => string;
		renderMarkdown: (md: string) => string;
		// Callbacks
		onClose: () => void;
		onFilterChange: (filter: string) => void;
		onSelectArtifact: (id: string) => void;
		onViewMessageArtifact: (artifact: ViewingArtifact) => void;
		onDeleteArtifact: (id: string) => void;
		onSaveToProfile: () => void;
		onCopyContent: (content: string) => void;
		onUpdateContent: (content: string) => void;
		onUpdateSelectedContent: (content: string) => void;
		onStartResize: (e: MouseEvent) => void;
	}

	let {
		allArtifacts,
		selectedArtifact,
		viewingArtifactFromMessage,
		generatingArtifact,
		generatingArtifactTitle,
		generatingArtifactType,
		generatingArtifactContent,
		loadingArtifacts,
		artifactFilter,
		getArtifactIcon,
		getArtifactColor,
		renderMarkdown,
		onClose,
		onFilterChange,
		onSelectArtifact,
		onViewMessageArtifact,
		onDeleteArtifact,
		onSaveToProfile,
		onCopyContent,
		onUpdateContent,
		onUpdateSelectedContent,
		onStartResize,
	}: Props = $props();

	const FILTER_TABS = ['all', 'proposal', 'sop', 'framework', 'plan', 'report'] as const;
</script>

<!-- Resize Handle — hidden on mobile (touch-dragging a 4px target is unusable) -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
	class="artifacts-resize-handle w-1 h-full bg-gray-200 hover:bg-blue-400 cursor-col-resize flex-shrink-0 transition-colors relative group"
	onmousedown={onStartResize}
	role="separator"
	aria-orientation="vertical"
	aria-label="Resize artifacts panel"
>
	<div class="absolute inset-y-0 -left-1 -right-1 group-hover:bg-blue-400/20"></div>
	<div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
		<div class="w-1 h-1 rounded-full bg-gray-400"></div>
		<div class="w-1 h-1 rounded-full bg-gray-400"></div>
		<div class="w-1 h-1 rounded-full bg-gray-400"></div>
	</div>
</div>

<!-- Panel -->
<div
	class="artifacts-panel h-full flex flex-col bg-white dark:bg-gray-900 flex-shrink-0"
	transition:fly={{ x: 320, duration: 200 }}
>
	<!-- Panel Header -->
	<div class="p-4 border-b border-gray-100 dark:border-gray-700 flex-shrink-0">
		<div class="flex items-center justify-between mb-3">
			<h3 class="font-semibold text-gray-900 dark:text-gray-100">Artifacts</h3>
			<button
				onclick={onClose}
				class="btn-pill btn-pill-ghost btn-pill-icon"
				aria-label="Close artifacts panel"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<!-- Filter tabs (only when not viewing a message artifact) -->
		{#if !viewingArtifactFromMessage}
			<div class="flex gap-1 overflow-x-auto">
				{#each FILTER_TABS as filter}
					<button
						onclick={() => onFilterChange(filter)}
						class="px-2.5 py-1 text-xs font-medium rounded-lg whitespace-nowrap transition-colors {artifactFilter === filter ? 'bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'}"
					>
						{filter === 'all' ? 'All' : filter.charAt(0).toUpperCase() + filter.slice(1)}
					</button>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Content: Generating | Message Artifact | Selected Artifact | List -->
	{#if generatingArtifact}
		<!-- Live Generation View -->
		<div class="flex-1 flex flex-col overflow-hidden bg-gray-50 dark:bg-[#1c1c1e]">
			<div class="p-4 border-b border-white/10 flex-shrink-0">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-lg {generatingArtifactType ? getArtifactColor(generatingArtifactType) : 'bg-white/10 text-gray-300'} flex items-center justify-center flex-shrink-0 relative">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={generatingArtifactType ? getArtifactIcon(generatingArtifactType) : 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z'} />
						</svg>
						<div class="absolute -top-1 -right-1 w-3 h-3">
							<span class="absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75 animate-ping"></span>
							<span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
						</div>
					</div>
					<div class="min-w-0 flex-1">
						<h4 class="font-medium text-gray-100 truncate">{generatingArtifactTitle || 'Generating artifact...'}</h4>
						<p class="text-xs text-gray-400 flex items-center gap-1.5">
							{#if generatingArtifactType}
								<span class="capitalize">{generatingArtifactType}</span>
								<span>&bull;</span>
							{/if}
							<span class="flex items-center gap-1 text-green-400">
								<svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
								Writing...
							</span>
						</p>
					</div>
				</div>
			</div>
			<div class="flex-1 overflow-y-auto p-4">
				<div class="prose prose-sm dark:prose-invert max-w-none" style="--tw-prose-body: var(--dt); --tw-prose-headings: var(--dt); --tw-prose-bold: var(--dt);">
					{@html renderMarkdown(generatingArtifactContent || 'Waiting for content...')}
					<span class="inline-block w-2 h-4 bg-green-500 animate-pulse ml-0.5"></span>
				</div>
			</div>
		</div>

	{:else if viewingArtifactFromMessage}
		<!-- Viewing artifact from message -->
		<div class="flex-1 flex flex-col overflow-hidden bg-gray-50 dark:bg-[#1c1c1e]">
			<div class="p-4 border-b border-white/10 flex-shrink-0">
				<div class="flex items-start gap-3">
					<div class="w-10 h-10 rounded-lg {getArtifactColor(viewingArtifactFromMessage.type)} flex items-center justify-center flex-shrink-0">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={getArtifactIcon(viewingArtifactFromMessage.type)} />
						</svg>
					</div>
					<div class="min-w-0 flex-1">
						<h4 class="font-medium text-gray-100">{viewingArtifactFromMessage.title}</h4>
						<p class="text-xs text-gray-400 capitalize">{viewingArtifactFromMessage.type}</p>
					</div>
					<div class="flex items-center gap-1">
						<button
							onclick={() => onCopyContent(viewingArtifactFromMessage?.content || '')}
							class="p-2 text-gray-400 hover:text-gray-200 hover:bg-white/10 rounded-lg transition-colors"
							title="Copy"
							aria-label="Copy artifact content"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
							</svg>
						</button>
						<button
							class="btn-pill btn-pill-ghost btn-pill-icon"
							title="Export"
							aria-label="Export artifact"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
							</svg>
						</button>
						<button
							onclick={onSaveToProfile}
							class="btn-pill btn-pill-ghost btn-pill-icon"
							title="Save to Knowledge Base"
							aria-label="Save artifact to profile"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
							</svg>
						</button>
					</div>
				</div>
			</div>
			<ArtifactEditor
				artifact={viewingArtifactFromMessage}
				onSave={onUpdateContent}
				darkMode={true}
			/>
		</div>

	{:else if selectedArtifact}
		<!-- Artifact Detail View (from API) -->
		<div class="flex-1 flex flex-col overflow-hidden bg-gray-50 dark:bg-[#1c1c1e]">
			<div class="p-4 border-b border-white/10 flex-shrink-0">
				<div class="flex items-start gap-3">
					<div class="w-10 h-10 rounded-lg {getArtifactColor(selectedArtifact.type)} flex items-center justify-center flex-shrink-0">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={getArtifactIcon(selectedArtifact.type)} />
						</svg>
					</div>
					<div class="min-w-0 flex-1">
						<h4 class="font-medium text-gray-100 truncate">{selectedArtifact.title}</h4>
						<p class="text-xs text-gray-400 capitalize">{selectedArtifact.type} &bull; v{selectedArtifact.version}</p>
					</div>
					<div class="flex items-center gap-1">
						<button
							onclick={() => onCopyContent(selectedArtifact?.content || '')}
							class="p-2 text-gray-400 hover:text-gray-200 hover:bg-white/10 rounded-lg transition-colors"
							title="Copy"
							aria-label="Copy artifact content"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
							</svg>
						</button>
						<button
							class="btn-pill btn-pill-ghost btn-pill-icon"
							title="Export"
							aria-label="Export artifact"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
							</svg>
						</button>
					</div>
				</div>
			</div>
			<ArtifactEditor
				artifact={{ title: selectedArtifact.title, type: selectedArtifact.type, content: selectedArtifact.content }}
				onSave={onUpdateSelectedContent}
				darkMode={true}
			/>
		</div>

	{:else}
		<!-- Artifacts List -->
		<div class="flex-1 overflow-y-auto bg-gray-50 dark:bg-[#1c1c1e]">
			{#if loadingArtifacts}
				<div class="flex items-center justify-center h-32">
					<div class="animate-spin h-6 w-6 border-2 border-white/30 border-t-transparent rounded-full"></div>
				</div>
			{:else if allArtifacts.length === 0}
				<div class="flex flex-col items-center justify-center h-48 text-center px-4">
					<div class="w-12 h-12 rounded-full bg-white/10 flex items-center justify-center mb-3">
						<svg class="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
						</svg>
					</div>
					<p class="text-sm text-gray-400">No artifacts yet</p>
					<p class="text-xs text-gray-500 mt-1">Ask OSA to create proposals, SOPs, or frameworks</p>
				</div>
			{:else}
				<div class="p-2 space-y-1">
					{#each allArtifacts as artifact (artifact.id)}
						<div class="group relative">
							<button
								onclick={() => {
									if (artifact.fromMessage) {
										onViewMessageArtifact({
											title: artifact.title,
											type: artifact.type,
											content: artifact.content || ''
										});
									} else {
										onSelectArtifact(artifact.id);
									}
								}}
								class="w-full flex items-start gap-3 p-3 rounded-lg hover:bg-white/5 transition-colors text-left"
							>
								<div class="w-9 h-9 rounded-lg {getArtifactColor(artifact.type)} flex items-center justify-center flex-shrink-0">
									<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={getArtifactIcon(artifact.type)} />
									</svg>
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium text-gray-100 truncate">{artifact.title}</p>
									{#if artifact.summary}
										<p class="text-xs text-gray-400 line-clamp-2 mt-0.5">{artifact.summary}</p>
									{/if}
									<div class="flex items-center gap-1.5 mt-1">
										<span class="text-xs text-gray-500 capitalize">{artifact.type}</span>
										{#if artifact.context_name}
											<span class="text-xs text-gray-600">&bull;</span>
											<span class="text-xs text-blue-400 truncate">{artifact.context_name}</span>
										{:else if artifact.project_id}
											<span class="text-xs text-gray-600">&bull;</span>
											<span class="text-xs text-purple-400 truncate">Linked to project</span>
										{:else}
											<span class="text-xs text-gray-600">&bull;</span>
											<span class="text-xs text-gray-500 italic">Unlinked</span>
										{/if}
									</div>
								</div>
							</button>
							<!-- Delete button: shows on hover -->
							<button
								onclick={(e) => { e.stopPropagation(); onDeleteArtifact(artifact.id); }}
								class="absolute right-2 top-2 p-1.5 rounded-md text-gray-400 hover:text-red-500 hover:bg-red-50 opacity-0 group-hover:opacity-100 transition-all"
								title="Delete artifact"
								aria-label="Delete artifact {artifact.title}"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
								</svg>
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	/* ── Mobile: ChatArtifactsPanel ────────────────────────────────── */

	@media (max-width: 767px) {
		/* Hide the column-resize handle — dragging a 4px strip is unusable on touch */
		:global(.artifacts-resize-handle) {
			display: none;
		}

		/* Artifacts panel: full-width full-height overlay instead of right-side column */
		.artifacts-panel {
			position: fixed;
			inset: 0;
			width: 100% !important;
			height: 100% !important;
			z-index: 55;
			overflow-y: auto;
		}
	}
</style>
