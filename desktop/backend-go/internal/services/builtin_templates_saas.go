package services

// --- SaaS Dashboard Template ---
func saaDashboardTemplate() *BuiltInTemplate {
	return &BuiltInTemplate{
		ID:          "saas_dashboard",
		Name:        "SaaS Dashboard",
		Description: "Full-featured SaaS dashboard with charts, user management, and analytics",
		Category:    "operations",
		StackType:   "svelte",
		ConfigSchema: map[string]ConfigField{
			"app_name":      {Type: "string", Label: "Application Name", Default: "My Dashboard", Required: true},
			"primary_color": {Type: "string", Label: "Primary Color", Default: "#3B82F6", Required: false},
			"chart_library": {Type: "select", Label: "Chart Library", Default: "chart.js", Options: []string{"chart.js", "d3", "echarts"}},
			"auth_enabled":  {Type: "boolean", Label: "Enable Authentication", Default: "true", Required: false},
		},
		FilesTemplate: map[string]string{
			"src/routes/+page.svelte": `<script lang="ts">
	import { onMount } from 'svelte';
	import StatsCard from '$lib/components/StatsCard.svelte';
	import RevenueChart from '$lib/components/RevenueChart.svelte';
	import UserTable from '$lib/components/UserTable.svelte';
	import ActivityFeed from '$lib/components/ActivityFeed.svelte';

	let stats = $state({
		totalUsers: 0,
		revenue: 0,
		activeSubscriptions: 0,
		churnRate: 0
	});

	onMount(async () => {
		// Fetch dashboard stats
		const response = await fetch('/api/dashboard/stats');
		if (response.ok) {
			stats = await response.json();
		}
	});
</script>

<svelte:head>
	<title>{{app_name}} - Dashboard</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
	<!-- Header -->
	<header class="bg-white border-b border-gray-200 px-6 py-4">
		<div class="flex items-center justify-between">
			<h1 class="text-2xl font-bold text-gray-900">{{app_name}}</h1>
			<div class="flex items-center gap-4">
				<button class="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700">
					Export Report
				</button>
			</div>
		</div>
	</header>

	<!-- Stats Grid -->
	<div class="p-6">
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
			<StatsCard title="Total Users" value={stats.totalUsers} trend="+12.5%" positive={true} />
			<StatsCard title="Revenue" value={'$' + stats.revenue.toLocaleString()} trend="+8.2%" positive={true} />
			<StatsCard title="Active Subscriptions" value={stats.activeSubscriptions} trend="+3.1%" positive={true} />
			<StatsCard title="Churn Rate" value={stats.churnRate + '%'} trend="-1.2%" positive={false} />
		</div>

		<!-- Charts -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
			<RevenueChart />
			<ActivityFeed />
		</div>

		<!-- User Table -->
		<UserTable />
	</div>
</div>
`,
			"src/lib/components/StatsCard.svelte": `<script lang="ts">
	import { TrendingUp, TrendingDown } from 'lucide-svelte';

	interface Props {
		title: string;
		value: string | number;
		trend: string;
		positive: boolean;
	}

	let { title, value, trend, positive }: Props = $props();
</script>

<div class="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-md transition-shadow">
	<div class="flex items-center justify-between mb-2">
		<span class="text-sm font-medium text-gray-600">{title}</span>
		<div class="flex items-center gap-1 text-sm {positive ? 'text-green-600' : 'text-red-600'}">
			{#if positive}
				<TrendingUp class="w-4 h-4" />
			{:else}
				<TrendingDown class="w-4 h-4" />
			{/if}
			<span>{trend}</span>
		</div>
	</div>
	<div class="text-3xl font-bold text-gray-900">{value}</div>
</div>
`,
			"src/lib/components/RevenueChart.svelte": `<script lang="ts">
	import { onMount } from 'svelte';

	let canvas: HTMLCanvasElement;
	let chartData = $state({
		labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
		values: [4200, 5100, 4800, 6200, 7100, 8400]
	});

	onMount(async () => {
		// Initialize chart with {{chart_library}}
		const ctx = canvas.getContext('2d');
		if (ctx) {
			drawChart(ctx);
		}
	});

	function drawChart(ctx: CanvasRenderingContext2D) {
		const width = canvas.width;
		const height = canvas.height;
		const maxValue = Math.max(...chartData.values);
		const barWidth = width / chartData.values.length - 10;

		ctx.clearRect(0, 0, width, height);
		ctx.fillStyle = '{{primary_color}}';

		chartData.values.forEach((value, index) => {
			const barHeight = (value / maxValue) * (height - 40);
			const x = index * (barWidth + 10) + 5;
			const y = height - barHeight - 20;
			ctx.fillRect(x, y, barWidth, barHeight);
		});
	}
</script>

<div class="bg-white rounded-xl border border-gray-200 p-6">
	<h3 class="text-lg font-semibold text-gray-900 mb-4">Revenue Overview</h3>
	<canvas bind:this={canvas} width="500" height="300" class="w-full"></canvas>
</div>
`,
			"src/lib/components/UserTable.svelte": `<script lang="ts">
	import { Search, MoreVertical } from 'lucide-svelte';

	interface User {
		id: string;
		name: string;
		email: string;
		plan: string;
		status: 'active' | 'inactive' | 'trial';
		joinedAt: string;
	}

	let users = $state<User[]>([
		{ id: '1', name: 'Alice Johnson', email: 'alice@example.com', plan: 'Pro', status: 'active', joinedAt: '2024-01-15' },
		{ id: '2', name: 'Bob Smith', email: 'bob@example.com', plan: 'Basic', status: 'active', joinedAt: '2024-02-20' },
		{ id: '3', name: 'Carol Williams', email: 'carol@example.com', plan: 'Enterprise', status: 'trial', joinedAt: '2024-03-10' },
	]);

	let searchQuery = $state('');

	const filteredUsers = $derived(
		users.filter(u =>
			u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			u.email.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	function getStatusColor(status: string): string {
		switch (status) {
			case 'active': return 'bg-green-100 text-green-700';
			case 'inactive': return 'bg-gray-100 text-gray-700';
			case 'trial': return 'bg-blue-100 text-blue-700';
			default: return 'bg-gray-100 text-gray-700';
		}
	}
</script>

<div class="bg-white rounded-xl border border-gray-200">
	<div class="flex items-center justify-between p-6 border-b border-gray-200">
		<h3 class="text-lg font-semibold text-gray-900">Users</h3>
		<div class="relative">
			<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
			<input
				type="text"
				placeholder="Search users..."
				bind:value={searchQuery}
				class="pl-10 pr-4 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
			/>
		</div>
	</div>
	<div class="overflow-x-auto">
		<table class="w-full">
			<thead>
				<tr class="border-b border-gray-200">
					<th class="text-left text-xs font-medium text-gray-500 uppercase px-6 py-3">User</th>
					<th class="text-left text-xs font-medium text-gray-500 uppercase px-6 py-3">Plan</th>
					<th class="text-left text-xs font-medium text-gray-500 uppercase px-6 py-3">Status</th>
					<th class="text-left text-xs font-medium text-gray-500 uppercase px-6 py-3">Joined</th>
					<th class="text-right text-xs font-medium text-gray-500 uppercase px-6 py-3">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each filteredUsers as user (user.id)}
					<tr class="border-b border-gray-100 hover:bg-gray-50">
						<td class="px-6 py-4">
							<div>
								<div class="font-medium text-gray-900">{user.name}</div>
								<div class="text-sm text-gray-500">{user.email}</div>
							</div>
						</td>
						<td class="px-6 py-4 text-sm text-gray-700">{user.plan}</td>
						<td class="px-6 py-4">
							<span class="px-2.5 py-1 text-xs font-medium rounded-full {getStatusColor(user.status)}">
								{user.status}
							</span>
						</td>
						<td class="px-6 py-4 text-sm text-gray-500">{user.joinedAt}</td>
						<td class="px-6 py-4 text-right">
							<button class="p-1 text-gray-400 hover:text-gray-600 rounded">
								<MoreVertical class="w-4 h-4" />
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
`,
			"src/lib/components/ActivityFeed.svelte": `<script lang="ts">
	interface Activity {
		id: string;
		user: string;
		action: string;
		timestamp: string;
		type: 'signup' | 'upgrade' | 'payment' | 'cancel';
	}

	let activities = $state<Activity[]>([
		{ id: '1', user: 'Alice', action: 'upgraded to Pro plan', timestamp: '2 min ago', type: 'upgrade' },
		{ id: '2', user: 'Bob', action: 'made a payment of $49', timestamp: '15 min ago', type: 'payment' },
		{ id: '3', user: 'Carol', action: 'signed up for trial', timestamp: '1 hour ago', type: 'signup' },
		{ id: '4', user: 'Dave', action: 'cancelled subscription', timestamp: '2 hours ago', type: 'cancel' },
	]);

	function getTypeColor(type: string): string {
		switch (type) {
			case 'signup': return 'bg-blue-500';
			case 'upgrade': return 'bg-green-500';
			case 'payment': return 'bg-purple-500';
			case 'cancel': return 'bg-red-500';
			default: return 'bg-gray-500';
		}
	}
</script>

<div class="bg-white rounded-xl border border-gray-200 p-6">
	<h3 class="text-lg font-semibold text-gray-900 mb-4">Recent Activity</h3>
	<div class="space-y-4">
		{#each activities as activity (activity.id)}
			<div class="flex items-start gap-3">
				<div class="w-2 h-2 mt-2 rounded-full {getTypeColor(activity.type)}"></div>
				<div class="flex-1">
					<p class="text-sm text-gray-900">
						<span class="font-medium">{activity.user}</span> {activity.action}
					</p>
					<p class="text-xs text-gray-500">{activity.timestamp}</p>
				</div>
			</div>
		{/each}
	</div>
</div>
`,
			"src/app.css": `@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
	--primary: {{primary_color}};
}
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
