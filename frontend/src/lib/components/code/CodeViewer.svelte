<script lang="ts">
	import MonacoEditor from '$lib/editor/MonacoEditor.svelte';
	import EditorToolbar from '$lib/editor/EditorToolbar.svelte';
	import EditorStatusBar from '$lib/editor/EditorStatusBar.svelte';
	import { detectLanguage } from '$lib/editor/utils/language-detection';
	import type { OSAFile } from '$lib/components/osa/types';
	import { FileCode } from 'lucide-svelte';

	interface Props {
		file: OSAFile | null;
		content: string;
		loading: boolean;
		error: string | null;
		moduleName: string;
		onCopy?: () => void;
	}

	let { file, content, loading, error, moduleName, onCopy }: Props = $props();

	let editorRef = $state<MonacoEditor | undefined>(undefined);
	let cursorLine = $state(1);
	let cursorColumn = $state(1);

	// Derive the display path — prefix with module name for breadcrumb clarity
	let displayPath = $derived(
		file ? (moduleName ? `${moduleName}/${file.path || file.name}` : file.path || file.name) : ''
	);

	let languageId = $derived(file ? detectLanguage(file.path || file.name) : 'plaintext');

	function handleCopy() {
		if (!content) return;
		navigator.clipboard.writeText(content).catch(() => null);
		onCopy?.();
	}
</script>

<div class="flex flex-col h-full min-h-0 bg-gray-950">
	{#if file}
		<!-- Toolbar -->
		<EditorToolbar
			filename={displayPath}
			readonly={true}
			isEditing={false}
			isDirty={false}
			{cursorLine}
			{cursorColumn}
			onCopy={handleCopy}
		/>

		<!-- Editor area -->
		<div class="flex-1 min-h-0 relative">
			{#if loading}
				<div class="absolute inset-0 flex items-center justify-center bg-gray-950 z-10">
					<div class="flex flex-col items-center gap-3">
						<svg class="w-8 h-8 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden="true">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
						</svg>
						<p class="text-sm text-gray-500">Loading file...</p>
					</div>
				</div>
			{:else if error}
				<div class="absolute inset-0 flex items-center justify-center bg-gray-950 z-10">
					<div class="text-center px-8">
						<div class="w-12 h-12 mx-auto mb-3 rounded-full bg-red-500/10 flex items-center justify-center">
							<svg class="w-6 h-6 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
							</svg>
						</div>
						<p class="text-sm text-red-400 font-medium">Failed to load file</p>
						<p class="text-xs text-gray-600 mt-1">{error}</p>
					</div>
				</div>
			{/if}

			<MonacoEditor
				bind:this={editorRef}
				value={content}
				filename={file.path || file.name}
				language={languageId}
				readonly={true}
			/>
		</div>

		<!-- Status bar -->
		<EditorStatusBar
			{languageId}
			isReadonly={true}
			isEditing={false}
		/>
	{:else}
		<!-- Empty state -->
		<div class="flex-1 flex items-center justify-center bg-gray-950">
			<div class="text-center px-8">
				<div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-white/5 flex items-center justify-center">
					<FileCode class="text-gray-600" size={32} aria-hidden="true" />
				</div>
				<h3 class="text-sm font-medium text-gray-400 mb-1">No file selected</h3>
				<p class="text-xs text-gray-600">
					Select a module from the sidebar, then choose a file to view its source.
				</p>
			</div>
		</div>
	{/if}
</div>
