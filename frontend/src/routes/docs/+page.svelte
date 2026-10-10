<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { useSession } from '$lib/auth-client';

	const session = useSession();
	let showContent = $state(false);

	onMount(() => {
		setTimeout(() => (showContent = true), 100);
	});

	const modules = [
		{
			name: 'Desktop',
			slug: 'desktop',
			desc: 'Native app for macOS, Windows & Linux with voice commands, global shortcuts, and offline support.',
			icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z'
		},
		{
			name: 'Dashboard',
			slug: 'dashboard',
			desc: 'Your command center with widgets, quick actions, and real-time metrics.',
			icon: 'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z'
		},
		{
			name: 'Chat',
			slug: 'chat',
			desc: 'AI-powered conversations with context awareness and tool integration via aMCP.',
			icon: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z'
		},
		{
			name: 'Tasks',
			slug: 'tasks',
			desc: 'Track work with priorities, due dates, subtasks, and AI-assisted task management.',
			icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4'
		},
		{
			name: 'Projects',
			slug: 'projects',
			desc: 'Organize work into projects with milestones, team assignments, and progress tracking.',
			icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10'
		},
		{
			name: 'Calendar',
			slug: 'calendar',
			desc: 'Schedule events, meetings, and deadlines with smart reminders and sync.',
			icon: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z'
		},
		{
			name: 'Clients',
			slug: 'clients',
			desc: 'CRM for managing client relationships, contacts, and communication history.',
			icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z'
		},
		{
			name: 'Contexts',
			slug: 'contexts',
			desc: 'Knowledge bases that inform AI agents. Upload documents, notes, and data.',
			icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10'
		},
		{
			name: 'Nodes',
			slug: 'nodes',
			desc: 'Graph visualization of connections between clients, projects, and contexts.',
			icon: 'M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1'
		},
		{
			name: 'Daily Log',
			slug: 'daily-log',
			desc: 'Journal your work, thoughts, and progress with timestamped entries.',
			icon: 'M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z'
		}
	];

	const shortcuts = [
		{ keys: ['⌘', 'Space'], action: 'Open Spotlight', desc: 'Quick access to everything' },
		{ keys: ['⌘', '1-5'], action: 'Switch Module', desc: 'Dashboard, Chat, Tasks...' },
		{ keys: ['⌘', 'W'], action: 'Close Window', desc: 'Close focused window' },
		{ keys: ['⌘', '`'], action: 'Cycle Windows', desc: 'Switch between windows' },
		{ keys: ['⌘', '⇧', 'T'], action: 'New Task', desc: 'Quick task creation' },
		{ keys: ['Ctrl', 'Alt', '←/→'], action: 'Snap Window', desc: 'Split screen' },
	];

	const howItWorks = [
		{ step: '01', title: 'Create Contexts', desc: 'Upload documents, add notes, connect data sources.' },
		{ step: '02', title: 'Organize Projects', desc: 'Group work, link contexts, assign team members.' },
		{ step: '03', title: 'Chat with AI', desc: 'Your AI understands your contexts. Ask anything.' },
		{ step: '04', title: 'Automate via aMCP', desc: 'Connect external tools. AI handles integrations.' },
	];

	const techStack = [
		{ category: 'Frontend', items: ['SvelteKit 5', 'TypeScript', 'Tailwind', 'Vite'] },
		{ category: 'Backend', items: ['Go', 'PostgreSQL', 'Redis', 'SQLc'] },
		{ category: 'Desktop', items: ['Electron', 'System Tray', 'Global Shortcuts'] },
		{ category: 'AI', items: ['Ollama', 'OpenAI', 'Anthropic', 'Local LLMs'] },
		{ category: 'Integration', items: ['aMCP Protocol', 'REST APIs', 'WebSockets'] },
		{ category: 'Infra', items: ['Docker', 'GCP', 'Self-hosted', 'E2B'] }
	];
</script>

<svelte:head>
	<title>Documentation - Business OS</title>
</svelte:head>

<div class="min-h-screen bg-white">
	<!-- Grid background -->
	{#if showContent}
		<div class="fixed inset-0 pointer-events-none" in:fade={{ duration: 1000 }}>
			<div
				class="w-full h-full opacity-[0.04]"
				style="
					background-image:
						linear-gradient(rgba(0,0,0,0.3) 1px, transparent 1px),
						linear-gradient(90deg, rgba(0,0,0,0.3) 1px, transparent 1px);
					background-size: 60px 60px;
				"
			></div>
		</div>
	{/if}

	<!-- Header -->
	<header class="border-b border-gray-200 bg-white/95 backdrop-blur-sm sticky top-0 z-50">
		<div class="max-w-6xl mx-auto px-6 h-14 flex items-center justify-between">
			<div class="flex items-center gap-4">
				<a href="/" class="flex items-baseline gap-0.5">
					<span class="text-black text-lg font-extrabold tracking-[0.15em] font-mono">BUSINESS</span>
					<span class="text-black/30 text-base font-light font-mono">OS</span>
				</a>
				<span class="text-gray-300 font-mono">/</span>
				<span class="font-mono text-xs text-gray-500">docs</span>
			</div>
			<div class="flex items-center gap-4">
				<a href="https://github.com" target="_blank" rel="noopener" class="text-xs text-gray-400 hover:text-black transition-colors font-mono">
					GitHub
				</a>
				{#if $session.data}
					<a href="/window" class="bg-black text-white px-4 py-2 rounded-lg text-xs font-medium hover:bg-gray-800 transition-colors font-mono flex items-center gap-2">
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
						</svg>
						Back to Desktop
					</a>
				{:else}
					<a href="/register" class="bg-black text-white px-4 py-2 rounded-lg text-xs font-medium hover:bg-gray-800 transition-colors font-mono">
						Get Started
					</a>
				{/if}
			</div>
		</div>
	</header>

	{#if showContent}
		<!-- Hero -->
		<section class="py-16 px-6 border-b border-gray-200">
			<div class="max-w-6xl mx-auto">
				<div in:fly={{ y: 20, duration: 500 }}>
					<div class="font-mono text-xs text-gray-400 mb-4 tracking-wider">> DOCUMENTATION</div>
					<h1 class="text-4xl font-bold text-black mb-4 font-mono tracking-tight">
						Learn <span class="tracking-[0.1em]">BUSINESS</span><span class="text-gray-300">OS</span>
					</h1>
					<p class="text-gray-500 max-w-xl font-mono text-sm">
						Everything you need to deploy, configure, and master your self-hosted business operating system.
					</p>
				</div>
			</div>
		</section>

		<!-- How It Works -->
		<section class="py-16 px-6 bg-gray-50 border-b border-gray-200">
			<div class="max-w-6xl mx-auto">
				<div class="mb-10" in:fly={{ y: 20, duration: 500, delay: 100 }}>
					<div class="font-mono text-xs text-gray-400 mb-2 tracking-wider">> THE_FLOW</div>
					<h2 class="text-xl font-bold text-black font-mono">How It Works</h2>
				</div>

				<div class="grid md:grid-cols-4 gap-4">
					{#each howItWorks as item, i}
						<div
							class="relative bg-white border border-gray-200 p-5 hover:border-gray-400 transition-colors group"
							in:fly={{ y: 20, duration: 400, delay: 150 + i * 50 }}
						>
							<div class="font-mono text-3xl font-bold text-gray-100 mb-3 group-hover:text-gray-200 transition-colors">{item.step}</div>
							<h3 class="font-bold text-black mb-1 text-sm font-mono">{item.title}</h3>
							<p class="text-xs text-gray-500 font-mono leading-relaxed">{item.desc}</p>
							{#if i < howItWorks.length - 1}
								<div class="hidden md:block absolute -right-2 top-1/2 -translate-y-1/2 text-gray-300 font-mono">→</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- Desktop & Voice Agent -->
		<section class="py-16 px-6 bg-black text-white border-b border-gray-800">
			<div class="max-w-6xl mx-auto">
				<div class="grid lg:grid-cols-2 gap-12 items-center">
					<div in:fly={{ y: 20, duration: 500, delay: 200 }}>
						<div class="font-mono text-xs text-gray-500 mb-2 tracking-wider">> OS_AGENT</div>
						<h2 class="text-2xl font-bold mb-4 font-mono">Your Agent, Everywhere</h2>
						<p class="text-gray-400 font-mono text-sm mb-8 leading-relaxed">
							Business OS Desktop runs in your system tray on macOS, Windows, and Linux.
							Access your AI assistant instantly with global shortcuts—from anywhere on your machine.
						</p>

						<div class="space-y-4">
							<div class="flex items-start gap-3 border border-gray-800 p-4 hover:border-gray-600 transition-colors">
								<div class="w-8 h-8 border border-gray-700 flex items-center justify-center flex-shrink-0">
									<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
									</svg>
								</div>
								<div>
									<h3 class="font-mono text-sm text-white mb-1">voice_commands</h3>
									<p class="text-xs text-gray-500 font-mono">Speak to your OS Agent. Dictate tasks, ask questions, create content.</p>
								</div>
							</div>

							<div class="flex items-start gap-3 border border-gray-800 p-4 hover:border-gray-600 transition-colors">
								<div class="w-8 h-8 border border-gray-700 flex items-center justify-center flex-shrink-0">
									<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
									</svg>
								</div>
								<div>
									<h3 class="font-mono text-sm text-white mb-1">global_hotkeys</h3>
									<p class="text-xs text-gray-500 font-mono">Press your shortcut from anywhere. Instantly summon Business OS.</p>
								</div>
							</div>

							<div class="flex items-start gap-3 border border-gray-800 p-4 hover:border-gray-600 transition-colors">
								<div class="w-8 h-8 border border-gray-700 flex items-center justify-center flex-shrink-0">
									<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
									</svg>
								</div>
								<div>
									<h3 class="font-mono text-sm text-white mb-1">terminal_agent</h3>
									<p class="text-xs text-gray-500 font-mono">Natural language commands execute real actions on your system.</p>
								</div>
							</div>
						</div>

						<!-- Platform badges -->
						<div class="flex items-center gap-3 mt-8">
							<span class="font-mono text-xs text-gray-600">platforms:</span>
							<span class="font-mono text-xs text-gray-400 border border-gray-800 px-2 py-1">macOS</span>
							<span class="font-mono text-xs text-gray-400 border border-gray-800 px-2 py-1">Windows</span>
							<span class="font-mono text-xs text-gray-400 border border-gray-800 px-2 py-1">Linux</span>
						</div>
					</div>

					<!-- Chat mock wrapper — forced dark context so classes resolve correctly in both light and dark app themes -->
					<div class="dark relative" in:fly={{ y: 20, duration: 500, delay: 250 }}>
						<!-- Chat mock - intentionally dark UI demo -->
						<div class="bg-[#1a1a1a] rounded-xl overflow-hidden shadow-2xl border border-gray-800">
							<!-- Window header -->
							<div class="flex items-center gap-2 px-4 py-3 bg-[#252525] border-b border-gray-800">
								<div class="w-3 h-3 rounded-full bg-[#3a3a3a]"></div>
								<div class="w-3 h-3 rounded-full bg-[#3a3a3a]"></div>
								<div class="w-3 h-3 rounded-full bg-[#3a3a3a]"></div>
								<span class="ml-3 text-sm text-gray-400 font-mono">OS Agent</span>
							</div>

							<!-- Chat messages -->
							<div class="p-5 space-y-4 min-h-[200px]">
								<!-- User message -->
								<div class="flex justify-end">
									<div class="bg-[#2d2d2d] text-white rounded-2xl rounded-br-sm px-4 py-2.5">
										<p class="text-sm">Create a task for tomorrow's meeting prep</p>
									</div>
								</div>

								<!-- AI response -->
								<div class="flex justify-start">
									<div class="bg-[#f5f5f5] text-gray-900 rounded-2xl rounded-bl-sm px-4 py-3 max-w-[90%]">
										<p class="text-sm mb-3">Done. I've created the task for you.</p>
										<!-- Task card -->
										<div class="bg-white border border-gray-200 rounded-lg p-3 shadow-sm">
											<div class="flex items-center gap-2.5 mb-1.5">
												<div class="w-4 h-4 border-2 border-gray-300 rounded"></div>
												<span class="text-sm font-medium text-gray-900">Meeting prep</span>
											</div>
											<div class="flex items-center gap-2 text-xs text-gray-500 ml-6">
												<span>Tomorrow 9:00 AM</span>
												<span class="px-1.5 py-0.5 bg-gray-900 text-white rounded text-[10px] font-medium">HIGH</span>
											</div>
										</div>
									</div>
								</div>
							</div>

							<!-- Input area -->
							<div class="px-4 py-3 bg-[#1a1a1a] border-t border-gray-800">
								<div class="flex items-center gap-3">
									<div class="flex-1 bg-[#252525] border border-gray-700 rounded-xl px-4 py-2.5 flex items-center gap-3">
										<svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
										</svg>
										<span class="text-sm text-gray-500">Ask anything...</span>
									</div>
									<button class="btn-pill btn-pill-primary btn-pill-icon">
										<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
										</svg>
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</section>

		<!-- Modules -->
		<section class="py-16 px-6">
			<div class="max-w-6xl mx-auto">
				<div class="mb-10" in:fly={{ y: 20, duration: 500, delay: 300 }}>
					<div class="font-mono text-xs text-gray-400 mb-2 tracking-wider">> MODULES</div>
					<h2 class="text-xl font-bold text-black font-mono">Core Features</h2>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					{#each modules as mod, i}
						<a
							href="/docs/{mod.slug}"
							class="group border border-gray-200 p-5 hover:border-black hover:bg-gray-50 transition-all"
							in:fly={{ y: 20, duration: 400, delay: 350 + i * 30 }}
						>
							<div class="flex items-start gap-4">
								<div class="w-10 h-10 border border-gray-200 group-hover:border-black group-hover:bg-black flex items-center justify-center flex-shrink-0 transition-all">
									<svg class="w-5 h-5 text-gray-400 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={mod.icon} />
									</svg>
								</div>
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 mb-1">
										<h3 class="font-mono text-sm font-bold text-black">{mod.name}</h3>
										<span class="text-gray-300 group-hover:text-black group-hover:translate-x-1 transition-all font-mono">→</span>
									</div>
									<p class="text-xs text-gray-500 font-mono leading-relaxed">{mod.desc}</p>
								</div>
							</div>
						</a>
					{/each}
				</div>
			</div>
		</section>

		<!-- Keyboard Shortcuts -->
		<section class="py-16 px-6 bg-gray-50 border-y border-gray-200">
			<div class="max-w-6xl mx-auto">
				<div class="mb-10" in:fly={{ y: 20, duration: 500, delay: 400 }}>
					<div class="font-mono text-xs text-gray-400 mb-2 tracking-wider">> SHORTCUTS</div>
					<h2 class="text-xl font-bold text-black font-mono">Keyboard Commands</h2>
				</div>

				<div class="grid md:grid-cols-2 lg:grid-cols-3 gap-3">
					{#each shortcuts as shortcut, i}
						<div
							class="bg-white border border-gray-200 p-4 hover:border-gray-400 transition-colors"
							in:fly={{ y: 20, duration: 400, delay: 450 + i * 30 }}
						>
							<div class="flex items-center gap-1 mb-2">
								{#each shortcut.keys as key}
									<kbd class="px-2 py-1 bg-gray-100 border border-gray-200 text-xs font-mono">{key}</kbd>
									{#if key !== shortcut.keys[shortcut.keys.length - 1]}
										<span class="text-gray-300 text-xs">+</span>
									{/if}
								{/each}
							</div>
							<div class="font-mono text-xs text-black font-medium">{shortcut.action}</div>
							<div class="font-mono text-xs text-gray-400">{shortcut.desc}</div>
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- Tech Stack -->
		<section class="py-16 px-6 border-b border-gray-200">
			<div class="max-w-6xl mx-auto">
				<div class="mb-10" in:fly={{ y: 20, duration: 500, delay: 500 }}>
					<div class="font-mono text-xs text-gray-400 mb-2 tracking-wider">> TECH_STACK</div>
					<h2 class="text-xl font-bold text-black font-mono">Built With</h2>
				</div>

				<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
					{#each techStack as stack, i}
						<div
							class="border border-gray-200 p-4 hover:border-gray-400 transition-colors"
							in:fly={{ y: 20, duration: 400, delay: 550 + i * 30 }}
						>
							<div class="font-mono text-xs text-gray-400 mb-3">{stack.category.toLowerCase()}/</div>
							<ul class="space-y-1">
								{#each stack.items as item}
									<li class="font-mono text-xs text-gray-600">{item}</li>
								{/each}
							</ul>
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- Quick Links -->
		<section class="py-16 px-6">
			<div class="max-w-6xl mx-auto">
				<div class="grid md:grid-cols-3 gap-4">
					<a
						href="https://amcp.ai"
						target="_blank"
						rel="noopener"
						class="group border border-gray-200 p-6 hover:border-black hover:bg-black transition-all"
						in:fly={{ y: 20, duration: 500, delay: 600 }}
					>
						<div class="font-mono text-xs text-gray-400 group-hover:text-gray-500 mb-2">> integration</div>
						<h3 class="font-mono text-lg font-bold text-black group-hover:text-white mb-2">aMCP Protocol</h3>
						<p class="font-mono text-xs text-gray-500 group-hover:text-gray-400 mb-4">Advanced Model Context Protocol for AI integrations.</p>
						<div class="flex items-center gap-2 font-mono text-xs text-gray-400 group-hover:text-white">
							amcp.ai <span class="group-hover:translate-x-1 transition-transform">→</span>
						</div>
					</a>

					<a
						href="https://osa.dev"
						target="_blank"
						rel="noopener"
						class="group border border-gray-200 p-6 hover:border-black hover:bg-black transition-all"
						in:fly={{ y: 20, duration: 500, delay: 650 }}
					>
						<div class="font-mono text-xs text-gray-400 group-hover:text-gray-500 mb-2">> platform</div>
						<h3 class="font-mono text-lg font-bold text-black group-hover:text-white mb-2">OSA Platform</h3>
						<p class="font-mono text-xs text-gray-500 group-hover:text-gray-400 mb-4">Enterprise AI orchestration and deployment.</p>
						<div class="flex items-center gap-2 font-mono text-xs text-gray-400 group-hover:text-white">
							osa.dev <span class="group-hover:translate-x-1 transition-transform">→</span>
						</div>
					</a>

					<a
						href="https://github.com"
						target="_blank"
						rel="noopener"
						class="group border border-gray-200 p-6 hover:border-black hover:bg-black transition-all"
						in:fly={{ y: 20, duration: 500, delay: 700 }}
					>
						<div class="font-mono text-xs text-gray-400 group-hover:text-gray-500 mb-2">> source</div>
						<h3 class="font-mono text-lg font-bold text-black group-hover:text-white mb-2">Open Source</h3>
						<p class="font-mono text-xs text-gray-500 group-hover:text-gray-400 mb-4">View source, contribute, self-host.</p>
						<div class="flex items-center gap-2 font-mono text-xs text-gray-400 group-hover:text-white">
							github.com <span class="group-hover:translate-x-1 transition-transform">→</span>
						</div>
					</a>
				</div>
			</div>
		</section>

		<!-- Footer -->
		<footer class="border-t border-gray-200 py-10 px-6">
			<div class="max-w-6xl mx-auto">
				<div class="flex flex-col md:flex-row items-center justify-between gap-6">
					<div class="flex items-baseline gap-0.5">
						<span class="text-black text-sm font-extrabold tracking-[0.1em] font-mono">BUSINESS</span>
						<span class="text-black/30 text-xs font-light font-mono">OS</span>
						<span class="text-gray-300 text-xs ml-3 font-mono">/ docs</span>
					</div>
					<div class="flex items-center gap-6 font-mono text-xs text-gray-400">
						<a href="/" class="hover:text-black transition-colors">home</a>
						<a href="/docs" class="hover:text-black transition-colors">docs</a>
						<a href="https://github.com" target="_blank" rel="noopener" class="hover:text-black transition-colors">github</a>
						<a href="https://amcp.ai" target="_blank" rel="noopener" class="hover:text-black transition-colors">amcp</a>
					</div>
				</div>
			</div>
		</footer>
	{/if}
</div>
