<script lang="ts">
	import { fly } from 'svelte/transition';

	interface Props {
		code: string;
		language?: string;
		filename?: string;
		showLineNumbers?: boolean;
		class?: string;
	}

	let {
		code,
		language = 'plaintext',
		filename,
		showLineNumbers = true,
		class: className = ''
	}: Props = $props();

	let copied = $state(false);

	function copyCode() {
		navigator.clipboard.writeText(code);
		copied = true;
		setTimeout(() => copied = false, 2000);
	}

	const lines = $derived(code.split('\n'));
</script>

<div class="ai-codeblock {className}" in:fly={{ y: 10, duration: 200 }}>
	<!-- Header -->
	<div class="ai-codeblock__header">
		<div class="ai-codeblock__info">
			{#if filename}
				<span class="ai-codeblock__filename">{filename}</span>
			{/if}
			<span class="ai-codeblock__language">{language}</span>
		</div>
		<button
			type="button"
			onclick={copyCode}
			class="btn-pill btn-pill-ghost ai-codeblock__copy"
			aria-label="Copy code"
		>
			{#if copied}
				<svg class="ai-codeblock__icon ai-codeblock__icon--success" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
				<span>Copied</span>
			{:else}
				<svg class="ai-codeblock__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
				</svg>
				<span>Copy</span>
			{/if}
		</button>
	</div>

	<!-- Code content -->
	<div class="ai-codeblock__content">
		<pre class="ai-codeblock__pre"><code class="ai-codeblock__code">{#each lines as line, i}{#if showLineNumbers}<span class="ai-codeblock__line-number">{i + 1}</span>{/if}<span class="ai-codeblock__line">{line}</span>
{/each}</code></pre>
	</div>
</div>

<style>
	.ai-codeblock {
		border-radius: 0.75rem;
		overflow: hidden;
		background-color: #1e1e1e;
		border: 1px solid var(--border);
	}

	.ai-codeblock__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.625rem 1rem;
		background-color: #2d2d2d;
		border-bottom: 1px solid #3d3d3d;
	}

	.ai-codeblock__info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.ai-codeblock__filename {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #e0e0e0;
	}

	.ai-codeblock__language {
		font-size: 0.75rem;
		color: #888;
		text-transform: lowercase;
	}

	.ai-codeblock__copy {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
		color: #888;
		background-color: transparent;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.ai-codeblock__copy:hover {
		color: #e0e0e0;
		background-color: #3d3d3d;
	}

	.ai-codeblock__icon {
		width: 0.875rem;
		height: 0.875rem;
	}

	.ai-codeblock__icon--success {
		color: #4ade80;
	}

	.ai-codeblock__content {
		overflow-x: auto;
	}

	.ai-codeblock__pre {
		margin: 0;
		padding: 1rem;
	}

	.ai-codeblock__code {
		display: block;
		font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
		font-size: 0.8125rem;
		line-height: 1.6;
		color: #e0e0e0;
		white-space: pre;
		tab-size: 2;
	}

	.ai-codeblock__line-number {
		display: inline-block;
		width: 2.5rem;
		padding-right: 1rem;
		text-align: right;
		color: #555;
		user-select: none;
	}

	.ai-codeblock__line {
		/* For potential syntax highlighting */
	}

	/* Light mode alternative (optional - uses dark by default) */
	:global(.light-code) .ai-codeblock {
		background-color: #f8f8f8;
		border-color: var(--border);
	}

	:global(.light-code) .ai-codeblock__header {
		background-color: #f0f0f0;
		border-bottom-color: #e0e0e0;
	}

	:global(.light-code) .ai-codeblock__filename {
		color: #333;
	}

	:global(.light-code) .ai-codeblock__code {
		color: #333;
	}

	:global(.light-code) .ai-codeblock__line-number {
		color: #999;
	}
</style>
