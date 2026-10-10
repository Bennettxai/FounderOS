<script lang="ts">
	interface Props {
		code: string;
		lang?: string;
		filename?: string;
	}

	let { code, lang = 'bash', filename = '' }: Props = $props();

	let copied = $state(false);

	async function copy() {
		await navigator.clipboard.writeText(code);
		copied = true;
		setTimeout(() => (copied = false), 1800);
	}
</script>

<div class="code-block">
	{#if filename}
		<div class="code-header">
			<span class="filename">{filename}</span>
			<button class="copy-btn" onclick={copy} aria-label="Copy code">
				{copied ? 'Copied' : 'Copy'}
			</button>
		</div>
	{:else}
		<button class="copy-btn float" onclick={copy} aria-label="Copy code">
			{copied ? 'Copied' : 'Copy'}
		</button>
	{/if}
	<pre class="lang-{lang}"><code>{code}</code></pre>
</div>

<style>
	.code-block {
		position: relative;
		margin-bottom: 1.5rem;
	}

	.code-block pre {
		margin-bottom: 0;
		border-radius: var(--r-md);
	}

	.code-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: var(--bg-elevated);
		border: 1px solid var(--border);
		border-bottom: none;
		border-radius: var(--r-md) var(--r-md) 0 0;
		padding: 0.5rem 1rem;
	}

	.code-header + pre {
		border-radius: 0 0 var(--r-md) var(--r-md);
	}

	.filename {
		font-family: var(--font-mono);
		font-size: 0.75rem;
		color: var(--text-muted);
	}

	.copy-btn {
		font-size: 0.75rem;
		font-family: var(--font-sans);
		color: var(--text-subtle);
		background: none;
		border: none;
		cursor: pointer;
		padding: 0.25rem 0.5rem;
		border-radius: var(--r-sm);
		transition: color 0.12s;
	}

	.copy-btn:hover {
		color: var(--text);
	}

	.copy-btn.float {
		position: absolute;
		top: 0.625rem;
		right: 0.75rem;
		z-index: 1;
	}
</style>
