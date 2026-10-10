<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { X, Loader2, Layers } from 'lucide-svelte';
	import { goto } from '$app/navigation';
	import { getCSRFToken } from '$lib/api/base';

	interface Props {
		workspaceId: string;
		open?: boolean;
	}

	let { workspaceId, open = $bindable(false) }: Props = $props();

	let moduleName = $state('');
	let description = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state('');
	let formErrors = $state<{ name?: string; description?: string }>({});

	const isValidUUID = (str: string) =>
		/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(str);

	const hasValidWorkspace = $derived(workspaceId && isValidUUID(workspaceId));

	async function handleSubmit() {
		formErrors = {};
		if (!moduleName.trim()) formErrors = { ...formErrors, name: 'Module name is required' };
		if (!description.trim()) formErrors = { ...formErrors, description: 'Description is required' };
		if (formErrors.name || formErrors.description) return;

		if (!hasValidWorkspace) {
			errorMessage = 'Please select a valid workspace first.';
			return;
		}

		isSubmitting = true;
		errorMessage = '';

		try {
			const csrfToken = getCSRFToken();
			const headers: Record<string, string> = { 'Content-Type': 'application/json' };
			if (csrfToken) headers['X-CSRF-Token'] = csrfToken;

			const response = await fetch(`/api/v1/workspaces/${workspaceId}/apps`, {
				method: 'POST',
				headers,
				credentials: 'include',
				body: JSON.stringify({
					app_name: moduleName.trim(),
					description: description.trim(),
				})
			});

			if (!response.ok) {
				const errorText = await response.text();
				let error;
				try { error = JSON.parse(errorText); } catch { error = { message: errorText || 'Unknown error' }; }
				throw new Error(error.details || error.message || `Server error: ${response.status}`);
			}

			const data = await response.json();
			handleClose();
			if (data.app_id || data.id) {
				goto(`/apps/${data.app_id || data.id}`);
			}
		} catch (err) {
			errorMessage = err instanceof Error ? err.message : 'Failed to create module';
		} finally {
			isSubmitting = false;
		}
	}

	function handleClose() {
		open = false;
		setTimeout(() => {
			moduleName = '';
			description = '';
			errorMessage = '';
			formErrors = {};
			isSubmitting = false;
		}, 300);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm" />
		<Dialog.Content
			class="fixed left-[50%] top-[50%] z-50 w-[90vw] max-w-md translate-x-[-50%] translate-y-[-50%] rounded-2xl border border-gray-200 bg-white p-6 shadow-lg dark:border-gray-700 dark:bg-gray-800"
		>
			<div class="flex items-center justify-between mb-5">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-xl bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
						<Layers class="w-5 h-5 text-blue-600 dark:text-blue-400" />
					</div>
					<div>
						<Dialog.Title class="text-lg font-semibold text-gray-900 dark:text-white">
							Create Module
						</Dialog.Title>
						<Dialog.Description class="text-sm text-gray-500 dark:text-gray-400">
							Add a new module to your workspace
						</Dialog.Description>
					</div>
				</div>
				<Dialog.Close
					class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-white transition-colors"
					onclick={handleClose}
				>
					<X class="w-5 h-5" />
				</Dialog.Close>
			</div>

			{#if !hasValidWorkspace}
				<div class="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg text-sm text-amber-800 dark:text-amber-200">
					Select a workspace before creating a module.
				</div>
			{/if}

			{#if errorMessage}
				<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-300">
					{errorMessage}
				</div>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
				<div>
					<label for="create-module-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
						Module Name <span class="text-red-500">*</span>
					</label>
					<input
						id="create-module-name"
						type="text"
						bind:value={moduleName}
						placeholder="e.g. Client Tracker, Invoice Manager"
						class="w-full px-3.5 py-2.5 bg-gray-50 dark:bg-gray-700 border rounded-xl text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent {formErrors.name ? 'border-red-500' : 'border-gray-200 dark:border-gray-600'}"
						required
						disabled={isSubmitting}
					/>
					{#if formErrors.name}
						<p class="text-xs text-red-500 mt-1">{formErrors.name}</p>
					{/if}
				</div>

				<div>
					<label for="create-module-description" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
						Description <span class="text-red-500">*</span>
					</label>
					<textarea
						id="create-module-description"
						bind:value={description}
						placeholder="What does this module do?"
						rows="3"
						class="w-full px-3.5 py-2.5 bg-gray-50 dark:bg-gray-700 border rounded-xl text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none {formErrors.description ? 'border-red-500' : 'border-gray-200 dark:border-gray-600'}"
						required
						disabled={isSubmitting}
					></textarea>
					{#if formErrors.description}
						<p class="text-xs text-red-500 mt-1">{formErrors.description}</p>
					{/if}
				</div>

				<div class="flex justify-end gap-3 pt-2">
					<button
						type="button"
						onclick={handleClose}
						class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-xl transition-colors"
						disabled={isSubmitting}
					>
						Cancel
					</button>
					<button
						type="submit"
						class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-xl transition-colors flex items-center gap-2 disabled:opacity-50"
						disabled={isSubmitting || !moduleName.trim() || !description.trim() || !hasValidWorkspace}
					>
						{#if isSubmitting}
							<Loader2 class="w-4 h-4 animate-spin" />
							Creating...
						{:else}
							Create Module
						{/if}
					</button>
				</div>
			</form>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
