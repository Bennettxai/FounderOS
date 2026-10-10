<script lang="ts">
	import { Code, Server, Database, TestTube, Loader2, CheckCircle, XCircle, Clock } from 'lucide-svelte';
	import type { AgentCard } from '$lib/types/agent';
	import { getBorderColorClass, getStatusColorClass, getStatusBadgeClass, AGENT_CONFIGS } from '$lib/types/agent';

	interface Props {
		agent: AgentCard;
		compact?: boolean;
	}

	let { agent, compact = false }: Props = $props();

	const STATUS_ICONS = {
		completed: CheckCircle,
		failed: XCircle,
		in_progress: Loader2,
		starting: Loader2,
		pending: Clock
	} as const;

	const AGENT_ICONS = {
		Code,
		Server,
		Database,
		TestTube
	} as const;

	let Icon = $derived(AGENT_ICONS[agent.icon]);
	let StatusIcon = $derived(STATUS_ICONS[agent.status] ?? Clock);
	let borderClass = $derived(getBorderColorClass(agent.status));
	let statusColorClass = $derived(getStatusColorClass(agent.status));
	let badgeClass = $derived(getStatusBadgeClass(agent.status));
	let isActive = $derived(agent.status === 'in_progress' || agent.status === 'starting');
</script>

<div
	class="border-2 rounded-lg transition-all duration-300 bg-white dark:bg-gray-800 {borderClass} {compact ? 'p-3' : 'p-4'} {isActive ? 'shadow-lg shadow-blue-500/20' : ''}"
>
	<div class="flex items-center justify-between" class:mb-3={!compact} class:mb-2={compact}>
		<div class="flex items-center gap-3">
			<div
				class="rounded-full flex items-center justify-center transition-colors"
				class:w-10={!compact}
				class:h-10={!compact}
				class:w-8={compact}
				class:h-8={compact}
				class:bg-blue-100={isActive}
				class:bg-green-100={agent.status === 'completed'}
				class:bg-red-100={agent.status === 'failed'}
				class:bg-gray-100={agent.status === 'pending'}
			>
				<Icon class="{statusColorClass} {compact ? 'w-4 h-4' : 'w-5 h-5'}" />
			</div>

			<div>
				<h4 class="font-semibold text-gray-900 dark:text-white" class:text-sm={compact}>
					{agent.name}
				</h4>
				{#if !compact}
					<p class="text-xs text-gray-500 dark:text-gray-400">
						{AGENT_CONFIGS[agent.type]?.description ?? ''}
					</p>
				{/if}
			</div>
		</div>

		<StatusIcon
			class="{statusColorClass} {compact ? 'w-4 h-4' : 'w-5 h-5'} {isActive ? 'animate-spin' : ''}"
		/>
	</div>

	{#if agent.status !== 'pending'}
		<div class:mb-3={!compact} class:mb-2={compact}>
			<div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
				<div
					class="h-full transition-all duration-300 rounded-full"
					class:bg-blue-500={isActive}
					class:bg-green-500={agent.status === 'completed'}
					class:bg-red-500={agent.status === 'failed'}
					style="width: {agent.progress}%"
				/>
			</div>
			<div class="flex items-center justify-between mt-1">
				<span class="text-xs text-gray-500 dark:text-gray-400">
					{agent.progress}%
				</span>
			</div>
		</div>
	{/if}

	<p class="text-gray-600 dark:text-gray-300 mb-2" class:text-sm={!compact} class:text-xs={compact}>
		{agent.message}
	</p>

	<div>
		<span
			class="inline-flex items-center rounded-full font-medium {badgeClass}"
			class:px-2.5={!compact}
			class:py-1={!compact}
			class:px-2={compact}
			class:py-0.5={compact}
			class:text-xs={true}
		>
			{agent.status.replace('_', ' ').toUpperCase()}
		</span>
	</div>
</div>
