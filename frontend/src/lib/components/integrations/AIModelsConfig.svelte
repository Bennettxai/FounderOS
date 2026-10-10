<script lang="ts">
	import type { AIModelPreferences } from '$lib/api/integrations';

	interface Props {
		aiPreferences: AIModelPreferences | null;
		savingAiPrefs: boolean;
		aiPrefsMessage: string | null;
		aiPrefsError: string | null;
		onsave: (updates: Partial<AIModelPreferences>) => void;
	}

	let {
		aiPreferences,
		savingAiPrefs,
		aiPrefsMessage,
		aiPrefsError,
		onsave
	}: Props = $props();
</script>

<div class="ih-section">
	<h2 class="ih-section__title">AI Model Configuration</h2>
	<p class="ih-section__desc">
		Configure which AI models to use for different task tiers. The system automatically selects
		the appropriate tier based on task complexity.
	</p>

	{#if aiPreferences}
		<div class="ih-tier-list">
			<!-- Tier 2 -->
			<div class="ih-tier">
				<h3 class="ih-tier__name">Tier 2: Fast Tasks</h3>
				<p class="ih-tier__desc">
					Quick, low-complexity operations like formatting and simple lookups.
				</p>
				<div class="ih-tier__model">
					<span>
						{aiPreferences.tier_2_model.provider}: {aiPreferences.tier_2_model.model_id}
					</span>
				</div>
			</div>

			<!-- Tier 3 -->
			<div class="ih-tier">
				<h3 class="ih-tier__name">Tier 3: Standard Tasks</h3>
				<p class="ih-tier__desc">
					Medium-complexity tasks requiring analysis and synthesis.
				</p>
				<div class="ih-tier__model">
					<span>
						{aiPreferences.tier_3_model.provider}: {aiPreferences.tier_3_model.model_id}
					</span>
				</div>
			</div>

			<!-- Tier 4 -->
			<div class="ih-tier">
				<h3 class="ih-tier__name">Tier 4: Complex Tasks</h3>
				<p class="ih-tier__desc">
					High-complexity tasks requiring deep reasoning and multi-step analysis.
				</p>
				<div class="ih-tier__model">
					<span>
						{aiPreferences.tier_4_model.provider}: {aiPreferences.tier_4_model.model_id}
					</span>
				</div>
			</div>

			<!-- Settings -->
			<div class="ih-ai-settings">
				<h3 class="ih-ai-settings__title">Settings</h3>
				{#if aiPrefsMessage}
					<div class="ih-alert ih-alert--success ih-alert--sm">
						<p>{aiPrefsMessage}</p>
					</div>
				{/if}
				{#if aiPrefsError}
					<div class="ih-alert ih-alert--error ih-alert--sm">
						<p>{aiPrefsError}</p>
					</div>
				{/if}
				<div class="ih-ai-settings__list">
					<label class="ih-checkbox-label">
						<input
							type="checkbox"
							checked={aiPreferences.allow_model_upgrade_on_failure}
							onchange={(e) => {
								const target = e.currentTarget as HTMLInputElement;
								onsave({ allow_model_upgrade_on_failure: target.checked });
							}}
							disabled={savingAiPrefs}
							class="ih-checkbox"
						/>
						<span>Allow automatic model upgrade on failure</span>
					</label>
					<label class="ih-checkbox-label">
						<input
							type="checkbox"
							checked={aiPreferences.prefer_local}
							onchange={(e) => {
								const target = e.currentTarget as HTMLInputElement;
								onsave({ prefer_local: target.checked });
							}}
							disabled={savingAiPrefs}
							class="ih-checkbox"
						/>
						<span>Prefer local models when available</span>
					</label>
					<div class="ih-latency-row">
						<span>Max latency:</span>
						<input
							type="number"
							value={aiPreferences.max_latency_ms}
							onchange={(e) => {
								const target = e.currentTarget as HTMLInputElement;
								const val = parseInt(target.value, 10);
								if (!isNaN(val) && val > 0) {
									onsave({ max_latency_ms: val });
								}
							}}
							disabled={savingAiPrefs}
							class="ih-latency-input"
							min="100"
							step="100"
						/>
						<span class="ih-latency-unit">ms</span>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<p class="ih-empty__text">
			AI preferences not configured. Default settings will be used.
		</p>
	{/if}
</div>

<style>
	.ih-section {
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 0.75rem;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
	}
	.ih-section__title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt);
		margin-bottom: 0.25rem;
	}
	.ih-section__desc {
		font-size: 0.875rem;
		color: var(--dt3);
		margin-bottom: 1rem;
	}
	.ih-tier-list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.ih-tier {
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		padding: 1rem;
	}
	.ih-tier__name {
		font-weight: 600;
		color: var(--dt);
		text-transform: capitalize;
		margin-bottom: 0.25rem;
	}
	.ih-tier__desc {
		font-size: 0.875rem;
		color: var(--dt3);
		margin-bottom: 0.5rem;
	}
	.ih-tier__model {
		font-size: 0.75rem;
		font-family: monospace;
		color: var(--dt4);
	}
	.ih-ai-settings {
		margin-top: 1rem;
	}
	.ih-ai-settings__title {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--dt);
		margin-bottom: 0.75rem;
	}
	.ih-ai-settings__list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.ih-checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: var(--dt2);
		cursor: pointer;
	}
	.ih-checkbox {
		width: 1rem;
		height: 1rem;
		border-radius: 0.25rem;
		accent-color: #3b82f6;
	}
	.ih-latency-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
	}
	.ih-latency-input {
		width: 5rem;
		padding: 0.25rem 0.5rem;
		font-family: monospace;
		font-size: 0.875rem;
		color: #22c55e;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 0.375rem;
		outline: none;
		text-align: right;
	}
	.ih-latency-input:focus {
		border-color: #3b82f6;
	}
	.ih-latency-unit {
		font-size: 0.75rem;
		color: var(--dt4);
	}
	.ih-empty__text {
		color: var(--dt3);
	}
	.ih-alert {
		margin-top: 0.75rem;
		padding: 0.75rem;
		border-radius: 0.5rem;
		font-size: 0.875rem;
	}
	.ih-alert--error {
		background: rgba(239, 68, 68, 0.08);
		border: 1px solid rgba(239, 68, 68, 0.2);
		color: #ef4444;
	}
	.ih-alert--success {
		background: rgba(34, 197, 94, 0.08);
		border: 1px solid rgba(34, 197, 94, 0.2);
		color: #22c55e;
	}
	.ih-alert--sm {
		margin-top: 0.5rem;
		margin-bottom: 0.5rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.8125rem;
	}
</style>
