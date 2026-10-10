<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { X, FilePlus, FileMinus, FileEdit, Loader2, AlertTriangle, Columns2, Rows2 } from 'lucide-svelte';
	import MonacoDiffEditor from '$lib/editor/MonacoDiffEditor.svelte';
	import { compareVersions } from '$lib/api/versions';
	import type { BackendVersionDiffResult, BackendFileDiff } from '$lib/api/versions/types';

	interface Props {
		isOpen: boolean;
		workspaceId: string;
		fromVersion: string;
		toVersion: string;
		fromDisplayNum: number;
		toDisplayNum: number;
		onClose: () => void;
	}

	let {
		isOpen,
		workspaceId,
		fromVersion,
		toVersion,
		fromDisplayNum,
		toDisplayNum,
		onClose,
	}: Props = $props();

	let diffResult = $state<BackendVersionDiffResult | null>(null);
	let selectedFile = $state<BackendFileDiff | null>(null);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let sideBySide = $state(true);

	let changedFiles = $derived(
		diffResult?.files.filter((f) => f.change_type !== 'unchanged') ?? []
	);

	$effect(() => {
		if (isOpen && workspaceId && fromVersion && toVersion) {
			loadDiff();
		}
	});

	async function loadDiff() {
		loading = true;
		error = null;
		diffResult = null;
		selectedFile = null;

		try {
			diffResult = await compareVersions(workspaceId, fromVersion, toVersion);
			// Auto-select first changed file
			const changed = diffResult.files.filter((f) => f.change_type !== 'unchanged');
			if (changed.length > 0) {
				selectedFile = changed[0];
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load diff';
		} finally {
			loading = false;
		}
	}

	function getFileName(path: string): string {
		return path.split('/').pop() ?? path;
	}

	function getFileDir(path: string): string {
		const parts = path.split('/');
		return parts.length > 1 ? parts.slice(0, -1).join('/') + '/' : '';
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onClose();
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		window.removeEventListener('keydown', handleKeydown);
	});
</script>

{#if isOpen}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
		onclick={handleBackdropClick}
	>
		<div class="diff-modal bg-white dark:bg-[var(--dbg)] border border-gray-200 dark:border-white/10 rounded-xl shadow-2xl flex flex-col overflow-hidden"
			style="width: 92vw; max-width: 1500px; height: 85vh;"
		>
			<!-- Header -->
			<header class="flex items-center justify-between px-5 py-3 border-b border-gray-200 dark:border-white/10 bg-gray-50/50 dark:bg-white/[0.02]">
				<div class="flex items-center gap-4">
					<h2 class="text-sm font-semibold text-gray-900 dark:text-white">
						Comparing
						<span class="text-indigo-400">v{fromDisplayNum}</span>
						<span class="text-gray-400 dark:text-gray-500 mx-1">&rarr;</span>
						<span class="text-indigo-400">v{toDisplayNum}</span>
					</h2>

					{#if diffResult}
						<div class="flex items-center gap-3 text-xs">
							{#if diffResult.summary.files_added > 0}
								<span class="text-emerald-400">+{diffResult.summary.files_added} added</span>
							{/if}
							{#if diffResult.summary.files_removed > 0}
								<span class="text-red-400">-{diffResult.summary.files_removed} removed</span>
							{/if}
							{#if diffResult.summary.files_modified > 0}
								<span class="text-amber-400">~{diffResult.summary.files_modified} modified</span>
							{/if}
							<span class="text-gray-500">|</span>
							<span class="text-emerald-400/70">+{diffResult.summary.total_lines_added}</span>
							<span class="text-red-400/70">-{diffResult.summary.total_lines_removed}</span>
						</div>
					{/if}
				</div>

				<button
					onclick={onClose}
					class="btn-pill btn-pill-ghost btn-pill-icon"
				>
					<X size={18} />
				</button>
			</header>

			<!-- Body -->
			<div class="flex flex-1 overflow-hidden">
				<!-- File list sidebar -->
				<div class="w-60 border-r border-gray-200 dark:border-white/10 overflow-y-auto bg-gray-50/30 dark:bg-white/[0.01] flex-shrink-0">
					{#if loading}
						<div class="flex items-center justify-center h-full">
							<Loader2 size={20} class="animate-spin text-indigo-400" />
						</div>
					{:else if changedFiles.length === 0}
						<div class="flex flex-col items-center justify-center h-full text-gray-500 dark:text-gray-500 text-xs p-4 text-center">
							<p>No file changes found</p>
						</div>
					{:else}
						<div class="py-1">
							{#each changedFiles as file}
								<button
									onclick={() => selectedFile = file}
									class="w-full px-3 py-2 flex items-center gap-2 text-left text-xs transition-colors
										{selectedFile === file ? 'bg-indigo-500/15 text-indigo-700 dark:text-white' : 'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-white/5 hover:text-gray-700 dark:hover:text-gray-200'}"
								>
									{#if file.change_type === 'added'}
										<FilePlus size={14} class="text-emerald-400 flex-shrink-0" />
									{:else if file.change_type === 'removed'}
										<FileMinus size={14} class="text-red-400 flex-shrink-0" />
									{:else}
										<FileEdit size={14} class="text-amber-400 flex-shrink-0" />
									{/if}
									<div class="min-w-0 flex-1">
										<div class="truncate font-medium">{getFileName(file.file_path)}</div>
										{#if getFileDir(file.file_path)}
											<div class="truncate text-[10px] text-gray-500 dark:text-gray-600">{getFileDir(file.file_path)}</div>
										{/if}
									</div>
									<div class="flex-shrink-0 text-[10px] tabular-nums">
										{#if file.lines_added > 0}
											<span class="text-emerald-500">+{file.lines_added}</span>
										{/if}
										{#if file.lines_removed > 0}
											<span class="text-red-500 ml-1">-{file.lines_removed}</span>
										{/if}
									</div>
								</button>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Diff editor area -->
				<div class="flex-1 flex flex-col overflow-hidden">
					{#if loading}
						<div class="flex items-center justify-center h-full">
							<div class="flex flex-col items-center gap-3 text-gray-400">
								<Loader2 size={28} class="animate-spin text-indigo-400" />
								<span class="text-sm">Loading diff...</span>
							</div>
						</div>
					{:else if error}
						<div class="flex items-center justify-center h-full">
							<div class="flex flex-col items-center gap-3 text-center p-6">
								<AlertTriangle size={28} class="text-amber-400" />
								<p class="text-sm text-gray-300">{error}</p>
								<button
									onclick={loadDiff}
									class="btn-pill btn-pill-soft btn-pill-xs"
								>
									Retry
								</button>
							</div>
						</div>
					{:else if selectedFile}
						<!-- File header -->
						<div class="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-white/10 bg-gray-50/50 dark:bg-white/[0.02]">
							<div class="flex items-center gap-2 text-xs">
								{#if selectedFile.change_type === 'added'}
									<FilePlus size={14} class="text-emerald-400" />
								{:else if selectedFile.change_type === 'removed'}
									<FileMinus size={14} class="text-red-400" />
								{:else}
									<FileEdit size={14} class="text-amber-400" />
								{/if}
								<span class="text-gray-600 dark:text-gray-300 font-mono">{selectedFile.file_path}</span>
								{#if selectedFile.language}
									<span class="px-1.5 py-0.5 bg-gray-100 dark:bg-white/5 rounded text-gray-500 dark:text-gray-500 text-[10px]">{selectedFile.language}</span>
								{/if}
							</div>
							<div class="flex items-center gap-1">
								<button
									onclick={() => sideBySide = false}
									class="p-1.5 rounded transition-colors {!sideBySide ? 'bg-indigo-500/20 text-indigo-300' : 'text-gray-500 hover:text-gray-300 hover:bg-white/5'}"
									title="Inline view"
								>
									<Rows2 size={14} />
								</button>
								<button
									onclick={() => sideBySide = true}
									class="p-1.5 rounded transition-colors {sideBySide ? 'bg-indigo-500/20 text-indigo-300' : 'text-gray-500 hover:text-gray-300 hover:bg-white/5'}"
									title="Side by side"
								>
									<Columns2 size={14} />
								</button>
							</div>
						</div>

						<!-- Monaco diff editor -->
						<div class="flex-1">
							<MonacoDiffEditor
								originalValue={selectedFile.old_content ?? ''}
								modifiedValue={selectedFile.new_content ?? ''}
								filename={selectedFile.file_path}
								language={selectedFile.language ?? ''}
								renderSideBySide={sideBySide}
							/>
						</div>
					{:else}
						<div class="flex items-center justify-center h-full text-gray-500 dark:text-gray-500 text-sm">
							Select a file to view changes
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
