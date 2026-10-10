<!--
	DiffViewer.svelte
	Renders file-by-file unified diffs from Pedro's sandbox edit API.
	Each DiffEntry has { filename, lines_added, lines_removed, diff }.
	The `diff` field contains unified diff text with +/- prefixed lines.
-->
<script lang="ts">
	import type { DiffEntry } from '$lib/api/sandbox-preview';

	interface Props {
		diffs: DiffEntry[];
		moduleName?: string;
	}

	let { diffs, moduleName }: Props = $props();

	let expandedFiles = $state<Set<string>>(new Set());

	// Auto-expand all files if 3 or fewer
	$effect(() => {
		if (diffs.length <= 3) {
			expandedFiles = new Set(diffs.map((d) => d.filename));
		}
	});

	function toggleFile(filename: string) {
		const next = new Set(expandedFiles);
		if (next.has(filename)) {
			next.delete(filename);
		} else {
			next.add(filename);
		}
		expandedFiles = next;
	}

	function expandAll() {
		expandedFiles = new Set(diffs.map((d) => d.filename));
	}

	function collapseAll() {
		expandedFiles = new Set();
	}

	/** Parse unified diff text into colored lines */
	function parseDiffLines(diff: string): { type: 'added' | 'removed' | 'context' | 'header'; text: string }[] {
		if (!diff) return [];
		return diff.split('\n').map((line) => {
			if (line.startsWith('+++') || line.startsWith('---') || line.startsWith('@@')) {
				return { type: 'header' as const, text: line };
			} else if (line.startsWith('+')) {
				return { type: 'added' as const, text: line };
			} else if (line.startsWith('-')) {
				return { type: 'removed' as const, text: line };
			} else {
				return { type: 'context' as const, text: line };
			}
		});
	}

	/** Determine file change type from diff stats */
	function getChangeType(entry: DiffEntry): 'added' | 'modified' | 'deleted' {
		if (entry.lines_removed === 0 && entry.lines_added > 0) return 'added';
		if (entry.lines_added === 0 && entry.lines_removed > 0) return 'deleted';
		return 'modified';
	}

	const changeBadgeClass: Record<string, string> = {
		added: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		modified: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
		deleted: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
	};

	const lineClass: Record<string, string> = {
		added: 'bg-green-50 text-green-900 dark:bg-green-950/30 dark:text-green-300',
		removed: 'bg-red-50 text-red-900 dark:bg-red-950/30 dark:text-red-300',
		context: 'text-gray-700 dark:text-gray-400',
		header: 'bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-400 font-semibold'
	};

	let totalAdded = $derived(diffs.reduce((sum, d) => sum + d.lines_added, 0));
	let totalRemoved = $derived(diffs.reduce((sum, d) => sum + d.lines_removed, 0));
</script>

<div class="diff-viewer flex flex-col gap-3">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-2">
			{#if moduleName}
				<span class="text-sm font-medium text-gray-900 dark:text-white">{moduleName}</span>
				<span class="text-xs text-gray-400">|</span>
			{/if}
			<span class="text-xs text-gray-500 dark:text-gray-400">
				{diffs.length} file{diffs.length !== 1 ? 's' : ''} changed
			</span>
			<span class="text-xs text-green-600 dark:text-green-400">+{totalAdded}</span>
			<span class="text-xs text-red-600 dark:text-red-400">-{totalRemoved}</span>
		</div>
		<div class="flex items-center gap-1">
			<button
				onclick={expandAll}
				class="btn-pill btn-pill-ghost btn-pill-xs"
			>
				Expand all
			</button>
			<button
				onclick={collapseAll}
				class="btn-pill btn-pill-ghost btn-pill-xs"
			>
				Collapse all
			</button>
		</div>
	</div>

	<!-- File diffs -->
	{#each diffs as entry (entry.filename)}
		{@const changeType = getChangeType(entry)}
		{@const isExpanded = expandedFiles.has(entry.filename)}
		{@const lines = parseDiffLines(entry.diff)}

		<div class="rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
			<!-- File header -->
			<button
				class="btn-pill btn-pill-soft btn-pill-sm flex w-full items-center gap-2 text-left"
				onclick={() => toggleFile(entry.filename)}
				aria-expanded={isExpanded}
				aria-label="View changes for {entry.filename}"
			>
				<svg
					class="h-3 w-3 flex-shrink-0 text-gray-400 transition-transform"
					class:rotate-90={isExpanded}
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
				>
					<polyline points="9 18 15 12 9 6" />
				</svg>

				<span class="font-mono text-xs text-gray-900 dark:text-gray-100 truncate">
					{entry.filename}
				</span>

				<span class="ml-auto flex items-center gap-2 flex-shrink-0">
					<span class="text-xs text-green-600 dark:text-green-400">+{entry.lines_added}</span>
					<span class="text-xs text-red-600 dark:text-red-400">-{entry.lines_removed}</span>
					<span class="rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase {changeBadgeClass[changeType]}">
						{changeType}
					</span>
				</span>
			</button>

			<!-- Diff content -->
			{#if isExpanded}
				<div class="max-h-[400px] overflow-y-auto border-t border-gray-200 dark:border-gray-700">
					{#if lines.length === 0}
						<div class="px-4 py-3 text-xs text-gray-400 italic">No diff content</div>
					{:else}
						<pre class="text-xs leading-relaxed"><code>{#each lines as line, i}<div class="px-3 py-0 {lineClass[line.type]}" aria-label="{line.type === 'added' ? 'Added' : line.type === 'removed' ? 'Removed' : ''} line">{line.text}</div>{/each}</code></pre>
					{/if}
				</div>
			{/if}
		</div>
	{/each}

	{#if diffs.length === 0}
		<div class="rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-8 text-center">
			<p class="text-sm text-gray-500 dark:text-gray-400">No changes to preview</p>
		</div>
	{/if}
</div>
