package services

// --- Task Manager Template ---
func taskManagerTemplate() *BuiltInTemplate {
	return &BuiltInTemplate{
		ID:          "task_manager",
		Name:        "Task Manager",
		Description: "Kanban-style task board with drag-and-drop, labels, due dates, and team assignment",
		Category:    "project_management",
		StackType:   "svelte",
		ConfigSchema: map[string]ConfigField{
			"app_name":      {Type: "string", Label: "App Name", Default: "Task Board", Required: true},
			"primary_color": {Type: "string", Label: "Primary Color", Default: "#8B5CF6", Required: false},
			"columns":       {Type: "string", Label: "Board Columns (comma-separated)", Default: "Backlog,To Do,In Progress,Review,Done", Required: false},
			"labels":        {Type: "string", Label: "Labels (comma-separated)", Default: "Bug,Feature,Enhancement,Urgent", Required: false},
		},
		FilesTemplate: map[string]string{
			"src/routes/+page.svelte": `<script lang="ts">
	import KanbanBoard from '$lib/components/KanbanBoard.svelte';
	import TaskModal from '$lib/components/TaskModal.svelte';
	import BoardHeader from '$lib/components/BoardHeader.svelte';

	let showModal = $state(false);
	let selectedTask = $state<any>(null);

	function handleAddTask() {
		selectedTask = null;
		showModal = true;
	}

	function handleEditTask(task: any) {
		selectedTask = task;
		showModal = true;
	}

	function handleCloseModal() {
		showModal = false;
		selectedTask = null;
	}
</script>

<svelte:head>
	<title>{{app_name}}</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 flex flex-col">
	<BoardHeader onAddTask={handleAddTask} />
	<div class="flex-1 overflow-hidden">
		<KanbanBoard onEditTask={handleEditTask} />
	</div>
	{#if showModal}
		<TaskModal task={selectedTask} onClose={handleCloseModal} />
	{/if}
</div>
`,
			"src/lib/components/BoardHeader.svelte": `<script lang="ts">
	import { Plus, Search, Filter } from 'lucide-svelte';

	interface Props {
		onAddTask: () => void;
	}

	let { onAddTask }: Props = $props();
	let searchQuery = $state('');
</script>

<header class="bg-white border-b border-gray-200 px-6 py-4">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-gray-900">{{app_name}}</h1>
		<div class="flex items-center gap-3">
			<div class="relative">
				<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
				<input
					type="text"
					placeholder="Search tasks..."
					bind:value={searchQuery}
					class="pl-10 pr-4 py-2 border border-gray-300 rounded-lg text-sm w-64"
				/>
			</div>
			<button class="flex items-center gap-2 px-4 py-2 text-white text-sm font-medium rounded-lg" style="background-color: {{primary_color}}" onclick={onAddTask}>
				<Plus class="w-4 h-4" />
				Add Task
			</button>
		</div>
	</div>
</header>
`,
			"src/lib/components/KanbanBoard.svelte": `<script lang="ts">
	import { GripVertical, Clock, Tag } from 'lucide-svelte';

	interface Task {
		id: string;
		title: string;
		description: string;
		column: string;
		labels: string[];
		assignee: string;
		dueDate: string | null;
		priority: 'low' | 'medium' | 'high';
	}

	interface Props {
		onEditTask: (task: Task) => void;
	}

	let { onEditTask }: Props = $props();

	const columns = '{{columns}}'.split(',').map(c => c.trim());
	const availableLabels = '{{labels}}'.split(',').map(l => l.trim());

	let tasks = $state<Task[]>([
		{ id: '1', title: 'Design user dashboard', description: 'Create wireframes and mockups', column: 'In Progress', labels: ['Feature'], assignee: 'Alice', dueDate: '2025-02-15', priority: 'high' },
		{ id: '2', title: 'Fix login redirect', description: 'Users are not redirected after login', column: 'To Do', labels: ['Bug', 'Urgent'], assignee: 'Bob', dueDate: '2025-02-10', priority: 'high' },
		{ id: '3', title: 'Add dark mode support', description: 'Implement theme switching', column: 'Backlog', labels: ['Enhancement'], assignee: 'Carol', dueDate: null, priority: 'low' },
		{ id: '4', title: 'API rate limiting', description: 'Add rate limiting to public endpoints', column: 'Review', labels: ['Feature'], assignee: 'Dave', dueDate: '2025-02-12', priority: 'medium' },
		{ id: '5', title: 'Update documentation', description: 'Add API docs for v2 endpoints', column: 'Done', labels: ['Enhancement'], assignee: 'Eve', dueDate: null, priority: 'low' },
	]);

	function getTasksForColumn(column: string): Task[] {
		return tasks.filter(t => t.column === column);
	}

	function getLabelColor(label: string): string {
		const colors: Record<string, string> = {
			'Bug': 'bg-red-100 text-red-700',
			'Feature': 'bg-blue-100 text-blue-700',
			'Enhancement': 'bg-green-100 text-green-700',
			'Urgent': 'bg-orange-100 text-orange-700'
		};
		return colors[label] || 'bg-gray-100 text-gray-700';
	}

	function getPriorityColor(priority: string): string {
		switch (priority) {
			case 'high': return 'border-l-red-500';
			case 'medium': return 'border-l-yellow-500';
			default: return 'border-l-blue-500';
		}
	}
</script>

<div class="flex gap-4 p-6 overflow-x-auto h-full">
	{#each columns as column}
		<div class="flex-shrink-0 w-72 flex flex-col bg-gray-100 rounded-xl">
			<div class="flex items-center justify-between px-4 py-3">
				<div class="flex items-center gap-2">
					<h3 class="font-semibold text-gray-900 text-sm">{column}</h3>
					<span class="text-xs text-gray-500 bg-gray-200 px-2 py-0.5 rounded-full">
						{getTasksForColumn(column).length}
					</span>
				</div>
			</div>
			<div class="flex-1 overflow-y-auto px-3 pb-3 space-y-2">
				{#each getTasksForColumn(column) as task (task.id)}
					<div
						class="bg-white rounded-lg p-3 border border-gray-200 border-l-4 {getPriorityColor(task.priority)} shadow-sm hover:shadow-md transition-shadow cursor-pointer"
						onclick={() => onEditTask(task)}
					>
						<div class="flex items-start justify-between mb-2">
							<h4 class="font-medium text-gray-900 text-sm flex-1">{task.title}</h4>
							<GripVertical class="w-4 h-4 text-gray-300 flex-shrink-0" />
						</div>
						{#if task.description}
							<p class="text-xs text-gray-500 mb-2 line-clamp-2">{task.description}</p>
						{/if}
						{#if task.labels.length > 0}
							<div class="flex flex-wrap gap-1 mb-2">
								{#each task.labels as label}
									<span class="px-1.5 py-0.5 text-xs font-medium rounded {getLabelColor(label)}">
										{label}
									</span>
								{/each}
							</div>
						{/if}
						<div class="flex items-center justify-between text-xs text-gray-400">
							<span>{task.assignee}</span>
							{#if task.dueDate}
								<div class="flex items-center gap-1">
									<Clock class="w-3 h-3" />
									<span>{task.dueDate}</span>
								</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/each}
</div>
`,
			"src/lib/components/TaskModal.svelte": `<script lang="ts">
	import { X } from 'lucide-svelte';

	interface Task {
		id: string;
		title: string;
		description: string;
		column: string;
		labels: string[];
		assignee: string;
		dueDate: string | null;
		priority: 'low' | 'medium' | 'high';
	}

	interface Props {
		task: Task | null;
		onClose: () => void;
	}

	let { task, onClose }: Props = $props();

	const columns = '{{columns}}'.split(',').map(c => c.trim());
	const availableLabels = '{{labels}}'.split(',').map(l => l.trim());

	let title = $state(task?.title || '');
	let description = $state(task?.description || '');
	let column = $state(task?.column || columns[0]);
	let assignee = $state(task?.assignee || '');
	let dueDate = $state(task?.dueDate || '');
	let priority = $state<'low' | 'medium' | 'high'>(task?.priority || 'medium');

	function handleSubmit(e: Event) {
		e.preventDefault();
		// Save task logic here
		onClose();
	}
</script>

<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={onClose}>
	<div class="bg-white rounded-xl shadow-2xl w-full max-w-lg mx-4" onclick|stopPropagation>
		<div class="flex items-center justify-between p-6 border-b border-gray-200">
			<h2 class="text-lg font-semibold text-gray-900">
				{task ? 'Edit Task' : 'New Task'}
			</h2>
			<button onclick={onClose} class="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100">
				<X class="w-5 h-5" />
			</button>
		</div>
		<form onsubmit={handleSubmit} class="p-6 space-y-4">
			<div>
				<label for="title" class="block text-sm font-medium text-gray-700 mb-1">Title</label>
				<input id="title" type="text" bind:value={title} required class="w-full px-4 py-2 border border-gray-300 rounded-lg" />
			</div>
			<div>
				<label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
				<textarea id="description" bind:value={description} rows="3" class="w-full px-4 py-2 border border-gray-300 rounded-lg"></textarea>
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="column" class="block text-sm font-medium text-gray-700 mb-1">Column</label>
					<select id="column" bind:value={column} class="w-full px-4 py-2 border border-gray-300 rounded-lg">
						{#each columns as col}
							<option value={col}>{col}</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="priority" class="block text-sm font-medium text-gray-700 mb-1">Priority</label>
					<select id="priority" bind:value={priority} class="w-full px-4 py-2 border border-gray-300 rounded-lg">
						<option value="low">Low</option>
						<option value="medium">Medium</option>
						<option value="high">High</option>
					</select>
				</div>
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="assignee" class="block text-sm font-medium text-gray-700 mb-1">Assignee</label>
					<input id="assignee" type="text" bind:value={assignee} class="w-full px-4 py-2 border border-gray-300 rounded-lg" />
				</div>
				<div>
					<label for="due-date" class="block text-sm font-medium text-gray-700 mb-1">Due Date</label>
					<input id="due-date" type="date" bind:value={dueDate} class="w-full px-4 py-2 border border-gray-300 rounded-lg" />
				</div>
			</div>
			<div class="flex justify-end gap-3 pt-4">
				<button type="button" onclick={onClose} class="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200">
					Cancel
				</button>
				<button type="submit" class="px-4 py-2 text-sm text-white rounded-lg" style="background-color: {{primary_color}}">
					{task ? 'Save Changes' : 'Create Task'}
				</button>
			</div>
		</form>
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
