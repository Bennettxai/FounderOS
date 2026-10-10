<!--
	OsaPill.svelte
	OSA Interface — fluid glassmorphism input pill with conversation overlay.
	Always-functional input: ModeSelector, textarea, voice, send.
	Protected module — no close/dismiss button.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { osaStore } from '$lib/stores/osa';
	import { initCSRF } from '$lib/api/base';
	import type { AttachedFile } from '$lib/stores/chat/types';
	import ModeSelector from './ModeSelector.svelte';
	import ModelSelector from './ModelSelector.svelte';
	import AgentRuntimeSelector from './AgentRuntimeSelector.svelte';
	import ChatInput from './ChatInput.svelte';
	import ResponseStream from './ResponseStream.svelte';

	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();

	onMount(async () => {
		await initCSRF().catch(() => {});
		osaStore.loadHealth();
		osaStore.loadModes();
	});

	let isExpanded = $derived($osaStore.isExpanded);
	let error = $derived($osaStore.error);
	let hasContent = $derived($osaStore.conversation.length > 0 || $osaStore.isStreaming || $osaStore.error !== null);
	let chatInputRef: ChatInput | undefined = $state(undefined);
	let pillElement: HTMLElement | undefined = $state(undefined);
	let fileInputElement: HTMLInputElement | undefined = $state(undefined);
	let charCount = $state(0);
	let lineCount = $state(1);
	let isDragging = $state(false);
	let previews = $state<Map<string, string>>(new Map());
	const MAX_ATTACHMENTS = 20;

	let activeProvider = $derived($osaStore.activeProvider);
	let activeModel = $derived($osaStore.activeModel);
	let activeRuntime = $derived($osaStore.activeRuntime);

	// Runtime auto-configures the model when switched
	// All runtimes use the same API chat path — the runtime selector just sets the model
	$effect(() => {
		const runtime = activeRuntime;
		if (runtime === 'claude') {
			osaStore.setModel('anthropic', 'claude-sonnet-4-20250514');
		} else if (runtime === 'codex') {
			osaStore.setModel('openai', 'gpt-4o');
		} else if (runtime === 'hermes') {
			osaStore.setModel('ollama', 'hermes3:latest');
		}
		// 'osa' and 'ollama' keep whatever model the user already selected
	});

	let chatPlaceholder = $derived.by(() => {
		const runtime = $osaStore.activeRuntime;
		if (runtime === 'claude') return 'Ask Claude Code...';
		if (runtime === 'codex') return 'Ask Codex...';
		if (runtime === 'ollama') return 'Ask Ollama...';
		if (runtime === 'hermes') return 'Ask Hermes...';
		// Default OSA mode — use model-based placeholder
		if (activeProvider === 'anthropic' || (activeModel && /claude/i.test(activeModel))) return 'Ask Claude...';
		if (activeProvider === 'openai' || (activeModel && /gpt|o3|o4|codex/i.test(activeModel))) return 'Ask GPT...';
		if (activeProvider === 'ollama' && activeModel) return `Ask ${activeModel.split(':')[0]}...`;
		return 'Ask OSA...';
	});

	let attachments = $derived($osaStore.attachments);
	let widthTier = $derived.by(() => {
		if (attachments.length > 0 || charCount > 80 || lineCount >= 3) return 3;
		if (charCount > 20 || lineCount >= 2) return 2;
		return 1;
	});

	export function focusInput() {
		osaStore.setExpanded(true);
		chatInputRef?.focus();
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			if (previewFile) {
				closePreview();
			} else if (isExpanded) {
				osaStore.setExpanded(false);
			}
		}
	}

	function handleInputFocus() {
		if (!isExpanded) {
			osaStore.setExpanded(true);
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (pillElement && !pillElement.contains(e.target as Node)) {
			previewFile = null;
			osaStore.setExpanded(false);
		}
	}

	function handleMetrics(metrics: { charCount: number; lineCount: number }) {
		charCount = metrics.charCount;
		lineCount = metrics.lineCount;
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer?.types.includes('Files')) {
			isDragging = true;
		}
	}

	function handleDragLeave(e: DragEvent) {
		const rect = pillElement?.getBoundingClientRect();
		if (rect && (e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom)) {
			isDragging = false;
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
		const files = e.dataTransfer?.files;
		if (files) addFiles(files);
	}

	function openFilePicker() {
		fileInputElement?.click();
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			addFiles(input.files);
			input.value = '';
		}
	}

	function addFiles(fileList: FileList) {
		const current = attachments.length;
		const remaining = MAX_ATTACHMENTS - current;
		if (remaining <= 0) return;
		const filesToAdd = Array.from(fileList).slice(0, remaining);

		for (const file of filesToAdd) {
			const id = crypto.randomUUID();
			const attached: AttachedFile = {
				id,
				name: file.name,
				type: file.type,
				size: file.size,
			};
			osaStore.addAttachment(attached);

			if (file.type.startsWith('image/')) {
				const url = URL.createObjectURL(file);
				previews.set(id, url);
				previews = new Map(previews);
			}
		}
		osaStore.setExpanded(true);
	}

	function handleRemoveAttachment(id: string) {
		const url = previews.get(id);
		if (url) {
			URL.revokeObjectURL(url);
			previews.delete(id);
			previews = new Map(previews);
		}
		osaStore.removeAttachment(id);
	}

	function getFileCategory(type: string): 'image' | 'pdf' | 'code' | 'file' {
		if (type.startsWith('image/')) return 'image';
		if (type === 'application/pdf') return 'pdf';
		if (type.startsWith('text/') || type.includes('javascript') || type.includes('json')) return 'code';
		return 'file';
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes}B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)}KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
	}

	// ===== PREVIEW =====
	let previewFile: AttachedFile | null = $state(null);

	function handleChipClick(file: AttachedFile) {
		previewFile = previewFile?.id === file.id ? null : file;
	}

	function closePreview() {
		previewFile = null;
	}

	$effect(() => {
		if (isExpanded) {
			document.addEventListener('pointerdown', handleClickOutside, true);
			return () => document.removeEventListener('pointerdown', handleClickOutside, true);
		}
	});
</script>

<section
	bind:this={pillElement}
	class="osa-pill {className}"
	class:tier-2={widthTier >= 2}
	class:tier-3={widthTier >= 3}
	class:dragging={isDragging}
	class:has-preview={previewFile !== null}
	role="region"
	aria-label="OSA Interface"
	aria-expanded={isExpanded}
	onkeydown={handleKeyDown}
	ondragover={handleDragOver}
	ondrop={handleDrop}
	ondragleave={handleDragLeave}
>
	{#if isExpanded && hasContent}
		<!-- Conversation card — topmost, grows upward -->
		<div class="osa-conversation">
			{#if error}
				<button class="osa-error" role="alert" onclick={() => osaStore.clearError()}>
					{error}
					<span class="osa-error-dismiss">Dismiss</span>
				</button>
			{/if}
			<ResponseStream maxHeight="320px" />
		</div>
	{/if}

	<!-- Input pill — Foundation ChatInput pattern -->
	<div class="osa-input-pill" class:expanded={isExpanded} class:multiline={widthTier >= 2} class:has-attachments={attachments.length > 0}>
		{#if isDragging}
			<div class="drop-overlay" role="presentation">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="drop-icon">
					<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
					<polyline points="7 10 12 15 17 10"/>
					<line x1="12" y1="15" x2="12" y2="3"/>
				</svg>
				<span class="drop-text">Drop files here</span>
			</div>
		{/if}

		{#if attachments.length > 0}
			<div class="osa-attachments">
				{#each attachments as file (file.id)}
					{@const category = getFileCategory(file.type)}
					<button type="button" class="attachment-chip" onclick={() => handleChipClick(file)}>
						<span class="chip-name">{file.name}</span>
						<span class="chip-size">{formatFileSize(file.size)}</span>
						<span class="chip-remove" role="button" tabindex="-1" onclick={(e) => { e.stopPropagation(); handleRemoveAttachment(file.id); }}>x</span>
					</button>
				{/each}
			</div>
		{/if}

		<!-- Textarea + unified action bar -->
		<ChatInput bind:this={chatInputRef} placeholder={chatPlaceholder} onfocus={handleInputFocus} onmetrics={handleMetrics} onattach={openFilePicker}>
			{#snippet actionBarCenter()}
				<ModeSelector compact />
				<ModelSelector compact />
				<AgentRuntimeSelector compact />
			{/snippet}
		</ChatInput>
	</div>

	<!-- File preview overlay — appears above pill when a chip is clicked -->
	{#if previewFile}
		{@const category = getFileCategory(previewFile.type)}
		<div class="file-preview-overlay">
			<div class="preview-header">
				<span class="preview-name">{previewFile.name}</span>
				<span class="preview-meta">{formatFileSize(previewFile.size)}</span>
				<button class="preview-close" onclick={closePreview} aria-label="Close preview">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
						<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
					</svg>
				</button>
			</div>
			<div class="preview-body">
				{#if category === 'image' && previews.get(previewFile.id)}
					<img src={previews.get(previewFile.id)} alt={previewFile.name} class="preview-image" />
				{:else if category === 'pdf'}
					<div class="preview-placeholder">
						<span class="chip-type-badge pdf" style="font-size: 14px; padding: 6px 10px;">PDF</span>
						<span>{previewFile.name}</span>
					</div>
				{:else}
					<div class="preview-placeholder">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="32" height="32" style="opacity: 0.4">
							<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
							<polyline points="14 2 14 8 20 8"/>
						</svg>
						<span>{previewFile.name}</span>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<input bind:this={fileInputElement} type="file" multiple hidden onchange={handleFileSelect} />
</section>

<style>
	.osa-pill {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		width: 100%;
		min-width: min(340px, calc(100vw - 40px));
		max-width: min(460px, calc(100vw - 40px));
		margin: 0 auto;
		pointer-events: auto;
		gap: 8px;
		transition: max-width 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94), min-width 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94);
		overflow: visible;
	}

	.osa-pill[aria-expanded='true'] {
		max-width: min(560px, calc(100vw - 40px));
	}

	.osa-pill.tier-2 {
		max-width: min(520px, calc(100vw - 40px));
	}

	.osa-pill.tier-3 {
		max-width: min(640px, calc(100vw - 40px));
	}

	/* ===== CONVERSATION CARD (above input) ===== */
	.osa-conversation {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 12px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(28px) saturate(1.5);
		-webkit-backdrop-filter: blur(28px) saturate(1.5);
		border: 1px solid rgba(255, 255, 255, 0.6);
		border-radius: 20px;
		box-shadow:
			0 8px 40px rgba(0, 0, 0, 0.08),
			0 2px 8px rgba(0, 0, 0, 0.04),
			inset 0 1px 0 rgba(255, 255, 255, 0.7);
	}

	:global(.dark) .osa-conversation {
		background: rgba(44, 44, 46, 0.82);
		border-color: rgba(255, 255, 255, 0.14);
		box-shadow:
			0 8px 40px rgba(0, 0, 0, 0.25),
			0 2px 8px rgba(0, 0, 0, 0.15),
			inset 0 1px 0 rgba(255, 255, 255, 0.08);
	}

	.osa-error {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding: 6px 12px;
		border-radius: 10px;
		font-size: 12px;
		background: rgba(239, 68, 68, 0.08);
		color: #dc2626;
		border: 1px solid rgba(239, 68, 68, 0.15);
		cursor: pointer;
		width: 100%;
		text-align: left;
	}

	.osa-error:hover {
		background: rgba(239, 68, 68, 0.14);
	}

	.osa-error-dismiss {
		font-size: 10px;
		opacity: 0.6;
		flex-shrink: 0;
	}

	:global(.dark) .osa-error {
		background: rgba(239, 68, 68, 0.12);
		color: #fca5a5;
		border-color: rgba(239, 68, 68, 0.2);
	}

	:global(.dark) .osa-error:hover {
		background: rgba(239, 68, 68, 0.18);
	}

	/* ===== INPUT PILL (always visible) ===== */
	.osa-input-pill {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 2px;
		width: 100%;
		padding: 10px 12px 8px;
		background: rgba(255, 255, 255, 0.82);
		backdrop-filter: blur(28px) saturate(1.5);
		-webkit-backdrop-filter: blur(28px) saturate(1.5);
		border: 1px solid rgba(255, 255, 255, 0.6);
		border-radius: 18px;
		box-shadow:
			0 4px 20px rgba(0, 0, 0, 0.06),
			0 1px 4px rgba(0, 0, 0, 0.03),
			inset 0 1px 0 rgba(255, 255, 255, 0.7);
		transition: border-radius 0.25s ease, box-shadow 0.25s ease, border-color 0.2s ease, padding 0.2s ease;
		overflow: visible;
	}

	.osa-input-pill.has-attachments {
		gap: 6px;
	}


	.osa-input-pill:hover:not(.expanded) {
		box-shadow:
			0 6px 24px rgba(0, 0, 0, 0.08),
			0 2px 6px rgba(0, 0, 0, 0.04),
			inset 0 1px 0 rgba(255, 255, 255, 0.8);
	}

	/* Subtle reactive expansion when focused */
	.osa-input-pill.expanded:not(.has-attachments) {
		border-radius: 18px;
		border-color: rgba(0, 122, 255, 0.25);
		box-shadow:
			0 6px 28px rgba(0, 0, 0, 0.1),
			0 2px 8px rgba(0, 0, 0, 0.05),
			0 0 0 3px rgba(0, 122, 255, 0.08),
			inset 0 1px 0 rgba(255, 255, 255, 0.8);
	}

	.osa-input-pill.expanded.has-attachments {
		border-color: rgba(0, 122, 255, 0.25);
		box-shadow:
			0 6px 28px rgba(0, 0, 0, 0.1),
			0 2px 8px rgba(0, 0, 0, 0.05),
			0 0 0 3px rgba(0, 122, 255, 0.08),
			inset 0 1px 0 rgba(255, 255, 255, 0.8);
	}

	:global(.dark) .osa-input-pill.expanded:not(.has-attachments) {
		border-color: rgba(10, 132, 255, 0.3);
		box-shadow:
			0 6px 28px rgba(0, 0, 0, 0.25),
			0 2px 8px rgba(0, 0, 0, 0.15),
			0 0 0 3px rgba(10, 132, 255, 0.1),
			inset 0 1px 0 rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .osa-input-pill.expanded.has-attachments {
		border-color: rgba(10, 132, 255, 0.3);
		box-shadow:
			0 6px 28px rgba(0, 0, 0, 0.25),
			0 2px 8px rgba(0, 0, 0, 0.15),
			0 0 0 3px rgba(10, 132, 255, 0.1),
			inset 0 1px 0 rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .osa-input-pill {
		background: rgba(44, 44, 46, 0.78);
		border-color: rgba(255, 255, 255, 0.14);
		box-shadow:
			0 4px 20px rgba(0, 0, 0, 0.2),
			0 1px 4px rgba(0, 0, 0, 0.12),
			inset 0 1px 0 rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .osa-input-pill:hover:not(.expanded) {
		box-shadow:
			0 6px 24px rgba(0, 0, 0, 0.25),
			0 2px 6px rgba(0, 0, 0, 0.15),
			inset 0 1px 0 rgba(255, 255, 255, 0.1);
	}

	/* ===== MODE SELECTOR inside pill ===== */
	/* Strip trigger background — pill provides the chrome */
	.osa-input-pill :global(.mode-trigger) {
		background: transparent;
		border-color: transparent;
		padding: 5px 8px 5px 6px;
		border-radius: 999px;
	}

	.osa-input-pill :global(.mode-trigger:hover) {
		background: rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .osa-input-pill :global(.mode-trigger:hover) {
		background: rgba(255, 255, 255, 0.08);
	}

	/* Mode selector seamless inside pill */
	.osa-input-pill :global(.mode-selector) {
		padding-right: 0;
		margin-right: 0;
	}

	/* ===== MODEL SELECTOR inside pill ===== */
	.osa-input-pill :global(.model-trigger) {
		background: transparent;
		border-color: transparent;
		padding: 3px 6px 3px 4px;
		border-radius: 6px;
	}

	.osa-input-pill :global(.model-trigger:hover) {
		background: rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .osa-input-pill :global(.model-trigger:hover) {
		background: rgba(255, 255, 255, 0.08);
	}

	.osa-input-pill :global(.model-selector) {
		padding-right: 0;
		margin-right: 0;
	}

	/* ===== CHAT INPUT inside pill ===== */
	.osa-input-pill :global(.chat-input) {
		display: flex;
		flex-direction: column;
		gap: 6px;
		background: transparent;
		border: none;
		padding: 0;
		border-radius: 0;
	}

	.osa-input-pill :global(.chat-input:focus-within) {
		border-color: transparent;
		box-shadow: none;
	}

	/* Textarea — full width, grows vertically */
	.osa-input-pill :global(.chat-textarea) {
		width: 100%;
		min-height: 24px;
		max-height: 190px;
		overflow-y: auto;
		font-size: 14.5px;
		line-height: 1.4;
	}

	/* Action bar — attach left, voice + send right */
	.osa-input-pill :global(.chat-action-bar) {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 0;
	}

	/* ===== BUTTONS inside pill — bigger & more visible ===== */
	.osa-input-pill :global(.chat-btn) {
		width: 36px;
		height: 36px;
	}

	.osa-input-pill :global(.btn-icon) {
		width: 17px;
		height: 17px;
	}

	/* Voice button — visible with subtle bg */
	.osa-input-pill :global(.chat-btn.voice) {
		color: #636366;
		background: rgba(0, 0, 0, 0.05);
	}

	.osa-input-pill :global(.chat-btn.voice:hover) {
		background: rgba(0, 0, 0, 0.1);
		color: #1c1c1e;
	}

	:global(.dark) .osa-input-pill :global(.chat-btn.voice) {
		color: #98989d;
		background: rgba(255, 255, 255, 0.06);
	}

	:global(.dark) .osa-input-pill :global(.chat-btn.voice:hover) {
		background: rgba(255, 255, 255, 0.12);
		color: #f5f5f7;
	}

	/* Send button — prominent dark pill when active */
	.osa-input-pill :global(.chat-btn.send) {
		background: #d1d1d6;
		color: white;
	}

	.osa-input-pill :global(.chat-btn.send.active) {
		background: #1c1c1e;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
	}

	.osa-input-pill :global(.chat-btn.send.active:hover) {
		background: #000;
	}

	:global(.dark) .osa-input-pill :global(.chat-btn.send) {
		background: #48484a;
	}

	:global(.dark) .osa-input-pill :global(.chat-btn.send.active) {
		background: #f5f5f7;
		color: #1c1c1e;
		box-shadow: 0 2px 8px rgba(255, 255, 255, 0.15);
	}

	/* Stop button */
	.osa-input-pill :global(.chat-btn.stop) {
		width: 36px;
		height: 36px;
	}

	/* Hide toolbar hints inside pill */
	.osa-input-pill :global(.toolbar-hints) {
		display: none;
	}

	/* Recording bar fits pill shape */
	.osa-input-pill :global(.recording-bar-container) {
		border-radius: 999px;
	}

	.osa-input-wrapper {
		flex: 1;
		min-width: 0;
	}

	/* ===== WIDTH TIERS — expand as user types more ===== */
	.osa-pill.tier-2,
	.osa-pill.tier-2[aria-expanded='true'] {
		max-width: 560px;
	}

	.osa-pill.tier-3,
	.osa-pill.tier-3[aria-expanded='true'] {
		max-width: 720px;
	}

	/* ===== MULTILINE PILL SHAPE ===== */
	.osa-input-pill.multiline {
		border-radius: 20px;
	}

	.osa-input-pill.multiline.expanded {
		border-radius: 20px;
	}

	/* ===== DRAG-DROP ===== */
	.osa-pill.dragging .osa-input-pill {
		border-color: rgba(0, 122, 255, 0.4);
	}

	:global(.dark) .osa-pill.dragging .osa-input-pill {
		border-color: rgba(10, 132, 255, 0.45);
	}

	/* Drop overlay — inside the pill glass */
	.drop-overlay {
		position: absolute;
		inset: 0;
		z-index: 10;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 6px;
		background: rgba(0, 122, 255, 0.08);
		backdrop-filter: blur(6px);
		border-radius: inherit;
		pointer-events: none;
	}

	:global(.dark) .drop-overlay {
		background: rgba(10, 132, 255, 0.12);
	}

	.drop-icon {
		width: 28px;
		height: 28px;
		color: rgba(0, 122, 255, 0.6);
	}

	:global(.dark) .drop-icon {
		color: rgba(10, 132, 255, 0.7);
	}

	.drop-text {
		font-size: 12px;
		font-weight: 500;
		color: rgba(0, 122, 255, 0.7);
	}

	:global(.dark) .drop-text {
		color: rgba(10, 132, 255, 0.8);
	}

	.drop-count {
		font-size: 10px;
		color: rgba(0, 122, 255, 0.45);
	}

	:global(.dark) .drop-count {
		color: rgba(10, 132, 255, 0.5);
	}

	/* ===== ATTACHMENT CHIPS ===== */
	.osa-attachments {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		width: 100%;
		padding: 0 4px;
		max-height: 120px;
		overflow-y: auto;
		overflow-x: hidden;
	}

	.attachment-chip {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px 4px 6px;
		background: rgba(0, 0, 0, 0.04);
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 8px;
		font-size: 11px;
		color: #3a3a3c;
		max-width: 200px;
		cursor: pointer;
		font-family: inherit;
		text-align: left;
		animation: chipIn 0.15s ease-out;
		transition: background 0.15s ease;
	}

	.attachment-chip:hover {
		background: rgba(0, 0, 0, 0.08);
	}

	.attachment-chip.has-preview {
		padding-left: 2px;
	}

	:global(.dark) .attachment-chip {
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.08);
		color: #d1d1d6;
	}

	:global(.dark) .attachment-chip:hover {
		background: rgba(255, 255, 255, 0.12);
	}

	.chip-preview {
		width: 22px;
		height: 22px;
		border-radius: 4px;
		object-fit: cover;
		flex-shrink: 0;
	}

	.chip-type-badge {
		font-size: 8px;
		font-weight: 700;
		letter-spacing: 0.5px;
		padding: 2px 4px;
		border-radius: 3px;
		flex-shrink: 0;
		line-height: 1;
	}

	.chip-type-badge.pdf {
		background: rgba(239, 68, 68, 0.12);
		color: #dc2626;
	}

	:global(.dark) .chip-type-badge.pdf {
		background: rgba(239, 68, 68, 0.2);
		color: #fca5a5;
	}

	.chip-icon {
		width: 12px;
		height: 12px;
		flex-shrink: 0;
		opacity: 0.5;
	}

	.chip-size {
		font-size: 9px;
		color: #aeaeb2;
		flex-shrink: 0;
	}

	:global(.dark) .chip-size {
		color: #636366;
	}

	.attachment-limit {
		font-size: 10px;
		color: #aeaeb2;
		padding: 2px 6px;
	}

	:global(.dark) .attachment-limit {
		color: #636366;
	}

	.chip-name {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chip-remove {
		width: 14px;
		height: 14px;
		border: none;
		background: rgba(0, 0, 0, 0.08);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		padding: 0;
		color: #636366;
		transition: background 0.1s ease;
	}

	.chip-remove:hover {
		background: rgba(0, 0, 0, 0.15);
		color: #1c1c1e;
	}

	:global(.dark) .chip-remove {
		background: rgba(255, 255, 255, 0.1);
		color: #98989d;
	}

	:global(.dark) .chip-remove:hover {
		background: rgba(255, 255, 255, 0.2);
		color: #f5f5f7;
	}

	.chip-remove-icon {
		width: 8px;
		height: 8px;
	}

	/* ===== ATTACH BUTTON inside pill ===== */
	.osa-input-pill :global(.chat-btn.attach) {
		color: #636366;
		background: transparent;
	}

	.osa-input-pill :global(.chat-btn.attach:hover) {
		background: rgba(0, 0, 0, 0.06);
		color: #1c1c1e;
	}

	:global(.dark) .osa-input-pill :global(.chat-btn.attach) {
		color: #98989d;
	}

	:global(.dark) .osa-input-pill :global(.chat-btn.attach:hover) {
		background: rgba(255, 255, 255, 0.08);
		color: #f5f5f7;
	}

	/* ===== FILE PREVIEW OVERLAY ===== */
	.file-preview-overlay {
		width: 100%;
		display: flex;
		flex-direction: column;
		background: rgba(255, 255, 255, 0.9);
		backdrop-filter: blur(28px) saturate(1.5);
		-webkit-backdrop-filter: blur(28px) saturate(1.5);
		border: 1px solid rgba(255, 255, 255, 0.6);
		border-radius: 16px;
		box-shadow:
			0 8px 32px rgba(0, 0, 0, 0.1),
			0 2px 8px rgba(0, 0, 0, 0.04);
		overflow: hidden;
		animation: previewIn 0.2s ease-out;
	}

	:global(.dark) .file-preview-overlay {
		background: rgba(28, 28, 30, 0.92);
		border-color: rgba(255, 255, 255, 0.08);
		box-shadow:
			0 8px 32px rgba(0, 0, 0, 0.4),
			0 2px 8px rgba(0, 0, 0, 0.25);
	}

	.preview-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 10px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .preview-header {
		border-bottom-color: rgba(255, 255, 255, 0.06);
	}

	.preview-name {
		font-size: 12px;
		font-weight: 500;
		color: #1c1c1e;
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.dark) .preview-name {
		color: #f5f5f7;
	}

	.preview-meta {
		font-size: 10px;
		color: #aeaeb2;
		flex-shrink: 0;
	}

	:global(.dark) .preview-meta {
		color: #636366;
	}

	.preview-close {
		width: 22px;
		height: 22px;
		border: none;
		background: rgba(0, 0, 0, 0.06);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		padding: 0;
		color: #636366;
		transition: background 0.1s ease;
	}

	.preview-close:hover {
		background: rgba(0, 0, 0, 0.12);
		color: #1c1c1e;
	}

	:global(.dark) .preview-close {
		background: rgba(255, 255, 255, 0.08);
		color: #98989d;
	}

	:global(.dark) .preview-close:hover {
		background: rgba(255, 255, 255, 0.15);
		color: #f5f5f7;
	}

	.preview-body {
		padding: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 80px;
		max-height: 200px;
	}

	.preview-image {
		max-width: 100%;
		max-height: 200px;
		border-radius: 8px;
		object-fit: contain;
	}

	.preview-placeholder {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 16px;
		font-size: 12px;
		color: #636366;
	}

	:global(.dark) .preview-placeholder {
		color: #98989d;
	}

	@keyframes previewIn {
		from {
			opacity: 0;
			transform: translateY(4px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	@keyframes chipIn {
		from {
			opacity: 0;
			transform: scale(0.9);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}
</style>
