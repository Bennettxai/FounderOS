<script lang="ts">
	import { onMount } from 'svelte';
	import { AlertCircle, CheckCircle2, ExternalLink, FileUp, Instagram, Plug, Youtube } from 'lucide-svelte';
	import { initiateAuth, getIntegrationStatus } from '$lib/api/integrations/integrations';
	import type { IntegrationProvider } from '$lib/api/integrations/types';

	type ConnectorStatus = 'Connected' | 'Not connected' | 'Missing setup' | 'Coming soon' | 'Checking...';

	type Connector = {
		name: string;
		provider: IntegrationProvider;
		description: string;
		status: ConnectorStatus;
		icon: typeof Instagram;
		enabled: boolean;
		accountName?: string;
		error?: string;
	};

	let loadingProvider = $state<IntegrationProvider | null>(null);

	let socialConnectors = $state<Connector[]>([
		{
			name: 'Instagram',
			provider: 'instagram',
			description: 'Connect Instagram so ContentOS can audit your profile, read Reels, track post links, and bring performance data back into the pipeline.',
			status: 'Checking...',
			icon: Instagram,
			enabled: true
		},
		{
			name: 'TikTok',
			provider: 'tiktok',
			description: 'Connect TikTok so short-form performance and post analytics can flow back into ContentOS.',
			status: 'Coming soon',
			icon: Plug,
			enabled: false
		},
		{
			name: 'YouTube',
			provider: 'youtube',
			description: 'Connect YouTube and Shorts for views, retention, comments, and video performance reporting.',
			status: 'Coming soon',
			icon: Youtube,
			enabled: false
		}
	]);

	const insightPaths = [
		{
			title: 'Best long-term path',
			label: 'Official Meta connector',
			status: 'Needs Meta app setup',
			body: 'Use this for the real product: connect an Instagram professional account through Meta OAuth, then pull account media, Reels, publishing status, and private owner insights through the Graph API.'
		},
		{
			title: 'Best right-now path',
			label: 'Owner insights import',
			status: 'Ready now',
			body: 'Use Instagram Professional Dashboard, Meta Business Suite, or a third-party report export, then import the missing metrics into ContentOS: reach, saves, shares, profile visits, follows, audience quality, and retention notes.'
		},
		{
			title: 'Best research path',
			label: 'Third-party analytics',
			status: 'Use for audience and competitors',
			body: 'Use tools like HypeAuditor, Modash, Socialinsider, Metricool, Later, or Sprout for demographics, audience quality, market comparison, and competitor benchmarking while the official connector is being finished.'
		}
	];

	onMount(() => {
		void refreshInstagramStatus();
	});

	async function refreshInstagramStatus() {
		try {
			const status = await getIntegrationStatus('instagram');
			updateConnector('instagram', {
				status: status.connected ? 'Connected' : 'Not connected',
				accountName: status.account_name,
				error: undefined
			});
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Unable to check Instagram status';
			updateConnector('instagram', {
				status: message.includes('404') ? 'Missing setup' : 'Not connected',
				error: message
			});
		}
	}

	function updateConnector(provider: IntegrationProvider, patch: Partial<Connector>) {
		socialConnectors = socialConnectors.map((connector) =>
			connector.provider === provider ? { ...connector, ...patch } : connector
		);
	}

	async function connectProvider(provider: IntegrationProvider) {
		loadingProvider = provider;
		try {
			const response = await initiateAuth(provider);
			if (response.auth_url) {
				window.open(response.auth_url, '_blank', 'noopener,noreferrer');
			}
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Unable to start connection';
			updateConnector(provider, {
				status: message.includes('412') ? 'Missing setup' : 'Not connected',
				error: message
			});
		} finally {
			loadingProvider = null;
		}
	}
</script>

<svelte:head><title>Connectors - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Plug size={22} strokeWidth={1.8} /></div>
		<div>
			<h1 class="page-title">Connectors</h1>
			<p class="page-desc">Integrations with external tools - CRMs, calendars, payment processors, and more.</p>
		</div>
	</header>

	<section class="connector-section">
		<div class="section-head">
			<h2>Social accounts</h2>
			<p>Start here for ContentOS publishing, profile data, and performance insights.</p>
		</div>
		<div class="connector-grid">
			{#each socialConnectors as connector}
				<article class="connector-card">
					<div class="connector-top">
						<div class="connector-icon">
							<svelte:component this={connector.icon} size={20} strokeWidth={1.8} />
						</div>
						<span class:connected={connector.status === 'Connected'} class:setup={connector.status === 'Missing setup'}>
							{#if connector.status === 'Connected'}
								<CheckCircle2 size={13} />
							{:else if connector.status === 'Missing setup'}
								<AlertCircle size={13} />
							{/if}
							{connector.status}
						</span>
					</div>
					<h3>{connector.name}</h3>
					{#if connector.accountName}
						<div class="account-line">@{connector.accountName}</div>
					{/if}
					<p>{connector.description}</p>
					{#if connector.error}
						<div class="connector-note">{connector.error}</div>
					{/if}
					<button
						class="connect-btn"
						disabled={!connector.enabled || loadingProvider === connector.provider || connector.status === 'Connected'}
						title={connector.enabled ? `Connect ${connector.name}` : `${connector.name} is not wired yet`}
						onclick={() => connectProvider(connector.provider)}
					>
						<ExternalLink size={15} />
						{#if loadingProvider === connector.provider}
							Opening...
						{:else if connector.status === 'Connected'}
							Connected
						{:else if connector.status === 'Missing setup'}
							Setup required
						{:else}
							Connect {connector.name}
						{/if}
					</button>
				</article>
			{/each}
		</div>
	</section>

	<section class="connector-section">
		<div class="section-head">
			<h2>Instagram insights plan</h2>
			<p>The best setup is not one button. BusinessOS should use the official Meta connector for scale, and use imports/third-party reports until that connector has approved credentials and permissions.</p>
		</div>
		<div class="path-grid">
			{#each insightPaths as path}
				<article class="path-card">
					<div class="path-top">
						<span>{path.title}</span>
						<strong>{path.status}</strong>
					</div>
					<h3>{path.label}</h3>
					<p>{path.body}</p>
				</article>
			{/each}
		</div>
		<div class="import-panel">
			<div>
				<h3>Owner insights import template</h3>
				<p>Use this when private metrics are available from Instagram or a third-party report before the Meta connector is fully configured.</p>
			</div>
			<a class="connect-btn" href="/instagram-owner-insights-import-template.csv" target="_blank" rel="noreferrer">
				<FileUp size={15} />
				Open import template
			</a>
		</div>
	</section>
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 24px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }
	.connector-section { display: flex; flex-direction: column; gap: 14px; }
	.section-head h2 { margin: 0; color: var(--dt); font-size: 1rem; font-weight: 700; letter-spacing: 0; }
	.section-head p { margin: 4px 0 0; color: var(--dt3); font-size: 0.84rem; }
	.connector-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 12px; }
	.connector-card { border: 1px solid var(--dbd); border-radius: 8px; padding: 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.connector-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 14px; }
	.connector-icon { width: 36px; height: 36px; border-radius: 8px; display: inline-flex; align-items: center; justify-content: center; color: var(--dt); background: color-mix(in srgb, var(--dt) 7%, transparent); }
	.connector-top span { color: var(--dt3); font-size: 0.72rem; font-weight: 700; display: inline-flex; align-items: center; gap: 5px; }
	.connector-top span.connected { color: #15803d; }
	.connector-top span.setup { color: #b45309; }
	.connector-card h3 { margin: 0; color: var(--dt); font-size: 0.98rem; font-weight: 720; letter-spacing: 0; }
	.account-line { margin-top: 3px; color: var(--dt2); font-size: 0.78rem; font-weight: 650; }
	.connector-card p { min-height: 58px; margin: 6px 0 14px; color: var(--dt3); font-size: 0.82rem; line-height: 1.45; }
	.connector-note { margin: 0 0 12px; padding: 9px 10px; border-radius: 8px; border: 1px solid color-mix(in srgb, #f59e0b 35%, transparent); background: color-mix(in srgb, #f59e0b 9%, transparent); color: #92400e; font-size: 0.76rem; line-height: 1.35; }
	.connect-btn { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 7px 12px; border-radius: 8px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); font: inherit; font-size: 0.8rem; font-weight: 650; }
	.connect-btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.path-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
	.path-card { border: 1px solid var(--dbd); border-radius: 8px; padding: 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.path-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 12px; }
	.path-top span { color: var(--dt3); font-size: 0.72rem; font-weight: 750; text-transform: uppercase; letter-spacing: 0.04em; }
	.path-top strong { color: #0f766e; font-size: 0.7rem; font-weight: 760; text-align: right; }
	.path-card h3 { margin: 0; color: var(--dt); font-size: 0.95rem; font-weight: 720; letter-spacing: 0; }
	.path-card p { margin: 7px 0 0; color: var(--dt2); font-size: 0.8rem; line-height: 1.45; }
	.import-panel { display: flex; align-items: center; justify-content: space-between; gap: 14px; border: 1px solid color-mix(in srgb, #0f766e 20%, var(--dbd)); border-radius: 8px; padding: 14px; background: color-mix(in srgb, #0f766e 5%, transparent); }
	.import-panel h3 { margin: 0; color: var(--dt); font-size: 0.94rem; letter-spacing: 0; }
	.import-panel p { margin: 4px 0 0; color: var(--dt3); font-size: 0.8rem; line-height: 1.4; }
	@media (max-width: 768px) { .page { padding: 16px 18px; } }
	@media (max-width: 900px) { .path-grid { grid-template-columns: 1fr; } .import-panel { align-items: flex-start; flex-direction: column; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
