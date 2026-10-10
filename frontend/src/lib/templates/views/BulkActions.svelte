<script lang="ts">
	/**
	 * BulkActions - Bulk action bar for app templates
	 */

	import type { BulkAction } from '../types/view';
	import { TemplateButton, TemplateDropdown } from '../primitives';

	interface Props {
		selectedCount: number;
		actions: BulkAction[];
		ondeselect?: () => void;
	}

	let {
		selectedCount,
		actions,
		ondeselect
	}: Props = $props();

	const primaryActions = $derived(actions.slice(0, 3));
	const overflowActions = $derived(actions.slice(3));
</script>

{#if selectedCount > 0}
	<div class="tpl-bulk-actions">
		<div class="tpl-bulk-info">
			<button class="tpl-bulk-close" onclick={ondeselect}>
				<svg viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
				</svg>
			</button>
			<span class="tpl-bulk-count">{selectedCount} selected</span>
		</div>

		<div class="tpl-bulk-divider"></div>

		<div class="tpl-bulk-buttons">
			{#each primaryActions as action}
				<TemplateButton
					variant={action.variant === 'danger' ? 'danger' : 'ghost'}
					size="sm"
					onclick={() => action.action([])}
				>
					{#if action.icon}
						<span class="tpl-bulk-icon">{action.icon}</span>
					{/if}
					{action.label}
				</TemplateButton>
			{/each}

			{#if overflowActions.length > 0}
				<TemplateDropdown
					items={overflowActions.map(a => ({
						id: a.id,
						label: a.label,
						icon: a.icon,
						danger: a.variant === 'danger'
					}))}
					onselect={(item) => {
						const action = actions.find(a => a.id === item.id);
						action?.action([]);
					}}
				>
					{#snippet trigger()}
						<TemplateButton variant="ghost" size="sm">
							<svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
								<path d="M6 10a2 2 0 11-4 0 2 2 0 014 0zM12 10a2 2 0 11-4 0 2 2 0 014 0zM16 12a2 2 0 100-4 2 2 0 000 4z" />
							</svg>
						</TemplateButton>
					{/snippet}
				</TemplateDropdown>
			{/if}
		</div>
	</div>
{/if}

<style>
	.tpl-bulk-actions {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-3);
		padding: var(--tpl-space-2) var(--tpl-space-4);
		background: var(--tpl-accent-primary);
		border-radius: var(--tpl-radius-lg);
		animation: tpl-slide-up var(--tpl-transition-fast) ease-out;
	}

	.tpl-bulk-info {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-bulk-close {
		width: 24px;
		height: 24px;
		padding: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(255, 255, 255, 0.1);
		border: none;
		border-radius: var(--tpl-radius-full);
		color: white;
		cursor: pointer;
		transition: background var(--tpl-transition-fast);
	}

	.tpl-bulk-close:hover {
		background: rgba(255, 255, 255, 0.2);
	}

	.tpl-bulk-close svg {
		width: 14px;
		height: 14px;
	}

	.tpl-bulk-count {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: white;
		white-space: nowrap;
	}

	.tpl-bulk-divider {
		width: 1px;
		height: 24px;
		background: rgba(255, 255, 255, 0.2);
	}

	.tpl-bulk-buttons {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-1);
	}

	.tpl-bulk-buttons :global(.tpl-btn) {
		color: white;
	}

	.tpl-bulk-buttons :global(.tpl-btn:hover) {
		background: rgba(255, 255, 255, 0.1);
	}

	.tpl-bulk-icon {
		font-size: var(--tpl-text-base);
	}
</style>
