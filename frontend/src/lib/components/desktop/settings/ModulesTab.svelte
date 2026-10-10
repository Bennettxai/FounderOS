<script lang="ts">
	import { onMount } from 'svelte';
	import { getApps } from '$lib/api/apps';
	import { windowStore } from '$lib/stores/windowStore';
	import { listWorkspaceAppMetadata } from '$lib/stores/windowModuleStore';
	import { currentWorkspace } from '$lib/stores/workspaces';

	type ModuleOption = {
		module: string;
		label: string;
		group: string;
	};

	const moduleOptions: ModuleOption[] = [
		{ group: 'Operate', module: 'dashboard', label: 'Command' },
		{ group: 'Operate', module: 'knowledge', label: 'Knowledge' },
		{ group: 'Operate', module: 'intelligence', label: 'Intelligence' },
		{ group: 'Operate', module: 'inbox', label: 'Inbox' },
		{ group: 'Operate', module: 'calendar', label: 'Calendar' },
		{ group: 'Business', module: 'relationships', label: 'Relationships' },
		{ group: 'Business', module: 'projects', label: 'Projects' },
		{ group: 'Business', module: 'tasks', label: 'Tasks' },
		{ group: 'Business', module: 'rhythm', label: 'Rhythm' },
		{ group: 'Business', module: 'pipelines', label: 'Pipelines' },
		{ group: 'Business', module: 'offers', label: 'Offers' },
		{ group: 'Growth', module: 'campaigns', label: 'Campaigns' },
		{ group: 'Growth', module: 'sites', label: 'Sites' },
		{ group: 'Growth', module: 'personas', label: 'Personas' },
		{ group: 'Growth', module: 'content', label: 'Content' },
		{ group: 'Build', module: 'apps', label: 'Apps' },
		{ group: 'Build', module: 'assets', label: 'Assets' },
		{ group: 'Build', module: 'deliverables', label: 'Deliverables' },
		{ group: 'Build', module: 'drive', label: 'Drive' },
		{ group: 'Manage', module: 'finance', label: 'Finance' },
		{ group: 'Manage', module: 'analytics', label: 'Analytics' },
		{ group: 'Manage', module: 'data', label: 'Data' },
		{ group: 'Manage', module: 'team', label: 'Team' },
		{ group: 'Manage', module: 'connectors', label: 'Connectors' },
		{ group: 'Manage', module: 'admin', label: 'Admin' },
	];

	const groups = $derived([...new Set(moduleOptions.map((option) => option.group))]);
	const workspaceApps = $derived(listWorkspaceAppMetadata());
	let appsLoading = $state(false);
	let appsError = $state('');
	let loadedWorkspaceId = $state<string | null>(null);
	const currentDesktopName = $derived(
		$windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId)?.name || 'Desktop'
	);
	const visibleModules = $derived(new Set($windowStore.desktopIcons.map((icon) => icon.module)));

	function toggleModule(option: ModuleOption) {
		if (visibleModules.has(option.module)) {
			windowStore.removeDesktopIconByModule(option.module);
		} else {
			windowStore.addDesktopIcon(option.module, option.label);
		}
	}

	function toggleWorkspaceApp(app: ReturnType<typeof listWorkspaceAppMetadata>[number]) {
		const id = app.module.replace('workspace-app-', '');
		windowStore.placeWorkspaceApp({
			id,
			name: app.name,
			url: app.url,
			launch_mode: app.launch_mode,
			logo_url: app.logo_url,
			color: app.color,
			show_on_desktop: !visibleModules.has(app.module),
			show_in_dock: true
		});
	}

	async function loadWorkspaceApps() {
		const workspaceId = $currentWorkspace?.id || null;
		if (!workspaceId) {
			loadedWorkspaceId = null;
			return;
		}
		appsLoading = true;
		appsError = '';
		try {
			const res = await getApps();
			windowStore.syncWorkspaceApps(res.apps);
			loadedWorkspaceId = workspaceId;
		} catch (error) {
			appsError = error instanceof Error ? error.message : 'Failed to load apps';
		} finally {
			appsLoading = false;
		}
	}

	onMount(() => {
		void loadWorkspaceApps();
	});

	$effect(() => {
		const workspaceId = $currentWorkspace?.id || null;
		if (workspaceId && workspaceId !== loadedWorkspaceId && !appsLoading) {
			void loadWorkspaceApps();
		}
	});
</script>

<section class="modules-tab">
	<div class="settings-header">
		<div>
			<h3>Desktop modules</h3>
			<p>{currentDesktopName} controls which module icons appear on this desktop.</p>
		</div>
		<span>{visibleModules.size} visible</span>
	</div>

	<div class="groups">
		{#each groups as group}
			<section class="group">
				<h4>{group}</h4>
				<div class="module-grid">
					{#each moduleOptions.filter((option) => option.group === group) as option}
						<button
							type="button"
							class="module-toggle"
							class:enabled={visibleModules.has(option.module)}
							onclick={() => toggleModule(option)}
						>
							<span>{option.label}</span>
							<strong>{visibleModules.has(option.module) ? 'Shown' : 'Hidden'}</strong>
						</button>
					{/each}
				</div>
			</section>
		{/each}

		<section class="group">
			<h4>Apps</h4>
			{#if appsLoading && workspaceApps.length === 0}
				<p class="muted">Loading installed apps...</p>
			{:else if appsError && workspaceApps.length === 0}
				<p class="error">{appsError}</p>
			{:else if workspaceApps.length === 0}
				<p class="muted">No workspace apps installed yet.</p>
			{:else}
				<div class="module-grid">
					{#each workspaceApps as app (app.module)}
						<button
							type="button"
							class="module-toggle"
							class:enabled={visibleModules.has(app.module)}
							onclick={() => toggleWorkspaceApp(app)}
						>
							<span>{app.name}</span>
							<strong>{visibleModules.has(app.module) ? 'Shown' : 'Hidden'}</strong>
						</button>
					{/each}
				</div>
			{/if}
		</section>
	</div>
</section>

<style>
	.modules-tab {
		display: flex;
		flex-direction: column;
		min-height: 0;
		height: 100%;
		gap: 14px;
		padding-bottom: 18px;
		color: #111827;
		overflow-y: auto;
		overscroll-behavior: contain;
	}

	.settings-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		padding-bottom: 14px;
		border-bottom: 1px solid #e5e7eb;
	}

	.settings-header h3 {
		margin: 0 0 4px;
		font-size: 16px;
		font-weight: 750;
	}

	.settings-header p {
		margin: 0;
		max-width: 520px;
		color: #6b7280;
		font-size: 12px;
		line-height: 1.45;
	}

	.settings-header span {
		padding: 5px 9px;
		border-radius: 999px;
		background: #f3f4f6;
		color: #4b5563;
		font-size: 11px;
		font-weight: 700;
		white-space: nowrap;
	}

	.groups {
		display: flex;
		flex-direction: column;
		gap: 14px;
		min-height: 0;
	}

	.group h4 {
		margin: 0 0 8px;
		color: #6b7280;
		font-size: 11px;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.module-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(142px, 1fr));
		gap: 8px;
	}

	.module-toggle {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		min-height: 36px;
		padding: 8px 10px;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		background: #fff;
		color: #111827;
		cursor: pointer;
		text-align: left;
	}

	.module-toggle:hover {
		border-color: #cbd5e1;
		background: #f9fafb;
	}

	.module-toggle.enabled {
		border-color: #111827;
		background: #111827;
		color: #fff;
	}

	.module-toggle span {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
		font-size: 12px;
		font-weight: 650;
	}

	.module-toggle strong {
		color: inherit;
		opacity: 0.7;
		font-size: 10px;
		font-weight: 800;
		text-transform: uppercase;
	}

	.muted,
	.error {
		margin: 0;
		font-size: 12px;
		line-height: 1.45;
	}

	.muted {
		color: #6b7280;
	}

	.error {
		color: #dc2626;
	}
</style>
