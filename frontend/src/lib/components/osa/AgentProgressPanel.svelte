<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader2, XCircle, Sparkles, Rocket, Download, StopCircle, Code2 } from 'lucide-svelte';
	import { goto } from '$app/navigation';
	import { createAgentProgressClient } from '$lib/utils/sse';
	import type { AgentCard, ProgressEvent, AgentType } from '$lib/types/agent';
	import { AGENT_CONFIGS, getRandomMessage } from '$lib/types/agent';
	import type { SSEClient } from '$lib/utils/sse';
	import { cancelGeneration } from '$lib/api/osa/osa';
	import { deploySandbox } from '$lib/api/sandbox';
	import AgentStatusCard from './AgentStatusCard.svelte';
	import ActivityFeed from './ActivityFeed.svelte';
	import type { ActivityItem } from './ActivityFeed.svelte';
	import ConnectionStatusIndicator from './ConnectionStatusIndicator.svelte';

	interface Props {
		queueItemId: string;
		appId?: string;
		onComplete?: () => void;
		onError?: (error: string) => void;
		onProgress?: (percentage: number) => void;
		onDeploy?: (url: string) => void;
	}

	let { queueItemId, appId, onComplete, onError, onProgress, onDeploy }: Props = $props();

	let isCancelling = $state(false);
	let isDeploying = $state(false);
	let deployUrl = $state<string | null>(null);

	async function handleCancel() {
		if (isCancelling) return;
		isCancelling = true;
		try {
			await cancelGeneration(queueItemId);
			addActivity('system', 'Generation cancelled by user');
			generationDone = true;
			stopSimulatedActivity();
			sseClient?.disconnect();
			onError?.('Generation cancelled');
		} catch (err) {
			console.error('[AgentProgressPanel] Cancel failed:', err);
			addActivity('system', 'Failed to cancel generation', 'error');
		} finally {
			isCancelling = false;
		}
	}

	async function handleDeploy() {
		const targetId = appId || queueItemId;
		if (isDeploying || !targetId) return;
		isDeploying = true;
		try {
			const result = await deploySandbox(targetId);
			deployUrl = result.url;
			addActivity('system', `Deployed to ${result.url}`, 'success');
			onDeploy?.(result.url);
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Deploy failed';
			addActivity('system', msg, 'error');
		} finally {
			isDeploying = false;
		}
	}

	let sseClient: SSEClient<ProgressEvent> | null = null;
	let agents = $state<AgentCard[]>([
		{
			id: 'task-frontend',
			...AGENT_CONFIGS.frontend,
			status: 'pending',
			progress: 0,
			message: 'Waiting to start...'
		},
		{
			id: 'task-backend',
			...AGENT_CONFIGS.backend,
			status: 'pending',
			progress: 0,
			message: 'Waiting to start...'
		},
		{
			id: 'task-database',
			...AGENT_CONFIGS.database,
			status: 'pending',
			progress: 0,
			message: 'Waiting to start...'
		},
		{
			id: 'task-test',
			...AGENT_CONFIGS.test,
			status: 'pending',
			progress: 0,
			message: 'Waiting to start...'
		}
	]);

	// Activity feed state
	let activities = $state<ActivityItem[]>([]);
	let activityIdCounter = 0;

	function addActivity(agentType: AgentType | 'system', message: string, icon?: ActivityItem['icon']) {
		activities = [...activities, {
			id: `activity-${++activityIdCounter}`,
			timestamp: new Date(),
			agentType,
			message,
			icon
		}];
	}

	let completedCount = $state(0);
	let failedCount = $state(0);
	let isConnected = $state(false);

	let overallProgress = $derived(
		Math.round(agents.reduce((sum, agent) => sum + agent.progress, 0) / agents.length)
	);
	let allDone = $derived(completedCount + failedCount === agents.length);

	let statusMessage = $state('Waiting for generation to start...');
	let generationDone = $state(false);
	let pollingInterval: ReturnType<typeof setInterval> | null = null;
	let pollCount = $state(0);
	let simulatedActivityInterval: ReturnType<typeof setInterval> | null = null;
	// Timeout tracking - used in template and timeout check
	let lastEventTime = $state<Date | null>(null);
	let showTimeoutWarning = $state(false); // Used in template
	let timeoutCheckInterval: ReturnType<typeof setInterval> | null = null;
	let reconnectAttempts = $state(0);
	const MAX_RECONNECT_ATTEMPTS = 5;

	let connectionStatus = $derived.by((): 'connected' | 'reconnecting' | 'failed' | 'timeout' => {
		if (generationDone || allDone) return 'connected';
		if (showTimeoutWarning) return 'timeout';
		if (!isConnected && reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) return 'failed';
		if (!isConnected) return 'reconnecting';
		return 'connected';
	});

	function handleRetryConnection() {
		reconnectAttempts = 0;
		if (sseClient) {
			sseClient.disconnect();
			sseClient = null;
		}
		sseClient = createAgentProgressClient(
			queueItemId,
			handleProgressEvent,
			handleSSEError,
			handleSSEOpen
		);
		sseClient.connect();
	}

	// Add simulated activity messages when agents are working
	function startSimulatedActivity() {
		if (simulatedActivityInterval) return;
		
		simulatedActivityInterval = setInterval(() => {
			// Find an active agent and add a simulated message
			const activeAgents = agents.filter(a => a.status === 'in_progress' || a.status === 'starting');
			if (activeAgents.length > 0) {
				const agent = activeAgents[Math.floor(Math.random() * activeAgents.length)];
				const message = getRandomMessage(agent.type, 'in_progress');
				addActivity(agent.type, message);
			}
		}, 2000);
	}

	function stopSimulatedActivity() {
		if (simulatedActivityInterval) {
			clearInterval(simulatedActivityInterval);
			simulatedActivityInterval = null;
		}
	}

	function handleProgressEvent(event: ProgressEvent): void {
		// Update last event time and hide timeout warning
		lastEventTime = new Date();
		showTimeoutWarning = false;

		// Handle event by its type field (from SSE payload)
		const eventType = event.type;

		// Add to activity feed
		if (event.message) {
			const agentType = event.agent_type || (event.task_id?.includes('frontend') ? 'frontend' : 
				event.task_id?.includes('backend') ? 'backend' :
				event.task_id?.includes('database') ? 'database' :
				event.task_id?.includes('test') ? 'test' : 'orchestrator');
			
			const icon = event.status === 'completed' ? 'success' : 
				event.status === 'failed' ? 'error' : undefined;
			
			addActivity(agentType as AgentType, event.message, icon);
		}

		// Handle generation_complete - trigger onComplete regardless of agent tracking
		if (eventType === 'generation_complete') {
			if (import.meta.env.DEV) console.log('[AgentProgressPanel] Generation complete!');
			addActivity('system', 'Generation complete! Your app is ready.', 'sparkle');
			stopSimulatedActivity();
			generationDone = true;
			// Mark all pending agents as completed
			for (const agent of agents) {
				if (agent.status !== 'completed' && agent.status !== 'failed') {
					agent.status = 'completed';
					agent.progress = 100;
					agent.message = 'Completed';
					completedCount++;
				}
			}
			setTimeout(() => {
				onComplete?.();
				sseClient?.disconnect();
			}, 500);
			return;
		}

		// Handle error/failed events
		if (eventType === 'error') {
			console.error('[AgentProgressPanel] Generation error:', event.message);
			statusMessage = event.message || 'Generation failed';
			addActivity('system', event.message || 'Generation failed', 'error');
			stopSimulatedActivity();
			onError?.(event.message || 'Generation failed');
			return;
		}

		// Update overall status message for non-agent events
		if (event.message) {
			statusMessage = event.message;
		}

		// Try to match to a specific agent
		const agent = agents.find((a) => a.id === event.task_id);
		if (!agent) {
			// Not an agent-specific event - that's OK for orchestrator/planning events
			if (event.task_id && !event.task_id.includes('orchestrator') && event.task_id !== undefined) {
				console.warn('[AgentProgressPanel] Unknown agent:', event.task_id);
			}
			return;
		}

		const wasCompleted = agent.status === 'completed';
		const wasFailed = agent.status === 'failed';

		agent.status = event.status;
		agent.progress = event.progress;
		agent.message = event.message;

		if (event.status === 'completed' && !wasCompleted) completedCount++;
		if (event.status === 'failed' && !wasFailed) failedCount++;

		// Notify parent of progress update
		if (onProgress) {
			onProgress(overallProgress);
		}

		// Delay before calling onComplete to show final state
		if (allDone && !generationDone) {
			generationDone = true;
			setTimeout(() => {
				onComplete?.();
				sseClient?.disconnect();
			}, 1000);
		}
	}

	function handleSSEError(error: Error): void {
		console.error('[AgentProgressPanel] SSE error:', error);
		isConnected = false;
		reconnectAttempts = sseClient?.getReconnectAttempts() ?? reconnectAttempts + 1;
		// Don't call onError for SSE connection issues - polling fallback will handle it
		if (import.meta.env.DEV) console.log('[AgentProgressPanel] SSE failed, polling fallback active');
	}

	function handleSSEOpen(): void {
		if (import.meta.env.DEV) console.log('[AgentProgressPanel] SSE connected');
		isConnected = true;
		lastEventTime = new Date();
		addActivity('system', 'Connected to generation server');
		startSimulatedActivity();
		startTimeoutCheck();
	}

	// Check if we haven't received any events for 30+ seconds
	function startTimeoutCheck(): void {
		if (timeoutCheckInterval) return;

		timeoutCheckInterval = setInterval(() => {
			if (generationDone || allDone) {
				stopTimeoutCheck();
				return;
			}

			if (lastEventTime) {
				const secondsSinceLastEvent = (Date.now() - lastEventTime.getTime()) / 1000;
				if (secondsSinceLastEvent > 30) {
					showTimeoutWarning = true;
				}
			}
		}, 5000); // Check every 5 seconds
	}

	function stopTimeoutCheck(): void {
		if (timeoutCheckInterval) {
			clearInterval(timeoutCheckInterval);
			timeoutCheckInterval = null;
		}
		showTimeoutWarning = false;
	}

	// Polling fallback: check queue item status every 8 seconds
	// This ensures the user sees generated code even if SSE fails
	async function pollQueueStatus(): Promise<void> {
		if (generationDone) return;
		pollCount++;
		try {
			const response = await fetch(`/api/osa/module-instances/queue/${queueItemId}/status`, {
				credentials: 'include'
			});
			if (!response.ok) {
				if (import.meta.env.DEV) console.log('[AgentProgressPanel] Poll status response:', response.status);
				return;
			}
			const data = await response.json();
			if (import.meta.env.DEV) console.log('[AgentProgressPanel] Poll status:', data.status, 'has_files:', data.has_files);

			if (data.status === 'completed') {
				if (import.meta.env.DEV) console.log('[AgentProgressPanel] Generation completed (detected via polling)', 'has_files:', data.has_files);
				statusMessage = data.has_files ? 'Generation complete! Loading files...' : 'Generation complete!';
				generationDone = true;
				addActivity('system', 'Generation complete! Your app is ready.', 'sparkle');
				stopSimulatedActivity();
				// Mark all agents as completed
				for (const agent of agents) {
					if (agent.status !== 'completed' && agent.status !== 'failed') {
						agent.status = 'completed';
						agent.progress = 100;
						agent.message = 'Completed';
						completedCount++;
					}
				}
				stopPolling();
				setTimeout(() => {
					onComplete?.();
					sseClient?.disconnect();
				}, 500);
			} else if (data.status === 'failed') {
				console.error('[AgentProgressPanel] Generation failed (detected via polling)');
				addActivity('system', 'Generation failed', 'error');
				stopSimulatedActivity();
				stopPolling();
				onError?.('Generation failed');
			} else if (data.status === 'processing') {
				statusMessage = 'AI agents are generating your app...';
				startSimulatedActivity();
				// Update agents to show activity
				for (const agent of agents) {
					if (agent.status === 'pending') {
						agent.status = 'in_progress';
						agent.progress = Math.min(30, pollCount * 5);
						agent.message = 'Working...';
					}
				}
			}
		} catch (err) {
			console.warn('[AgentProgressPanel] Poll error:', err);
		}
	}

	function startPolling(): void {
		// Start polling after a short delay to give SSE a chance
		setTimeout(() => {
			if (!generationDone) {
				if (import.meta.env.DEV) console.log('[AgentProgressPanel] Starting polling fallback');
				pollQueueStatus(); // Immediate first poll
				pollingInterval = setInterval(pollQueueStatus, 8000);
			}
		}, 3000);
	}

	function stopPolling(): void {
		if (pollingInterval) {
			clearInterval(pollingInterval);
			pollingInterval = null;
		}
	}

	onMount(() => {
		// Add initial activity
		addActivity('system', 'Initializing app generation...');

		// Start SSE connection
		sseClient = createAgentProgressClient(
			queueItemId,
			handleProgressEvent,
			handleSSEError,
			handleSSEOpen
		);
		sseClient.connect();

		// Also start polling as fallback (works even if SSE fails)
		startPolling();
	});

	onDestroy(() => {
		stopPolling();
		stopSimulatedActivity();
		stopTimeoutCheck();
		if (sseClient) {
			sseClient.disconnect();
			sseClient = null;
		}
	});
</script>

<div class="agent-progress-panel space-y-5">
	<!-- Header with progress -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="relative">
				<div class="w-12 h-12 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
					<Sparkles class="w-6 h-6 text-white" />
				</div>
				{#if !allDone && !generationDone}
					<div class="absolute inset-0 rounded-full bg-blue-400/30 animate-ping"></div>
				{/if}
			</div>
			<div>
				<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
					{#if allDone || generationDone}
						Generation Complete
					{:else}
						Generating Your App
					{/if}
				</h3>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					{#if allDone || generationDone}
						{#if failedCount > 0}
							Completed with {failedCount} failure{failedCount !== 1 ? 's' : ''}
						{:else}
							All agents finished successfully!
						{/if}
					{:else}
						4 AI agents working in parallel
					{/if}
				</p>
			</div>
		</div>

		<div class="flex items-center gap-4">
			{#if !allDone && !generationDone}
				<button
					onclick={handleCancel}
					disabled={isCancelling}
					class="btn-pill btn-pill-danger btn-pill-sm flex items-center gap-1.5"
				>
					{#if isCancelling}
						<Loader2 class="w-3.5 h-3.5 animate-spin" />
					{:else}
						<StopCircle class="w-3.5 h-3.5" />
					{/if}
					Cancel
				</button>
			{/if}

			{#if !allDone && !generationDone}
				<ConnectionStatusIndicator
					status={connectionStatus}
					reconnectAttempt={reconnectAttempts}
					maxReconnectAttempts={MAX_RECONNECT_ATTEMPTS}
					onRetry={handleRetryConnection}
					compact={true}
				/>
			{/if}

			<div class="text-right">
				<div class="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
					{overallProgress}%
				</div>
				<div class="text-xs text-gray-500 dark:text-gray-400">
					Overall Progress
				</div>
			</div>
		</div>
	</div>

	<!-- Main progress bar -->
	<div class="w-full h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden shadow-inner">
		<div
			class="h-full bg-gradient-to-r from-blue-500 via-purple-500 to-blue-600 transition-all duration-700 ease-out relative overflow-hidden rounded-full"
			style="width: {overallProgress}%"
		>
			{#if overallProgress < 100}
				<div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/40 to-transparent animate-shimmer"></div>
			{/if}
		</div>
	</div>

	<!-- Status message -->
	<div class="text-center text-sm text-gray-600 dark:text-gray-400 py-1">
		{statusMessage}
	</div>

	<!-- Connection status banner (timeout / failed states) -->
	{#if connectionStatus === 'timeout' && !allDone && !generationDone}
		<ConnectionStatusIndicator status="timeout" />
	{:else if connectionStatus === 'failed' && !allDone && !generationDone}
		<ConnectionStatusIndicator
			status="failed"
			onRetry={handleRetryConnection}
		/>
	{/if}

	<!-- Agent cards grid -->
	<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
		{#each agents as agent, index (agent.id)}
			<AgentStatusCard {agent} />
		{/each}
	</div>

	<!-- Live activity feed -->
	<ActivityFeed {activities} />

	<!-- Error banner if failed -->
	{#if failedCount > 0 && allDone}
		<div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
			<div class="flex items-start gap-3">
				<XCircle class="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
				<div>
					<h4 class="font-semibold text-red-900 dark:text-red-200">
						{failedCount} Agent{failedCount !== 1 ? 's' : ''} Failed
					</h4>
					<p class="text-sm text-red-700 dark:text-red-300 mt-1">
						Some agents encountered errors during generation. Check the activity feed for details.
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Deploy / Download actions on completion -->
	{#if (allDone || generationDone) && failedCount === 0}
		<div class="flex items-center justify-between p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
			<div class="text-sm text-green-800 dark:text-green-200 font-medium">
				{#if deployUrl}
					Deployed at <a href={deployUrl} target="_blank" rel="noopener" class="underline">{deployUrl}</a>
				{:else}
					Ready to deploy to sandbox
				{/if}
			</div>
			<div class="flex gap-2">
				{#if !deployUrl}
					<button
						onclick={handleDeploy}
						disabled={isDeploying}
						class="flex items-center gap-1.5 btn-pill btn-pill-success disabled:opacity-50"
					>
						{#if isDeploying}
							<Loader2 class="w-4 h-4 animate-spin" />
							Deploying...
						{:else}
							<Rocket class="w-4 h-4" />
							Deploy
						{/if}
					</button>
				{/if}
				<button
					onclick={() => goto(`/generated-apps/${appId || queueItemId}`)}
					class="flex items-center gap-1.5 btn-pill btn-pill-primary"
				>
					<Code2 class="w-4 h-4" />
					Open in Editor
				</button>
				<a
					href="/api/osa/module-instances/{appId || queueItemId}/download"
					download
					class="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
				>
					<Download class="w-4 h-4" />
					Download ZIP
				</a>
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

	.agent-progress-panel {
		animation: fadeIn 0.3s ease-in;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>
