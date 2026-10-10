<script lang="ts">
	import type { CalendarEvent, GoogleConnectionStatus } from '$lib/api';
	import type { EventFormData } from './calendarUtils';

	interface Props {
		editingEvent: CalendarEvent | null;
		formData: EventFormData;
		isSaving: boolean;
		formError: string;
		connectionStatus: GoogleConnectionStatus | null;
		onSave: () => void;
		onClose: () => void;
	}

	let {
		editingEvent,
		formData = $bindable(),
		isSaving,
		formError,
		connectionStatus,
		onSave,
		onClose
	}: Props = $props();
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
	role="dialog"
	aria-modal="true"
	aria-label="{editingEvent ? 'Edit event' : 'Create event'}"
	onclick={onClose}
>
	<div class="rounded-xl shadow-xl max-w-lg w-full mx-4 max-h-[90vh] overflow-auto" style="background: var(--dbg); border: 1px solid var(--dbd);" onclick={(e) => e.stopPropagation()}>
		<div class="p-6">
			<!-- Header -->
			<div class="flex items-center justify-between mb-6">
				<h2 class="text-xl font-semibold" style="color: var(--dt);">
					{editingEvent ? 'Edit Event' : 'Create Event'}
				</h2>
				<button onclick={onClose} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Close">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			{#if formError}
				<div class="mb-4 p-3 rounded-lg text-sm" style="background: color-mix(in srgb, var(--bos-status-error-text) 10%, var(--dbg)); border: 1px solid color-mix(in srgb, var(--bos-status-error-text) 20%, var(--dbg)); color: var(--bos-status-error-text);">
					{formError}
				</div>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); onSave(); }} class="space-y-4">
				<!-- Title -->
				<div>
					<label for="event-title" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Title</label>
					<input
						id="event-title"
						type="text"
						bind:value={formData.title}
						placeholder="Event title"
						class="input w-full"
						required
					/>
				</div>

				<!-- Description -->
				<div>
					<label for="event-description" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Description</label>
					<textarea
						id="event-description"
						bind:value={formData.description}
						placeholder="Add description..."
						rows="3"
						class="input w-full resize-none"
					></textarea>
				</div>

				<!-- All Day Toggle -->
				<div class="flex items-center gap-2">
					<input
						id="event-allday"
						type="checkbox"
						bind:checked={formData.all_day}
						class="w-4 h-4 rounded focus:ring-gray-900"
						style="color: var(--dt); border-color: var(--dbd);"
					/>
					<label for="event-allday" class="text-sm font-medium" style="color: var(--dt2);">All day event</label>
				</div>

				<!-- Start Date/Time -->
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="event-start-date" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Start Date</label>
						<input
							id="event-start-date"
							type="date"
							bind:value={formData.start_date}
							class="input w-full"
							required
						/>
					</div>
					{#if !formData.all_day}
						<div>
							<label for="event-start-time" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Start Time</label>
							<input
								id="event-start-time"
								type="time"
								bind:value={formData.start_time}
								class="input w-full"
							/>
						</div>
					{/if}
				</div>

				<!-- End Date/Time -->
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="event-end-date" class="block text-sm font-medium mb-1" style="color: var(--dt2);">End Date</label>
						<input
							id="event-end-date"
							type="date"
							bind:value={formData.end_date}
							class="input w-full"
							required
						/>
					</div>
					{#if !formData.all_day}
						<div>
							<label for="event-end-time" class="block text-sm font-medium mb-1" style="color: var(--dt2);">End Time</label>
							<input
								id="event-end-time"
								type="time"
								bind:value={formData.end_time}
								class="input w-full"
							/>
						</div>
					{/if}
				</div>

				<!-- Location -->
				<div>
					<label for="event-location" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Location</label>
					<input
						id="event-location"
						type="text"
						bind:value={formData.location}
						placeholder="Add location..."
						class="input w-full"
					/>
				</div>

				<!-- Meeting Type -->
				<div>
					<label for="event-type" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Meeting Type</label>
					<select id="event-type" bind:value={formData.meeting_type} class="input w-full">
						<option value="">No type</option>
						<option value="team">Team</option>
						<option value="sales">Sales</option>
						<option value="client">Client</option>
						<option value="onboarding">Onboarding</option>
						<option value="kickoff">Kickoff</option>
						<option value="implementation">Implementation</option>
						<option value="standup">Standup</option>
						<option value="planning">Planning</option>
						<option value="review">Review</option>
						<option value="one_on_one">1:1</option>
						<option value="retrospective">Retrospective</option>
						<option value="internal">Internal</option>
						<option value="external">External</option>
						<option value="other">Other</option>
					</select>
				</div>

				<!-- Meeting Link -->
				<div>
					<label for="event-link" class="block text-sm font-medium mb-1" style="color: var(--dt2);">Meeting Link</label>
					<input
						id="event-link"
						type="url"
						bind:value={formData.meeting_link}
						placeholder="https://..."
						class="input w-full"
					/>
				</div>

				<!-- Google Sync Info -->
				{#if connectionStatus?.connected}
					<div class="flex items-center gap-2 text-sm" style="color: var(--dt3);">
						<svg class="w-4 h-4" viewBox="0 0 24 24">
							<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
							<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
							<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
							<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
						</svg>
						Will sync to Google Calendar
					</div>
				{/if}

				<!-- Buttons -->
				<div class="flex items-center justify-end gap-2 pt-4">
					<button type="button" onclick={onClose} class="btn btn-secondary">Cancel</button>
					<button type="submit" disabled={isSaving} class="btn btn-primary disabled:opacity-50">
						{#if isSaving}
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Saving...
						{:else}
							{editingEvent ? 'Update Event' : 'Create Event'}
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
</div>
