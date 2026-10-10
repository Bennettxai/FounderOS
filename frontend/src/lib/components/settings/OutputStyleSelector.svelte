<script lang="ts">
	import { fade } from 'svelte/transition';

	interface OutputStyle {
		id: string;
		name: string;
		display_name: string;
		description?: string;
		is_active: boolean;
		sort_order: number;
	}

	interface Props {
		styles: OutputStyle[];
		selectedStyleId?: string;
		onSelect: (styleId: string) => void;
		disabled?: boolean;
	}

	let { styles, selectedStyleId = '', onSelect, disabled = false }: Props = $props();

	let expandedStyleId = $state<string | null>(null);

	function toggleExpand(styleId: string) {
		expandedStyleId = expandedStyleId === styleId ? null : styleId;
	}

	function handleSelect(styleId: string) {
		onSelect(styleId);
	}

	// Get example output for each style
	function getStyleExample(styleName: string): string {
		const examples: Record<string, string> = {
			'concise': '**Key Point:** Direct answer with minimal elaboration.',
			'detailed': '**Comprehensive Analysis:**\n\n1. Context and background\n2. Detailed explanation\n3. Multiple perspectives\n4. Conclusion with next steps',
			'technical': '```\nCode-focused response with:\n- Technical specifications\n- Implementation details\n- Performance considerations\n```',
			'conversational': 'Friendly, easy-to-understand explanation with examples and casual tone.',
			'structured': '## Main Topic\n\n### Subtopic 1\n- Point A\n- Point B\n\n### Subtopic 2\n- Point C\n- Point D'
		};

		return examples[styleName] || 'Example output in this style...';
	}
</script>

<div class="output-styles-selector">
	<!-- Auto / Default Option -->
	<button
		class="btn-pill btn-pill-ghost style-card"
		class:selected={selectedStyleId === ''}
		class:disabled
		onclick={() => !disabled && handleSelect('')}
	>
		<div class="style-header">
			<div class="style-info">
				<div class="style-icon">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
					</svg>
				</div>
				<div>
					<h4 class="style-name">Auto</h4>
					<p class="style-description">Let the AI choose the best style for your message</p>
				</div>
			</div>
			{#if selectedStyleId === ''}
				<div class="selected-badge">
					<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
						<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
					</svg>
				</div>
			{/if}
		</div>
	</button>

	<!-- Style Options -->
	{#each styles.filter(s => s.is_active).sort((a, b) => a.sort_order - b.sort_order) as style (style.id)}
		<div class="style-card-wrapper">
			<button
				class="btn-pill btn-pill-ghost style-card"
				class:selected={selectedStyleId === style.id}
				class:expanded={expandedStyleId === style.id}
				class:disabled
				onclick={() => !disabled && handleSelect(style.id)}
			>
				<div class="style-header">
					<div class="style-info">
						<div class="style-icon">
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								{#if style.name === 'concise'}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
								{:else if style.name === 'detailed'}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
								{:else if style.name === 'technical'}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
								{:else if style.name === 'conversational'}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
								{:else}
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
								{/if}
							</svg>
						</div>
						<div>
							<h4 class="style-name">{style.display_name}</h4>
							{#if style.description}
								<p class="style-description">{style.description}</p>
							{/if}
						</div>
					</div>
					<div class="style-actions">
						{#if selectedStyleId === style.id}
							<div class="selected-badge">
								<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
									<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
								</svg>
							</div>
						{/if}
						<span
							class="expand-btn"
							role="button"
							tabindex="0"
							onclick={(e) => {
								e.stopPropagation();
								toggleExpand(style.id);
							}}
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') {
									e.stopPropagation();
									e.preventDefault();
									toggleExpand(style.id);
								}
							}}
							title="Preview style"
						>
							<svg
								class="w-4 h-4 transition-transform"
								class:rotate-180={expandedStyleId === style.id}
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
							</svg>
						</span>
					</div>
				</div>
			</button>

			{#if expandedStyleId === style.id}
				<div class="style-preview" transition:fade={{ duration: 150 }}>
					<h5 class="preview-label">Example Output:</h5>
					<div class="preview-content">
						<pre>{getStyleExample(style.name)}</pre>
					</div>
				</div>
			{/if}
		</div>
	{/each}
</div>

<style>
	.output-styles-selector {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.style-card-wrapper {
		display: flex;
		flex-direction: column;
	}

	.style-card {
		width: 100%;
		padding: 1rem;
		background: var(--dbg, var(--bos-card, #fff));
		border: 2px solid var(--dbd, var(--bos-border, #e0e0e0));
		border-radius: 0.75rem;
		cursor: pointer;
		transition: all 0.2s;
		text-align: left;
	}

	.style-card:hover:not(.disabled) {
		border-color: var(--dt4, #bbb);
		box-shadow: var(--bos-shadow-1, 0 2px 8px rgba(0,0,0,.05));
	}

	.style-card.selected {
		border-color: var(--dt, var(--bos-text-primary, #111));
		background: var(--bos-settings-card-bg);
	}

	.style-card.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.style-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}

	.style-info {
		display: flex;
		gap: 0.75rem;
		flex: 1;
	}

	.style-icon {
		flex-shrink: 0;
		width: 2.5rem;
		height: 2.5rem;
		display: flex;
		align-items: center;
		justify-content: center;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		border-radius: 0.5rem;
		color: white;
	}

	.style-name {
		font-size: 0.9375rem;
		font-weight: 600;
		color: var(--dt, var(--bos-text-primary, #111));
		margin: 0;
	}

	.style-description {
		font-size: 0.8125rem;
		color: var(--dt3, var(--bos-text-tertiary, #888));
		margin: 0.25rem 0 0;
	}

	.style-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.selected-badge {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.5rem;
		height: 1.5rem;
		background: #10b981;
		border-radius: 9999px;
		color: white;
	}

	.expand-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.25rem;
		background: transparent;
		border: none;
		color: var(--dt3, var(--bos-text-tertiary, #888));
		cursor: pointer;
		border-radius: 0.25rem;
		transition: all 0.2s;
	}

	.expand-btn:hover {
		background: var(--dbg3, #eee);
		color: var(--dt, var(--bos-text-primary, #111));
	}

	.style-preview {
		padding: 1rem;
		background: var(--bos-settings-card-bg);
		border: 1px solid var(--bos-settings-card-border);
		border-top: none;
		border-radius: 0 0 0.75rem 0.75rem;
		margin-top: -0.5rem;
	}

	.preview-label {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--dt3, var(--bos-text-tertiary, #888));
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin: 0 0 0.5rem;
	}

	.preview-content {
		background: var(--dbg, var(--bos-card, #fff));
		border: 1px solid var(--dbd, var(--bos-border, #e0e0e0));
		border-radius: 0.5rem;
		padding: 0.75rem;
	}

	.preview-content pre {
		margin: 0;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		font-size: 0.8125rem;
		line-height: 1.6;
		color: var(--dt2, var(--bos-text-secondary, #555));
		white-space: pre-wrap;
		word-wrap: break-word;
	}

	@media (max-width: 640px) {
		.style-card {
			padding: 0.75rem;
		}

		.style-name {
			font-size: 0.875rem;
		}

		.style-description {
			font-size: 0.75rem;
		}
	}
</style>
