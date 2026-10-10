<script lang="ts">
	import { desktopSettings, iconSizePresets } from '$lib/stores/desktopStore';

	function getSizeLabel(size: number): string {
		const preset = iconSizePresets.find(p => p.value === size);
		if (preset) return preset.label;
		return `${size}px`;
	}

	function handleIconSizeChange(event: Event) {
		const target = event.target as HTMLInputElement;
		desktopSettings.setIconSize(parseInt(target.value, 10));
	}
</script>

<!-- Icon Size -->
<div class="section">
	<div class="section-header">
		<label class="section-title">Icon Size</label>
		<span class="section-value">{getSizeLabel($desktopSettings.iconSize)}</span>
	</div>
	<div class="slider-row">
		<span class="slider-label">Small</span>
		<input
			type="range"
			min="32"
			max="128"
			step="8"
			value={$desktopSettings.iconSize}
			oninput={handleIconSizeChange}
			class="slider"
		/>
		<span class="slider-label">Large</span>
	</div>
	<!-- Size Preview -->
	<div class="size-preview">
		{#each [48, 64, 96] as previewSize}
			<div
				class="preview-icon"
				class:active={$desktopSettings.iconSize === previewSize}
			>
				<div
					class="preview-box"
					style="width: {previewSize * 0.6}px; height: {previewSize * 0.6}px;"
				>
					<svg
						style="width: {previewSize * 0.35}px; height: {previewSize * 0.35}px;"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
					</svg>
				</div>
				<span class="preview-label">{previewSize}px</span>
			</div>
		{/each}
	</div>
</div>

<style>
	.section {
		margin-bottom: 24px;
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: #333;
		display: block;
		margin-bottom: 0;
	}

	.section-value {
		font-size: 12px;
		color: #666;
	}

	.slider-row {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.slider-label {
		font-size: 11px;
		color: #999;
	}

	.slider {
		flex: 1;
		height: 4px;
		background: #e5e5e5;
		border-radius: 2px;
		appearance: none;
		cursor: pointer;
	}

	.slider::-webkit-slider-thumb {
		appearance: none;
		width: 16px;
		height: 16px;
		background: #333;
		border-radius: 50%;
		cursor: pointer;
	}

	.size-preview {
		display: flex;
		align-items: flex-end;
		justify-content: center;
		gap: 24px;
		padding: 20px;
		background: white;
		border-radius: 8px;
		margin-top: 16px;
	}

	.preview-icon {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		opacity: 0.4;
		transition: all 0.2s ease;
	}

	.preview-icon.active {
		opacity: 1;
		transform: scale(1.1);
	}

	.preview-box {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
	}

	.preview-label {
		font-size: 10px;
		color: #666;
	}
</style>
