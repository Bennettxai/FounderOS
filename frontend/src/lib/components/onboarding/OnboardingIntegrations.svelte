<!--
  OnboardingIntegrations.svelte
  Integration connection phase of the onboarding flow.
  Shows recommended and other available integrations with OAuth connect/disconnect.
-->
<script lang="ts">
	import IntegrationCard from './IntegrationCard.svelte';
	import type { IntegrationDef } from './onboardingTypes.ts';

	interface Props {
		integrations: IntegrationDef[];
		recommendedIntegrations: string[];
		integrationStatuses: Record<string, 'disconnected' | 'connecting' | 'connected' | 'error'>;
		selectedIntegrations: string[];
		allRecommendedConnected: boolean;
		onConnect: (id: string) => void;
		onDisconnect: (id: string) => void;
		onConnectAll: () => void;
		onComplete: () => void;
	}

	let {
		integrations,
		recommendedIntegrations,
		integrationStatuses,
		selectedIntegrations,
		allRecommendedConnected,
		onConnect,
		onDisconnect,
		onConnectAll,
		onComplete
	}: Props = $props();

	let recommended = $derived(integrations.filter(i => recommendedIntegrations.includes(i.id)));
	let others = $derived(integrations.filter(i => !recommendedIntegrations.includes(i.id)));
</script>

<div class="integrations-layout">
	<div class="integrations-container">
		<h2 class="section-title">Connect your tools</h2>
		<p class="section-subtitle">
			Connect your favorite tools and we'll sync your data automatically.
		</p>

		{#if recommended.length > 0}
			<div class="integrations-section">
				<div class="section-header">
					<h3 class="section-label">Recommended for you</h3>
					<button
						class="btn-pill btn-pill-ghost connect-all-btn"
						onclick={onConnectAll}
						disabled={allRecommendedConnected}
					>
						{allRecommendedConnected ? 'All connected' : 'Connect all'}
					</button>
				</div>
				<div class="integrations-grid">
					{#each recommended as integration (integration.id)}
						<div class="integration-wrapper recommended">
							<IntegrationCard
								name={integration.name}
								icon={integration.icon}
								status={integrationStatuses[integration.id] || 'disconnected'}
								onConnect={() => onConnect(integration.id)}
								onDisconnect={() => onDisconnect(integration.id)}
							/>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div class="integrations-section">
			<h3 class="section-label">
				{recommended.length > 0 ? 'Other integrations' : 'Available integrations'}
			</h3>
			<div class="integrations-grid">
				{#each others as integration (integration.id)}
					<div class="integration-wrapper">
						<IntegrationCard
							name={integration.name}
							icon={integration.icon}
							status={integrationStatuses[integration.id] || 'disconnected'}
							onConnect={() => onConnect(integration.id)}
							onDisconnect={() => onDisconnect(integration.id)}
						/>
					</div>
				{/each}
			</div>
		</div>

		<button class="btn-pill btn-pill-ghost continue-btn" onclick={onComplete}>
			{selectedIntegrations.length > 0 ? 'Continue' : "I'll do this later"}
		</button>
	</div>
</div>

<style>
	.integrations-layout {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
	}

	.integrations-container {
		display: flex;
		flex-direction: column;
		gap: 24px;
		max-width: 500px;
		width: 100%;
	}

	.section-title {
		font-size: 24px;
		font-weight: 600;
		color: var(--foreground, #1f2937);
		margin: 0;
		text-align: center;
	}

	.section-subtitle {
		font-size: 15px;
		color: var(--muted-foreground, #6b7280);
		margin: 0;
		text-align: center;
	}

	.integrations-section {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.section-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--muted-foreground, #6b7280);
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin: 0;
	}

	.connect-all-btn {
		padding: 6px 12px;
		font-size: 12px;
		font-weight: 500;
		border: 1px solid var(--primary, #6366f1);
		border-radius: 16px;
		background: transparent;
		color: var(--primary, #6366f1);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.connect-all-btn:hover:not(:disabled) {
		background: var(--primary, #6366f1);
		color: white;
	}

	.connect-all-btn:disabled {
		opacity: 0.5;
		cursor: default;
		border-color: var(--success, #10b981);
		color: var(--success, #10b981);
	}

	.integrations-grid {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.integration-wrapper {
		position: relative;
		transition: transform 0.2s ease, box-shadow 0.2s ease;
	}

	.integration-wrapper:hover {
		transform: translateY(-2px);
	}

	.integration-wrapper.recommended {
		order: -1;
	}

	.continue-btn {
		margin-top: 8px;
		padding: 14px 28px;
		font-size: 15px;
		font-weight: 500;
		border: none;
		border-radius: 24px;
		background-color: var(--primary, #6366f1);
		color: white;
		cursor: pointer;
		transition: opacity 0.2s, transform 0.2s;
		align-self: center;
	}

	.continue-btn:hover {
		opacity: 0.9;
		transform: translateY(-1px);
	}

	:global(.dark) .section-title {
		color: var(--foreground, #f9fafb);
	}
</style>
