<script lang="ts">
	import { proposeSchedule, createEventFromProposal } from '$lib/api/calendar';
	import type { ScheduleRequest, ScheduleProposal, ProposedSlot, MeetingType, TimePreferences } from '$lib/api/calendar';
	import { fly, fade, scale } from 'svelte/transition';

	interface Props {
		isOpen: boolean;
		onClose: () => void;
		onEventCreated?: (eventId: string) => void;
		defaultAttendees?: string[];
		defaultMeetingType?: MeetingType;
		projectId?: string;
		clientId?: string;
	}

	let { 
		isOpen, 
		onClose, 
		onEventCreated,
		defaultAttendees = [],
		defaultMeetingType = 'other',
		projectId,
		clientId
	}: Props = $props();

	// Form state
	let title = $state('');
	let description = $state('');
	let durationMinutes = $state(30);
	let attendees = $state<string[]>([]);
	let emailInput = $state('');
	let emailError = $state<string | null>(null);
	let meetingType = $state<MeetingType>('other');
	
	// Search window
	let searchStart = $state(getDefaultSearchStart());
	let searchEnd = $state(getDefaultSearchEnd());
	
	// Preferences
	let preferredStartHour = $state(9);
	let preferredEndHour = $state(17);
	let avoidBackToBack = $state(true);
	let bufferMinutes = $state(15);
	let includeWeekends = $state(false);
	
	// API state
	let loading = $state(false);
	let error = $state<string | null>(null);
	let proposal = $state<ScheduleProposal | null>(null);
	let selectedSlot = $state<ProposedSlot | null>(null);
	let creating = $state(false);
	let step = $state<1 | 2>(1);

	// Timezone
	const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
	const timezoneAbbr = new Date().toLocaleTimeString('en-US', { timeZoneName: 'short' }).split(' ').pop();

	// Initialize from props when modal opens
	$effect(() => {
		if (isOpen) {
			attendees = [...defaultAttendees];
			meetingType = defaultMeetingType;
			step = 1;
		}
	});

	// Email validation regex
	const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

	function getDefaultSearchStart(): string {
		const now = new Date();
		now.setHours(0, 0, 0, 0);
		return now.toISOString().slice(0, 16);
	}

	function getDefaultSearchEnd(): string {
		const end = new Date();
		end.setDate(end.getDate() + 5);
		end.setHours(23, 59, 0, 0);
		return end.toISOString().slice(0, 16);
	}

	function validateEmail(email: string): boolean {
		return EMAIL_REGEX.test(email.trim().toLowerCase());
	}

	function addAttendee(email: string) {
		const normalized = email.trim().toLowerCase();
		if (!normalized) return;
		
		if (!validateEmail(normalized)) {
			emailError = 'Please enter a valid email address';
			return;
		}
		
		if (attendees.includes(normalized)) {
			emailError = 'This attendee is already added';
			return;
		}
		
		attendees = [...attendees, normalized];
		emailInput = '';
		emailError = null;
	}

	function removeAttendee(email: string) {
		attendees = attendees.filter(a => a !== email);
	}

	function handleEmailKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			addAttendee(emailInput);
		} else if (e.key === 'Backspace' && !emailInput && attendees.length > 0) {
			// Remove last attendee when backspace on empty input
			attendees = attendees.slice(0, -1);
		}
	}

	function handleEmailPaste(e: ClipboardEvent) {
		e.preventDefault();
		const pastedText = e.clipboardData?.getData('text') || '';
		const emails = pastedText.split(/[,;\s\n]+/).filter(Boolean);
		
		for (const email of emails) {
			if (validateEmail(email)) {
				const normalized = email.trim().toLowerCase();
				if (!attendees.includes(normalized)) {
					attendees = [...attendees, normalized];
				}
			}
		}
	}

	function getInitials(email: string): string {
		const name = email.split('@')[0];
		const parts = name.split(/[._-]/);
		if (parts.length >= 2) {
			return (parts[0][0] + parts[1][0]).toUpperCase();
		}
		return name.slice(0, 2).toUpperCase();
	}

	function getAvatarColor(email: string): string {
		const colors = [
			'bg-blue-500', 'bg-green-500', 'bg-purple-500', 'bg-pink-500',
			'bg-indigo-500', 'bg-teal-500', 'bg-orange-500', 'bg-cyan-500'
		];
		const hash = email.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
		return colors[hash % colors.length];
	}

	async function handleFindTimes() {
		if (!title.trim()) {
			error = 'Please enter a meeting title';
			return;
		}
		
		if (attendees.length === 0) {
			error = 'Please add at least one attendee';
			return;
		}

		loading = true;
		error = null;
		proposal = null;
		selectedSlot = null;

		try {
			const preferredDays = includeWeekends ? [0, 1, 2, 3, 4, 5, 6] : [1, 2, 3, 4, 5];
			
			const preferences: TimePreferences = {
				preferred_start_hour: preferredStartHour,
				preferred_end_hour: preferredEndHour,
				avoid_back_to_back: avoidBackToBack,
				buffer_minutes: bufferMinutes,
				preferred_days: preferredDays
			};

			const request: ScheduleRequest = {
				title: title.trim(),
				description: description.trim() || undefined,
				duration_minutes: durationMinutes,
				attendees,
				search_start: new Date(searchStart).toISOString(),
				search_end: new Date(searchEnd).toISOString(),
				preferences,
				meeting_type: meetingType,
				project_id: projectId,
				client_id: clientId
			};

			proposal = await proposeSchedule(request);
			step = 2;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to find available times';
		} finally {
			loading = false;
		}
	}

	async function handleConfirmSlot() {
		if (!selectedSlot || !proposal) return;

		creating = true;
		error = null;

		try {
			const event = await createEventFromProposal(proposal.request, {
				start: selectedSlot.start,
				end: selectedSlot.end
			});
			
			onEventCreated?.(event.id);
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create event';
		} finally {
			creating = false;
		}
	}

	function handleClose() {
		// Reset all state
		title = '';
		description = '';
		durationMinutes = 30;
		attendees = [];
		emailInput = '';
		emailError = null;
		meetingType = 'other';
		searchStart = getDefaultSearchStart();
		searchEnd = getDefaultSearchEnd();
		preferredStartHour = 9;
		preferredEndHour = 17;
		avoidBackToBack = true;
		bufferMinutes = 15;
		includeWeekends = false;
		proposal = null;
		selectedSlot = null;
		error = null;
		step = 1;
		onClose();
	}

	function goBack() {
		step = 1;
		selectedSlot = null;
	}

	function formatDateTime(isoString: string): string {
		return new Date(isoString).toLocaleString('en-US', {
			weekday: 'short',
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	}

	function formatTimeOnly(isoString: string): string {
		return new Date(isoString).toLocaleTimeString('en-US', {
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	}

	function getScoreColor(score: number): string {
		if (score >= 80) return 'text-green-600 bg-green-100';
		if (score >= 60) return 'text-yellow-600 bg-yellow-100';
		if (score >= 40) return 'text-orange-600 bg-orange-100';
		return 'text-red-600 bg-red-100';
	}

	function getScoreEmoji(score: number): string {
		if (score >= 80) return '🌟';
		if (score >= 60) return '👍';
		if (score >= 40) return '🤔';
		return '⚠️';
	}

	const meetingTypeOptions: { value: MeetingType; label: string; icon: string }[] = [
		{ value: 'team', label: 'Team Meeting', icon: '👥' },
		{ value: 'one_on_one', label: '1:1 Meeting', icon: '🤝' },
		{ value: 'client', label: 'Client Meeting', icon: '💼' },
		{ value: 'sales', label: 'Sales Call', icon: '📞' },
		{ value: 'standup', label: 'Standup', icon: '🧍' },
		{ value: 'planning', label: 'Planning', icon: '📋' },
		{ value: 'review', label: 'Review', icon: '🔍' },
		{ value: 'retrospective', label: 'Retrospective', icon: '🔄' },
		{ value: 'kickoff', label: 'Kickoff', icon: '🚀' },
		{ value: 'onboarding', label: 'Onboarding', icon: '👋' },
		{ value: 'implementation', label: 'Implementation', icon: '⚙️' },
		{ value: 'internal', label: 'Internal', icon: '🏢' },
		{ value: 'external', label: 'External', icon: '🌐' },
		{ value: 'other', label: 'Other', icon: '📅' }
	];

	const durationOptions = [
		{ value: 15, label: '15 min' },
		{ value: 30, label: '30 min' },
		{ value: 45, label: '45 min' },
		{ value: 60, label: '1 hour' },
		{ value: 90, label: '1.5 hours' },
		{ value: 120, label: '2 hours' }
	];

	function formatHour(hour: number): string {
		if (hour === 0) return '12 AM';
		if (hour < 12) return `${hour} AM`;
		if (hour === 12) return '12 PM';
		return `${hour - 12} PM`;
	}
</script>

{#if isOpen}
	<!-- Backdrop -->
	<div 
		class="fixed inset-0 bg-black/50 z-40"
		onclick={handleClose}
		onkeydown={(e) => e.key === 'Escape' && handleClose()}
		role="button"
		tabindex="-1"
		aria-label="Close modal"
		transition:fade={{ duration: 150 }}
	></div>

	<!-- Modal -->
	<div 
		class="fixed inset-4 md:inset-auto md:top-1/2 md:left-1/2 md:-translate-x-1/2 md:-translate-y-1/2 
			md:w-full md:max-w-2xl md:max-h-[90vh] bg-white rounded-xl shadow-2xl z-50 flex flex-col overflow-hidden"
		transition:scale={{ duration: 200, start: 0.95 }}
	>
		<!-- Header -->
		<div class="flex items-center justify-between px-6 py-4 border-b bg-gradient-to-r from-blue-50 to-indigo-50">
			<div class="flex items-center gap-3">
				<!-- Step Indicator -->
				<div class="flex items-center gap-2">
					<div class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium
						{step === 1 ? 'bg-blue-600 text-white' : 'bg-blue-100 text-blue-600'}">
						1
					</div>
					<div class="w-8 h-0.5 {step === 2 ? 'bg-blue-600' : 'bg-gray-300'}"></div>
					<div class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium
						{step === 2 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'}">
						2
					</div>
				</div>
				<div class="ml-2">
					<h2 class="text-lg font-semibold text-gray-900">
						{step === 1 ? 'Meeting Details' : 'Select Time'}
					</h2>
					<p class="text-xs text-gray-500 flex items-center gap-1">
						<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						{timezoneAbbr} ({userTimezone})
					</p>
				</div>
			</div>
			<button 
				onclick={handleClose}
				aria-label="Close modal"
				class="btn-pill btn-pill-ghost btn-pill-icon"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<!-- Content -->
		<div class="flex-1 overflow-y-auto p-6">
			{#if error}
				<div 
					class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm flex items-start gap-2"
					transition:fly={{ y: -10, duration: 200 }}
				>
					<svg class="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span>{error}</span>
				</div>
			{/if}

			{#if step === 1}
				<!-- Step 1: Meeting Details Form -->
				<div class="space-y-5" transition:fly={{ x: -20, duration: 200 }}>
					<!-- Title -->
					<div>
						<label for="title" class="block text-sm font-medium text-gray-700 mb-1.5">
							Meeting Title <span class="text-red-500">*</span>
						</label>
						<input
							id="title"
							type="text"
							bind:value={title}
							placeholder="Sprint Planning, Client Check-in, etc."
							class="w-full px-3 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500
								transition-shadow"
						/>
					</div>

					<!-- Attendees - Chip Input -->
					<div>
						<label for="attendees" class="block text-sm font-medium text-gray-700 mb-1.5">
							Attendees <span class="text-red-500">*</span>
						</label>
						<div class="min-h-[46px] p-2 border border-gray-300 rounded-lg focus-within:ring-2 focus-within:ring-blue-500 
							focus-within:border-blue-500 transition-shadow bg-white">
							<div class="flex flex-wrap gap-2">
								{#each attendees as email (email)}
									<span 
										class="inline-flex items-center gap-1.5 pl-1 pr-2 py-1 bg-gray-100 rounded-full text-sm
											hover:bg-gray-200 transition-colors group"
										transition:scale={{ duration: 150 }}
									>
										<span class="w-6 h-6 rounded-full {getAvatarColor(email)} text-white text-xs 
											flex items-center justify-center font-medium">
											{getInitials(email)}
										</span>
										<span class="text-gray-700 max-w-[150px] truncate">{email}</span>
										<button
											onclick={() => removeAttendee(email)}
											class="btn-pill btn-pill-ghost btn-pill-icon w-4 h-4"
											aria-label="Remove {email}"
										>
											<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
											</svg>
										</button>
									</span>
								{/each}
								<input
									id="attendees"
									type="email"
									bind:value={emailInput}
									onkeydown={handleEmailKeydown}
									onpaste={handleEmailPaste}
									onblur={() => emailInput && addAttendee(emailInput)}
									placeholder={attendees.length === 0 ? "Type email and press Enter..." : "Add more..."}
									class="flex-1 min-w-[180px] px-1 py-1 outline-none text-sm bg-transparent"
								/>
							</div>
						</div>
						{#if emailError}
							<p class="mt-1 text-xs text-red-500" transition:fly={{ y: -5, duration: 150 }}>{emailError}</p>
						{:else}
							<p class="mt-1 text-xs text-gray-500">Press Enter or comma to add. Paste multiple emails at once.</p>
						{/if}
					</div>

					<!-- Duration & Type Row -->
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="duration" class="block text-sm font-medium text-gray-700 mb-1.5">
								Duration
							</label>
							<select
								id="duration"
								bind:value={durationMinutes}
								class="w-full px-3 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 
									focus:border-blue-500 bg-white"
							>
								{#each durationOptions as option}
									<option value={option.value}>{option.label}</option>
								{/each}
							</select>
						</div>
						<div>
							<label for="meetingType" class="block text-sm font-medium text-gray-700 mb-1.5">
								Meeting Type
							</label>
							<select
								id="meetingType"
								bind:value={meetingType}
								class="w-full px-3 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 
									focus:border-blue-500 bg-white"
							>
								{#each meetingTypeOptions as option}
									<option value={option.value}>{option.icon} {option.label}</option>
								{/each}
							</select>
						</div>
					</div>

					<!-- Search Window -->
					<div>
						<span class="block text-sm font-medium text-gray-700 mb-1.5">
							Search Window
						</span>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label for="searchStart" class="block text-xs text-gray-500 mb-1">From</label>
								<input
									id="searchStart"
									type="datetime-local"
									bind:value={searchStart}
									class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 
										focus:border-blue-500 text-sm"
								/>
							</div>
							<div>
								<label for="searchEnd" class="block text-xs text-gray-500 mb-1">Until</label>
								<input
									id="searchEnd"
									type="datetime-local"
									bind:value={searchEnd}
									class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 
										focus:border-blue-500 text-sm"
								/>
							</div>
						</div>
					</div>

					<!-- Preferences -->
					<details class="border border-gray-200 rounded-lg overflow-hidden" open>
						<summary class="px-4 py-3 cursor-pointer text-sm font-medium text-gray-700 hover:bg-gray-50 
							flex items-center justify-between select-none">
							<span class="flex items-center gap-2">
								<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
								</svg>
								Scheduling Preferences
							</span>
							<svg class="w-4 h-4 text-gray-400 transition-transform details-open:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
							</svg>
						</summary>
						<div class="px-4 pb-4 pt-3 space-y-4 border-t bg-gray-50/50">
							<!-- Preferred Hours -->
							<div>
								<span class="block text-xs font-medium text-gray-600 mb-2">Preferred Hours</span>
								<div class="flex items-center gap-3">
									<select
										bind:value={preferredStartHour}
										class="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 bg-white"
									>
										{#each Array(24).fill(0).map((_, i) => i) as hour}
											<option value={hour}>{formatHour(hour)}</option>
										{/each}
									</select>
									<span class="text-gray-400">to</span>
									<select
										bind:value={preferredEndHour}
										class="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 bg-white"
									>
										{#each Array(24).fill(0).map((_, i) => i) as hour}
											<option value={hour}>{formatHour(hour)}</option>
										{/each}
									</select>
								</div>
							</div>

							<!-- Checkboxes Row -->
							<div class="flex flex-wrap gap-x-6 gap-y-3">
								<label class="flex items-center gap-2 cursor-pointer">
									<input
										type="checkbox"
										bind:checked={avoidBackToBack}
										class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
									/>
									<span class="text-sm text-gray-700">Avoid back-to-back</span>
								</label>
								<label class="flex items-center gap-2 cursor-pointer">
									<input
										type="checkbox"
										bind:checked={includeWeekends}
										class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
									/>
									<span class="text-sm text-gray-700">Include weekends</span>
								</label>
							</div>
							
							{#if avoidBackToBack}
								<div transition:fly={{ y: -10, duration: 150 }}>
									<span class="block text-xs font-medium text-gray-600 mb-1.5">Buffer time</span>
									<div class="flex gap-2">
										{#each [5, 10, 15, 30] as mins}
											<button
												onclick={() => bufferMinutes = mins}
												class="btn-pill btn-pill-sm {bufferMinutes === mins ? 'btn-pill-primary' : 'btn-pill-secondary'}"
											>
												{mins} min
											</button>
										{/each}
									</div>
								</div>
							{/if}
						</div>
					</details>

					<!-- Description -->
					<div>
						<label for="description" class="block text-sm font-medium text-gray-700 mb-1.5">
							Description <span class="text-gray-400 font-normal">(optional)</span>
						</label>
						<textarea
							id="description"
							bind:value={description}
							rows="2"
							placeholder="Meeting agenda, context, etc."
							class="w-full px-3 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 
								focus:border-blue-500 resize-none text-sm"
						></textarea>
					</div>
				</div>
			{:else if proposal}
				<!-- Step 2: Proposal Results -->
				<div class="space-y-4" transition:fly={{ x: 20, duration: 200 }}>
					<!-- Meeting Summary Card -->
					<div class="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg p-4 border border-blue-100">
						<div class="flex items-start justify-between">
							<div>
								<h3 class="font-semibold text-gray-900 text-lg">{proposal.request.title}</h3>
								<div class="flex items-center gap-3 mt-1 text-sm text-gray-600">
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
										{proposal.request.duration_minutes} min
									</span>
									<span class="flex items-center gap-1">
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
										</svg>
										{proposal.request.attendees.length} attendee(s)
									</span>
								</div>
								<!-- Attendee chips -->
								<div class="flex flex-wrap gap-1.5 mt-3">
									{#each proposal.request.attendees.slice(0, 4) as email}
										<span class="inline-flex items-center gap-1 px-2 py-0.5 bg-white/70 rounded-full text-xs">
											<span class="w-4 h-4 rounded-full {getAvatarColor(email)} text-white text-[10px] 
												flex items-center justify-center font-medium">
												{getInitials(email)}
											</span>
											<span class="text-gray-600 max-w-[100px] truncate">{email.split('@')[0]}</span>
										</span>
									{/each}
									{#if proposal.request.attendees.length > 4}
										<span class="text-xs text-gray-500 px-2 py-0.5">
											+{proposal.request.attendees.length - 4} more
										</span>
									{/if}
								</div>
							</div>
							<button
								onclick={goBack}
								class="btn-pill btn-pill-ghost btn-pill-xs"
							>
								← Edit
							</button>
						</div>
					</div>

					{#if proposal.proposed_slots.length === 0}
						<!-- Empty State -->
						<div class="text-center py-12 px-4">
							<div class="w-16 h-16 mx-auto mb-4 bg-gray-100 rounded-full flex items-center justify-center">
								<svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
								</svg>
							</div>
							<h3 class="text-lg font-medium text-gray-900 mb-1">No Available Times</h3>
							<p class="text-gray-500 mb-4">We couldn't find any times when everyone is free.</p>
							<div class="text-sm text-gray-600 space-y-1">
								<p>Try:</p>
								<ul class="list-disc list-inside text-left max-w-xs mx-auto">
									<li>Expanding the search window</li>
									<li>Including weekends</li>
									<li>Adjusting preferred hours</li>
									<li>Reducing the meeting duration</li>
								</ul>
							</div>
						</div>
					{:else}
						<div class="flex items-center justify-between">
							<p class="text-sm text-gray-600">
								Found <span class="font-semibold text-gray-900">{proposal.proposed_slots.length}</span> available time(s)
							</p>
							<span class="text-xs text-gray-400">Click to select</span>
						</div>

						<div class="space-y-2 max-h-[350px] overflow-y-auto pr-1">
							{#each proposal.proposed_slots as slot, index (slot.start)}
								<button
									onclick={() => selectedSlot = slot}
									class="w-full p-4 text-left {selectedSlot === slot ? 'btn-pill btn-pill-soft' : 'btn-pill btn-pill-secondary'}"
								>
									<div class="flex items-start justify-between gap-3">
										<div class="flex-1">
											<div class="flex items-center gap-2 mb-1">
												{#if index === 0}
													<span class="px-2 py-0.5 text-xs font-semibold bg-gradient-to-r from-blue-500 to-indigo-500 text-white rounded-full">
														⭐ Best Match
													</span>
												{:else if index < 3}
													<span class="px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded-full">
														#{index + 1}
													</span>
												{/if}
											</div>
											<p class="font-semibold text-gray-900 text-base">
												{formatDateTime(slot.start)}
											</p>
											<p class="text-sm text-gray-600 mt-0.5 flex items-center gap-1">
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
												{formatTimeOnly(slot.start)} – {formatTimeOnly(slot.end)}
											</p>
											<p class="text-xs text-gray-500 mt-2 flex items-center gap-1">
												{getScoreEmoji(slot.score)} {slot.reason}
											</p>
										</div>
										<div class="flex flex-col items-end gap-2">
											<span class="inline-flex items-center px-2.5 py-1 rounded-lg text-sm font-semibold {getScoreColor(slot.score)}">
												{slot.score}%
											</span>
											{#if selectedSlot === slot}
												<svg class="w-5 h-5 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
													<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
												</svg>
											{/if}
										</div>
									</div>
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-between px-6 py-4 border-t bg-gray-50">
			<div>
				{#if step === 2 && proposal && proposal.proposed_slots.length > 0}
					<p class="text-xs text-gray-500">
						{selectedSlot ? 'Ready to schedule!' : 'Select a time slot above'}
					</p>
				{/if}
			</div>
			<div class="flex items-center gap-3">
				<button
					onclick={handleClose}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					Cancel
				</button>
				
				{#if step === 1}
					<button
						onclick={handleFindTimes}
						disabled={loading || !title.trim() || attendees.length === 0}
						class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
					>
						{#if loading}
							<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Finding times...
						{:else}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
							</svg>
							Find Available Times
						{/if}
					</button>
				{:else}
					<button
						onclick={handleConfirmSlot}
						disabled={!selectedSlot || creating}
						class="btn-pill btn-pill-success btn-pill-sm flex items-center gap-2"
					>
						{#if creating}
							<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Creating...
						{:else}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
							Create Meeting
						{/if}
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	/* Handle details marker rotation */
	details[open] summary svg:last-child {
		transform: rotate(180deg);
	}
</style>
