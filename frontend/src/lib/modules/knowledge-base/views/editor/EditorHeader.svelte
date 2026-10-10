<script lang="ts">
	import { FileText, Image, X, Upload, Link2, Move } from 'lucide-svelte';
	import { Popover, Button } from '$lib/ui';
	import type { DocumentIcon } from '../../entities/types';
	import PageIconPicker, { iconLibrary } from './PageIconPicker.svelte';

	interface Props {
		title: string;
		icon: DocumentIcon | null;
		cover: string | null;
		readOnly?: boolean;
		isHoveringCover?: boolean;
		isRepositioning?: boolean;
		onTitleChange?: (title: string) => void;
		onIconChange?: (icon: string | null) => void;
		onCoverChange?: (cover: string | null) => void;
		onStartReposition?: () => void;
	}

	let {
		title,
		icon,
		cover,
		readOnly = false,
		isHoveringCover = false,
		isRepositioning = false,
		onTitleChange,
		onIconChange,
		onCoverChange,
		onStartReposition
	}: Props = $props();

	let showIconPicker = $state(false);
	let showCoverPicker = $state(false);
	let isHoveringHeader = $state(false);
	let titleElement: HTMLHeadingElement | undefined = $state();
	let lastExternalTitle = $state('');
	let coverUrlInput = $state('');
	let coverFileInput: HTMLInputElement | null = $state(null);
	let coverPickerTab = $state<'gallery' | 'upload' | 'link'>('gallery');

	// Solid colors
	const solidColors = [
		'#e03e3e', // Red
		'#d9730d', // Orange
		'#dfab01', // Yellow
		'#0f7b6c', // Teal
		'#0b6e99', // Blue
		'#6940a5', // Purple
		'#ad1a72', // Pink
		'#64473a'  // Brown
	];

	// Predefined gradient covers (like Notion)
	const gradientCovers = [
		'linear-gradient(135deg, #fad0c4 0%, #ffd1ff 100%)',
		'linear-gradient(135deg, #a1c4fd 0%, #c2e9fb 100%)',
		'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
		'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
		'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
		'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
		'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
		'linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%)'
	];

	// Curated image collections (NASA/Space themed like Notion)
	const imageCollections = [
		{
			name: 'Abstract',
			images: [
				'https://images.unsplash.com/photo-1557672172-298e090bd0f1?w=800&q=80',
				'https://images.unsplash.com/photo-1579546929518-9e396f3cc809?w=800&q=80',
				'https://images.unsplash.com/photo-1558591710-4b4a1ae0f04d?w=800&q=80',
				'https://images.unsplash.com/photo-1604076913837-52ab5629fba9?w=800&q=80'
			]
		},
		{
			name: 'Nature',
			images: [
				'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=800&q=80',
				'https://images.unsplash.com/photo-1470071459604-3b5ec3a7fe05?w=800&q=80',
				'https://images.unsplash.com/photo-1441974231531-c6227db76b6e?w=800&q=80',
				'https://images.unsplash.com/photo-1469474968028-56623f02e42e?w=800&q=80'
			]
		},
		{
			name: 'Space',
			images: [
				'https://images.unsplash.com/photo-1462331940025-496dfbfc7564?w=800&q=80',
				'https://images.unsplash.com/photo-1446776811953-b23d57bd21aa?w=800&q=80',
				'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=800&q=80',
				'https://images.unsplash.com/photo-1419242902214-272b3f66ee7a?w=800&q=80'
			]
		}
	];

	// Sync title from prop ONLY when it changes externally (document switch)
	$effect(() => {
		if (title !== lastExternalTitle && titleElement) {
			// External change (document switched) - update the contenteditable
			titleElement.textContent = title;
			lastExternalTitle = title;
		}
	});

	function handleTitleInput(e: Event) {
		const target = e.target as HTMLElement;
		const newTitle = target.textContent || '';
		lastExternalTitle = newTitle; // Track as user input so we don't overwrite
		onTitleChange?.(newTitle);
	}

	function handleTitleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			// Move focus to first block
		}
	}

	function handleTitleMount(node: HTMLHeadingElement) {
		titleElement = node;
		node.textContent = title;
	}

	function handleIconSelect(iconName: string | null) {
		onIconChange?.(iconName);
		showIconPicker = false;
	}

	function removeCover() {
		onCoverChange?.(null);
		showCoverPicker = false;
	}

	function handleCoverFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (file) {
			const url = URL.createObjectURL(file);
			onCoverChange?.(url);
			showCoverPicker = false;
		}
	}

	function handleCoverUrlSubmit() {
		if (coverUrlInput.trim()) {
			onCoverChange?.(coverUrlInput.trim());
			coverUrlInput = '';
			showCoverPicker = false;
		}
	}

	function handleGradientSelect(gradient: string) {
		onCoverChange?.(gradient);
		showCoverPicker = false;
	}

	function triggerFileUpload() {
		coverFileInput?.click();
	}

	// Get icon name from DocumentIcon - handles both string and object formats
	function getIconName(icon: DocumentIcon | null): string | null {
		if (!icon) return null;
		if (typeof icon === 'string') return icon;
		if (icon.type === 'icon') return icon.value;
		return null;
	}

	// Get SVG path for icon
	function getIconPath(iconName: string | null): string | null {
		if (!iconName) return null;
		return iconLibrary[iconName] || null;
	}

	const currentIconName = $derived(getIconName(icon));
	const currentIconPath = $derived(getIconPath(currentIconName));
	const isGradientCover = $derived(cover?.startsWith('linear-gradient'));
	const isSolidColor = $derived(cover?.startsWith('#') && cover?.length === 7);
</script>

<div
	class="editor-header"
	class:editor-header--has-cover={cover}
	onmouseenter={() => (isHoveringHeader = true)}
	onmouseleave={() => (isHoveringHeader = false)}
>
	<!-- Cover controls overlay (cover image rendered by DocumentEditor) -->
	{#if cover}
		<div class="editor-header__cover-overlay">
			{#if !readOnly && (isHoveringCover || isHoveringHeader || showCoverPicker)}
				<div class="editor-header__cover-actions">
					<Popover bind:open={showCoverPicker} side="bottom" align="end">
						{#snippet trigger()}
							<button class="btn-pill btn-pill-ghost editor-header__cover-btn" aria-label="Change cover">
								<span>Change cover</span>
							</button>
						{/snippet}
						<div class="cover-picker">
							<!-- Tabs -->
							<div class="cover-picker__tabs">
								<button
									class="cover-picker__tab"
									class:cover-picker__tab--active={coverPickerTab === 'gallery'}
									onclick={() => coverPickerTab = 'gallery'}
								>Gallery</button>
								<button
									class="cover-picker__tab"
									class:cover-picker__tab--active={coverPickerTab === 'upload'}
									onclick={() => coverPickerTab = 'upload'}
								>Upload</button>
								<button
									class="cover-picker__tab"
									class:cover-picker__tab--active={coverPickerTab === 'link'}
									onclick={() => coverPickerTab = 'link'}
								>Link</button>
								<button class="cover-picker__tab cover-picker__tab--remove" onclick={removeCover}>
									Remove
								</button>
							</div>

							<div class="cover-picker__content">
								{#if coverPickerTab === 'gallery'}
									<!-- Color & Gradient -->
									<div class="cover-picker__section">
										<div class="cover-picker__label">Color & Gradient</div>
										<div class="cover-picker__colors">
											{#each solidColors as color}
												<button
													class="cover-picker__color"
													style="background: {color}"
													onclick={() => handleGradientSelect(color)}
													aria-label="Select color"
												></button>
											{/each}
										</div>
										<div class="cover-picker__gradients">
											{#each gradientCovers as gradient}
												<button
													class="cover-picker__gradient"
													style="background: {gradient}"
													onclick={() => handleGradientSelect(gradient)}
													aria-label="Select gradient"
												></button>
											{/each}
										</div>
									</div>

									<!-- Image collections -->
									{#each imageCollections as collection}
										<div class="cover-picker__section">
											<div class="cover-picker__label">{collection.name}</div>
											<div class="cover-picker__images">
												{#each collection.images as imageUrl}
													<button
														class="cover-picker__image"
														onclick={() => { onCoverChange?.(imageUrl); showCoverPicker = false; }}
														aria-label="Select image"
													>
														<img src={imageUrl} alt="" loading="lazy" />
													</button>
												{/each}
											</div>
										</div>
									{/each}
								{:else if coverPickerTab === 'upload'}
									<div class="cover-picker__upload-section">
										<input
											type="file"
											accept="image/*"
											class="hidden"
											bind:this={coverFileInput}
											onchange={handleCoverFileSelect}
										/>
										<button class="cover-picker__upload-btn" onclick={triggerFileUpload}>
											<Upload class="h-5 w-5" />
											<span>Upload an image</span>
											<span class="cover-picker__upload-hint">Recommended size: 1500 x 300 pixels</span>
										</button>
									</div>
								{:else if coverPickerTab === 'link'}
									<div class="cover-picker__link-section">
										<div class="cover-picker__url-input">
											<input
												type="url"
												placeholder="Paste an image link..."
												bind:value={coverUrlInput}
												onkeydown={(e) => e.key === 'Enter' && handleCoverUrlSubmit()}
											/>
										</div>
										<button
											class="cover-picker__submit-btn"
											onclick={handleCoverUrlSubmit}
											disabled={!coverUrlInput.trim()}
										>
											Submit
										</button>
										<p class="cover-picker__link-hint">Works with any image from the web.</p>
									</div>
								{/if}
							</div>
						</div>
					</Popover>
					<button
						class="btn-pill btn-pill-ghost editor-header__cover-btn"
						class:editor-header__cover-btn--active={isRepositioning}
						onclick={onStartReposition}
						aria-label="Reposition cover"
					>
						<Move class="h-3.5 w-3.5" />
						<span>{isRepositioning ? 'Done' : 'Reposition'}</span>
					</button>
				</div>
			{/if}
		</div>
	{/if}

	<div class="editor-header__content" class:editor-header__content--with-cover={cover}>
		{#if !readOnly && (isHoveringHeader || showIconPicker || showCoverPicker) && (!icon || !cover)}
			<div class="editor-header__actions">
				{#if !icon}
					<Popover bind:open={showIconPicker} side="bottom" align="start">
						{#snippet trigger()}
							<button class="btn-pill btn-pill-ghost editor-header__action-btn">
								<FileText class="h-4 w-4" />
								<span>Add icon</span>
							</button>
						{/snippet}
						<PageIconPicker currentIcon={currentIconName} onSelect={handleIconSelect} />
					</Popover>
				{/if}

				{#if !cover}
					<Popover bind:open={showCoverPicker} side="bottom" align="start">
						{#snippet trigger()}
							<button class="btn-pill btn-pill-ghost editor-header__action-btn">
								<Image class="h-4 w-4" />
								<span>Add cover</span>
							</button>
						{/snippet}
						<div class="cover-picker">
							<!-- Tabs -->
							<div class="cover-picker__tabs">
								<button
									class="cover-picker__tab"
									class:cover-picker__tab--active={coverPickerTab === 'gallery'}
									onclick={() => coverPickerTab = 'gallery'}
								>Gallery</button>
								<button
									class="cover-picker__tab"
									class:cover-picker__tab--active={coverPickerTab === 'upload'}
									onclick={() => coverPickerTab = 'upload'}
								>Upload</button>
								<button
									class="cover-picker__tab"
									class:cover-picker__tab--active={coverPickerTab === 'link'}
									onclick={() => coverPickerTab = 'link'}
								>Link</button>
							</div>

							<div class="cover-picker__content">
								{#if coverPickerTab === 'gallery'}
									<!-- Color & Gradient -->
									<div class="cover-picker__section">
										<div class="cover-picker__label">Color & Gradient</div>
										<div class="cover-picker__colors">
											{#each solidColors as color}
												<button
													class="cover-picker__color"
													style="background: {color}"
													onclick={() => handleGradientSelect(color)}
													aria-label="Select color"
												></button>
											{/each}
										</div>
										<div class="cover-picker__gradients">
											{#each gradientCovers as gradient}
												<button
													class="cover-picker__gradient"
													style="background: {gradient}"
													onclick={() => handleGradientSelect(gradient)}
													aria-label="Select gradient"
												></button>
											{/each}
										</div>
									</div>

									<!-- Image collections -->
									{#each imageCollections as collection}
										<div class="cover-picker__section">
											<div class="cover-picker__label">{collection.name}</div>
											<div class="cover-picker__images">
												{#each collection.images as imageUrl}
													<button
														class="cover-picker__image"
														onclick={() => { onCoverChange?.(imageUrl); showCoverPicker = false; }}
														aria-label="Select image"
													>
														<img src={imageUrl} alt="" loading="lazy" />
													</button>
												{/each}
											</div>
										</div>
									{/each}
								{:else if coverPickerTab === 'upload'}
									<div class="cover-picker__upload-section">
										<input
											type="file"
											accept="image/*"
											class="hidden"
											bind:this={coverFileInput}
											onchange={handleCoverFileSelect}
										/>
										<button class="cover-picker__upload-btn" onclick={triggerFileUpload}>
											<Upload class="h-5 w-5" />
											<span>Upload an image</span>
											<span class="cover-picker__upload-hint">Recommended size: 1500 x 300 pixels</span>
										</button>
									</div>
								{:else if coverPickerTab === 'link'}
									<div class="cover-picker__link-section">
										<div class="cover-picker__url-input">
											<input
												type="url"
												placeholder="Paste an image link..."
												bind:value={coverUrlInput}
												onkeydown={(e) => e.key === 'Enter' && handleCoverUrlSubmit()}
											/>
										</div>
										<button
											class="cover-picker__submit-btn"
											onclick={handleCoverUrlSubmit}
											disabled={!coverUrlInput.trim()}
										>
											Submit
										</button>
										<p class="cover-picker__link-hint">Works with any image from the web.</p>
									</div>
								{/if}
							</div>
						</div>
					</Popover>
				{/if}
			</div>
		{/if}

		{#if icon}
			<div class="editor-header__icon-wrapper">
				<Popover bind:open={showIconPicker} side="bottom" align="start">
					{#snippet trigger()}
						<button class="btn-pill btn-pill-ghost editor-header__icon" disabled={readOnly}>
							{#if currentIconPath}
								<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
									<path d={currentIconPath} />
								</svg>
							{:else}
								<FileText class="h-12 w-12" />
							{/if}
						</button>
					{/snippet}
					<PageIconPicker currentIcon={currentIconName} onSelect={handleIconSelect} />
				</Popover>
			</div>
		{/if}

		<h1
			class="editor-header__title"
			contenteditable={!readOnly}
			oninput={handleTitleInput}
			onkeydown={handleTitleKeydown}
			data-placeholder="Untitled"
			use:handleTitleMount
		></h1>
	</div>
</div>

<style>
	.editor-header {
		position: relative;
		margin-bottom: 1rem;
	}

	/* Cover controls overlay - positioned over DocumentEditor's cover */
	.editor-header__cover-overlay {
		position: absolute;
		/* Pull up to cover the DocumentEditor cover which is above the container */
		top: -200px;
		left: -4rem;
		right: -4rem;
		height: 200px;
		pointer-events: none;
		width: calc(100% + 8rem);
	}

	.editor-header--has-cover {
		position: relative;
	}

	.editor-header__cover-actions {
		position: absolute;
		bottom: 0.75rem;
		right: 1rem;
		display: flex;
		gap: 0.5rem;
		opacity: 0;
		transition: opacity 0.15s;
		pointer-events: auto;
	}

	.editor-header__cover-overlay:hover .editor-header__cover-actions,
	.editor-header:hover .editor-header__cover-actions {
		opacity: 1;
	}

	.editor-header__cover-btn {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.5rem 0.75rem;
		background-color: hsl(var(--popover) / 0.85);
		backdrop-filter: blur(8px);
		border: 1px solid hsl(var(--border) / 0.5);
		border-radius: 0.375rem;
		color: hsl(var(--popover-foreground));
		font-size: 0.8125rem;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s, border-color 0.15s;
		box-shadow: 0 2px 8px hsl(var(--foreground) / 0.1);
	}

	.editor-header__cover-btn:hover {
		background-color: hsl(var(--popover) / 0.95);
		border-color: hsl(var(--border));
	}

	.editor-header__cover-btn--active {
		background-color: hsl(var(--primary));
		color: hsl(var(--primary-foreground));
		border-color: hsl(var(--primary));
	}

	.editor-header__cover-btn--active:hover {
		background-color: hsl(var(--primary) / 0.9);
		border-color: hsl(var(--primary) / 0.9);
	}

	/* Cover picker styles - Notion-like tabbed picker */
	.cover-picker {
		width: 480px;
		background: hsl(var(--background));
		border-radius: 0.5rem;
		overflow: hidden;
	}

	.cover-picker__tabs {
		display: flex;
		padding: 0.5rem 0.75rem;
		border-bottom: 1px solid hsl(var(--border));
		gap: 0.25rem;
	}

	.cover-picker__tab {
		padding: 0.375rem 0.75rem;
		background: transparent;
		border: none;
		border-radius: 0.25rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: hsl(var(--muted-foreground));
		cursor: pointer;
		transition: color 0.15s, background-color 0.15s;
	}

	.cover-picker__tab:hover {
		color: hsl(var(--foreground));
		background-color: hsl(var(--muted));
	}

	.cover-picker__tab--active {
		color: hsl(var(--foreground));
		background-color: hsl(var(--muted));
	}

	.cover-picker__tab--remove {
		margin-left: auto;
		color: hsl(var(--muted-foreground));
	}

	.cover-picker__tab--remove:hover {
		color: hsl(var(--destructive));
	}

	.cover-picker__content {
		max-height: 400px;
		overflow-y: auto;
		padding: 0.75rem;
	}

	.cover-picker__section {
		margin-bottom: 1rem;
	}

	.cover-picker__section:last-child {
		margin-bottom: 0;
	}

	.cover-picker__label {
		font-size: 0.75rem;
		font-weight: 500;
		color: hsl(var(--muted-foreground));
		margin-bottom: 0.5rem;
	}

	.cover-picker__colors {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.5rem;
		margin-bottom: 0.5rem;
	}

	.cover-picker__color {
		aspect-ratio: 2 / 1;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: transform 0.15s, box-shadow 0.15s;
	}

	.cover-picker__color:hover {
		transform: scale(1.05);
		box-shadow: 0 2px 8px hsl(var(--foreground) / 0.15);
	}

	.cover-picker__gradients {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.5rem;
	}

	.cover-picker__gradient {
		aspect-ratio: 2 / 1;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: transform 0.15s, box-shadow 0.15s;
	}

	.cover-picker__gradient:hover {
		transform: scale(1.05);
		box-shadow: 0 2px 8px hsl(var(--foreground) / 0.15);
	}

	.cover-picker__images {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.5rem;
	}

	.cover-picker__image {
		aspect-ratio: 16 / 10;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		overflow: hidden;
		padding: 0;
		background: hsl(var(--muted));
		transition: transform 0.15s, box-shadow 0.15s;
	}

	.cover-picker__image:hover {
		transform: scale(1.05);
		box-shadow: 0 2px 8px hsl(var(--foreground) / 0.15);
	}

	.cover-picker__image img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.cover-picker__upload-section,
	.cover-picker__link-section {
		padding: 1rem;
	}

	.cover-picker__upload-btn {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		padding: 2rem;
		background: hsl(var(--muted) / 0.5);
		border: 2px dashed hsl(var(--border));
		border-radius: 0.5rem;
		color: hsl(var(--muted-foreground));
		font-size: 0.875rem;
		cursor: pointer;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.cover-picker__upload-btn:hover {
		border-color: hsl(var(--muted-foreground));
		background: hsl(var(--muted));
	}

	.cover-picker__upload-hint {
		font-size: 0.75rem;
		color: hsl(var(--muted-foreground) / 0.7);
	}

	.cover-picker__url-input {
		display: flex;
		margin-bottom: 0.75rem;
	}

	.cover-picker__url-input input {
		flex: 1;
		padding: 0.625rem 0.875rem;
		background: hsl(var(--muted));
		border: 1px solid hsl(var(--border));
		border-radius: 0.375rem;
		font-size: 0.875rem;
		color: hsl(var(--foreground));
	}

	.cover-picker__url-input input::placeholder {
		color: hsl(var(--muted-foreground));
	}

	.cover-picker__url-input input:focus {
		outline: none;
		border-color: hsl(var(--primary));
	}

	.cover-picker__submit-btn {
		width: 100%;
		padding: 0.625rem;
		background: hsl(var(--primary));
		border: none;
		border-radius: 0.375rem;
		color: hsl(var(--primary-foreground));
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: opacity 0.15s;
	}

	.cover-picker__submit-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.cover-picker__submit-btn:hover:not(:disabled) {
		opacity: 0.9;
	}

	.cover-picker__link-hint {
		margin-top: 0.75rem;
		font-size: 0.75rem;
		color: hsl(var(--muted-foreground));
		text-align: center;
	}

	.hidden {
		display: none;
	}

	.editor-header__content {
		padding-top: 3rem;
	}

	.editor-header__content--with-cover {
		padding-top: 1.5rem;
	}

	.editor-header__actions {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.editor-header:hover .editor-header__actions {
		opacity: 1;
	}

	.editor-header__action-btn {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.625rem;
		background: transparent;
		border: none;
		border-radius: 0.375rem;
		color: hsl(var(--muted-foreground));
		font-size: 0.875rem;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.editor-header__action-btn:hover {
		background-color: hsl(var(--muted));
	}

	.editor-header__icon-wrapper {
		margin-bottom: 0.5rem;
	}

	.editor-header__icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 78px;
		height: 78px;
		background: transparent;
		border: none;
		border-radius: 0.5rem;
		cursor: pointer;
		transition: background-color 0.15s;
		color: hsl(var(--foreground));
	}

	.editor-header__icon:hover:not(:disabled) {
		background-color: hsl(var(--muted));
	}

	.editor-header__icon svg {
		width: 48px;
		height: 48px;
	}

	.editor-header__title {
		font-size: 2.5rem;
		font-weight: 700;
		line-height: 1.2;
		color: hsl(var(--foreground));
		outline: none;
		word-break: break-word;
	}

	.editor-header__title:empty::before {
		content: attr(data-placeholder);
		color: hsl(var(--muted-foreground) / 0.5);
	}
</style>
