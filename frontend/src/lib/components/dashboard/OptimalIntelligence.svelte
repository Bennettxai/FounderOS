<script lang="ts">
	import { SurfacingFeed, ProfileSummary } from '@miosa/optimal-engine';
	import { getEngine } from '$lib/optimal-engine/context';
	import type { EngineContext } from '@miosa/optimal-engine';

	let engine = $state<EngineContext | null>(null);
	let initError = $state<string | null>(null);

	$effect(() => {
		try {
			engine = getEngine();
		} catch (err) {
			initError = err instanceof Error ? err.message : 'Engine unavailable';
		}
	});
</script>

{#if initError}
	<div class="oi-fallback" role="status">
		<span class="oi-fallback-label">Intelligence feed unavailable</span>
		<span class="oi-fallback-reason">{initError}</span>
	</div>
{:else if engine}
	<section class="oi-root" aria-label="Optimal Intelligence">
		<div class="oi-section-header">
			<span class="oi-section-label">Intelligence</span>
		</div>

		<div class="oi-panels">
			<!-- Profile overview -->
			<div class="oi-panel oi-panel--profile">
				<div class="oi-panel-title">Workspace Profile</div>
				<ProfileSummary {engine} />
			</div>

			<!-- Surfacing feed -->
			<div class="oi-panel oi-panel--feed">
				<div class="oi-panel-title">Proactive Signals</div>
				<SurfacingFeed {engine} maxEvents={20} />
			</div>
		</div>
	</section>
{:else}
	<div class="oi-fallback" role="status">
		<span class="oi-fallback-label">Connecting to intelligence engine...</span>
	</div>
{/if}

<style>
	/* ── Root ──────────────────────────────────────────────────────────────── */
	.oi-root {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		padding: 1.25rem 1.5rem;
		background: #ffffff;
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 0.875rem;
	}

	:global(.dark) .oi-root {
		background: rgba(255, 255, 255, 0.03);
		border-color: rgba(255, 255, 255, 0.06);
	}

	/* ── Header ────────────────────────────────────────────────────────────── */
	.oi-section-header {
		margin-bottom: 0.25rem;
	}

	.oi-section-label {
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: #9ca3af;
	}

	:global(.dark) .oi-section-label {
		color: rgba(255, 255, 255, 0.3);
	}

	/* ── Two-column layout ─────────────────────────────────────────────────── */
	.oi-panels {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.25rem;
		align-items: start;
	}

	@media (max-width: 768px) {
		.oi-panels {
			grid-template-columns: 1fr;
		}
	}

	/* ── Panel ─────────────────────────────────────────────────────────────── */
	.oi-panel {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 1rem 1.125rem;
		background: rgba(0, 0, 0, 0.02);
		border: 1px solid rgba(0, 0, 0, 0.05);
		border-radius: 0.75rem;
		min-height: 120px;
	}

	:global(.dark) .oi-panel {
		background: rgba(255, 255, 255, 0.025);
		border-color: rgba(255, 255, 255, 0.05);
	}

	.oi-panel-title {
		font-size: 0.68rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.07em;
		color: #9ca3af;
	}

	:global(.dark) .oi-panel-title {
		color: rgba(255, 255, 255, 0.3);
	}

	/* ── Fallback ──────────────────────────────────────────────────────────── */
	.oi-fallback {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		padding: 1rem 1.25rem;
		background: #ffffff;
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 0.875rem;
	}

	:global(.dark) .oi-fallback {
		background: rgba(255, 255, 255, 0.03);
		border-color: rgba(255, 255, 255, 0.06);
	}

	.oi-fallback-label {
		font-size: 0.8rem;
		color: #9ca3af;
	}

	:global(.dark) .oi-fallback-label {
		color: rgba(255, 255, 255, 0.35);
	}

	.oi-fallback-reason {
		font-size: 0.72rem;
		color: #d1d5db;
		font-family: ui-monospace, monospace;
	}

	:global(.dark) .oi-fallback-reason {
		color: rgba(255, 255, 255, 0.2);
	}
</style>
