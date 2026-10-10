<script lang="ts">
	import { currentUserRoleContext, currentWorkspace } from '$lib/stores/workspaces';

	interface Props {
		showLabel?: boolean;
		size?: 'sm' | 'md';
		showTooltip?: boolean;
	}

	let { showLabel = true, size = 'sm', showTooltip = true }: Props = $props();

	// Role configuration — monochromatic, token-based
	const roleConfig: Record<string, { color: string; bg: string; icon: string }> = {
		owner: { color: 'role-dot', bg: 'role-badge', icon: '' },
		admin: { color: 'role-dot', bg: 'role-badge', icon: '' },
		editor: { color: 'role-dot', bg: 'role-badge', icon: '' },
		member: { color: 'role-dot', bg: 'role-badge', icon: '' },
		viewer: { color: 'role-dot role-dot-muted', bg: 'role-badge', icon: '' },
		guest: { color: 'role-dot role-dot-muted', bg: 'role-badge', icon: '' }
	};

	// Derive current role configuration
	const currentRole = $derived($currentUserRoleContext?.role_name || 'guest');
	const config = $derived(roleConfig[currentRole] || roleConfig.guest);
	const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2.5 py-1';

	// Build tooltip text
	const tooltipText = $derived.by(() => {
		if (!$currentUserRoleContext) return 'No role context';

		const permissions = Object.entries($currentUserRoleContext.permissions || {})
			.flatMap(([resource, perms]) =>
				Object.entries(perms)
					.filter(([_, value]) => value === true)
					.map(([perm]) => `${resource}.${perm}`)
			);

		return `Role: ${$currentUserRoleContext.role_display_name}\nLevel: ${$currentUserRoleContext.hierarchy_level}\nPermissions: ${permissions.length}`;
	});

	// Show/hide tooltip
	let showTooltipPopup = $state(false);
</script>

{#if $currentUserRoleContext}
	{#if showLabel}
		<div class="relative inline-flex">
			<button
				onmouseenter={() => showTooltip && (showTooltipPopup = true)}
				onmouseleave={() => showTooltipPopup = false}
				onclick={() => showTooltipPopup = !showTooltipPopup}
				class="role-badge-btn {sizeClasses}"
			>
				<span class="role-dot {config.color}"></span>
				<span class="role-name">{$currentUserRoleContext.role_display_name}</span>
				{#if $currentWorkspace}
					<span class="role-workspace">in {$currentWorkspace.name}</span>
				{/if}
			</button>

			{#if showTooltip && showTooltipPopup}
				<div class="role-tooltip">
					<div class="role-tooltip-title">{$currentUserRoleContext.role_display_name}</div>
					<div class="role-tooltip-body">
						<div>Level: {$currentUserRoleContext.hierarchy_level}</div>
						{#if $currentUserRoleContext.title}
							<div>Title: {$currentUserRoleContext.title}</div>
						{/if}
						{#if $currentUserRoleContext.department}
							<div>Department: {$currentUserRoleContext.department}</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<span class="role-dot {config.color}" title={tooltipText}></span>
	{/if}
{/if}

<style>
	.role-badge-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		border-radius: 9999px;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		cursor: default;
		transition: border-color 0.15s;
	}

	.role-badge-btn:hover {
		border-color: var(--dt3);
	}

	.role-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--dt2);
		flex-shrink: 0;
	}

	.role-dot-muted {
		background: var(--dt3);
	}

	.role-name {
		font-size: 12px;
		font-weight: 600;
		color: var(--dt2);
	}

	.role-workspace {
		font-size: 10px;
		color: var(--dt3);
	}

	/* Tooltip */
	.role-tooltip {
		position: absolute;
		top: calc(100% + 8px);
		left: 0;
		width: 200px;
		background: var(--dt);
		color: var(--dbg);
		font-size: 12px;
		border-radius: 10px;
		padding: 12px;
		z-index: 50;
		pointer-events: none;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
	}

	.role-tooltip-title {
		font-weight: 600;
		margin-bottom: 6px;
	}

	.role-tooltip-body {
		display: flex;
		flex-direction: column;
		gap: 3px;
		opacity: 0.75;
	}
</style>
