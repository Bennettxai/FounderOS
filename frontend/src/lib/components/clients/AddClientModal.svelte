<script lang="ts">
	import type { CreateClientData, ClientType, ClientStatus } from '$lib/api';
	import { fade, scale } from 'svelte/transition';

	interface Props {
		open: boolean;
		onCreate: (data: CreateClientData) => void;
	}

	let { open = $bindable(), onCreate }: Props = $props();

	// Form state
	let name = $state('');
	let type = $state<ClientType>('company');
	let email = $state('');
	let phone = $state('');
	let website = $state('');
	let industry = $state('');
	let companySize = $state('');
	let status = $state<ClientStatus>('lead');
	let source = $state('');
	let notes = $state('');
	let tagsInput = $state('');
	let submitting = $state(false);
	let errorMessage = $state<string | null>(null);

	// Address fields
	let address = $state('');
	let city = $state('');
	let stateRegion = $state('');
	let zipCode = $state('');
	let country = $state('');

	// Toggle for showing more fields
	let showMoreFields = $state(false);

	const companySizes = [
		{ value: '', label: 'Select size' },
		{ value: '1-10', label: '1-10 employees' },
		{ value: '11-50', label: '11-50 employees' },
		{ value: '51-200', label: '51-200 employees' },
		{ value: '201-500', label: '201-500 employees' },
		{ value: '501-1000', label: '501-1000 employees' },
		{ value: '1000+', label: '1000+ employees' }
	];

	const sources = [
		{ value: '', label: 'Select source' },
		{ value: 'website', label: 'Website' },
		{ value: 'referral', label: 'Referral' },
		{ value: 'cold-call', label: 'Cold Call' },
		{ value: 'linkedin', label: 'LinkedIn' },
		{ value: 'conference', label: 'Conference' },
		{ value: 'advertising', label: 'Advertising' },
		{ value: 'other', label: 'Other' }
	];

	function resetForm() {
		name = '';
		type = 'company';
		email = '';
		phone = '';
		website = '';
		industry = '';
		companySize = '';
		status = 'lead';
		source = '';
		notes = '';
		tagsInput = '';
		address = '';
		city = '';
		stateRegion = '';
		zipCode = '';
		country = '';
		showMoreFields = false;
		errorMessage = null;
	}

	function handleClose() {
		open = false;
		resetForm();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim()) return;

		submitting = true;
		errorMessage = null;

		try {
			const tags = tagsInput
				.split(',')
				.map((t) => t.trim())
				.filter((t) => t);

			const data: CreateClientData = {
				name: name.trim(),
				type,
				email: email.trim() || undefined,
				phone: phone.trim() || undefined,
				website: website.trim() || undefined,
				industry: industry.trim() || undefined,
				company_size: companySize || undefined,
				status,
				source: source || undefined,
				notes: notes.trim() || undefined,
				tags: tags.length > 0 ? tags : undefined,
				address: address.trim() || undefined,
				city: city.trim() || undefined,
				state: stateRegion.trim() || undefined,
				zip_code: zipCode.trim() || undefined,
				country: country.trim() || undefined
			};

			await onCreate(data);
			handleClose();
		} catch (err) {
			errorMessage = err instanceof Error ? err.message : 'Failed to create client. Please try again.';
		} finally {
			submitting = false;
		}
	}
</script>

{#if open}
	<div
		class="bos-modal-overlay"
		onclick={handleClose}
		role="dialog"
		aria-modal="true"
		aria-label="Add new client"
		transition:fade={{ duration: 150 }}
	>
		<div
			class="bos-modal bos-modal--md"
			onclick={(e) => e.stopPropagation()}
			role="document"
			transition:scale={{ duration: 200, start: 0.95 }}
		>
			<!-- Header -->
			<div class="bos-modal-header">
				<div>
					<h2 class="bos-modal-title">Add New Client</h2>
					<p class="bos-modal-subtitle">Create a new client record</p>
				</div>
				<button
					class="bos-modal-close"
					onclick={handleClose}
					aria-label="Close modal"
				>
					<svg width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit} class="bos-modal-body">
				{#if errorMessage}
					<div class="bos-error-banner">
						<p>{errorMessage}</p>
					</div>
				{/if}

				<div style="display:flex;flex-direction:column;gap:16px;">
					<!-- Type Toggle -->
					<div class="type-toggle">
						<button
							type="button"
							onclick={() => (type = 'company')}
							class="type-toggle__btn {type === 'company' ? 'type-toggle__btn--active' : ''}"
							aria-pressed={type === 'company'}
						>
							Company
						</button>
						<button
							type="button"
							onclick={() => (type = 'individual')}
							class="type-toggle__btn {type === 'individual' ? 'type-toggle__btn--active' : ''}"
							aria-pressed={type === 'individual'}
						>
							Individual
						</button>
					</div>

					<!-- Name -->
					<div style="display:flex;flex-direction:column;">
						<label for="client-name" class="bos-label">
							{type === 'company' ? 'Company Name' : 'Full Name'} *
						</label>
						<input
							id="client-name"
							type="text"
							bind:value={name}
							required
							disabled={submitting}
							class="bos-input"
							placeholder={type === 'company' ? 'Acme Inc.' : 'John Doe'}
						/>
					</div>

					<!-- Email & Phone -->
					<div class="bos-form-row">
						<div style="display:flex;flex-direction:column;">
							<label for="client-email" class="bos-label">Email</label>
							<input
								id="client-email"
								type="email"
								bind:value={email}
								disabled={submitting}
								class="bos-input"
								placeholder="email@example.com"
							/>
						</div>
						<div style="display:flex;flex-direction:column;">
							<label for="client-phone" class="bos-label">Phone</label>
							<input
								id="client-phone"
								type="tel"
								bind:value={phone}
								disabled={submitting}
								class="bos-input"
								placeholder="+1 (555) 123-4567"
							/>
						</div>
					</div>

					<!-- Status & Source -->
					<div class="bos-form-row">
						<div style="display:flex;flex-direction:column;">
							<label for="client-status" class="bos-label">Status</label>
							<select
								id="client-status"
								bind:value={status}
								disabled={submitting}
								class="bos-select"
							>
								<option value="lead">Lead</option>
								<option value="prospect">Prospect</option>
								<option value="active">Active</option>
								<option value="inactive">Inactive</option>
							</select>
						</div>
						<div style="display:flex;flex-direction:column;">
							<label for="client-source" class="bos-label">Source</label>
							<select
								id="client-source"
								bind:value={source}
								disabled={submitting}
								class="bos-select"
							>
								{#each sources as s}
									<option value={s.value}>{s.label}</option>
								{/each}
							</select>
						</div>
					</div>

					<!-- Company-specific fields -->
					{#if type === 'company'}
						<div class="bos-form-row">
							<div style="display:flex;flex-direction:column;">
								<label for="client-industry" class="bos-label">Industry</label>
								<input
									id="client-industry"
									type="text"
									bind:value={industry}
									disabled={submitting}
									class="bos-input"
									placeholder="Technology"
								/>
							</div>
							<div style="display:flex;flex-direction:column;">
								<label for="client-size" class="bos-label">Company Size</label>
								<select
									id="client-size"
									bind:value={companySize}
									disabled={submitting}
									class="bos-select"
								>
									{#each companySizes as size}
										<option value={size.value}>{size.label}</option>
									{/each}
								</select>
							</div>
						</div>

						<div style="display:flex;flex-direction:column;">
							<label for="client-website" class="bos-label">Website</label>
							<input
								id="client-website"
								type="url"
								bind:value={website}
								disabled={submitting}
								class="bos-input"
								placeholder="https://example.com"
							/>
						</div>
					{/if}

					<!-- Tags -->
					<div style="display:flex;flex-direction:column;">
						<label for="client-tags" class="bos-label">Tags</label>
						<input
							id="client-tags"
							type="text"
							bind:value={tagsInput}
							disabled={submitting}
							class="bos-input"
							placeholder="vip, enterprise, tech (comma separated)"
						/>
					</div>

					<!-- Show More Toggle -->
					<button
						type="button"
						onclick={() => (showMoreFields = !showMoreFields)}
						class="toggle-link"
						aria-expanded={showMoreFields}
					>
						<svg
							class="toggle-icon {showMoreFields ? 'toggle-icon--open' : ''}"
							width="14"
							height="14"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
						</svg>
						{showMoreFields ? 'Show less' : 'Show more fields'}
					</button>

					{#if showMoreFields}
						<!-- Address -->
						<div style="display:flex;flex-direction:column;">
							<label for="client-address" class="bos-label">Address</label>
							<input
								id="client-address"
								type="text"
								bind:value={address}
								disabled={submitting}
								class="bos-input"
								placeholder="123 Main St"
							/>
						</div>

						<div class="bos-form-row">
							<div style="display:flex;flex-direction:column;">
								<label for="client-city" class="bos-label">City</label>
								<input
									id="client-city"
									type="text"
									bind:value={city}
									disabled={submitting}
									class="bos-input"
									placeholder="San Francisco"
								/>
							</div>
							<div style="display:flex;flex-direction:column;">
								<label for="client-state" class="bos-label">State/Region</label>
								<input
									id="client-state"
									type="text"
									bind:value={stateRegion}
									disabled={submitting}
									class="bos-input"
									placeholder="CA"
								/>
							</div>
						</div>

						<div class="bos-form-row">
							<div style="display:flex;flex-direction:column;">
								<label for="client-zip" class="bos-label">ZIP/Postal Code</label>
								<input
									id="client-zip"
									type="text"
									bind:value={zipCode}
									disabled={submitting}
									class="bos-input"
									placeholder="94102"
								/>
							</div>
							<div style="display:flex;flex-direction:column;">
								<label for="client-country" class="bos-label">Country</label>
								<input
									id="client-country"
									type="text"
									bind:value={country}
									disabled={submitting}
									class="bos-input"
									placeholder="United States"
								/>
							</div>
						</div>

						<!-- Notes -->
						<div style="display:flex;flex-direction:column;">
							<label for="client-notes" class="bos-label">Notes</label>
							<textarea
								id="client-notes"
								bind:value={notes}
								rows="3"
								disabled={submitting}
								class="bos-textarea"
								placeholder="Additional notes about this client..."
							></textarea>
						</div>
					{/if}
				</div>

				<!-- Footer -->
				<div class="bos-modal-footer">
					<button
						type="button"
						onclick={handleClose}
						disabled={submitting}
						class="btn-rounded btn-rounded-ghost"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={submitting || !name.trim()}
						class="btn-pill btn-pill-primary btn-pill-sm"
					>
						{submitting ? 'Creating...' : 'Create Client'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	/* Type toggle */
	.type-toggle {
		display: flex;
		gap: 0.5rem;
	}
	.type-toggle__btn {
		flex: 1;
		padding: 0.5rem 1rem;
		font-size: var(--text-sm, 0.875rem);
		font-weight: var(--font-medium, 500);
		border-radius: var(--radius-sm, 8px);
		border: 1px solid var(--bos-border-color);
		cursor: pointer;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
		background: var(--bos-v2-layer-background-secondary);
		color: var(--bos-text-secondary-color);
	}
	.type-toggle__btn--active {
		background: var(--bos-text-primary-color);
		color: var(--bos-surface-on-color);
		border-color: var(--bos-text-primary-color);
	}

	/* Show more toggle link */
	.toggle-link {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		font-size: var(--text-sm, 0.875rem);
		color: var(--bos-text-secondary-color);
		background: none;
		border: none;
		cursor: pointer;
		padding: 0;
		transition: color 0.15s;
	}
	.toggle-link:hover {
		color: var(--bos-text-primary-color);
	}
	.toggle-icon {
		transition: transform 0.2s;
	}
	.toggle-icon--open {
		transform: rotate(180deg);
	}
</style>
