<script lang="ts">
	import { Globe, Lock, Copy, Check, ExternalLink } from 'lucide-svelte';
	import { enableSharing, disableSharing } from '../../services/documents.service';

	interface Props {
		open: boolean;
		documentId: string;
		documentTitle: string;
		isPublic: boolean;
		shareId: string | null;
	}

	let { open = $bindable(false), documentId, documentTitle, isPublic, shareId }: Props = $props();

	let isToggling = $state(false);
	let copied = $state(false);
	let error = $state<string | null>(null);

	const shareUrl = $derived(shareId ? `${window.location.origin}/pages/public/${shareId}` : null);

	async function handleToggleShare() {
		isToggling = true;
		error = null;
		try {
			if (isPublic) {
				await disableSharing(documentId);
			} else {
				await enableSharing(documentId);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update sharing';
		} finally {
			isToggling = false;
		}
	}

	function handleCopyLink() {
		if (!shareUrl) return;
		navigator.clipboard.writeText(shareUrl);
		copied = true;
		setTimeout(() => { copied = false; }, 2000);
	}

	function handleClose() {
		open = false;
		error = null;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) handleClose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div class="share-backdrop" onclick={handleBackdropClick} onkeydown={handleKeydown} role="dialog" aria-modal="true" aria-label="Share document">
		<div class="share-modal">
			<div class="share-modal__header">
				<h3 class="share-modal__title">Share "{documentTitle || 'Untitled'}"</h3>
				<button class="share-modal__close" onclick={handleClose} aria-label="Close">×</button>
			</div>

			<div class="share-modal__body">
				<div class="share-modal__toggle-row">
					<div class="share-modal__toggle-info">
						{#if isPublic}
							<Globe class="share-modal__icon share-modal__icon--public" />
						{:else}
							<Lock class="share-modal__icon" />
						{/if}
						<div>
							<span class="share-modal__label">
								{isPublic ? 'Published to web' : 'Private'}
							</span>
							<span class="share-modal__desc">
								{isPublic ? 'Anyone with the link can view' : 'Only you can access this page'}
							</span>
						</div>
					</div>
					<button
						class="share-modal__toggle-btn"
						class:share-modal__toggle-btn--active={isPublic}
						onclick={handleToggleShare}
						disabled={isToggling}
					>
						<span class="share-modal__toggle-knob"></span>
					</button>
				</div>

				{#if error}
					<p class="share-modal__error">{error}</p>
				{/if}

				{#if isPublic && shareUrl}
					<div class="share-modal__link-row">
						<input
							type="text"
							class="share-modal__link-input"
							value={shareUrl}
							readonly
							onclick={(e) => (e.target as HTMLInputElement).select()}
						/>
						<button class="share-modal__copy-btn" onclick={handleCopyLink}>
							{#if copied}
								<Check class="h-4 w-4" />
								Copied
							{:else}
								<Copy class="h-4 w-4" />
								Copy
							{/if}
						</button>
					</div>

					<a href={shareUrl} target="_blank" rel="noopener noreferrer" class="share-modal__open-link">
						<ExternalLink class="h-3.5 w-3.5" />
						Open public page
					</a>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.share-backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(0, 0, 0, 0.5);
		animation: shareBackdropIn 0.15s ease;
	}

	@keyframes shareBackdropIn {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	.share-modal {
		width: 420px;
		max-width: 90vw;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
		animation: shareModalIn 0.2s ease;
	}

	@keyframes shareModalIn {
		from { opacity: 0; transform: scale(0.95) translateY(8px); }
		to { opacity: 1; transform: scale(1) translateY(0); }
	}

	.share-modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px 12px;
		border-bottom: 1px solid var(--dbd);
	}

	.share-modal__title {
		font-size: 15px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.share-modal__close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		border-radius: 6px;
		color: var(--dt3);
		font-size: 18px;
		cursor: pointer;
	}

	.share-modal__close:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	.share-modal__body {
		padding: 16px 20px 20px;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.share-modal__toggle-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
	}

	.share-modal__toggle-info {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.share-modal__icon {
		width: 20px;
		height: 20px;
		color: var(--dt3);
		flex-shrink: 0;
	}

	.share-modal__icon--public {
		color: #22c55e;
	}

	.share-modal__label {
		display: block;
		font-size: 14px;
		font-weight: 500;
		color: var(--dt);
	}

	.share-modal__desc {
		display: block;
		font-size: 12px;
		color: var(--dt3);
		margin-top: 1px;
	}

	.share-modal__toggle-btn {
		position: relative;
		width: 44px;
		height: 24px;
		border: none;
		border-radius: 12px;
		background: var(--dbg3);
		cursor: pointer;
		transition: background 0.2s;
		flex-shrink: 0;
	}

	.share-modal__toggle-btn--active {
		background: #1e96eb;
	}

	.share-modal__toggle-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.share-modal__toggle-knob {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: white;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
		transition: transform 0.2s;
	}

	.share-modal__toggle-btn--active .share-modal__toggle-knob {
		transform: translateX(20px);
	}

	.share-modal__error {
		font-size: 13px;
		color: #ef4444;
		margin: 0;
	}

	.share-modal__link-row {
		display: flex;
		gap: 8px;
	}

	.share-modal__link-input {
		flex: 1;
		height: 36px;
		padding: 0 10px;
		border: 1px solid var(--dbd);
		border-radius: 8px;
		background: var(--dbg2);
		color: var(--dt);
		font-size: 13px;
		outline: none;
	}

	.share-modal__link-input:focus {
		border-color: #1e96eb;
	}

	.share-modal__copy-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 0 14px;
		height: 36px;
		border: none;
		border-radius: 8px;
		background: #1e96eb;
		color: white;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.15s;
	}

	.share-modal__copy-btn:hover {
		background: #1a82d0;
	}

	.share-modal__open-link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: #1e96eb;
		text-decoration: none;
	}

	.share-modal__open-link:hover {
		text-decoration: underline;
	}
</style>
