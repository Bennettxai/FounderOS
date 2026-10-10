<script lang="ts" module>
	import type { AgentType } from '$lib/types/agent';

	export interface ActivityItem {
		id: string;
		timestamp: Date;
		agentType: AgentType | 'system';
		message: string;
		icon?: 'success' | 'error' | 'sparkle';
	}
</script>

<script lang="ts">
	import { CheckCircle, XCircle, Sparkles, Loader2 } from 'lucide-svelte';

	interface Props {
		activities: ActivityItem[];
	}

	let { activities }: Props = $props();

	let feedContainer: HTMLDivElement | undefined;

	$effect(() => {
		if (activities.length && feedContainer) {
			feedContainer.scrollTop = feedContainer.scrollHeight;
		}
	});

	function formatTime(date: Date): string {
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}
</script>

{#if activities.length > 0}
	<div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
		<div class="flex items-center justify-between px-3 py-2 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
			<span class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Activity Feed</span>
			<span class="text-xs text-gray-400">{activities.length} events</span>
		</div>
		<div
			bind:this={feedContainer}
			class="max-h-40 overflow-y-auto bg-white dark:bg-gray-900 p-2 space-y-1"
		>
			{#each activities as activity (activity.id)}
				<div class="flex items-start gap-2 px-2 py-1 text-xs rounded hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
					<span class="text-gray-400 dark:text-gray-500 flex-shrink-0 font-mono">
						{formatTime(activity.timestamp)}
					</span>
					{#if activity.icon === 'success'}
						<CheckCircle class="w-3.5 h-3.5 text-green-500 flex-shrink-0 mt-0.5" />
					{:else if activity.icon === 'error'}
						<XCircle class="w-3.5 h-3.5 text-red-500 flex-shrink-0 mt-0.5" />
					{:else if activity.icon === 'sparkle'}
						<Sparkles class="w-3.5 h-3.5 text-purple-500 flex-shrink-0 mt-0.5" />
					{:else}
						<Loader2 class="w-3.5 h-3.5 text-gray-400 flex-shrink-0 mt-0.5" />
					{/if}
					<span class="text-gray-700 dark:text-gray-300">{activity.message}</span>
				</div>
			{/each}
		</div>
	</div>
{/if}
