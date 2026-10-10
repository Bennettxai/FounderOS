<script lang="ts">
	import UserFactsPanel from '$lib/components/settings/UserFactsPanel.svelte';
	import type { SystemInfo } from '$lib/api';

	interface Props {
		systemInfo?: SystemInfo | null;
	}

	let { systemInfo = null }: Props = $props();
</script>

<div class="space-y-6">
	<!-- User Facts Management -->
	<div class="card" style="padding: 0; overflow: hidden; max-height: 600px;">
		<UserFactsPanel />
	</div>

	<!-- AI Settings Links -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
		<!-- Advanced AI Settings -->
		<div class="card text-center py-6">
			<svg class="w-8 h-8 mx-auto mb-3 st-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
			</svg>
			<h3 class="font-medium st-title mb-2">Advanced Settings</h3>
			<p class="text-sm st-muted mb-4">
				Model configuration, providers, and agent settings
			</p>
			<a href="/settings/ai" class="inline-flex items-center gap-2 btn btn-secondary text-sm">
				Configure
				<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</a>
		</div>

		<!-- Thinking Settings -->
		<div class="card text-center py-6">
			<svg class="w-8 h-8 mx-auto mb-3 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
			</svg>
			<h3 class="font-medium st-title mb-2">Thinking Settings</h3>
			<p class="text-sm st-muted mb-4">
				Chain-of-thought, reasoning display, and thinking preferences
			</p>
			<a href="/settings/ai/thinking" class="inline-flex items-center gap-2 btn btn-secondary text-sm">
				Configure
				<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</a>
		</div>

		<!-- Reasoning Templates -->
		<div class="card text-center py-6">
			<svg class="w-8 h-8 mx-auto mb-3 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<h3 class="font-medium st-title mb-2">Reasoning Templates</h3>
			<p class="text-sm st-muted mb-4">
				Create and manage reasoning templates for AI thinking
			</p>
			<a href="/settings/ai/templates" class="inline-flex items-center gap-2 btn btn-secondary text-sm">
				Manage
				<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</a>
		</div>
	</div>

	<!-- Quick status -->
	<div class="card">
		<h2 class="text-lg font-medium st-title mb-4">Current Status</h2>
		<div class="flex items-center gap-3">
			<div class="w-3 h-3 rounded-full {systemInfo?.ollama_mode === 'local' ? 'bg-green-500' : 'bg-blue-500'}"></div>
			<div>
				<p class="font-medium st-title">
					{#if systemInfo?.ollama_mode === 'local'}
						Local Mode (Ollama)
					{:else if systemInfo?.active_provider === 'groq'}
						Groq Cloud
					{:else if systemInfo?.active_provider === 'anthropic'}
						Claude (Anthropic)
					{:else}
						Cloud Mode
					{/if}
				</p>
				<p class="text-sm st-muted">
					{systemInfo?.ollama_mode === 'local'
						? 'Running AI models locally on your machine'
						: 'Using cloud-hosted AI models'}
				</p>
			</div>
		</div>
	</div>
</div>

<style>
	.st-title { color: var(--dt, var(--bos-text-primary, #111)); }
	.st-muted { color: var(--dt3, var(--bos-text-secondary, #6b7280)); }
	.st-icon { color: var(--dt4, var(--bos-text-tertiary, #9ca3af)); }
</style>
