<script lang="ts">
	import { Sparkles, Star, ArrowRight } from 'lucide-svelte';
	import type { AppTemplate } from '$lib/api/templates';
	import { categoryLabels, categoryColors } from '$lib/stores/templateStore';

	interface Props {
		template: AppTemplate;
		matchScore?: number;
		onUse?: (templateId: string) => void;
		onPreview?: (templateId: string) => void;
	}

	let { template, matchScore, onUse, onPreview }: Props = $props();

	function handleUse() {
		onUse?.(template.id);
	}

	function handlePreview() {
		onPreview?.(template.id);
	}

	function formatPopularity(score: number): string {
		if (score >= 1000) return `${(score / 1000).toFixed(1)}k`;
		return score.toString();
	}
</script>

<div
	class="group relative bg-white border border-gray-200 rounded-xl p-5 hover:shadow-lg hover:border-gray-300 transition-all duration-200"
>
	<!-- Premium Badge -->
	{#if template.is_premium}
		<div class="absolute top-3 right-3 z-10">
			<div
				class="flex items-center gap-1 px-2 py-1 bg-gradient-to-r from-amber-400 to-orange-500 text-white text-xs font-semibold rounded-full"
			>
				<Sparkles class="w-3 h-3" />
				<span>Premium</span>
			</div>
		</div>
	{/if}

	<!-- Icon/Image -->
	<div class="mb-4">
		{#if template.preview_image_url}
			<div class="w-full h-32 rounded-lg overflow-hidden bg-gray-100">
				<img
					src={template.preview_image_url}
					alt={template.name}
					class="w-full h-full object-cover"
				/>
			</div>
		{:else if template.icon_url}
			<div
				class="w-16 h-16 rounded-lg bg-gradient-to-br from-gray-100 to-gray-200 flex items-center justify-center"
			>
				<img src={template.icon_url} alt={template.name} class="w-10 h-10" />
			</div>
		{:else}
			<div
				class="w-16 h-16 rounded-lg bg-gradient-to-br from-blue-100 to-purple-100 flex items-center justify-center text-2xl"
			>
				{template.name.charAt(0).toUpperCase()}
			</div>
		{/if}
	</div>

	<!-- Header -->
	<div class="mb-3">
		<h3 class="font-semibold text-gray-900 text-lg mb-1 line-clamp-1">
			{template.name}
		</h3>
		<p class="text-sm text-gray-600 line-clamp-2 min-h-[40px]">
			{template.description}
		</p>
	</div>

	<!-- Category & Match Score -->
	<div class="flex items-center gap-2 mb-4">
		<span
			class="inline-flex items-center px-2 py-1 text-xs font-medium rounded border {categoryColors[
				template.category
			]}"
		>
			{categoryLabels[template.category]}
		</span>

		{#if matchScore !== undefined}
			<div class="flex items-center gap-1 text-xs text-amber-600 font-medium">
				<Star class="w-3 h-3 fill-amber-400" />
				<span>{matchScore}% match</span>
			</div>
		{/if}
	</div>

	<!-- Features -->
	{#if template.features && template.features.length > 0}
		<div class="mb-4">
			<div class="flex flex-wrap gap-1.5">
				{#each template.features.slice(0, 3) as feature}
					<span class="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded">
						{feature}
					</span>
				{/each}
				{#if template.features.length > 3}
					<span class="px-2 py-0.5 text-xs text-gray-400">
						+{template.features.length - 3} more
					</span>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Footer -->
	<div class="flex items-center justify-between pt-4 border-t border-gray-100">
		<div class="flex items-center gap-1 text-xs text-gray-500">
			<Star class="w-3 h-3" />
			<span>{formatPopularity(template.popularity_score)}</span>
		</div>

		<div class="flex items-center gap-2">
			{#if onPreview}
				<button
					onclick={handlePreview}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					Preview
				</button>
			{/if}
			{#if onUse}
				<button
					onclick={handleUse}
					class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-1.5 group-hover:shadow-md"
				>
					<span>Use Template</span>
					<ArrowRight class="w-3.5 h-3.5" />
				</button>
			{/if}
		</div>
	</div>
</div>
