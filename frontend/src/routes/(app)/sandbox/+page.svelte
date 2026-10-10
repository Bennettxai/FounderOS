<script lang="ts">
	import { browser } from '$app/environment';
	import {
		AlertTriangle,
		Check,
		ExternalLink,
		Globe2,
		RefreshCw,
		Shield
	} from 'lucide-svelte';
	import { onMount } from 'svelte';

	const STORAGE_KEY = 'businessos_sandbox_url';
	const FALLBACK_URL = 'https://example.com';

	let urlInput = $state(FALLBACK_URL);
	let sandboxUrl = $state(FALLBACK_URL);
	let frameKey = $state(0);
	let message = $state('');
	let urlError = $state('');

	const canPreview = $derived(!urlError && sandboxUrl.length > 0);
	const displayHost = $derived.by(() => {
		try {
			return new URL(sandboxUrl).host;
		} catch {
			return 'No URL selected';
		}
	});

	onMount(() => {
		if (!browser) return;

		const savedUrl = localStorage.getItem(STORAGE_KEY);
		if (savedUrl) {
			urlInput = savedUrl;
			sandboxUrl = savedUrl;
		}
	});

	function normalizeUrl(value: string): string {
		const trimmed = value.trim();
		if (!trimmed) return '';
		if (/^https?:\/\//i.test(trimmed)) return trimmed;
		return `https://${trimmed}`;
	}

	function validateUrl(value: string): string | null {
		try {
			const parsed = new URL(value);
			if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
				return 'Use an http or https URL.';
			}
			return null;
		} catch {
			return 'Enter a valid URL.';
		}
	}

	function loadSandbox() {
		const normalized = normalizeUrl(urlInput);
		if (!normalized) {
			urlError = 'Enter a sandbox URL to preview.';
			message = '';
			return;
		}

		const validationError = validateUrl(normalized);
		if (validationError) {
			urlError = validationError;
			message = '';
			return;
		}

		urlError = '';
		sandboxUrl = normalized;
		urlInput = normalized;
		frameKey += 1;
		message = 'Sandbox URL updated.';

		if (browser) {
			localStorage.setItem(STORAGE_KEY, normalized);
		}
	}

	function refreshFrame() {
		frameKey += 1;
		message = 'Sandbox refreshed.';
	}
</script>

<svelte:head>
	<title>Sandbox - BusinessOS</title>
</svelte:head>

<div class="sandbox-module">
	<header class="sandbox-toolbar">
		<div class="sandbox-title">
			<div class="sandbox-mark" aria-hidden="true">
				<Shield size={20} />
			</div>
			<div>
				<p>BusinessOS mini app</p>
				<h1>Sandbox</h1>
			</div>
		</div>

		<form class="url-form" onsubmit={(event) => { event.preventDefault(); loadSandbox(); }}>
			<label for="sandbox-url">Sandbox URL</label>
			<div class="url-control">
				<Globe2 size={16} />
				<input
					id="sandbox-url"
					bind:value={urlInput}
					type="url"
					inputmode="url"
					placeholder="https://your-sandbox-url.com"
					aria-invalid={urlError ? 'true' : 'false'}
				/>
				<button type="submit">
					<Check size={15} />
					Load
				</button>
			</div>
			{#if urlError}
				<p class="form-message error"><AlertTriangle size={14} /> {urlError}</p>
			{:else if message}
				<p class="form-message success"><Check size={14} /> {message}</p>
			{/if}
		</form>

		<div class="toolbar-actions">
			<a class="icon-action" href={sandboxUrl} target="_blank" rel="noreferrer" title="Open in new tab">
				<ExternalLink size={16} />
			</a>
			<button class="icon-action" type="button" onclick={refreshFrame} title="Refresh sandbox">
				<RefreshCw size={16} />
			</button>
		</div>
	</header>

	<section class="sandbox-frame-shell" aria-label="Sandbox preview">
		<div class="frame-status">
			<span class="status-dot"></span>
			<span>{displayHost}</span>
		</div>

		{#if canPreview}
			{#key frameKey}
				<iframe
					src={sandboxUrl}
					title="Sandbox URL preview"
					sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-downloads"
					referrerpolicy="no-referrer"
				></iframe>
			{/key}
		{:else}
			<div class="empty-state">
				<AlertTriangle size={26} />
				<p>Load a valid URL to start the sandbox preview.</p>
			</div>
		{/if}
	</section>
</div>

<style>
	.sandbox-module {
		height: 100%;
		min-height: 0;
		display: grid;
		grid-template-rows: auto 1fr;
		background: #f6f7f9;
		color: #14171f;
	}

	.sandbox-toolbar {
		display: grid;
		grid-template-columns: minmax(190px, auto) minmax(280px, 1fr) auto;
		align-items: center;
		gap: 16px;
		padding: 14px 16px;
		border-bottom: 1px solid #dde2ea;
		background: rgba(255, 255, 255, 0.92);
	}

	.sandbox-title {
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
	}

	.sandbox-mark {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		display: grid;
		place-items: center;
		background: #e9f7f3;
		color: #047857;
		border: 1px solid #c5eadf;
	}

	.sandbox-title p,
	.sandbox-title h1,
	.form-message,
	.frame-status,
	.empty-state p {
		margin: 0;
	}

	.sandbox-title p {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: #667085;
	}

	.sandbox-title h1 {
		font-size: 1.12rem;
		line-height: 1.15;
		font-weight: 720;
		letter-spacing: 0;
	}

	.url-form {
		display: grid;
		gap: 6px;
		min-width: 0;
	}

	.url-form label {
		font-size: 0.74rem;
		font-weight: 700;
		color: #475467;
	}

	.url-control {
		height: 40px;
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 8px;
		padding: 0 6px 0 12px;
		border: 1px solid #cfd7e3;
		border-radius: 8px;
		background: #fff;
		color: #667085;
	}

	.url-control:focus-within {
		border-color: #0f766e;
		box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
	}

	input {
		min-width: 0;
		border: 0;
		outline: 0;
		background: transparent;
		color: #14171f;
		font: inherit;
		font-size: 0.88rem;
	}

	.url-control button {
		height: 30px;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		border: 0;
		border-radius: 7px;
		padding: 0 12px;
		background: #0f766e;
		color: #fff;
		font-size: 0.8rem;
		font-weight: 700;
		cursor: pointer;
	}

	.url-control button:hover {
		background: #0b625c;
	}

	.form-message {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 0.76rem;
	}

	.form-message.error {
		color: #b42318;
	}

	.form-message.success {
		color: #067647;
	}

	.toolbar-actions {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.icon-action {
		width: 38px;
		height: 38px;
		display: grid;
		place-items: center;
		border: 1px solid #cfd7e3;
		border-radius: 8px;
		background: #fff;
		color: #344054;
		text-decoration: none;
		cursor: pointer;
	}

	.icon-action:hover {
		background: #f2f4f7;
		border-color: #b8c2d0;
	}

	.sandbox-frame-shell {
		min-height: 0;
		display: grid;
		grid-template-rows: auto 1fr;
		padding: 12px;
	}

	.frame-status {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 30px;
		padding: 0 10px;
		border: 1px solid #dde2ea;
		border-bottom: 0;
		border-radius: 8px 8px 0 0;
		background: #fff;
		color: #475467;
		font-size: 0.78rem;
		font-weight: 650;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 999px;
		background: #17b26a;
		box-shadow: 0 0 0 3px rgba(23, 178, 106, 0.12);
	}

	iframe,
	.empty-state {
		width: 100%;
		height: 100%;
		min-height: 0;
		border: 1px solid #dde2ea;
		border-radius: 0 0 8px 8px;
		background: #fff;
	}

	iframe {
		border-top: 0;
	}

	.empty-state {
		display: grid;
		place-items: center;
		align-content: center;
		gap: 10px;
		color: #667085;
	}

	:global(.dark) .sandbox-module {
		background: #0f141b;
		color: #f5f7fa;
	}

	:global(.dark) .sandbox-toolbar,
	:global(.dark) .frame-status,
	:global(.dark) .url-control,
	:global(.dark) .icon-action,
	:global(.dark) iframe,
	:global(.dark) .empty-state {
		background: #171d26;
		border-color: #293241;
	}

	:global(.dark) .sandbox-title p,
	:global(.dark) .url-form label,
	:global(.dark) .frame-status,
	:global(.dark) .empty-state {
		color: #aab4c3;
	}

	:global(.dark) input,
	:global(.dark) .icon-action {
		color: #f5f7fa;
	}

	@media (max-width: 760px) {
		.sandbox-toolbar {
			grid-template-columns: 1fr auto;
		}

		.url-form {
			grid-column: 1 / -1;
			order: 3;
		}
	}

	@media (max-width: 480px) {
		.sandbox-toolbar {
			grid-template-columns: 1fr;
			gap: 12px;
		}
		.sandbox-title {
			grid-column: 1 / -1;
		}
		.toolbar-actions {
			grid-column: 1 / -1;
			justify-content: flex-end;
		}
		.url-form {
			grid-column: 1 / -1;
			order: 2;
		}
		.sandbox-frame-shell {
			padding: 8px;
		}
	}
</style>
