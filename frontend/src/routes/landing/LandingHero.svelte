<script lang="ts">
	import { fly, fade } from 'svelte/transition';

	interface Module {
		name: string;
		desc: string;
		slug: string;
	}

	interface Props {
		typedTagline: string;
		showTaglineCursor: boolean;
		terminalLine1: string;
		terminalLine2: string;
		showTerminalOutput: boolean;
		showModuleList: boolean;
		modules: Module[];
		cmd1: string;
		cmd2: string;
	}

	let {
		typedTagline,
		showTaglineCursor,
		terminalLine1,
		terminalLine2,
		showTerminalOutput,
		showModuleList,
		modules,
		cmd1,
		cmd2
	}: Props = $props();
</script>

<!-- Hero -->
<section class="pt-28 pb-20 px-6">
	<div class="max-w-3xl mx-auto">
		<!-- Terminal-style status -->
		<div
			class="font-mono text-xs text-gray-400 mb-8 flex items-center gap-3"
			in:fly={{ y: 20, duration: 500 }}
		>
			<span class="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse"></span>
			<span>SYSTEM ONLINE</span>
			<span class="text-gray-300">|</span>
			<span>v0.0.1</span>
		</div>

		<!-- Main headline -->
		<h1
			class="text-4xl md:text-6xl font-bold text-black leading-[1.1] mb-6 tracking-tight"
			in:fly={{ y: 30, duration: 600, delay: 100 }}
		>
			Your business operating system for the{' '}
			<span class="text-gray-400 glitch-text" data-text="agentic era">agentic era</span>
		</h1>

		<p
			class="text-lg text-gray-500 max-w-xl mb-12 font-mono h-7"
			in:fly={{ y: 30, duration: 600, delay: 200 }}
		>
			{typedTagline}<span class="inline-block w-0.5 h-5 bg-gray-400 ml-0.5 align-middle {showTaglineCursor ? 'animate-pulse' : 'opacity-0'}"></span>
		</p>

		<!-- CTA -->
		<div
			class="flex flex-col sm:flex-row items-start gap-4"
			in:fly={{ y: 30, duration: 600, delay: 300 }}
		>
			<a href="/register" class="bg-black text-white px-8 py-3 rounded-lg text-sm font-medium hover:bg-gray-800 transition-colors font-mono flex items-center gap-2">
				Initialize workspace
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
				</svg>
			</a>
			<a href="/login" class="text-gray-600 px-4 py-3 text-sm font-mono hover:text-black transition-colors">
				Sign in
			</a>
		</div>
	</div>
</section>

<!-- Terminal Preview -->
<section class="pb-24 px-6" in:fly={{ y: 40, duration: 800, delay: 400 }}>
	<div class="max-w-4xl mx-auto">
		<div class="bg-black rounded-xl overflow-hidden shadow-2xl">
			<!-- Terminal header -->
			<div class="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
				<div class="flex gap-1.5">
					<div class="w-3 h-3 rounded-full bg-red-500"></div>
					<div class="w-3 h-3 rounded-full bg-yellow-500"></div>
					<div class="w-3 h-3 rounded-full bg-green-500"></div>
				</div>
				<span class="ml-4 text-xs text-gray-500 font-mono">businessos ~ terminal</span>
			</div>
			<!-- Terminal content -->
			<div class="p-6 font-mono text-sm space-y-3">
				<div class="text-gray-500 flex items-center">
					<span>{terminalLine1}</span>
					{#if terminalLine1.length > 0 && terminalLine1.length < cmd1.length}
						<span class="inline-block w-2 h-4 bg-gray-500 ml-0.5 animate-pulse"></span>
					{/if}
				</div>
				{#if showTerminalOutput}
					<div class="text-green-400" in:fade={{ duration: 200 }}>
						<span class="inline-block animate-pulse mr-2">></span>
						All systems operational
					</div>
				{/if}
				{#if terminalLine2.length > 0}
					<div class="text-gray-500 flex items-center">
						<span>{terminalLine2}</span>
						{#if terminalLine2.length > 0 && terminalLine2.length < cmd2.length}
							<span class="inline-block w-2 h-4 bg-gray-500 ml-0.5 animate-pulse"></span>
						{/if}
					</div>
				{/if}
				{#if showModuleList}
					<div class="grid grid-cols-2 sm:grid-cols-3 gap-2 pl-2" in:fade={{ duration: 300 }}>
						{#each modules as mod, i}
							<div class="text-gray-400 flex gap-2" in:fly={{ y: 10, duration: 300, delay: i * 80 }}>
								<span class="text-gray-600 w-4">{String(i + 1).padStart(2, '0')}</span>
								<span class="typewriter-text" style="animation-delay: {i * 80}ms">{mod.name.toLowerCase().replace(' ', '-')}</span>
							</div>
						{/each}
					</div>
				{/if}
				<div class="text-gray-500 pt-2 flex items-center">
					<span>$ </span>
					<span class="inline-block w-2 h-4 bg-green-500 ml-0.5 animate-blink"></span>
				</div>
			</div>
		</div>
	</div>
</section>

<style>
	.typewriter-text {
		opacity: 0;
		animation: fadeIn 0.3s ease forwards;
	}

	@keyframes fadeIn {
		from { opacity: 0; transform: translateX(-5px); }
		to { opacity: 1; transform: translateX(0); }
	}

	.glitch-text {
		position: relative;
		display: inline-block;
	}

	.glitch-text::before,
	.glitch-text::after {
		content: attr(data-text);
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		opacity: 0;
	}

	.glitch-text::before {
		color: #00ffff;
		animation: glitch-1 4s infinite linear alternate-reverse;
	}

	.glitch-text::after {
		color: #ff00ff;
		animation: glitch-2 3s infinite linear alternate-reverse;
	}

	@keyframes glitch-1 {
		0%, 94% { opacity: 0; transform: translate(0); }
		95% { opacity: 0.3; transform: translate(-2px, 1px); }
		96% { opacity: 0; transform: translate(0); }
		97% { opacity: 0.2; transform: translate(2px, -1px); }
		98%, 100% { opacity: 0; transform: translate(0); }
	}

	@keyframes glitch-2 {
		0%, 92% { opacity: 0; transform: translate(0); }
		93% { opacity: 0.2; transform: translate(1px, -1px); }
		94% { opacity: 0; transform: translate(0); }
		95% { opacity: 0.3; transform: translate(-1px, 1px); }
		96%, 100% { opacity: 0; transform: translate(0); }
	}
</style>
