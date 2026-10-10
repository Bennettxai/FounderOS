<script lang="ts">
	/**
	 * TemplateModal - Modal dialog component for app templates
	 */

	type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | 'full';

	interface Props {
		open?: boolean;
		size?: ModalSize;
		title?: string;
		closable?: boolean;
		onclose?: () => void;
	}

	let {
		open = $bindable(false),
		size = 'md',
		title,
		closable = true,
		onclose,
		children
	}: Props & { children?: any } = $props();

	const sizeClasses: Record<ModalSize, string> = {
		sm: 'tpl-modal-sm',
		md: 'tpl-modal-md',
		lg: 'tpl-modal-lg',
		xl: 'tpl-modal-xl',
		full: 'tpl-modal-full'
	};

	function handleClose() {
		if (closable) {
			open = false;
			onclose?.();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && closable) {
			handleClose();
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			handleClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		class="tpl-modal-backdrop"
		onclick={handleBackdropClick}
		role="dialog"
		aria-modal="true"
		aria-labelledby={title ? 'modal-title' : undefined}
		tabindex="-1"
	>
		<div class="tpl-modal {sizeClasses[size]}">
			{#if title || closable}
				<div class="tpl-modal-header">
					{#if title}
						<h2 id="modal-title" class="tpl-modal-title">{title}</h2>
					{/if}
					{#if closable}
						<button type="button" class="tpl-modal-close" onclick={handleClose} aria-label="Close">
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M6 6l12 12M18 6l-12 12" />
							</svg>
						</button>
					{/if}
				</div>
			{/if}
			<div class="tpl-modal-body">
				{@render children?.()}
			</div>
		</div>
	</div>
{/if}

<style>
	.tpl-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: var(--tpl-z-modal);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--tpl-space-4);
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(4px);
		animation: tpl-modal-backdrop-in 0.2s ease-out;
	}

	@keyframes tpl-modal-backdrop-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	.tpl-modal {
		background: var(--tpl-bg-primary);
		border-radius: var(--tpl-radius-xl);
		box-shadow: var(--tpl-shadow-xl);
		max-height: calc(100vh - var(--tpl-space-8));
		display: flex;
		flex-direction: column;
		animation: tpl-modal-in 0.2s ease-out;
	}

	@keyframes tpl-modal-in {
		from {
			opacity: 0;
			transform: scale(0.95) translateY(-10px);
		}
		to {
			opacity: 1;
			transform: scale(1) translateY(0);
		}
	}

	/* Sizes */
	.tpl-modal-sm { width: 400px; }
	.tpl-modal-md { width: 500px; }
	.tpl-modal-lg { width: 640px; }
	.tpl-modal-xl { width: 800px; }
	.tpl-modal-full { width: calc(100vw - var(--tpl-space-8)); height: calc(100vh - var(--tpl-space-8)); }

	.tpl-modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--tpl-space-3);
		padding: var(--tpl-space-4) var(--tpl-space-5);
		border-bottom: 1px solid var(--tpl-border-default);
		min-height: 56px;
	}

	.tpl-modal-title {
		margin: 0;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-lg);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-modal-close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: var(--tpl-size-sm); /* 32px */
		height: var(--tpl-size-sm);
		padding: 0;
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-md);
		color: var(--tpl-text-muted);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
		flex-shrink: 0;
	}

	.tpl-modal-close:hover {
		background: var(--tpl-bg-hover);
		color: var(--tpl-text-primary);
	}

	.tpl-modal-close:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-modal-close svg {
		width: var(--tpl-icon-md);
		height: var(--tpl-icon-md);
	}

	.tpl-modal-body {
		flex: 1;
		padding: var(--tpl-space-5);
		overflow-y: auto;
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   FOOTER (optional, styled via class)
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-modal :global(.tpl-modal-footer) {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: var(--tpl-space-2);
		padding: var(--tpl-space-4) var(--tpl-space-5);
		border-top: 1px solid var(--tpl-border-default);
	}
</style>
