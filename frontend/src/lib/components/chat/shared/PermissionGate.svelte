<script lang="ts">
	import { currentUserRoleContext } from '$lib/stores/workspaces';
	import type { Snippet } from 'svelte';

	interface Props {
		/**
		 * Required permission resource (e.g., "agents", "projects", "contexts")
		 */
		resource?: string;

		/**
		 * Required permission action (e.g., "create", "edit", "delete")
		 */
		permission?: string;

		/**
		 * Minimum hierarchy level required (0-5, where 0 is highest/owner)
		 * User must have this level or lower (higher authority)
		 */
		minLevel?: number;

		/**
		 * Maximum hierarchy level allowed (0-5)
		 * User must have this level or higher (lower authority)
		 */
		maxLevel?: number;

		/**
		 * Show fallback content when permission is denied
		 */
		showFallback?: boolean;

		/**
		 * Fallback message to show when permission is denied
		 */
		fallbackMessage?: string;

		/**
		 * Children to render when permission is granted
		 */
		children?: Snippet;

		/**
		 * Fallback snippet to render when permission is denied
		 */
		fallback?: Snippet;
	}

	let {
		resource,
		permission,
		minLevel,
		maxLevel,
		showFallback = false,
		fallbackMessage = 'You do not have permission to access this feature.',
		children,
		fallback
	}: Props = $props();

	// Check if user has the required permission
	const hasRequiredPermission = $derived.by(() => {
		if (!$currentUserRoleContext) return false;

		// If no specific permission required, just check if role context exists
		if (!resource && !permission && minLevel === undefined && maxLevel === undefined) {
			return true;
		}

		// Check resource.permission if specified
		if (resource && permission) {
			const hasPermission = $currentUserRoleContext.permissions?.[resource]?.[permission];
			if (!hasPermission) return false;
		}

		// Check minimum level (user must be at or below this level for higher authority)
		if (minLevel !== undefined) {
			if ($currentUserRoleContext.hierarchy_level > minLevel) return false;
		}

		// Check maximum level (user must be at or above this level for lower authority)
		if (maxLevel !== undefined) {
			if ($currentUserRoleContext.hierarchy_level < maxLevel) return false;
		}

		return true;
	});
</script>

{#if hasRequiredPermission}
	{@render children?.()}
{:else if showFallback}
	{#if fallback}
		{@render fallback()}
	{:else}
		<div class="text-sm text-gray-500 p-4 bg-gray-50 rounded-lg border border-gray-200">
			<div class="flex items-start gap-2">
				<svg class="w-5 h-5 text-gray-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
				</svg>
				<div>
					<div class="font-medium text-gray-700">Permission Required</div>
					<div class="text-gray-600 mt-1">{fallbackMessage}</div>
					{#if $currentUserRoleContext}
						<div class="text-xs text-gray-500 mt-2">
							Your role: <span class="font-medium">{$currentUserRoleContext.role_display_name}</span>
						</div>
					{/if}
				</div>
			</div>
		</div>
	{/if}
{/if}
