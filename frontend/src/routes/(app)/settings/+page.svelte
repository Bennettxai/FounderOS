<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { page } from '$app/stores';
	import {
		currentWorkspace,
		currentWorkspaceId,
		currentUserRole,
		loadSavedWorkspace
	} from '$lib/stores/workspaces';
	import { User, Building2, Users, Shield, Cpu, Landmark, UsersRound, Boxes, RefreshCw, Sparkles } from 'lucide-svelte';
	import ProfileSection from '$lib/components/settings/ProfileSection.svelte';
	import OrganizationSection from '$lib/components/settings/OrganizationSection.svelte';
	import WorkspaceSection from '$lib/components/settings/WorkspaceSection.svelte';
	import MembersSection from '$lib/components/settings/MembersSection.svelte';
	import TeamsSection from '$lib/components/settings/TeamsSection.svelte';
	import RolesSection from '$lib/components/settings/RolesSection.svelte';
	import EngineSection from '$lib/components/settings/EngineSection.svelte';
	import ModelConnectionPanel from '$lib/components/settings/ModelConnectionPanel.svelte';
	import EngineStatusPanel from '$lib/components/settings/EngineStatusPanel.svelte';
	import ModulesSection from '$lib/components/settings/ModulesSection.svelte';
	import SyncSection from '$lib/components/settings/SyncSection.svelte';

	type TabId = 'profile' | 'organization' | 'workspace' | 'members' | 'teams' | 'roles' | 'engine' | 'model' | 'enginestatus' | 'modules' | 'sync';

	const tabs: { id: TabId; label: string; icon: any; desc: string }[] = [
		{ id: 'profile', label: 'Profile', icon: User, desc: 'Your account' },
		{ id: 'organization', label: 'Organization', icon: Landmark, desc: 'The account' },
		{ id: 'workspace', label: 'Workspace', icon: Building2, desc: 'Name and details' },
		{ id: 'members', label: 'Members', icon: Users, desc: 'Workspace access' },
		{ id: 'teams', label: 'Teams', icon: UsersRound, desc: 'Groups of people' },
		{ id: 'roles', label: 'Roles', icon: Shield, desc: 'Permissions' },
		{ id: 'sync', label: 'Sync', icon: RefreshCw, desc: 'Local vs cloud' },
		{ id: 'modules', label: 'Modules', icon: Boxes, desc: 'Sharing & scope' },
		{ id: 'engine', label: 'Optimal Engine', icon: Cpu, desc: 'Connect your brain' },
		{ id: 'model', label: 'Model', icon: Sparkles, desc: 'AI model connection' },
		{ id: 'enginestatus', label: 'Built-in Engine', icon: Cpu, desc: 'Status and data files' }
	];

	// Allow deep-linking to a tab: /settings?tab=engine
	let active = $state<TabId>('profile');
	$effect(() => {
		const t = $page.url.searchParams.get('tab') as TabId | null;
		if (t && tabs.some((x) => x.id === t)) active = t;
	});

	let bootError = $state<string | null>(null);

	onMount(async () => {
		if (!browser) return;
		if (!$currentWorkspaceId) {
			try {
				await loadSavedWorkspace();
			} catch (e) {
				bootError = e instanceof Error ? e.message : 'Failed to load workspace';
			}
		}
	});

	const wsId = $derived($currentWorkspaceId);
	const role = $derived($currentUserRole ?? 'member');
	const canManage = $derived(['owner', 'admin', 'manager'].includes(role));
</script>

<div class="settings-root">
	<aside class="settings-nav">
		<div class="settings-brand">
			<h1>Settings</h1>
			{#if $currentWorkspace}
				<p class="ws-name">{$currentWorkspace.name}</p>
			{/if}
		</div>
		<nav>
			{#each tabs as tab}
				{@const Ico = tab.icon}
				<button
					class="nav-item {active === tab.id ? 'nav-item--active' : ''}"
					onclick={() => (active = tab.id)}
				>
					<Ico size={17} strokeWidth={1.8} class="flex-shrink-0" />
					<span class="nav-text">
						<span class="nav-label">{tab.label}</span>
						<span class="nav-desc">{tab.desc}</span>
					</span>
				</button>
			{/each}
		</nav>
		<div class="role-badge">
			<span class="role-dot"></span>
			You are <strong>{role}</strong>
		</div>
	</aside>

	<main class="settings-content">
		{#if bootError}
			<div class="banner banner--error">{bootError}</div>
		{:else if !wsId}
			<div class="empty-state">
				<Cpu size={28} strokeWidth={1.5} />
				<p>Loading your workspace…</p>
			</div>
		{:else}
			{#key active + wsId}
				<div class="section-wrap">
					{#if active === 'profile'}
						<ProfileSection />
					{:else if active === 'organization'}
						<OrganizationSection />
					{:else if active === 'workspace'}
						<WorkspaceSection workspaceId={wsId} {canManage} />
					{:else if active === 'members'}
						<MembersSection workspaceId={wsId} {role} />
					{:else if active === 'teams'}
						<TeamsSection workspaceId={wsId} {canManage} />
					{:else if active === 'roles'}
						<RolesSection workspaceId={wsId} />
					{:else if active === 'sync'}
						<SyncSection workspaceId={wsId} {canManage} />
					{:else if active === 'modules'}
						<ModulesSection workspaceId={wsId} {canManage} />
					{:else if active === 'engine'}
						<EngineSection workspaceId={wsId} {canManage} />
					{:else if active === 'model'}
						<ModelConnectionPanel />
					{:else if active === 'enginestatus'}
						<EngineStatusPanel />
					{/if}
				</div>
			{/key}
		{/if}
	</main>
</div>

<style>
	.settings-root {
		display: flex;
		height: 100%;
		width: 100%;
		overflow: hidden;
		background: var(--dbg);
		color: var(--dt);
	}

	/* ── Left nav ── */
	.settings-nav {
		width: 248px;
		flex-shrink: 0;
		border-right: 1px solid var(--dbd);
		display: flex;
		flex-direction: column;
		padding: 20px 12px 12px;
		gap: 4px;
		overflow-y: auto;
	}
	.settings-brand {
		padding: 0 8px 14px;
	}
	.settings-brand h1 {
		font-size: 1.15rem;
		font-weight: 680;
		letter-spacing: -0.02em;
		margin: 0;
	}
	.ws-name {
		margin: 2px 0 0;
		font-size: 0.78rem;
		color: var(--dt3);
	}
	.nav-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 9px 10px;
		border-radius: 9px;
		border: none;
		background: transparent;
		color: var(--dt2);
		cursor: pointer;
		text-align: left;
		transition: background 140ms ease, color 140ms ease;
	}
	.nav-item:hover {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
		color: var(--dt);
	}
	.nav-item--active {
		background: color-mix(in srgb, var(--dt) 9%, transparent);
		color: var(--dt);
	}
	.nav-text {
		display: flex;
		flex-direction: column;
		line-height: 1.2;
		min-width: 0;
	}
	.nav-label {
		font-size: 0.88rem;
		font-weight: 550;
	}
	.nav-desc {
		font-size: 0.7rem;
		color: var(--dt3);
	}
	.role-badge {
		margin-top: auto;
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 9px 10px;
		font-size: 0.74rem;
		color: var(--dt3);
		border-top: 1px solid var(--dbd);
	}
	.role-badge strong {
		color: var(--dt);
		text-transform: capitalize;
	}
	.role-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: #22c55e;
	}

	/* ── Content ── */
	.settings-content {
		flex: 1;
		overflow-y: auto;
		min-width: 0;
	}
	.section-wrap {
		max-width: 760px;
		margin: 0 auto;
		padding: 32px 40px 64px;
	}
	.empty-state {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		color: var(--dt3);
	}
	.banner {
		margin: 24px;
		padding: 12px 16px;
		border-radius: 10px;
		font-size: 0.85rem;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
		border: 1px solid color-mix(in srgb, #ef4444 30%, transparent);
	}

	/* ── Mobile: 768px and below ── */
	@media (max-width: 768px) {
		.settings-root {
			flex-direction: column;
			overflow: auto;
		}

		/* Convert vertical sidebar into horizontal scroll bar */
		.settings-nav {
			width: 100%;
			flex-shrink: 0;
			border-right: none;
			border-bottom: 1px solid var(--dbd);
			flex-direction: row;
			padding: 0;
			gap: 0;
			overflow-x: auto;
			overflow-y: hidden;
			/* Hide scrollbar visually but keep it scrollable */
			scrollbar-width: none;
		}
		.settings-nav::-webkit-scrollbar {
			display: none;
		}

		/* Hide the brand block on mobile - too much vertical space */
		.settings-brand {
			display: none;
		}

		/* Role badge pinned at the right end of the scroll bar */
		.role-badge {
			margin-top: 0;
			border-top: none;
			border-left: 1px solid var(--dbd);
			flex-shrink: 0;
			white-space: nowrap;
			padding: 0 14px;
		}

		/* Nav items become horizontal tabs */
		nav {
			display: flex;
			flex-direction: row;
			flex: 1;
		}

		.nav-item {
			flex-direction: column;
			align-items: center;
			justify-content: center;
			gap: 4px;
			padding: 10px 14px;
			border-radius: 0;
			white-space: nowrap;
			flex-shrink: 0;
			border-bottom: 2px solid transparent;
		}
		.nav-item:hover {
			background: color-mix(in srgb, var(--dt) 6%, transparent);
		}
		.nav-item--active {
			background: transparent;
			border-bottom-color: var(--dt);
			color: var(--dt);
		}

		/* Hide the description under the label on mobile */
		.nav-desc {
			display: none;
		}
		.nav-text {
			flex-direction: row;
		}
		.nav-label {
			font-size: 0.82rem;
		}

		/* Content fills the remaining height */
		.settings-content {
			flex: 1;
			min-height: 0;
		}

		.section-wrap {
			padding: 20px 16px 48px;
		}
	}

	/* ── Narrowest: 480px and below ── */
	@media (max-width: 480px) {
		.section-wrap {
			padding: 16px 12px 40px;
		}

		.nav-item {
			padding: 10px 11px;
		}
		.nav-label {
			font-size: 0.78rem;
		}
	}
</style>
