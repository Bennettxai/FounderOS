<script lang="ts">
	import { Copy, Check, Pencil, Save, FileCode } from 'lucide-svelte';
	import { detectLanguage, getLanguageLabel, getLanguageColor } from './utils/language-detection';

	interface Props {
		filename: string;
		isEditing?: boolean;
		isDirty?: boolean;
		readonly?: boolean;
		cursorLine?: number;
		cursorColumn?: number;
		onToggleEdit?: () => void;
		onSave?: () => void;
		onCopy?: () => void;
	}

	let {
		filename,
		isEditing = false,
		isDirty = false,
		readonly = true,
		cursorLine = 1,
		cursorColumn = 1,
		onToggleEdit,
		onSave,
		onCopy,
	}: Props = $props();

	let copied = $state(false);
	let languageId = $derived(detectLanguage(filename));
	let languageLabel = $derived(getLanguageLabel(languageId));
	let languageColor = $derived(getLanguageColor(languageId));

	// Breadcrumb from filepath
	let breadcrumbs = $derived(filename.split('/').filter(Boolean));

	function handleCopy() {
		onCopy?.();
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<div class="editor-toolbar">
	<div class="toolbar-left">
		<FileCode class="toolbar-file-icon" style="color: {languageColor}" size={14} />
		<div class="toolbar-breadcrumbs">
			{#each breadcrumbs as segment, i}
				{#if i > 0}
					<span class="breadcrumb-sep">›</span>
				{/if}
				<span class="breadcrumb-segment" class:breadcrumb-active={i === breadcrumbs.length - 1}>
					{segment}
				</span>
			{/each}
			{#if isDirty}
				<span class="dirty-dot" title="Unsaved changes"></span>
			{/if}
		</div>
	</div>

	<div class="toolbar-right">
		<span class="toolbar-cursor">
			Ln {cursorLine}, Col {cursorColumn}
		</span>

		<span class="toolbar-lang-badge" style="border-color: {languageColor}40">
			{languageLabel}
		</span>

		<button class="toolbar-btn" onclick={handleCopy} title="Copy file contents">
			{#if copied}
				<Check size={14} />
			{:else}
				<Copy size={14} />
			{/if}
		</button>

		{#if isEditing}
			<button
				class="toolbar-btn toolbar-btn-save"
				class:toolbar-btn-save-dirty={isDirty}
				onclick={onSave}
				title="Save changes (Ctrl+S)"
			>
				<Save size={14} />
				<span>Save</span>
			</button>
		{:else}
			<button
				class="toolbar-btn toolbar-btn-edit"
				onclick={onToggleEdit}
				title="Edit file"
			>
				<Pencil size={14} />
				<span>Edit</span>
			</button>
		{/if}
	</div>
</div>

<style>
	.editor-toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 40px;
		padding: 0 12px;
		background: rgba(15, 15, 16, 0.6);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		flex-shrink: 0;
	}

	.toolbar-left {
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
		flex: 1;
	}

	.toolbar-file-icon {
		flex-shrink: 0;
	}

	.toolbar-breadcrumbs {
		display: flex;
		align-items: center;
		gap: 4px;
		min-width: 0;
		overflow: hidden;
	}

	.breadcrumb-sep {
		color: #71717a;
		font-size: 11px;
		flex-shrink: 0;
	}

	.breadcrumb-segment {
		color: #71717a;
		font-size: 12px;
		font-family: 'JetBrains Mono', monospace;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.breadcrumb-active {
		color: #f4f4f5;
	}

	.toolbar-right {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.toolbar-cursor {
		color: #71717a;
		font-size: 11px;
		font-family: 'JetBrains Mono', monospace;
	}

	.toolbar-lang-badge {
		padding: 2px 8px;
		border-radius: 9999px;
		border: 1px solid;
		font-size: 11px;
		color: #a1a1aa;
		font-family: 'JetBrains Mono', monospace;
	}

	.toolbar-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px;
		border-radius: 6px;
		border: none;
		background: transparent;
		color: #a1a1aa;
		font-size: 12px;
		cursor: pointer;
		transition: all 150ms ease;
	}

	.toolbar-btn:hover {
		background: rgba(255, 255, 255, 0.08);
		color: #f4f4f5;
	}

	.toolbar-btn-edit {
		border: 1px solid rgba(255, 255, 255, 0.12);
		color: #f4f4f5;
		font-weight: 500;
	}

	.toolbar-btn-edit:hover {
		background: rgba(99, 102, 241, 0.15);
		border-color: rgba(99, 102, 241, 0.4);
		color: #a5b4fc;
	}

	.toolbar-btn-save {
		background: rgba(99, 102, 241, 0.15);
		color: #818cf8;
	}

	.toolbar-btn-save:hover {
		background: rgba(99, 102, 241, 0.25);
		color: #a5b4fc;
	}

	.toolbar-btn-save-dirty {
		background: rgba(99, 102, 241, 0.25);
		color: #a5b4fc;
		border: 1px solid rgba(99, 102, 241, 0.4);
	}

	.dirty-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #6366f1;
		flex-shrink: 0;
		margin-left: 4px;
		animation: dirty-pulse 2s ease-in-out infinite;
	}

	@keyframes dirty-pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}
</style>
