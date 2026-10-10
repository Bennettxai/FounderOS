package services

// --- CRM Module Template ---
func crmModuleTemplate() *BuiltInTemplate {
	return &BuiltInTemplate{
		ID:          "crm_module",
		Name:        "CRM Module",
		Description: "Contact and deal management with pipeline view, activity tracking, and reporting",
		Category:    "crm",
		StackType:   "svelte",
		ConfigSchema: map[string]ConfigField{
			"app_name":        {Type: "string", Label: "CRM Name", Default: "My CRM", Required: true},
			"primary_color":   {Type: "string", Label: "Primary Color", Default: "#10B981", Required: false},
			"pipeline_stages": {Type: "string", Label: "Pipeline Stages (comma-separated)", Default: "Lead,Qualified,Proposal,Negotiation,Won,Lost", Required: false},
			"currency":        {Type: "select", Label: "Currency", Default: "USD", Options: []string{"USD", "EUR", "GBP", "BRL"}},
		},
		FilesTemplate: map[string]string{
			"src/routes/+page.svelte": `<script lang="ts">
	import ContactList from '$lib/components/ContactList.svelte';
	import DealPipeline from '$lib/components/DealPipeline.svelte';
	import CRMStats from '$lib/components/CRMStats.svelte';

	let activeView = $state<'contacts' | 'pipeline' | 'stats'>('pipeline');
</script>

<svelte:head>
	<title>{{app_name}}</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
	<!-- Header -->
	<header class="bg-white border-b border-gray-200 px-6 py-4">
		<div class="flex items-center justify-between">
			<h1 class="text-2xl font-bold text-gray-900">{{app_name}}</h1>
			<div class="flex items-center gap-2">
				<button
					onclick={() => activeView = 'pipeline'}
					class="px-4 py-2 text-sm rounded-lg transition-colors {activeView === 'pipeline' ? 'bg-emerald-100 text-emerald-700' : 'text-gray-600 hover:bg-gray-100'}"
				>Pipeline</button>
				<button
					onclick={() => activeView = 'contacts'}
					class="px-4 py-2 text-sm rounded-lg transition-colors {activeView === 'contacts' ? 'bg-emerald-100 text-emerald-700' : 'text-gray-600 hover:bg-gray-100'}"
				>Contacts</button>
				<button
					onclick={() => activeView = 'stats'}
					class="px-4 py-2 text-sm rounded-lg transition-colors {activeView === 'stats' ? 'bg-emerald-100 text-emerald-700' : 'text-gray-600 hover:bg-gray-100'}"
				>Stats</button>
			</div>
		</div>
	</header>

	<!-- Content -->
	<div class="p-6">
		{#if activeView === 'pipeline'}
			<DealPipeline />
		{:else if activeView === 'contacts'}
			<ContactList />
		{:else}
			<CRMStats />
		{/if}
	</div>
</div>
`,
			"src/lib/components/DealPipeline.svelte": `<script lang="ts">
	interface Deal {
		id: string;
		name: string;
		company: string;
		value: number;
		stage: string;
		probability: number;
	}

	const stages = '{{pipeline_stages}}'.split(',').map(s => s.trim());
	const currency = '{{currency}}';

	let deals = $state<Deal[]>([
		{ id: '1', name: 'Enterprise License', company: 'Acme Corp', value: 50000, stage: 'Proposal', probability: 60 },
		{ id: '2', name: 'Consulting Package', company: 'Tech Inc', value: 25000, stage: 'Qualified', probability: 40 },
		{ id: '3', name: 'Annual Subscription', company: 'Global Ltd', value: 12000, stage: 'Negotiation', probability: 80 },
		{ id: '4', name: 'Platform Migration', company: 'StartupXYZ', value: 75000, stage: 'Lead', probability: 20 },
	]);

	function getDealsForStage(stage: string): Deal[] {
		return deals.filter(d => d.stage === stage);
	}

	function getStageTotal(stage: string): number {
		return getDealsForStage(stage).reduce((sum, d) => sum + d.value, 0);
	}

	function formatCurrency(value: number): string {
		return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency }).format(value);
	}
</script>

<div class="flex gap-4 overflow-x-auto pb-4">
	{#each stages as stage}
		<div class="flex-shrink-0 w-72 bg-gray-100 rounded-xl p-4">
			<div class="flex items-center justify-between mb-4">
				<h3 class="font-semibold text-gray-900">{stage}</h3>
				<span class="text-xs text-gray-500">{formatCurrency(getStageTotal(stage))}</span>
			</div>
			<div class="space-y-3">
				{#each getDealsForStage(stage) as deal (deal.id)}
					<div class="bg-white rounded-lg p-4 border border-gray-200 shadow-sm hover:shadow-md transition-shadow cursor-pointer">
						<h4 class="font-medium text-gray-900 text-sm mb-1">{deal.name}</h4>
						<p class="text-xs text-gray-500 mb-2">{deal.company}</p>
						<div class="flex items-center justify-between">
							<span class="text-sm font-semibold" style="color: {{primary_color}}">{formatCurrency(deal.value)}</span>
							<span class="text-xs text-gray-400">{deal.probability}%</span>
						</div>
					</div>
				{/each}
				{#if getDealsForStage(stage).length === 0}
					<div class="text-center py-4 text-sm text-gray-400">No deals</div>
				{/if}
			</div>
		</div>
	{/each}
</div>
`,
			"src/lib/components/ContactList.svelte": `<script lang="ts">
	import { Search, Plus, Mail, Phone } from 'lucide-svelte';

	interface Contact {
		id: string;
		name: string;
		email: string;
		phone: string;
		company: string;
		status: 'active' | 'inactive' | 'prospect';
		lastContact: string;
	}

	let contacts = $state<Contact[]>([
		{ id: '1', name: 'John Doe', email: 'john@acme.com', phone: '+1-555-0101', company: 'Acme Corp', status: 'active', lastContact: '2 days ago' },
		{ id: '2', name: 'Jane Smith', email: 'jane@tech.com', phone: '+1-555-0102', company: 'Tech Inc', status: 'active', lastContact: '1 week ago' },
		{ id: '3', name: 'Bob Wilson', email: 'bob@global.com', phone: '+1-555-0103', company: 'Global Ltd', status: 'prospect', lastContact: '3 days ago' },
	]);

	let searchQuery = $state('');

	const filteredContacts = $derived(
		contacts.filter(c =>
			c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			c.company.toLowerCase().includes(searchQuery.toLowerCase()) ||
			c.email.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);
</script>

<div class="bg-white rounded-xl border border-gray-200">
	<div class="flex items-center justify-between p-6 border-b border-gray-200">
		<div class="flex items-center gap-4">
			<div class="relative">
				<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
				<input
					type="text"
					placeholder="Search contacts..."
					bind:value={searchQuery}
					class="pl-10 pr-4 py-2 border border-gray-300 rounded-lg text-sm"
				/>
			</div>
		</div>
		<button class="flex items-center gap-2 px-4 py-2 text-sm text-white rounded-lg" style="background-color: {{primary_color}}">
			<Plus class="w-4 h-4" />
			Add Contact
		</button>
	</div>
	<div class="divide-y divide-gray-100">
		{#each filteredContacts as contact (contact.id)}
			<div class="flex items-center justify-between p-4 hover:bg-gray-50">
				<div class="flex items-center gap-4">
					<div class="w-10 h-10 rounded-full bg-gray-200 flex items-center justify-center text-sm font-medium text-gray-600">
						{contact.name.split(' ').map(n => n[0]).join('')}
					</div>
					<div>
						<div class="font-medium text-gray-900">{contact.name}</div>
						<div class="text-sm text-gray-500">{contact.company}</div>
					</div>
				</div>
				<div class="flex items-center gap-4">
					<span class="text-xs text-gray-400">Last: {contact.lastContact}</span>
					<button class="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
						<Mail class="w-4 h-4" />
					</button>
					<button class="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
						<Phone class="w-4 h-4" />
					</button>
				</div>
			</div>
		{/each}
	</div>
</div>
`,
			"src/lib/components/CRMStats.svelte": `<script lang="ts">
	const stats = [
		{ label: 'Total Contacts', value: '248', change: '+12%' },
		{ label: 'Active Deals', value: '34', change: '+5%' },
		{ label: 'Pipeline Value', value: '$1.2M', change: '+23%' },
		{ label: 'Win Rate', value: '68%', change: '+4%' },
	];
</script>

<div class="space-y-6">
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
		{#each stats as stat}
			<div class="bg-white rounded-xl border border-gray-200 p-6">
				<div class="text-sm text-gray-500 mb-1">{stat.label}</div>
				<div class="text-3xl font-bold text-gray-900">{stat.value}</div>
				<div class="text-sm text-green-600 mt-1">{stat.change} from last month</div>
			</div>
		{/each}
	</div>
</div>
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
