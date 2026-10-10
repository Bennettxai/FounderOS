<script lang="ts">
	import {
		desktopSettings,
		type AnimatedBackgroundEffect,
		type AnimatedBackgroundIntensity
	} from '$lib/stores/desktopStore';

	interface Props {
		previewEffect: AnimatedBackgroundEffect | null;
		previewIntensity: AnimatedBackgroundIntensity | null;
		previewSpeed: number | null;
		hasUnsavedEffectChanges: boolean;
		effectiveEffect: AnimatedBackgroundEffect;
		effectiveIntensity: AnimatedBackgroundIntensity;
		effectiveSpeed: number;
		onPreviewEffect: (effect: AnimatedBackgroundEffect) => void;
		onPreviewIntensity: (intensity: AnimatedBackgroundIntensity) => void;
		onPreviewSpeed: (speed: number) => void;
		onApply: () => void;
		onCancel: () => void;
	}

	let {
		previewEffect,
		hasUnsavedEffectChanges,
		effectiveEffect,
		effectiveIntensity,
		effectiveSpeed,
		onPreviewEffect,
		onPreviewIntensity,
		onPreviewSpeed,
		onApply,
		onCancel
	}: Props = $props();

	let effectBasicScroll: HTMLDivElement | undefined = $state(undefined);
	let effectNatureScroll: HTMLDivElement | undefined = $state(undefined);
	let effectTechScroll: HTMLDivElement | undefined = $state(undefined);

	function scrollCarousel(container: HTMLDivElement | undefined, direction: 'left' | 'right') {
		if (!container) return;
		const scrollAmount = 200;
		container.scrollBy({
			left: direction === 'right' ? scrollAmount : -scrollAmount,
			behavior: 'smooth'
		});
	}
</script>

<!-- Animated Background Effects -->
<div class="section">
	<div class="section-header-row">
		<div>
			<label class="section-title">Animated Background</label>
			<p class="section-subtitle">Add subtle animated effects to your desktop background</p>
		</div>
		{#if hasUnsavedEffectChanges}
			<div class="unsaved-indicator">
				<span class="unsaved-dot"></span>
				<span>Unsaved changes</span>
			</div>
		{/if}
	</div>

	<!-- Basic Effects -->
	<div class="effect-category">
		<span class="effect-category-label">Basic</span>
		<div class="carousel-container">
			<button class="carousel-btn left" onclick={() => scrollCarousel(effectBasicScroll, 'left')} aria-label="Scroll left">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M15 18l-6-6 6-6"/>
				</svg>
			</button>
			<div class="carousel-scroll" bind:this={effectBasicScroll}>
				<div class="effect-carousel-grid">
					{#each [
						{ id: 'none', name: 'None', desc: 'No animation' },
						{ id: 'particles', name: 'Particles', desc: 'Floating particles' },
						{ id: 'gradient', name: 'Gradient', desc: 'Flowing colors' },
						{ id: 'pulse', name: 'Pulse', desc: 'Gentle pulsing' },
						{ id: 'ripples', name: 'Ripples', desc: 'Water ripples' },
						{ id: 'dots', name: 'Dots', desc: 'Pulsing dot grid' },
						{ id: 'floatingShapes', name: 'Shapes', desc: 'Floating shapes' },
						{ id: 'smoke', name: 'Smoke', desc: 'Rising smoke' }
					] as effect}
						<button
							class="effect-card"
							class:selected={effectiveEffect === effect.id}
							class:previewing={previewEffect === effect.id && previewEffect !== $desktopSettings.animatedBackground.effect}
							onclick={() => onPreviewEffect(effect.id as AnimatedBackgroundEffect)}
						>
							<div class="effect-card-preview anim-{effect.id}"></div>
							<span class="effect-card-name">{effect.name}</span>
						</button>
					{/each}
				</div>
			</div>
			<button class="carousel-btn right" onclick={() => scrollCarousel(effectBasicScroll, 'right')} aria-label="Scroll right">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 18l6-6-6-6"/>
				</svg>
			</button>
		</div>
	</div>

	<!-- Nature Effects -->
	<div class="effect-category">
		<span class="effect-category-label">Nature</span>
		<div class="carousel-container">
			<button class="carousel-btn left" onclick={() => scrollCarousel(effectNatureScroll, 'left')} aria-label="Scroll left">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M15 18l-6-6 6-6"/>
				</svg>
			</button>
			<div class="carousel-scroll" bind:this={effectNatureScroll}>
				<div class="effect-carousel-grid">
					{#each [
						{ id: 'aurora', name: 'Aurora', desc: 'Northern lights' },
						{ id: 'starfield', name: 'Starfield', desc: 'Twinkling stars' },
						{ id: 'waves', name: 'Waves', desc: 'Flowing waves' },
						{ id: 'bubbles', name: 'Bubbles', desc: 'Floating bubbles' },
						{ id: 'fireflies', name: 'Fireflies', desc: 'Glowing fireflies' },
						{ id: 'rain', name: 'Rain', desc: 'Falling rain' },
						{ id: 'snow', name: 'Snow', desc: 'Gentle snowfall' },
						{ id: 'nebula', name: 'Nebula', desc: 'Space clouds' }
					] as effect}
						<button
							class="effect-card"
							class:selected={effectiveEffect === effect.id}
							class:previewing={previewEffect === effect.id && previewEffect !== $desktopSettings.animatedBackground.effect}
							onclick={() => onPreviewEffect(effect.id as AnimatedBackgroundEffect)}
						>
							<div class="effect-card-preview anim-{effect.id}"></div>
							<span class="effect-card-name">{effect.name}</span>
						</button>
					{/each}
				</div>
			</div>
			<button class="carousel-btn right" onclick={() => scrollCarousel(effectNatureScroll, 'right')} aria-label="Scroll right">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 18l6-6-6-6"/>
				</svg>
			</button>
		</div>
	</div>

	<!-- Tech Effects -->
	<div class="effect-category">
		<span class="effect-category-label">Tech</span>
		<div class="carousel-container">
			<button class="carousel-btn left" onclick={() => scrollCarousel(effectTechScroll, 'left')} aria-label="Scroll left">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M15 18l-6-6 6-6"/>
				</svg>
			</button>
			<div class="carousel-scroll" bind:this={effectTechScroll}>
				<div class="effect-carousel-grid">
					{#each [
						{ id: 'matrix', name: 'Matrix', desc: 'Digital rain' },
						{ id: 'geometric', name: 'Geometric', desc: 'Floating shapes' },
						{ id: 'circuit', name: 'Circuit', desc: 'Tech circuits' },
						{ id: 'confetti', name: 'Confetti', desc: 'Celebration' },
						{ id: 'scanlines', name: 'Scanlines', desc: 'CRT scanlines' },
						{ id: 'grid', name: 'Grid', desc: 'Neon grid' },
						{ id: 'warp', name: 'Warp', desc: 'Star warp speed' },
						{ id: 'hexgrid', name: 'Hexgrid', desc: 'Honeycomb' },
						{ id: 'binary', name: 'Binary', desc: 'Falling 0s and 1s' }
					] as effect}
						<button
							class="effect-card"
							class:selected={effectiveEffect === effect.id}
							class:previewing={previewEffect === effect.id && previewEffect !== $desktopSettings.animatedBackground.effect}
							onclick={() => onPreviewEffect(effect.id as AnimatedBackgroundEffect)}
						>
							<div class="effect-card-preview anim-{effect.id}"></div>
							<span class="effect-card-name">{effect.name}</span>
						</button>
					{/each}
				</div>
			</div>
			<button class="carousel-btn right" onclick={() => scrollCarousel(effectTechScroll, 'right')} aria-label="Scroll right">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 18l6-6-6-6"/>
				</svg>
			</button>
		</div>
	</div>
</div>

{#if effectiveEffect !== 'none'}
	<div class="section">
		<label class="section-title">Effect Settings</label>
		<div class="settings-row-group">
			<div class="settings-row">
				<div class="settings-row-label">
					<span class="settings-label-text">Intensity</span>
					<span class="settings-label-desc">How visible the effect appears</span>
				</div>
				<select
					class="settings-select"
					value={effectiveIntensity}
					onchange={(e) => onPreviewIntensity(e.currentTarget.value as AnimatedBackgroundIntensity)}
				>
					<option value="subtle">Subtle</option>
					<option value="medium">Medium</option>
					<option value="high">High</option>
				</select>
			</div>

			<div class="settings-row">
				<div class="settings-row-label">
					<span class="settings-label-text">Animation Speed</span>
					<span class="settings-label-desc">{effectiveSpeed}x speed</span>
				</div>
				<div class="slider-compact">
					<input
						type="range"
						min="0.5"
						max="2"
						step="0.1"
						value={effectiveSpeed}
						oninput={(e) => onPreviewSpeed(parseFloat((e.target as HTMLInputElement).value))}
						class="slider-input"
					/>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Apply/Cancel buttons for effect changes -->
{#if hasUnsavedEffectChanges}
	<div class="effect-action-bar">
		<button class="effect-cancel-btn" onclick={onCancel}>
			Cancel
		</button>
		<button class="effect-apply-btn" onclick={onApply}>
			Apply Changes
		</button>
	</div>
{/if}

<style>
	.section {
		margin-bottom: 24px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: #333;
		display: block;
		margin-bottom: 12px;
	}

	.section-subtitle {
		font-size: 11px;
		color: #999;
		margin: -4px 0 8px 0;
	}

	.section-header-row {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 12px;
	}

	.unsaved-indicator {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		background: #FEF3C7;
		border-radius: 12px;
		font-size: 11px;
		font-weight: 500;
		color: #92400E;
	}

	.unsaved-dot {
		width: 6px;
		height: 6px;
		background: #F59E0B;
		border-radius: 50%;
		animation: pulse-dot 1.5s ease-in-out infinite;
	}

	@keyframes pulse-dot {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.6; transform: scale(0.9); }
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
		background: white;
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

	/* Effect Category Carousel */
	.effect-category {
		margin-bottom: 16px;
	}

	.effect-category-label {
		display: block;
		font-size: 11px;
		font-weight: 600;
		color: #666;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin-bottom: 8px;
	}

	.effect-carousel-grid {
		display: flex;
		gap: 12px;
		padding: 4px 0;
	}

	.effect-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 12px;
		min-width: 100px;
		background: #fafafa;
		border: 2px solid transparent;
		border-radius: 12px;
		cursor: pointer;
		transition: all 0.2s ease;
		flex-shrink: 0;
	}

	.effect-card:hover {
		background: #f0f0f0;
		transform: translateY(-2px);
	}

	.effect-card.selected {
		background: #e8f4fc;
		border-color: #0077cc;
	}

	.effect-card.previewing:not(.selected) {
		background: var(--bos-status-warning-bg);
		border-color: #ffaa00;
	}

	.effect-card-preview {
		width: 64px;
		height: 40px;
		border-radius: 8px;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		position: relative;
		overflow: hidden;
	}

	/* Effect preview animations */
	.effect-card-preview.anim-none {
		background: #f5f5f5;
	}

	.effect-card-preview.anim-particles::before {
		content: '';
		position: absolute;
		width: 4px;
		height: 4px;
		background: rgba(255,255,255,0.8);
		border-radius: 50%;
		top: 30%;
		left: 20%;
		box-shadow:
			20px 10px 0 rgba(255,255,255,0.6),
			40px -5px 0 rgba(255,255,255,0.7),
			10px 20px 0 rgba(255,255,255,0.5);
		animation: float 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-gradient {
		background: linear-gradient(135deg, #667eea, #764ba2, #f093fb, #f5576c);
		background-size: 300% 300%;
		animation: gradientShift 3s ease infinite;
	}

	.effect-card-preview.anim-aurora {
		background: linear-gradient(to bottom, #0a0a1f, #1a1a3f);
	}

	.effect-card-preview.anim-aurora::before {
		content: '';
		position: absolute;
		inset: 0;
		background: linear-gradient(45deg,
			transparent 20%,
			rgba(0,255,127,0.3) 40%,
			rgba(0,191,255,0.3) 60%,
			transparent 80%);
		animation: aurora 3s ease-in-out infinite;
	}

	.effect-card-preview.anim-starfield {
		background: #0a0a1f;
	}

	.effect-card-preview.anim-starfield::before {
		content: '';
		position: absolute;
		width: 2px;
		height: 2px;
		background: white;
		border-radius: 50%;
		top: 20%;
		left: 30%;
		box-shadow:
			20px 15px 0 rgba(255,255,255,0.8),
			10px 25px 0 rgba(255,255,255,0.6),
			35px 8px 0 rgba(255,255,255,0.9),
			45px 22px 0 rgba(255,255,255,0.7);
		animation: twinkle 1.5s ease-in-out infinite;
	}

	.effect-card-preview.anim-waves {
		background: linear-gradient(180deg, #1a5276 0%, #2980b9 100%);
	}

	.effect-card-preview.anim-waves::before {
		content: '';
		position: absolute;
		bottom: 0;
		left: -50%;
		width: 200%;
		height: 60%;
		background: rgba(255,255,255,0.1);
		border-radius: 50% 50% 0 0;
		animation: wave 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-bubbles {
		background: linear-gradient(180deg, #2193b0 0%, #6dd5ed 100%);
	}

	.effect-card-preview.anim-bubbles::before {
		content: '';
		position: absolute;
		width: 8px;
		height: 8px;
		background: rgba(255,255,255,0.4);
		border-radius: 50%;
		bottom: 5px;
		left: 25%;
		animation: bubble 2s ease-in-out infinite;
		box-shadow:
			15px 5px 0 5px rgba(255,255,255,0.3),
			30px 10px 0 3px rgba(255,255,255,0.5);
	}

	.effect-card-preview.anim-matrix {
		background: #000;
	}

	.effect-card-preview.anim-matrix::before {
		content: '01';
		position: absolute;
		color: #00ff00;
		font-size: 10px;
		font-family: monospace;
		top: 5px;
		left: 10px;
		text-shadow:
			20px 10px 0 #00ff00,
			10px 20px 0 #00aa00,
			30px 5px 0 #00dd00;
		animation: matrixFall 1s linear infinite;
		opacity: 0.8;
	}

	.effect-card-preview.anim-geometric {
		background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
	}

	.effect-card-preview.anim-geometric::before {
		content: '';
		position: absolute;
		width: 0;
		height: 0;
		border-left: 12px solid transparent;
		border-right: 12px solid transparent;
		border-bottom: 20px solid rgba(255,255,255,0.2);
		top: 10px;
		left: 20px;
		animation: geoFloat 3s ease-in-out infinite;
	}

	.effect-card-preview.anim-pulse {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		animation: pulseEffect 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-ripples {
		background: linear-gradient(180deg, #1a5276 0%, #2980b9 100%);
	}

	.effect-card-preview.anim-ripples::before {
		content: '';
		position: absolute;
		width: 20px;
		height: 20px;
		border: 2px solid rgba(255,255,255,0.3);
		border-radius: 50%;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		animation: ripple 2s ease-out infinite;
	}

	.effect-card-preview.anim-fireflies {
		background: linear-gradient(180deg, #1a1a2e 0%, #0f0f23 100%);
	}

	.effect-card-preview.anim-fireflies::before {
		content: '';
		position: absolute;
		width: 4px;
		height: 4px;
		background: var(--bos-status-warning);
		border-radius: 50%;
		top: 30%;
		left: 25%;
		box-shadow:
			25px 10px 0 #ffff66,
			10px -8px 0 #ffffaa,
			40px 15px 0 var(--bos-status-warning);
		animation: fireflyGlow 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-rain {
		background: linear-gradient(180deg, #4a5568 0%, #2d3748 100%);
	}

	.effect-card-preview.anim-rain::before {
		content: '';
		position: absolute;
		width: 1px;
		height: 8px;
		background: rgba(255,255,255,0.4);
		top: 0;
		left: 20%;
		box-shadow:
			15px 5px 0 rgba(255,255,255,0.3),
			30px -3px 0 rgba(255,255,255,0.5),
			45px 8px 0 rgba(255,255,255,0.4);
		animation: rainFall 0.8s linear infinite;
	}

	.effect-card-preview.anim-snow {
		background: linear-gradient(180deg, #a0aec0 0%, #718096 100%);
	}

	.effect-card-preview.anim-snow::before {
		content: '';
		position: absolute;
		width: 4px;
		height: 4px;
		background: white;
		border-radius: 50%;
		top: 5px;
		left: 20%;
		box-shadow:
			20px 8px 0 white,
			40px 3px 0 white,
			10px 15px 0 white,
			35px 20px 0 white;
		animation: snowFall 3s linear infinite;
	}

	.effect-card-preview.anim-nebula {
		background: linear-gradient(135deg, #0a0a1f 0%, #1a0a2e 50%, #0f1a2e 100%);
	}

	.effect-card-preview.anim-nebula::before {
		content: '';
		position: absolute;
		inset: 0;
		background: radial-gradient(ellipse at 30% 50%, rgba(138,43,226,0.4) 0%, transparent 50%),
					radial-gradient(ellipse at 70% 60%, rgba(0,191,255,0.3) 0%, transparent 40%);
		animation: nebulaShift 4s ease-in-out infinite;
	}

	.effect-card-preview.anim-circuit {
		background: #0a1628;
	}

	.effect-card-preview.anim-circuit::before {
		content: '';
		position: absolute;
		width: 100%;
		height: 100%;
		background:
			linear-gradient(90deg, transparent 45%, rgba(0,255,136,0.3) 50%, transparent 55%),
			linear-gradient(0deg, transparent 45%, rgba(0,255,136,0.3) 50%, transparent 55%);
		background-size: 20px 20px;
		animation: circuitPulse 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-confetti {
		background: linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%);
	}

	.effect-card-preview.anim-confetti::before {
		content: '';
		position: absolute;
		width: 6px;
		height: 6px;
		background: #ff6b6b;
		top: 10%;
		left: 20%;
		box-shadow:
			15px 5px 0 #4ecdc4,
			30px 10px 0 #ffe66d,
			10px 20px 0 #95e1d3,
			40px 15px 0 #f38181;
		animation: confettiFall 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-dots {
		background: #f0f4f8;
	}

	.effect-card-preview.anim-dots::before {
		content: '';
		position: absolute;
		inset: 0;
		background:
			radial-gradient(circle at 20% 30%, #667eea 3px, transparent 3px),
			radial-gradient(circle at 50% 50%, #667eea 3px, transparent 3px),
			radial-gradient(circle at 80% 70%, #667eea 3px, transparent 3px),
			radial-gradient(circle at 35% 80%, #667eea 3px, transparent 3px),
			radial-gradient(circle at 65% 20%, #667eea 3px, transparent 3px);
		animation: dotPulse 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-floatingShapes {
		background: linear-gradient(135deg, #fef9f3 0%, #f0e6f6 100%);
	}

	.effect-card-preview.anim-floatingShapes::before {
		content: '';
		position: absolute;
		width: 12px;
		height: 12px;
		background: transparent;
		border: 2px solid rgba(102,126,234,0.4);
		top: 20%;
		left: 25%;
		transform: rotate(45deg);
		box-shadow:
			25px 15px 0 0 rgba(118,75,162,0.3),
			10px 25px 0 0 rgba(102,126,234,0.3);
		animation: shapeFloat 3s ease-in-out infinite;
	}

	.effect-card-preview.anim-smoke {
		background: linear-gradient(180deg, #1a1a2e 0%, #2d2d44 100%);
	}

	.effect-card-preview.anim-smoke::before {
		content: '';
		position: absolute;
		width: 100%;
		height: 100%;
		background:
			radial-gradient(ellipse at 30% 90%, rgba(150,150,150,0.4) 0%, transparent 40%),
			radial-gradient(ellipse at 60% 85%, rgba(120,120,120,0.3) 0%, transparent 35%),
			radial-gradient(ellipse at 45% 80%, rgba(100,100,100,0.2) 0%, transparent 30%);
		animation: smokeRise 3s ease-out infinite;
	}

	.effect-card-preview.anim-scanlines {
		background: #0a0a0a;
	}

	.effect-card-preview.anim-scanlines::before {
		content: '';
		position: absolute;
		inset: 0;
		background: repeating-linear-gradient(
			0deg,
			transparent,
			transparent 2px,
			rgba(0,255,0,0.1) 2px,
			rgba(0,255,0,0.1) 4px
		);
		animation: scanlineMove 0.1s linear infinite;
	}

	.effect-card-preview.anim-scanlines::after {
		content: '';
		position: absolute;
		width: 100%;
		height: 4px;
		background: linear-gradient(90deg, transparent, rgba(0,255,0,0.4), transparent);
		animation: scanlineSweep 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-grid {
		background: #0a0a1f;
	}

	.effect-card-preview.anim-grid::before {
		content: '';
		position: absolute;
		inset: 0;
		background:
			linear-gradient(90deg, rgba(59,130,246,0.2) 1px, transparent 1px),
			linear-gradient(0deg, rgba(59,130,246,0.2) 1px, transparent 1px);
		background-size: 10px 10px;
		animation: gridPulse 2s ease-in-out infinite;
	}

	.effect-card-preview.anim-warp {
		background: radial-gradient(ellipse at center, #0f172a 0%, #000000 100%);
	}

	.effect-card-preview.anim-warp::before {
		content: '';
		position: absolute;
		width: 2px;
		height: 2px;
		background: white;
		top: 50%;
		left: 50%;
		box-shadow:
			10px -5px 0 white,
			-8px 10px 0 white,
			15px 8px 0 white,
			-12px -8px 0 white,
			5px 12px 0 white;
		animation: warpSpeed 0.5s linear infinite;
	}

	.effect-card-preview.anim-hexgrid {
		background: #0a0a1f;
	}

	.effect-card-preview.anim-hexgrid::before {
		content: '';
		position: absolute;
		inset: 0;
		background:
			conic-gradient(from 30deg at 25% 33%, transparent 60deg, rgba(102,126,234,0.3) 60deg, rgba(102,126,234,0.3) 120deg, transparent 120deg),
			conic-gradient(from 30deg at 75% 33%, transparent 60deg, rgba(102,126,234,0.3) 60deg, rgba(102,126,234,0.3) 120deg, transparent 120deg),
			conic-gradient(from 30deg at 50% 75%, transparent 60deg, rgba(102,126,234,0.3) 60deg, rgba(102,126,234,0.3) 120deg, transparent 120deg);
		animation: hexPulse 3s ease-in-out infinite;
	}

	.effect-card-preview.anim-binary {
		background: #000000;
	}

	.effect-card-preview.anim-binary::before {
		content: '10110100';
		position: absolute;
		font-family: monospace;
		font-size: 8px;
		color: #00ff00;
		top: 5%;
		left: 10%;
		text-shadow: 0 15px 0 rgba(0,255,0,0.6), 0 30px 0 rgba(0,255,0,0.3);
		animation: binaryFall 2s linear infinite;
	}

	.effect-card-name {
		font-size: 12px;
		font-weight: 500;
		color: #333;
	}

	.effect-card.selected .effect-card-name {
		color: #0077cc;
	}

	/* Effect preview keyframes */
	@keyframes float {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(-5px); }
	}

	@keyframes gradientShift {
		0% { background-position: 0% 50%; }
		50% { background-position: 100% 50%; }
		100% { background-position: 0% 50%; }
	}

	@keyframes aurora {
		0%, 100% { transform: translateX(-20%); opacity: 0.5; }
		50% { transform: translateX(20%); opacity: 0.8; }
	}

	@keyframes twinkle {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}

	@keyframes wave {
		0%, 100% { transform: translateX(0) translateY(0); }
		50% { transform: translateX(5px) translateY(-3px); }
	}

	@keyframes bubble {
		0% { transform: translateY(0) scale(1); opacity: 0.4; }
		100% { transform: translateY(-30px) scale(0.5); opacity: 0; }
	}

	@keyframes matrixFall {
		0% { transform: translateY(-5px); opacity: 0; }
		50% { opacity: 0.8; }
		100% { transform: translateY(25px); opacity: 0; }
	}

	@keyframes geoFloat {
		0%, 100% { transform: translateY(0) rotate(0deg); }
		50% { transform: translateY(-5px) rotate(15deg); }
	}

	@keyframes pulseEffect {
		0%, 100% { transform: scale(1); opacity: 1; }
		50% { transform: scale(1.05); opacity: 0.8; }
	}

	@keyframes ripple {
		0% { transform: translate(-50%, -50%) scale(0.5); opacity: 0.8; }
		100% { transform: translate(-50%, -50%) scale(2); opacity: 0; }
	}

	@keyframes fireflyGlow {
		0%, 100% { opacity: 0.3; }
		50% { opacity: 1; }
	}

	@keyframes rainFall {
		0% { transform: translateY(-10px); }
		100% { transform: translateY(40px); }
	}

	@keyframes snowFall {
		0% { transform: translateY(0) translateX(0); }
		50% { transform: translateY(15px) translateX(3px); }
		100% { transform: translateY(30px) translateX(0); }
	}

	@keyframes nebulaShift {
		0%, 100% { opacity: 0.6; transform: scale(1); }
		50% { opacity: 1; transform: scale(1.1); }
	}

	@keyframes circuitPulse {
		0%, 100% { opacity: 0.3; }
		50% { opacity: 0.8; }
	}

	@keyframes confettiFall {
		0% { transform: translateY(0) rotate(0deg); }
		100% { transform: translateY(25px) rotate(180deg); }
	}

	@keyframes dotPulse {
		0%, 100% { opacity: 0.5; transform: scale(1); }
		50% { opacity: 1; transform: scale(1.2); }
	}

	@keyframes shapeFloat {
		0%, 100% { transform: rotate(45deg) translateY(0); }
		50% { transform: rotate(50deg) translateY(-5px); }
	}

	@keyframes smokeRise {
		0% { transform: translateY(0); opacity: 0.5; }
		100% { transform: translateY(-20px); opacity: 0; }
	}

	@keyframes scanlineMove {
		0% { transform: translateY(0); }
		100% { transform: translateY(4px); }
	}

	@keyframes scanlineSweep {
		0% { top: 0; }
		100% { top: 100%; }
	}

	@keyframes gridPulse {
		0%, 100% { opacity: 0.5; }
		50% { opacity: 1; }
	}

	@keyframes warpSpeed {
		0% { transform: scale(0.5); opacity: 0; }
		50% { transform: scale(1); opacity: 1; }
		100% { transform: scale(2); opacity: 0; }
	}

	@keyframes hexPulse {
		0%, 100% { opacity: 0.4; }
		50% { opacity: 0.8; }
	}

	@keyframes binaryFall {
		0% { transform: translateY(-10px); opacity: 0; }
		10% { opacity: 1; }
		90% { opacity: 1; }
		100% { transform: translateY(40px); opacity: 0; }
	}

	/* Settings Row Styles */
	.settings-row-group {
		display: flex;
		flex-direction: column;
		gap: 0;
		background: #fafafa;
		border: 1px solid #e5e5e5;
		border-radius: 8px;
		overflow: hidden;
	}

	.settings-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid #e5e5e5;
	}

	.settings-row:last-child {
		border-bottom: none;
	}

	.settings-row-label {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.settings-label-text {
		font-size: 13px;
		font-weight: 500;
		color: #333;
	}

	.settings-label-desc {
		font-size: 11px;
		color: #888;
	}

	.settings-select {
		padding: 8px 32px 8px 12px;
		background: white;
		border: 1px solid #e0e0e0;
		border-radius: 6px;
		font-size: 13px;
		color: #333;
		cursor: pointer;
		min-width: 120px;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23666' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 10px center;
		transition: border-color 0.15s ease, box-shadow 0.15s ease;
	}

	.settings-select:hover {
		border-color: #999;
	}

	.settings-select:focus {
		outline: none;
		border-color: #666;
		box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.08);
	}

	/* Compact slider for settings rows */
	.slider-compact {
		width: 120px;
	}

	.slider-input {
		width: 100%;
		height: 4px;
		background: #e5e5e5;
		border-radius: 2px;
		appearance: none;
		cursor: pointer;
	}

	.slider-input::-webkit-slider-thumb {
		appearance: none;
		width: 14px;
		height: 14px;
		background: #333;
		border-radius: 50%;
		cursor: pointer;
	}

	.slider-input::-webkit-slider-thumb:hover {
		background: #555;
	}

	.effect-action-bar {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding: 16px 0;
		margin-top: 16px;
		border-top: 1px solid #e5e5e5;
		position: sticky;
		bottom: 0;
		background: linear-gradient(180deg, transparent 0%, #f9f9f9 20%);
		padding-bottom: 8px;
	}

	.effect-cancel-btn {
		padding: 10px 20px;
		background: #f5f5f5;
		border: 1px solid #e0e0e0;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 500;
		color: #666;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.effect-cancel-btn:hover {
		background: #eee;
		color: #333;
	}

	.effect-apply-btn {
		padding: 10px 24px;
		background: #333;
		border: none;
		border-radius: 8px;
		font-size: 13px;
		font-weight: 600;
		color: white;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.effect-apply-btn:hover {
		background: #444;
	}
</style>
