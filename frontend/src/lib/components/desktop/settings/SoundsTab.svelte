<script lang="ts">
	import { soundStore, builtInPacks, soundEventLabels, type SoundEvent } from '$lib/stores/soundStore';
</script>

<!-- Sound Settings -->
<div class="section">
	<div class="section-header">
		<label class="section-title">System Sounds</label>
		<button
			onclick={() => soundStore.setEnabled(!$soundStore.enabled)}
			class="toggle-switch"
			class:active={$soundStore.enabled}
			role="switch"
			aria-checked={$soundStore.enabled}
		>
			<span class="toggle-thumb"></span>
		</button>
	</div>
	<p class="section-subtitle">Enable sound effects for window events and interactions</p>
</div>

{#if $soundStore.enabled}
	<div class="section">
		<label class="section-title">Master Volume</label>
		<div class="slider-row">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; color: #999;">
				<path d="M11 5L6 9H2v6h4l5 4V5z"/>
			</svg>
			<input
				type="range"
				min="0"
				max="1"
				step="0.1"
				value={$soundStore.masterVolume}
				oninput={(e) => soundStore.setMasterVolume(parseFloat((e.target as HTMLInputElement).value))}
				class="slider"
			/>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; color: #999;">
				<path d="M11 5L6 9H2v6h4l5 4V5zM19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/>
			</svg>
		</div>
		<span class="volume-value">{Math.round($soundStore.masterVolume * 100)}%</span>
	</div>

	<div class="section">
		<label class="section-title">Sound Pack</label>
		<p class="section-subtitle">Choose a preset sound pack for your desktop</p>
		<div class="sound-pack-grid">
			{#each builtInPacks as pack}
				<button
					class="sound-pack-option"
					class:selected={$soundStore.currentPack === pack.id}
					onclick={() => {
						soundStore.setCurrentPack(pack.id);
						if (pack.id !== 'silent') soundStore.previewPack(pack.id);
					}}
				>
					<div class="pack-icon">
						{#if pack.id === 'silent'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M11 5L6 9H2v6h4l5 4V5zM23 9l-6 6M17 9l6 6"/>
							</svg>
						{:else if pack.id === 'classic'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M9 18V5l12-2v13"/>
								<circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
							</svg>
						{:else if pack.id === 'modern'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
								<path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
							</svg>
						{:else if pack.id === 'retro'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect x="2" y="6" width="20" height="12" rx="2"/>
								<circle cx="8" cy="12" r="2"/><circle cx="16" cy="12" r="2"/>
							</svg>
						{:else if pack.id === 'minimal'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="12" cy="12" r="1"/>
								<path d="M12 8v1M12 15v1M8 12h1M15 12h1"/>
							</svg>
						{:else if pack.id === 'bubbly'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="12" cy="12" r="5"/>
								<circle cx="6" cy="8" r="2"/>
								<circle cx="18" cy="16" r="3"/>
							</svg>
						{:else if pack.id === 'mechanical'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="12" cy="12" r="3"/>
								<path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
							</svg>
						{:else if pack.id === 'nature'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M12 22c4-4 8-7 8-12a8 8 0 1 0-16 0c0 5 4 8 8 12z"/>
								<path d="M12 12v5"/>
								<path d="M9 15l3-3 3 3"/>
							</svg>
						{:else if pack.id === 'scifi'}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M12 2L2 7l10 5 10-5-10-5z"/>
								<path d="M2 17l10 5 10-5"/>
								<path d="M2 12l10 5 10-5"/>
							</svg>
						{:else}
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707"/>
							</svg>
						{/if}
					</div>
					<div class="pack-info">
						<span class="pack-name">{pack.name}</span>
						<span class="pack-desc">{pack.description}</span>
					</div>
					{#if $soundStore.currentPack === pack.id}
						<div class="pack-check">
							<svg viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
							</svg>
						</div>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	<div class="section">
		<label class="section-title">Sound Events</label>
		<p class="section-subtitle">Enable or disable individual sound events</p>
		<div class="sound-events-list">
			{#each Object.entries(soundEventLabels) as [event, label]}
				{@const eventConfig = $soundStore.perEventSettings[event as SoundEvent]}
				{@const isEnabled = eventConfig?.enabled !== false}
				<div class="sound-event-row">
					<div class="event-info">
						<span class="event-label">{label}</span>
					</div>
					<div class="event-controls">
						<button
							class="preview-sound-btn"
							onclick={() => soundStore.playSound(event as SoundEvent)}
							title="Preview sound"
							aria-label="Preview {label} sound"
							disabled={!isEnabled}
						>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polygon points="5 3 19 12 5 21 5 3"/>
							</svg>
						</button>
						<button
							onclick={() => soundStore.setEventSettings(event as SoundEvent, { enabled: !isEnabled })}
							class="event-toggle"
							class:active={isEnabled}
							role="switch"
							aria-checked={isEnabled}
							title={isEnabled ? 'Disable sound' : 'Enable sound'}
						>
							<span class="toggle-thumb"></span>
						</button>
					</div>
				</div>
			{/each}
		</div>
	</div>
{/if}

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

	.volume-value {
		display: block;
		text-align: center;
		font-size: 12px;
		color: #666;
		margin-top: 8px;
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

	.sound-pack-grid {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.sound-pack-option {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 14px;
		background: white;
		border: 2px solid #e5e5e5;
		border-radius: 10px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: left;
	}

	.sound-pack-option:hover {
		border-color: #ccc;
	}

	.sound-pack-option.selected {
		border-color: #333;
		background: #f9f9f9;
	}

	.pack-icon {
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #f5f5f5;
		border-radius: 8px;
		flex-shrink: 0;
	}

	.sound-pack-option.selected .pack-icon {
		background: #333;
	}

	.pack-icon svg {
		width: 20px;
		height: 20px;
		color: #666;
	}

	.sound-pack-option.selected .pack-icon svg {
		color: white;
	}

	.pack-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.pack-name {
		font-size: 13px;
		font-weight: 600;
		color: #333;
	}

	.pack-desc {
		font-size: 11px;
		color: #999;
	}

	.pack-check {
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #28a745;
		border-radius: 50%;
		color: white;
	}

	.pack-check svg {
		width: 14px;
		height: 14px;
	}

	.sound-events-list {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 8px;
		background: white;
		border-radius: 8px;
		padding: 12px;
	}

	.sound-event-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 10px;
		background: #f9f9f9;
		border-radius: 6px;
	}

	.event-label {
		font-size: 12px;
		font-weight: 500;
		color: #555;
	}

	.preview-sound-btn {
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #333;
		border: none;
		border-radius: 50%;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.preview-sound-btn:hover {
		background: #555;
		transform: scale(1.1);
	}

	.preview-sound-btn svg {
		width: 12px;
		height: 12px;
		color: white;
		margin-left: 2px;
	}

	.preview-sound-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
		transform: none;
	}

	.preview-sound-btn:disabled:hover {
		background: #333;
		transform: none;
	}

	.event-info {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.event-controls {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.event-toggle {
		width: 36px;
		height: 20px;
		background: #ccc;
		border: none;
		border-radius: 10px;
		position: relative;
		cursor: pointer;
		transition: background 0.2s ease;
	}

	.event-toggle.active {
		background: #333;
	}

	.event-toggle .toggle-thumb {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 16px;
		height: 16px;
		background: white;
		border-radius: 50%;
		transition: transform 0.2s ease;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
	}

	.event-toggle.active .toggle-thumb {
		transform: translateX(16px);
	}
</style>
