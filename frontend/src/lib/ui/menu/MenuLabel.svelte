<script lang="ts">
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<!-- Menu label - a simple text element for labeling groups -->
<div role="none" class={cn('bos-menu-label', className)}>
	{@render children()}
</div>

<style>
	:global(.bos-menu-label) {
		padding: 4px 8px 6px;
		font-size: 13px;
		font-weight: 600;
		color: var(--dt2);
	}
</style>
