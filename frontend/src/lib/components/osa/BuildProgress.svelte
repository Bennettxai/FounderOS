<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { CheckCircle, AlertCircle, Loader2, Terminal, Clock, ExternalLink, RefreshCw, ChevronDown, ChevronRight, Zap, Package, TestTube, Rocket } from 'lucide-svelte';

	// Types
	export interface BuildResult {
		buildId: string;
		status: 'completed' | 'failed';
		deploymentUrl?: string;
		duration?: number;
		error?: string;
	}

	export interface LogEntry {
		timestamp: Date;
		message: string;
		phase: string;
		level: 'info' | 'warn' | 'error' | 'success';
	}

	interface BuildEvent {
		progress: number;
		phase: string;
		log?: string;
		status: 'in_progress' | 'completed' | 'failed';
		error?: string;
		deploymentUrl?: string;
		estimatedTimeRemaining?: number;
		logLevel?: 'info' | 'warn' | 'error' | 'success';
	}

	interface PhaseInfo {
		name: string;
		icon: typeof Zap;
		logs: LogEntry[];
		collapsed: boolean;
		startTime?: Date;
		endTime?: Date;
	}

	// Props
	interface Props {
		buildId: string;
		onComplete?: (result: BuildResult) => void;
		onError?: (error: Error) => void;
	}

	let { buildId, onComplete = () => {}, onError = () => {} }: Props = $props();

	// State
	let progress = $state(0);
	let phase = $state('Initializing');
	let logs = $state<LogEntry[]>([]);
	let status = $state<'connecting' | 'building' | 'success' | 'error'>('connecting');
	let deploymentUrl = $state('');
	let errorMessage = $state<string | null>(null);
	let estimatedTimeRemaining = $state<number | null>(null);
	let connected = $state(false);
	let reconnectAttempts = $state(0);
	let startTime = $state<Date | null>(null);

	// Phase tracking
	let phases = $state<PhaseInfo[]>([
		{ name: 'Planning', icon: Zap, logs: [], collapsed: false },
		{ name: 'Building', icon: Package, logs: [], collapsed: true },
		{ name: 'Testing', icon: TestTube, logs: [], collapsed: true },
		{ name: 'Deploying', icon: Rocket, logs: [], collapsed: true }
	]);

	// References
	let eventSource: EventSource | null = null;
	let logsContainer: HTMLDivElement | null = null;
	let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

	// Constants
	const MAX_RECONNECT_ATTEMPTS = 5;
	const RECONNECT_DELAY = 2000;
	const MAX_LOGS = 500;

	// Derived values
	let currentPhaseIndex = $derived(phases.findIndex((p) => p.name === phase));
	let elapsedTime = $derived.by(() => {
		if (!startTime) return '0:00';
		const elapsed = Math.floor((Date.now() - startTime.getTime()) / 1000);
		const minutes = Math.floor(elapsed / 60);
		const seconds = elapsed % 60;
		return `${minutes}:${seconds.toString().padStart(2, '0')}`;
	});

	let formattedETA = $derived.by(() => {
		if (estimatedTimeRemaining === null || estimatedTimeRemaining <= 0) return null;
		const minutes = Math.floor(estimatedTimeRemaining / 60);
		const seconds = estimatedTimeRemaining % 60;
		if (minutes > 0) {
			return `~${minutes}m ${seconds}s remaining`;
		}
		return `~${seconds}s remaining`;
	});

	// Connect to SSE endpoint
	function connect() {
		try {
			const url = `/api/osa/builds/${buildId}/stream`;
			eventSource = new EventSource(url);
			startTime = new Date();

			eventSource.onopen = () => {
				connected = true;
				reconnectAttempts = 0;
				status = 'building';
				if (import.meta.env.DEV) console.log('[BuildProgress] SSE connection opened');
			};

			eventSource.onmessage = (event) => {
				try {
					const data: BuildEvent = JSON.parse(event.data);
					handleBuildEvent(data);
				} catch (err) {
					console.error('[BuildProgress] Failed to parse event:', err);
				}
			};

			eventSource.onerror = (err) => {
				console.error('[BuildProgress] SSE error:', err);
				connected = false;

				if (eventSource) {
					eventSource.close();
					eventSource = null;
				}

				if (status === 'building' && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
					scheduleReconnect();
				} else if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
					errorMessage = 'Connection lost. Maximum reconnection attempts reached.';
					status = 'error';
					onError(new Error(errorMessage));
				}
			};
		} catch (err) {
			console.error('[BuildProgress] Failed to connect:', err);
			errorMessage = 'Failed to connect to build stream';
			status = 'error';
			onError(new Error(errorMessage));
		}
	}

	// Schedule reconnection attempt
	function scheduleReconnect() {
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
		}

		reconnectAttempts++;
		if (import.meta.env.DEV) {
  		console.log(
  			`[BuildProgress] Reconnecting in ${RECONNECT_DELAY}ms (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`
  		);
		}

		reconnectTimeout = setTimeout(() => {
			if (import.meta.env.DEV) console.log('[BuildProgress] Attempting reconnection...');
			connect();
		}, RECONNECT_DELAY);
	}

	// Handle build event
	function handleBuildEvent(event: BuildEvent) {
		// Update progress
		if (event.progress !== undefined) {
			progress = Math.min(100, Math.max(0, event.progress));
		}

		// Update estimated time
		if (event.estimatedTimeRemaining !== undefined) {
			estimatedTimeRemaining = event.estimatedTimeRemaining;
		}

		// Update phase
		if (event.phase && event.phase !== phase) {
			const prevPhaseIndex = phases.findIndex((p) => p.name === phase);
			const newPhaseIndex = phases.findIndex((p) => p.name === event.phase);

			// Mark previous phase as complete and collapse it
			if (prevPhaseIndex >= 0) {
				phases[prevPhaseIndex].endTime = new Date();
				phases[prevPhaseIndex].collapsed = true;
			}

			// Start new phase and expand it
			if (newPhaseIndex >= 0) {
				phases[newPhaseIndex].startTime = new Date();
				phases[newPhaseIndex].collapsed = false;
			}

			phase = event.phase;
		}

		// Add log entry
		if (event.log) {
			const logEntry: LogEntry = {
				timestamp: new Date(),
				message: event.log,
				phase: event.phase || phase,
				level: event.logLevel || 'info'
			};

			// Add to main logs
			if (logs.length >= MAX_LOGS) {
				logs = [...logs.slice(-(MAX_LOGS - 1)), logEntry];
			} else {
				logs = [...logs, logEntry];
			}

			// Add to phase logs
			const phaseIndex = phases.findIndex((p) => p.name === logEntry.phase);
			if (phaseIndex >= 0) {
				phases[phaseIndex].logs = [...phases[phaseIndex].logs, logEntry];
			}

			// Auto-scroll to bottom
			setTimeout(() => {
				if (logsContainer) {
					logsContainer.scrollTop = logsContainer.scrollHeight;
				}
			}, 0);
		}

		// Update status
		if (event.status) {
			if (event.status === 'completed') {
				progress = 100;
				status = 'success';
				deploymentUrl = event.deploymentUrl || '';

				if (eventSource) {
					eventSource.close();
					eventSource = null;
				}

				const result: BuildResult = {
					buildId,
					status: 'completed',
					deploymentUrl: event.deploymentUrl,
					duration: startTime ? Math.floor((Date.now() - startTime.getTime()) / 1000) : undefined
				};
				onComplete(result);
			}

			if (event.status === 'failed') {
				errorMessage = event.error || 'Build failed';
				status = 'error';

				if (eventSource) {
					eventSource.close();
					eventSource = null;
				}

				const result: BuildResult = {
					buildId,
					status: 'failed',
					error: event.error
				};
				onError(new Error(errorMessage));
			}
		}
	}

	// Retry build
	function handleRetry() {
		// Reset state
		progress = 0;
		phase = 'Initializing';
		logs = [];
		status = 'connecting';
		errorMessage = null;
		estimatedTimeRemaining = null;
		reconnectAttempts = 0;
		startTime = null;

		// Reset phases
		phases = phases.map((p) => ({
			...p,
			logs: [],
			collapsed: p.name !== 'Planning',
			startTime: undefined,
			endTime: undefined
		}));

		// Reconnect
		connect();
	}

	// Toggle phase collapse
	function togglePhase(index: number) {
		phases[index].collapsed = !phases[index].collapsed;
	}

	// Get phase status
	function getPhaseStatus(index: number): 'pending' | 'active' | 'completed' {
		if (index < currentPhaseIndex) return 'completed';
		if (index === currentPhaseIndex) return 'active';
		return 'pending';
	}

	// Highlight log line syntax
	function highlightLog(message: string): string {
		// Highlight commands (starting with $, >, or npm/yarn/pnpm)
		if (/^[$>]\s/.test(message) || /^(npm|yarn|pnpm|go|docker)\s/.test(message)) {
			return `<span class="text-cyan-400">${escapeHtml(message)}</span>`;
		}
		// Highlight success messages
		if (/success|complete|done|passed/i.test(message)) {
			return `<span class="text-green-400">${escapeHtml(message)}</span>`;
		}
		// Highlight warnings
		if (/warn|warning/i.test(message)) {
			return `<span class="text-yellow-400">${escapeHtml(message)}</span>`;
		}
		// Highlight errors
		if (/error|fail|failed/i.test(message)) {
			return `<span class="text-red-400">${escapeHtml(message)}</span>`;
		}
		return escapeHtml(message);
	}

	function escapeHtml(text: string): string {
		const div = document.createElement('div');
		div.textContent = text;
		return div.innerHTML;
	}

	// Lifecycle
	onMount(() => {
		connect();
	});

	onDestroy(() => {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
			reconnectTimeout = null;
		}
	});
</script>

<div class="build-progress bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
	<!-- Header -->
	<div class="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-gray-900/50">
		<div class="flex items-center gap-3">
			{#if status === 'connecting'}
				<div class="w-8 h-8 rounded-full bg-blue-500/20 flex items-center justify-center">
					<Loader2 class="w-5 h-5 text-blue-400 animate-spin" />
				</div>
				<div>
					<h3 class="text-white font-semibold">Connecting...</h3>
					<p class="text-gray-400 text-sm">Establishing build stream</p>
				</div>
			{:else if status === 'building'}
				<div class="w-8 h-8 rounded-full bg-blue-500/20 flex items-center justify-center animate-pulse">
					<Terminal class="w-5 h-5 text-blue-400" />
				</div>
				<div>
					<h3 class="text-white font-semibold">{phase}</h3>
					<p class="text-gray-400 text-sm">
						Build in progress
						{#if formattedETA}
							<span class="text-blue-400 ml-2">{formattedETA}</span>
						{/if}
					</p>
				</div>
			{:else if status === 'success'}
				<div class="w-8 h-8 rounded-full bg-green-500/20 flex items-center justify-center">
					<CheckCircle class="w-5 h-5 text-green-400" />
				</div>
				<div>
					<h3 class="text-white font-semibold">Build Complete</h3>
					<p class="text-gray-400 text-sm">Deployed successfully</p>
				</div>
			{:else if status === 'error'}
				<div class="w-8 h-8 rounded-full bg-red-500/20 flex items-center justify-center">
					<AlertCircle class="w-5 h-5 text-red-400" />
				</div>
				<div>
					<h3 class="text-white font-semibold">Build Failed</h3>
					<p class="text-red-400 text-sm">{errorMessage}</p>
				</div>
			{/if}
		</div>

		<div class="flex items-center gap-4">
			{#if !connected && status === 'building'}
				<span class="text-yellow-400 text-sm flex items-center gap-2">
					<Loader2 class="w-4 h-4 animate-spin" />
					Reconnecting...
				</span>
			{/if}
			<div class="flex items-center gap-2 text-gray-500 text-sm">
				<Clock class="w-4 h-4" />
				<span>{elapsedTime}</span>
			</div>
			<span class="text-gray-600 font-mono text-xs bg-gray-800 px-2 py-1 rounded">
				{buildId.substring(0, 8)}
			</span>
		</div>
	</div>

	<!-- Progress Bar -->
	<div class="px-6 py-4 bg-gray-900/30">
		<div class="flex items-center gap-4">
			<div class="flex-1 h-2 bg-gray-800 rounded-full overflow-hidden">
				<div
					class="h-full rounded-full transition-all duration-500 ease-out relative overflow-hidden"
					class:bg-blue-500={status === 'building' || status === 'connecting'}
					class:bg-green-500={status === 'success'}
					class:bg-red-500={status === 'error'}
					style="width: {progress}%"
				>
					{#if status === 'building'}
						<div
							class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"
						></div>
					{/if}
				</div>
			</div>
			<span class="text-white font-semibold text-sm min-w-[3rem] text-right">{progress}%</span>
		</div>
	</div>

	<!-- Phase Indicators -->
	<div class="px-6 py-4 border-t border-gray-800/50">
		<div class="flex items-center justify-between">
			{#each phases as phaseInfo, index}
				{@const phaseStatus = getPhaseStatus(index)}
				<div class="flex flex-col items-center gap-2 flex-1 relative">
					<!-- Connection line -->
					{#if index < phases.length - 1}
						<div
							class="absolute top-3 left-1/2 w-full h-0.5 -z-10"
							class:bg-gray-700={phaseStatus === 'pending'}
							class:bg-blue-500={phaseStatus === 'active'}
							class:bg-green-500={phaseStatus === 'completed'}
						></div>
					{/if}

					<!-- Phase dot -->
					<div
						class="w-6 h-6 rounded-full flex items-center justify-center transition-all duration-300"
						class:bg-gray-700={phaseStatus === 'pending'}
						class:bg-blue-500={phaseStatus === 'active'}
						class:bg-green-500={phaseStatus === 'completed'}
						class:ring-4={phaseStatus === 'active'}
						class:ring-blue-500={phaseStatus === 'active'}
						class:ring-opacity-30={phaseStatus === 'active'}
						class:animate-pulse={phaseStatus === 'active'}
					>
						{#if phaseStatus === 'completed'}
							<CheckCircle class="w-4 h-4 text-white" />
						{:else if phaseStatus === 'active'}
							<svelte:component this={phaseInfo.icon} class="w-3 h-3 text-white" />
						{:else}
							<svelte:component this={phaseInfo.icon} class="w-3 h-3 text-gray-500" />
						{/if}
					</div>

					<!-- Phase name -->
					<span
						class="text-xs font-medium transition-colors"
						class:text-gray-500={phaseStatus === 'pending'}
						class:text-blue-400={phaseStatus === 'active'}
						class:text-green-400={phaseStatus === 'completed'}
					>
						{phaseInfo.name}
					</span>
				</div>
			{/each}
		</div>
	</div>

	<!-- Collapsible Phase Logs -->
	<div class="border-t border-gray-800">
		{#each phases as phaseInfo, index}
			{@const phaseStatus = getPhaseStatus(index)}
			{#if phaseInfo.logs.length > 0 || phaseStatus === 'active'}
				<div class="border-b border-gray-800/50 last:border-b-0">
					<button
						class="btn-pill btn-pill-ghost w-full flex items-center justify-between"
						onclick={() => togglePhase(index)}
					>
						<div class="flex items-center gap-3">
							{#if phaseInfo.collapsed}
								<ChevronRight class="w-4 h-4 text-gray-500" />
							{:else}
								<ChevronDown class="w-4 h-4 text-gray-400" />
							{/if}
							<svelte:component
								this={phaseInfo.icon}
								class="w-4 h-4 {phaseStatus === 'pending' ? 'text-gray-500' : phaseStatus === 'active' ? 'text-blue-400' : 'text-green-400'}"
							/>
							<span
								class="text-sm font-medium"
								class:text-gray-500={phaseStatus === 'pending'}
								class:text-white={phaseStatus === 'active'}
								class:text-gray-300={phaseStatus === 'completed'}
							>
								{phaseInfo.name}
							</span>
							<span class="text-gray-600 text-xs">({phaseInfo.logs.length} logs)</span>
						</div>
						{#if phaseStatus === 'active'}
							<span class="text-blue-400 text-xs flex items-center gap-1">
								<Loader2 class="w-3 h-3 animate-spin" />
								Running
							</span>
						{:else if phaseStatus === 'completed'}
							<span class="text-green-400 text-xs">Completed</span>
						{/if}
					</button>

					{#if !phaseInfo.collapsed}
						<div class="px-6 pb-3">
							<div
								class="bg-gray-950 rounded-lg p-3 max-h-40 overflow-y-auto font-mono text-xs text-gray-300 space-y-1"
							>
								{#each phaseInfo.logs as log, logIndex}
									<div class="flex gap-2">
										<span class="text-gray-600 select-none"
											>{String(logIndex + 1).padStart(3, '0')}</span
										>
										<!-- eslint-disable-next-line svelte/no-at-html-tags -->
										<span>{@html highlightLog(log.message)}</span>
									</div>
								{/each}
								{#if phaseStatus === 'active'}
									<div class="flex gap-2 items-center">
										<span class="text-gray-600 select-none"
											>{String(phaseInfo.logs.length + 1).padStart(3, '0')}</span
										>
										<span class="text-gray-500 animate-pulse">|</span>
									</div>
								{/if}
							</div>
						</div>
					{/if}
				</div>
			{/if}
		{/each}
	</div>

	<!-- Main Log Terminal -->
	<div class="border-t border-gray-800">
		<div class="flex items-center justify-between px-4 py-2 bg-gray-950 border-b border-gray-800">
			<div class="flex items-center gap-2">
				<div class="flex gap-1.5">
					<span class="w-3 h-3 rounded-full bg-red-500"></span>
					<span class="w-3 h-3 rounded-full bg-yellow-500"></span>
					<span class="w-3 h-3 rounded-full bg-green-500"></span>
				</div>
				<span class="text-gray-400 text-xs ml-2">Build Output</span>
			</div>
			<span class="text-gray-600 text-xs">{logs.length} lines</span>
		</div>

		<div
			bind:this={logsContainer}
			class="h-64 overflow-y-auto bg-gray-950 p-4 font-mono text-sm text-gray-300 scroll-smooth"
		>
			{#if logs.length === 0}
				<div class="h-full flex items-center justify-center text-gray-600">
					<div class="flex items-center gap-2">
						<span class="animate-pulse">|</span>
						Waiting for build logs...
					</div>
				</div>
			{:else}
				{#each logs as log, index (index)}
					<div class="flex gap-3 hover:bg-gray-900/50 px-2 -mx-2 rounded">
						<span class="text-gray-600 select-none shrink-0"
							>{String(index + 1).padStart(4, '0')}</span
						>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						<span class="break-all">{@html highlightLog(log.message)}</span>
					</div>
				{/each}
			{/if}
		</div>
	</div>

	<!-- Status Footer -->
	{#if status === 'success' && deploymentUrl}
		<div
			class="px-6 py-4 bg-gradient-to-r from-green-500/10 to-emerald-500/10 border-t border-green-500/20"
		>
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<CheckCircle class="w-5 h-5 text-green-400" />
					<div>
						<p class="text-green-400 font-medium">Deployment Successful</p>
						<p class="text-gray-400 text-sm">Your application is now live</p>
					</div>
				</div>
				<a
					href={deploymentUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="btn-pill btn-pill-success flex items-center gap-2"
				>
					<ExternalLink class="w-4 h-4" />
					Open App
				</a>
			</div>
		</div>
	{:else if status === 'error'}
		<div
			class="px-6 py-4 bg-gradient-to-r from-red-500/10 to-rose-500/10 border-t border-red-500/20"
		>
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<AlertCircle class="w-5 h-5 text-red-400" />
					<div>
						<p class="text-red-400 font-medium">Build Failed</p>
						<p class="text-gray-400 text-sm">{errorMessage}</p>
					</div>
				</div>
				<button
					onclick={handleRetry}
					class="btn-pill btn-pill-secondary flex items-center gap-2"
				>
					<RefreshCw class="w-4 h-4" />
					Retry Build
				</button>
			</div>
		</div>
	{/if}
</div>

<style>
	@keyframes shimmer {
		0% {
			transform: translateX(-100%);
		}
		100% {
			transform: translateX(200%);
		}
	}

	.animate-shimmer {
		animation: shimmer 2s infinite;
	}

	/* Custom scrollbar for terminal */
	.scroll-smooth::-webkit-scrollbar {
		width: 8px;
	}

	.scroll-smooth::-webkit-scrollbar-track {
		background: transparent;
	}

	.scroll-smooth::-webkit-scrollbar-thumb {
		background: rgba(255, 255, 255, 0.1);
		border-radius: 4px;
	}

	.scroll-smooth::-webkit-scrollbar-thumb:hover {
		background: rgba(255, 255, 255, 0.2);
	}
</style>
