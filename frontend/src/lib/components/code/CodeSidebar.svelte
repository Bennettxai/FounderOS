<script lang="ts">
	import { ChevronRight, ChevronDown, Search, Box } from 'lucide-svelte';
	import { getLanguageColor } from '$lib/editor/utils/language-detection';
	import type { OSAFile } from '$lib/components/osa/types';

	interface ModuleEntry {
		id: string;
		title: string;
		color: string;
		icon: string;
	}

	interface ModuleFiles {
		[moduleId: string]: OSAFile[];
	}

	interface Props {
		modules: ModuleEntry[];
		moduleFiles: ModuleFiles;
		selectedModuleId: string | null;
		selectedFile: OSAFile | null;
		loadingModuleId: string | null;
		onModuleSelect: (moduleId: string) => void;
		onFileSelect: (file: OSAFile) => void;
	}

	let {
		modules,
		moduleFiles,
		selectedModuleId,
		selectedFile,
		loadingModuleId,
		onModuleSelect,
		onFileSelect,
	}: Props = $props();

	let searchQuery = $state('');
	let expandedModules = $state<Set<string>>(new Set());

	function toggleModule(moduleId: string) {
		const next = new Set(expandedModules);
		if (next.has(moduleId)) {
			next.delete(moduleId);
		} else {
			next.add(moduleId);
			onModuleSelect(moduleId);
		}
		expandedModules = next;
	}

	function handleFileClick(file: OSAFile) {
		onFileSelect(file);
	}

	// Filter modules by search query
	let filteredModules = $derived(
		searchQuery.trim()
			? modules.filter((m) => m.title.toLowerCase().includes(searchQuery.toLowerCase()))
			: modules
	);

	// Filter files within each module by search query
	function filteredFiles(moduleId: string): OSAFile[] {
		const files = moduleFiles[moduleId] ?? [];
		if (!searchQuery.trim()) return files;
		const q = searchQuery.toLowerCase();
		return files.filter((f) => f.name.toLowerCase().includes(q) || f.path.toLowerCase().includes(q));
	}

	function getIconColor(languageName: string): string {
		return getLanguageColor(languageName);
	}
</script>

<aside class="flex flex-col h-full bg-gray-950 border-r border-white/8 min-w-0 w-64 flex-shrink-0">
	<!-- Header -->
	<div class="px-3 py-3 border-b border-white/8 flex-shrink-0">
		<h2 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Code Browser</h2>
		<!-- Search -->
		<div class="relative">
			<Search class="absolute left-2 top-1/2 -translate-y-1/2 text-gray-500" size={13} />
			<input
				type="text"
				placeholder="Search modules & files..."
				bind:value={searchQuery}
				aria-label="Search modules and files"
				class="w-full pl-7 pr-3 py-1.5 text-xs bg-white/5 border border-white/8 rounded-md text-gray-300 placeholder-gray-600 focus:outline-none focus:ring-1 focus:ring-blue-500/50 focus:border-blue-500/50 transition-colors"
			/>
		</div>
	</div>

	<!-- Module list -->
	<nav class="flex-1 overflow-y-auto py-1" aria-label="Modules">
		{#if filteredModules.length === 0}
			<p class="px-3 py-4 text-xs text-gray-600 text-center">No modules match your search</p>
		{:else}
			{#each filteredModules as module (module.id)}
				{@const isExpanded = expandedModules.has(module.id)}
				{@const isSelected = selectedModuleId === module.id}
				{@const isLoading = loadingModuleId === module.id}
				{@const files = filteredFiles(module.id)}

				<!-- Module row -->
				<button
					class="w-full flex items-center gap-2 px-3 py-2 text-left text-sm transition-colors
						{isSelected
							? 'bg-blue-600/15 text-blue-300'
							: 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
					onclick={() => toggleModule(module.id)}
					aria-expanded={isExpanded}
					aria-label="Toggle {module.title} module"
				>
					<!-- Chevron -->
					<span class="flex-shrink-0 text-gray-600 w-3.5">
						{#if isExpanded}
							<ChevronDown size={14} />
						{:else}
							<ChevronRight size={14} />
						{/if}
					</span>

					<!-- Color dot -->
					<span
						class="w-2 h-2 rounded-full flex-shrink-0"
						style="background-color: {module.color}"
						aria-hidden="true"
					></span>

					<!-- Title -->
					<span class="flex-1 font-medium text-xs truncate">{module.title}</span>

					<!-- Loading spinner -->
					{#if isLoading}
						<svg class="w-3 h-3 animate-spin text-gray-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" aria-hidden="true">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
						</svg>
					{:else if moduleFiles[module.id] !== undefined}
						<span class="text-xs text-gray-600 flex-shrink-0">{files.length}</span>
					{/if}
				</button>

				<!-- File list -->
				{#if isExpanded}
					{#if isLoading}
						<div class="px-6 py-2">
							<p class="text-xs text-gray-600 italic">Loading files...</p>
						</div>
					{:else if files.length === 0 && moduleFiles[module.id] !== undefined}
						<div class="px-6 py-2">
							<p class="text-xs text-gray-600 italic">No source files available</p>
						</div>
					{:else}
						{#each files as file (file.id)}
							{@const isFileSelected = selectedFile?.id === file.id}
							<button
								class="w-full flex items-center gap-2 pl-8 pr-3 py-1.5 text-left text-xs transition-colors
									{isFileSelected
										? 'bg-blue-600/20 text-blue-300'
										: 'text-gray-500 hover:text-gray-300 hover:bg-white/4'}"
								onclick={() => handleFileClick(file)}
								aria-label="Open {file.name}"
								aria-current={isFileSelected ? 'true' : undefined}
							>
								<!-- Language color indicator -->
								<span
									class="w-1.5 h-1.5 rounded-full flex-shrink-0 opacity-70"
									style="background-color: {getIconColor(file.language ?? '')}"
									aria-hidden="true"
								></span>
								<span class="truncate">{file.name}</span>
							</button>
						{/each}
					{/if}
				{/if}
			{/each}
		{/if}
	</nav>
</aside>
