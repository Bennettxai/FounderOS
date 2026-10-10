<script lang="ts">
	import { getApiBaseUrl, getCSRFToken } from '$lib/api/base';
	import SkeletonLoader from '$lib/components/ui/SkeletonLoader.svelte';

	// ─── Types ────────────────────────────────────────────────────────────────

	interface OptimalNode {
		slug: string;
		name: string;
		type: string;
		signal_count: number;
		has_signals: boolean;
	}

	interface Props {
		selectedSources: string[];
		onchange: (sources: string[]) => void;
		disabled?: boolean;
	}

	// ─── State ────────────────────────────────────────────────────────────────

	let { selectedSources, onchange, disabled = false }: Props = $props();

	let nodes: OptimalNode[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);

	// ─── Node type → visual config ────────────────────────────────────────────

	const TYPE_CONFIG: Record<string, { label: string; color: string; border: string }> = {
		person: {
			label: 'Person',
			color: 'color-mix(in srgb, #6366f1 12%, transparent)',
			border: '#6366f1'
		},
		entity: {
			label: 'Entity',
			color: 'color-mix(in srgb, #0ea5e9 12%, transparent)',
			border: '#0ea5e9'
		},
		'operation:program': {
			label: 'Program',
			color: 'color-mix(in srgb, #10b981 12%, transparent)',
			border: '#10b981'
		},
		'operation:research': {
			label: 'Research',
			color: 'color-mix(in srgb, #f59e0b 12%, transparent)',
			border: '#f59e0b'
		},
		'operation:network': {
			label: 'Network',
			color: 'color-mix(in srgb, #ec4899 12%, transparent)',
			border: '#ec4899'
		},
		'unit:community': {
			label: 'Community',
			color: 'color-mix(in srgb, #8b5cf6 12%, transparent)',
			border: '#8b5cf6'
		},
		'domain:media': {
			label: 'Media',
			color: 'color-mix(in srgb, #f97316 12%, transparent)',
			border: '#f97316'
		},
		inbox: {
			label: 'Inbox',
			color: 'color-mix(in srgb, #64748b 12%, transparent)',
			border: '#64748b'
		},
		registry: {
			label: 'Registry',
			color: 'color-mix(in srgb, #14b8a6 12%, transparent)',
			border: '#14b8a6'
		},
		'cross-cutting': {
			label: 'Cross-cutting',
			color: 'color-mix(in srgb, #ef4444 12%, transparent)',
			border: '#ef4444'
		}
	};

	const FALLBACK_CONFIG = {
		label: 'Node',
		color: 'color-mix(in srgb, #64748b 12%, transparent)',
		border: '#64748b'
	};

	function typeConfig(type: string) {
		return TYPE_CONFIG[type] ?? FALLBACK_CONFIG;
	}

	// ─── Derived ──────────────────────────────────────────────────────────────

	let allSelected = $derived(
		nodes.length > 0 && nodes.every((n) => selectedSources.includes(n.slug))
	);

	let noneSelected = $derived(selectedSources.length === 0);

	// ─── Fetch ────────────────────────────────────────────────────────────────

	async function fetchNodes() {
		loading = true;
		error = null;
		try {
			const res = await fetch(`${getApiBaseUrl()}/optimal/nodes`, {
				method: 'GET',
				headers: { 'X-CSRF-Token': getCSRFToken() || '' },
				credentials: 'include',
				signal: AbortSignal.timeout(8000)
			});

			if (!res.ok) {
				throw new Error(`Server returned ${res.status}`);
			}

			const data = await res.json();
			nodes = Array.isArray(data.nodes) ? data.nodes : [];
		} catch (err) {
			if (err instanceof Error && err.name === 'TimeoutError') {
				error = 'Request timed out. Check that the backend is running.';
			} else {
				error = 'Could not load OptimalOS nodes.';
			}
		} finally {
			loading = false;
		}
	}

	// ─── Selection handlers ───────────────────────────────────────────────────

	function toggle(slug: string) {
		if (disabled) return;
		const current = new Set(selectedSources);
		if (current.has(slug)) {
			current.delete(slug);
		} else {
			current.add(slug);
		}
		onchange([...current]);
	}

	function selectAll() {
		if (disabled) return;
		onchange(nodes.map((n) => n.slug));
	}

	function clearAll() {
		if (disabled) return;
		onchange([]);
	}

	// ─── Mount ────────────────────────────────────────────────────────────────

	$effect(() => {
		fetchNodes();
	});
</script>

<div class="ods-root" class:ods-root--disabled={disabled}>
	<!-- Header bar -->
	<div class="ods-header">
		<div class="ods-header__left">
			<span class="ods-header__title">OptimalOS Nodes</span>
			{#if !loading && !error && nodes.length > 0}
				<span class="ods-header__count">
					{selectedSources.length} / {nodes.length} selected
				</span>
			{/if}
		</div>
		{#if !loading && !error && nodes.length > 0}
			<div class="ods-header__actions">
				<button
					class="ods-btn-text"
					onclick={selectAll}
					disabled={disabled || allSelected}
					aria-label="Select all nodes"
				>
					Select all
				</button>
				<span class="ods-btn-divider" aria-hidden="true">/</span>
				<button
					class="ods-btn-text"
					onclick={clearAll}
					disabled={disabled || noneSelected}
					aria-label="Clear all selections"
				>
					Clear all
				</button>
			</div>
		{/if}
	</div>

	<!-- Body -->
	<div class="ods-body">
		{#if loading}
			<!-- Skeleton state -->
			<div class="ods-grid" aria-busy="true" aria-label="Loading nodes">
				{#each Array.from({ length: 6 }, (_, i) => i) as i (i)}
					<div class="ods-skeleton-card">
						<div class="ods-skeleton-card__header">
							<SkeletonLoader variant="button" />
						</div>
						<SkeletonLoader variant="text" count={2} />
					</div>
				{/each}
			</div>
		{:else if error}
			<!-- Error state -->
			<div class="ods-empty" role="alert">
				<svg
					class="ods-empty__icon"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					aria-hidden="true"
				>
					<circle cx="12" cy="12" r="9" />
					<line x1="12" y1="8" x2="12" y2="12" />
					<line x1="12" y1="16" x2="12.01" y2="16" />
				</svg>
				<p class="ods-empty__message">{error}</p>
				<button class="ods-btn-retry" onclick={fetchNodes}>Retry</button>
			</div>
		{:else if nodes.length === 0}
			<!-- Empty state -->
			<div class="ods-empty">
				<svg
					class="ods-empty__icon"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					aria-hidden="true"
				>
					<path d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7" />
					<path d="M4 13h16v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5z" />
				</svg>
				<p class="ods-empty__message">No OptimalOS nodes found.</p>
			</div>
		{:else}
			<!-- Node grid -->
			<div class="ods-grid" role="group" aria-label="OptimalOS node selection">
				{#each nodes as node (node.slug)}
					{@const cfg = typeConfig(node.type)}
					{@const isSelected = selectedSources.includes(node.slug)}
					{@const checkboxId = `ods-node-${node.slug}`}
					<label
						for={checkboxId}
						class="ods-card"
						class:ods-card--selected={isSelected}
						class:ods-card--disabled={disabled}
						style="--node-border: {cfg.border}; --node-bg: {cfg.color};"
					>
						<input
							id={checkboxId}
							type="checkbox"
							class="ods-card__checkbox"
							checked={isSelected}
							{disabled}
							onchange={() => toggle(node.slug)}
							aria-label="Select {node.name} node"
						/>
						<div class="ods-card__body">
							<div class="ods-card__top">
								<span class="ods-card__name">{node.name}</span>
								<span
									class="ods-card__badge"
									style="background: {cfg.color}; color: {cfg.border};"
								>
									{cfg.label}
								</span>
							</div>
							<div class="ods-card__meta">
								<span class="ods-card__slug">{node.slug}</span>
								<span class="ods-card__signals">
									{node.signal_count}
									{node.signal_count === 1 ? 'signal' : 'signals'}
								</span>
							</div>
						</div>
					</label>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	/* ═══════════════════════════════════════════════════════════════════════ */
	/*  OPTIMAL DATA SOURCE SELECTOR (ods-)                                   */
	/* ═══════════════════════════════════════════════════════════════════════ */

	.ods-root {
		display: flex;
		flex-direction: column;
		gap: 12px;
		width: 100%;
	}

	.ods-root--disabled {
		opacity: 0.6;
		pointer-events: none;
	}

	/* ── Header ──────────────────────────────────────────────────────────── */

	.ods-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		flex-wrap: wrap;
	}

	.ods-header__left {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.ods-header__title {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
	}

	.ods-header__count {
		font-size: 12px;
		color: var(--dt3);
		background: var(--dbg2);
		padding: 2px 8px;
		border-radius: 999px;
		border: 1px solid var(--dbd2);
	}

	.ods-header__actions {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.ods-btn-text {
		font-size: 12px;
		font-weight: 500;
		color: var(--dt3);
		background: none;
		border: none;
		cursor: pointer;
		padding: 2px 4px;
		border-radius: 4px;
		transition: color 0.12s;
		line-height: 1;
	}

	.ods-btn-text:hover:not(:disabled) {
		color: var(--dt);
	}

	.ods-btn-text:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.ods-btn-divider {
		font-size: 12px;
		color: var(--dbd);
		user-select: none;
	}

	/* ── Body / Grid ─────────────────────────────────────────────────────── */

	.ods-body {
		width: 100%;
	}

	.ods-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 10px;
	}

	@media (max-width: 480px) {
		.ods-grid {
			grid-template-columns: 1fr;
		}
	}

	/* ── Skeleton card ───────────────────────────────────────────────────── */

	.ods-skeleton-card {
		border: 1px solid var(--dbd2);
		border-radius: 10px;
		padding: 14px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		background: var(--dbg);
	}

	.ods-skeleton-card__header {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	/* ── Node card ───────────────────────────────────────────────────────── */

	.ods-card {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 12px 14px;
		border-radius: 10px;
		border: 1px solid var(--dbd2);
		border-left: 3px solid var(--node-border, var(--dbd));
		background: var(--dbg);
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
		user-select: none;
	}

	.ods-card:hover:not(.ods-card--disabled) {
		border-color: var(--dbd);
		border-left-color: var(--node-border, var(--dbd));
		background: var(--dbg2);
	}

	.ods-card--selected {
		border-color: color-mix(in srgb, var(--node-border, #6366f1) 40%, var(--dbd2));
		border-left-color: var(--node-border, #6366f1);
		background: var(--node-bg, var(--dbg2));
		box-shadow: 0 0 0 1px color-mix(in srgb, var(--node-border, #6366f1) 20%, transparent);
	}

	.ods-card--disabled {
		cursor: not-allowed;
	}

	.ods-card__checkbox {
		flex-shrink: 0;
		width: 15px;
		height: 15px;
		margin-top: 2px;
		accent-color: var(--node-border, #6366f1);
		cursor: pointer;
	}

	.ods-card--disabled .ods-card__checkbox {
		cursor: not-allowed;
	}

	.ods-card__body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 5px;
	}

	.ods-card__top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 6px;
		flex-wrap: wrap;
	}

	.ods-card__name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.ods-card__badge {
		display: inline-flex;
		align-items: center;
		padding: 1px 7px;
		border-radius: 999px;
		font-size: 10px;
		font-weight: 600;
		white-space: nowrap;
		flex-shrink: 0;
	}

	.ods-card__meta {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.ods-card__slug {
		font-size: 11px;
		color: var(--dt4, var(--dt3));
		font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.ods-card__signals {
		font-size: 11px;
		color: var(--dt3);
		white-space: nowrap;
		flex-shrink: 0;
	}

	/* ── Empty / Error states ────────────────────────────────────────────── */

	.ods-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 32px 16px;
		border: 1px dashed var(--dbd2);
		border-radius: 10px;
		text-align: center;
	}

	.ods-empty__icon {
		width: 32px;
		height: 32px;
		color: var(--dt4, var(--dt3));
	}

	.ods-empty__message {
		font-size: 13px;
		color: var(--dt3);
		margin: 0;
	}

	.ods-btn-retry {
		font-size: 12px;
		font-weight: 500;
		color: var(--dt);
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 6px;
		padding: 6px 14px;
		cursor: pointer;
		transition: background 0.12s, border-color 0.12s;
	}

	.ods-btn-retry:hover {
		background: var(--dbg3, var(--dbg2));
		border-color: var(--dbd2, var(--dbd));
	}

	/* ── Dark mode ───────────────────────────────────────────────────────── */

	:global(.dark) .ods-card--selected {
		box-shadow: 0 0 0 1px color-mix(in srgb, var(--node-border, #6366f1) 30%, transparent);
	}
</style>
