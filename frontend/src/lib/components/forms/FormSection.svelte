<script lang="ts">
	interface Props {
		title: string;
		description?: string;
	}

	let { title, description = '' }: Props = $props();
</script>

<div class="form-section">
	<div class="section-header">
		<h3 class="section-title">{title}</h3>
		{#if description}
			<p class="section-description">{description}</p>
		{/if}
	</div>

	<div class="section-content">
		<slot />
	</div>
</div>

<style>
	.form-section {
		@apply space-y-4;
	}

	.section-header {
		@apply border-b border-base-300 pb-4;
	}

	.section-title {
		@apply text-lg font-semibold text-base-content;
	}

	.section-description {
		@apply text-sm text-base-content/60 mt-1;
	}

	.section-content {
		@apply space-y-4;
	}
</style>
