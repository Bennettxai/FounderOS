<script lang="ts">
	import OptimalGraphView from '$lib/components/knowledge/OptimalGraphView.svelte';

	interface Props {
		onSelectEntity?: (name: string, type: string) => void;
	}

	let { onSelectEntity }: Props = $props();
</script>

<OptimalGraphView
	onNodeSelect={(node) => {
		onSelectEntity?.(node.slug, 'node');
	}}
/>
