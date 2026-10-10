<!-- The operator chrome for every /os view (FounderOS v1 app/layout.tsx, template.tsx,
     loading.tsx): fixed Sidebar, sticky Topbar, ⌘K palette, the toast stack,
     and the Conductor dock, which pushes the content column aside. Each view
     remounts into the slide-up .bn-view entrance, and a click to another view
     shows the page skeleton at once while it loads. The hover lens and the
     page-wide spotlight run here only, so BusinessOS never carries them. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { navigating, page } from '$app/state';
	import '$lib/founderos/kit/kit.css';
	import { installLens } from '$lib/founderos/kit/lens';
	import PageSpotlight from '$lib/founderos/kit/PageSpotlight.svelte';
	import CommandPalette from '$lib/founderos/chrome/CommandPalette.svelte';
	import PageSkeleton from '$lib/founderos/chrome/PageSkeleton.svelte';
	import Toaster from '$lib/founderos/chrome/Toaster.svelte';
	import CohortBanner from '$lib/founderos/chrome/CohortBanner.svelte';
	import CohortModal from '$lib/founderos/chrome/CohortModal.svelte';
	import SetupNotice from '$lib/founderos/chrome/SetupNotice.svelte';
	import { needsSetup } from '$lib/founderos/chrome/chrome';
	import osIcon from '$lib/founderos/chrome/os-icon.png';
	import ConductorPanel from '$lib/founderos/chrome/ConductorPanel.svelte';
	import Sidebar from '$lib/founderos/chrome/Sidebar.svelte';
	import Topbar from '$lib/founderos/chrome/Topbar.svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { summarizeConnections, type ConnectionsBody } from '$lib/founderos/connections';
	import { WorkspaceSwitcher } from '$lib/components/workspace';
	import { currentWorkspace, loadSavedWorkspace } from '$lib/stores/workspaces';
	import { founderosHrefsFor } from '$lib/founderos/commandModules';
	import { isElectron, isMacOS } from '$lib/utils/platform';
	import { applyFounderosTheme, leaveFounderosTheme, readStoredTheme } from '$lib/founderos/theme/theme';
	import { themeStore } from '$lib/stores/themeStore';

	let { children } = $props();

	let live = $state<{ up: number; total: number } | null>(null);
	// The owner gate said this account has no Founder OS yet: show the setup panel.
	let setup = $state(false);
	let host = $state<string | null>(null);
	// Top-level Electron window on macOS only; a desktop window's iframe has none.
	let trafficLights = $state(false);
	// Each workspace shows only the the operator pages switched on for it.
	const visibleHrefs = $derived($currentWorkspace ? founderosHrefsFor($currentWorkspace.settings ?? {}) : null);
	// loading.tsx: the old page never sits there looking dead after a click.
	const loadingOther = $derived(!!navigating.to && navigating.to.url.pathname !== page.url.pathname);

	onMount(() => {
		host = window.location.host;
		const stopLens = installLens(document);
		// The tab wears the OS emblem (FounderOS v1 app/icon.png): BusinessOS's own
		// icons stand down while /os is open and come back on the way out.
		const bosIcons = [...document.head.querySelectorAll<HTMLLinkElement>('link[rel="icon"]:not([data-founderos])')];
		for (const l of bosIcons) l.rel = 'bos-icon';
		trafficLights = isElectron() && isMacOS() && window.top === window;
		// Monolith is /os-only; client-side navigation in or out must switch it.
		applyFounderosTheme(readStoredTheme());
		// The workspace switcher needs the list and the saved choice, which the
		// (app) shell restores and this layout does not share.
		loadSavedWorkspace().catch(() => {});
		founderosFetch<ConnectionsBody>('/connections')
			.then(({ connections }) => {
				if (Array.isArray(connections)) live = summarizeConnections(connections);
			})
			.catch((err: unknown) => {
				if (needsSetup(err)) setup = true;
				// unknown stays unknown (—/—); the 401 case already redirected
			});
		return () => {
			stopLens();
			for (const l of bosIcons) l.rel = 'icon';
			leaveFounderosTheme(themeStore.isDark());
		};
	});
</script>

<svelte:head>
	<title>FOUNDER OS</title>
	<link rel="icon" type="image/png" sizes="128x128" href={osIcon} data-founderos />
</svelte:head>

<div class="bn-os min-h-screen">
	<PageSpotlight />
	<Sidebar pathname={page.url.pathname} {live} {host} {trafficLights} {visibleHrefs}>
		{#snippet workspace()}<WorkspaceSwitcher />{/snippet}
	</Sidebar>
	<div class="bn-shell flex min-h-screen min-w-0 flex-col">
		<Topbar pathname={page.url.pathname} />
		<main class="min-w-0 flex-1 px-6 pb-12 pt-5 max-[600px]:px-4">
			<!-- Fluid: the page fills the column beside the sidebar at every width. -->
			<div data-part="page-frame" class="w-full min-w-0">
				{#if setup}
					<SetupNotice email={String((page.data?.user as { email?: unknown } | undefined)?.email ?? '')} />
				{:else if loadingOther}
					<PageSkeleton />
				{:else}
					{#key page.url.pathname}
						<div class="bn-view">{@render children()}</div>
					{/key}
				{/if}
				<!-- Cohort invite: the last thing on every view, by construction -->
				<CohortBanner />
			</div>
		</main>
	</div>
	<CommandPalette navigate={(href) => goto(href)} />
	<ConductorPanel pathname={page.url.pathname} />
	<Toaster />
	<CohortModal pathname={page.url.pathname} />
</div>

<style>
	.bn-os {
		background: var(--bn-bg);
		/* FounderOS v1 body: a 48px grid off --bn-grid (transparent on Monolith) */
		background-image:
			repeating-linear-gradient(0deg, var(--bn-grid) 0 1px, transparent 1px 48px),
			repeating-linear-gradient(90deg, var(--bn-grid) 0 1px, transparent 1px 48px);
		color: var(--bn-text);
		font-family: var(--bn-font);
	}
	.bn-shell {
		margin-left: var(--bn-sidebar-w, 232px);
		margin-right: var(--bn-conductor-w, 0px);
		transition: margin-right 420ms var(--bn-ease);
	}
	:global(html.bn-conductor-dragging) .bn-shell {
		transition: none;
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-shell {
			transition: none;
		}
	}
</style>
