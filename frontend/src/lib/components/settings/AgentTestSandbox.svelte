<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import type { CustomAgent } from '$lib/api/ai';
	import { getApiBaseUrl } from '$lib/api/base';

	interface Props {
		agent?: CustomAgent;
		systemPrompt?: string;
	}

	let { agent, systemPrompt = '' }: Props = $props();

	let testMessage = $state('');
	let testResult = $state<{
		response: string;
		tokens_used: number;
		duration_ms: number;
		model: string;
	} | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	async function testAgent() {
		if (!testMessage.trim()) {
			error = 'Please enter a test message';
			return;
		}

		isLoading = true;
		error = null;
		testResult = null;

		try {
			const endpoint = agent
				? `/api/agents/${agent.id}/test`
				: '/api/agents/sandbox';

			const body: any = {
				test_message: testMessage
			};

			if (!agent && systemPrompt) {
				body.system_prompt = systemPrompt;
			}

			const response = await fetch(`${getApiBaseUrl()}${endpoint}`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify(body)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || 'Failed to test agent');
			}

			testResult = await response.json();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to test agent';
		} finally {
			isLoading = false;
		}
	}

	function clearResults() {
		testResult = null;
		error = null;
	}
</script>

<div class="space-y-4">
	<!-- Test Input -->
	<div>
		<label for="test-message" class="block text-sm font-medium st-label mb-2">
			Test Message
		</label>
		<textarea
			id="test-message"
			bind:value={testMessage}
			placeholder="Enter a test message to see how the agent responds..."
			rows={4}
			class="w-full px-3 py-2 st-textarea rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-transparent resize-none"
			disabled={isLoading}
		></textarea>
	</div>

	<!-- Actions -->
	<div class="flex items-center gap-3">
		<button
			onclick={testAgent}
			disabled={isLoading || !testMessage.trim()}
			class="btn-pill btn-pill-primary flex items-center gap-2"
		>
			{#if isLoading}
				<svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Testing...
			{:else}
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
				</svg>
				Test Agent
			{/if}
		</button>

		{#if testResult || error}
			<button
				onclick={clearResults}
				class="btn-pill btn-pill-ghost"
			>
				Clear Results
			</button>
		{/if}
	</div>

	<!-- Error -->
	{#if error}
		<div
			class="p-4 bg-red-50 border border-red-200 rounded-lg"
			transition:slide={{ duration: 200 }}
		>
			<div class="flex items-start gap-3">
				<svg class="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				<div>
					<h4 class="font-medium text-red-900">Test Failed</h4>
					<p class="text-sm text-red-700 mt-1">{error}</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Results -->
	{#if testResult}
		<div
			class="space-y-3 st-results-panel rounded-lg p-4"
			transition:slide={{ duration: 200 }}
		>
			<!-- Metadata -->
			<div class="flex items-center gap-4 text-xs st-meta pb-3 st-results-divider">
				<div class="flex items-center gap-1.5">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
					</svg>
					<span>{testResult.model}</span>
				</div>
				<div class="flex items-center gap-1.5">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span>{testResult.duration_ms}ms</span>
				</div>
				<div class="flex items-center gap-1.5">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
					</svg>
					<span>{testResult.tokens_used} tokens</span>
				</div>
			</div>

<style>
	.st-title { color: var(--dt, var(--bos-text-primary, #111)); }
	.st-label { color: var(--dt2, var(--bos-text-secondary, #555)); }
	.st-meta  { color: var(--dt3, var(--bos-text-tertiary, #888)); }
	.st-textarea {
		border: 1px solid var(--dbd, var(--bos-border, #e0e0e0));
		background: var(--dbg, var(--bos-card, #fff));
		color: var(--dt, var(--bos-text-primary, #111));
	}
	.st-results-panel {
		background: var(--bos-settings-card-bg);
		border: 1px solid var(--bos-settings-card-border);
		border-radius: var(--bos-settings-card-radius);
	}
	.st-results-divider {
		border-bottom: 1px solid var(--dbd, var(--bos-border, #e0e0e0));
	}
	.st-response-card {
		background: var(--dbg, var(--bos-card, #fff));
		border: 1px solid var(--dbd, var(--bos-border, #e0e0e0));
	}
</style>
			<div>
				<h4 class="text-sm font-medium st-label mb-2">Agent Response:</h4>
				<div class="st-response-card rounded-lg p-4">
					<div class="prose prose-sm max-w-none st-title">
						{testResult.response}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
