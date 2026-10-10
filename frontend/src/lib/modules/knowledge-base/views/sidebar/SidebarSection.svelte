<script lang="ts">
	import { type Snippet } from 'svelte';
	import { ChevronRight } from 'lucide-svelte';

	interface Props {
		title: string;
		collapsible?: boolean;
		defaultExpanded?: boolean;
		children: Snippet;
		actions?: Snippet;
	}

	let {
		title,
		collapsible = false,
		defaultExpanded = true,
		children,
		actions
	}: Props = $props();

	let isExpanded = $state(defaultExpanded);

	function toggleExpanded() {
		if (collapsible) {
			isExpanded = !isExpanded;
		}
	}
</script>

<div class="sidebar-section">
	<button
		class="sidebar-section__header"
		class:sidebar-section__header--collapsible={collapsible}
		onclick={toggleExpanded}
		disabled={!collapsible}
	>
		{#if collapsible}
			<ChevronRight
				class="sidebar-section__chevron {isExpanded ? 'sidebar-section__chevron--expanded' : ''}"
			/>
		{/if}
		<span class="sidebar-section__title">{title}</span>
		{#if actions}
			<div class="sidebar-section__actions" onclick={(e) => e.stopPropagation()}>
				{@render actions()}
			</div>
		{/if}
	</button>

	<div class="sidebar-section__content" class:sidebar-section__content--collapsed={!isExpanded}>
		{@render children()}
	</div>
</div>

<style>
	.sidebar-section {
		padding: 0.25rem 0;
	}

	.sidebar-section__header {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		width: 100%;
		padding: 0.375rem 0.75rem;
		background: transparent;
		border: none;
		cursor: default;
		text-align: left;
	}

	.sidebar-section__header--collapsible {
		cursor: pointer;
	}

	.sidebar-section__header--collapsible:hover {
		background-color: var(--dbg3);
	}

	.sidebar-section__chevron {
		width: 14px;
		height: 14px;
		color: var(--dt3);
		transition: transform 0.15s ease;
	}

	.sidebar-section__chevron--expanded {
		transform: rotate(90deg);
	}

	.sidebar-section__title {
		flex: 1;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--dt3);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.sidebar-section__actions {
		display: flex;
		gap: 0.25rem;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.sidebar-section__header:hover .sidebar-section__actions {
		opacity: 1;
	}

	.sidebar-section__content {
		padding: 0.25rem 0;
		overflow: hidden;
		max-height: 2000px;
		opacity: 1;
		transition: max-height 0.25s ease, opacity 0.2s ease;
	}

	.sidebar-section__content--collapsed {
		max-height: 0;
		opacity: 0;
		padding: 0;
	}

	:global(.sidebar-section__chevron) {
		width: 14px;
		height: 14px;
	}
</style>
