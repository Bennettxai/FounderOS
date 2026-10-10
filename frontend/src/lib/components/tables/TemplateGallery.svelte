<script lang="ts">
	/**
	 * TemplateGallery - NocoDB-style template gallery for quick table creation
	 * Features: Category tabs, template cards, preview, quick start
	 */
	import {
		Table2,
		Users,
		Briefcase,
		Package,
		Calendar,
		FileText,
		Target,
		DollarSign,
		CheckSquare,
		MessageSquare,
		BarChart3,
		Boxes,
		Truck,
		Building2,
		GraduationCap,
		Heart,
		Zap,
		ArrowRight,
		Search
	} from 'lucide-svelte';
	import type { ComponentType, SvelteComponent } from 'svelte';

	type IconComponent = ComponentType<SvelteComponent>;

	interface Template {
		id: string;
		name: string;
		description: string;
		icon: IconComponent;
		iconColor: string;
		iconBg: string;
		category: string;
		columns: { name: string; type: string }[];
		rowCount: number;
	}

	interface Props {
		onSelectTemplate: (template: Template) => void;
		onStartBlank: () => void;
	}

	let { onSelectTemplate, onStartBlank }: Props = $props();

	let searchQuery = $state('');
	let selectedCategory = $state('all');

	const categories = [
		{ id: 'all', label: 'All Templates' },
		{ id: 'business', label: 'Business' },
		{ id: 'sales', label: 'Sales & CRM' },
		{ id: 'project', label: 'Project Management' },
		{ id: 'operations', label: 'Operations' },
		{ id: 'hr', label: 'HR & Team' },
		{ id: 'personal', label: 'Personal' }
	];

	const templates: Template[] = [
		{
			id: 'crm-contacts',
			name: 'CRM Contacts',
			description: 'Track customers, leads, and business relationships',
			icon: Users as unknown as IconComponent,
			iconColor: 'text-blue-600',
			iconBg: 'bg-blue-100',
			category: 'sales',
			columns: [
				{ name: 'Name', type: 'text' },
				{ name: 'Email', type: 'email' },
				{ name: 'Phone', type: 'phone' },
				{ name: 'Company', type: 'text' },
				{ name: 'Status', type: 'single_select' },
				{ name: 'Last Contact', type: 'date' }
			],
			rowCount: 0
		},
		{
			id: 'sales-pipeline',
			name: 'Sales Pipeline',
			description: 'Manage deals and track sales opportunities',
			icon: DollarSign as unknown as IconComponent,
			iconColor: 'text-green-600',
			iconBg: 'bg-green-100',
			category: 'sales',
			columns: [
				{ name: 'Deal Name', type: 'text' },
				{ name: 'Company', type: 'text' },
				{ name: 'Value', type: 'currency' },
				{ name: 'Stage', type: 'single_select' },
				{ name: 'Close Date', type: 'date' },
				{ name: 'Owner', type: 'user' }
			],
			rowCount: 0
		},
		{
			id: 'project-tracker',
			name: 'Project Tracker',
			description: 'Organize projects, milestones, and deliverables',
			icon: Briefcase as unknown as IconComponent,
			iconColor: 'text-purple-600',
			iconBg: 'bg-purple-100',
			category: 'project',
			columns: [
				{ name: 'Project', type: 'text' },
				{ name: 'Status', type: 'single_select' },
				{ name: 'Priority', type: 'single_select' },
				{ name: 'Due Date', type: 'date' },
				{ name: 'Assignee', type: 'user' },
				{ name: 'Progress', type: 'percent' }
			],
			rowCount: 0
		},
		{
			id: 'task-list',
			name: 'Task List',
			description: 'Simple task management with priorities and deadlines',
			icon: CheckSquare as unknown as IconComponent,
			iconColor: 'text-indigo-600',
			iconBg: 'bg-indigo-100',
			category: 'project',
			columns: [
				{ name: 'Task', type: 'text' },
				{ name: 'Done', type: 'checkbox' },
				{ name: 'Priority', type: 'single_select' },
				{ name: 'Due Date', type: 'date' },
				{ name: 'Assignee', type: 'user' },
				{ name: 'Notes', type: 'long_text' }
			],
			rowCount: 0
		},
		{
			id: 'inventory',
			name: 'Inventory',
			description: 'Track products, stock levels, and suppliers',
			icon: Package as unknown as IconComponent,
			iconColor: 'text-orange-600',
			iconBg: 'bg-orange-100',
			category: 'operations',
			columns: [
				{ name: 'Product', type: 'text' },
				{ name: 'SKU', type: 'text' },
				{ name: 'Quantity', type: 'number' },
				{ name: 'Price', type: 'currency' },
				{ name: 'Category', type: 'single_select' },
				{ name: 'Supplier', type: 'text' }
			],
			rowCount: 0
		},
		{
			id: 'content-calendar',
			name: 'Content Calendar',
			description: 'Plan and schedule content across channels',
			icon: Calendar as unknown as IconComponent,
			iconColor: 'text-pink-600',
			iconBg: 'bg-pink-100',
			category: 'business',
			columns: [
				{ name: 'Title', type: 'text' },
				{ name: 'Type', type: 'single_select' },
				{ name: 'Channel', type: 'multi_select' },
				{ name: 'Publish Date', type: 'datetime' },
				{ name: 'Status', type: 'single_select' },
				{ name: 'Author', type: 'user' }
			],
			rowCount: 0
		},
		{
			id: 'meeting-notes',
			name: 'Meeting Notes',
			description: 'Document meetings, decisions, and action items',
			icon: MessageSquare as unknown as IconComponent,
			iconColor: 'text-cyan-600',
			iconBg: 'bg-cyan-100',
			category: 'business',
			columns: [
				{ name: 'Meeting', type: 'text' },
				{ name: 'Date', type: 'datetime' },
				{ name: 'Attendees', type: 'multi_select' },
				{ name: 'Notes', type: 'long_text' },
				{ name: 'Action Items', type: 'long_text' },
				{ name: 'Recording', type: 'url' }
			],
			rowCount: 0
		},
		{
			id: 'employee-directory',
			name: 'Employee Directory',
			description: 'Manage team members and organizational structure',
			icon: Building2 as unknown as IconComponent,
			iconColor: 'text-slate-600',
			iconBg: 'bg-slate-100',
			category: 'hr',
			columns: [
				{ name: 'Name', type: 'text' },
				{ name: 'Email', type: 'email' },
				{ name: 'Department', type: 'single_select' },
				{ name: 'Role', type: 'text' },
				{ name: 'Start Date', type: 'date' },
				{ name: 'Manager', type: 'user' }
			],
			rowCount: 0
		},
		{
			id: 'applicant-tracker',
			name: 'Applicant Tracker',
			description: 'Manage job applications and hiring pipeline',
			icon: GraduationCap as unknown as IconComponent,
			iconColor: 'text-amber-600',
			iconBg: 'bg-amber-100',
			category: 'hr',
			columns: [
				{ name: 'Candidate', type: 'text' },
				{ name: 'Position', type: 'single_select' },
				{ name: 'Email', type: 'email' },
				{ name: 'Stage', type: 'single_select' },
				{ name: 'Resume', type: 'attachment' },
				{ name: 'Rating', type: 'rating' }
			],
			rowCount: 0
		},
		{
			id: 'bug-tracker',
			name: 'Bug Tracker',
			description: 'Track issues, bugs, and feature requests',
			icon: Zap as unknown as IconComponent,
			iconColor: 'text-red-600',
			iconBg: 'bg-red-100',
			category: 'project',
			columns: [
				{ name: 'Issue', type: 'text' },
				{ name: 'Type', type: 'single_select' },
				{ name: 'Severity', type: 'single_select' },
				{ name: 'Status', type: 'single_select' },
				{ name: 'Assignee', type: 'user' },
				{ name: 'Description', type: 'long_text' }
			],
			rowCount: 0
		},
		{
			id: 'expense-tracker',
			name: 'Expense Tracker',
			description: 'Track expenses, receipts, and reimbursements',
			icon: BarChart3 as unknown as IconComponent,
			iconColor: 'text-emerald-600',
			iconBg: 'bg-emerald-100',
			category: 'business',
			columns: [
				{ name: 'Description', type: 'text' },
				{ name: 'Amount', type: 'currency' },
				{ name: 'Category', type: 'single_select' },
				{ name: 'Date', type: 'date' },
				{ name: 'Receipt', type: 'attachment' },
				{ name: 'Approved', type: 'checkbox' }
			],
			rowCount: 0
		},
		{
			id: 'asset-inventory',
			name: 'Asset Inventory',
			description: 'Track company assets, equipment, and devices',
			icon: Boxes as unknown as IconComponent,
			iconColor: 'text-violet-600',
			iconBg: 'bg-violet-100',
			category: 'operations',
			columns: [
				{ name: 'Asset', type: 'text' },
				{ name: 'Type', type: 'single_select' },
				{ name: 'Serial Number', type: 'text' },
				{ name: 'Assigned To', type: 'user' },
				{ name: 'Purchase Date', type: 'date' },
				{ name: 'Value', type: 'currency' }
			],
			rowCount: 0
		},
		{
			id: 'order-management',
			name: 'Order Management',
			description: 'Track orders, shipments, and fulfillment',
			icon: Truck as unknown as IconComponent,
			iconColor: 'text-teal-600',
			iconBg: 'bg-teal-100',
			category: 'operations',
			columns: [
				{ name: 'Order ID', type: 'text' },
				{ name: 'Customer', type: 'text' },
				{ name: 'Items', type: 'long_text' },
				{ name: 'Total', type: 'currency' },
				{ name: 'Status', type: 'single_select' },
				{ name: 'Ship Date', type: 'date' }
			],
			rowCount: 0
		},
		{
			id: 'goals-okrs',
			name: 'Goals & OKRs',
			description: 'Track objectives, key results, and progress',
			icon: Target as unknown as IconComponent,
			iconColor: 'text-rose-600',
			iconBg: 'bg-rose-100',
			category: 'business',
			columns: [
				{ name: 'Objective', type: 'text' },
				{ name: 'Key Result', type: 'text' },
				{ name: 'Owner', type: 'user' },
				{ name: 'Progress', type: 'percent' },
				{ name: 'Quarter', type: 'single_select' },
				{ name: 'Status', type: 'single_select' }
			],
			rowCount: 0
		},
		{
			id: 'personal-journal',
			name: 'Personal Journal',
			description: 'Daily journal entries and reflections',
			icon: FileText as unknown as IconComponent,
			iconColor: 'text-sky-600',
			iconBg: 'bg-sky-100',
			category: 'personal',
			columns: [
				{ name: 'Date', type: 'date' },
				{ name: 'Title', type: 'text' },
				{ name: 'Entry', type: 'long_text' },
				{ name: 'Mood', type: 'single_select' },
				{ name: 'Tags', type: 'multi_select' }
			],
			rowCount: 0
		},
		{
			id: 'habit-tracker',
			name: 'Habit Tracker',
			description: 'Track daily habits and build consistency',
			icon: Heart as unknown as IconComponent,
			iconColor: 'text-fuchsia-600',
			iconBg: 'bg-fuchsia-100',
			category: 'personal',
			columns: [
				{ name: 'Habit', type: 'text' },
				{ name: 'Category', type: 'single_select' },
				{ name: 'Frequency', type: 'single_select' },
				{ name: 'Streak', type: 'number' },
				{ name: 'Last Done', type: 'date' },
				{ name: 'Active', type: 'checkbox' }
			],
			rowCount: 0
		}
	];

	const filteredTemplates = $derived(
		templates.filter((t) => {
			const matchesCategory = selectedCategory === 'all' || t.category === selectedCategory;
			const matchesSearch =
				searchQuery === '' ||
				t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				t.description.toLowerCase().includes(searchQuery.toLowerCase());
			return matchesCategory && matchesSearch;
		})
	);
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="border-b border-gray-200 px-6 py-4">
		<h2 class="text-lg font-semibold text-gray-900">Start with a template</h2>
		<p class="mt-1 text-sm text-gray-500">Choose a template to get started quickly</p>
	</div>

	<!-- Search and Categories -->
	<div class="border-b border-gray-200 px-6 py-4">
		<!-- Search -->
		<div class="relative mb-4">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
			<input
				type="text"
				placeholder="Search templates..."
				bind:value={searchQuery}
				class="w-full rounded-lg border border-gray-300 py-2 pl-10 pr-4 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
			/>
		</div>

		<!-- Category Tabs -->
		<div class="flex flex-wrap gap-2">
			{#each categories as category}
				<button
					type="button"
					class="btn-pill btn-pill-sm {selectedCategory === category.id ? 'btn-pill-primary' : 'btn-pill-soft'}"
					onclick={() => (selectedCategory = category.id)}
				>
					{category.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Template Grid -->
	<div class="flex-1 overflow-y-auto p-6">
		<!-- Blank Table Option -->
		<button
			type="button"
			class="mb-6 flex w-full items-center gap-4 border-2 border-dashed p-4 text-left rounded-xl hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
			onclick={onStartBlank}
		>
			<div class="flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100">
				<Table2 class="h-6 w-6 text-gray-500" />
			</div>
			<div class="flex-1">
				<h3 class="font-medium text-gray-900">Start from scratch</h3>
				<p class="text-sm text-gray-500">Create a blank table with custom columns</p>
			</div>
			<ArrowRight class="h-5 w-5 text-gray-400" />
		</button>

		<!-- Templates Grid -->
		{#if filteredTemplates.length === 0}
			<div class="py-12 text-center">
				<Search class="mx-auto h-10 w-10 text-gray-300" />
				<p class="mt-4 text-sm text-gray-500">No templates found matching your search</p>
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each filteredTemplates as template (template.id)}
					<button
						type="button"
						class="btn-pill btn-pill-secondary group flex flex-col p-4 text-left"
						onclick={() => onSelectTemplate(template)}
					>
						<!-- Icon and Title -->
						<div class="mb-3 flex items-start gap-3">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg {template.iconBg}"
							>
								<svelte:component this={template.icon} class="h-5 w-5 {template.iconColor}" />
							</div>
							<div class="min-w-0 flex-1">
								<h3
									class="truncate font-medium text-gray-900 group-hover:text-blue-600"
								>
									{template.name}
								</h3>
								<p class="mt-0.5 line-clamp-2 text-xs text-gray-500">
									{template.description}
								</p>
							</div>
						</div>

						<!-- Column Preview -->
						<div class="flex flex-wrap gap-1">
							{#each template.columns.slice(0, 4) as col}
								<span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
									{col.name}
								</span>
							{/each}
							{#if template.columns.length > 4}
								<span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-400">
									+{template.columns.length - 4} more
								</span>
							{/if}
						</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
