<script lang="ts">
	import { fly, slide } from 'svelte/transition';
	import type { Snippet } from 'svelte';
	import DOMPurify from 'dompurify';
	import CodeBlock from './CodeBlock.svelte';

	interface Props {
		title?: string;
		type?: 'code' | 'document' | 'image' | 'html';
		language?: string;
		content?: string;
		children?: Snippet;
		collapsible?: boolean;
		class?: string;
	}

	let {
		title = 'Artifact',
		type = 'code',
		language = 'plaintext',
		content = '',
		children,
		collapsible = true,
		class: className = ''
	}: Props = $props();

	let expanded = $state(true);
	let copied = $state(false);

	function copyContent() {
		navigator.clipboard.writeText(content);
		copied = true;
		setTimeout(() => copied = false, 2000);
	}

	function toggleExpand() {
		if (collapsible) {
			expanded = !expanded;
		}
	}

	function handleHeaderKeydown(e: KeyboardEvent) {
		if (collapsible && (e.key === 'Enter' || e.key === ' ')) {
			e.preventDefault();
			toggleExpand();
		}
	}

	const typeIcons: Record<string, string> = {
		code: 'M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4',
		document: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
		image: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z',
		html: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9'
	};
</script>

<div class="ai-artifact {className}" in:fly={{ y: 10, duration: 200 }}>
	<!-- Header -->
	<div
		role={collapsible ? 'button' : undefined}
		tabindex={collapsible ? 0 : undefined}
		onclick={toggleExpand}
		onkeydown={handleHeaderKeydown}
		class="ai-artifact__header"
		class:ai-artifact__header--collapsible={collapsible}
	>
		<div class="ai-artifact__title-row">
			<div class="ai-artifact__icon-wrapper">
				<svg class="ai-artifact__type-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={typeIcons[type] || typeIcons.code} />
				</svg>
			</div>
			<div class="ai-artifact__title-info">
				<span class="ai-artifact__title">{title}</span>
				{#if type === 'code' && language}
					<span class="ai-artifact__language">{language}</span>
				{/if}
			</div>
		</div>

		<div class="ai-artifact__actions">
			{#if type === 'code' || type === 'document'}
				<button
					type="button"
					onclick={(e) => { e.stopPropagation(); copyContent(); }}
					class="ai-artifact__action-btn"
					aria-label="Copy content"
				>
					{#if copied}
						<svg class="ai-artifact__action-icon ai-artifact__action-icon--success" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
					{:else}
						<svg class="ai-artifact__action-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
					{/if}
				</button>
			{/if}

			{#if collapsible}
				<svg
					class="ai-artifact__expand-icon"
					class:ai-artifact__expand-icon--collapsed={!expanded}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			{/if}
		</div>
	</div>

	<!-- Content -->
	{#if expanded}
		<div class="ai-artifact__content" transition:slide={{ duration: 200 }}>
			{#if children}
				{@render children()}
			{:else if type === 'code'}
				<CodeBlock code={content} {language} showLineNumbers={true} />
			{:else if type === 'html'}
				<div class="ai-artifact__html-preview">
					{@html DOMPurify.sanitize(content)}
				</div>
			{:else}
				<div class="ai-artifact__text">
					{content}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.ai-artifact {
		border: 1px solid var(--border);
		border-radius: 0.75rem;
		overflow: hidden;
		background-color: var(--card);
	}

	.ai-artifact__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 0.875rem 1rem;
		background-color: var(--muted);
		border: none;
		text-align: left;
		cursor: default;
	}

	.ai-artifact__header--collapsible {
		cursor: pointer;
	}

	.ai-artifact__header--collapsible:hover {
		background-color: var(--accent);
	}

	.ai-artifact__title-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.ai-artifact__icon-wrapper {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 0.5rem;
		background-color: var(--background);
	}

	.ai-artifact__type-icon {
		width: 1rem;
		height: 1rem;
		color: var(--muted-foreground);
	}

	.ai-artifact__title-info {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.ai-artifact__title {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--foreground);
	}

	.ai-artifact__language {
		font-size: 0.75rem;
		color: var(--muted-foreground);
		text-transform: lowercase;
	}

	.ai-artifact__actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.ai-artifact__action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border: none;
		border-radius: 0.375rem;
		background-color: transparent;
		cursor: pointer;
		transition: background-color 0.2s ease;
	}

	.ai-artifact__action-btn:hover {
		background-color: var(--accent);
	}

	.ai-artifact__action-icon {
		width: 1rem;
		height: 1rem;
		color: var(--muted-foreground);
	}

	.ai-artifact__action-icon--success {
		color: #4ade80;
	}

	.ai-artifact__expand-icon {
		width: 1.25rem;
		height: 1.25rem;
		color: var(--muted-foreground);
		transition: transform 0.2s ease;
	}

	.ai-artifact__expand-icon--collapsed {
		transform: rotate(-90deg);
	}

	.ai-artifact__content {
		border-top: 1px solid var(--border);
	}

	.ai-artifact__html-preview {
		padding: 1rem;
		background-color: white;
		min-height: 200px;
	}

	.ai-artifact__text {
		padding: 1rem;
		font-size: 0.875rem;
		line-height: 1.6;
		color: var(--foreground);
		white-space: pre-wrap;
	}

	/* Nested CodeBlock shouldn't have border radius */
	.ai-artifact__content :global(.ai-codeblock) {
		border-radius: 0;
		border: none;
	}
</style>
