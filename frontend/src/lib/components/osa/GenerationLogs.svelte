<script lang="ts">
	import { Copy, Download, Filter, Search, ChevronDown, ChevronUp } from 'lucide-svelte';
	import type { GenerationLog, LogLevel } from '$lib/stores/generatedAppsStore';
	import type { AgentType } from '$lib/types/agent';

	interface Props {
		logs: GenerationLog[];
		maxHeight?: string;
	}

	let { logs, maxHeight = '300px' }: Props = $props();

	let searchQuery = $state('');
	let filterLevel = $state<LogLevel | 'all'>('all');
	let filterAgent = $state<AgentType | 'all'>('all');
	let autoScroll = $state(true);
	let expanded = $state(true);

	let logContainer = $state<HTMLDivElement | null>(null);

	const LOG_LEVEL_STYLES: Record<LogLevel, { bg: string; text: string; label: string }> = {
		info: { bg: 'bg-blue-50 dark:bg-blue-900/20', text: 'text-blue-600 dark:text-blue-400', label: 'INFO' },
		warn: { bg: 'bg-yellow-50 dark:bg-yellow-900/20', text: 'text-yellow-600 dark:text-yellow-400', label: 'WARN' },
		error: { bg: 'bg-red-50 dark:bg-red-900/20', text: 'text-red-600 dark:text-red-400', label: 'ERROR' },
		debug: { bg: 'bg-gray-50 dark:bg-gray-800', text: 'text-gray-500 dark:text-gray-400', label: 'DEBUG' }
	};

	const AGENT_COLORS: Record<AgentType, string> = {
		orchestrator: 'text-indigo-500',
		frontend: 'text-blue-500',
		backend: 'text-green-500',
		database: 'text-purple-500',
		test: 'text-orange-500'
	};

	let filteredLogs = $derived(() => {
		return logs.filter(log => {
			// Level filter
			if (filterLevel !== 'all' && log.level !== filterLevel) return false;

			// Agent filter
			if (filterAgent !== 'all' && log.agent_type !== filterAgent) return false;

			// Search filter
			if (searchQuery.trim()) {
				const query = searchQuery.toLowerCase();
				if (!log.message.toLowerCase().includes(query)) return false;
			}

			return true;
		});
	});

	// Auto-scroll to bottom when new logs arrive
	$effect(() => {
		if (autoScroll && logContainer && logs.length > 0) {
			logContainer.scrollTop = logContainer.scrollHeight;
		}
	});

	function formatTime(timestamp: string): string {
		return new Date(timestamp).toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			fractionalSecondDigits: 3
		});
	}

	async function copyAllLogs() {
		const text = filteredLogs()
			.map(log => `[${formatTime(log.timestamp)}] [${log.level.toUpperCase()}] ${log.agent_type ? `[${log.agent_type}] ` : ''}${log.message}`)
			.join('\n');

		await navigator.clipboard.writeText(text);
	}

	function downloadLogs() {
		const text = filteredLogs()
			.map(log => JSON.stringify(log))
			.join('\n');

		const blob = new Blob([text], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `generation-logs-${Date.now()}.jsonl`;
		a.click();
		URL.revokeObjectURL(url);
	}

	function handleScroll() {
		if (!logContainer) return;
		const isAtBottom = logContainer.scrollHeight - logContainer.scrollTop <= logContainer.clientHeight + 50;
		autoScroll = isAtBottom;
	}
</script>

<div class="generation-logs border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
	<!-- Header -->
	<div class="flex items-center justify-between px-3 py-2 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
		<button
			onclick={() => expanded = !expanded}
			class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
		>
			{#if expanded}
				<ChevronUp class="w-4 h-4" />
			{:else}
				<ChevronDown class="w-4 h-4" />
			{/if}
			Logs ({logs.length})
		</button>

		{#if expanded}
			<div class="flex items-center gap-2">
				<button
					onclick={copyAllLogs}
					class="btn-pill btn-pill-ghost btn-pill-icon"
					title="Copy all logs"
				>
					<Copy class="w-4 h-4" />
				</button>
				<button
					onclick={downloadLogs}
					class="btn-pill btn-pill-ghost btn-pill-icon"
					title="Download logs"
				>
					<Download class="w-4 h-4" />
				</button>
			</div>
		{/if}
	</div>

	{#if expanded}
		<!-- Filters -->
		<div class="flex flex-wrap items-center gap-2 px-3 py-2 bg-gray-50/50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700">
			<!-- Search -->
			<div class="relative flex-1 min-w-[150px]">
				<Search class="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Search logs..."
					class="w-full pl-8 pr-3 py-1.5 text-sm border rounded bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-600 focus:ring-1 focus:ring-blue-500"
				/>
			</div>

			<!-- Level Filter -->
			<select
				bind:value={filterLevel}
				class="text-sm px-2 py-1.5 border rounded bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-600"
			>
				<option value="all">All Levels</option>
				<option value="info">Info</option>
				<option value="warn">Warning</option>
				<option value="error">Error</option>
				<option value="debug">Debug</option>
			</select>

			<!-- Agent Filter -->
			<select
				bind:value={filterAgent}
				class="text-sm px-2 py-1.5 border rounded bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-600"
			>
				<option value="all">All Agents</option>
				<option value="frontend">Frontend</option>
				<option value="backend">Backend</option>
				<option value="database">Database</option>
				<option value="test">Test</option>
			</select>
		</div>

		<!-- Log entries -->
		<div
			bind:this={logContainer}
			onscroll={handleScroll}
			class="overflow-y-auto font-mono text-xs"
			style="max-height: {maxHeight}"
		>
			{#if filteredLogs().length === 0}
				<div class="p-4 text-center text-gray-500 dark:text-gray-400">
					{logs.length === 0 ? 'No logs yet' : 'No logs match your filters'}
				</div>
			{:else}
				{#each filteredLogs() as log (log.id)}
					{@const styles = LOG_LEVEL_STYLES[log.level]}
					<div class="flex items-start gap-2 px-3 py-1.5 border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50 {styles.bg}">
						<span class="text-gray-400 flex-shrink-0 w-20">{formatTime(log.timestamp)}</span>
						<span class="font-semibold flex-shrink-0 w-12 {styles.text}">{styles.label}</span>
						{#if log.agent_type}
							<span class="flex-shrink-0 w-16 {AGENT_COLORS[log.agent_type]}">[{log.agent_type}]</span>
						{:else}
							<span class="flex-shrink-0 w-16"></span>
						{/if}
						<span class="text-gray-700 dark:text-gray-300 break-all">{log.message}</span>
					</div>
				{/each}
			{/if}
		</div>

		<!-- Auto-scroll indicator -->
		{#if !autoScroll && logs.length > 10}
			<button
				onclick={() => { autoScroll = true; if (logContainer) logContainer.scrollTop = logContainer.scrollHeight; }}
				class="w-full py-1 text-xs text-center text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 hover:bg-blue-100 dark:hover:bg-blue-900/30"
			>
				Click to resume auto-scroll
			</button>
		{/if}
	{/if}
</div>
