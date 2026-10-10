<script lang="ts">
	import { fade, fly } from 'svelte/transition';

	interface Computer {
		id: string;
		status: 'running' | 'hibernating' | 'stopped';
		domain: string;
		region: string;
		created_at: string;
	}

	interface Metrics {
		ram_used_gb: number;
		ram_total_gb: number;
		cpu_percent: number;
		cpu_cores: number;
		storage_used_gb: number;
		storage_total_gb: number;
	}

	interface Runtime {
		id: string;
		name: string;
		status: 'active' | 'idle' | 'stopped';
		memory_gb: number;
		uptime_seconds: number;
	}

	interface Subscription {
		plan_name: string;
		price_cents: number;
		interval: 'month' | 'year';
		credits_used: number;
		credits_total: number;
		seats: number;
		next_billing_date: string;
	}

	interface Props {
		computer: Computer;
		metrics: Metrics | null;
		runtimes: Runtime[];
		subscription: Subscription | null;
		stoppingRuntime: string | null;
		startingComputer: boolean;
		activeView: 'dashboard' | 'desktop';
		desktopSrc: string;
		onStartComputer: () => void;
		onSwitchToDashboard: () => void;
		onSwitchToDesktop: () => void;
		onLoadDesktopStream: () => void;
		onStopRuntime: (id: string) => void;
		onOpenPricing: () => void;
	}

	let {
		computer,
		metrics,
		runtimes,
		subscription,
		stoppingRuntime,
		startingComputer,
		activeView,
		desktopSrc,
		onStartComputer,
		onSwitchToDashboard,
		onSwitchToDesktop,
		onLoadDesktopStream,
		onStopRuntime,
		onOpenPricing,
	}: Props = $props();

	let ramPercent = $derived(
		metrics ? Math.round((metrics.ram_used_gb / metrics.ram_total_gb) * 100) : 0
	);
	let storagePercent = $derived(
		metrics && metrics.storage_total_gb > 0
			? Math.round((metrics.storage_used_gb / metrics.storage_total_gb) * 100)
			: 0
	);
	let creditsPercent = $derived(
		subscription
			? Math.round(((subscription.credits_total - subscription.credits_used) / subscription.credits_total) * 100)
			: 0
	);

	function formatUptime(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		return m > 0 ? `${h}h ${m}m` : `${h}h`;
	}

	function formatPrice(cents: number): string {
		return `$${(cents / 100).toFixed(0)}`;
	}

	function formatBillingDate(iso: string): string {
		try {
			return new Date(iso).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
		} catch {
			return iso;
		}
	}

	function formatDate(iso: string): string {
		try {
			return new Date(iso).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
		} catch {
			return iso;
		}
	}

	function calcUptime(createdAt: string): string {
		try {
			const seconds = Math.floor((Date.now() - new Date(createdAt).getTime()) / 1000);
			if (seconds < 60) return `${seconds}s`;
			if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
			const h = Math.floor(seconds / 3600);
			const m = Math.floor((seconds % 3600) / 60);
			if (h < 24) return m > 0 ? `${h}h ${m}m` : `${h}h`;
			const d = Math.floor(h / 24);
			const rh = h % 24;
			return rh > 0 ? `${d}d ${rh}h` : `${d}d`;
		} catch {
			return '—';
		}
	}

	function extractSlug(domain: string): string {
		try {
			const url = domain.startsWith('http') ? new URL(domain) : new URL(`https://${domain}`);
			return url.hostname.split('.')[0];
		} catch {
			return domain.split('.')[0] ?? domain;
		}
	}

	function creditsColor(pct: number): string {
		if (pct > 50) return '#22c55e';
		if (pct > 25) return '#f59e0b';
		return '#ef4444';
	}

	function statusLabel(status: Computer['status']): string {
		return { running: 'Running', hibernating: 'Hibernating', stopped: 'Stopped' }[status];
	}

	function runtimeStatusLabel(status: Runtime['status']): string {
		return { active: 'Active', idle: 'Idle', stopped: 'Stopped' }[status];
	}
</script>

<!-- Section 1: Computer Status Hero -->
<section class="cp-hero" aria-label="Computer status" in:fly={{ y: -8, duration: 280 }}>
	<div class="cp-hero-left">
		<div class="cp-status-indicator cp-status-indicator--{computer.status}" aria-label="Status: {statusLabel(computer.status)}"></div>
		<div class="cp-hero-info">
			<div class="cp-hero-title-row">
				<h1 class="cp-hero-title">Your Computer</h1>
				<span class="cp-status-badge cp-status-badge--{computer.status}">{statusLabel(computer.status)}</span>
			</div>
			<p class="cp-hero-domain">{computer.domain}</p>
		</div>
	</div>
	<div class="cp-hero-actions">
		{#if computer.status === 'stopped' || computer.status === 'hibernating'}
			<button
				class="cp-btn cp-btn--primary"
				disabled={startingComputer}
				onclick={onStartComputer}
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<polygon points="5 3 19 12 5 21 5 3"/>
				</svg>
				{startingComputer ? 'Starting...' : 'Start Computer'}
			</button>
		{:else}
			<a
				href="/terminal"
				class="cp-btn cp-btn--outline"
				aria-label="Open terminal"
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<polyline points="4 17 10 11 4 5"/>
					<line x1="12" y1="19" x2="20" y2="19"/>
				</svg>
				Open Terminal
			</a>
			<button
				class="cp-btn cp-btn--outline"
				aria-label="Open desktop"
				onclick={onSwitchToDesktop}
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<rect x="2" y="3" width="20" height="14" rx="2"/>
					<path d="M8 21h8M12 17v4"/>
				</svg>
				Open Desktop
			</button>
			<button class="cp-btn cp-btn--outline" aria-label="Restart computer">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<polyline points="23 4 23 10 17 10"/>
					<path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
				</svg>
				Restart
			</button>
			<button class="cp-btn cp-btn--outline" aria-label="Hibernate computer">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
				</svg>
				Hibernate
			</button>
		{/if}
	</div>
</section>

<!-- View Toggle: Dashboard / Desktop -->
{#if computer.status === 'running'}
	<div class="cp-view-tabs">
		<button class="cp-view-tab" class:active={activeView === 'dashboard'} onclick={onSwitchToDashboard}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>
			Dashboard
		</button>
		<button class="cp-view-tab" class:active={activeView === 'desktop'} onclick={onSwitchToDesktop}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
			Desktop
		</button>
	</div>
{/if}

{#if activeView === 'desktop' && computer.status === 'running'}
	<!-- Desktop VNC Stream -->
	<section class="cp-desktop-container">
		{#if desktopSrc}
			<iframe
				src={desktopSrc}
				title="BusinessOS Desktop"
				class="cp-desktop-iframe"
				allow="clipboard-read; clipboard-write; autoplay"
			></iframe>
		{:else}
			<div class="cp-desktop-loading">
				<div class="cp-desktop-spinner"></div>
				<p>Connecting to desktop...</p>
				<button class="cp-btn cp-btn--outline" style="margin-top: 12px;" onclick={onLoadDesktopStream}>
					Retry
				</button>
				{#if computer?.domain}
					<a href={computer.domain} target="_blank" class="cp-btn cp-btn--outline" style="margin-top: 8px;">
						Open in new tab
					</a>
				{/if}
			</div>
		{/if}
	</section>
{:else}

<!-- Section 2: Resource Meters -->
<section class="cp-metrics-grid" aria-label="Resource usage" in:fly={{ y: 12, duration: 300, delay: 50 }}>

	<!-- RAM -->
	<div class="cp-metric-card">
		<div class="cp-metric-header">
			<div class="cp-metric-icon cp-metric-icon--ram" aria-hidden="true">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
					<rect x="2" y="6" width="20" height="12" rx="2"/>
					<path d="M6 10v4M10 10v4M14 10v4M18 10v4M2 12h2M20 12h2"/>
				</svg>
			</div>
			<span class="cp-metric-label">Memory</span>
		</div>
		<div class="cp-metric-value">
			{metrics ? `${metrics.ram_used_gb.toFixed(1)} / ${metrics.ram_total_gb} GB` : '— GB'}
		</div>
		<div class="cp-progress-track" role="progressbar" aria-valuenow={ramPercent} aria-valuemin={0} aria-valuemax={100} aria-label="RAM usage {ramPercent}%">
			<div class="cp-progress-fill" style="width: {ramPercent}%"></div>
		</div>
		<div class="cp-metric-pct">{ramPercent}%</div>
	</div>

	<!-- CPU -->
	<div class="cp-metric-card">
		<div class="cp-metric-header">
			<div class="cp-metric-icon cp-metric-icon--cpu" aria-hidden="true">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
					<rect x="4" y="4" width="16" height="16" rx="2"/>
					<rect x="9" y="9" width="6" height="6"/>
					<path d="M15 2v2M9 2v2M2 9h2M2 15h2M22 9h-2M22 15h-2M15 22v-2M9 22v-2"/>
				</svg>
			</div>
			<span class="cp-metric-label">CPU</span>
		</div>
		<div class="cp-metric-value">
			{metrics ? `${metrics.cpu_percent}%` : '—%'}
		</div>
		<div class="cp-progress-track" role="progressbar" aria-valuenow={metrics?.cpu_percent ?? 0} aria-valuemin={0} aria-valuemax={100} aria-label="CPU usage {metrics?.cpu_percent ?? 0}%">
			<div class="cp-progress-fill" style="width: {metrics?.cpu_percent ?? 0}%"></div>
		</div>
		<div class="cp-metric-pct cp-metric-sub">{metrics ? `${metrics.cpu_cores} cores` : '—'}</div>
	</div>

	<!-- Storage -->
	<div class="cp-metric-card">
		<div class="cp-metric-header">
			<div class="cp-metric-icon cp-metric-icon--storage" aria-hidden="true">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
					<ellipse cx="12" cy="5" rx="9" ry="3"/>
					<path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
					<path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
				</svg>
			</div>
			<span class="cp-metric-label">Storage</span>
		</div>
		<div class="cp-metric-value">
			{metrics ? `${metrics.storage_used_gb.toFixed(1)} / ${metrics.storage_total_gb} GB` : '— GB'}
		</div>
		<div class="cp-progress-track" role="progressbar" aria-valuenow={storagePercent} aria-valuemin={0} aria-valuemax={100} aria-label="Storage usage {storagePercent}%">
			<div class="cp-progress-fill" style="width: {storagePercent}%"></div>
		</div>
		<div class="cp-metric-pct">{storagePercent}%</div>
	</div>
</section>

<!-- Section 2b: Computer Info -->
<section class="cp-info-card" aria-label="Computer information" in:fly={{ y: 12, duration: 300, delay: 75 }}>
	<div class="cp-info-header">
		<h2 class="cp-panel-title">Computer Info</h2>
	</div>
	<div class="cp-info-grid">
		<div class="cp-info-row">
			<span class="cp-info-label">Name</span>
			<span class="cp-info-value">Your Computer</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">Slug</span>
			<span class="cp-info-value cp-info-value--mono">{extractSlug(computer.domain)}</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">Template</span>
			<span class="cp-info-value">MIOSA Desktop</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">Region</span>
			<span class="cp-info-value">{computer.region}</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">Created</span>
			<span class="cp-info-value">{formatDate(computer.created_at)}</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">Uptime</span>
			<span class="cp-info-value">{calcUptime(computer.created_at)}</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">IP Address</span>
			<span class="cp-info-value">Cloud (MIOSA)</span>
		</div>
		<div class="cp-info-row">
			<span class="cp-info-label">Desktop URL</span>
			<a
				href={computer.domain.startsWith('http') ? computer.domain : `https://${computer.domain}`}
				target="_blank"
				rel="noopener noreferrer"
				class="cp-info-link"
			>
				{computer.domain}
				<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
					<polyline points="15 3 21 3 21 9"/>
					<line x1="10" y1="14" x2="21" y2="3"/>
				</svg>
			</a>
		</div>
	</div>
</section>

<!-- Section 3: Runtimes + Plan -->
<section class="cp-bottom-grid" in:fly={{ y: 12, duration: 300, delay: 100 }}>

	<!-- Runtimes column -->
	<div class="cp-panel">
		<div class="cp-panel-header">
			<h2 class="cp-panel-title">Installed Tools</h2>
			<span class="cp-panel-count">{runtimes.filter((r) => r.status !== 'stopped').length + 5} installed</span>
		</div>

		<div class="cp-runtimes-list">
			{#each [
				{ id: 'cc', name: 'Claude Code', status: 'active', label: 'AI Agent' },
				{ id: 'codex', name: 'Codex', status: 'active', label: 'AI Agent' },
				{ id: 'pg', name: 'PostgreSQL', status: 'active', label: 'Database' },
				{ id: 'redis', name: 'Redis', status: 'active', label: 'Cache' },
				{ id: 'go', name: 'Go', status: 'idle', label: 'Runtime' },
			] as tool}
				<div class="cp-runtime-row">
					<div class="cp-runtime-left">
						<div class="cp-runtime-status-dot cp-runtime-status-dot--{tool.status}" aria-hidden="true"></div>
						<div class="cp-runtime-info">
							<span class="cp-runtime-name">{tool.name}</span>
							<span class="cp-runtime-meta">
								<span class="cp-runtime-status-badge cp-runtime-status-badge--{tool.status}">{tool.status === 'active' ? 'Active' : 'Installed'}</span>
								<span class="cp-runtime-stat">{tool.label}</span>
							</span>
						</div>
					</div>
				</div>
			{/each}

			{#each runtimes as runtime (runtime.id)}
				<div class="cp-runtime-row" in:fade={{ duration: 150 }}>
					<div class="cp-runtime-left">
						<div class="cp-runtime-status-dot cp-runtime-status-dot--{runtime.status}" aria-hidden="true"></div>
						<div class="cp-runtime-info">
							<span class="cp-runtime-name">{runtime.name}</span>
							<span class="cp-runtime-meta">
								<span class="cp-runtime-status-badge cp-runtime-status-badge--{runtime.status}">{runtimeStatusLabel(runtime.status)}</span>
								<span class="cp-runtime-stat">{runtime.memory_gb.toFixed(1)} GB</span>
								<span class="cp-runtime-sep" aria-hidden="true"></span>
								<span class="cp-runtime-stat">{formatUptime(runtime.uptime_seconds)}</span>
							</span>
						</div>
					</div>
					<button
						class="cp-runtime-stop-btn"
						onclick={() => onStopRuntime(runtime.id)}
						disabled={stoppingRuntime === runtime.id || runtime.status === 'stopped'}
						aria-label="Stop {runtime.name}"
					>
						{stoppingRuntime === runtime.id ? 'Stopping…' : 'Stop'}
					</button>
				</div>
			{/each}

			{#if runtimes.length === 0}
				<p class="cp-empty-list-text">No runtimes running.</p>
			{/if}
		</div>

		<div class="cp-panel-footer">
			<button class="cp-btn cp-btn--outline cp-btn--full">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<circle cx="12" cy="12" r="10"/>
					<line x1="12" y1="8" x2="12" y2="16"/>
					<line x1="8" y1="12" x2="16" y2="12"/>
				</svg>
				Start Runtime
			</button>
		</div>
	</div>

	<!-- Plan & Credits column -->
	<div class="cp-panel">
		{#if subscription}
			<div class="cp-panel-header">
				<h2 class="cp-panel-title">Plan & Credits</h2>
			</div>

			<div class="cp-plan-summary">
				<div class="cp-plan-summary-row">
					<span class="cp-plan-summary-name">{subscription.plan_name}</span>
					<span class="cp-plan-summary-price">
						{formatPrice(subscription.price_cents)}<span class="cp-plan-summary-interval">/{subscription.interval}</span>
					</span>
				</div>

				<div class="cp-credits-section">
					<div class="cp-credits-header">
						<span class="cp-credits-label">Credits</span>
						<span class="cp-credits-value">
							{(subscription.credits_total - subscription.credits_used).toLocaleString()} / {subscription.credits_total.toLocaleString()} remaining
						</span>
					</div>
					<div
						class="cp-progress-track"
						role="progressbar"
						aria-valuenow={creditsPercent}
						aria-valuemin={0}
						aria-valuemax={100}
						aria-label="Credits: {creditsPercent}% remaining"
					>
						<div
							class="cp-progress-fill cp-progress-fill--solid"
							style="width: {creditsPercent}%; background: {creditsColor(creditsPercent)}"
						></div>
					</div>
				</div>

				<div class="cp-plan-details">
					<div class="cp-plan-detail">
						<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
							<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
							<circle cx="9" cy="7" r="4"/>
							<path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>
						</svg>
						{subscription.seats} {subscription.seats === 1 ? 'member' : 'members'}
					</div>
					<div class="cp-plan-detail">
						<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
							<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
							<line x1="16" y1="2" x2="16" y2="6"/>
							<line x1="8" y1="2" x2="8" y2="6"/>
							<line x1="3" y1="10" x2="21" y2="10"/>
						</svg>
						Next billing {formatBillingDate(subscription.next_billing_date)}
					</div>
				</div>
			</div>

			<div class="cp-plan-actions">
				<button class="cp-btn cp-btn--primary" onclick={onOpenPricing}>
					Upgrade Plan
				</button>
				<button class="cp-btn cp-btn--outline">
					Buy Credits
				</button>
			</div>

		{:else}
			<div class="cp-no-sub-cta">
				<div class="cp-no-sub-icon" aria-hidden="true">
					<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
						<rect x="2" y="3" width="20" height="14" rx="2"/>
						<path d="M8 21h8M12 17v4"/>
					</svg>
				</div>
				<h3 class="cp-no-sub-title">Get a Cloud Computer</h3>
				<p class="cp-no-sub-body">Access BusinessOS from anywhere. Invite your team. Run AI agents.</p>
				<button class="cp-btn cp-btn--primary" onclick={onOpenPricing}>
					Create Computer
				</button>
			</div>
		{/if}
	</div>

</section>
{/if}

<style>
	/* ── View Tabs ─────────────────────────────────────────────────────────── */
	.cp-view-tabs {
		display: flex;
		gap: 2px;
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e5e7eb);
		border-radius: 8px;
		padding: 3px;
		width: fit-content;
	}

	.cp-view-tab {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 16px;
		border-radius: 8px;
		border: none;
		background: transparent;
		color: var(--dt3, #6b7280);
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.cp-view-tab:hover { background: var(--dbg2, #f3f4f6); color: var(--dt, #111); }
	.cp-view-tab.active { background: var(--dt, #111); color: #fff; }
	:global(.dark) .cp-view-tab.active { background: #3b82f6; }

	/* ── Desktop Stream ───────────────────────────────────────────────────── */
	.cp-desktop-container {
		border-radius: 8px;
		overflow: hidden;
		border: 1px solid var(--dbd, #e5e7eb);
		background: #000;
		position: relative;
		flex: 1;
		min-height: 0;
	}

	.cp-desktop-iframe {
		width: 100%;
		height: 100%;
		border: none;
		display: block;
	}

	.cp-desktop-loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 500px;
		color: #888;
		gap: 12px;
	}

	.cp-desktop-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid rgba(255,255,255,0.1);
		border-top-color: #7c3aed;
		border-radius: 50%;
		animation: cp-dspin 0.8s linear infinite;
	}

	@keyframes cp-dspin {
		to { transform: rotate(360deg); }
	}

	/* ── Status Hero ───────────────────────────────────────────────────────── */
	.cp-hero {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		padding: 10px 16px;
		background: var(--dbg, #ffffff);
		border: 1px solid var(--dbd, rgba(0,0,0,0.07));
		border-radius: 8px;
		flex-wrap: wrap;
	}

	:global(.dark) .cp-hero {
		background: var(--dbg, rgba(255,255,255,0.03));
		border-color: var(--dbd, rgba(255,255,255,0.07));
	}

	.cp-hero-left {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.cp-status-indicator {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.cp-status-indicator--running {
		background: #22c55e;
		box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.2);
	}

	.cp-status-indicator--hibernating {
		background: #f59e0b;
		box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.2);
	}

	.cp-status-indicator--stopped {
		background: #9ca3af;
	}

	.cp-hero-info {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.cp-hero-title-row {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.cp-hero-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--dt, #111827);
		margin: 0;
		letter-spacing: -0.02em;
	}

	:global(.dark) .cp-hero-title { color: var(--dt, #f3f4f6); }

	.cp-hero-domain {
		font-size: 13px;
		color: var(--dt3, #9ca3af);
		margin: 0;
		font-family: ui-monospace, 'SF Mono', 'Fira Code', monospace;
	}

	.cp-status-badge {
		font-size: 11px;
		font-weight: 600;
		padding: 3px 8px;
		border-radius: 20px;
		letter-spacing: 0.02em;
	}

	.cp-status-badge--running {
		background: rgba(34, 197, 94, 0.12);
		color: #16a34a;
	}
	:global(.dark) .cp-status-badge--running {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}

	.cp-status-badge--hibernating {
		background: rgba(245, 158, 11, 0.12);
		color: #d97706;
	}
	:global(.dark) .cp-status-badge--hibernating {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}

	.cp-status-badge--stopped {
		background: rgba(156, 163, 175, 0.15);
		color: #6b7280;
	}
	:global(.dark) .cp-status-badge--stopped {
		background: rgba(156, 163, 175, 0.1);
		color: #9ca3af;
	}

	.cp-hero-actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	/* ── Buttons ───────────────────────────────────────────────────────────── */
	.cp-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		font-weight: 500;
		padding: 8px 14px;
		border-radius: 8px;
		cursor: pointer;
		border: none;
		transition: background 140ms ease-out, color 140ms ease-out, opacity 140ms ease-out;
		text-decoration: none;
		line-height: 1;
		white-space: nowrap;
	}

	.cp-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.cp-btn--primary {
		background: #3b82f6;
		color: #ffffff;
	}
	.cp-btn--primary:hover:not(:disabled) { background: #2563eb; }

	.cp-btn--outline {
		background: transparent;
		color: var(--dt2, #374151);
		border: 1px solid var(--dbd, rgba(0,0,0,0.1));
	}
	.cp-btn--outline:hover:not(:disabled) {
		background: var(--dbg3, rgba(0,0,0,0.04));
	}
	:global(.dark) .cp-btn--outline {
		color: var(--dt2, rgba(255,255,255,0.7));
		border-color: var(--dbd, rgba(255,255,255,0.1));
	}
	:global(.dark) .cp-btn--outline:hover:not(:disabled) {
		background: rgba(255,255,255,0.06);
	}

	.cp-btn--full { width: 100%; justify-content: center; }

	/* ── Metrics grid ──────────────────────────────────────────────────────── */
	.cp-metrics-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 12px;
	}

	@media (max-width: 900px) {
		.cp-metrics-grid { grid-template-columns: 1fr; }
	}

	.cp-metric-card {
		background: var(--dbg, #ffffff);
		border: 1px solid var(--dbd, rgba(0,0,0,0.07));
		border-radius: 12px;
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		transition: border-color 140ms ease-out;
	}
	.cp-metric-card:hover { border-color: rgba(59, 130, 246, 0.2); }

	:global(.dark) .cp-metric-card {
		background: var(--dbg, rgba(255,255,255,0.03));
		border-color: var(--dbd, rgba(255,255,255,0.07));
	}
	:global(.dark) .cp-metric-card:hover {
		border-color: rgba(59, 130, 246, 0.25);
	}

	.cp-metric-header {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.cp-metric-icon {
		width: 28px;
		height: 28px;
		border-radius: 7px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.cp-metric-icon--ram { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
	.cp-metric-icon--cpu { background: rgba(139, 92, 246, 0.1); color: #8b5cf6; }
	.cp-metric-icon--storage { background: rgba(16, 185, 129, 0.1); color: #10b981; }

	.cp-metric-label {
		font-size: 11px;
		font-weight: 600;
		color: var(--dt3, #9ca3af);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.cp-metric-value {
		font-size: 22px;
		font-weight: 600;
		color: var(--dt, #111827);
		letter-spacing: -0.02em;
		line-height: 1;
	}
	:global(.dark) .cp-metric-value { color: var(--dt, #f3f4f6); }

	.cp-metric-pct {
		font-size: 12px;
		color: var(--dt3, #9ca3af);
		font-weight: 500;
	}

	.cp-metric-sub {
		font-size: 12px;
		color: var(--dt3, #9ca3af);
	}

	/* ── Progress bars ─────────────────────────────────────────────────────── */
	.cp-progress-track {
		height: 8px;
		background: var(--dbg2, rgba(0,0,0,0.06));
		border-radius: 100px;
		overflow: hidden;
	}

	:global(.dark) .cp-progress-track {
		background: rgba(255,255,255,0.08);
	}

	.cp-progress-fill {
		height: 100%;
		background: linear-gradient(90deg, #3b82f6, #8b5cf6);
		border-radius: 100px;
		transition: width 600ms cubic-bezier(0.34, 1.56, 0.64, 1);
	}

	.cp-progress-fill--solid {
		background: none;
		transition: width 600ms cubic-bezier(0.34, 1.56, 0.64, 1);
	}

	/* ── Bottom two-column grid ────────────────────────────────────────────── */
	.cp-bottom-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
		align-items: start;
	}

	@media (max-width: 900px) {
		.cp-bottom-grid { grid-template-columns: 1fr; }
	}

	/* ── Panel shared ──────────────────────────────────────────────────────── */
	.cp-panel {
		background: var(--dbg, #ffffff);
		border: 1px solid var(--dbd, rgba(0,0,0,0.07));
		border-radius: 12px;
		overflow: hidden;
	}

	:global(.dark) .cp-panel {
		background: var(--dbg, rgba(255,255,255,0.03));
		border-color: var(--dbd, rgba(255,255,255,0.07));
	}

	.cp-panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px 0;
	}

	.cp-panel-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt2, #374151);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin: 0;
	}
	:global(.dark) .cp-panel-title { color: var(--dt2, rgba(255,255,255,0.5)); }

	.cp-panel-count {
		font-size: 11px;
		color: var(--dt3, #9ca3af);
		font-weight: 500;
	}

	.cp-panel-footer {
		padding: 12px 20px 20px;
		border-top: 1px solid var(--dbd, rgba(0,0,0,0.06));
		margin-top: 4px;
	}
	:global(.dark) .cp-panel-footer {
		border-color: rgba(255,255,255,0.06);
	}

	/* ── Runtimes list ─────────────────────────────────────────────────────── */
	.cp-runtimes-list {
		padding: 12px 20px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cp-runtime-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 12px;
		border-radius: 8px;
		transition: background 120ms ease-out;
	}
	.cp-runtime-row:hover { background: var(--dbg2, rgba(0,0,0,0.03)); }
	:global(.dark) .cp-runtime-row:hover { background: rgba(255,255,255,0.04); }

	.cp-runtime-left {
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
	}

	.cp-runtime-status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.cp-runtime-status-dot--active { background: #22c55e; }
	.cp-runtime-status-dot--idle { background: #f59e0b; }
	.cp-runtime-status-dot--stopped { background: #9ca3af; }

	.cp-runtime-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.cp-runtime-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt, #111827);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	:global(.dark) .cp-runtime-name { color: var(--dt, #f3f4f6); }

	.cp-runtime-meta {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.cp-runtime-status-badge {
		font-size: 10px;
		font-weight: 600;
		padding: 1px 6px;
		border-radius: 20px;
	}
	.cp-runtime-status-badge--active {
		background: rgba(34, 197, 94, 0.12);
		color: #16a34a;
	}
	:global(.dark) .cp-runtime-status-badge--active {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}
	.cp-runtime-status-badge--idle {
		background: rgba(245, 158, 11, 0.12);
		color: #d97706;
	}
	:global(.dark) .cp-runtime-status-badge--idle {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}
	.cp-runtime-status-badge--stopped {
		background: rgba(156, 163, 175, 0.12);
		color: #6b7280;
	}

	.cp-runtime-stat {
		font-size: 11px;
		color: var(--dt3, #9ca3af);
	}

	.cp-runtime-sep {
		display: inline-block;
		width: 1px;
		height: 10px;
		background: var(--dbd, rgba(0,0,0,0.1));
	}
	:global(.dark) .cp-runtime-sep {
		background: rgba(255,255,255,0.1);
	}

	.cp-runtime-stop-btn {
		font-size: 11px;
		font-weight: 500;
		color: var(--dt3, #9ca3af);
		background: transparent;
		border: 1px solid var(--dbd, rgba(0,0,0,0.1));
		border-radius: 6px;
		padding: 4px 10px;
		cursor: pointer;
		transition: color 120ms, border-color 120ms, background 120ms;
		white-space: nowrap;
		flex-shrink: 0;
	}
	.cp-runtime-stop-btn:hover:not(:disabled) {
		color: #ef4444;
		border-color: rgba(239, 68, 68, 0.3);
		background: rgba(239, 68, 68, 0.05);
	}
	.cp-runtime-stop-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}
	:global(.dark) .cp-runtime-stop-btn {
		color: var(--dt3, rgba(255,255,255,0.35));
		border-color: rgba(255,255,255,0.08);
	}
	:global(.dark) .cp-runtime-stop-btn:hover:not(:disabled) {
		color: #f87171;
		border-color: rgba(248, 113, 113, 0.3);
		background: rgba(248, 113, 113, 0.06);
	}

	.cp-empty-list-text {
		font-size: 13px;
		color: var(--dt3, #9ca3af);
		text-align: center;
		padding: 24px 0;
		margin: 0;
	}

	/* ── Plan summary ──────────────────────────────────────────────────────── */
	.cp-plan-summary {
		padding: 16px 20px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.cp-plan-summary-row {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 8px;
	}

	.cp-plan-summary-name {
		font-size: 18px;
		font-weight: 700;
		color: var(--dt, #111827);
		letter-spacing: -0.02em;
	}
	:global(.dark) .cp-plan-summary-name { color: var(--dt, #f3f4f6); }

	.cp-plan-summary-price {
		font-size: 20px;
		font-weight: 700;
		color: var(--dt, #111827);
		letter-spacing: -0.02em;
	}
	:global(.dark) .cp-plan-summary-price { color: var(--dt, #f3f4f6); }

	.cp-plan-summary-interval {
		font-size: 13px;
		font-weight: 400;
		color: var(--dt3, #9ca3af);
	}

	.cp-credits-section {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.cp-credits-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.cp-credits-label {
		font-size: 11px;
		font-weight: 600;
		color: var(--dt3, #9ca3af);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.cp-credits-value {
		font-size: 12px;
		color: var(--dt2, #6b7280);
		font-weight: 500;
	}
	:global(.dark) .cp-credits-value { color: var(--dt2, rgba(255,255,255,0.5)); }

	.cp-plan-details {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.cp-plan-detail {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 12px;
		color: var(--dt3, #6b7280);
	}
	:global(.dark) .cp-plan-detail { color: var(--dt3, rgba(255,255,255,0.4)); }

	.cp-plan-actions {
		padding: 12px 20px 20px;
		border-top: 1px solid var(--dbd, rgba(0,0,0,0.06));
		display: flex;
		gap: 8px;
	}
	:global(.dark) .cp-plan-actions { border-color: rgba(255,255,255,0.06); }

	/* ── No subscription CTA ───────────────────────────────────────────────── */
	.cp-no-sub-cta {
		padding: 32px 24px;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 12px;
	}

	.cp-no-sub-icon {
		width: 56px;
		height: 56px;
		border-radius: 14px;
		background: rgba(59, 130, 246, 0.1);
		color: #3b82f6;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.cp-no-sub-title {
		font-size: 16px;
		font-weight: 600;
		color: var(--dt, #111827);
		margin: 0;
	}
	:global(.dark) .cp-no-sub-title { color: var(--dt, #f3f4f6); }

	.cp-no-sub-body {
		font-size: 13px;
		color: var(--dt3, #6b7280);
		margin: 0;
		max-width: 260px;
		line-height: 1.5;
	}
	:global(.dark) .cp-no-sub-body { color: var(--dt3, rgba(255,255,255,0.4)); }

	/* ── Computer Info card ────────────────────────────────────────────────── */
	.cp-info-card {
		background: var(--dbg, #ffffff);
		border: 1px solid var(--dbd, rgba(0,0,0,0.07));
		border-radius: 12px;
		overflow: hidden;
	}

	:global(.dark) .cp-info-card {
		background: var(--dbg, rgba(255,255,255,0.03));
		border-color: var(--dbd, rgba(255,255,255,0.07));
	}

	.cp-info-header {
		padding: 16px 20px 0;
	}

	.cp-info-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0;
		padding: 12px 20px 16px;
	}

	@media (max-width: 1100px) {
		.cp-info-grid { grid-template-columns: repeat(2, 1fr); }
	}

	@media (max-width: 600px) {
		.cp-info-grid { grid-template-columns: 1fr; }
	}

	.cp-info-row {
		display: flex;
		flex-direction: column;
		gap: 3px;
		padding: 10px 12px;
	}

	.cp-info-label {
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.07em;
		color: var(--dt3, #9ca3af);
	}

	.cp-info-value {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt, #111827);
	}
	:global(.dark) .cp-info-value { color: var(--dt, #f3f4f6); }

	.cp-info-value--mono {
		font-family: ui-monospace, 'SF Mono', 'Fira Code', monospace;
		font-size: 12px;
	}

	.cp-info-link {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		font-weight: 500;
		color: #3b82f6;
		text-decoration: none;
		font-family: ui-monospace, 'SF Mono', 'Fira Code', monospace;
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.cp-info-link:hover { text-decoration: underline; }
	:global(.dark) .cp-info-link { color: #60a5fa; }
</style>
