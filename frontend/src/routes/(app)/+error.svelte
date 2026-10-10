<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
</script>

<div class="error-shell flex-1 flex items-center justify-center p-8">
	<div class="error-card">
		<p class="error-code">{$page.status}</p>
		<h1 class="error-title">
			{$page.status === 404 ? 'Page not found' : 'Something went wrong'}
		</h1>
		<p class="error-message">
			{$page.error?.message || (
				$page.status === 404
					? 'The page you are looking for does not exist or has been moved.'
					: 'An unexpected error occurred. Try going back to the app.'
			)}
		</p>
		<div class="error-actions">
			<button
				class="error-btn error-btn--primary"
				onclick={() => goto('/')}
			>
				Go to app
			</button>
			<button
				class="error-btn error-btn--ghost"
				onclick={() => history.back()}
			>
				Go back
			</button>
		</div>
	</div>
</div>

<style>
	.error-shell {
		background: var(--dbg);
		min-height: 0;
	}

	.error-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		max-width: 400px;
		gap: 12px;
	}

	.error-code {
		font-size: 4rem;
		font-weight: 700;
		line-height: 1;
		color: var(--dbd);
		letter-spacing: -0.04em;
	}

	.error-title {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.error-message {
		font-size: 0.9rem;
		color: var(--dt3);
		line-height: 1.6;
		margin: 0;
	}

	.error-actions {
		display: flex;
		gap: 8px;
		margin-top: 8px;
	}

	.error-btn {
		padding: 8px 18px;
		border-radius: 8px;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		border: none;
		transition: background 150ms ease, color 150ms ease, border-color 150ms ease;
	}

	.error-btn--primary {
		background: var(--dt);
		color: var(--dbg);
	}
	.error-btn--primary:hover {
		opacity: 0.88;
	}

	.error-btn--ghost {
		background: transparent;
		color: var(--dt2);
		border: 1px solid var(--dbd);
	}
	.error-btn--ghost:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
		color: var(--dt);
	}
</style>
