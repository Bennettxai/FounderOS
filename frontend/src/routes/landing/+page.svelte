<script lang="ts">
	import { onMount } from 'svelte';
	import Grainient from '$lib/components/landing/Grainient.svelte';

	let scrolled = $state(false);
	let visible = $state(false);

	onMount(() => {
		visible = true;
		const handleScroll = () => { scrolled = window.scrollY > 20; };
		window.addEventListener('scroll', handleScroll);
		return () => window.removeEventListener('scroll', handleScroll);
	});

	const modules = [
		{
			name: 'Dashboard',
			desc: 'Command center for your business',
			icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6'
		},
		{
			name: 'Chat',
			desc: 'AI assistant that knows your business',
			icon: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z'
		},
		{
			name: 'Projects',
			desc: 'Track work from start to finish',
			icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01'
		},
		{
			name: 'CRM',
			desc: 'Manage clients, deals, and pipeline',
			icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z'
		},
		{
			name: 'Tasks',
			desc: 'Organize and prioritize your work',
			icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
		},
		{
			name: 'Team',
			desc: 'Manage your people and capacity',
			icon: 'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z'
		},
		{
			name: 'Terminal',
			desc: 'Full cloud computer with Claude Code',
			icon: 'M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z'
		},
		{
			name: 'Pages',
			desc: 'Wiki and knowledge base',
			icon: 'M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253'
		},
		{
			name: 'Agents',
			desc: 'Custom AI agents for any task',
			icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z'
		}
	];

	const computerFeatures = [
		'Bring your own API keys — use any model',
		'Pairs with Claude Code, Codex, OSA-Agent, and more',
		'Schedule agents to run automations 24/7',
		'Full Linux desktop — your environment, your rules'
	];

	const agentRuntimes = [
		'Claude Code', 'Codex', 'OSA-Agent', 'Mes-Agent', 'Devin', 'Custom Agents'
	];

	const hierarchy = [
		{ level: 'Company', desc: 'Your organization. One source of truth.', icon: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4' },
		{ level: 'Teams', desc: 'People + AI agents working together.', icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z' },
		{ level: 'Projects', desc: 'Work streams with tasks and milestones.', icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2' },
		{ level: 'Signals', desc: 'Every action produces a signal. The system learns.', icon: 'M13 10V3L4 14h7v7l9-11h-7z' },
	];

	const plans = [
		{
			name: 'Free',
			price: '$0',
			period: '',
			desc: 'Local only. No credit card.',
			features: ['Local app only', 'Unlimited modules', 'Bring your own API keys', 'Apache 2.0 licensed'],
			cta: 'Download Free',
			href: '#download',
			highlight: false
		},
		{
			name: 'Pro',
			price: '$40',
			period: '/mo',
			desc: 'Cloud computer + credits',
			features: ['4 GB cloud computer', '500 AI credits/mo', 'Unlimited seats', 'Cloud sync'],
			cta: 'Start Pro',
			href: '/register',
			highlight: false
		},
		{
			name: 'Growth',
			price: '$100',
			period: '/mo',
			desc: 'More power, more credits',
			features: ['8 GB cloud computer', '2,000 AI credits/mo', 'Unlimited seats', 'Priority support'],
			cta: 'Start Growth',
			href: '/register',
			highlight: true,
			badge: 'Popular'
		},
		{
			name: 'Business',
			price: '$200',
			period: '/mo',
			desc: 'Enterprise-grade everything',
			features: ['16 GB cloud computer', '5,000 AI credits/mo', 'Unlimited seats', 'SSO + audit logs'],
			cta: 'Start Business',
			href: '/register',
			highlight: false
		}
	];
</script>

<svelte:head>
	<title>BusinessOS — Your Business, One Operating System</title>
	<meta name="description" content="The AI-native platform that runs your entire business. CRM, projects, tasks, agents, and a cloud computer — all in one desktop app." />
</svelte:head>

<!-- Full-page Grainient background -->
<div class="lp-grainient-bg" aria-hidden="true">
	<Grainient
		color1="#4f4f4f"
		color2="#000000"
		color3="#c4c4c4"
		timeSpeed={0.25}
		colorBalance={0}
		warpStrength={1}
		warpFrequency={5}
		warpSpeed={2}
		warpAmplitude={50}
		blendAngle={0}
		blendSoftness={0.05}
		rotationAmount={500}
		noiseScale={2}
		grainAmount={0.1}
		grainScale={2}
		grainAnimated={false}
		contrast={1.5}
		gamma={1}
		saturation={1}
		centerX={0}
		centerY={0}
		zoom={0.9}
	/>
</div>

<!-- Nav -->
<header class="lp-nav {scrolled ? 'lp-nav--scrolled' : ''}">
	<div class="lp-container lp-nav__inner">
		<a href="/landing" class="lp-logo" aria-label="BusinessOS home">
			<span class="lp-logo__word">BUSINESS</span><span class="lp-logo__suffix">OS</span>
		</a>
		<nav class="lp-nav__links" aria-label="Site navigation">
			<a href="#modules" class="lp-nav__link">Modules</a>
			<a href="#pricing" class="lp-nav__link">Pricing</a>
			<a href="https://github.com/Miosa-osa/businessos" target="_blank" rel="noopener" class="lp-nav__link">Docs</a>
			<a href="https://github.com/miosa-osa/businessos" target="_blank" rel="noopener" class="lp-nav__link">GitHub</a>
		</nav>
		<div class="lp-nav__actions">
			<a href="/login" class="lp-btn lp-btn--ghost">Sign in</a>
			<a href="/register" class="lp-btn lp-btn--primary">Get Started</a>
		</div>
	</div>
</header>

<main>
	<!-- Section 1: Video in app frame -->
	<section class="lp-video-hero">
		<div class="lp-video-hero__container">
			<div class="lp-video-hero__frame">
				<div class="lp-video-hero__titlebar" aria-hidden="true">
					<span class="lp-demo__dot lp-demo__dot--red"></span>
					<span class="lp-demo__dot lp-demo__dot--yellow"></span>
					<span class="lp-demo__dot lp-demo__dot--green"></span>
					<span class="lp-video-hero__titlebar-label">BusinessOS</span>
				</div>
				<div class="lp-video-hero__screen">
					<!-- Replace src with your MP4: /demo.mp4 in static/ -->
					<video
						class="lp-video-hero__video"
						autoplay
						muted
						loop
						playsinline
					>
						<!-- <source src="/demo.mp4" type="video/mp4" /> -->
					</video>
					<!-- Placeholder until video is added -->
					<div class="lp-video-hero__placeholder">
						<svg width="48" height="48" fill="none" stroke="rgba(255,255,255,0.25)" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						<span class="lp-video-hero__placeholder-text">Product Demo</span>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Section 2: Headline + CTAs -->
	<section class="lp-hero" aria-labelledby="lp-hero-headline">
		<div class="lp-container lp-hero__content {visible ? 'lp-hero__content--visible' : ''}">
			<h1 class="lp-hero__headline" id="lp-hero-headline">
				Your First Self-Learning Business OS
			</h1>
			<p class="lp-hero__subtitle">
				Proactive agents that run in your cloud computer. Automations that learn your workflows. A system that gets smarter with every decision you make.
			</p>
			<div class="lp-hero__ctas">
				<a href="https://github.com/Miosa-osa/businessos/releases" target="_blank" rel="noopener" class="lp-btn lp-btn--primary lp-btn--lg">
					Download for Mac
				</a>
				<a href="/register" class="lp-btn lp-btn--ghost lp-btn--lg">
					Start in Browser
				</a>
			</div>
			<div class="lp-hero__pills">
				<span class="lp-hero__pill">Proactive AI agents</span>
				<span class="lp-hero__pill">Cloud computer</span>
				<span class="lp-hero__pill">Self-evolving</span>
				<span class="lp-hero__pill">Bring your own keys</span>
			</div>
		</div>
	</section>

	<!-- Section 3: Modules Grid -->
	<section class="lp-modules" id="modules" aria-labelledby="lp-modules-headline">
		<div class="lp-container">
			<div class="lp-section-header">
				<p class="lp-eyebrow">Nine modules</p>
				<h2 class="lp-headline" id="lp-modules-headline">Everything you need</h2>
				<p class="lp-subheadline">One app that replaces your entire stack. No integrations required.</p>
			</div>
			<div class="lp-modules__grid" role="list">
				{#each modules as mod}
					<a href="/modules/{mod.name.toLowerCase()}" class="lp-module-card" role="listitem">
						<div class="lp-module-card__icon" aria-hidden="true">
							<svg class="lp-icon lp-icon--md" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={mod.icon} />
							</svg>
						</div>
						<h3 class="lp-module-card__name">{mod.name}</h3>
						<p class="lp-module-card__desc">{mod.desc}</p>
						<span class="lp-module-card__arrow">→</span>
					</a>
				{/each}
			</div>
		</div>
	</section>

	<!-- Section 4: How It Works — Hierarchy -->
	<section class="lp-hierarchy" aria-labelledby="lp-hierarchy-headline">
		<div class="lp-container">
			<div class="lp-section-header">
				<p class="lp-eyebrow">Built on Signal Theory</p>
				<h2 class="lp-headline" id="lp-hierarchy-headline">Not just an OS.<br />The optimal system.</h2>
				<p class="lp-subheadline">Every action in your business produces a signal. BusinessOS captures, routes, and learns from every signal — so nothing falls through the cracks. A continuous feedback loop from company to code.</p>
			</div>
			<div class="lp-hierarchy__flow" role="list">
				{#each hierarchy as item, i}
					<div class="lp-hierarchy__item" role="listitem">
						<div class="lp-hierarchy__icon">
							<svg class="lp-icon lp-icon--md" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={item.icon} />
							</svg>
						</div>
						<h3 class="lp-hierarchy__level">{item.level}</h3>
						<p class="lp-hierarchy__desc">{item.desc}</p>
					</div>
					{#if i < hierarchy.length - 1}
						<div class="lp-hierarchy__connector" aria-hidden="true">
							<svg width="24" height="24" fill="none" stroke="rgba(255,255,255,0.2)" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14 5l7 7m0 0l-7 7m7-7H3" />
							</svg>
						</div>
					{/if}
				{/each}
			</div>
			<div class="lp-hierarchy__cycle">
				<p class="lp-hierarchy__cycle-text">
					<span class="lp-hierarchy__cycle-icon">↻</span>
					Signals flow up. Context flows down. Daily reports, weekly reviews, and continuous feedback — your business gets smarter every cycle.
				</p>
			</div>
		</div>
	</section>

	<!-- Section 5: Your Computer -->
	<section class="lp-computer" aria-labelledby="lp-computer-headline">
		<div class="lp-container lp-computer__inner">
			<div class="lp-computer__text">
				<p class="lp-eyebrow lp-eyebrow--light">Your Computer</p>
				<h2 class="lp-headline lp-headline--light" id="lp-computer-headline">
					Bring your own keys.<br />Run any agent.
				</h2>
				<p class="lp-subheadline lp-subheadline--light">
					Your cloud computer runs the agent runtimes you choose. Bring your own API keys, schedule automations, and let AI agents power your business around the clock.
				</p>
				<div class="lp-computer__runtimes">
					{#each agentRuntimes as runtime}
						<span class="lp-computer__runtime-badge">{runtime}</span>
					{/each}
				</div>
				<ul class="lp-computer__features" role="list">
					{#each computerFeatures as feat}
						<li class="lp-computer__feature">
							<svg class="lp-icon lp-icon--sm lp-computer__check" aria-hidden="true" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
							{feat}
						</li>
					{/each}
				</ul>
				<a href="#pricing" class="lp-btn lp-btn--outline-light lp-btn--lg">
					See plans
					<svg class="lp-icon" aria-hidden="true" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
					</svg>
				</a>
			</div>
			<div class="lp-computer__preview" aria-hidden="true">
				<div class="lp-computer__screen">
					<div class="lp-computer__screen-bar">
						<span class="lp-demo__dot lp-demo__dot--red"></span>
						<span class="lp-demo__dot lp-demo__dot--yellow"></span>
						<span class="lp-demo__dot lp-demo__dot--green"></span>
						<span class="lp-computer__screen-label">Cloud Terminal — businessos.dev</span>
					</div>
					<div class="lp-computer__terminal">
						<p><span class="lp-term__prompt">$</span> claude</p>
						<p class="lp-term__output">Claude Code ready. How can I help?</p>
						<p><span class="lp-term__prompt">$</span> businessos agents --run crm-sync</p>
						<p class="lp-term__output lp-term__output--green">Running CRM sync agent... done in 2.3s</p>
						<p><span class="lp-term__prompt">$</span> <span class="lp-term__cursor" aria-hidden="true"></span></p>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Section 5: Pricing -->
	<section class="lp-pricing" id="pricing" aria-labelledby="lp-pricing-headline">
		<div class="lp-container">
			<div class="lp-section-header">
				<p class="lp-eyebrow">Transparent</p>
				<h2 class="lp-headline" id="lp-pricing-headline">Simple Pricing</h2>
				<p class="lp-subheadline">Unlimited seats on every plan. Pay for compute, not per user.</p>
			</div>
			<div class="lp-pricing__grid" role="list">
				{#each plans as plan}
					<div class="lp-plan {plan.highlight ? 'lp-plan--highlight' : ''}" role="listitem">
						{#if plan.badge}
							<span class="lp-plan__badge">{plan.badge}</span>
						{/if}
						<div class="lp-plan__header">
							<h3 class="lp-plan__name">{plan.name}</h3>
							<div class="lp-plan__price">
								<span class="lp-plan__amount">{plan.price}</span><span class="lp-plan__period">{plan.period}</span>
							</div>
							<p class="lp-plan__desc">{plan.desc}</p>
						</div>
						<ul class="lp-plan__features" role="list">
							{#each plan.features as feat}
								<li class="lp-plan__feature">
									<svg class="lp-icon lp-icon--sm" aria-hidden="true" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
									{feat}
								</li>
							{/each}
						</ul>
						<a href={plan.href} class="lp-btn lp-btn--full {plan.highlight ? 'lp-btn--primary' : 'lp-btn--outline'}">
							{plan.cta}
						</a>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Section 6: Open Source -->
	<section class="lp-oss" aria-labelledby="lp-oss-headline">
		<div class="lp-container lp-oss__inner">
			<div class="lp-oss__icon" aria-hidden="true">
				<svg class="lp-icon lp-icon--xl" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
			</div>
			<h2 class="lp-headline" id="lp-oss-headline">Free and Open Source</h2>
			<p class="lp-subheadline lp-subheadline--narrow">
				BusinessOS is Apache 2.0 licensed. Run it locally, customize every module, bring your own API keys. The cloud syncs your data — use it if you want, skip it if you don't.
			</p>
			<div class="lp-oss__actions">
				<a
					href="https://github.com/miosa-osa/businessos"
					target="_blank"
					rel="noopener noreferrer"
					class="lp-btn lp-btn--primary lp-btn--lg"
					aria-label="View BusinessOS on GitHub"
				>
					<svg class="lp-icon" aria-hidden="true" fill="currentColor" viewBox="0 0 24 24">
						<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
					</svg>
					View on GitHub
				</a>
				<a href="https://github.com/Miosa-osa/businessos" target="_blank" rel="noopener" class="lp-btn lp-btn--outline lp-btn--lg">
					Read the Docs
				</a>
			</div>
			<div class="lp-oss__badges" role="list">
				{#each ['Apache 2.0 Licensed', 'Self-hostable', 'No telemetry', 'Open roadmap'] as badge}
					<span class="lp-oss__badge" role="listitem">{badge}</span>
				{/each}
			</div>
		</div>
	</section>
</main>

<!-- Section 7: Footer -->
<footer class="lp-footer">
	<div class="lp-container lp-footer__inner">
		<div class="lp-footer__brand">
			<a href="/landing" class="lp-logo" aria-label="BusinessOS home">
				<span class="lp-logo__word">BUSINESS</span><span class="lp-logo__suffix">OS</span>
			</a>
			<p class="lp-footer__tagline">Built by MIOSA</p>
		</div>
		<nav class="lp-footer__links" aria-label="Footer navigation">
			<a href="https://github.com/miosa-osa/businessos" target="_blank" rel="noopener" class="lp-footer__link">GitHub</a>
			<a href="https://github.com/Miosa-osa/businessos" target="_blank" rel="noopener" class="lp-footer__link">Documentation</a>
			<a href="#pricing" class="lp-footer__link">Pricing</a>
			<a href="/privacy" class="lp-footer__link">Privacy</a>
			<a href="mailto:hello@businessos.dev" class="lp-footer__link">Contact</a>
		</nav>
		<p class="lp-footer__copy">&copy; {new Date().getFullYear()} MIOSA. Apache 2.0 License.</p>
	</div>
</footer>

<style>
	/* ─── Reset & Base ───────────────────────────────────────────── */
	:global(html) { scroll-behavior: smooth; }

	/* ─── Layout ─────────────────────────────────────────────────── */
	.lp-container {
		width: 100%;
		max-width: 1120px;
		margin-inline: auto;
		padding-inline: 1.5rem;
	}

	/* ─── Logo ───────────────────────────────────────────────────── */
	.lp-logo {
		display: flex;
		align-items: baseline;
		gap: 0;
		text-decoration: none;
	}
	.lp-logo__word {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 1rem;
		font-weight: 800;
		letter-spacing: 0.15em;
		color: #fff;
	}
	.lp-logo__suffix {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.875rem;
		font-weight: 300;
		color: rgba(255,255,255,0.35);
	}

	/* ─── Nav ────────────────────────────────────────────────────── */
	.lp-nav {
		position: fixed;
		inset-block-start: 0;
		inset-inline: 0;
		z-index: 50;
		transition: background 0.3s, border-color 0.3s, backdrop-filter 0.3s;
		border-bottom: 1px solid transparent;
	}
	.lp-nav--scrolled {
		background: rgba(10, 10, 14, 0.92);
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		border-bottom-color: rgba(255,255,255,0.06);
	}
	.lp-nav__inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 3.5rem;
	}
	.lp-nav__links {
		display: flex;
		align-items: center;
		gap: 2rem;
	}
	.lp-nav__link {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.75rem;
		color: rgba(255,255,255,0.5);
		text-decoration: none;
		transition: color 0.15s;
	}
	.lp-nav__link:hover { color: #fff; }
	.lp-nav__actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	@media (max-width: 640px) {
		.lp-nav__links { display: none; }
	}

	/* ─── Buttons ────────────────────────────────────────────────── */
	.lp-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.8125rem;
		font-weight: 500;
		border-radius: 0.5rem;
		padding: 0.5rem 1rem;
		text-decoration: none;
		transition: background 0.15s, color 0.15s, border-color 0.15s, opacity 0.15s;
		border: 1px solid transparent;
		cursor: pointer;
		white-space: nowrap;
	}
	.lp-btn--lg {
		font-size: 0.875rem;
		padding: 0.75rem 1.5rem;
		border-radius: 0.625rem;
	}
	.lp-btn--full { width: 100%; }

	.lp-btn--primary {
		background: #fff;
		color: #0a0a0e;
		border-color: #fff;
	}
	.lp-btn--primary:hover { background: rgba(255,255,255,0.88); }

	.lp-btn--ghost {
		background: transparent;
		color: rgba(255,255,255,0.6);
		border-color: transparent;
	}
	.lp-btn--ghost:hover { color: #fff; }

	.lp-btn--outline {
		background: transparent;
		color: rgba(255,255,255,0.7);
		border-color: rgba(255,255,255,0.2);
	}
	.lp-btn--outline:hover {
		color: #fff;
		border-color: rgba(255,255,255,0.5);
	}

	.lp-btn--outline-light {
		background: transparent;
		color: rgba(255,255,255,0.8);
		border-color: rgba(255,255,255,0.3);
	}
	.lp-btn--outline-light:hover {
		background: rgba(255,255,255,0.08);
		color: #fff;
		border-color: rgba(255,255,255,0.6);
	}

	/* ─── Icons ──────────────────────────────────────────────────── */
	.lp-icon { width: 1rem; height: 1rem; flex-shrink: 0; }
	.lp-icon--sm { width: 0.875rem; height: 0.875rem; }
	.lp-icon--md { width: 1.25rem; height: 1.25rem; }
	.lp-icon--xl { width: 2.5rem; height: 2.5rem; }

	/* ─── Full-page Grainient background ────────────────────────── */
	.lp-grainient-bg {
		position: fixed;
		inset: 0;
		z-index: 0;
		pointer-events: none;
	}

	/* All content sits above the shader */
	:global(.lp-nav),
	main,
	footer {
		position: relative;
		z-index: 1;
	}

	/* ─── Section typography ─────────────────────────────────────── */
	.lp-section-header {
		text-align: center;
		margin-bottom: 3.5rem;
	}
	.lp-eyebrow {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: rgba(255,255,255,0.4);
		margin-bottom: 0.75rem;
	}
	.lp-eyebrow--light { color: rgba(255,255,255,0.5); }
	.lp-headline {
		font-size: clamp(1.75rem, 3.5vw, 2.5rem);
		font-weight: 700;
		letter-spacing: -0.02em;
		line-height: 1.15;
		color: #fff;
		margin-bottom: 1rem;
	}
	.lp-headline--light { color: #fff; }
	.lp-subheadline {
		font-size: 1rem;
		line-height: 1.65;
		color: rgba(255,255,255,0.5);
		max-width: 42rem;
		margin-inline: auto;
	}
	.lp-subheadline--light { color: rgba(255,255,255,0.55); }
	.lp-subheadline--narrow { max-width: 34rem; }

	/* ─── Hero ───────────────────────────────────────────────────── */
	/* ─── Video Hero ────────────────────────────────────────────── */
	.lp-video-hero {
		position: relative;
		width: 100%;
		padding: 7rem 0 2rem;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.lp-video-hero__container {
		width: 100%;
		max-width: 1100px;
		margin: 0 auto;
		padding: 0 1.5rem;
	}
	.lp-video-hero__frame {
		border-radius: 0.875rem;
		overflow: hidden;
		border: 1px solid rgba(255,255,255,0.1);
		backdrop-filter: blur(4px);
		box-shadow: 0 25px 80px rgba(0, 0, 0, 0.5);
	}
	.lp-video-hero__titlebar {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.625rem 1rem;
		background: rgba(24, 24, 31, 0.9);
		border-bottom: 1px solid rgba(255,255,255,0.07);
	}
	.lp-video-hero__titlebar-label {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.3);
		margin-left: 0.5rem;
	}
	.lp-video-hero__screen {
		position: relative;
		aspect-ratio: 16 / 9;
		background: rgba(10, 10, 14, 0.9);
	}
	.lp-video-hero__video {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.lp-video-hero__placeholder {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
	}
	.lp-video-hero__placeholder-text {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.75rem;
		color: rgba(255,255,255,0.2);
		letter-spacing: 0.05em;
	}

	/* ─── Headline section ───────────────────────────────────────── */
	.lp-hero {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		padding-block: 4rem 5rem;
		text-align: center;
	}
	.lp-hero__bg {
		position: absolute;
		inset: 0;
		pointer-events: none;
	}
	/* Old mesh/glow replaced by Grainient WebGL shader */
	.lp-hero__content {
		position: relative;
		z-index: 1;
		opacity: 0;
		transform: translateY(24px);
		transition: opacity 0.7s ease, transform 0.7s ease;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		max-width: 48rem;
		margin-inline: auto;
	}
	.lp-hero__content--visible {
		opacity: 1;
		transform: translateY(0);
	}
	.lp-hero__eyebrow {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: rgba(255,255,255,0.4);
		margin-bottom: 1.25rem;
		display: flex;
		align-items: center;
		gap: 0.625rem;
	}
	.lp-hero__eyebrow::before {
		content: '';
		display: inline-block;
		width: 0.375rem;
		height: 0.375rem;
		border-radius: 50%;
		background: #22c55e;
		animation: lp-pulse 2s ease-in-out infinite;
	}
	.lp-hero__headline {
		font-size: clamp(2rem, 4.5vw, 3.5rem);
		font-weight: 700;
		letter-spacing: -0.025em;
		line-height: 1.15;
		color: #fff;
		margin-bottom: 1.25rem;
	}
	.lp-hero__subtitle {
		font-size: clamp(1rem, 1.5vw, 1.125rem);
		line-height: 1.7;
		color: rgba(255,255,255,0.55);
		max-width: 38rem;
		margin-bottom: 2.5rem;
	}
	.lp-hero__ctas {
		display: flex;
		justify-content: center;
		gap: 1rem;
		margin-bottom: 1.25rem;
		width: 100%;
		max-width: 28rem;
		margin-inline: auto;
	}
	.lp-hero__trust {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.75rem;
		color: rgba(255,255,255,0.3);
	}
	.lp-hero__pills {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.5rem;
		margin-top: 1.5rem;
	}
	.lp-hero__pill {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.5);
		border: 1px solid rgba(255,255,255,0.1);
		padding: 0.375rem 0.875rem;
		border-radius: 100px;
		letter-spacing: 0.02em;
	}

	@keyframes lp-pulse {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.5; transform: scale(0.85); }
	}

	/* ─── Demo ───────────────────────────────────────────────────── */
	.lp-demo {
		background: transparent;
		padding-block: 0 5rem;
	}
	.lp-demo__frame {
		border-radius: 0.875rem;
		overflow: hidden;
		backdrop-filter: blur(4px);
		border: 1px solid rgba(255,255,255,0.1);
		background: #111117;
		box-shadow: 0 32px 80px rgba(0,0,0,0.6);
	}
	.lp-demo__titlebar {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.625rem 1rem;
		background: rgba(24, 24, 31, 0.8);
		border-bottom: 1px solid rgba(255,255,255,0.07);
	}
	.lp-demo__dot {
		display: inline-block;
		width: 0.6875rem;
		height: 0.6875rem;
		border-radius: 50%;
	}
	.lp-demo__dot--red { background: #ff5f57; }
	.lp-demo__dot--yellow { background: #febc2e; }
	.lp-demo__dot--green { background: #28c840; }
	.lp-demo__titlebar-label {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.3);
		margin-left: 0.5rem;
	}
	.lp-demo__screen {
		aspect-ratio: 16 / 9;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(13, 13, 18, 0.8);
	}
	.lp-demo__placeholder {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
		color: rgba(255,255,255,0.2);
	}
	.lp-demo__play {
		width: 3.5rem;
		height: 3.5rem;
	}
	.lp-demo__label {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.8125rem;
		letter-spacing: 0.05em;
	}

	/* ─── Modules ────────────────────────────────────────────────── */
	.lp-modules {
		background: transparent;
		padding-block: 5rem;
	}
	.lp-modules__grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1px;
		background: rgba(255,255,255,0.04);
		border: 1px solid rgba(255,255,255,0.06);
		border-radius: 0.875rem;
		overflow: hidden;
	}

	@media (max-width: 768px) {
		.lp-modules__grid { grid-template-columns: repeat(2, 1fr); }
	}
	@media (max-width: 480px) {
		.lp-modules__grid { grid-template-columns: 1fr; }
	}

	.lp-module-card {
		background: rgba(17, 17, 23, 0.6);
		backdrop-filter: blur(4px);
		padding: 1.75rem;
		transition: background 0.15s;
	}
	.lp-module-card:hover { background: rgba(24, 24, 31, 0.7); }
	.lp-module-card__icon {
		width: 2.25rem;
		height: 2.25rem;
		border-radius: 0.5rem;
		background: rgba(255,255,255,0.06);
		display: flex;
		align-items: center;
		justify-content: center;
		color: rgba(255,255,255,0.7);
		margin-bottom: 1rem;
	}
	.lp-module-card__name {
		font-size: 0.9375rem;
		font-weight: 600;
		color: #fff;
		margin-bottom: 0.3rem;
	}
	.lp-module-card__desc {
		font-size: 0.8125rem;
		color: rgba(255,255,255,0.4);
		line-height: 1.5;
	}
	.lp-module-card__arrow {
		position: absolute;
		top: 1.25rem;
		right: 1.25rem;
		font-size: 1rem;
		color: rgba(255,255,255,0.15);
		transition: color 0.15s, transform 0.15s;
	}
	.lp-module-card:hover .lp-module-card__arrow {
		color: rgba(255,255,255,0.5);
		transform: translateX(2px);
	}
	a.lp-module-card {
		text-decoration: none;
		position: relative;
		display: block;
		cursor: pointer;
	}

	/* ─── Hierarchy / Signal Theory ──────────────────────────────── */
	.lp-hierarchy {
		background: transparent;
		padding-block: 5rem;
	}
	.lp-hierarchy__flow {
		display: flex;
		align-items: flex-start;
		justify-content: center;
		gap: 0;
		flex-wrap: wrap;
		margin-bottom: 2.5rem;
	}
	.lp-hierarchy__item {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 0.625rem;
		flex: 0 0 auto;
		width: 140px;
	}
	.lp-hierarchy__icon {
		width: 3rem;
		height: 3rem;
		border-radius: 0.75rem;
		background: rgba(255,255,255,0.06);
		border: 1px solid rgba(255,255,255,0.08);
		display: flex;
		align-items: center;
		justify-content: center;
		color: rgba(255,255,255,0.6);
	}
	.lp-hierarchy__level {
		font-size: 0.9375rem;
		font-weight: 600;
		color: #fff;
	}
	.lp-hierarchy__desc {
		font-size: 0.75rem;
		color: rgba(255,255,255,0.4);
		line-height: 1.4;
	}
	.lp-hierarchy__connector {
		display: flex;
		align-items: center;
		padding-top: 0.75rem;
	}
	.lp-hierarchy__cycle {
		text-align: center;
		max-width: 36rem;
		margin-inline: auto;
	}
	.lp-hierarchy__cycle-text {
		font-size: 0.875rem;
		color: rgba(255,255,255,0.35);
		line-height: 1.6;
	}
	.lp-hierarchy__cycle-icon {
		display: inline-block;
		font-size: 1.125rem;
		margin-right: 0.375rem;
		color: rgba(255,255,255,0.25);
	}
	@media (max-width: 640px) {
		.lp-hierarchy__flow { flex-direction: column; align-items: center; gap: 1rem; }
		.lp-hierarchy__connector { transform: rotate(90deg); }
	}

	/* ─── Computer ───────────────────────────────────────────────── */
	.lp-computer {
		background: rgba(10, 10, 14, 0.5);
		padding-block: 5rem;
	}
	.lp-computer__inner {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 4rem;
		align-items: center;
	}

	@media (max-width: 768px) {
		.lp-computer__inner { grid-template-columns: 1fr; }
		.lp-computer__preview { order: -1; }
	}

	.lp-computer__features {
		list-style: none;
		padding: 0;
		margin: 1.75rem 0 2rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.lp-computer__feature {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		font-size: 0.9rem;
		color: rgba(255,255,255,0.65);
	}
	.lp-computer__check { color: #22c55e; }
	.lp-computer__runtimes {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
	}
	.lp-computer__runtime-badge {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.6);
		background: rgba(255,255,255,0.06);
		border: 1px solid rgba(255,255,255,0.1);
		padding: 0.3125rem 0.75rem;
		border-radius: 100px;
		letter-spacing: 0.02em;
	}
	.lp-computer__screen {
		border-radius: 0.75rem;
		overflow: hidden;
		border: 1px solid rgba(255,255,255,0.1);
		background: rgba(24, 24, 31, 0.7);
		backdrop-filter: blur(4px);
	}
	.lp-computer__screen-bar {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.625rem 0.875rem;
		background: rgba(17, 17, 23, 0.8);
		border-bottom: 1px solid rgba(255,255,255,0.07);
	}
	.lp-computer__screen-label {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.25);
		margin-left: 0.375rem;
	}
	.lp-computer__terminal {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.8125rem;
		padding: 1.25rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		line-height: 1.6;
		color: rgba(255,255,255,0.5);
	}
	.lp-term__prompt { color: #22c55e; margin-right: 0.5rem; }
	.lp-term__output { color: rgba(255,255,255,0.35); padding-left: 1rem; }
	.lp-term__output--green { color: #86efac; }
	.lp-term__cursor {
		display: inline-block;
		width: 0.5rem;
		height: 1rem;
		background: #22c55e;
		vertical-align: middle;
		animation: lp-blink 1s step-end infinite;
	}
	@keyframes lp-blink {
		0%, 50% { opacity: 1; }
		51%, 100% { opacity: 0; }
	}

	/* ─── Pricing ────────────────────────────────────────────────── */
	.lp-pricing {
		background: rgba(10, 10, 14, 0.6);
		backdrop-filter: blur(2px);
		padding-block: 5rem;
	}
	.lp-pricing__grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
	}

	@media (max-width: 1024px) {
		.lp-pricing__grid { grid-template-columns: repeat(2, 1fr); }
	}
	@media (max-width: 560px) {
		.lp-pricing__grid { grid-template-columns: 1fr; }
	}

	.lp-plan {
		position: relative;
		background: rgba(24, 24, 31, 0.6);
		backdrop-filter: blur(4px);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 0.875rem;
		padding: 1.75rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		transition: border-color 0.15s;
	}
	.lp-plan:hover { border-color: rgba(255,255,255,0.18); }
	.lp-plan--highlight {
		border-color: rgba(255,255,255,0.3);
		background: rgba(30, 30, 40, 0.7);
	}
	.lp-plan__badge {
		position: absolute;
		top: -0.6875rem;
		left: 50%;
		transform: translateX(-50%);
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.625rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: #0a0a0e;
		background: #fff;
		padding: 0.25rem 0.75rem;
		border-radius: 9999px;
	}
	.lp-plan__header { display: flex; flex-direction: column; gap: 0.375rem; }
	.lp-plan__name {
		font-size: 0.9375rem;
		font-weight: 600;
		color: #fff;
	}
	.lp-plan__price {
		display: flex;
		align-items: baseline;
		gap: 0.125rem;
	}
	.lp-plan__amount {
		font-size: 2rem;
		font-weight: 800;
		letter-spacing: -0.03em;
		color: #fff;
	}
	.lp-plan__period {
		font-size: 0.875rem;
		color: rgba(255,255,255,0.4);
	}
	.lp-plan__desc {
		font-size: 0.8125rem;
		color: rgba(255,255,255,0.35);
	}
	.lp-plan__features {
		list-style: none;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
		flex: 1;
	}
	.lp-plan__feature {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		color: rgba(255,255,255,0.6);
	}
	.lp-plan__feature .lp-icon { color: #22c55e; }

	/* ─── Open Source ────────────────────────────────────────────── */
	.lp-oss {
		background: rgba(10, 10, 14, 0.5);
		padding-block: 5rem;
	}
	.lp-oss__inner {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
	}
	.lp-oss__icon {
		color: rgba(255,255,255,0.2);
		margin-bottom: 1.5rem;
	}
	.lp-oss__actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		margin-top: 2rem;
		justify-content: center;
	}
	.lp-oss__badges {
		display: flex;
		flex-wrap: wrap;
		gap: 0.625rem;
		margin-top: 2rem;
		justify-content: center;
	}
	.lp-oss__badge {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.4);
		background: rgba(255,255,255,0.06);
		border: 1px solid rgba(255,255,255,0.1);
		border-radius: 9999px;
		padding: 0.25rem 0.75rem;
	}

	/* ─── Footer ─────────────────────────────────────────────────── */
	.lp-footer {
		background: rgba(10, 10, 14, 0.7);
		border-top: 1px solid rgba(255,255,255,0.07);
		padding-block: 2.5rem;
	}
	.lp-footer__inner {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		align-items: center;
		text-align: center;
	}
	.lp-footer__brand {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.375rem;
	}
	.lp-footer__tagline {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.2);
	}
	.lp-footer__links {
		display: flex;
		flex-wrap: wrap;
		gap: 1.5rem;
		justify-content: center;
	}
	.lp-footer__link {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.75rem;
		color: rgba(255,255,255,0.35);
		text-decoration: none;
		transition: color 0.15s;
	}
	.lp-footer__link:hover { color: rgba(255,255,255,0.8); }
	.lp-footer__copy {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
		font-size: 0.6875rem;
		color: rgba(255,255,255,0.18);
	}
</style>
