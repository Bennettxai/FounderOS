<script lang="ts">
	/**
	 * GalleryView - Image gallery view for app templates
	 */

	import type { Field } from '../types/field';
	import type { GalleryViewConfig } from '../types/view';
	import { TemplateSkeleton, TemplateModal } from '../primitives';

	interface Props {
		config: GalleryViewConfig;
		fields: Field[];
		data: Record<string, unknown>[];
		loading?: boolean;
		selectedIds?: Set<string>;
		onselect?: (id: string, selected: boolean) => void;
		onrowclick?: (record: Record<string, unknown>) => void;
	}

	let {
		config,
		fields,
		data,
		loading = false,
		selectedIds = new Set(),
		onselect,
		onrowclick
	}: Props = $props();

	let lightboxOpen = $state(false);
	let lightboxIndex = $state(0);

	const aspectRatios = {
		square: '1 / 1',
		'4:3': '4 / 3',
		'16:9': '16 / 9',
		auto: 'auto'
	};

	function openLightbox(index: number) {
		lightboxIndex = index;
		lightboxOpen = true;
	}

	function closeLightbox() {
		lightboxOpen = false;
	}

	function nextImage() {
		lightboxIndex = (lightboxIndex + 1) % data.length;
	}

	function prevImage() {
		lightboxIndex = (lightboxIndex - 1 + data.length) % data.length;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!lightboxOpen) return;
		if (e.key === 'ArrowRight') nextImage();
		if (e.key === 'ArrowLeft') prevImage();
		if (e.key === 'Escape') closeLightbox();
	}

	const gridStyle = $derived(`
		grid-template-columns: repeat(${config.columns || 4}, 1fr);
		gap: ${config.gap || 16}px;
	`);

	const aspectRatio = $derived(aspectRatios[config.aspectRatio || 'square']);
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="tpl-gallery-view">
	{#if loading}
		<div class="tpl-gallery-grid" style={gridStyle}>
			{#each Array(8) as _}
				<div class="tpl-gallery-skeleton">
					<TemplateSkeleton variant="rectangular" width="100%" height="200px" />
				</div>
			{/each}
		</div>
	{:else if data.length === 0}
		<div class="tpl-gallery-empty">
			<div class="tpl-gallery-empty-icon">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<rect x="3" y="3" width="18" height="18" rx="2" />
					<circle cx="8.5" cy="8.5" r="1.5" />
					<path d="M21 15l-5-5L5 21" />
				</svg>
			</div>
			<p>No images to display</p>
		</div>
	{:else}
		<div class="tpl-gallery-grid" style={gridStyle}>
			{#each data as record, index}
				{@const id = String(record.id || record._id || '')}
				{@const isSelected = selectedIds.has(id)}
				{@const image = record[config.imageField]}
				{@const title = record[config.titleField]}
				{@const subtitle = config.subtitleField ? record[config.subtitleField] : null}

				<div
					class="tpl-gallery-item"
					class:tpl-gallery-item-selected={isSelected}
					style:aspect-ratio={aspectRatio}
				>
					<button
						class="tpl-gallery-image-btn"
						onclick={() => openLightbox(index)}
					>
						{#if image}
							<img
								src={String(image)}
								alt={String(title || '')}
								class="tpl-gallery-image"
								loading="lazy"
							/>
						{:else}
							<div class="tpl-gallery-placeholder">
								<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
									<rect x="3" y="3" width="18" height="18" rx="2" />
									<circle cx="8.5" cy="8.5" r="1.5" />
									<path d="M21 15l-5-5L5 21" />
								</svg>
							</div>
						{/if}
					</button>
					<div class="tpl-gallery-overlay">
						<div class="tpl-gallery-info">
							{#if title}
								<span class="tpl-gallery-title">{title}</span>
							{/if}
							{#if subtitle}
								<span class="tpl-gallery-subtitle">{subtitle}</span>
							{/if}
						</div>
						{#if onselect}
							<input
								type="checkbox"
								class="tpl-gallery-checkbox"
								checked={isSelected}
								onchange={(e) => onselect?.(id, e.currentTarget.checked)}
								onclick={(e) => e.stopPropagation()}
							/>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Lightbox Modal -->
{#if lightboxOpen && data.length > 0}
	{@const currentRecord = data[lightboxIndex]}
	{@const currentImage = currentRecord[config.imageField]}
	{@const currentTitle = currentRecord[config.titleField]}

	<div class="tpl-lightbox" role="dialog" aria-modal="true">
		<div class="tpl-lightbox-backdrop" onclick={closeLightbox}></div>
		<div class="tpl-lightbox-content">
			<button class="tpl-lightbox-close" onclick={closeLightbox}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
			<button class="tpl-lightbox-nav tpl-lightbox-prev" onclick={prevImage}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<img
				src={String(currentImage || '')}
				alt={String(currentTitle || '')}
				class="tpl-lightbox-image"
			/>
			<button class="tpl-lightbox-nav tpl-lightbox-next" onclick={nextImage}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 5l7 7-7 7" />
				</svg>
			</button>
			<div class="tpl-lightbox-footer">
				{#if currentTitle}
					<span class="tpl-lightbox-title">{currentTitle}</span>
				{/if}
				<span class="tpl-lightbox-counter">{lightboxIndex + 1} / {data.length}</span>
			</div>
		</div>
	</div>
{/if}

<style>
	.tpl-gallery-view {
		padding: var(--tpl-space-4);
	}

	.tpl-gallery-grid {
		display: grid;
	}

	.tpl-gallery-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--tpl-space-16);
		color: var(--tpl-text-muted);
		font-family: var(--tpl-font-sans);
	}

	.tpl-gallery-empty-icon {
		width: 64px;
		height: 64px;
		margin-bottom: var(--tpl-space-4);
	}

	.tpl-gallery-empty-icon svg {
		width: 100%;
		height: 100%;
	}

	.tpl-gallery-skeleton {
		border-radius: var(--tpl-radius-lg);
		overflow: hidden;
	}

	.tpl-gallery-item {
		position: relative;
		border-radius: var(--tpl-radius-lg);
		overflow: hidden;
		background: var(--tpl-bg-secondary);
		transition: all var(--tpl-transition-fast);
	}

	.tpl-gallery-item:hover {
		transform: scale(1.02);
		box-shadow: var(--tpl-shadow-lg);
	}

	.tpl-gallery-item-selected {
		outline: 3px solid var(--tpl-accent-primary);
		outline-offset: 2px;
	}

	.tpl-gallery-image-btn {
		width: 100%;
		height: 100%;
		padding: 0;
		border: none;
		background: none;
		cursor: pointer;
	}

	.tpl-gallery-image {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.tpl-gallery-placeholder {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--tpl-bg-tertiary);
		color: var(--tpl-text-muted);
	}

	.tpl-gallery-placeholder svg {
		width: 48px;
		height: 48px;
	}

	.tpl-gallery-overlay {
		position: absolute;
		inset: 0;
		background: linear-gradient(to top, rgba(0,0,0,0.7) 0%, transparent 50%);
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		padding: var(--tpl-space-3);
		opacity: 0;
		transition: opacity var(--tpl-transition-fast);
	}

	.tpl-gallery-item:hover .tpl-gallery-overlay {
		opacity: 1;
	}

	.tpl-gallery-info {
		display: flex;
		flex-direction: column;
		gap: var(--tpl-space-0-5);
	}

	.tpl-gallery-title {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: white;
	}

	.tpl-gallery-subtitle {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: rgba(255,255,255,0.8);
	}

	.tpl-gallery-checkbox {
		position: absolute;
		top: var(--tpl-space-3);
		right: var(--tpl-space-3);
		width: 20px;
		height: 20px;
		cursor: pointer;
		accent-color: var(--tpl-accent-primary);
	}

	/* Lightbox */
	.tpl-lightbox {
		position: fixed;
		inset: 0;
		z-index: var(--tpl-z-modal);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.tpl-lightbox-backdrop {
		position: absolute;
		inset: 0;
		background: rgba(0, 0, 0, 0.9);
	}

	.tpl-lightbox-content {
		position: relative;
		max-width: 90vw;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.tpl-lightbox-close {
		position: absolute;
		top: -48px;
		right: 0;
		width: 40px;
		height: 40px;
		padding: 0;
		border: none;
		background: transparent;
		color: white;
		cursor: pointer;
		transition: opacity var(--tpl-transition-fast);
	}

	.tpl-lightbox-close:hover {
		opacity: 0.7;
	}

	.tpl-lightbox-close svg {
		width: 24px;
		height: 24px;
	}

	.tpl-lightbox-nav {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		width: 48px;
		height: 48px;
		padding: 0;
		border: none;
		background: rgba(255,255,255,0.1);
		border-radius: var(--tpl-radius-full);
		color: white;
		cursor: pointer;
		transition: background var(--tpl-transition-fast);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.tpl-lightbox-nav:hover {
		background: rgba(255,255,255,0.2);
	}

	.tpl-lightbox-nav svg {
		width: 24px;
		height: 24px;
	}

	.tpl-lightbox-prev {
		left: -64px;
	}

	.tpl-lightbox-next {
		right: -64px;
	}

	.tpl-lightbox-image {
		max-width: 100%;
		max-height: 80vh;
		object-fit: contain;
		border-radius: var(--tpl-radius-lg);
	}

	.tpl-lightbox-footer {
		margin-top: var(--tpl-space-4);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--tpl-space-1);
		color: white;
		font-family: var(--tpl-font-sans);
	}

	.tpl-lightbox-title {
		font-size: var(--tpl-text-base);
		font-weight: var(--tpl-font-medium);
	}

	.tpl-lightbox-counter {
		font-size: var(--tpl-text-sm);
		opacity: 0.7;
	}
</style>
