<script lang="ts">
	import { fly } from 'svelte/transition';

	interface Module {
		name: string;
		desc: string;
		slug: string;
	}

	interface Capability {
		title: string;
		desc: string;
	}

	interface HowItWorksItem {
		step: string;
		title: string;
		desc: string;
		icon: string;
	}

	interface Integration {
		name: string;
		category: string;
	}

	interface Props {
		modules: Module[];
		capabilities: Capability[];
		howItWorks: HowItWorksItem[];
		integrations: Integration[];
	}

	let { modules, capabilities, howItWorks, integrations }: Props = $props();
</script>

<!-- Download Desktop App -->
<section id="download" class="py-20 px-6 bg-gray-50">
	<div class="max-w-5xl mx-auto">
		<div class="text-center mb-10">
			<div class="font-mono text-xs text-gray-400 mb-3 tracking-wider">DESKTOP APP</div>
			<h3 class="text-2xl md:text-3xl font-bold text-black mb-3">Download BusinessOS Desktop</h3>
			<p class="text-gray-500 max-w-xl mx-auto text-sm">
				Native app with global shortcuts, voice input, screenshot capture, and meeting recording.
			</p>
		</div>

		<div class="grid md:grid-cols-3 gap-4 max-w-3xl mx-auto">
			<!-- macOS -->
			<div class="bg-white border-2 border-gray-200 rounded-xl p-5 hover:border-black hover:shadow-lg transition-all group">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center group-hover:bg-black group-hover:text-white transition-colors">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>
						</svg>
					</div>
					<div>
						<div class="font-bold text-black text-sm">macOS</div>
						<div class="text-[10px] text-gray-400 font-mono">Apple Silicon</div>
					</div>
				</div>
				<div class="flex gap-2">
					<a
						href="https://storage.googleapis.com/businessos-downloads/BusinessOS-arm64.dmg"
						download
						class="flex-1 bg-black text-white py-2.5 rounded-lg text-xs font-medium hover:bg-gray-800 transition-colors font-mono flex items-center justify-center gap-1.5"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
						</svg>
						Silicon
					</a>
					<div
						class="flex-1 bg-gray-100 text-gray-400 py-2.5 rounded-lg text-xs font-medium font-mono flex items-center justify-center gap-1.5 cursor-not-allowed"
						title="Intel, Windows & Linux builds coming soon"
					>
						Intel soon
					</div>
				</div>
			</div>

			<!-- Windows -->
			<div class="bg-white border-2 border-gray-200 rounded-xl p-5 transition-all opacity-60">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M3 5.548l7.262-1.006v7.021H3V5.548zm0 12.904l7.262 1.006v-6.895H3v5.889zm8.138 1.118L21 21v-7.437h-9.862v6.007zm0-14.036v6.029H21V3l-9.862 1.534z"/>
						</svg>
					</div>
					<div>
						<div class="font-bold text-black text-sm">Windows</div>
						<div class="text-[10px] text-gray-400 font-mono">Windows 10+</div>
					</div>
				</div>
				<button disabled class="btn-pill btn-pill-soft btn-pill-xs w-full font-mono">Coming Soon</button>
			</div>

			<!-- Linux -->
			<div class="bg-white border-2 border-gray-200 rounded-xl p-5 hover:border-black hover:shadow-lg transition-all group">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center group-hover:bg-black group-hover:text-white transition-colors">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139z"/>
						</svg>
					</div>
					<div>
						<div class="font-bold text-black text-sm">Linux</div>
						<div class="text-[10px] text-gray-400 font-mono">.deb / .rpm</div>
					</div>
				</div>
				<div class="flex gap-2">
					<a
						href="https://storage.googleapis.com/businessos-downloads/BusinessOS-linux-x64.deb"
						download
						class="flex-1 bg-black text-white py-2.5 rounded-lg text-xs font-medium hover:bg-gray-800 transition-colors font-mono flex items-center justify-center gap-1.5"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
						</svg>
						.deb
					</a>
					<a
						href="https://storage.googleapis.com/businessos-downloads/BusinessOS-linux-x64.rpm"
						download
						class="flex-1 bg-gray-100 text-black py-2.5 rounded-lg text-xs font-medium hover:bg-gray-200 transition-colors font-mono flex items-center justify-center gap-1.5"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
						</svg>
						.rpm
					</a>
				</div>
			</div>
		</div>

		<!-- Features row -->
		<div class="mt-10 flex flex-wrap justify-center gap-6 text-center">
			<div class="flex items-center gap-2">
				<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
				</svg>
				<span class="text-xs text-gray-500 font-mono">Global Shortcuts</span>
			</div>
			<div class="flex items-center gap-2">
				<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
				</svg>
				<span class="text-xs text-gray-500 font-mono">Voice Input</span>
			</div>
			<div class="flex items-center gap-2">
				<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
				<span class="text-xs text-gray-500 font-mono">Screenshot Capture</span>
			</div>
			<div class="flex items-center gap-2">
				<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
				</svg>
				<span class="text-xs text-gray-500 font-mono">Meeting Recording</span>
			</div>
		</div>
	</div>
</section>

<!-- Capabilities -->
<section class="py-20 px-6">
	<div class="max-w-5xl mx-auto">
		<div class="grid grid-cols-2 md:grid-cols-4 gap-8">
			{#each capabilities as cap, i}
				<div in:fly={{ y: 30, duration: 500, delay: 100 * i }}>
					<h3 class="font-mono text-sm font-bold text-black mb-2 tracking-wider">{cap.title.toUpperCase()}</h3>
					<p class="text-gray-500 text-sm">{cap.desc}</p>
				</div>
			{/each}
		</div>
	</div>
</section>

<!-- Modules Grid -->
<section class="py-20 px-6">
	<div class="max-w-5xl mx-auto">
		<div class="flex items-center justify-between mb-8">
			<h2 class="font-mono text-xs text-gray-400 tracking-wider">AVAILABLE MODULES</h2>
			<a href="/docs" class="text-xs text-gray-500 hover:text-black transition-colors font-mono flex items-center gap-1">
				View all docs
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</a>
		</div>
		<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-4">
			{#each modules as mod, i}
				<a
					href="/docs/{mod.slug}"
					class="border border-gray-200 rounded-lg p-4 hover:border-gray-400 hover:bg-gray-50 hover:shadow-sm transition-all group block"
					in:fly={{ y: 20, duration: 400, delay: 50 * i }}
				>
					<div class="font-medium text-gray-900 text-sm mb-1 group-hover:text-black">{mod.name}</div>
					<div class="text-xs text-gray-400 font-mono">{mod.desc}</div>
				</a>
			{/each}
		</div>
	</div>
</section>

<!-- How It Works -->
<section class="py-24 px-6 bg-gray-50">
	<div class="max-w-5xl mx-auto">
		<div class="text-center mb-16">
			<h2 class="font-mono text-xs text-gray-400 mb-4 tracking-wider">HOW IT WORKS</h2>
			<h3 class="text-3xl md:text-4xl font-bold text-black mb-4">Context-driven intelligence</h3>
			<p class="text-gray-500 max-w-2xl mx-auto">
				Business OS uses a context-first approach. Your data, documents, and knowledge inform AI agents that understand your business.
			</p>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
			{#each howItWorks as item, i}
				<div
					class="relative bg-white border border-gray-200 rounded-xl p-6 hover:border-gray-300 hover:shadow-lg transition-all group"
					in:fly={{ y: 30, duration: 500, delay: 100 * i }}
				>
					<div class="flex items-center gap-3 mb-4">
						<div class="w-10 h-10 bg-black rounded-lg flex items-center justify-center group-hover:scale-110 transition-transform">
							<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={item.icon} />
							</svg>
						</div>
						<span class="font-mono text-xs text-gray-400">{item.step}</span>
					</div>
					<h4 class="font-bold text-black mb-2">{item.title}</h4>
					<p class="text-sm text-gray-500 leading-relaxed">{item.desc}</p>
					{#if i < howItWorks.length - 1}
						<div class="hidden lg:block absolute -right-3 top-1/2 transform -translate-y-1/2 text-gray-300">
							<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
							</svg>
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Flow diagram -->
		<div class="mt-16 bg-white border border-gray-200 rounded-xl p-8 md:p-12">
			<div class="flex flex-col md:flex-row items-center justify-between gap-8">
				<div class="text-center md:text-left">
					<div class="font-mono text-xs text-gray-400 mb-2">THE FLOW</div>
					<h4 class="text-xl font-bold text-black mb-3">From data to action</h4>
					<p class="text-gray-500 text-sm max-w-md">
						Import your existing data, create contexts for each domain, link them to projects, and let AI agents handle the rest.
					</p>
				</div>
				<div class="flex items-center gap-3 font-mono text-sm">
					<div class="px-4 py-2 bg-gray-100 rounded-lg text-gray-600">Data</div>
					<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
					<div class="px-4 py-2 bg-gray-100 rounded-lg text-gray-600">Context</div>
					<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
					<div class="px-4 py-2 bg-gray-100 rounded-lg text-gray-600">Project</div>
					<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
					<div class="px-4 py-2 bg-black rounded-lg text-white">Agent</div>
				</div>
			</div>
		</div>
	</div>
</section>

<!-- aMCP Integrations -->
<section class="py-24 px-6">
	<div class="max-w-5xl mx-auto">
		<div class="grid md:grid-cols-2 gap-12 items-center">
			<div>
				<div class="font-mono text-xs text-gray-400 mb-4 tracking-wider">AMCP PROTOCOL</div>
				<h3 class="text-3xl font-bold text-black mb-4">Connect everything via aMCP</h3>
				<p class="text-gray-500 mb-6">
					Business OS uses <a href="https://amcp.ai" target="_blank" rel="noopener" class="text-black hover:underline font-medium">aMCP</a> (Advanced Model Context Protocol) to integrate with your existing tools.
				</p>
				<div class="space-y-4">
					{#each [
						{ title: 'AI-Generated Integrations', desc: 'OSA can build custom aMCP integrations for your legacy systems' },
						{ title: 'Enterprise Scale', desc: 'Handle thousands of connections with proper rate limiting and auth' },
						{ title: 'Bi-directional Sync', desc: 'Keep data synchronized across all your connected systems' }
					] as feature}
						<div class="flex items-start gap-3">
							<div class="w-6 h-6 bg-green-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
								<svg class="w-3.5 h-3.5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							</div>
							<div>
								<div class="font-medium text-black text-sm">{feature.title}</div>
								<div class="text-gray-500 text-sm">{feature.desc}</div>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- Integrations grid -->
			<div class="bg-gray-50 rounded-xl p-6 border border-gray-200">
				<div class="font-mono text-xs text-gray-400 mb-4">SUPPORTED INTEGRATIONS</div>
				<div class="grid grid-cols-3 gap-3">
					{#each integrations as int, i}
						<div
							class="bg-white border border-gray-200 rounded-lg p-3 hover:border-gray-400 hover:shadow-sm transition-all text-center"
							in:fly={{ y: 10, duration: 300, delay: 50 * i }}
						>
							<div class="font-medium text-gray-900 text-xs">{int.name}</div>
							<div class="text-[10px] text-gray-400 font-mono mt-1">{int.category}</div>
						</div>
					{/each}
				</div>
				<div class="mt-4 text-center">
					<a href="https://amcp.ai" target="_blank" rel="noopener" class="text-xs text-gray-400 font-mono hover:text-gray-600 transition-colors">+ Custom integrations via aMCP →</a>
				</div>
			</div>
		</div>
	</div>
</section>
