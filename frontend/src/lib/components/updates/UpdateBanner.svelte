<script lang="ts">
	import { onMount } from 'svelte';
	import { Download, FileText, RefreshCw, X } from 'lucide-svelte';
	import { updateStore } from '$lib/stores/updateStore';

	let showReleaseNotes = $state(false);

	onMount(() => {
		updateStore.init();
	});

	const visible = $derived(
		$updateStore.isDesktop &&
			['available', 'downloading', 'downloaded', 'unsupported'].includes($updateStore.state) &&
			($updateStore.state === 'unsupported' || $updateStore.latestVersion !== $updateStore.dismissedVersion)
	);

	function formatReleaseNotes(notes: unknown): string {
		if (!notes) return 'No release notes were attached to this update.';
		if (typeof notes === 'string') return notes.trim() || 'No release notes were attached to this update.';
		return JSON.stringify(notes, null, 2);
	}

	function close() {
		updateStore.dismiss($updateStore.latestVersion);
	}
</script>

{#if visible}
	<div class="update-banner" role="status" aria-live="polite">
		<div class="update-banner__icon" aria-hidden="true">
			{#if $updateStore.state === 'downloaded'}
				<RefreshCw size={18} />
			{:else}
				<Download size={18} />
			{/if}
		</div>
		<div class="update-banner__body">
			<p class="update-banner__title">
				{#if $updateStore.state === 'unsupported'}
					Required update
				{:else if $updateStore.state === 'downloaded'}
					Update ready
				{:else}
					BusinessOS {$updateStore.latestVersion} available
				{/if}
			</p>
			<p class="update-banner__meta">
				{#if $updateStore.state === 'downloading'}
					Downloading {Math.round($updateStore.progress?.percent ?? 0)}%
				{:else if $updateStore.state === 'unsupported'}
					This build is below {$updateStore.minimumSupportedVersion || 'the supported version'}.
				{:else if $updateStore.state === 'downloaded'}
					Restart to install {$updateStore.latestVersion}.
				{:else if $updateStore.currentVersion}
					Current {$updateStore.currentVersion} - {$updateStore.channel}
				{:else}
					Desktop update available.
				{/if}
			</p>
			{#if $updateStore.state === 'downloading'}
				<div class="update-banner__progress" aria-hidden="true">
					<div style={`width: ${Math.max(0, Math.min(100, $updateStore.progress?.percent ?? 0))}%`}></div>
				</div>
			{/if}
		</div>
		<div class="update-banner__actions">
			{#if $updateStore.releaseNotes}
				<button type="button" class="update-banner__ghost" onclick={() => (showReleaseNotes = true)} aria-label="View release notes">
					<FileText size={15} aria-hidden="true" />
				</button>
			{/if}
			{#if $updateStore.state === 'available'}
				<button type="button" class="btn-pill btn-pill-primary btn-pill-sm" onclick={() => updateStore.download()}>
					Download
				</button>
			{:else if $updateStore.state === 'unsupported'}
				<button type="button" class="btn-pill btn-pill-primary btn-pill-sm" onclick={() => updateStore.check()}>
					Check
				</button>
			{:else if $updateStore.state === 'downloaded'}
				<button type="button" class="btn-pill btn-pill-primary btn-pill-sm" onclick={() => updateStore.install()}>
					Install
				</button>
			{/if}
			<button type="button" class="update-banner__close" onclick={close} aria-label="Remind me later">
				<X size={16} aria-hidden="true" />
			</button>
		</div>
	</div>
{/if}

{#if showReleaseNotes}
	<div class="release-backdrop">
		<button type="button" class="release-scrim" onclick={() => (showReleaseNotes = false)} aria-label="Close release notes"></button>
		<div class="release-modal" role="dialog" aria-modal="true" aria-label="Release notes" tabindex="-1">
			<div class="release-modal__head">
				<div>
					<p>Release notes</p>
					<h2>BusinessOS {$updateStore.latestVersion}</h2>
				</div>
				<button type="button" onclick={() => (showReleaseNotes = false)} aria-label="Close release notes">×</button>
			</div>
			<pre>{formatReleaseNotes($updateStore.releaseNotes)}</pre>
			<div class="release-modal__actions">
				<button type="button" class="btn btn-secondary text-sm" onclick={() => (showReleaseNotes = false)}>
					Close
				</button>
				{#if $updateStore.state === 'available'}
					<button type="button" class="btn-pill btn-pill-primary btn-pill-sm" onclick={() => updateStore.download()}>
						Download
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.update-banner {
		position: fixed;
		right: 20px;
		bottom: 20px;
		z-index: 60;
		display: flex;
		align-items: center;
		gap: 12px;
		width: min(440px, calc(100vw - 40px));
		padding: 12px;
		background-color: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 14px;
		box-shadow: 0 18px 50px rgba(0, 0, 0, 0.24);
	}

	.update-banner__icon {
		flex: 0 0 auto;
		display: grid;
		place-items: center;
		width: 34px;
		height: 34px;
		border-radius: 9px;
		background-color: var(--dt);
		color: var(--dbg);
	}

	.update-banner__body {
		flex: 1;
		min-width: 0;
	}

	.update-banner__title {
		margin: 0;
		overflow: hidden;
		color: var(--dt);
		font-size: 0.86rem;
		font-weight: 720;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.update-banner__meta {
		margin: 2px 0 0;
		overflow: hidden;
		color: var(--dt3);
		font-size: 0.73rem;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.update-banner__actions {
		display: flex;
		flex: 0 0 auto;
		align-items: center;
		gap: 7px;
	}

	.update-banner__ghost,
	.update-banner__close {
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		border: 1px solid var(--dbd);
		border-radius: 8px;
		background: var(--dbg);
		color: var(--dt2);
		cursor: pointer;
	}

	.update-banner__close {
		background: transparent;
		color: var(--dt3);
	}

	.update-banner__ghost:hover,
	.update-banner__close:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	.update-banner__progress {
		height: 5px;
		margin-top: 8px;
		overflow: hidden;
		border-radius: 999px;
		background: var(--dbg);
		border: 1px solid var(--dbd);
	}

	.update-banner__progress div {
		height: 100%;
		border-radius: inherit;
		background: var(--dt);
		transition: width 0.18s ease;
	}

	.release-backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;
		display: grid;
		place-items: center;
		padding: 24px;
		background: rgba(0, 0, 0, 0.38);
	}

	.release-scrim {
		position: absolute;
		inset: 0;
		border: 0;
		background: transparent;
		cursor: default;
	}

	.release-modal {
		position: relative;
		width: min(640px, 100%);
		max-height: min(700px, calc(100vh - 48px));
		overflow: hidden;
		display: flex;
		flex-direction: column;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 16px;
		box-shadow: 0 24px 70px rgba(0, 0, 0, 0.36);
	}

	.release-modal__head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		padding: 20px 22px 14px;
		border-bottom: 1px solid var(--dbd);
	}

	.release-modal__head p {
		margin: 0 0 4px;
		color: var(--dt3);
		font-size: 0.68rem;
		font-weight: 760;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.release-modal__head h2 {
		margin: 0;
		color: var(--dt);
		font-size: 1.05rem;
		font-weight: 740;
	}

	.release-modal__head button {
		width: 30px;
		height: 30px;
		border: 1px solid var(--dbd);
		border-radius: 8px;
		background: var(--dbg2);
		color: var(--dt2);
		font-size: 1.2rem;
		line-height: 1;
		cursor: pointer;
	}

	.release-modal pre {
		flex: 1;
		min-height: 180px;
		max-height: 420px;
		overflow: auto;
		margin: 0;
		padding: 18px 22px;
		white-space: pre-wrap;
		color: var(--dt2);
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		font-size: 0.82rem;
		line-height: 1.55;
	}

	.release-modal__actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding: 14px 22px 20px;
		border-top: 1px solid var(--dbd);
	}

	@media (max-width: 560px) {
		.update-banner {
			align-items: flex-start;
		}

		.update-banner__actions {
			flex-wrap: wrap;
			justify-content: flex-end;
		}
	}
</style>
