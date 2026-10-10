<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import type { ProjectRole } from '$lib/api/projects/types';
	import { Shield, Users, Eye, Edit3 } from 'lucide-svelte';

	interface Props {
		value: ProjectRole;
		disabled?: boolean;
		onChange?: (role: ProjectRole) => void;
	}

	let { value = $bindable('viewer'), disabled = false, onChange }: Props = $props();

	const roles: Array<{
		value: ProjectRole;
		label: string;
		description: string;
		icon: typeof Shield;
		color: string;
	}> = [
		{
			value: 'lead',
			label: 'Project Lead',
			description: 'Full control - can manage members and settings',
			icon: Shield,
			color: 'var(--dt, #111)'
		},
		{
			value: 'contributor',
			label: 'Contributor',
			description: 'Can edit and contribute to project',
			icon: Edit3,
			color: 'var(--dt2, #555)'
		},
		{
			value: 'reviewer',
			label: 'Reviewer',
			description: 'Can review and comment',
			icon: Users,
			color: 'var(--dt2, #555)'
		},
		{
			value: 'viewer',
			label: 'Viewer',
			description: 'Read-only access',
			icon: Eye,
			color: 'var(--dt3, #888)'
		}
	];

	const selectedRole = $derived(roles.find((r) => r.value === value) || roles[3]);

	function handleRoleSelect(role: ProjectRole) {
		value = role;
		onChange?.(role);
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class="prm-role-trigger"
		{disabled}
	>
		<div class="flex items-center gap-2">
			<svelte:component this={selectedRole.icon} class="w-4 h-4" style="color: {selectedRole.color}" />
			<span class="prm-role-label">{selectedRole.label}</span>
		</div>
		<svg class="w-4 h-4 prm-role-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</DropdownMenu.Trigger>

	<DropdownMenu.Portal>
		<DropdownMenu.Content
			class="prm-role-dropdown z-50 min-w-[280px] rounded-xl shadow-lg p-1 animate-in fade-in-0 zoom-in-95"
			sideOffset={4}
		>
			{#each roles as role}
				<DropdownMenu.Item
					class="prm-role-item"
					onclick={() => handleRoleSelect(role.value)}
				>
					<div class="flex items-start gap-3">
						<svelte:component this={role.icon} class="w-4 h-4 mt-0.5" style="color: {role.color}" />
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<span class="prm-role-item-name">{role.label}</span>
								{#if role.value === value}
								<svg class="w-4 h-4 prm-role-check" fill="currentColor" viewBox="0 0 20 20">
										<path
											fill-rule="evenodd"
											d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
											clip-rule="evenodd"
										/>
									</svg>
								{/if}
							</div>
							<p class="prm-role-item-desc">{role.description}</p>
						</div>
					</div>
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>

<style>
	.prm-role-trigger {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.5rem;
		transition: background 0.15s;
		min-width: 180px;
	}
	.prm-role-trigger:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-role-trigger:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	.prm-role-label {
		font-weight: 500;
		color: var(--dt, #111);
	}
	.prm-role-chevron {
		color: var(--dt3, #888);
	}
	.prm-role-dropdown {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
	}
	.prm-role-item {
		padding: 0.5rem 0.75rem;
		border-radius: 0.5rem;
		cursor: pointer;
		transition: background 0.15s;
	}
	.prm-role-item:hover {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-role-item-name {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt, #111);
	}
	.prm-role-item-desc {
		font-size: 0.75rem;
		color: var(--dt2, #555);
		margin-top: 0.125rem;
	}
	.prm-role-check { color: var(--dt, #111); }
</style>
