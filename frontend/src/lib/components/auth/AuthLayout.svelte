<script lang="ts">
	import { fade, fly, scale } from 'svelte/transition';
	import { onMount } from 'svelte';

	let { children } = $props();

	// Boot-like animation
	let showGrid = $state(false);
	let showContent = $state(false);
	let showTerminal = $state(false);

	// Typewriter states
	let statusLine = $state('');
	let authLine = $state('');
	let showModules = $state(false);
	let terminalReady = $state(false);

	const statusText = '> System ready';
	const authText = '> Awaiting authentication_';

	function typeWriter(text: string, setter: (val: string) => void, speed: number = 40): Promise<void> {
		return new Promise((resolve) => {
			let i = 0;
			const interval = setInterval(() => {
				if (i < text.length) {
					setter(text.slice(0, i + 1));
					i++;
				} else {
					clearInterval(interval);
					resolve();
				}
			}, speed);
		});
	}

	onMount(() => {
		setTimeout(() => (showGrid = true), 100);
		setTimeout(() => (showContent = true), 300);
		setTimeout(() => (showTerminal = true), 600);

		// Start typewriter sequence
		setTimeout(async () => {
			await typeWriter(statusText, (val) => statusLine = val, 35);
			await new Promise(r => setTimeout(r, 200));
			await typeWriter(authText, (val) => authLine = val, 35);
			await new Promise(r => setTimeout(r, 300));
			showModules = true;
			await new Promise(r => setTimeout(r, 800));
			terminalReady = true;
		}, 800);
	});

	const modules = [
		{ id: '01', name: 'dashboard', status: 'ready' },
		{ id: '02', name: 'chat', status: 'ready' },
		{ id: '03', name: 'tasks', status: 'ready' },
		{ id: '04', name: 'projects', status: 'ready' },
		{ id: '05', name: 'calendar', status: 'ready' },
	];
</script>

<div class="min-h-screen flex bg-white">
	<!-- Left Panel - Terminal Style Branding -->
	<div class="hidden lg:flex lg:w-1/2 bg-black p-12 flex-col justify-between relative overflow-hidden">
		<!-- Grid Pattern Overlay - More visible -->
		{#if showGrid}
			<div
				class="absolute inset-0"
				in:fade={{ duration: 800 }}
			>
				<!-- Primary grid - increased visibility -->
				<div
					class="w-full h-full opacity-[0.12]"
					style="
						background-image:
							linear-gradient(rgba(255,255,255,0.25) 1px, transparent 1px),
							linear-gradient(90deg, rgba(255,255,255,0.25) 1px, transparent 1px);
						background-size: 50px 50px;
					"
				></div>
				<!-- Secondary finer grid -->
				<div
					class="absolute inset-0 w-full h-full opacity-[0.06]"
					style="
						background-image:
							linear-gradient(rgba(255,255,255,0.2) 1px, transparent 1px),
							linear-gradient(90deg, rgba(255,255,255,0.2) 1px, transparent 1px);
						background-size: 10px 10px;
					"
				></div>
				<!-- Dot pattern -->
				<div
					class="absolute inset-0 w-full h-full opacity-[0.04]"
					style="
						background-image: radial-gradient(circle, rgba(255,255,255,0.5) 1px, transparent 1px);
						background-size: 25px 25px;
					"
				></div>
				<!-- Corner brackets - larger and more visible -->
				<div class="absolute top-6 left-6 w-24 h-24 border-l-2 border-t-2 border-white/20"></div>
				<div class="absolute top-6 right-6 w-24 h-24 border-r-2 border-t-2 border-white/20"></div>
				<div class="absolute bottom-6 left-6 w-24 h-24 border-l-2 border-b-2 border-white/20"></div>
				<div class="absolute bottom-6 right-6 w-24 h-24 border-r-2 border-b-2 border-white/20"></div>
				<!-- Center crosshair accent -->
				<div class="absolute inset-0 flex items-center justify-center pointer-events-none opacity-[0.05]">
					<div class="w-px h-full bg-gradient-to-b from-transparent via-white to-transparent"></div>
				</div>
				<div class="absolute inset-0 flex items-center justify-center pointer-events-none opacity-[0.05]">
					<div class="h-px w-full bg-gradient-to-r from-transparent via-white to-transparent"></div>
				</div>
				<!-- Floating orbs -->
				<div class="absolute top-1/4 left-1/4 w-32 h-32 bg-green-500/5 rounded-full blur-3xl animate-pulse-slow"></div>
				<div class="absolute bottom-1/3 right-1/4 w-40 h-40 bg-blue-500/5 rounded-full blur-3xl animate-pulse-slower"></div>
			</div>
		{/if}

		<!-- Scan Line Effect -->
		<div class="absolute inset-0 pointer-events-none overflow-hidden">
			<div class="scan-line-auth absolute w-full h-px bg-gradient-to-r from-transparent via-green-500/30 to-transparent"></div>
		</div>

		{#if showContent}
			<!-- Logo & Tagline with typewriter -->
			<div class="relative z-10" in:fly={{ y: -20, duration: 500 }}>
				<div class="flex items-baseline gap-0.5 mb-6">
					<span class="text-white text-3xl font-extrabold tracking-[0.2em] font-mono glitch-text-white" data-text="FOUNDER">FOUNDER</span>
					<span class="text-white/40 text-2xl font-light font-mono">OS</span>
				</div>
				<div class="font-mono text-gray-500 text-sm space-y-1">
					<p class="flex items-center">
						{statusLine}
						{#if statusLine.length > 0 && statusLine.length < statusText.length}
							<span class="inline-block w-2 h-4 bg-green-500 ml-0.5 animate-blink"></span>
						{/if}
					</p>
					<p class="flex items-center">
						{authLine}
						{#if authLine.length > 0 && authLine.length < authText.length}
							<span class="inline-block w-2 h-4 bg-green-500 ml-0.5 animate-blink"></span>
						{:else if authLine.length === authText.length}
							<span class="inline-block w-2 h-4 bg-green-500 ml-0.5 animate-blink"></span>
						{/if}
					</p>
				</div>
			</div>

			<!-- Terminal Preview with animations -->
			{#if showTerminal}
				<div class="relative z-10 flex-1 flex items-center justify-center py-12" in:fly={{ y: 30, duration: 600 }}>
					<div class="w-full max-w-sm">
						<!-- Terminal Window -->
						<div class="bg-gray-900/90 border border-gray-700 rounded-lg overflow-hidden shadow-2xl shadow-black/50 backdrop-blur-sm">
							<!-- Terminal Header -->
							<div class="flex items-center gap-2 px-4 py-2.5 bg-gray-900 border-b border-gray-800">
								<div class="flex gap-1.5">
									<div class="w-3 h-3 rounded-full bg-red-500 hover:bg-red-400 transition-colors cursor-pointer"></div>
									<div class="w-3 h-3 rounded-full bg-yellow-500 hover:bg-yellow-400 transition-colors cursor-pointer"></div>
									<div class="w-3 h-3 rounded-full bg-green-500 hover:bg-green-400 transition-colors cursor-pointer"></div>
								</div>
								<span class="ml-2 text-xs text-gray-500 font-mono">businessos ~ auth</span>
								<div class="ml-auto flex items-center gap-1">
									<span class="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse"></span>
									<span class="text-[10px] text-green-500/70 font-mono">SECURE</span>
								</div>
							</div>
							<!-- Terminal Content -->
							<div class="p-4 font-mono text-sm space-y-2">
								<div class="text-gray-500">$ businessos --status</div>
								<div class="text-green-400 flex items-center gap-2">
									<span class="inline-block w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse"></span>
									STATUS: ONLINE
								</div>
								<div class="text-gray-500">$ modules list</div>
								{#if showModules}
									<div class="text-gray-400 pl-2 space-y-1">
										{#each modules as mod, i}
											<div
												class="flex gap-4 items-center"
												in:fly={{ x: -10, duration: 200, delay: i * 100 }}
											>
												<span class="text-gray-600 w-4">{mod.id}</span>
												<span class="flex-1">{mod.name}</span>
												<span class="text-green-500/60 text-xs">[{mod.status}]</span>
											</div>
										{/each}
									</div>
								{/if}
								<div class="text-gray-500 pt-2 flex items-center">
									<span>$ </span>
									{#if terminalReady}
										<span class="text-green-400" in:fade={{ duration: 300 }}>auth --init</span>
									{/if}
									<span class="inline-block w-2 h-4 bg-green-500 ml-0.5 animate-blink"></span>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<!-- Version Info -->
			<div class="relative z-10 font-mono text-xs text-gray-600" in:fly={{ y: 20, duration: 500, delay: 400 }}>
				<div class="flex items-center gap-4">
					<span class="text-gray-500">v0.0.1</span>
					<span class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span>
					<span class="text-gray-500">All systems operational</span>
				</div>
				<div class="mt-2 flex items-center gap-3 text-gray-600">
					<span class="flex items-center gap-1">
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
						</svg>
						Encrypted
					</span>
					<span class="flex items-center gap-1">
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7" />
						</svg>
						Self-hosted
					</span>
				</div>
			</div>
		{/if}
	</div>

	<!-- Right Panel - Form -->
	<div class="flex-1 flex items-center justify-center p-4 sm:p-6 lg:p-12 bg-white relative overflow-hidden auth-right-panel">
		<!-- Grid for right panel - more visible -->
		<div class="absolute inset-0 pointer-events-none">
			<div
				class="w-full h-full opacity-[0.06]"
				style="
					background-image:
						linear-gradient(rgba(0,0,0,0.2) 1px, transparent 1px),
						linear-gradient(90deg, rgba(0,0,0,0.2) 1px, transparent 1px);
					background-size: 50px 50px;
				"
			></div>
			<!-- Finer grid -->
			<div
				class="absolute inset-0 w-full h-full opacity-[0.03]"
				style="
					background-image:
						linear-gradient(rgba(0,0,0,0.15) 1px, transparent 1px),
						linear-gradient(90deg, rgba(0,0,0,0.15) 1px, transparent 1px);
					background-size: 10px 10px;
				"
			></div>
			<!-- Dot pattern -->
			<div
				class="absolute inset-0 w-full h-full opacity-[0.02]"
				style="
					background-image: radial-gradient(circle, rgba(0,0,0,0.5) 1px, transparent 1px);
					background-size: 20px 20px;
				"
			></div>
			<!-- Corner accents -->
			<div class="absolute top-4 left-4 w-16 h-16 border-l border-t border-gray-200"></div>
			<div class="absolute top-4 right-4 w-16 h-16 border-r border-t border-gray-200"></div>
			<div class="absolute bottom-4 left-4 w-16 h-16 border-l border-b border-gray-200"></div>
			<div class="absolute bottom-4 right-4 w-16 h-16 border-r border-b border-gray-200"></div>
		</div>

		<!-- Floating particles on right side -->
		<div class="absolute top-1/4 right-1/4 w-1 h-1 bg-gray-300 rounded-full animate-float-slow"></div>
		<div class="absolute top-1/3 left-1/3 w-1.5 h-1.5 bg-gray-200 rounded-full animate-float-medium"></div>
		<div class="absolute bottom-1/4 right-1/3 w-1 h-1 bg-gray-300 rounded-full animate-float-fast"></div>

		<div class="w-full max-w-md relative z-10 auth-form-container" in:fade={{ duration: 300 }}>
			<!-- Back to home link -->
			<a
				href="/"
				class="inline-flex items-center gap-2 text-xs text-gray-400 hover:text-gray-900 transition-colors font-mono mb-8 group"
			>
				<svg class="w-4 h-4 transition-transform group-hover:-translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				Back to home
			</a>

			<!-- Mobile Logo -->
			<div class="lg:hidden flex items-baseline gap-0.5 mb-10 justify-center">
				<a href="/" class="flex items-baseline gap-0.5 hover:opacity-70 transition-opacity">
					<span class="text-black text-2xl font-extrabold tracking-[0.15em] font-mono">FOUNDER</span>
					<span class="text-black/30 text-xl font-light font-mono">OS</span>
				</a>
			</div>

			{@render children()}
		</div>
	</div>
</div>

<style>
	@keyframes scanline {
		0% { transform: translateY(-100%); }
		100% { transform: translateY(100vh); }
	}

	.scan-line-auth {
		animation: scanlineAuth 4s linear infinite;
	}

	@keyframes scanlineAuth {
		0% { top: -2px; opacity: 0; }
		5% { opacity: 1; }
		95% { opacity: 1; }
		100% { top: 100%; opacity: 0; }
	}

	@keyframes blink {
		0%, 50% { opacity: 1; }
		51%, 100% { opacity: 0; }
	}

	:global(.animate-blink) {
		animation: blink 1s step-end infinite;
	}

	:global(.animate-pulse-slow) {
		animation: pulse 4s ease-in-out infinite;
	}

	:global(.animate-pulse-slower) {
		animation: pulse 6s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 0.5; transform: scale(1); }
		50% { opacity: 0.8; transform: scale(1.1); }
	}

	/* Floating particles */
	:global(.animate-float-slow) {
		animation: floatSlow 8s ease-in-out infinite;
	}

	:global(.animate-float-medium) {
		animation: floatMedium 6s ease-in-out infinite;
	}

	:global(.animate-float-fast) {
		animation: floatFast 4s ease-in-out infinite;
	}

	@keyframes floatSlow {
		0%, 100% { transform: translate(0, 0); }
		25% { transform: translate(10px, -15px); }
		50% { transform: translate(-5px, -25px); }
		75% { transform: translate(-15px, -10px); }
	}

	@keyframes floatMedium {
		0%, 100% { transform: translate(0, 0); }
		33% { transform: translate(-12px, -20px); }
		66% { transform: translate(8px, -10px); }
	}

	@keyframes floatFast {
		0%, 100% { transform: translate(0, 0); }
		50% { transform: translate(15px, -20px); }
	}

	/* Glitch effect for white text */
	.glitch-text-white {
		position: relative;
		display: inline-block;
	}

	.glitch-text-white::before,
	.glitch-text-white::after {
		content: attr(data-text);
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		opacity: 0;
	}

	.glitch-text-white::before {
		color: #00ffff;
		animation: glitchWhite1 5s infinite linear alternate-reverse;
	}

	.glitch-text-white::after {
		color: #ff00ff;
		animation: glitchWhite2 4s infinite linear alternate-reverse;
	}

	@keyframes glitchWhite1 {
		0%, 94% { opacity: 0; transform: translate(0); }
		95% { opacity: 0.4; transform: translate(-2px, 1px); }
		96% { opacity: 0; transform: translate(0); }
		97% { opacity: 0.3; transform: translate(2px, -1px); }
		98%, 100% { opacity: 0; transform: translate(0); }
	}

	@keyframes glitchWhite2 {
		0%, 92% { opacity: 0; transform: translate(0); }
		93% { opacity: 0.3; transform: translate(1px, -1px); }
		94% { opacity: 0; transform: translate(0); }
		95% { opacity: 0.4; transform: translate(-1px, 1px); }
		96%, 100% { opacity: 0; transform: translate(0); }
	}

	/* Mobile: full-width card, no overflow */
	@media (max-width: 480px) {
		.auth-right-panel {
			padding: 1rem;
			align-items: flex-start;
			padding-top: 2rem;
		}

		.auth-form-container {
			max-width: 100%;
		}
	}

	@media (max-width: 320px) {
		.auth-right-panel {
			padding: 0.75rem;
		}
	}
</style>
