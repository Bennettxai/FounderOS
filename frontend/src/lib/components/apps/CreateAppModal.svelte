<script lang="ts">
	import type { AppTemplate } from '$lib/types/apps';
	import { Users, CheckSquare, Wallet, Kanban, Sparkles, Loader2 } from 'lucide-svelte';

	interface Props {
		open: boolean;
		template: AppTemplate | null;
		onClose: () => void;
		onCreate: (data: { templateId: string; name: string; description: string }) => Promise<{ appId: string } | null>;
	}

	let { open = $bindable(), template, onClose, onCreate }: Props = $props();

	// Form state
	let name = $state('');
	let description = $state('');
	let submitting = $state(false);
	let error = $state('');

	// Initialize form when template changes
	$effect(() => {
		if (template && open) {
			name = template.name;
			description = template.description;
			error = '';
		}
	});

	function resetForm() {
		name = '';
		description = '';
		error = '';
		submitting = false;
	}

	function handleClose() {
		if (submitting) return; // Prevent closing while submitting
		open = false;
		resetForm();
		onClose();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim() || !template) return;

		submitting = true;
		error = '';

		try {
			const result = await onCreate({
				templateId: template.id,
				name: name.trim(),
				description: description.trim()
			});

			if (result?.appId) {
				handleClose();
			}
		} catch (err) {
			console.error('Failed to create app:', err);
			error = err instanceof Error ? err.message : 'Failed to create app. Please try again.';
		} finally {
			submitting = false;
		}
	}

	// Get icon component based on template
	function getIconComponent(iconName: string) {
		switch (iconName) {
			case 'Users':
				return Users;
			case 'CheckSquare':
				return CheckSquare;
			case 'Receipt':
			case 'Wallet':
				return Wallet;
			case 'Kanban':
			default:
				return Kanban;
		}
	}

	// Get gradient based on template category
	function getGradient(category: string): string {
		switch (category) {
			case 'business':
				return 'from-violet-500 to-purple-600';
			case 'productivity':
				return 'from-blue-500 to-cyan-500';
			case 'finance':
				return 'from-green-500 to-emerald-600';
			default:
				return 'from-gray-500 to-gray-600';
		}
	}
</script>

{#if open && template}
	{@const TemplateIcon = getIconComponent(template.icon)}
	<div class="fixed inset-0 z-50 flex items-center justify-center">
		<!-- Backdrop -->
		<div
			class="absolute inset-0 bg-black/50 backdrop-blur-sm"
			onclick={handleClose}
			onkeydown={(e) => e.key === 'Escape' && handleClose()}
			role="button"
			tabindex="-1"
			aria-label="Close modal"
		></div>

		<!-- Modal -->
		<div class="relative bg-white dark:bg-gray-900 rounded-2xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
			<!-- Header with Template Preview -->
			<div class="px-6 pt-6 pb-4">
				<div class="flex items-start gap-4">
					<!-- Template Icon -->
					<div class="w-14 h-14 rounded-xl bg-gradient-to-br {getGradient(template.category)} flex items-center justify-center text-white shadow-lg">
						<TemplateIcon class="w-7 h-7" strokeWidth={1.75} />
					</div>
					<div class="flex-1 min-w-0">
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
							Create from Template
						</h2>
						<p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
							{template.name} template
						</p>
					</div>
					<!-- Close button -->
					<button
						type="button"
						onclick={handleClose}
						disabled={submitting}
						aria-label="Close create app modal"
						class="btn-pill btn-pill-ghost btn-pill-icon"
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit}>
				<div class="px-6 pb-4 space-y-4">
					<!-- Error message -->
					{#if error}
						<div class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg" role="alert" aria-live="polite">
							<div class="flex items-start gap-2">
								<svg class="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
								<div>
									<p class="text-sm font-medium text-red-800 dark:text-red-300">Unable to create app</p>
									<p class="text-sm text-red-600 dark:text-red-400 mt-0.5">{error}</p>
								</div>
							</div>
						</div>
					{/if}

					<!-- App Name -->
					<div>
						<label for="create-app-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
							App Name *
						</label>
						<input
							id="create-app-name"
							type="text"
							bind:value={name}
							required
							disabled={submitting}
							class="w-full px-3.5 py-2.5 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl
								focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
								disabled:opacity-50 disabled:cursor-not-allowed
								text-gray-900 dark:text-white placeholder-gray-400"
							placeholder="My {template.name} App"
						/>
					</div>

					<!-- Description -->
					<div>
						<label for="create-app-description" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
							Description
						</label>
						<textarea
							id="create-app-description"
							bind:value={description}
							rows="3"
							disabled={submitting}
							class="w-full px-3.5 py-2.5 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl
								focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
								disabled:opacity-50 disabled:cursor-not-allowed resize-none
								text-gray-900 dark:text-white placeholder-gray-400"
							placeholder="Describe what you want this app to do..."
						></textarea>
						<p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
							The more detail you provide, the better the AI can customize your app.
						</p>
					</div>

					<!-- AI Generation Notice -->
					<div class="flex items-start gap-3 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-100 dark:border-blue-800 rounded-xl">
						<Sparkles class="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" strokeWidth={1.75} />
						<div>
							<p class="text-sm font-medium text-blue-900 dark:text-blue-100">
								AI-Powered Generation
							</p>
							<p class="text-xs text-blue-700 dark:text-blue-300 mt-0.5">
								Your app will be generated using AI based on the template and your description.
							</p>
						</div>
					</div>
				</div>

				<!-- Footer -->
				<div class="px-6 py-4 bg-gray-50 dark:bg-gray-800/50 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
					<button
						type="button"
						onclick={handleClose}
						disabled={submitting}
						class="btn-pill btn-pill-ghost btn-pill-sm"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={submitting || !name.trim()}
						class="btn-pill btn-pill-primary btn-pill-sm inline-flex items-center gap-2"
					>
						{#if submitting}
							<Loader2 class="w-4 h-4 animate-spin" />
							<span>Creating...</span>
						{:else}
							<Sparkles class="w-4 h-4" />
							<span>Create App</span>
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
