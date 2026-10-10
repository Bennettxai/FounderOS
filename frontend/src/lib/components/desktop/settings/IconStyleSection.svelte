<script lang="ts">
	import {
		desktopSettings,
		iconStyles,
		iconLibraries,
		type IconStyle,
		type IconLibrary
	} from '$lib/stores/desktopStore';

	type StyleCategory = 'modern' | 'classic' | 'creative' | 'all';

	let selectedStyleCategory = $state<StyleCategory>('modern');

	const styleCategories: Record<string, string[]> = {
		modern: ['default', 'minimal', 'rounded', 'square', 'macos', 'glassmorphism', 'frosted', 'flat', 'paper', 'depth', 'neumorphism', 'material', 'fluent', 'aero'],
		classic: ['macos-classic', 'retro', 'win95', 'pixel', 'ios', 'android', 'windows11', 'amiga'],
		creative: ['outlined', 'neon', 'gradient', 'glow', 'terminal', 'brutalist', 'aurora', 'crystal', 'holographic', 'vaporwave', 'cyberpunk', 'synthwave', 'matrix', 'glitch', 'chrome', 'rainbow', 'sketch', 'comic', 'watercolor']
	};

	function getFilteredStyles() {
		if (selectedStyleCategory === 'all') {
			return iconStyles;
		}
		const categoryIds = styleCategories[selectedStyleCategory] || [];
		return iconStyles.filter(style => categoryIds.includes(style.id));
	}

	function handleIconStyleChange(style: IconStyle) {
		desktopSettings.setIconStyle(style);
	}

	function handleIconLibraryChange(library: IconLibrary) {
		desktopSettings.setIconLibrary(library);
	}
</script>

<!-- Icon Style -->
<div class="section">
	<label class="section-title">Icon Style</label>
	<p class="section-subtitle">Choose how your icons look</p>

	<!-- Style Category Filter -->
	<div class="style-filter">
		<button
			class="filter-btn"
			class:active={selectedStyleCategory === 'modern'}
			onclick={() => selectedStyleCategory = 'modern'}
		>
			Modern
		</button>
		<button
			class="filter-btn"
			class:active={selectedStyleCategory === 'classic'}
			onclick={() => selectedStyleCategory = 'classic'}
		>
			Classic
		</button>
		<button
			class="filter-btn"
			class:active={selectedStyleCategory === 'creative'}
			onclick={() => selectedStyleCategory = 'creative'}
		>
			Creative
		</button>
		<button
			class="filter-btn"
			class:active={selectedStyleCategory === 'all'}
			onclick={() => selectedStyleCategory = 'all'}
		>
			All
		</button>
	</div>

	<!-- Styles Grid with Visual Previews -->
	<div class="styles-grid">
		{#each getFilteredStyles() as style}
			<button
				class="style-item"
				class:selected={$desktopSettings.iconStyle === style.id}
				onclick={() => handleIconStyleChange(style.id)}
			>
				<div class="style-icon preview-{style.id}">
					<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
						<polyline points="9 22 9 12 15 12 15 22" />
					</svg>
				</div>
				<div class="style-text">
					<div class="style-name">{style.name}</div>
					<div class="style-desc">{style.description}</div>
				</div>
			</button>
		{/each}
	</div>
</div>

<!-- Advanced Customization -->
<div class="section">
	<label class="section-title">Advanced Customization</label>
	<div class="advanced-options">
		<!-- Icon Spacing -->
		<div class="option-row">
			<div class="option-label">
				<span class="option-name">Icon Spacing</span>
				<span class="option-hint">Space between desktop icons</span>
			</div>
			<div class="option-control">
				<input
					type="range"
					min="8"
					max="32"
					value={$desktopSettings.iconSpacing ?? 16}
					oninput={(e) => desktopSettings.setIconSpacing(parseInt((e.target as HTMLInputElement).value, 10))}
					class="slider-modern"
				/>
				<span class="value-display">{$desktopSettings.iconSpacing ?? 16}px</span>
			</div>
		</div>

		<!-- Icon Shadow -->
		<div class="option-row">
			<div class="option-label">
				<span class="option-name">Icon Shadow</span>
				<span class="option-hint">Add shadow effect to icons</span>
			</div>
			<div class="option-control">
				<label class="adv-toggle-switch" aria-label="Toggle icon shadow">
					<input
						type="checkbox"
						checked={$desktopSettings.iconShadow !== false}
						onchange={(e) => desktopSettings.setIconShadow((e.target as HTMLInputElement).checked)}
					/>
					<span class="adv-toggle-slider"></span>
				</label>
			</div>
		</div>

		<!-- Icon Border -->
		<div class="option-row">
			<div class="option-label">
				<span class="option-name">Icon Border</span>
				<span class="option-hint">Add border around icons</span>
			</div>
			<div class="option-control">
				<label class="adv-toggle-switch" aria-label="Toggle icon border">
					<input
						type="checkbox"
						checked={$desktopSettings.iconBorder || false}
						onchange={(e) => desktopSettings.setIconBorder((e.target as HTMLInputElement).checked)}
					/>
					<span class="adv-toggle-slider"></span>
				</label>
			</div>
		</div>

		<!-- Icon Hover Effect -->
		<div class="option-row">
			<div class="option-label">
				<span class="option-name">Hover Animation</span>
				<span class="option-hint">Scale up icons on hover</span>
			</div>
			<div class="option-control">
				<label class="adv-toggle-switch" aria-label="Toggle icon hover animation">
					<input
						type="checkbox"
						checked={$desktopSettings.iconHoverEffect !== false}
						onchange={(e) => desktopSettings.setIconHoverEffect((e.target as HTMLInputElement).checked)}
					/>
					<span class="adv-toggle-slider"></span>
				</label>
			</div>
		</div>
	</div>
</div>

<!-- Line Weight / Icon Rendering -->
<div class="section">
	<label class="section-title">Line Weight</label>
	<div class="library-grid">
		{#each iconLibraries as lib}
			<button
				class="library-option"
				class:selected={$desktopSettings.iconLibrary === lib.id}
				onclick={() => handleIconLibraryChange(lib.id)}
			>
				<div class="library-header">
					<span class="library-name">{lib.name}</span>
					<span class="library-preview">{lib.preview}</span>
				</div>
				<div class="library-desc">{lib.description}</div>
				<!-- Visual preview of stroke weight -->
				<div class="stroke-preview stroke-{lib.id}">
					<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor"
						stroke-width={lib.id === 'lucide' ? 2 : lib.id === 'phosphor' ? 3 : lib.id === 'tabler' ? 1.2 : 2.5}
						stroke-linecap="round" stroke-linejoin="round">
						<path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
					</svg>
				</div>
			</button>
		{/each}
	</div>
	<p class="library-hint">Changes how thick the icon lines appear</p>
</div>

<!-- Toggles -->
<div class="section">
	<label class="section-title">Options</label>
	<div class="toggles">
		<div class="toggle-row">
			<div class="toggle-info">
				<div class="toggle-label">Show Icon Labels</div>
				<div class="toggle-desc">Display text labels under icons</div>
			</div>
			<button
				onclick={() => desktopSettings.toggleIconLabels()}
				class="toggle-switch"
				class:active={$desktopSettings.showIconLabels}
				role="switch"
				aria-checked={$desktopSettings.showIconLabels}
			>
				<span class="toggle-thumb"></span>
			</button>
		</div>

		<div class="toggle-row">
			<div class="toggle-info">
				<div class="toggle-label">Snap to Grid</div>
				<div class="toggle-desc">Align icons to grid when dragging</div>
			</div>
			<button
				onclick={() => desktopSettings.toggleGridSnap()}
				class="toggle-switch"
				class:active={$desktopSettings.gridSnap}
				role="switch"
				aria-checked={$desktopSettings.gridSnap}
			>
				<span class="toggle-thumb"></span>
			</button>
		</div>

		<div class="toggle-row">
			<div class="toggle-info">
				<div class="toggle-label">Noise Texture</div>
				<div class="toggle-desc">Add subtle noise overlay to background</div>
			</div>
			<button
				onclick={() => desktopSettings.toggleNoise()}
				class="toggle-switch"
				class:active={$desktopSettings.showNoise}
				role="switch"
				aria-checked={$desktopSettings.showNoise}
			>
				<span class="toggle-thumb"></span>
			</button>
		</div>
	</div>
</div>

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

	.section-subtitle {
		font-size: 12px;
		color: var(--dt2);
		margin: -4px 0 16px 0;
	}

	/* Style Category Filter */
	.style-filter {
		display: inline-flex;
		gap: 4px;
		margin-bottom: 16px;
		padding: 3px;
		background: rgba(0, 0, 0, 0.04);
		border-radius: 8px;
	}

	.filter-btn {
		padding: 6px 14px;
		border: none;
		background: transparent;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		color: var(--dt2);
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.filter-btn:hover {
		background: rgba(0, 0, 0, 0.04);
		color: var(--dt);
	}

	.filter-btn.active {
		background: var(--dbg);
		color: #0066FF;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}

	/* Styles Grid */
	.styles-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 10px;
	}

	.style-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 12px 10px;
		border: 1.5px solid rgba(0, 0, 0, 0.08);
		border-radius: 10px;
		background: var(--dbg);
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: center;
	}

	.style-item:hover {
		border-color: #0066FF;
		background: rgba(0, 102, 255, 0.02);
		transform: translateY(-1px);
	}

	.style-item.selected {
		border-color: #0066FF;
		background: rgba(0, 102, 255, 0.06);
		box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.1);
	}

	.style-icon {
		width: 44px;
		height: 44px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		border-radius: 10px;
		flex-shrink: 0;
		color: white;
	}

	/* Style-specific visual previews */
	.style-icon.preview-default { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; border-radius: 10px !important; }
	.style-icon.preview-rounded { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; border-radius: 50% !important; }
	.style-icon.preview-square { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; border-radius: 4px !important; }
	.style-icon.preview-minimal { background: transparent !important; border: 2px solid #667eea !important; color: #667eea !important; }
	.style-icon.preview-macos { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; border-radius: 28% !important; }
	.style-icon.preview-outlined { background: var(--dbg) !important; border: 2.5px solid #667eea !important; color: #667eea !important; }
	.style-icon.preview-glassmorphism { background: rgba(255, 255, 255, 0.2) !important; backdrop-filter: blur(10px) !important; border: 1px solid rgba(255, 255, 255, 0.4) !important; }
	.style-icon.preview-neon { background: #1a1a2e !important; box-shadow: 0 0 10px #667eea, 0 0 20px #667eea !important; border: 1.5px solid #667eea !important; }
	.style-icon.preview-flat { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; box-shadow: none !important; border-radius: 8px !important; }
	.style-icon.preview-retro { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; border-radius: 0 !important; box-shadow: 4px 4px 0 rgba(0, 0, 0, 0.3), inset -2px -2px 0 rgba(0, 0, 0, 0.2) !important; }
	.style-icon.preview-win95 { border-radius: 0 !important; background: #C0C0C0 !important; border: 2px solid !important; border-color: #DFDFDF #808080 #808080 #DFDFDF !important; color: var(--dt) !important; }
	.style-icon.preview-frosted { background: rgba(255, 255, 255, 0.6) !important; backdrop-filter: blur(12px) saturate(180%) !important; border: 1px solid rgba(255, 255, 255, 0.3) !important; }
	.style-icon.preview-terminal { background: #0a0a0a !important; border: 1.5px solid #00ff00 !important; box-shadow: 0 0 10px rgba(0, 255, 0, 0.3) !important; color: #00ff00 !important; }
	.style-icon.preview-glow { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; box-shadow: 0 0 15px #667eea, 0 0 30px rgba(102, 126, 234, 0.3) !important; }
	.style-icon.preview-paper { background: var(--dbg) !important; box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08) !important; border-radius: 6px !important; }
	.style-icon.preview-pixel { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; border-radius: 0 !important; image-rendering: pixelated !important; }
	.style-icon.preview-brutalist { background: var(--dbg) !important; border-radius: 0 !important; border: 3px solid #000 !important; box-shadow: 4px 4px 0 #000 !important; color: var(--dt) !important; }
	.style-icon.preview-depth { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; box-shadow: 0 1px 2px rgba(0,0,0,0.07), 0 2px 4px rgba(0,0,0,0.07), 0 4px 8px rgba(0,0,0,0.07), 0 8px 16px rgba(0,0,0,0.07) !important; }
	.style-icon.preview-macos-classic { border-radius: 4px !important; background: linear-gradient(180deg, #EAEAEA 0%, #D4D4D4 50%, #C4C4C4 100%) !important; border: 1px solid #999 !important; color: var(--dt) !important; }
	.style-icon.preview-gradient { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; }
	.style-icon.preview-neumorphism { background: #e0e0e0 !important; box-shadow: 8px 8px 16px #bebebe, -8px -8px 16px #ffffff !important; }
	.style-icon.preview-material { background: #667eea !important; box-shadow: 0 3px 6px rgba(0,0,0,0.16), 0 3px 6px rgba(0,0,0,0.23) !important; }
	.style-icon.preview-fluent { background: rgba(255, 255, 255, 0.7) !important; backdrop-filter: blur(30px) saturate(120%) !important; border: 1px solid rgba(255, 255, 255, 0.5) !important; }
	.style-icon.preview-aero { background: linear-gradient(180deg, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0.6) 100%) !important; backdrop-filter: blur(20px) !important; border: 1px solid rgba(255,255,255,0.5) !important; }
	.style-icon.preview-ios { border-radius: 22% !important; background: linear-gradient(135deg, #007AFF 0%, #5AC8FA 100%) !important; }
	.style-icon.preview-android { border-radius: 28% !important; background: linear-gradient(135deg, #34A853 0%, #FBBC05 50%, #EA4335 100%) !important; }
	.style-icon.preview-windows11 { border-radius: 8px !important; background: linear-gradient(180deg, #0067C0 0%, #003D82 100%) !important; }
	.style-icon.preview-amiga { border-radius: 0 !important; background: linear-gradient(180deg, #FF8800 0%, #FF6600 100%) !important; }
	.style-icon.preview-aurora { background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%) !important; }
	.style-icon.preview-crystal { background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%) !important; box-shadow: inset 0 0 20px rgba(255,255,255,0.8), 0 0 20px rgba(168,237,234,0.6) !important; }
	.style-icon.preview-holographic { background: linear-gradient(135deg, #ff0080, #ff8c00, #40e0d0, #ff0080) !important; background-size: 400% 400% !important; }
	.style-icon.preview-vaporwave { background: linear-gradient(135deg, #ff71ce 0%, #01cdfe 100%) !important; box-shadow: 0 0 25px #ff71ce, inset 0 0 15px rgba(255,113,206,0.4) !important; }
	.style-icon.preview-cyberpunk { background: #0a0a0a !important; border: 3px solid #00ff41 !important; box-shadow: 0 0 15px #00ff41 !important; color: #00ff41 !important; }
	.style-icon.preview-synthwave { background: linear-gradient(135deg, #ff006e 0%, #8338ec 50%, #3a86ff 100%) !important; box-shadow: 0 0 25px #ff006e, 0 4px 20px rgba(255,0,110,0.5) !important; }
	.style-icon.preview-matrix { background: #000 !important; border: 3px solid #00ff00 !important; box-shadow: 0 0 20px #00ff00 !important; color: #00ff00 !important; }
	.style-icon.preview-glitch { background: #ff00ff !important; border: 2px solid #00ffff !important; box-shadow: 2px 2px 0 #ff0000, -2px -2px 0 #00ff00 !important; }
	.style-icon.preview-chrome { background: linear-gradient(135deg, #f5f5f5 0%, #b0b0b0 50%, #f5f5f5 100%) !important; box-shadow: inset 0 2px 4px rgba(255,255,255,0.9), inset 0 -2px 4px rgba(0,0,0,0.4), 0 4px 8px rgba(0,0,0,0.2) !important; }
	.style-icon.preview-rainbow { background: linear-gradient(135deg, #ff0000, #ff7f00, #ffff00, #00ff00, #0000ff, #4b0082, #9400d3) !important; }
	.style-icon.preview-sketch { background: var(--dbg) !important; border: 3px dashed #333 !important; color: var(--dt) !important; }
	.style-icon.preview-comic { background: #ffeb3b !important; border: 5px solid #000 !important; border-radius: 4px !important; color: #000 !important; }
	.style-icon.preview-watercolor { background: radial-gradient(circle, rgba(255,182,193,0.8) 0%, rgba(173,216,230,0.6) 100%) !important; backdrop-filter: blur(8px) !important; border: none !important; }

	.style-text {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.style-name {
		font-size: 12px;
		font-weight: 600;
		color: var(--dt);
	}

	.style-desc {
		font-size: 10px;
		color: var(--dt2);
		line-height: 1.3;
	}

	/* Advanced Options */
	.advanced-options {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.option-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		background: #f9f9f9;
		border-radius: 10px;
		border: 1px solid #e5e5e5;
	}

	.option-label {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.option-name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
	}

	.option-hint {
		font-size: 11px;
		color: var(--dt2);
	}

	.option-control {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.slider-modern {
		width: 160px;
		height: 6px;
		border-radius: 3px;
		background: #e5e5e5;
		outline: none;
		-webkit-appearance: none;
		cursor: pointer;
	}

	.slider-modern::-webkit-slider-thumb {
		-webkit-appearance: none;
		appearance: none;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: #0066FF;
		cursor: pointer;
		box-shadow: 0 2px 6px rgba(0, 102, 255, 0.3);
	}

	.slider-modern::-moz-range-thumb {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: #0066FF;
		cursor: pointer;
		border: none;
		box-shadow: 0 2px 6px rgba(0, 102, 255, 0.3);
	}

	.value-display {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		min-width: 48px;
		text-align: right;
	}

	/* Advanced Toggle Switch */
	.adv-toggle-switch {
		position: relative;
		display: inline-block;
		width: 48px;
		height: 26px;
		cursor: pointer;
	}

	.adv-toggle-switch input {
		opacity: 0;
		width: 0;
		height: 0;
	}

	.adv-toggle-slider {
		position: absolute;
		cursor: pointer;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background-color: #ccc;
		border-radius: 26px;
		transition: 0.3s;
	}

	.adv-toggle-slider:before {
		position: absolute;
		content: "";
		height: 20px;
		width: 20px;
		left: 3px;
		bottom: 3px;
		background-color: white;
		border-radius: 50%;
		transition: 0.3s;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
	}

	.adv-toggle-switch input:checked + .adv-toggle-slider {
		background-color: #0066FF;
	}

	.adv-toggle-switch input:checked + .adv-toggle-slider:before {
		transform: translateX(22px);
	}

	/* Library Grid */
	.library-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 8px;
	}

	.library-option {
		padding: 12px;
		border-radius: 8px;
		border: 2px solid #e5e5e5;
		background: var(--dbg);
		cursor: pointer;
		text-align: left;
		transition: all 0.15s ease;
	}

	.library-option:hover {
		border-color: #ccc;
		background: var(--dbg2);
	}

	.library-option.selected {
		border-color: #0077cc;
		background: #e8f4fc;
	}

	.library-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 4px;
	}

	.library-name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
	}

	.library-preview {
		font-size: 9px;
		font-weight: 500;
		font-family: monospace;
		color: #666;
		background: #f0f0f0;
		padding: 2px 6px;
		border-radius: 4px;
	}

	.library-option.selected .library-preview {
		background: #cce5f7;
		color: #0066aa;
	}

	.library-desc {
		font-size: 11px;
		color: var(--dt2);
		margin-bottom: 8px;
	}

	.stroke-preview {
		display: flex;
		justify-content: center;
		align-items: center;
		padding: 8px;
		background: var(--dbg2);
		border-radius: 6px;
		color: var(--dt);
	}

	.stroke-preview.stroke-phosphor svg {
		filter: drop-shadow(0 1px 2px rgba(0,0,0,0.2));
	}

	.stroke-preview.stroke-tabler {
		opacity: 0.7;
	}

	.library-option.selected .stroke-preview {
		background: #d8eef9;
	}

	.library-hint {
		font-size: 11px;
		color: #999;
		margin-top: 8px;
		text-align: center;
	}

	/* Options Toggles */
	.toggles {
		display: flex;
		flex-direction: column;
		gap: 12px;
		background: var(--dbg);
		border-radius: 8px;
		padding: 8px;
	}

	.toggle-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
	}

	.toggle-info {
		flex: 1;
	}

	.toggle-label {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt);
	}

	.toggle-desc {
		font-size: 11px;
		color: #999;
		margin-top: 2px;
	}

	.toggle-switch {
		position: relative;
		width: 44px;
		height: 24px;
		background: #ddd;
		border-radius: 12px;
		border: none;
		cursor: pointer;
		transition: background 0.2s ease;
	}

	.toggle-switch.active {
		background: #333;
	}

	.toggle-thumb {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 20px;
		height: 20px;
		background: white;
		border-radius: 50%;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
		transition: transform 0.2s ease;
	}

	.toggle-switch.active .toggle-thumb {
		transform: translateX(20px);
	}
</style>
