<script lang="ts">
	/**
	 * WikiPanel
	 * Sidebar panel that surfaces wiki curation, rendering, and verification
	 * for the currently active Page/document.
	 *
	 * Mount inside the [id]/+page.svelte right-side column, only when a
	 * document is loaded. The panel is self-contained: it owns all wiki state
	 * and calls the /api/wiki/* endpoints directly.
	 *
	 * Slug derivation rule (spec-compliant):
	 *   1. If doc.share_id is set  →  use share_id as-is
	 *   2. Otherwise               →  slugify doc.title (lowercase, spaces→hyphens,
	 *                                  strip non-alphanumeric except hyphens)
	 */

	import { onMount } from 'svelte';
	import {
		BookOpen,
		CheckCircle,
		AlertCircle,
		Info,
		ChevronDown,
		ChevronUp,
		Loader2,
		X,
		Plus,
		Trash2
	} from 'lucide-svelte';
	import type { Document } from '$lib/modules/knowledge-base/entities/types';
	import {
		getWikiPage,
		curateWikiPage,
		renderWikiPage,
		verifyWikiPage
	} from '$lib/api/wiki';
	import type {
		WikiPage,
		WikiCitation,
		RenderFormat,
		RenderResponse,
		VerifyResponse,
		VerifyIssue
	} from '$lib/api/wiki';

	// -------------------------------------------------------------------------
	// Props
	// -------------------------------------------------------------------------

	interface Props {
		doc: Document;
	}

	let { doc }: Props = $props();

	// -------------------------------------------------------------------------
	// Slug derivation
	// -------------------------------------------------------------------------

	function slugify(text: string): string {
		return text
			.toLowerCase()
			.trim()
			.replace(/[^a-z0-9\s-]/g, '')
			.replace(/\s+/g, '-')
			.replace(/-+/g, '-')
			.replace(/^-|-$/g, '');
	}

	let slug = $derived(
		doc.share_id ? doc.share_id : slugify(doc.title || doc.id)
	);

	// -------------------------------------------------------------------------
	// Wiki status
	// -------------------------------------------------------------------------

	let wikiPage = $state<WikiPage | null>(null);
	let statusLoading = $state(true);
	let statusError = $state<string | null>(null);

	async function fetchStatus() {
		statusLoading = true;
		statusError = null;
		try {
			wikiPage = await getWikiPage(slug, 'default');
		} catch {
			// 404 or any error → not yet curated
			wikiPage = null;
		} finally {
			statusLoading = false;
		}
	}

	// Re-fetch whenever the slug changes (i.e. the document changes)
	$effect(() => {
		// Access slug reactively
		const _slug = slug;
		void _slug;
		fetchStatus();
	});

	// -------------------------------------------------------------------------
	// Curate modal state
	// -------------------------------------------------------------------------

	let showCurateModal = $state(false);
	let curateAudience = $state('default');
	let curateCitations = $state<WikiCitation[]>([]);
	let curateLoading = $state(false);
	let curateError = $state<string | null>(null);

	function openCurateModal() {
		curateAudience = 'default';
		curateCitations = wikiPage?.citations ? [...wikiPage.citations] : [];
		curateError = null;
		showCurateModal = true;
	}

	function addCitation() {
		curateCitations = [...curateCitations, { chunk_id: '', claim_hash: '' }];
	}

	function removeCitation(index: number) {
		curateCitations = curateCitations.filter((_, i) => i !== index);
	}

	function updateCitationField(
		index: number,
		field: keyof WikiCitation,
		value: string
	) {
		curateCitations = curateCitations.map((c, i) =>
			i === index ? { ...c, [field]: value } : c
		);
	}

	async function submitCurate() {
		curateLoading = true;
		curateError = null;
		try {
			// Build the plain-text body from the document blocks
			const body = doc.content
				.map((block) =>
					Array.isArray(block.content)
						? block.content.map((rt) => rt.plain_text ?? '').join('')
						: ''
				)
				.join('\n\n');

			const citations = curateCitations
				.filter((c) => c.chunk_id.trim())
				.map((c) => ({
					chunk_id: c.chunk_id.trim(),
					...(c.claim_hash?.trim() ? { claim_hash: c.claim_hash.trim() } : {})
				}));

			await curateWikiPage({
				slug,
				audience: curateAudience.trim() || 'default',
				body,
				citations
			});

			showCurateModal = false;
			await fetchStatus();
		} catch (e) {
			curateError = e instanceof Error ? e.message : 'Curate failed';
		} finally {
			curateLoading = false;
		}
	}

	// -------------------------------------------------------------------------
	// Render modal state
	// -------------------------------------------------------------------------

	let showRenderModal = $state(false);
	let renderFormat = $state<RenderFormat>('markdown');
	let renderAudience = $state('default');
	let renderLoading = $state(false);
	let renderResult = $state<RenderResponse | null>(null);
	let renderError = $state<string | null>(null);

	function openRenderModal() {
		renderFormat = 'markdown';
		renderAudience = wikiPage?.audience ?? 'default';
		renderResult = null;
		renderError = null;
		showRenderModal = true;
	}

	async function submitRender() {
		renderLoading = true;
		renderError = null;
		renderResult = null;
		try {
			renderResult = await renderWikiPage(slug, {
				audience: renderAudience.trim() || 'default',
				format: renderFormat
			});
		} catch (e) {
			renderError = e instanceof Error ? e.message : 'Render failed';
		} finally {
			renderLoading = false;
		}
	}

	// -------------------------------------------------------------------------
	// Verify modal state
	// -------------------------------------------------------------------------

	let showVerifyModal = $state(false);
	let verifyAudience = $state('default');
	let verifyLoading = $state(false);
	let verifyResult = $state<VerifyResponse | null>(null);
	let verifyError = $state<string | null>(null);

	function openVerifyModal() {
		verifyAudience = wikiPage?.audience ?? 'default';
		verifyResult = null;
		verifyError = null;
		showVerifyModal = true;
	}

	async function submitVerify() {
		verifyLoading = true;
		verifyError = null;
		verifyResult = null;
		try {
			verifyResult = await verifyWikiPage(slug, {
				audience: verifyAudience.trim() || 'default'
			});
		} catch (e) {
			verifyError = e instanceof Error ? e.message : 'Verify failed';
		} finally {
			verifyLoading = false;
		}
	}

	// -------------------------------------------------------------------------
	// Helpers
	// -------------------------------------------------------------------------

	function severityIcon(severity: VerifyIssue['severity']) {
		if (severity === 'error') return AlertCircle;
		if (severity === 'warning') return AlertCircle;
		return Info;
	}

	function severityClass(severity: VerifyIssue['severity']) {
		if (severity === 'error') return 'wp-issue--error';
		if (severity === 'warning') return 'wp-issue--warning';
		return 'wp-issue--info';
	}
</script>

<!-- =========================================================================
     Panel
========================================================================== -->
<aside class="wp-panel" aria-label="Wiki panel">
	<!-- Header -->
	<div class="wp-panel__header">
		<BookOpen class="wp-panel__header-icon" size={14} />
		<span class="wp-panel__header-title">Wiki</span>
	</div>

	<!-- Status row -->
	<div class="wp-panel__status">
		{#if statusLoading}
			<Loader2 class="wp-spinner" size={13} />
			<span class="wp-panel__status-text">Checking...</span>
		{:else if wikiPage}
			<CheckCircle class="wp-panel__status-ok" size={13} />
			<span class="wp-panel__status-text">
				Wiki version: v{wikiPage.version}
			</span>
		{:else}
			<span class="wp-panel__status-text wp-panel__status-text--muted">
				Not yet curated
			</span>
		{/if}
	</div>

	<!-- Slug -->
	<div class="wp-panel__slug">
		<span class="wp-panel__slug-label">Slug</span>
		<code class="wp-panel__slug-value">{slug}</code>
	</div>

	<!-- Actions -->
	<div class="wp-panel__actions">
		<button class="wp-btn wp-btn--primary" onclick={openCurateModal} aria-label="Curate this page">
			Curate
		</button>
		<button
			class="wp-btn"
			onclick={openRenderModal}
			disabled={!wikiPage}
			aria-label="Render wiki page"
		>
			Render
		</button>
		<button
			class="wp-btn"
			onclick={openVerifyModal}
			disabled={!wikiPage}
			aria-label="Verify wiki page"
		>
			Verify
		</button>
	</div>
</aside>

<!-- =========================================================================
     Curate Modal
========================================================================== -->
{#if showCurateModal}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="wp-overlay" onclick={() => (showCurateModal = false)} aria-label="Close curate modal" role="dialog" tabindex="-1">
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_no_noninteractive_element_interactions -->
		<div class="wp-modal" onclick={(e) => e.stopPropagation()} role="document">
			<div class="wp-modal__header">
				<span class="wp-modal__title">Curate Wiki Page</span>
				<button class="wp-icon-btn" onclick={() => (showCurateModal = false)} aria-label="Close">
					<X size={15} />
				</button>
			</div>

			<div class="wp-modal__body">
				<!-- Slug (read-only) -->
				<div class="wp-field">
					<label class="wp-label" for="curate-slug">Slug</label>
					<input id="curate-slug" class="wp-input wp-input--readonly" type="text" value={slug} readonly />
				</div>

				<!-- Audience -->
				<div class="wp-field">
					<label class="wp-label" for="curate-audience">Audience</label>
					<input
						id="curate-audience"
						class="wp-input"
						type="text"
						bind:value={curateAudience}
						placeholder="default"
					/>
				</div>

				<!-- Citations -->
				<div class="wp-field">
					<div class="wp-field__row">
						<span class="wp-label">Citations</span>
						<button class="wp-ghost-btn" onclick={addCitation} aria-label="Add citation">
							<Plus size={13} />
							Add
						</button>
					</div>

					{#each curateCitations as citation, i (i)}
						<div class="wp-citation-row">
							<input
								class="wp-input wp-input--sm"
								type="text"
								placeholder="chunk_id"
								value={citation.chunk_id}
								oninput={(e) =>
									updateCitationField(i, 'chunk_id', (e.currentTarget as HTMLInputElement).value)}
							/>
							<input
								class="wp-input wp-input--sm"
								type="text"
								placeholder="claim_hash (optional)"
								value={citation.claim_hash ?? ''}
								oninput={(e) =>
									updateCitationField(
										i,
										'claim_hash',
										(e.currentTarget as HTMLInputElement).value
									)}
							/>
							<button class="wp-icon-btn" onclick={() => removeCitation(i)} aria-label="Remove citation">
								<Trash2 size={13} />
							</button>
						</div>
					{/each}
				</div>

				{#if curateError}
					<div class="wp-error">{curateError}</div>
				{/if}
			</div>

			<div class="wp-modal__footer">
				<button class="wp-btn" onclick={() => (showCurateModal = false)}>Cancel</button>
				<button class="wp-btn wp-btn--primary" onclick={submitCurate} disabled={curateLoading}>
					{#if curateLoading}
						<Loader2 class="wp-spinner" size={13} />
						Curating...
					{:else}
						Curate
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- =========================================================================
     Render Modal
========================================================================== -->
{#if showRenderModal}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="wp-overlay" onclick={() => (showRenderModal = false)} role="dialog" aria-label="Close render modal" tabindex="-1">
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_no_noninteractive_element_interactions -->
		<div class="wp-modal wp-modal--lg" onclick={(e) => e.stopPropagation()} role="document">
			<div class="wp-modal__header">
				<span class="wp-modal__title">Render Wiki Page</span>
				<button class="wp-icon-btn" onclick={() => (showRenderModal = false)} aria-label="Close">
					<X size={15} />
				</button>
			</div>

			<div class="wp-modal__body">
				<!-- Audience -->
				<div class="wp-field">
					<label class="wp-label" for="render-audience">Audience</label>
					<input
						id="render-audience"
						class="wp-input"
						type="text"
						bind:value={renderAudience}
						placeholder="default"
					/>
				</div>

				<!-- Format -->
				<div class="wp-field">
					<label class="wp-label" for="render-format">Format</label>
					<select id="render-format" class="wp-select" bind:value={renderFormat}>
						<option value="plain">Plain text</option>
						<option value="markdown">Markdown</option>
						<option value="claude">Claude</option>
						<option value="openai">OpenAI</option>
					</select>
				</div>

				{#if !renderResult}
					<button
						class="wp-btn wp-btn--primary"
						onclick={submitRender}
						disabled={renderLoading}
					>
						{#if renderLoading}
							<Loader2 class="wp-spinner" size={13} />
							Rendering...
						{:else}
							Render
						{/if}
					</button>
				{/if}

				{#if renderError}
					<div class="wp-error">{renderError}</div>
				{/if}

				{#if renderResult}
					<div class="wp-result">
						<div class="wp-result__header">
							<span class="wp-result__label">Result — {renderResult.format}</span>
							{#if renderResult.citations.length > 0}
								<span class="wp-result__label wp-result__label--muted">
									{renderResult.citations.length} citation{renderResult.citations.length !== 1 ? 's' : ''}
								</span>
							{/if}
						</div>
						<pre class="wp-result__body">{renderResult.body}</pre>
						{#if renderResult.citations.length > 0}
							<div class="wp-result__citations">
								{#each renderResult.citations as c, idx}
									<div class="wp-result__citation">
										<span class="wp-result__citation-idx">[{idx + 1}]</span>
										<code class="wp-result__citation-chunk">{c.chunk_id}</code>
										{#if c.claim_hash}
											<code class="wp-result__citation-hash">{c.claim_hash}</code>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<div class="wp-modal__footer">
				<button class="wp-btn" onclick={() => (showRenderModal = false)}>Close</button>
				{#if renderResult}
					<button
						class="wp-btn wp-btn--primary"
						onclick={submitRender}
						disabled={renderLoading}
					>
						Re-render
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<!-- =========================================================================
     Verify Modal
========================================================================== -->
{#if showVerifyModal}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div class="wp-overlay" onclick={() => (showVerifyModal = false)} role="dialog" aria-label="Close verify modal" tabindex="-1">
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions a11y_no_noninteractive_element_interactions -->
		<div class="wp-modal" onclick={(e) => e.stopPropagation()} role="document">
			<div class="wp-modal__header">
				<span class="wp-modal__title">Verify Wiki Page</span>
				<button class="wp-icon-btn" onclick={() => (showVerifyModal = false)} aria-label="Close">
					<X size={15} />
				</button>
			</div>

			<div class="wp-modal__body">
				<!-- Audience -->
				<div class="wp-field">
					<label class="wp-label" for="verify-audience">Audience</label>
					<input
						id="verify-audience"
						class="wp-input"
						type="text"
						bind:value={verifyAudience}
						placeholder="default"
					/>
				</div>

				{#if !verifyResult}
					<button
						class="wp-btn wp-btn--primary"
						onclick={submitVerify}
						disabled={verifyLoading}
					>
						{#if verifyLoading}
							<Loader2 class="wp-spinner" size={13} />
							Verifying...
						{:else}
							Run Verify
						{/if}
					</button>
				{/if}

				{#if verifyError}
					<div class="wp-error">{verifyError}</div>
				{/if}

				{#if verifyResult}
					<div class="wp-verify">
						<!-- Overall status -->
						<div class="wp-verify__status" class:wp-verify__status--ok={verifyResult.ok} class:wp-verify__status--fail={!verifyResult.ok}>
							{#if verifyResult.ok}
								<CheckCircle size={15} />
								<span>All checks passed</span>
							{:else}
								<AlertCircle size={15} />
								<span>{verifyResult.issues.length} issue{verifyResult.issues.length !== 1 ? 's' : ''} found</span>
							{/if}
						</div>

						<!-- Issue list -->
						{#if verifyResult.issues.length > 0}
							<ul class="wp-issue-list">
								{#each verifyResult.issues as issue}
									{@const IssueIcon = severityIcon(issue.severity)}
									<li class="wp-issue {severityClass(issue.severity)}">
										<IssueIcon size={13} class="wp-issue__icon" />
										<div class="wp-issue__body">
											<span class="wp-issue__code">{issue.code}</span>
											<span class="wp-issue__msg">{issue.message}</span>
										</div>
									</li>
								{/each}
							</ul>
						{/if}
					</div>
				{/if}
			</div>

			<div class="wp-modal__footer">
				<button class="wp-btn" onclick={() => (showVerifyModal = false)}>Close</button>
				{#if verifyResult}
					<button
						class="wp-btn wp-btn--primary"
						onclick={submitVerify}
						disabled={verifyLoading}
					>
						Re-verify
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	/* ------------------------------------------------------------------
	   Panel shell  (wp = wiki-panel)
	------------------------------------------------------------------ */
	.wp-panel {
		display: flex;
		flex-direction: column;
		gap: 10px;
		width: 220px;
		flex-shrink: 0;
		padding: 12px;
		border-left: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.08));
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		overflow-y: auto;
	}

	.wp-panel__header {
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.wp-panel__header-icon {
		flex-shrink: 0;
	}

	.wp-panel__header-title {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	/* Status row */
	.wp-panel__status {
		display: flex;
		align-items: center;
		gap: 5px;
	}

	.wp-panel__status-text {
		font-size: 12px;
		color: var(--bos-v2-text-primary, #121212);
	}

	.wp-panel__status-text--muted {
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.wp-panel__status-ok {
		color: #22c55e;
		flex-shrink: 0;
	}

	/* Slug row */
	.wp-panel__slug {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}

	.wp-panel__slug-label {
		font-size: 10px;
		font-weight: 500;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.wp-panel__slug-value {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 11px;
		color: var(--bos-v2-text-primary, #121212);
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
		border-radius: 4px;
		padding: 2px 6px;
		word-break: break-all;
	}

	/* Action buttons */
	.wp-panel__actions {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	/* ------------------------------------------------------------------
	   Shared button styles
	------------------------------------------------------------------ */
	.wp-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 5px;
		height: 30px;
		padding: 0 12px;
		font-size: 12px;
		font-weight: 500;
		border-radius: 6px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		background: var(--bos-v2-layer-background-primary, #ffffff);
		color: var(--bos-v2-text-primary, #121212);
		cursor: pointer;
		transition: background 0.12s, opacity 0.12s;
	}

	.wp-btn:hover:not(:disabled) {
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
	}

	.wp-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.wp-btn--primary {
		background: var(--bos-v2-button-primary, #1e96eb);
		color: var(--bos-v2-button-pureWhiteText, #ffffff);
		border-color: transparent;
	}

	.wp-btn--primary:hover:not(:disabled) {
		opacity: 0.88;
		background: var(--bos-v2-button-primary, #1e96eb);
	}

	.wp-ghost-btn {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 6px;
		font-size: 11px;
		font-weight: 500;
		border-radius: 4px;
		border: none;
		background: transparent;
		color: var(--bos-v2-text-secondary, #8e8d91);
		cursor: pointer;
		transition: color 0.1s, background 0.1s;
	}

	.wp-ghost-btn:hover {
		color: var(--bos-v2-text-primary, #121212);
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
	}

	.wp-icon-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		border: none;
		background: transparent;
		border-radius: 5px;
		color: var(--bos-v2-text-secondary, #8e8d91);
		cursor: pointer;
		transition: color 0.1s, background 0.1s;
		flex-shrink: 0;
	}

	.wp-icon-btn:hover {
		color: var(--bos-v2-text-primary, #121212);
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
	}

	/* ------------------------------------------------------------------
	   Spinner (spin animation on Loader2 icon)
	------------------------------------------------------------------ */
	:global(.wp-spinner) {
		animation: wp-spin 0.7s linear infinite;
	}

	@keyframes wp-spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* ------------------------------------------------------------------
	   Modal overlay + shell
	------------------------------------------------------------------ */
	.wp-overlay {
		position: fixed;
		inset: 0;
		z-index: 200;
		background: rgba(0, 0, 0, 0.45);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 24px;
	}

	.wp-modal {
		width: 100%;
		max-width: 420px;
		max-height: 80vh;
		overflow-y: auto;
		background: var(--bos-v2-layer-background-primary, #ffffff);
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		border-radius: 12px;
		box-shadow: 0 12px 48px rgba(0, 0, 0, 0.18);
		display: flex;
		flex-direction: column;
	}

	.wp-modal--lg {
		max-width: 580px;
	}

	.wp-modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px 12px;
		border-bottom: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.08));
		flex-shrink: 0;
	}

	.wp-modal__title {
		font-size: 14px;
		font-weight: 600;
		color: var(--bos-v2-text-primary, #121212);
	}

	.wp-modal__body {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		overflow-y: auto;
	}

	.wp-modal__footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 12px 16px;
		border-top: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.08));
		flex-shrink: 0;
	}

	/* ------------------------------------------------------------------
	   Form elements
	------------------------------------------------------------------ */
	.wp-field {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}

	.wp-field__row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.wp-label {
		font-size: 11px;
		font-weight: 500;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.wp-input {
		height: 32px;
		padding: 0 10px;
		font-size: 13px;
		border-radius: 6px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.12));
		background: var(--bos-v2-layer-background-primary, #ffffff);
		color: var(--bos-v2-text-primary, #121212);
		outline: none;
		transition: border-color 0.12s;
		width: 100%;
	}

	.wp-input:focus {
		border-color: var(--bos-brand-color, #1e96eb);
	}

	.wp-input--readonly {
		background: var(--bos-v2-layer-background-tertiary, #eeeef0);
		color: var(--bos-v2-text-secondary, #8e8d91);
		cursor: default;
	}

	.wp-input--sm {
		height: 28px;
		font-size: 12px;
		flex: 1;
	}

	.wp-select {
		height: 32px;
		padding: 0 10px;
		font-size: 13px;
		border-radius: 6px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.12));
		background: var(--bos-v2-layer-background-primary, #ffffff);
		color: var(--bos-v2-text-primary, #121212);
		outline: none;
		cursor: pointer;
		width: 100%;
	}

	/* Citations */
	.wp-citation-row {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-top: 4px;
	}

	/* Error message */
	.wp-error {
		font-size: 12px;
		color: #ef4444;
		background: rgba(239, 68, 68, 0.08);
		border-radius: 6px;
		padding: 8px 10px;
	}

	/* ------------------------------------------------------------------
	   Render result
	------------------------------------------------------------------ */
	.wp-result {
		display: flex;
		flex-direction: column;
		gap: 8px;
		border: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		border-radius: 8px;
		overflow: hidden;
	}

	.wp-result__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		border-bottom: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.08));
	}

	.wp-result__label {
		font-size: 11px;
		font-weight: 500;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.wp-result__label--muted {
		color: var(--bos-v2-text-tertiary, #bfbfc3);
	}

	.wp-result__body {
		margin: 0;
		padding: 12px;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 12px;
		color: var(--bos-v2-text-primary, #121212);
		white-space: pre-wrap;
		word-break: break-word;
		max-height: 320px;
		overflow-y: auto;
	}

	.wp-result__citations {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 8px 12px;
		border-top: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.08));
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
	}

	.wp-result__citation {
		display: flex;
		align-items: baseline;
		gap: 6px;
	}

	.wp-result__citation-idx {
		font-size: 10px;
		color: var(--bos-v2-text-tertiary, #bfbfc3);
		flex-shrink: 0;
	}

	.wp-result__citation-chunk,
	.wp-result__citation-hash {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 11px;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	/* ------------------------------------------------------------------
	   Verify result
	------------------------------------------------------------------ */
	.wp-verify {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.wp-verify__status {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 12px;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 500;
	}

	.wp-verify__status--ok {
		background: rgba(34, 197, 94, 0.1);
		color: #16a34a;
	}

	.wp-verify__status--fail {
		background: rgba(239, 68, 68, 0.08);
		color: #dc2626;
	}

	.wp-issue-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.wp-issue {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 8px 10px;
		border-radius: 6px;
		border: 1px solid transparent;
	}

	.wp-issue--error {
		background: rgba(239, 68, 68, 0.07);
		border-color: rgba(239, 68, 68, 0.15);
	}

	.wp-issue--warning {
		background: rgba(245, 158, 11, 0.07);
		border-color: rgba(245, 158, 11, 0.15);
	}

	.wp-issue--info {
		background: rgba(59, 130, 246, 0.07);
		border-color: rgba(59, 130, 246, 0.15);
	}

	:global(.wp-issue__icon) {
		flex-shrink: 0;
		margin-top: 1px;
	}

	.wp-issue--error :global(.wp-issue__icon) {
		color: #ef4444;
	}

	.wp-issue--warning :global(.wp-issue__icon) {
		color: #f59e0b;
	}

	.wp-issue--info :global(.wp-issue__icon) {
		color: #3b82f6;
	}

	.wp-issue__body {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.wp-issue__code {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 10px;
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	.wp-issue__msg {
		font-size: 12px;
		color: var(--bos-v2-text-primary, #121212);
		line-height: 1.4;
	}

	/* ------------------------------------------------------------------
	   Dark mode
	------------------------------------------------------------------ */
	:global(.dark) .wp-panel {
		background: var(--bos-v2-layer-background-secondary, #2c2c2c);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.07));
	}

	:global(.dark) .wp-panel__slug-value {
		background: var(--bos-v2-layer-background-tertiary, #3a3a3a);
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .wp-panel__status-text {
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .wp-btn {
		background: var(--bos-v2-layer-background-secondary, #2c2c2c);
		color: var(--bos-v2-text-primary, #e6e6e6);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.1));
	}

	:global(.dark) .wp-btn:hover:not(:disabled) {
		background: var(--bos-v2-layer-background-tertiary, #3a3a3a);
	}

	:global(.dark) .wp-modal {
		background: var(--bos-v2-layer-background-primary, #1e1e1e);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.08));
	}

	:global(.dark) .wp-modal__header {
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.06));
	}

	:global(.dark) .wp-modal__title {
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .wp-modal__footer {
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.06));
	}

	:global(.dark) .wp-input {
		background: var(--bos-v2-layer-background-secondary, #2c2c2c);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.1));
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .wp-input--readonly {
		background: var(--bos-v2-layer-background-tertiary, #3a3a3a);
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	:global(.dark) .wp-select {
		background: var(--bos-v2-layer-background-secondary, #2c2c2c);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.1));
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .wp-result {
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.08));
	}

	:global(.dark) .wp-result__header,
	:global(.dark) .wp-result__citations {
		background: var(--bos-v2-layer-background-secondary, #2c2c2c);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.06));
	}

	:global(.dark) .wp-result__body {
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .wp-issue__msg {
		color: var(--bos-v2-text-primary, #e6e6e6);
	}
</style>
