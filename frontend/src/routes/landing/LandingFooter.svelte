<script lang="ts">
	// No props needed — this is a static footer
</script>

<!-- Data Import -->
<section class="py-24 px-6 bg-black text-white">
	<div class="max-w-5xl mx-auto">
		<div class="text-center mb-12">
			<div class="font-mono text-xs text-gray-500 mb-4 tracking-wider">DATA MIGRATION</div>
			<h3 class="text-3xl font-bold mb-4">Import your existing data</h3>
			<p class="text-gray-400 max-w-2xl mx-auto">
				Migrate from your current tools in minutes. We support direct imports from popular platforms and custom data formats.
			</p>
		</div>

		<div class="grid md:grid-cols-3 gap-6">
			<div class="bg-gray-900 border border-gray-800 rounded-xl p-6">
				<div class="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center mb-4">
					<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
					</svg>
				</div>
				<h4 class="font-bold text-white mb-2">CRM Data</h4>
				<p class="text-gray-400 text-sm mb-4">Import contacts, deals, and activities from Salesforce, HubSpot, or GHL.</p>
				<div class="flex flex-wrap gap-2">
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">Contacts</span>
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">Deals</span>
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">Activities</span>
				</div>
			</div>

			<div class="bg-gray-900 border border-gray-800 rounded-xl p-6">
				<div class="w-10 h-10 bg-green-500/20 rounded-lg flex items-center justify-center mb-4">
					<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
					</svg>
				</div>
				<h4 class="font-bold text-white mb-2">Spreadsheets & Databases</h4>
				<p class="text-gray-400 text-sm mb-4">Migrate from Airtable, Google Sheets, Excel, or connect directly to databases.</p>
				<div class="flex flex-wrap gap-2">
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">CSV</span>
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">JSON</span>
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">SQL</span>
				</div>
			</div>

			<div class="bg-gray-900 border border-gray-800 rounded-xl p-6">
				<div class="w-10 h-10 bg-purple-500/20 rounded-lg flex items-center justify-center mb-4">
					<svg class="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
				</div>
				<h4 class="font-bold text-white mb-2">Documents & Files</h4>
				<p class="text-gray-400 text-sm mb-4">Import documents from Notion, Google Docs, or upload files directly.</p>
				<div class="flex flex-wrap gap-2">
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">PDF</span>
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">Markdown</span>
					<span class="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400">Docs</span>
				</div>
			</div>
		</div>

		<div class="mt-12 text-center">
			<div class="inline-flex items-center gap-4 bg-gray-900 border border-gray-800 rounded-full px-6 py-3">
				<span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
				<span class="font-mono text-sm text-gray-400">AI-assisted mapping automatically matches your data schema</span>
			</div>
		</div>
	</div>
</section>

<!-- OSA & Agent Capabilities Section -->
<section class="py-24 px-6 bg-gray-50">
	<div class="max-w-5xl mx-auto">
		<div class="grid md:grid-cols-2 gap-12 items-start">
			<!-- OSA Info -->
			<div class="bg-white border border-gray-200 rounded-xl p-8">
				<div class="font-mono text-xs text-gray-400 mb-4 tracking-wider">POWERED BY</div>
				<h3 class="text-2xl font-bold text-black mb-4">OSA - The OS Agent</h3>
				<p class="text-gray-500 mb-6">
					Business OS was built with OSA. The same agent that created this system can extend it for your needs.
				</p>
				<div class="space-y-3 mb-6">
					{#each [
						'Auto-generate aMCP integrations for legacy systems',
						'Create custom modules and workflows',
						'Extend the system without developers'
					] as item}
						<div class="flex items-center gap-3 text-sm">
							<svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
							<span class="text-gray-600">{item}</span>
						</div>
					{/each}
				</div>
				<a
					href="https://osa.dev"
					target="_blank"
					rel="noopener"
					class="inline-flex items-center gap-2 text-sm font-medium text-black hover:text-gray-600 transition-colors font-mono"
				>
					Learn more at osa.dev
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
					</svg>
				</a>
			</div>

			<!-- Enterprise Features -->
			<div class="space-y-6">
				{#each [
					{ icon: 'M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z', title: 'Enterprise Security', desc: 'SOC2 ready. Self-hosted on your infrastructure with full data sovereignty.' },
					{ icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z', title: 'Team Collaboration', desc: 'Role-based access, shared contexts, and team workspaces built for scale.' },
					{ icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z', title: 'Local or Cloud LLMs', desc: 'Run Ollama locally or connect to OpenAI, Anthropic, or your preferred provider.' }
				] as feature}
					<div class="bg-white border border-gray-200 rounded-xl p-6">
						<div class="flex items-start gap-4">
							<div class="w-10 h-10 bg-black rounded-lg flex items-center justify-center flex-shrink-0">
								<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={feature.icon} />
								</svg>
							</div>
							<div>
								<h4 class="font-bold text-black mb-1">{feature.title}</h4>
								<p class="text-gray-500 text-sm">{feature.desc}</p>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</section>

<!-- CTA -->
<section class="py-24 px-6 bg-black text-white">
	<div class="max-w-3xl mx-auto text-center">
		<div class="font-mono text-xs text-gray-500 mb-4 tracking-wider">GET STARTED</div>
		<h2 class="text-3xl md:text-4xl font-bold mb-4">Ready to take control?</h2>
		<p class="text-gray-400 mb-8 max-w-xl mx-auto">
			Deploy on your infrastructure, import your data, and let AI agents handle the rest.
		</p>
		<div class="flex flex-col sm:flex-row items-center justify-center gap-4">
			<a href="/register" class="inline-flex items-center gap-2 bg-white text-black px-8 py-3 rounded-lg text-sm font-medium hover:bg-gray-100 transition-colors font-mono">
				Initialize workspace
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
				</svg>
			</a>
			<a href="https://github.com" target="_blank" rel="noopener" class="inline-flex items-center gap-2 border border-gray-700 text-gray-300 px-6 py-3 rounded-lg text-sm font-medium hover:border-gray-500 hover:text-white transition-colors font-mono">
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
				View on GitHub
			</a>
		</div>
		<div class="mt-12 flex flex-wrap justify-center gap-8 font-mono text-xs text-gray-500">
			{#each ['Self-hosted', 'End-to-end encrypted', 'Open source', 'MIT Licensed'] as badge}
				<div class="flex items-center gap-2">
					<svg class="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					{badge}
				</div>
			{/each}
		</div>
	</div>
</section>

<!-- Footer -->
<footer class="border-t border-gray-100 py-12 px-6">
	<div class="max-w-4xl mx-auto">
		<div class="flex flex-col md:flex-row items-center justify-between gap-6">
			<div class="flex items-baseline gap-0.5">
				<span class="text-black text-sm font-extrabold tracking-[0.1em] font-mono">BUSINESS</span>
				<span class="text-black/30 text-xs font-light font-mono">OS</span>
			</div>
			<div class="flex items-center gap-6 font-mono text-xs text-gray-400">
				<a href="/docs" class="hover:text-black transition-colors">Docs</a>
				<a href="https://github.com" target="_blank" rel="noopener" class="hover:text-black transition-colors">GitHub</a>
				<a href="https://amcp.ai" target="_blank" rel="noopener" class="hover:text-black transition-colors">aMCP</a>
				<a href="https://osa.dev" target="_blank" rel="noopener" class="hover:text-black transition-colors">OSA</a>
				<a href="/terms" class="hover:text-black transition-colors">Terms</a>
				<a href="/privacy" class="hover:text-black transition-colors">Privacy</a>
			</div>
		</div>
		<div class="mt-8 text-center md:text-left">
			<p class="font-mono text-xs text-gray-400">
				Open Source · MIT Licensed · v0.0.1
			</p>
		</div>
	</div>
</footer>
