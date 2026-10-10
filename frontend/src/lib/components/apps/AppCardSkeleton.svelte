<script lang="ts">
	interface Props {
		count?: number;
	}

	let { count = 1 }: Props = $props();
</script>

{#each Array(count) as _, i (i)}
	<div class="app-card-skeleton" role="presentation" aria-hidden="true">
		<!-- Header Row -->
		<div class="skeleton-header">
			<div class="skeleton-icon"></div>
			<div class="skeleton-title"></div>
		</div>

		<!-- Description -->
		<div class="skeleton-description">
			<div class="skeleton-line"></div>
			<div class="skeleton-line short"></div>
		</div>

		<!-- Footer -->
		<div class="skeleton-footer">
			<div class="skeleton-version"></div>
			<div class="skeleton-badge"></div>
		</div>
	</div>
{/each}

<style>
	.app-card-skeleton {
		position: relative;
		display: flex;
		flex-direction: column;
		padding: 16px;
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 12px;
		overflow: hidden;
	}

	:global(.dark) .app-card-skeleton {
		background: #1f2937;
		border-color: #374151;
	}

	/* Shimmer animation */
	.app-card-skeleton::after {
		content: '';
		position: absolute;
		top: 0;
		left: -100%;
		width: 100%;
		height: 100%;
		background: linear-gradient(
			90deg,
			transparent,
			rgba(255, 255, 255, 0.4),
			transparent
		);
		animation: shimmer 1.5s infinite;
	}

	:global(.dark) .app-card-skeleton::after {
		background: linear-gradient(
			90deg,
			transparent,
			rgba(255, 255, 255, 0.1),
			transparent
		);
	}

	@keyframes shimmer {
		0% {
			left: -100%;
		}
		100% {
			left: 100%;
		}
	}

	.skeleton-header {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 12px;
	}

	.skeleton-icon {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		background: #e5e7eb;
	}

	:global(.dark) .skeleton-icon {
		background: #374151;
	}

	.skeleton-title {
		flex: 1;
		height: 20px;
		border-radius: 4px;
		background: #e5e7eb;
		max-width: 60%;
	}

	:global(.dark) .skeleton-title {
		background: #374151;
	}

	.skeleton-description {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-bottom: auto;
		min-height: 40px;
	}

	.skeleton-line {
		height: 14px;
		border-radius: 4px;
		background: #e5e7eb;
	}

	:global(.dark) .skeleton-line {
		background: #374151;
	}

	.skeleton-line.short {
		width: 70%;
	}

	.skeleton-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid #f3f4f6;
	}

	:global(.dark) .skeleton-footer {
		border-top-color: #374151;
	}

	.skeleton-version {
		width: 40px;
		height: 14px;
		border-radius: 4px;
		background: #e5e7eb;
	}

	:global(.dark) .skeleton-version {
		background: #374151;
	}

	.skeleton-badge {
		width: 60px;
		height: 22px;
		border-radius: 9999px;
		background: #e5e7eb;
	}

	:global(.dark) .skeleton-badge {
		background: #374151;
	}
</style>
