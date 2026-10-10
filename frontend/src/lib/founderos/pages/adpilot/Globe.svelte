<!-- The AdPilot globe (FounderOS v1 components/adpilot/Globe.tsx): dot-matrix
     earth from real coastline samples, black body, a red atmospheric bleed,
     teardrop pins where leads book from, glass chips on the top cities.
     Drag to rotate with inertia; idles into a slow spin (none under reduced
     motion). Plain 2D canvas with its own orthographic projection. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import dotsUrl from './globe-dots.json?url';
	import { globeChips, latLngToVec, rotate, type Vec } from './globe';
	import type { GeoPoint } from './types';

	let { geo, height = 560 }: { geo: GeoPoint[]; height?: number } = $props();

	const TILT = 0.32;
	const OVERHANG = 40;
	const RED = '255, 52, 72';
	const PIN = '#ff4557';
	const PIN_CORE = '#8f1520';

	let mount: HTMLDivElement;
	let canvas: HTMLCanvasElement;
	let ready = $state(false);
	let tooltip = $state<{ x: number; y: number; g: GeoPoint } | null>(null);
	const chipEls: (HTMLDivElement | null)[] = [];

	const chips = $derived(globeChips(geo));

	function drawPin(ctx: CanvasRenderingContext2D, x: number, y: number, h: number) {
		const headR = h * 0.23;
		const headY = y - h + headR + h * 0.06;
		ctx.fillStyle = PIN;
		ctx.beginPath();
		ctx.arc(x, headY, headR, Math.PI * 0.93, Math.PI * 0.07, false);
		ctx.quadraticCurveTo(x + headR * 0.55, headY + headR * 1.15, x, y);
		ctx.quadraticCurveTo(x - headR * 0.55, headY + headR * 1.15, x - headR * Math.cos(Math.PI * 0.07), headY + headR * Math.sin(Math.PI * 0.07));
		ctx.closePath();
		ctx.fill();
		ctx.fillStyle = PIN_CORE;
		ctx.beginPath();
		ctx.arc(x, headY, headR * 0.42, 0, Math.PI * 2);
		ctx.fill();
	}

	onMount(() => {
		let ctx: CanvasRenderingContext2D | null = null;
		try {
			ctx = canvas.getContext('2d');
		} catch {
			ctx = null;
		}
		if (!ctx) {
			ready = true; // no canvas (tests, very old browsers): chips and page still render
			return;
		}
		const c = ctx;
		const reduced = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const canvasH = height + OVERHANG * 2;
		let w = mount.clientWidth;
		let dpr = Math.min(window.devicePixelRatio || 1, 2);
		const resize = () => {
			w = mount.clientWidth;
			dpr = Math.min(window.devicePixelRatio || 1, 2);
			canvas.width = Math.round(w * dpr);
			canvas.height = Math.round(canvasH * dpr);
			canvas.style.width = `${w}px`;
			canvas.style.height = `${canvasH}px`;
		};
		resize();
		const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(resize) : null;
		ro?.observe(mount);

		let land: Vec[] = [];
		let disposed = false;
		fetch(dotsUrl)
			.then((r) => r.json())
			.then((dots: [number, number][]) => {
				if (disposed) return;
				land = dots.map(([lat, lng]) => latLngToVec(lat, lng));
				ready = true;
			})
			.catch(() => (ready = true)); // body, atmosphere and pins still render

		let yaw = 0;
		let pitch = TILT;
		let dragging = false;
		let lastX = 0;
		let lastY = 0;
		let velocity = reduced ? 0 : 0.0016;
		let idle = 0;
		let hits: { x: number; y: number; r: number; g: GeoPoint }[] = [];

		const onDown = (e: PointerEvent) => {
			dragging = true;
			lastX = e.clientX;
			lastY = e.clientY;
			canvas.setPointerCapture?.(e.pointerId);
		};
		const onMove = (e: PointerEvent) => {
			if (dragging) {
				const dx = e.clientX - lastX;
				const dy = e.clientY - lastY;
				lastX = e.clientX;
				lastY = e.clientY;
				yaw += dx * 0.005;
				pitch = Math.max(-0.9, Math.min(0.9, pitch + dy * 0.0028));
				velocity = dx * 0.0006;
				idle = 0;
			}
			const rect = canvas.getBoundingClientRect();
			const mrect = mount.getBoundingClientRect();
			const px = e.clientX - rect.left;
			const py = e.clientY - rect.top;
			const hit = hits.find((h) => Math.hypot(px - h.x, py - h.y) <= h.r);
			tooltip = hit ? { x: e.clientX - mrect.left, y: e.clientY - mrect.top, g: hit.g } : null;
			canvas.style.cursor = hit ? 'pointer' : dragging ? 'grabbing' : 'grab';
		};
		const onUp = () => (dragging = false);
		canvas.addEventListener('pointerdown', onDown);
		canvas.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);

		let raf = 0;
		const frame = () => {
			raf = requestAnimationFrame(frame);
			if (!dragging && !reduced) {
				yaw += velocity;
				idle += 1;
				if (idle > 120) velocity += (0.0016 - velocity) * 0.02;
				else velocity *= 0.985;
			}
			const cx = w / 2;
			const cy = canvasH / 2;
			const R = Math.min(w, canvasH) * 0.36;
			c.setTransform(dpr, 0, 0, dpr, 0, 0);
			c.clearRect(0, 0, w, canvasH);

			const bleed = c.createRadialGradient(cx, cy, R * 0.96, cx, cy, R * 1.3);
			bleed.addColorStop(0, `rgba(${RED}, 0.55)`);
			bleed.addColorStop(0.35, `rgba(${RED}, 0.16)`);
			bleed.addColorStop(1, `rgba(${RED}, 0)`);
			c.fillStyle = bleed;
			c.beginPath();
			c.arc(cx, cy, R * 1.3, 0, Math.PI * 2);
			c.fill();
			c.fillStyle = '#05080a';
			c.beginPath();
			c.arc(cx, cy, R, 0, Math.PI * 2);
			c.fill();
			const limb = c.createRadialGradient(cx, cy, R * 0.8, cx, cy, R);
			limb.addColorStop(0, `rgba(${RED}, 0)`);
			limb.addColorStop(0.75, `rgba(${RED}, 0.12)`);
			limb.addColorStop(1, `rgba(${RED}, 0.6)`);
			c.fillStyle = limb;
			c.beginPath();
			c.arc(cx, cy, R, 0, Math.PI * 2);
			c.fill();

			const dot = Math.max(1, R * 0.0065);
			for (const p of land) {
				const v = rotate(p, yaw, pitch);
				if (v.z <= 0) continue;
				c.fillStyle = `rgba(168, 162, 159, ${(0.25 + 0.67 * v.z).toFixed(3)})`;
				c.fillRect(cx + v.x * R - dot / 2, cy - v.y * R - dot / 2, dot, dot);
			}

			const points = geo;
			const max = Math.max(...points.map((g) => g.leads), 1);
			hits = [];
			for (const g of points) {
				const v = rotate(latLngToVec(g.lat, g.lng), yaw, pitch);
				if (v.z <= 0) continue;
				const x = cx + v.x * R * 1.002;
				const y = cy - v.y * R * 1.002;
				const s = (0.052 + 0.03 * Math.sqrt(g.leads / max)) * R;
				const halo = c.createRadialGradient(x, y, 0, x, y, s * 0.8);
				halo.addColorStop(0, 'rgba(255,69,87,0.75)');
				halo.addColorStop(0.4, 'rgba(255,69,87,0.22)');
				halo.addColorStop(1, 'rgba(255,69,87,0)');
				c.fillStyle = halo;
				c.beginPath();
				c.arc(x, y, s * 0.8, 0, Math.PI * 2);
				c.fill();
				drawPin(c, x, y, s);
				hits.push({ x, y: y - s * 0.7, r: Math.max(8, s * 0.3), g });
			}

			chips.forEach((g, i) => {
				const el = chipEls[i];
				if (!el) return;
				const v = rotate(latLngToVec(g.lat, g.lng), yaw, pitch);
				const x = cx + v.x * R * 1.01;
				const y = cy - v.y * R * 1.01 - OVERHANG;
				const inBounds = x > 60 && x < w - 216 && y > 26 && y < height - 14;
				el.style.transform =
					i % 2 === 0
						? `translate(${Math.round(x + 12)}px, ${Math.round(y - 30)}px)`
						: `translate(${Math.round(x - 12)}px, ${Math.round(y - 46)}px) translateX(-100%)`;
				el.style.opacity = v.z > 0 && inBounds ? '1' : '0';
			});
		};
		frame();

		return () => {
			disposed = true;
			cancelAnimationFrame(raf);
			ro?.disconnect();
			canvas.removeEventListener('pointerdown', onDown);
			canvas.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
		};
	});
</script>

<div
	bind:this={mount}
	data-part="globe"
	class="relative w-full touch-none"
	style="height: {height}px; opacity: {ready ? 1 : 0}; transform: scale({ready ? 1 : 0.965}); transition: opacity 1100ms ease-out, transform 1100ms ease-out"
>
	<canvas bind:this={canvas} class="absolute left-0 z-0" style="top: -{OVERHANG}px" aria-hidden="true"></canvas>
	{#each chips as g, i (`${g.city}-${g.country}`)}
		<div
			bind:this={chipEls[i]}
			class="pointer-events-none absolute left-0 top-0 z-20 flex items-center gap-1.5 whitespace-nowrap rounded-full border border-[rgba(255,69,87,0.28)] bg-[#0a0507e6] px-2.5 py-1 opacity-0"
			style="transition: opacity 200ms"
		>
			<span class="bn-text text-[12px]">{g.city}</span>
			<span class="bn-muted text-[11px] tabular-nums">{g.leads} leads</span>
		</div>
	{/each}
	{#if tooltip}
		<div
			class="pointer-events-none absolute z-30 rounded-[9px] border border-[rgba(255,69,87,0.35)] bg-[#0a0507ee] px-3 py-2"
			style="left: {tooltip.x + 14}px; top: {tooltip.y - 10}px"
		>
			<div class="bn-text text-[12.5px]">{tooltip.g.city}, {tooltip.g.country}</div>
			<div class="bn-muted text-[11.5px]">{tooltip.g.leads} leads · {tooltip.g.bookings} booked</div>
		</div>
	{/if}
</div>
