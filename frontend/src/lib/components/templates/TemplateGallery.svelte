<script lang="ts">
	import { Search, Filter, X, Sparkles } from 'lucide-svelte';
	import TemplateCard from './TemplateCard.svelte';
	import LoadingSpinner from '../ui/LoadingSpinner.svelte';
	import SkeletonLoader from '../ui/SkeletonLoader.svelte';
	import type { AppTemplate, TemplateCategory, BusinessType, TeamSize } from '$lib/api/templates';

	interface Props {
		templates: AppTemplate[];
		loading?: boolean;
		error?: string | null;
		showRecommendations?: boolean;
		onUseTemplate?: (templateId: string) => void;
		onPreviewTemplate?: (templateId: string) => void;
		onFilterChange?: (filters: {
			category?: TemplateCategory | null;
			business_type?: BusinessType | null;
			team_size?: TeamSize | null;
			search?: string;
			sort?: 'popular' | 'newest' | 'name';
		}) => void;
	}

	let {
		templates,
		loading = false,
		error = null,
		showRecommendations = false,
		onUseTemplate,
		onPreviewTemplate,
		onFilterChange
	}: Props = $props();

	// Local state
	let searchQuery = $state('');
	let selectedCategory = $state<TemplateCategory | null>(null);
	let selectedBusinessType = $state<BusinessType | null>(null);
	let selectedTeamSize = $state<TeamSize | null>(null);
	let selectedSort = $state<'popular' | 'newest' | 'name'>('popular');
	let showFilters = $state(false);

	// Categories
	const categories: { value: TemplateCategory; label: string }[] = [
		{ value: 'crm', label: 'CRM' },
		{ value: 'project_management', label: 'Project Management' },
		{ value: 'hr', label: 'HR' },
		{ value: 'finance', label: 'Finance' },
		{ value: 'marketing', label: 'Marketing' },
		{ value: 'operations', label: 'Operations' },
		{ value: 'custom', label: 'Custom' }
	];

	const businessTypes: { value: BusinessType; label: string }[] = [
		{ value: 'startup', label: 'Startup' },
		{ value: 'small_business', label: 'Small Business' },
		{ value: 'enterprise', label: 'Enterprise' },
		{ value: 'agency', label: 'Agency' },
		{ value: 'consulting', label: 'Consulting' },
		{ value: 'ecommerce', label: 'E-commerce' },
		{ value: 'saas', label: 'SaaS' },
		{ value: 'nonprofit', label: 'Nonprofit' }
	];

	const teamSizes: { value: TeamSize; label: string }[] = [
		{ value: 'solo', label: 'Solo' },
		{ value: 'small', label: 'Small (2-10)' },
		{ value: 'medium', label: 'Medium (11-50)' },
		{ value: 'large', label: 'Large (51+)' }
	];

	const sortOptions: { value: 'popular' | 'newest' | 'name'; label: string }[] = [
		{ value: 'popular', label: 'Most Popular' },
		{ value: 'newest', label: 'Newest' },
		{ value: 'name', label: 'Name' }
	];

	// Active filters count
	const activeFiltersCount = $derived(
		[selectedCategory, selectedBusinessType, selectedTeamSize].filter(Boolean).length
	);

	// Filtered templates (client-side search)
	const filteredTemplates = $derived(() => {
		if (!searchQuery) return templates;
		const query = searchQuery.toLowerCase();
		return templates.filter(
			(t) =>
				t.name.toLowerCase().includes(query) ||
				t.description.toLowerCase().includes(query) ||
				t.features.some((f) => f.toLowerCase().includes(query))
		);
	});

	function handleSearchChange(e: Event) {
		const target = e.target as HTMLInputElement;
		searchQuery = target.value;
		notifyFilterChange();
	}

	function handleCategoryChange(category: TemplateCategory | null) {
		selectedCategory = category;
		notifyFilterChange();
	}

	function handleBusinessTypeChange(type: BusinessType | null) {
		selectedBusinessType = type;
		notifyFilterChange();
	}

	function handleTeamSizeChange(size: TeamSize | null) {
		selectedTeamSize = size;
		notifyFilterChange();
	}

	function handleSortChange(sort: 'popular' | 'newest' | 'name') {
		selectedSort = sort;
		notifyFilterChange();
	}

	function clearFilters() {
		selectedCategory = null;
		selectedBusinessType = null;
		selectedTeamSize = null;
		searchQuery = '';
		notifyFilterChange();
	}

	function notifyFilterChange() {
		onFilterChange?.({
			category: selectedCategory,
			business_type: selectedBusinessType,
			team_size: selectedTeamSize,
			search: searchQuery,
			sort: selectedSort
		});
	}
</script>

<div class="flex flex-col h-full">
	<!-- Header with Search -->
	<div class="px-6 py-4 border-b border-gray-200 bg-white">
		<div class="flex items-center gap-3">
			<!-- Search -->
			<div class="flex-1 relative">
				<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
				<input
					type="text"
					placeholder="Search templates..."
					value={searchQuery}
					oninput={handleSearchChange}
					class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
				/>
			</div>

			<!-- Sort -->
			<select
				value={selectedSort}
				onchange={(e) => handleSortChange(e.currentTarget.value as any)}
				class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
			>
				{#each sortOptions as option}
					<option value={option.value}>{option.label}</option>
				{/each}
			</select>

			<!-- Filter Toggle -->
			<button
				onclick={() => (showFilters = !showFilters)}
				class="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors relative"
			>
				<Filter class="w-4 h-4" />
				<span>Filters</span>
				{#if activeFiltersCount > 0}
					<span
						class="absolute -top-1 -right-1 w-5 h-5 bg-gray-900 text-white text-xs rounded-full flex items-center justify-center"
					>
						{activeFiltersCount}
					</span>
				{/if}
			</button>
		</div>

		<!-- Recommendations Badge -->
		{#if showRecommendations}
			<div class="mt-3 flex items-center gap-2 text-sm text-amber-700 bg-amber-50 px-3 py-2 rounded-lg">
				<Sparkles class="w-4 h-4" />
				<span>Showing personalized recommendations based on your profile</span>
			</div>
		{/if}
	</div>

	<!-- Filters Panel -->
	{#if showFilters}
		<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
			<div class="space-y-4">
				<!-- Category Filter -->
				<div>
					<label class="block text-xs font-medium text-gray-700 mb-2">Category</label>
					<div class="flex flex-wrap gap-2">
						<button
							onclick={() => handleCategoryChange(null)}
							class="px-3 py-1.5 text-sm rounded-lg border transition-colors {selectedCategory === null
								? 'bg-gray-900 text-white border-gray-900'
								: 'bg-white border-gray-300 hover:border-gray-400'}"
						>
							All
						</button>
						{#each categories as cat}
							<button
								onclick={() => handleCategoryChange(cat.value)}
								class="px-3 py-1.5 text-sm rounded-lg border transition-colors {selectedCategory ===
								cat.value
									? 'bg-gray-900 text-white border-gray-900'
									: 'bg-white border-gray-300 hover:border-gray-400'}"
							>
								{cat.label}
							</button>
						{/each}
					</div>
				</div>

				<!-- Business Type Filter -->
				<div>
					<label class="block text-xs font-medium text-gray-700 mb-2">Business Type</label>
					<div class="flex flex-wrap gap-2">
						<button
							onclick={() => handleBusinessTypeChange(null)}
							class="px-3 py-1.5 text-sm rounded-lg border transition-colors {selectedBusinessType ===
							null
								? 'bg-gray-900 text-white border-gray-900'
								: 'bg-white border-gray-300 hover:border-gray-400'}"
						>
							All
						</button>
						{#each businessTypes as type}
							<button
								onclick={() => handleBusinessTypeChange(type.value)}
								class="px-3 py-1.5 text-sm rounded-lg border transition-colors {selectedBusinessType ===
								type.value
									? 'bg-gray-900 text-white border-gray-900'
									: 'bg-white border-gray-300 hover:border-gray-400'}"
							>
								{type.label}
							</button>
						{/each}
					</div>
				</div>

				<!-- Team Size Filter -->
				<div>
					<label class="block text-xs font-medium text-gray-700 mb-2">Team Size</label>
					<div class="flex flex-wrap gap-2">
						<button
							onclick={() => handleTeamSizeChange(null)}
							class="px-3 py-1.5 text-sm rounded-lg border transition-colors {selectedTeamSize === null
								? 'bg-gray-900 text-white border-gray-900'
								: 'bg-white border-gray-300 hover:border-gray-400'}"
						>
							All
						</button>
						{#each teamSizes as size}
							<button
								onclick={() => handleTeamSizeChange(size.value)}
								class="px-3 py-1.5 text-sm rounded-lg border transition-colors {selectedTeamSize ===
								size.value
									? 'bg-gray-900 text-white border-gray-900'
									: 'bg-white border-gray-300 hover:border-gray-400'}"
							>
								{size.label}
							</button>
						{/each}
					</div>
				</div>

				<!-- Clear Filters -->
				{#if activeFiltersCount > 0}
					<div class="flex items-center justify-end">
						<button
							onclick={clearFilters}
							class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-1.5"
						>
							<X class="w-4 h-4" />
							<span>Clear Filters</span>
						</button>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Error State -->
	{#if error}
		<div class="mx-6 mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
			<p class="text-sm text-red-700">{error}</p>
		</div>
	{/if}

	<!-- Loading State -->
	{#if loading && templates.length === 0}
		<div class="flex-1 p-6">
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				<SkeletonLoader variant="card" count={6} />
			</div>
		</div>
	{:else if templates.length === 0 && !loading}
		<!-- Empty State -->
		<div class="flex-1 flex items-center justify-center p-6">
			<div class="flex flex-col items-center gap-3 text-gray-500 max-w-md text-center">
				<div
					class="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center text-gray-400"
				>
					<Search class="w-8 h-8" />
				</div>
				<p class="text-lg font-medium text-gray-900">No templates found</p>
				<p class="text-sm">
					{#if searchQuery || activeFiltersCount > 0}
						Try adjusting your search or filters to find what you're looking for.
					{:else}
						No templates available at the moment. Check back later!
					{/if}
				</p>
				{#if searchQuery || activeFiltersCount > 0}
					<button
						onclick={clearFilters}
						class="btn-pill btn-pill-primary btn-pill-sm mt-2"
					>
						Clear Filters
					</button>
				{/if}
			</div>
		</div>
	{:else}
		<!-- Template Grid -->
		<div class="flex-1 overflow-auto p-6">
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each filteredTemplates() as template (template.id)}
					<TemplateCard {template} onUse={onUseTemplate} onPreview={onPreviewTemplate} />
				{/each}
			</div>

			<!-- Loading More Indicator -->
			{#if loading && templates.length > 0}
				<div class="mt-6 flex justify-center">
					<LoadingSpinner message="Loading more templates..." />
				</div>
			{/if}
		</div>
	{/if}
</div>
