<script lang="ts">
	import { fly } from 'svelte/transition';
	import type { ModelOption, ModelCapability } from '$lib/utils/chatHelpers';
	import { getModelCapabilities, capabilityInfo, cloudModelsMap } from '$lib/utils/chatHelpers';
	import { isCloudModelId } from '$lib/ai/modelCatalog';

	interface Props {
		selectedModel: string;
		currentModelName: string;
		warmingUpModel: string | null;
		installedModels: ModelOption[];
		ollamaCloudModels: ModelOption[];
		configuredProviders: Set<string>;
		loadingModels: boolean;
		onSelectModel: (modelId: string) => void;
	}

	let {
		selectedModel,
		currentModelName,
		warmingUpModel,
		installedModels,
		ollamaCloudModels,
		configuredProviders,
		loadingModels,
		onSelectModel,
	}: Props = $props();

	let showDropdown = $state(false);

	function select(id: string) {
		onSelectModel(id);
		showDropdown = false;
	}
</script>

<div class="relative">
	<button
		onclick={() => (showDropdown = !showDropdown)}
		class="trigger-btn flex items-center gap-1.5 px-2 py-1.5 text-sm rounded-lg transition-colors"
		title="Select AI Model"
	>
		{#if warmingUpModel === selectedModel}
			<svg class="w-3.5 h-3.5 animate-spin icon-warming flex-shrink-0" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
		{/if}
		<span class="truncate max-w-[140px]">{currentModelName || 'Select model'}</span>
		{#if warmingUpModel === selectedModel}
			<span class="text-xs icon-warming">warming...</span>
		{:else}
			<svg class="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		{/if}
	</button>

	{#if showDropdown}
		<!-- Click-outside backdrop -->
		<button
			class="fixed inset-0 z-[5] cursor-default bg-transparent"
			onclick={() => (showDropdown = false)}
			aria-label="Close model dropdown"
		></button>

		<div
			class="dropdown-menu absolute left-0 top-full mt-2 w-72 rounded-xl shadow-lg py-2 z-30 max-h-96 overflow-y-auto"
			transition:fly={{ y: -10, duration: 200 }}
		>
			{#if loadingModels}
				<div class="px-4 py-3 text-sm text-muted text-center">Loading models...</div>
			{:else if installedModels.length === 0 && ollamaCloudModels.length === 0 && configuredProviders.size === 0}
				<div class="px-4 py-3 text-center">
					<p class="text-sm text-muted mb-2">No models available</p>
					<a href="/settings/ai" class="text-xs text-link hover:underline">Configure in AI Settings</a>
				</div>
			{:else}
				<!-- Selected model at top -->
				{@const allModels = [...installedModels, ...ollamaCloudModels, ...Array.from(configuredProviders).flatMap(p => cloudModelsMap[p] || [])]}
				{@const selectedModelObj = allModels.find(m => m.id === selectedModel)}
				{#if selectedModelObj}
					{@const caps = selectedModelObj.capabilities || getModelCapabilities(selectedModelObj.id) || []}
					{@const isCloud = selectedModelObj.type === 'cloud' || isCloudModelId(selectedModelObj.id)}
					<div class="px-3 py-1.5">
						<span class="section-label-selected text-xs font-semibold uppercase tracking-wider flex items-center gap-1">
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
							Selected
						</span>
					</div>
					<button
						onclick={() => { showDropdown = false; }}
						class="model-selected-row w-full px-4 py-2.5 text-left transition-colors"
					>
						<div class="flex items-start gap-2">
							<svg class="w-4 h-4 icon-model mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								{#if isCloud}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
								{:else}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								{/if}
							</svg>
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-1.5 flex-wrap">
									<span class="text-sm font-medium model-name-selected">{selectedModelObj.name}</span>
									<span class="model-badge text-[10px] px-1.5 py-0.5 rounded">{isCloud ? 'Cloud' : 'Local'}</span>
								</div>
								{#if selectedModelObj.size}
									<div class="text-xs text-muted mt-0.5">{selectedModelObj.size}</div>
								{/if}
								{#if caps.length > 0}
									<div class="flex flex-wrap gap-1 mt-1">
										{#each caps.slice(0, 4) as cap}
											<span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[9px] font-medium rounded {capabilityInfo[cap].color}">
												<span>{capabilityInfo[cap].icon}</span>
												<span>{capabilityInfo[cap].label}</span>
											</span>
										{/each}
									</div>
								{/if}
							</div>
							<svg class="w-4 h-4 icon-checkmark flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
						</div>
					</button>
				{/if}

				<!-- Cloud (Ollama Remote) Section -->
				{#if ollamaCloudModels.length > 0}
					<div class="section-header px-3 py-1.5">
						<span class="section-label text-xs font-semibold uppercase tracking-wider flex items-center gap-1">
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
							</svg>
							Cloud (Ollama Remote)
						</span>
					</div>
					{#each ollamaCloudModels.filter(m => m.id !== selectedModel) as model}
						{@const caps = model.capabilities || getModelCapabilities(model.id) || []}
						<button
							onclick={() => select(model.id)}
							class="model-item w-full px-4 py-2.5 text-left transition-colors"
						>
							<div class="flex items-start gap-2">
								<svg class="w-4 h-4 icon-model mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
								</svg>
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-1.5 flex-wrap">
										<span class="text-sm font-medium model-name">{model.name}</span>
									</div>
									{#if model.size}
										<div class="text-xs text-muted mt-0.5">{model.size}</div>
									{/if}
									{#if caps.length > 0}
										<div class="flex flex-wrap gap-1 mt-1">
											{#each caps.slice(0, 4) as cap}
												<span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[9px] font-medium rounded {capabilityInfo[cap].color}">
													<span>{capabilityInfo[cap].icon}</span>
													<span>{capabilityInfo[cap].label}</span>
												</span>
											{/each}
											{#if caps.length > 4}
												<span class="text-[9px] text-muted">+{caps.length - 4}</span>
											{/if}
										</div>
									{/if}
								</div>
							</div>
						</button>
					{/each}
				{/if}

				<!-- Local Models Section -->
				{#if installedModels.length > 0}
					<div class="section-header {ollamaCloudModels.length > 0 ? 'section-divider' : ''} px-3 py-1.5">
						<span class="section-label text-xs font-semibold uppercase tracking-wider flex items-center gap-1">
							<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
							</svg>
							Local (Ollama)
						</span>
					</div>
					{#each installedModels.filter(m => m.id !== selectedModel) as model}
						{@const caps = model.capabilities || getModelCapabilities(model.id) || []}
						<button
							onclick={() => select(model.id)}
							class="model-item w-full px-4 py-2.5 text-left transition-colors"
						>
							<div class="flex items-start gap-2">
								<svg class="w-4 h-4 icon-model mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-1.5 flex-wrap">
										<span class="text-sm font-medium model-name">{model.name}</span>
									</div>
									{#if model.size}
										<div class="text-xs text-muted mt-0.5">{model.size}</div>
									{/if}
									{#if caps.length > 0}
										<div class="flex flex-wrap gap-1 mt-1">
											{#each caps.slice(0, 4) as cap}
												<span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[9px] font-medium rounded {capabilityInfo[cap].color}">
													<span>{capabilityInfo[cap].icon}</span>
													<span>{capabilityInfo[cap].label}</span>
												</span>
											{/each}
											{#if caps.length > 4}
												<span class="text-[9px] text-muted">+{caps.length - 4}</span>
											{/if}
										</div>
									{/if}
								</div>
							</div>
						</button>
					{/each}
				{/if}

				<!-- Cloud Models by Provider -->
				{#each Array.from(configuredProviders) as provider}
					{@const providerModels = (cloudModelsMap[provider] || []).filter(m => m.id !== selectedModel)}
					{#if providerModels.length > 0}
						<div class="section-header section-divider px-3 py-1.5">
							<span class="section-label text-xs font-semibold uppercase tracking-wider flex items-center gap-1">
								<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
								</svg>
								{provider.charAt(0).toUpperCase() + provider.slice(1)}
							</span>
						</div>
						{#each providerModels as model}
							{@const caps = model.capabilities || []}
							<button
								onclick={() => select(model.id)}
								class="model-item w-full px-4 py-2.5 text-left transition-colors"
							>
								<div class="flex items-start gap-2">
									<svg class="w-4 h-4 icon-model mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
									</svg>
									<div class="flex-1 min-w-0">
										<div class="text-sm font-medium model-name">{model.name}</div>
										{#if model.description}
											<div class="text-xs text-muted mt-0.5">{model.description}</div>
										{/if}
										{#if caps.length > 0}
											<div class="flex flex-wrap gap-1 mt-1">
												{#each caps.slice(0, 4) as cap}
													<span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[9px] font-medium rounded {capabilityInfo[cap].color}">
														<span>{capabilityInfo[cap].icon}</span>
														<span>{capabilityInfo[cap].label}</span>
													</span>
												{/each}
												{#if caps.length > 4}
													<span class="text-[9px] text-muted">+{caps.length - 4}</span>
												{/if}
											</div>
										{/if}
									</div>
								</div>
							</button>
						{/each}
					{/if}
				{/each}

				<!-- Settings link -->
				<div class="section-divider mt-1 pt-1">
					<a
						href="/settings/ai"
						onclick={() => (showDropdown = false)}
						class="settings-link w-full px-4 py-2 text-left text-sm transition-colors flex items-center gap-2"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
						AI Settings
					</a>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	/* Trigger button */
	.trigger-btn {
		color: var(--dt2);
	}
	.trigger-btn:hover {
		color: var(--dt);
		background: var(--dbg2);
	}

	/* Warming indicator */
	.icon-warming {
		color: var(--dt2);
	}

	/* Dropdown panel */
	.dropdown-menu {
		background: var(--dbg);
		border: 1px solid var(--dbd);
	}

	/* Text helpers */
	.text-muted {
		color: var(--dt3);
	}
	.text-link {
		color: var(--dt2);
	}
	.text-link:hover {
		color: var(--dt);
	}

	/* Section header divider */
	.section-header {
		margin-top: 4px;
		padding-top: 4px;
	}
	.section-divider {
		border-top: 1px solid var(--dbd);
	}

	/* Section label */
	.section-label {
		color: var(--dt3);
	}
	.section-label-selected {
		color: var(--dt2);
	}

	/* Selected model row */
	.model-selected-row {
		background: var(--dbg2);
	}
	.model-selected-row:hover {
		background: var(--dbg3);
	}

	/* Model item rows */
	.model-item:hover {
		background: var(--dbg2);
	}

	/* Model name */
	.model-name {
		color: var(--dt);
	}
	.model-name-selected {
		color: var(--dt);
	}

	/* Model type badge */
	.model-badge {
		background: var(--dbg3);
		color: var(--dt3);
	}

	/* Model icons */
	.icon-model {
		color: var(--dt3);
	}

	/* Checkmark icon */
	.icon-checkmark {
		color: var(--dt2);
	}

	/* Settings link */
	.settings-link {
		color: var(--dt3);
		display: flex;
	}
	.settings-link:hover {
		background: var(--dbg2);
		color: var(--dt2);
	}
</style>
