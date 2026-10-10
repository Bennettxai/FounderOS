<script lang="ts">
	import { desktopSettings, type BootAnimation } from '$lib/stores/desktopStore';
</script>

<!-- Boot Screen Customization -->
<div class="section">
	<label class="section-title">Boot Animation</label>
	<p class="section-subtitle">Choose the animation style for your boot screen</p>
	<div class="boot-anim-grid">
		{#each [
			{ id: 'terminal', name: 'Terminal', desc: 'Classic terminal text' },
			{ id: 'spinner', name: 'Spinner', desc: 'Circular loading' },
			{ id: 'progress', name: 'Progress', desc: 'Progress bar' },
			{ id: 'pulse', name: 'Pulse', desc: 'Breathing glow' },
			{ id: 'glitch', name: 'Glitch', desc: 'Cyberpunk effect' }
		] as anim}
			<button
				class="boot-anim-option"
				class:selected={$desktopSettings.bootScreen.animation === anim.id}
				onclick={() => desktopSettings.setBootScreen({ animation: anim.id as BootAnimation })}
			>
				<div class="boot-preview boot-{anim.id}">
					<div class="boot-preview-inner"></div>
				</div>
				<span class="boot-name">{anim.name}</span>
				<span class="boot-desc">{anim.desc}</span>
			</button>
		{/each}
	</div>
</div>

<div class="section">
	<label class="section-title">Boot Messages</label>
	<div class="toggle-row" style="padding: 0;">
		<div class="toggle-info">
			<div class="toggle-label">Show Boot Messages</div>
			<div class="toggle-desc">Display loading messages during boot</div>
		</div>
		<button
			onclick={() => desktopSettings.setBootScreen({
				messages: { ...$desktopSettings.bootScreen.messages, enabled: !$desktopSettings.bootScreen.messages.enabled }
			})}
			class="toggle-switch"
			class:active={$desktopSettings.bootScreen.messages.enabled}
			role="switch"
			aria-checked={$desktopSettings.bootScreen.messages.enabled}
		>
			<span class="toggle-thumb"></span>
		</button>
	</div>
</div>

<div class="section">
	<div class="section-header">
		<label class="section-title">Boot Duration</label>
		<span class="section-value">{$desktopSettings.bootScreen.duration}s</span>
	</div>
	<div class="slider-row">
		<span class="slider-label">Fast</span>
		<input
			type="range"
			min="1"
			max="10"
			step="0.5"
			value={$desktopSettings.bootScreen.duration}
			oninput={(e) => desktopSettings.setBootScreen({ duration: parseFloat((e.target as HTMLInputElement).value) })}
			class="slider"
		/>
		<span class="slider-label">Slow</span>
	</div>
</div>

<div class="section">
	<label class="section-title">Boot Colors</label>
	<div class="color-pickers">
		<div class="color-picker-row">
			<span class="color-label">Background</span>
			<div class="color-input-wrapper">
				<input
					type="color"
					value={$desktopSettings.bootScreen.colors.background}
					oninput={(e) => desktopSettings.setBootScreen({
						colors: { ...$desktopSettings.bootScreen.colors, background: (e.target as HTMLInputElement).value }
					})}
					class="color-input"
				/>
				<input
					type="text"
					value={$desktopSettings.bootScreen.colors.background}
					oninput={(e) => desktopSettings.setBootScreen({
						colors: { ...$desktopSettings.bootScreen.colors, background: (e.target as HTMLInputElement).value }
					})}
					class="color-text-input"
				/>
			</div>
		</div>
		<div class="color-picker-row">
			<span class="color-label">Text</span>
			<div class="color-input-wrapper">
				<input
					type="color"
					value={$desktopSettings.bootScreen.colors.text}
					oninput={(e) => desktopSettings.setBootScreen({
						colors: { ...$desktopSettings.bootScreen.colors, text: (e.target as HTMLInputElement).value }
					})}
					class="color-input"
				/>
				<input
					type="text"
					value={$desktopSettings.bootScreen.colors.text}
					oninput={(e) => desktopSettings.setBootScreen({
						colors: { ...$desktopSettings.bootScreen.colors, text: (e.target as HTMLInputElement).value }
					})}
					class="color-text-input"
				/>
			</div>
		</div>
		<div class="color-picker-row">
			<span class="color-label">Accent</span>
			<div class="color-input-wrapper">
				<input
					type="color"
					value={$desktopSettings.bootScreen.colors.accent}
					oninput={(e) => desktopSettings.setBootScreen({
						colors: { ...$desktopSettings.bootScreen.colors, accent: (e.target as HTMLInputElement).value }
					})}
					class="color-input"
				/>
				<input
					type="text"
					value={$desktopSettings.bootScreen.colors.accent}
					oninput={(e) => desktopSettings.setBootScreen({
						colors: { ...$desktopSettings.bootScreen.colors, accent: (e.target as HTMLInputElement).value }
					})}
					class="color-text-input"
				/>
			</div>
		</div>
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
		margin-bottom: 12px;
	}

	.section-header .section-title {
		margin-bottom: 0;
	}

	.section-value {
		font-size: 12px;
		color: #666;
	}

	.section-subtitle {
		font-size: 11px;
		color: #999;
		margin: -4px 0 8px 0;
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
		color: #333;
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

	/* Boot Settings Styles */
	.boot-anim-grid {
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 8px;
	}

	.boot-anim-option {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 12px 8px;
		background: white;
		border: 2px solid #e5e5e5;
		border-radius: 10px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.boot-anim-option:hover {
		border-color: #ccc;
	}

	.boot-anim-option.selected {
		border-color: #333;
		background: #f9f9f9;
	}

	.boot-preview {
		width: 48px;
		height: 48px;
		border-radius: 8px;
		background: #1a1a1a;
		overflow: hidden;
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.boot-preview-inner {
		width: 16px;
		height: 16px;
		background: #28a745;
		border-radius: 2px;
	}

	.boot-terminal .boot-preview-inner {
		width: 24px;
		height: 2px;
		background: #28a745;
		animation: blink 1s infinite;
	}

	.boot-spinner .boot-preview-inner {
		width: 20px;
		height: 20px;
		border: 2px solid #333;
		border-top-color: #28a745;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.boot-progress .boot-preview-inner {
		width: 32px;
		height: 4px;
		background: linear-gradient(90deg, #28a745 50%, #333 50%);
		border-radius: 2px;
	}

	.boot-pulse .boot-preview-inner {
		width: 16px;
		height: 16px;
		background: #28a745;
		border-radius: 50%;
		animation: pulse 1.5s ease-in-out infinite;
	}

	.boot-glitch .boot-preview-inner {
		width: 24px;
		height: 12px;
		background: #28a745;
		animation: glitch 0.5s infinite;
	}

	@keyframes blink {
		0%, 50% { opacity: 1; }
		51%, 100% { opacity: 0; }
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	@keyframes pulse {
		0%, 100% { transform: scale(1); opacity: 0.8; }
		50% { transform: scale(1.2); opacity: 1; }
	}

	@keyframes glitch {
		0% { transform: translate(0); }
		20% { transform: translate(-2px, 1px); }
		40% { transform: translate(2px, -1px); }
		60% { transform: translate(-1px, 2px); }
		80% { transform: translate(1px, -2px); }
		100% { transform: translate(0); }
	}

	.boot-name {
		font-size: 11px;
		font-weight: 600;
		color: #333;
	}

	.boot-desc {
		font-size: 10px;
		color: #999;
	}

	.color-pickers {
		display: flex;
		flex-direction: column;
		gap: 12px;
		background: white;
		border-radius: 8px;
		padding: 16px;
	}

	.color-picker-row {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.color-label {
		font-size: 13px;
		font-weight: 500;
		color: #555;
		width: 80px;
		flex-shrink: 0;
	}

	.color-input-wrapper {
		display: flex;
		align-items: center;
		gap: 8px;
		flex: 1;
	}

	.color-input {
		width: 40px;
		height: 40px;
		padding: 2px;
		border: 1px solid #ddd;
		border-radius: 8px;
		cursor: pointer;
		flex-shrink: 0;
	}

	.color-text-input {
		flex: 1;
		padding: 10px 12px;
		border: 1px solid #ddd;
		border-radius: 6px;
		font-size: 13px;
		font-family: 'SF Mono', Monaco, 'Fira Code', monospace;
		text-transform: uppercase;
		outline: none;
		transition: border-color 0.15s ease;
	}

	.color-text-input:focus {
		border-color: #333;
	}
</style>
