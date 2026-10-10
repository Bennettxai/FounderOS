<script lang="ts">
	import {
		desktopSettings,
		desktopBackgrounds,
		backgroundFitOptions
	} from '$lib/stores/desktopStore';

	let customImageUrl = $state('');
	let fileInput: HTMLInputElement | undefined = $state(undefined);

	// Carousel scroll refs
	let colorScrollContainer: HTMLDivElement | undefined = $state(undefined);
	let gradientScrollContainer: HTMLDivElement | undefined = $state(undefined);
	let patternScrollContainer: HTMLDivElement | undefined = $state(undefined);

	// Tooltip state
	let tooltipText = $state('');
	let tooltipVisible = $state(false);
	let tooltipX = $state(0);
	let tooltipY = $state(0);

	function showTooltip(event: MouseEvent, text: string) {
		tooltipText = text;
		tooltipX = event.clientX;
		tooltipY = event.clientY - 30;
		tooltipVisible = true;
	}

	function hideTooltip() {
		tooltipVisible = false;
	}

	function moveTooltip(event: MouseEvent) {
		tooltipX = event.clientX;
		tooltipY = event.clientY - 30;
	}

	function handleBackgroundChange(backgroundId: string) {
		desktopSettings.setBackground(backgroundId);
	}

	function applyCustomImage() {
		if (customImageUrl.trim()) {
			desktopSettings.setCustomBackground(customImageUrl.trim());
		}
	}

	function handleFileUpload(event: Event) {
		const target = event.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file && file.type.startsWith('image/')) {
			const reader = new FileReader();
			reader.onload = (e) => {
				const dataUrl = e.target?.result as string;
				desktopSettings.setCustomBackground(dataUrl);
			};
			reader.readAsDataURL(file);
		}
	}

	function triggerFileUpload() {
		fileInput?.click();
	}

	function scrollCarousel(container: HTMLDivElement | undefined, direction: 'left' | 'right') {
		if (!container) return;
		const scrollAmount = 200;
		container.scrollBy({
			left: direction === 'right' ? scrollAmount : -scrollAmount,
			behavior: 'smooth'
		});
	}
</script>

<!-- Background Selection -->
<div class="section">
	<label class="section-title">Solid Colors</label>
	<div class="carousel-container">
		<button class="carousel-btn left" onclick={() => scrollCarousel(colorScrollContainer, 'left')} aria-label="Scroll left">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M15 18l-6-6 6-6"/>
			</svg>
		</button>
		<div class="carousel-scroll" bind:this={colorScrollContainer}>
			<div class="color-grid">
				{#each desktopBackgrounds.filter(b => b.type === 'solid') as bg}
					<button
						class="color-swatch"
						class:selected={$desktopSettings.backgroundId === bg.id}
						style="background: {bg.preview};"
						onclick={() => handleBackgroundChange(bg.id)}
						onmouseenter={(e) => showTooltip(e, bg.name)}
						onmousemove={moveTooltip}
						onmouseleave={hideTooltip}
						aria-label={bg.name}
					></button>
				{/each}
			</div>
		</div>
		<button class="carousel-btn right" onclick={() => scrollCarousel(colorScrollContainer, 'right')} aria-label="Scroll right">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M9 18l6-6-6-6"/>
			</svg>
		</button>
	</div>
</div>

<div class="section">
	<label class="section-title">Gradients</label>
	<div class="carousel-container">
		<button class="carousel-btn left" onclick={() => scrollCarousel(gradientScrollContainer, 'left')} aria-label="Scroll left">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M15 18l-6-6 6-6"/>
			</svg>
		</button>
		<div class="carousel-scroll" bind:this={gradientScrollContainer}>
			<div class="gradient-grid">
				{#each desktopBackgrounds.filter(b => b.type === 'gradient') as bg}
					<button
						class="gradient-swatch"
						class:selected={$desktopSettings.backgroundId === bg.id}
						style="background: {bg.preview};"
						onclick={() => handleBackgroundChange(bg.id)}
						onmouseenter={(e) => showTooltip(e, bg.name)}
						onmousemove={moveTooltip}
						onmouseleave={hideTooltip}
						aria-label={bg.name}
					></button>
				{/each}
			</div>
		</div>
		<button class="carousel-btn right" onclick={() => scrollCarousel(gradientScrollContainer, 'right')} aria-label="Scroll right">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M9 18l6-6-6-6"/>
			</svg>
		</button>
	</div>
</div>

<div class="section">
	<label class="section-title">Patterns</label>
	<div class="carousel-container">
		<button class="carousel-btn left" onclick={() => scrollCarousel(patternScrollContainer, 'left')} aria-label="Scroll left">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M15 18l-6-6 6-6"/>
			</svg>
		</button>
		<div class="carousel-scroll" bind:this={patternScrollContainer}>
			<div class="pattern-grid">
				{#each desktopBackgrounds.filter(b => b.type === 'pattern') as bg}
					<button
						class="pattern-swatch"
						class:selected={$desktopSettings.backgroundId === bg.id}
						style="background: {bg.preview}; background-size: 10px 10px;"
						onclick={() => handleBackgroundChange(bg.id)}
						onmouseenter={(e) => showTooltip(e, bg.name)}
						onmousemove={moveTooltip}
						onmouseleave={hideTooltip}
						aria-label={bg.name}
					></button>
				{/each}
			</div>
		</div>
		<button class="carousel-btn right" onclick={() => scrollCarousel(patternScrollContainer, 'right')} aria-label="Scroll right">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M9 18l6-6-6-6"/>
			</svg>
		</button>
	</div>
</div>

<div class="section">
	<label class="section-title">Custom Image</label>
	<input
		type="file"
		accept="image/*"
		bind:this={fileInput}
		onchange={handleFileUpload}
		class="hidden-file-input"
	/>

	{#if $desktopSettings.backgroundId === 'custom' && $desktopSettings.customBackgroundUrl}
		<!-- Show current custom background with preview -->
		<div class="custom-preview-container">
			<div
				class="custom-preview-image"
				style="background-image: url({$desktopSettings.customBackgroundUrl});"
			></div>
			<div class="custom-preview-info">
				<span class="preview-label">
					{#if $desktopSettings.customBackgroundUrl.startsWith('data:')}
						Uploaded Image
					{:else}
						Custom URL
					{/if}
				</span>
				<span class="preview-status">Currently active</span>
			</div>
			<div class="custom-preview-actions">
				<button class="change-btn" onclick={triggerFileUpload}>
					Change
				</button>
				<button class="remove-btn" onclick={() => desktopSettings.setBackground('classic-gray')}>
					Remove
				</button>
			</div>
		</div>

		<!-- Fit options -->
		<div class="fit-options">
			<span class="fit-label">Image Fit:</span>
			<div class="fit-buttons">
				{#each backgroundFitOptions as fit}
					<button
						class="fit-btn"
						class:active={$desktopSettings.backgroundFit === fit.id}
						onclick={() => desktopSettings.setBackgroundFit(fit.id)}
						title={fit.description}
					>
						{fit.name}
					</button>
				{/each}
			</div>
		</div>
	{:else}
		<!-- Show upload options when no custom background -->
		<div class="custom-image-options">
			<button class="upload-btn" onclick={triggerFileUpload}>
				<svg class="upload-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M17 8l-5-5-5 5M12 3v12"/>
				</svg>
				Upload Image
			</button>
			<span class="or-divider">or</span>
			<div class="url-input-row">
				<input
					type="text"
					placeholder="Paste image URL..."
					class="custom-url-input"
					bind:value={customImageUrl}
					onkeydown={(e) => e.key === 'Enter' && applyCustomImage()}
				/>
				<button class="apply-btn" onclick={applyCustomImage}>
					Apply
				</button>
			</div>
		</div>
	{/if}
</div>

<!-- Custom Tooltip -->
{#if tooltipVisible}
	<div
		class="custom-tooltip"
		style="left: {tooltipX}px; top: {tooltipY}px;"
	>
		{tooltipText}
	</div>
{/if}

<style>
	.section {
		margin-bottom: 24px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		display: block;
		margin-bottom: 12px;
	}

	/* Carousel styles */
	.carousel-container {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.carousel-btn {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		border: 1px solid #ddd;
		background: var(--dbg);
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: all 0.15s ease;
	}

	.carousel-btn:hover {
		background: #f5f5f5;
		border-color: #ccc;
	}

	.carousel-btn svg {
		width: 14px;
		height: 14px;
		color: #666;
	}

	.carousel-scroll {
		flex: 1;
		overflow-x: auto;
		overflow-y: hidden;
		scrollbar-width: none;
		-ms-overflow-style: none;
	}

	.carousel-scroll::-webkit-scrollbar {
		display: none;
	}

	.color-grid {
		display: flex;
		gap: 8px;
		padding: 4px 0;
	}

	.color-swatch {
		width: 36px;
		height: 36px;
		flex-shrink: 0;
		border-radius: 8px;
		border: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.color-swatch:hover {
		transform: scale(1.1);
	}

	.color-swatch.selected {
		border-color: #333;
		box-shadow: 0 0 0 2px white, 0 0 0 4px #333;
	}

	.gradient-grid {
		display: flex;
		gap: 8px;
		padding: 4px 0;
	}

	.gradient-swatch {
		width: 72px;
		height: 48px;
		flex-shrink: 0;
		border-radius: 8px;
		border: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.gradient-swatch:hover {
		transform: scale(1.02);
	}

	.gradient-swatch.selected {
		border-color: #333;
		box-shadow: 0 0 0 2px white, 0 0 0 4px #333;
	}

	.pattern-grid {
		display: flex;
		gap: 8px;
		padding: 4px 0;
	}

	.pattern-swatch {
		width: 64px;
		height: 48px;
		flex-shrink: 0;
		border-radius: 8px;
		border: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.pattern-swatch:hover {
		transform: scale(1.02);
	}

	.pattern-swatch.selected {
		border-color: #333;
		box-shadow: 0 0 0 2px white, 0 0 0 4px #333;
	}

	.hidden-file-input {
		display: none;
	}

	.custom-image-options {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.upload-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 14px 20px;
		background: #f5f5f5;
		border: 2px dashed #ccc;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 500;
		color: #666;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.upload-btn:hover {
		background: #eee;
		border-color: #999;
		color: #333;
	}

	.upload-icon {
		width: 18px;
		height: 18px;
	}

	.or-divider {
		text-align: center;
		font-size: 11px;
		color: #999;
		text-transform: uppercase;
	}

	.url-input-row {
		display: flex;
		gap: 8px;
	}

	.custom-url-input {
		flex: 1;
		padding: 10px 12px;
		border: 1px solid #ddd;
		border-radius: 6px;
		font-size: 13px;
		outline: none;
		transition: border-color 0.15s ease;
	}

	.custom-url-input:focus {
		border-color: #333;
	}

	.custom-url-input::placeholder {
		color: #999;
	}

	.apply-btn {
		padding: 10px 16px;
		background: #333;
		color: white;
		border: none;
		border-radius: 6px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.apply-btn:hover {
		background: #555;
	}

	.custom-preview-container {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px;
		background: var(--dbg);
		border: 1px solid #e5e5e5;
		border-radius: 8px;
	}

	.custom-preview-image {
		width: 64px;
		height: 64px;
		border-radius: 6px;
		background-size: cover;
		background-position: center;
		border: 1px solid #ddd;
		flex-shrink: 0;
	}

	.custom-preview-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.preview-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
	}

	.preview-status {
		font-size: 11px;
		color: #28a745;
	}

	.custom-preview-actions {
		display: flex;
		gap: 8px;
	}

	.change-btn {
		padding: 8px 14px;
		background: #f5f5f5;
		border: 1px solid #ddd;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		color: #333;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.change-btn:hover {
		background: #eee;
		border-color: #ccc;
	}

	.remove-btn {
		padding: 8px 14px;
		background: var(--dbg);
		border: 1px solid #dc3545;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		color: #dc3545;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.remove-btn:hover {
		background: #dc3545;
		color: white;
	}

	.fit-options {
		margin-top: 12px;
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.fit-label {
		font-size: 12px;
		font-weight: 500;
		color: #666;
	}

	.fit-buttons {
		display: flex;
		gap: 4px;
	}

	.fit-btn {
		padding: 6px 12px;
		background: #f5f5f5;
		border: 1px solid #ddd;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
		color: #666;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.fit-btn:hover {
		background: #eee;
		color: #333;
	}

	.fit-btn.active {
		background: #333;
		border-color: #333;
		color: white;
	}

	.custom-tooltip {
		position: fixed;
		background: rgba(0, 0, 0, 0.85);
		color: white;
		padding: 6px 10px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 500;
		pointer-events: none;
		z-index: 9999;
		transform: translateX(-50%);
		white-space: nowrap;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
	}
</style>
