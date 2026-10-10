<script lang="ts">
	import { fly } from 'svelte/transition';

	interface Props {
		onAction?: (action: string) => void;
	}

	let { onAction }: Props = $props();

	const actions = [
		{
			id: 'new-chat',
			label: 'New Chat',
			shortcut: '⌘N'
		},
		{
			id: 'new-project',
			label: 'New Project',
			shortcut: '⌘P'
		},
		{
			id: 'new-task',
			label: 'Add Task',
			shortcut: '⌘T'
		},
		{
			id: 'daily-log',
			label: 'Daily Log',
			shortcut: '⌘L'
		}
	];

	let hoveredId = $state<string | null>(null);
</script>

{#snippet chatIcon()}
	<svg class="dw-action-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
			d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
	</svg>
{/snippet}

{#snippet projectIcon()}
	<svg class="dw-action-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
			d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
	</svg>
{/snippet}

{#snippet taskIcon()}
	<svg class="dw-action-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
			d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
	</svg>
{/snippet}

{#snippet logIcon()}
	<svg class="dw-action-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
			d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
	</svg>
{/snippet}

<div class="dw-actions-card">
	<!-- Header -->
	<div class="dw-actions-header">
		<div class="dw-actions-icon-wrap" aria-hidden="true">
			<svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
					d="M13 10V3L4 14h7v7l9-11h-7z" />
			</svg>
		</div>
		<h2 class="dw-actions-title">Quick Actions</h2>
	</div>

	<!-- Actions grid -->
	<div class="dw-actions-grid">
		{#each actions as action, index (action.id)}
			<button
				onclick={() => onAction?.(action.id)}
				onmouseenter={() => (hoveredId = action.id)}
				onmouseleave={() => (hoveredId = null)}
				class="dw-action-card {hoveredId === action.id ? 'dw-action-card--hovered' : ''}"
				aria-label="{action.label} ({action.shortcut})"
				in:fly={{ y: 10, duration: 300, delay: index * 60 }}
			>
				<span class="dw-action-icon-wrap">
					{#if action.id === 'new-chat'}
						{@render chatIcon()}
					{:else if action.id === 'new-project'}
						{@render projectIcon()}
					{:else if action.id === 'new-task'}
						{@render taskIcon()}
					{:else if action.id === 'daily-log'}
						{@render logIcon()}
					{/if}
				</span>
				<span class="dw-action-label">{action.label}</span>
				<span class="dw-action-shortcut">{action.shortcut}</span>
			</button>
		{/each}
	</div>
</div>

<style>
	/* ── Quick Actions Widget (dw-action-*) — Foundation Tokens ── */

	.dw-actions-card {
		background: var(--dbg);
		border-radius: var(--radius-md);
		border: 1px solid var(--dbd);
		padding: var(--space-5);
		box-shadow: var(--shadow-sm);
		transition: box-shadow 0.25s ease;
	}

	.dw-actions-card:hover {
		box-shadow: var(--shadow-md);
	}

	/* ── Header ── */
	.dw-actions-header {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.dw-actions-icon-wrap {
		width: 2rem;
		height: 2rem;
		border-radius: var(--radius-sm);
		background: var(--dt);
		color: var(--dbg);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.dw-actions-title {
		font-size: var(--text-base);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	/* ── Grid ── */
	.dw-actions-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--space-3);
	}

	@media (min-width: 480px) {
		.dw-actions-grid {
			grid-template-columns: repeat(4, 1fr);
		}
	}

	/* ── Action card ── */
	.dw-action-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-5) var(--space-3);
		background: transparent;
		cursor: pointer;
		transition: transform 0.18s ease, background 0.18s ease;
		font-family: inherit;
	}

	.dw-action-card:hover,
	.dw-action-card--hovered {
		transform: translateY(-2px);
		background: var(--dbg2);
	}

	.dw-action-card:active {
		transform: translateY(0);
		background: var(--dbg3);
	}

	/* ── Icon ── */
	.dw-action-icon-wrap {
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.18s ease;
	}

	.dw-action-card:hover .dw-action-icon-wrap {
		transform: scale(1.1);
	}

	.dw-action-icon {
		width: 1.5rem;
		height: 1.5rem;
		color: var(--dt2);
		transition: color 0.15s;
	}

	.dw-action-card:hover .dw-action-icon {
		color: var(--dt);
	}

	/* ── Label ── */
	.dw-action-label {
		font-size: 0.82rem;
		font-weight: var(--font-semibold);
		color: var(--dt);
		text-align: center;
		line-height: 1.2;
	}

	/* ── Keyboard shortcut ── */
	.dw-action-shortcut {
		font-size: 0.68rem;
		color: var(--dt4);
		font-weight: var(--font-medium);
		letter-spacing: 0.02em;
	}

	@media (max-width: 479px) {
		.dw-action-shortcut {
			display: none;
		}
	}
</style>
