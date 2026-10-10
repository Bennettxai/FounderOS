package services

// --- Landing Page Template ---
func landingPageTemplate() *BuiltInTemplate {
	return &BuiltInTemplate{
		ID:          "landing_page",
		Name:        "Landing Page",
		Description: "Modern landing page with hero section, features, pricing, and contact form",
		Category:    "marketing",
		StackType:   "svelte",
		ConfigSchema: map[string]ConfigField{
			"app_name":        {Type: "string", Label: "Product Name", Default: "My Product", Required: true},
			"tagline":         {Type: "string", Label: "Tagline", Default: "The best solution for your needs", Required: false},
			"primary_color":   {Type: "string", Label: "Primary Color", Default: "#6366F1", Required: false},
			"cta_text":        {Type: "string", Label: "CTA Button Text", Default: "Get Started Free", Required: false},
			"pricing_enabled": {Type: "boolean", Label: "Show Pricing Section", Default: "true", Required: false},
		},
		FilesTemplate: map[string]string{
			"src/routes/+page.svelte": `<script lang="ts">
	import Hero from '$lib/components/Hero.svelte';
	import Features from '$lib/components/Features.svelte';
	import Pricing from '$lib/components/Pricing.svelte';
	import ContactForm from '$lib/components/ContactForm.svelte';
	import Footer from '$lib/components/Footer.svelte';
</script>

<svelte:head>
	<title>{{app_name}} - {{tagline}}</title>
	<meta name="description" content="{{tagline}}" />
</svelte:head>

<div class="min-h-screen">
	<Hero />
	<Features />
	<Pricing />
	<ContactForm />
	<Footer />
</div>
`,
			"src/lib/components/Hero.svelte": `<script lang="ts">
	import { ArrowRight } from 'lucide-svelte';
</script>

<section class="relative bg-gradient-to-br from-indigo-50 via-white to-purple-50 pt-20 pb-32">
	<div class="max-w-7xl mx-auto px-6 text-center">
		<h1 class="text-5xl md:text-7xl font-bold text-gray-900 mb-6 tracking-tight">
			{{app_name}}
		</h1>
		<p class="text-xl md:text-2xl text-gray-600 mb-10 max-w-3xl mx-auto">
			{{tagline}}
		</p>
		<div class="flex flex-col sm:flex-row items-center justify-center gap-4">
			<a
				href="#pricing"
				class="px-8 py-4 text-lg font-semibold text-white rounded-xl shadow-lg hover:shadow-xl transition-all"
				style="background-color: {{primary_color}}"
			>
				{{cta_text}}
				<ArrowRight class="w-5 h-5 inline ml-2" />
			</a>
			<a href="#features" class="px-8 py-4 text-lg font-semibold text-gray-700 bg-white border-2 border-gray-200 rounded-xl hover:border-gray-300">
				Learn More
			</a>
		</div>
	</div>
</section>
`,
			"src/lib/components/Features.svelte": `<script lang="ts">
	import { Zap, Shield, BarChart3, Users } from 'lucide-svelte';

	const features = [
		{
			icon: Zap,
			title: 'Lightning Fast',
			description: 'Built for performance with optimized workflows that save you hours every week.'
		},
		{
			icon: Shield,
			title: 'Enterprise Security',
			description: 'Bank-grade encryption and compliance standards to keep your data safe.'
		},
		{
			icon: BarChart3,
			title: 'Advanced Analytics',
			description: 'Deep insights into your business with real-time dashboards and reports.'
		},
		{
			icon: Users,
			title: 'Team Collaboration',
			description: 'Work together seamlessly with real-time editing and shared workspaces.'
		}
	];
</script>

<section id="features" class="py-24 bg-white">
	<div class="max-w-7xl mx-auto px-6">
		<div class="text-center mb-16">
			<h2 class="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
				Everything you need
			</h2>
			<p class="text-lg text-gray-600 max-w-2xl mx-auto">
				Powerful features designed to help your team succeed.
			</p>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
			{#each features as feature}
				<div class="p-6 rounded-2xl border border-gray-200 hover:border-indigo-200 hover:shadow-lg transition-all">
					<div class="w-12 h-12 rounded-xl flex items-center justify-center mb-4" style="background-color: {{primary_color}}20">
						<svelte:component this={feature.icon} class="w-6 h-6" style="color: {{primary_color}}" />
					</div>
					<h3 class="text-lg font-semibold text-gray-900 mb-2">{feature.title}</h3>
					<p class="text-gray-600">{feature.description}</p>
				</div>
			{/each}
		</div>
	</div>
</section>
`,
			"src/lib/components/Pricing.svelte": `<script lang="ts">
	import { Check } from 'lucide-svelte';

	const plans = [
		{
			name: 'Starter',
			price: '19',
			description: 'Perfect for individuals',
			features: ['5 projects', '1 GB storage', 'Email support', 'Basic analytics']
		},
		{
			name: 'Pro',
			price: '49',
			description: 'Best for growing teams',
			features: ['Unlimited projects', '10 GB storage', 'Priority support', 'Advanced analytics', 'API access', 'Custom integrations'],
			popular: true
		},
		{
			name: 'Enterprise',
			price: '99',
			description: 'For large organizations',
			features: ['Everything in Pro', 'Unlimited storage', '24/7 phone support', 'Custom contracts', 'SLA guarantee', 'Dedicated manager']
		}
	];
</script>

<section id="pricing" class="py-24 bg-gray-50">
	<div class="max-w-7xl mx-auto px-6">
		<div class="text-center mb-16">
			<h2 class="text-3xl md:text-4xl font-bold text-gray-900 mb-4">Simple Pricing</h2>
			<p class="text-lg text-gray-600">Choose the plan that works for you.</p>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
			{#each plans as plan}
				<div class="relative bg-white rounded-2xl border-2 p-8 {plan.popular ? 'border-indigo-500 shadow-xl' : 'border-gray-200'}">
					{#if plan.popular}
						<div class="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 text-sm font-semibold text-white rounded-full" style="background-color: {{primary_color}}">
							Most Popular
						</div>
					{/if}
					<h3 class="text-xl font-bold text-gray-900 mb-2">{plan.name}</h3>
					<p class="text-gray-600 mb-4">{plan.description}</p>
					<div class="mb-6">
						<span class="text-4xl font-bold text-gray-900">${plan.price}</span>
						<span class="text-gray-500">/month</span>
					</div>
					<ul class="space-y-3 mb-8">
						{#each plan.features as feature}
							<li class="flex items-center gap-2 text-sm text-gray-700">
								<Check class="w-4 h-4 text-green-500 flex-shrink-0" />
								<span>{feature}</span>
							</li>
						{/each}
					</ul>
					<button
						class="w-full py-3 rounded-xl font-semibold transition-colors {plan.popular ? 'text-white' : 'text-gray-700 bg-gray-100 hover:bg-gray-200'}"
						style={plan.popular ? 'background-color: {{primary_color}}' : ''}
					>
						{{cta_text}}
					</button>
				</div>
			{/each}
		</div>
	</div>
</section>
`,
			"src/lib/components/ContactForm.svelte": `<script lang="ts">
	let name = $state('');
	let email = $state('');
	let message = $state('');
	let submitted = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		// Submit form logic here
		submitted = true;
		setTimeout(() => { submitted = false; }, 3000);
	}
</script>

<section id="contact" class="py-24 bg-white">
	<div class="max-w-3xl mx-auto px-6">
		<div class="text-center mb-12">
			<h2 class="text-3xl font-bold text-gray-900 mb-4">Get in Touch</h2>
			<p class="text-gray-600">Have questions? We would love to hear from you.</p>
		</div>
		{#if submitted}
			<div class="p-6 bg-green-50 border border-green-200 rounded-xl text-center">
				<p class="text-green-700 font-medium">Thank you! We will get back to you soon.</p>
			</div>
		{:else}
			<form onsubmit={handleSubmit} class="space-y-6">
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div>
						<label for="name" class="block text-sm font-medium text-gray-700 mb-2">Name</label>
						<input id="name" type="text" bind:value={name} required class="w-full px-4 py-3 border border-gray-300 rounded-xl focus:ring-2 focus:ring-indigo-500" />
					</div>
					<div>
						<label for="email" class="block text-sm font-medium text-gray-700 mb-2">Email</label>
						<input id="email" type="email" bind:value={email} required class="w-full px-4 py-3 border border-gray-300 rounded-xl focus:ring-2 focus:ring-indigo-500" />
					</div>
				</div>
				<div>
					<label for="message" class="block text-sm font-medium text-gray-700 mb-2">Message</label>
					<textarea id="message" bind:value={message} rows="4" required class="w-full px-4 py-3 border border-gray-300 rounded-xl focus:ring-2 focus:ring-indigo-500"></textarea>
				</div>
				<button type="submit" class="w-full py-4 text-white font-semibold rounded-xl" style="background-color: {{primary_color}}">
					Send Message
				</button>
			</form>
		{/if}
	</div>
</section>
`,
			"src/lib/components/Footer.svelte": `<footer class="bg-gray-900 text-gray-400 py-12">
	<div class="max-w-7xl mx-auto px-6">
		<div class="flex flex-col md:flex-row items-center justify-between">
			<div class="text-lg font-bold text-white mb-4 md:mb-0">{{app_name}}</div>
			<div class="flex items-center gap-6 text-sm">
				<a href="#features" class="hover:text-white transition-colors">Features</a>
				<a href="#pricing" class="hover:text-white transition-colors">Pricing</a>
				<a href="#contact" class="hover:text-white transition-colors">Contact</a>
			</div>
		</div>
		<div class="mt-8 pt-8 border-t border-gray-800 text-center text-sm">
			<p>Generated with BusinessOS Template System</p>
		</div>
	</div>
</footer>
`,
			"package.json": `{
	"name": "{{app_name}}",
	"version": "1.0.0",
	"type": "module",
	"scripts": {
		"dev": "vite dev",
		"build": "vite build",
		"preview": "vite preview"
	},
	"devDependencies": {
		"@sveltejs/adapter-auto": "^3.0.0",
		"@sveltejs/kit": "^2.0.0",
		"svelte": "^5.0.0",
		"tailwindcss": "^3.4.0",
		"typescript": "^5.0.0",
		"vite": "^5.0.0"
	},
	"dependencies": {
		"lucide-svelte": "^0.300.0"
	}
}
`,
		},
	}
}
