<script lang="ts">
	import { getLanguageLabel } from './utils/language-detection';

	interface Props {
		languageId?: string;
		encoding?: string;
		lineEnding?: string;
		indentSize?: number;
		isReadonly?: boolean;
		isEditing?: boolean;
	}

	let {
		languageId = 'plaintext',
		encoding = 'UTF-8',
		lineEnding = 'LF',
		indentSize = 2,
		isReadonly = true,
		isEditing = false,
	}: Props = $props();

	let languageLabel = $derived(getLanguageLabel(languageId));
</script>

<div class="editor-status-bar">
	<div class="status-left">
		<span class="status-item">{languageLabel}</span>
		<span class="status-divider">|</span>
		<span class="status-item">{encoding}</span>
		<span class="status-divider">|</span>
		<span class="status-item">{lineEnding}</span>
		<span class="status-divider">|</span>
		<span class="status-item">Spaces: {indentSize}</span>
	</div>

	<div class="status-right">
		{#if isEditing}
			<span class="status-editing">Editing</span>
		{:else if isReadonly}
			<span class="status-readonly">Read Only</span>
		{/if}
	</div>
</div>

<style>
	.editor-status-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 28px;
		padding: 0 12px;
		background: rgba(15, 15, 16, 0.6);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		flex-shrink: 0;
	}

	.status-left,
	.status-right {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.status-item {
		color: #71717a;
		font-size: 11px;
		font-family: 'JetBrains Mono', monospace;
		cursor: default;
		transition: color 150ms ease;
	}

	.status-item:hover {
		color: #a1a1aa;
	}

	.status-divider {
		color: #3f3f46;
		font-size: 11px;
	}

	.status-readonly {
		color: #71717a;
		font-size: 11px;
		font-family: 'JetBrains Mono', monospace;
		padding: 1px 6px;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.04);
	}

	.status-editing {
		color: #818cf8;
		font-size: 11px;
		font-family: 'JetBrains Mono', monospace;
		padding: 1px 6px;
		border-radius: 4px;
		background: rgba(99, 102, 241, 0.15);
	}
</style>
