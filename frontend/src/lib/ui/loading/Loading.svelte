<script lang="ts">
	type LoadingSize = 'xs' | 'sm' | 'default' | 'lg';

	interface Props {
		size?: LoadingSize;
		class?: string;
	}

	let { size = 'default', class: className = '' }: Props = $props();
</script>

<svg
	class="bos-loading bos-loading--{size} {className}"
	xmlns="http://www.w3.org/2000/svg"
	fill="none"
	viewBox="0 0 24 24"
>
	<circle
		class="bos-loading__track"
		cx="12"
		cy="12"
		r="10"
		stroke="currentColor"
		stroke-width="4"
	></circle>
	<path
		class="bos-loading__arc"
		fill="currentColor"
		d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
	></path>
</svg>

<style>
	.bos-loading {
		animation: spin 1s linear infinite;
	}

	.bos-loading--xs { width: 12px; height: 12px; }
	.bos-loading--sm { width: 16px; height: 16px; }
	.bos-loading--default { width: 20px; height: 20px; }
	.bos-loading--lg { width: 24px; height: 24px; }

	.bos-loading__track { opacity: 0.25; }
	.bos-loading__arc { opacity: 0.75; }

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
