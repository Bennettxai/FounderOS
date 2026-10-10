<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import {
		type AnimationEffect,
		type AnimationIntensity,
		type AnimationState,
		createAnimationState,
		intensityConfig,
		initParticles,
		initStars,
		initMatrix,
		initBubbles,
		initGeometric,
		initFireflies,
		initRain,
		initSnow,
		initConfetti,
		initRipples,
		initCircuit,
		initDots,
		initFloatingShapes,
		initSmoke,
		initGrid,
		initWarp,
		initHexgrid,
		drawParticles,
		drawGradient,
		drawAurora,
		drawStarfield,
		drawMatrix,
		drawWaves,
		drawBubbles,
		drawGeometric,
		drawFireflies,
		drawRain,
		drawSnow,
		drawConfetti,
		drawPulse,
		drawRipples,
		drawNebula,
		drawCircuit,
		drawDots,
		drawFloatingShapes,
		drawSmoke,
		drawScanlines,
		drawGrid,
		drawWarp,
		drawHexgrid,
		drawBinary
	} from './animationEffects';

	// Re-export types for consumers that import from this file
	export type { AnimationEffect, AnimationIntensity };

	interface Props {
		effectType?: AnimationEffect;
		intensity?: AnimationIntensity;
		colors?: string[];
		speed?: number;
	}

	let {
		effectType = 'none',
		intensity = 'subtle',
		colors = ['#667eea', '#764ba2'],
		speed = 1
	}: Props = $props();

	let canvas: HTMLCanvasElement;
	let ctx: CanvasRenderingContext2D | null = null;
	let animationId: number;
	let state: AnimationState = createAnimationState();

	function initCanvas() {
		if (!browser || !canvas) return;
		ctx = canvas.getContext('2d');
		resizeCanvas();
	}

	function resizeCanvas() {
		if (!canvas) return;
		canvas.width = window.innerWidth;
		canvas.height = window.innerHeight;
		initEffect();
	}

	function initEffect() {
		const config = intensityConfig[intensity];

		switch (effectType) {
			case 'particles':
				initParticles(state, canvas, config.particleCount, colors, speed);
				break;
			case 'starfield':
				initStars(state, canvas, config.particleCount * 2);
				break;
			case 'matrix':
				initMatrix(state, canvas, config.particleCount);
				break;
			case 'bubbles':
				initBubbles(state, canvas, config.particleCount, colors);
				break;
			case 'geometric':
				initGeometric(state, canvas, Math.floor(config.particleCount / 3), colors);
				break;
			case 'fireflies':
				initFireflies(state, canvas, Math.floor(config.particleCount / 2), colors);
				break;
			case 'rain':
				initRain(state, canvas, config.particleCount * 3);
				break;
			case 'snow':
				initSnow(state, canvas, config.particleCount * 2);
				break;
			case 'confetti':
				initConfetti(state, canvas, config.particleCount);
				break;
			case 'ripples':
				initRipples(state, canvas, Math.floor(config.particleCount / 5), colors);
				break;
			case 'circuit':
				initCircuit(state, canvas, Math.floor(config.particleCount / 2));
				break;
			case 'dots':
				initDots(state, canvas, config.particleCount);
				break;
			case 'floatingShapes':
				initFloatingShapes(state, canvas, Math.floor(config.particleCount / 2), colors);
				break;
			case 'smoke':
				initSmoke(state, canvas, Math.floor(config.particleCount / 2));
				break;
			case 'scanlines':
				// Scanlines don't need particle init
				break;
			case 'grid':
				initGrid(state, canvas, Math.floor(config.particleCount / 4));
				break;
			case 'warp':
				initWarp(state, canvas, config.particleCount * 2);
				break;
			case 'hexgrid':
				initHexgrid(state, canvas);
				break;
			case 'binary':
				initMatrix(state, canvas, Math.floor(config.particleCount / 2));
				break;
		}
	}

	function animate(time: number) {
		if (!ctx || effectType === 'none') return;

		// Don't clear for matrix (it has its own fade)
		if (effectType !== 'matrix') {
			ctx.clearRect(0, 0, canvas.width, canvas.height);
		}

		switch (effectType) {
			case 'particles':
				drawParticles(ctx, state, canvas, intensity, speed);
				break;
			case 'gradient':
				drawGradient(ctx, canvas, intensity, colors, speed, time);
				break;
			case 'aurora':
				drawAurora(ctx, canvas, intensity, colors, speed, time);
				break;
			case 'starfield':
				// signature: (ctx, state, intensity, speed, time) — no canvas
				drawStarfield(ctx, state, intensity, speed, time);
				break;
			case 'matrix':
				// signature: (ctx, state, canvas, intensity, speed) — no colors, no time
				drawMatrix(ctx, state, canvas, intensity, speed);
				break;
			case 'waves':
				drawWaves(ctx, canvas, intensity, colors, speed, time);
				break;
			case 'bubbles':
				drawBubbles(ctx, state, canvas, intensity, speed);
				break;
			case 'geometric':
				// signature: (ctx, state, canvas, intensity, speed) — no time
				drawGeometric(ctx, state, canvas, intensity, speed);
				break;
			case 'fireflies':
				// signature: (ctx, state, canvas, intensity, speed) — no time
				drawFireflies(ctx, state, canvas, intensity, speed);
				break;
			case 'rain':
				drawRain(ctx, state, canvas, intensity, speed);
				break;
			case 'snow':
				// signature: (ctx, state, canvas, intensity, speed, time)
				drawSnow(ctx, state, canvas, intensity, speed, time);
				break;
			case 'confetti':
				drawConfetti(ctx, state, canvas, intensity, speed);
				break;
			case 'pulse':
				// signature: (ctx, state, canvas, intensity, colors, speed) — no time
				drawPulse(ctx, state, canvas, intensity, colors, speed);
				break;
			case 'ripples':
				drawRipples(ctx, state, canvas, intensity, speed);
				break;
			case 'nebula':
				// signature: (ctx, state, canvas, intensity, colors, speed) — no time
				drawNebula(ctx, state, canvas, intensity, colors, speed);
				break;
			case 'circuit':
				// signature: (ctx, state, intensity, colors, speed, time) — no canvas
				drawCircuit(ctx, state, intensity, colors, speed, time);
				break;
			case 'dots':
				// signature: (ctx, state, intensity, colors, speed) — no canvas
				drawDots(ctx, state, intensity, colors, speed);
				break;
			case 'floatingShapes':
				// signature: (ctx, state, canvas, intensity, speed) — no time
				drawFloatingShapes(ctx, state, canvas, intensity, speed);
				break;
			case 'smoke':
				// signature: (ctx, state, canvas, intensity, colors, speed)
				drawSmoke(ctx, state, canvas, intensity, colors, speed);
				break;
			case 'scanlines':
				drawScanlines(ctx, state, canvas, intensity, colors, speed, time);
				break;
			case 'grid':
				// signature: (ctx, state, intensity, colors, speed, time) — no canvas
				drawGrid(ctx, state, intensity, colors, speed, time);
				break;
			case 'warp':
				// signature: (ctx, state, canvas, intensity, speed) — no time
				drawWarp(ctx, state, canvas, intensity, speed);
				break;
			case 'hexgrid':
				// signature: (ctx, state, intensity, colors, speed) — no canvas, no time
				drawHexgrid(ctx, state, intensity, colors, speed);
				break;
			case 'binary':
				// signature: (ctx, state, canvas, intensity, colors, speed) — no time
				drawBinary(ctx, state, canvas, intensity, colors, speed);
				break;
		}

		animationId = requestAnimationFrame(animate);
	}

	onMount(() => {
		if (!browser) return;

		initCanvas();
		window.addEventListener('resize', resizeCanvas);

		if (effectType !== 'none') {
			animationId = requestAnimationFrame(animate);
		}
	});

	onDestroy(() => {
		if (browser) {
			cancelAnimationFrame(animationId);
			window.removeEventListener('resize', resizeCanvas);
		}
	});

	// React to prop changes
	$effect(() => {
		if (browser && canvas) {
			cancelAnimationFrame(animationId);
			// Clear canvas on effect transition
			if (ctx) {
				ctx.clearRect(0, 0, canvas.width, canvas.height);
			}
			state = createAnimationState();
			initEffect();
			if (effectType !== 'none') {
				animationId = requestAnimationFrame(animate);
			}
		}
	});
</script>

{#if effectType !== 'none'}
	<canvas
		bind:this={canvas}
		class="animated-background"
		class:matrix-bg={effectType === 'matrix'}
	></canvas>
{/if}

<style>
	.animated-background {
		position: fixed;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		pointer-events: none;
		z-index: 0;
	}

	.matrix-bg {
		background: rgba(0, 0, 0, 0.9);
	}
</style>
