<script lang="ts">
	import PageListLayout from './PageListLayout.svelte';

	interface Props {
		title: string;
		message: string;
	}

	let { title, message }: Props = $props();
</script>

<PageListLayout {title}>
	<div class="kb-empty">
		<p>{message}</p>
	</div>
</PageListLayout>

<style>
	.kb-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		text-align: center;
		flex: 1;
		padding: 64px 32px;
		color: var(--bos-v2-text-secondary, #8e8d91);
		font-size: 14px;
		line-height: 1.5;
	}

	.kb-empty p {
		margin: 0;
	}
</style>
