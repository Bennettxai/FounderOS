<script lang="ts">
	import { apiClient } from '$lib/api';
	import { getActiveWorkspaceHeaders, getApiBaseUrl } from '$lib/api/base';
	import type { CalendarEvent, ActionItem } from '$lib/api';
	import { getEventDisplayStart, sanitizeHtml } from './calendarUtils';

	interface Props {
		event: CalendarEvent;
		onClose: () => void;
		onEdit: (event: CalendarEvent) => void;
		onDelete: (event: CalendarEvent) => void;
	}

	let { event, onClose, onEdit, onDelete }: Props = $props();

	// AI / notes section state
	let meetingNotes = $state(event.meeting_notes || '');
	let meetingSummary = $state(event.meeting_summary || '');
	let actionItems = $state<ActionItem[]>(event.action_items || []);
	let isGeneratingSummary = $state(false);
	let isExtractingActions = $state(false);
	let showNotesSection = $state(
		!!(event.meeting_notes || event.meeting_summary || event.action_items?.length)
	);

	async function generateSummary() {
		if (!meetingNotes.trim()) {
			alert('Please add meeting notes first');
			return;
		}
		isGeneratingSummary = true;
		try {
			const response = await fetch(`${getApiBaseUrl()}/chat/message`, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json', ...getActiveWorkspaceHeaders() },
				body: JSON.stringify({
					message: `Please summarize the following meeting notes in 2-3 concise bullet points. Focus on key decisions and outcomes.\n\nMeeting: ${event.title || 'Meeting'}\n\nNotes:\n${meetingNotes}`,
					model: 'llama3.2:latest',
					stream: false
				})
			});
			if (response.ok) {
				const reader = response.body?.getReader();
				const decoder = new TextDecoder();
				let fullContent = '';
				if (reader) {
					while (true) {
						const { done, value } = await reader.read();
						if (done) break;
						fullContent += decoder.decode(value, { stream: true });
					}
				}
				meetingSummary = fullContent.trim();
				await apiClient.put(`/calendar/events/${event.id}`, { meeting_summary: meetingSummary });
			}
		} catch (error) {
			console.error('Error generating summary:', error);
			alert('Failed to generate summary. Please try again.');
		} finally {
			isGeneratingSummary = false;
		}
	}

	async function extractActionItems() {
		if (!meetingNotes.trim()) {
			alert('Please add meeting notes first');
			return;
		}
		isExtractingActions = true;
		try {
			const response = await apiClient.post('/chat/ai/extract-tasks', {
				content: meetingNotes
			});
			if (response.ok) {
				const data = await response.json();
				if (data.tasks && Array.isArray(data.tasks)) {
					actionItems = data.tasks.map(
						(t: { title: string; description?: string }, index: number) => ({
							id: `action-${index}-${Date.now()}`,
							text: t.title,
							completed: false
						})
					);
					await apiClient.put(`/calendar/events/${event.id}`, { action_items: actionItems });
				}
			}
		} catch (error) {
			console.error('Error extracting action items:', error);
			alert('Failed to extract action items. Please try again.');
		} finally {
			isExtractingActions = false;
		}
	}

	async function saveNotes() {
		try {
			await apiClient.put(`/calendar/events/${event.id}`, { notes: meetingNotes });
		} catch (error) {
			console.error('Error saving notes:', error);
		}
	}

	async function createTaskFromActionItem(item: ActionItem) {
		try {
			const response = await apiClient.post('/tasks', {
				title: item.text,
				description: `From meeting: ${event.title || 'Meeting'}`,
				status: 'todo',
				priority: 'medium',
				assignee_id: item.assignee_id,
				due_date: item.due_date
			});
			if (response.ok) {
				alert('Task created successfully!');
			}
		} catch (error) {
			console.error('Error creating task:', error);
			alert('Failed to create task');
		}
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
	role="dialog"
	aria-modal="true"
	aria-label="Event details"
	onclick={onClose}
>
	<div class="rounded-xl shadow-xl max-w-lg w-full mx-4 max-h-[80vh] overflow-auto" style="background: var(--dbg); border: 1px solid var(--dbd);" onclick={(e) => e.stopPropagation()}>
		<div class="p-6">
			<!-- Header -->
			<div class="flex items-start justify-between mb-4">
				<div>
					<h2 class="text-xl font-semibold" style="color: var(--dt);">{event.title || 'Untitled Event'}</h2>
					{#if event.meeting_type && event.meeting_type !== 'other'}
						<span class="inline-block mt-1 px-2 py-0.5 text-xs font-medium rounded-full" style="background: var(--dbg2); color: var(--dt2);">
							{event.meeting_type.replace('_', ' ')}
						</span>
					{/if}
				</div>
				<button onclick={onClose} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Close modal">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="space-y-4">
				<!-- Time -->
				<div class="flex items-start gap-3">
					<svg class="w-5 h-5 mt-0.5" style="color: var(--dt3);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<div>
						<p style="color: var(--dt);">
							{getEventDisplayStart(event).toLocaleDateString('en-US', {
								weekday: 'long',
								month: 'long',
								day: 'numeric',
								year: 'numeric'
							})}
						</p>
						<p class="text-sm" style="color: var(--dt3);">
							{#if event.all_day}
								All day
							{:else}
								{new Date(event.start_time).toLocaleTimeString('en-US', {
									hour: 'numeric',
									minute: '2-digit'
								})} - {new Date(event.end_time).toLocaleTimeString('en-US', {
									hour: 'numeric',
									minute: '2-digit'
								})}
							{/if}
						</p>
					</div>
				</div>

				<!-- Location -->
				{#if event.location}
					<div class="flex items-start gap-3">
					<svg class="w-5 h-5 mt-0.5" style="color: var(--dt3);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
					</svg>
					<p style="color: var(--dt);">{event.location}</p>
					</div>
				{/if}

				<!-- Meeting Link -->
				{#if event.meeting_link}
					<div class="flex items-start gap-3">
					<svg class="w-5 h-5 mt-0.5" style="color: var(--dt3);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
						<a href={event.meeting_link} target="_blank" rel="noopener noreferrer" class="hover:underline" style="color: var(--bos-nav-active);">
							Join Meeting
						</a>
					</div>
				{/if}

				<!-- Description -->
				{#if event.description}
					<div class="flex items-start gap-3">
					<svg class="w-5 h-5 mt-0.5 flex-shrink-0" style="color: var(--dt3);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7" />
					</svg>
					<div class="text-sm prose prose-sm max-w-none prose-p:my-1 prose-ul:my-1 prose-li:my-0.5" style="color: var(--dt2);">
							{@html sanitizeHtml(event.description)}
						</div>
					</div>
				{/if}

				<!-- Attendees -->
				{#if event.attendees && event.attendees.length > 0}
					<div class="flex items-start gap-3">
					<svg class="w-5 h-5 mt-0.5" style="color: var(--dt3);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
					</svg>
					<div>
						<p class="text-sm font-medium mb-1" style="color: var(--dt2);">
								{event.attendees.length} Attendees
							</p>
							<div class="space-y-1">
								{#each event.attendees as attendee}
									<p class="text-sm" style="color: var(--dt2);">{attendee.name || attendee.email}</p>
								{/each}
							</div>
						</div>
					</div>
				{/if}

				<!-- Google Calendar Link -->
				{#if event.html_link}
				<div class="pt-4" style="border-top: 1px solid var(--dbd2);">
					<a
						href={event.html_link}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-flex items-center gap-2 text-sm" style="color: var(--dt3);"
						>
							<svg class="w-4 h-4" viewBox="0 0 24 24">
								<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
								<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
								<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
								<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
							</svg>
							Open in Google Calendar
						</a>
					</div>
				{/if}

				<!-- Meeting Notes & AI -->
				<div class="pt-4" style="border-top: 1px solid var(--dbd2);">
					<button
						onclick={() => (showNotesSection = !showNotesSection)}
						class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-2 mb-3"
					>
						<svg
							class="w-4 h-4 transition-transform {showNotesSection ? 'rotate-90' : ''}"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
						<svg class="w-4 h-4" style="color: var(--bos-v2-icon-secondary);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
						</svg>
						Meeting Notes & AI
					</button>

					{#if showNotesSection}
						<div class="space-y-4">
							<!-- Notes textarea -->
							<div>
								<label class="block text-sm font-medium mb-1" style="color: var(--dt2);" for="meeting-notes">
									Meeting Notes / Transcription
								</label>
								<textarea
									id="meeting-notes"
									bind:value={meetingNotes}
									onblur={saveNotes}
									placeholder="Add meeting notes, transcription, or paste voice recording text here..."
									rows="4"
									class="w-full px-3 py-2 rounded-lg text-sm resize-none focus:outline-none"
									style="border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt);"
								></textarea>
							</div>

							<!-- AI Action Buttons -->
							<div class="flex items-center gap-2">
								<button
									onclick={generateSummary}
									disabled={isGeneratingSummary || !meetingNotes.trim()}
									class="btn-pill btn-pill-soft btn-pill-sm flex items-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									{#if isGeneratingSummary}
										<svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
											<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
										</svg>
										Summarizing...
									{:else}
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
										</svg>
										Summarize
									{/if}
								</button>

								<button
									onclick={extractActionItems}
									disabled={isExtractingActions || !meetingNotes.trim()}
									class="btn-pill btn-pill-soft btn-pill-sm flex items-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									{#if isExtractingActions}
										<svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
											<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
										</svg>
										Extracting...
									{:else}
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
										</svg>
										Extract Actions
									{/if}
								</button>
							</div>

							<!-- Summary -->
							{#if meetingSummary}
								<div class="p-3 rounded-lg" style="background: var(--dbg2);">
									<div class="flex items-center gap-2 text-sm font-medium mb-2" style="color: var(--dt2);">
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
										</svg>
										Summary
									</div>
									<p class="text-sm whitespace-pre-wrap" style="color: var(--dt2);">{meetingSummary}</p>
								</div>
							{/if}

							<!-- Action Items -->
							{#if actionItems.length > 0}
								<div class="p-3 rounded-lg" style="background: var(--bos-status-info-bg);">
									<div class="flex items-center gap-2 text-sm font-medium mb-2" style="color: var(--bos-status-info-text);">
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
										</svg>
										Action Items ({actionItems.length})
									</div>
									<ul class="space-y-2">
										{#each actionItems as item, index (item.id)}
											<li class="flex items-start gap-2 text-sm" style="color: var(--dt2);">
												<span class="flex-shrink-0 w-5 h-5 flex items-center justify-center rounded-full text-xs font-medium" style="background: var(--bos-status-info-bg); color: var(--bos-nav-active);">
													{index + 1}
												</span>
												<span class="flex-1">{item.text}</span>
												<button
													onclick={() => createTaskFromActionItem(item)}
													class="btn-pill btn-pill-ghost btn-pill-xs flex-shrink-0"
													aria-label="Create task from action item"
												>
													+ Task
												</button>
											</li>
										{/each}
									</ul>
								</div>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Action Buttons -->
				<div class="pt-4 flex items-center justify-end gap-2" style="border-top: 1px solid var(--dbd2);">
					<button
						onclick={() => onEdit(event)}
						class="btn btn-secondary text-sm"
					>
						<svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
						</svg>
						Edit
					</button>
					<button
						onclick={() => onDelete(event)}
						class="btn-pill btn-pill-danger btn-pill-sm"
					>
						<svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
						Delete
					</button>
				</div>
			</div>
		</div>
	</div>
</div>
