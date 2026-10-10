<script lang="ts">
	import { learning } from '$lib/stores/learning';
	import type { PersonalizationProfile, DetectedPattern } from '$lib/api/learning';

	let personalizationProfile = $state<PersonalizationProfile | null>(null);
	let detectedPatterns = $state<DetectedPattern[]>([]);
	let isLoadingPersonalization = $state(false);
	let isSavingPersonalization = $state(false);
	let saveMessage = $state('');

	$effect(() => {
		loadPersonalizationData();
	});

	async function loadPersonalizationData() {
		isLoadingPersonalization = true;
		try {
			const [profile, patterns] = await Promise.all([
				learning.loadProfile(),
				learning.detectPatterns().catch(() => [] as DetectedPattern[])
			]);
			personalizationProfile = profile;
			detectedPatterns = patterns;
		} catch (error) {
			console.error('Error loading personalization data:', error);
			personalizationProfile = {
				user_id: '',
				preferred_tone: 'professional',
				preferred_verbosity: 'balanced',
				preferred_format: 'structured',
				prefers_examples: true,
				prefers_analogies: false,
				prefers_code_samples: false,
				prefers_visual_aids: false,
				expertise_areas: [],
				learning_areas: [],
				common_topics: [],
				most_active_hours: [],
				total_conversations: 0,
				total_feedback_given: 0,
				positive_feedback_ratio: 0.5,
				profile_completeness: 0
			};
			detectedPatterns = [];
		} finally {
			isLoadingPersonalization = false;
		}
	}

	async function savePersonalizationProfile() {
		if (!personalizationProfile) return;
		isSavingPersonalization = true;
		try {
			await learning.updateProfile(personalizationProfile);
			saveMessage = 'Personalization settings saved!';
			setTimeout(() => (saveMessage = ''), 2000);
		} catch (error) {
			console.error('Error saving personalization:', error);
			saveMessage = 'Error saving personalization settings';
		} finally {
			isSavingPersonalization = false;
		}
	}
</script>

<div class="space-y-4">
	{#if isLoadingPersonalization}
		<div class="flex items-center justify-center py-12">
			<div class="animate-spin h-8 w-8 border-2 st-spinner border-t-transparent rounded-full"></div>
		</div>
	{:else}
		{#if saveMessage}
			<div
				class="p-3 rounded-lg text-sm"
				style="{saveMessage.includes('Error')
					? 'background: var(--bos-status-error-bg); color: var(--bos-status-error);'
					: 'background: var(--bos-status-success-bg); color: var(--bos-status-success);'}"
			>
				{saveMessage}
			</div>
		{/if}

		<!-- AI Response Preferences -->
		<div class="card">
			<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-2">Response Preferences</h2>
			<p class="text-sm st-muted mb-6">
				Customize how the AI responds to you. These preferences help personalize your experience.
			</p>

			<div class="space-y-4">
				<!-- Tone -->
				<div>
					<span class="block text-sm font-medium st-label mb-2">Preferred Tone</span>
					<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
						{#each ['formal', 'professional', 'casual', 'friendly'] as tone}
							<button
								onclick={() => personalizationProfile && (personalizationProfile.preferred_tone = tone as PersonalizationProfile['preferred_tone'])}
								class="p-2.5 rounded-lg border text-sm font-medium transition-colors {personalizationProfile?.preferred_tone === tone
									? 'st-opt-selected'
									: 'st-opt'}"
							>
								{tone.charAt(0).toUpperCase() + tone.slice(1)}
							</button>
						{/each}
					</div>
				</div>

				<!-- Verbosity -->
				<div>
					<span class="block text-sm font-medium st-label mb-2">Response Length</span>
					<div class="grid grid-cols-3 gap-3">
						{#each ['concise', 'balanced', 'detailed'] as verbosity}
							<button
								onclick={() => personalizationProfile && (personalizationProfile.preferred_verbosity = verbosity as PersonalizationProfile['preferred_verbosity'])}
								class="p-2.5 rounded-lg border text-sm font-medium transition-colors {personalizationProfile?.preferred_verbosity === verbosity
									? 'st-opt-selected'
									: 'st-opt'}"
							>
								{verbosity.charAt(0).toUpperCase() + verbosity.slice(1)}
							</button>
						{/each}
					</div>
				</div>

				<!-- Format -->
				<div>
					<span class="block text-sm font-medium st-label mb-2">Response Format</span>
					<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
						{#each ['prose', 'bullets', 'structured', 'mixed'] as format}
							<button
								onclick={() => personalizationProfile && (personalizationProfile.preferred_format = format as PersonalizationProfile['preferred_format'])}
								class="p-2.5 rounded-lg border text-sm font-medium transition-colors {personalizationProfile?.preferred_format === format
									? 'st-opt-selected'
									: 'st-opt'}"
							>
								{format.charAt(0).toUpperCase() + format.slice(1)}
							</button>
						{/each}
					</div>
				</div>

				<!-- Toggles -->
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<div>
							<p class="font-medium st-title">Include Examples</p>
							<p class="text-sm st-muted">AI will include relevant examples in responses</p>
						</div>
						<button
							aria-label="Toggle include examples"
							onclick={() => personalizationProfile && (personalizationProfile.prefers_examples = !personalizationProfile.prefers_examples)}
							class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {personalizationProfile?.prefers_examples
								? 'st-toggle-on'
								: 'st-toggle-off'}"
						>
							<span class="inline-block h-4 w-4 transform rounded-full transition-transform {personalizationProfile?.prefers_examples
								? 'translate-x-6 st-toggle-knob'
								: 'translate-x-1 st-toggle-knob'}"></span>
						</button>
					</div>

					<div class="flex items-center justify-between">
						<div>
							<p class="font-medium st-title">Include Code Samples</p>
							<p class="text-sm st-muted">AI will include code snippets when relevant</p>
						</div>
						<button
							aria-label="Toggle include code samples"
							onclick={() => personalizationProfile && (personalizationProfile.prefers_code_samples = !personalizationProfile.prefers_code_samples)}
							class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {personalizationProfile?.prefers_code_samples
								? 'st-toggle-on'
								: 'st-toggle-off'}"
						>
							<span class="inline-block h-4 w-4 transform rounded-full transition-transform {personalizationProfile?.prefers_code_samples
								? 'translate-x-6 st-toggle-knob'
								: 'translate-x-1 st-toggle-knob'}"></span>
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Detected Patterns -->
		{#if detectedPatterns.length > 0}
			<div class="card">
				<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-2">Detected Patterns</h2>
				<p class="text-sm st-muted mb-4">
					The AI has learned these patterns from your interactions.
				</p>
				<div class="space-y-3">
					{#each detectedPatterns as pattern}
						<div class="flex items-center justify-between p-3 rounded-lg st-pattern-card">
							<div>
								<p class="font-medium st-title text-sm">{pattern.pattern_key}</p>
								<p class="text-xs st-muted">{pattern.pattern_value}</p>
							</div>
							<div class="flex items-center gap-2">
								<span class="text-xs st-icon">{pattern.observation_count} observations</span>
								<span
								class="px-2 py-1 text-xs font-medium rounded-full {pattern.confidence_score < 0.5 ? 'st-low-confidence' : ''}"
								style="{pattern.confidence_score >= 0.8
									? 'background: var(--bos-status-success-bg); color: var(--bos-status-success);'
									: pattern.confidence_score >= 0.5
									? 'background: var(--bos-status-warning-bg); color: var(--bos-status-warning);'
									: ''}"
							>
									{Math.round(pattern.confidence_score * 100)}%
								</span>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Expertise & Learning Areas -->
		<div class="card">
			<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Knowledge Areas</h2>
			<div class="space-y-4">
				<div>
					<span class="block text-sm font-medium st-label mb-2">Your Expertise</span>
					<p class="text-sm st-muted mb-2">Areas where you have strong knowledge</p>
					<div class="flex flex-wrap gap-2">
						{#each personalizationProfile?.expertise_areas || [] as area}
							<span class="px-3 py-1 rounded-full text-sm" style="background: var(--bos-status-info-bg); color: var(--bos-status-info)">
								{area}
							</span>
						{/each}
						{#if !personalizationProfile?.expertise_areas?.length}
							<span class="text-sm st-icon italic">No expertise areas detected yet</span>
						{/if}
					</div>
				</div>

				<div>
					<span class="block text-sm font-medium st-label mb-2">Learning Interests</span>
					<p class="text-sm st-muted mb-2">Topics you're actively learning about</p>
					<div class="flex flex-wrap gap-2">
						{#each personalizationProfile?.learning_areas || [] as area}
							<span class="px-3 py-1 rounded-full text-sm" style="background: color-mix(in srgb, var(--bos-category-ai) 12%, transparent); color: var(--bos-category-ai)">
								{area}
							</span>
						{/each}
						{#if !personalizationProfile?.learning_areas?.length}
							<span class="text-sm st-icon italic">No learning areas detected yet</span>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Save Button -->
		<div class="flex justify-end">
			<button
				onclick={savePersonalizationProfile}
				disabled={isSavingPersonalization || !personalizationProfile}
				class="btn-pill btn-pill-ghost btn btn-primary"
			>
				{#if isSavingPersonalization}
					<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Saving...
				{:else}
					Save Preferences
				{/if}
			</button>
		</div>
	{/if}
</div>

<style>
	.st-title { color: var(--dt); }
	.st-muted { color: var(--dt3); }
	.st-label { color: var(--dt2); }
	.st-icon  { color: var(--dt4); }
	.st-spinner { border-color: var(--dt); }
	.st-toggle-on  { background: var(--dt); }
	.st-toggle-off { background: var(--dbg3); }
	.st-toggle-knob { background: var(--dbg); }
	.st-opt-selected {
		border-color: var(--dt);
		background: var(--bos-settings-card-bg);
		color: var(--dt);
	}
	.st-opt {
		border-color: var(--dbd);
		color: var(--dt2);
	}
	.st-opt:hover { border-color: var(--dt4); }
	.st-pattern-card {
		background: var(--bos-settings-card-bg);
		border: 1px solid var(--bos-settings-card-border);
		border-radius: var(--bos-settings-card-radius);
	}
	.st-low-confidence {
		background: var(--dbg3);
		color: var(--dt2);
	}
</style>
