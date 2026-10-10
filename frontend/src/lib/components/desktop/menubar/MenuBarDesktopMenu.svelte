<script lang="ts">
	import { desktopSettings } from '$lib/stores/desktopStore';
	import { windowStore } from '$lib/stores/windowStore';

	interface Props {
		isOpen: boolean;
		onToggle: () => void;
		onAction: (action: string) => void;
	}

	let { isOpen, onToggle, onAction }: Props = $props();

	const enable3D = $derived($desktopSettings.enable3DDesktop);
	const activeDesktop = $derived(
		$windowStore.desktopSpaces.find((space) => space.id === $windowStore.activeDesktopId)
	);
	const regularDesktops = $derived(
		$windowStore.desktopSpaces.filter((space) =>
			!['infinity desktop', 'workspace desktop'].includes(space.name.toLowerCase())
		)
	);
	const infinityDesktop = $derived(
		$windowStore.desktopSpaces.find((space) => space.name.toLowerCase() === 'infinity desktop')
	);

	function desktopTypeLabel(space: { id: string; name: string; kind: 'personal' | 'team' | 'workspace' }) {
		if (space.name.toLowerCase() === 'infinity desktop') return 'Infinity Canvas';
		if (space.kind === 'workspace' || space.kind === 'team') return 'Workspace Desktop';
		return 'Personal';
	}
</script>

<!-- Logo / Desktop Settings -->
<div class="menu-bar-item-wrapper">
	<button class="menu-bar-logo" onclick={onToggle} aria-label="Desktop menu" aria-haspopup="menu" aria-expanded={isOpen}>
		<svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
			<path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="currentColor" stroke-width="2" fill="none"/>
		</svg>
	</button>

	{#if isOpen}
		<div class="menu-dropdown" role="menu">
			<button class="menu-item" role="menuitem" onclick={() => onAction('desktop-settings')}>
				<span class="menu-item-check">
					<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<circle cx="12" cy="12" r="3"/>
						<path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
					</svg>
				</span>
				<span class="menu-item-label">Desktop Settings...</span>
			</button>

			<div class="menu-separator" role="separator"></div>

			<button class="menu-item experimental-3d" role="menuitem" onclick={() => { desktopSettings.toggle3DDesktop(); onAction(''); }}>
				<span class="menu-item-check">
					{#if enable3D}
						<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
					{:else}
						<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
							<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
							<polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
							<line x1="12" y1="22.08" x2="12" y2="12"/>
						</svg>
					{/if}
				</span>
				<span class="menu-item-label">
					{enable3D ? 'Exit 3D Desktop' : 'Experimental 3D Desktop'}
				</span>
				<span class="menu-item-shortcut beta-badge">Beta</span>
			</button>

			<div class="collab-section" role="group" aria-label="Desktop spaces">
				<div class="collab-heading">
					<span>Desktop Modes</span>
				</div>

				<button class="menu-item collab-primary" role="menuitem" onclick={() => onAction('desktop:open-collab')}>
					<span class="menu-item-check">
						{#if (activeDesktop?.kind === 'workspace' || activeDesktop?.kind === 'team') && activeDesktop?.name.toLowerCase() !== 'infinity desktop'}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
						{:else}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path d="M5 12h14M12 5l7 7-7 7"/>
							</svg>
						{/if}
					</span>
					<span class="menu-item-label">Shared Workspace</span>
					<span class="menu-item-shortcut">Synced</span>
				</button>

				<button class="menu-item collab-primary" class:checked={infinityDesktop?.id === $windowStore.activeDesktopId} role="menuitem" onclick={() => onAction('desktop:open-infinity')}>
					<span class="menu-item-check">
						{#if infinityDesktop?.id === $windowStore.activeDesktopId}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
						{:else}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path d="M4 12h16M12 4v16"/>
								<path d="M7 7h10v10H7z"/>
							</svg>
						{/if}
					</span>
					<span class="menu-item-label">Infinite Canvas</span>
					<span class="menu-item-shortcut">Unbounded</span>
				</button>

				<div class="collab-heading collab-heading--sub">
					<span>Your Desktops</span>
				</div>

				{#each regularDesktops as desktop}
					<button class="menu-item compact" class:checked={desktop.id === $windowStore.activeDesktopId} role="menuitem" onclick={() => onAction(`desktop:switch:${desktop.id}`)}>
						<span class="menu-item-check">
							{#if desktop.id === $windowStore.activeDesktopId}
								<svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true">
									<polyline points="20 6 9 17 4 12"/>
								</svg>
							{/if}
						</span>
						<span class="menu-item-label">{desktop.name}</span>
						<span class="menu-item-shortcut">{desktopTypeLabel(desktop)}</span>
					</button>
				{/each}

				<button class="menu-item compact" role="menuitem" onclick={() => onAction('new-desktop')}>
					<span class="menu-item-check"></span>
					<span class="menu-item-label">New Desktop...</span>
				</button>
			</div>

			<div class="menu-separator" role="separator"></div>

			<button class="menu-item" role="menuitem" onclick={() => onAction('open-terminal')}>
				<span class="menu-item-check">
					<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M4 17l6-6-6-6M12 19h8"/>
					</svg>
				</span>
				<span class="menu-item-label">Open Terminal</span>
			</button>

			<div class="menu-separator" role="separator"></div>

			<button class="menu-item" role="menuitem" onclick={() => onAction('exit-desktop')}>
				<span class="menu-item-check"></span>
				<span class="menu-item-label">Exit Desktop View</span>
			</button>
		</div>
	{/if}
</div>

<style>
	.menu-bar-item-wrapper {
		position: relative;
	}

	.menu-bar-logo {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px 10px;
		border-radius: 4px;
		background: none;
		border: none;
		cursor: pointer;
		color: #333;
	}

	.menu-bar-logo:hover {
		background: rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .menu-bar-logo {
		color: #f5f5f7;
	}

	:global(.dark) .menu-bar-logo:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.collab-section {
		padding: 4px 0;
	}

	.collab-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 5px 12px 4px 32px;
		color: #6b7280;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.collab-heading strong {
		color: #059669;
		font-size: 9px;
	}

	.collab-heading--sub {
		margin-top: 4px;
	}

	:global(.menu-item.collab-primary) {
		font-weight: 700;
	}

	:global(.menu-item.compact) {
		padding-top: 4px;
		padding-bottom: 4px;
		font-size: 12px;
	}
</style>
