<!--
	Onboarding Screen 2: Quick Info
	Collects workspace name, role, business type, and team size
-->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { get } from 'svelte/store';
	import { PillButton, RoundedInput, PillSelect } from '$lib/components/osa';
	import { onboardingStore } from '$lib/stores/onboardingStore';
	import { cloudServerUrl } from '$lib/auth-client';

	let role = $state('');
	let workspaceName = $state('');
	let inviteCode = $state('');
	let businessType = $state('');
	let teamSize = $state('');
	let isValidating = $state(false);
	let invitePreview = $state<{
		valid: boolean;
		workspaceName?: string;
		workspaceId?: string;
		role?: string;
	} | null>(null);

	let errors = $state({
		role: '',
		workspaceName: '',
		inviteCode: '',
		businessType: '',
		teamSize: ''
	});

	// Validate invite code with backend
	async function validateInviteCode(code: string): Promise<void> {
		if (!code.trim()) {
			invitePreview = null;
			return;
		}

		isValidating = true;
		errors.inviteCode = '';

		try {
			const response = await fetch(`${get(cloudServerUrl)}/api/workspaces/invites/validate`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ token: code.trim() })
			});

			const data = await response.json();

			if (data.valid) {
				invitePreview = {
					valid: true,
					workspaceName: data.workspace_name,
					workspaceId: data.workspace_id,
					role: data.role
				};
			} else {
				invitePreview = { valid: false };
				errors.inviteCode = data.error || 'Invalid invite code';
			}
		} catch (err) {
			console.error('Error validating invite:', err);
			invitePreview = { valid: false };
			errors.inviteCode = 'Failed to validate invite code';
		} finally {
			isValidating = false;
		}
	}

	// Debounced invite code validation
	let validateTimeout: ReturnType<typeof setTimeout>;
	$effect(() => {
		if (role === 'employee' && inviteCode) {
			clearTimeout(validateTimeout);
			validateTimeout = setTimeout(() => {
				validateInviteCode(inviteCode);
			}, 500);
		} else {
			invitePreview = null;
		}
	});

	// Accept invite and join workspace
	async function acceptInviteAndJoin(): Promise<boolean> {
		try {
			const response = await fetch(`${get(cloudServerUrl)}/api/workspaces/invites/accept`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ token: inviteCode.trim() })
			});

			if (!response.ok) {
				const data = await response.json();
				errors.inviteCode = data.error || 'Failed to accept invite';
				return false;
			}

			return true;
		} catch (err) {
			console.error('Error accepting invite:', err);
			errors.inviteCode = 'Failed to join workspace';
			return false;
		}
	}

	// Owner roles that create workspaces
	const ownerRoles = ['founder', 'consultant', 'freelancer', 'other'];
	const isOwner = $derived(ownerRoles.includes(role));

	const roleOptions = [
		{ value: 'founder', label: 'Founder / CEO' },
		{ value: 'consultant', label: 'Consultant' },
		{ value: 'employee', label: 'Employee' },
		{ value: 'freelancer', label: 'Freelancer' },
		{ value: 'other', label: 'Other' }
	];

	const businessTypeOptions = [
		{ value: 'agency', label: 'Agency' },
		{ value: 'startup', label: 'Startup' },
		{ value: 'freelance', label: 'Freelance / Solo' },
		{ value: 'consulting', label: 'Consulting' },
		{ value: 'ecommerce', label: 'E-commerce' },
		{ value: 'saas', label: 'SaaS' },
		{ value: 'other', label: 'Other' }
	];

	const teamSizeOptions = [
		{ value: 'solo', label: 'Just me' },
		{ value: '2-5', label: '2-5 people' },
		{ value: '6-10', label: '6-10 people' },
		{ value: '11-50', label: '11-50 people' },
		{ value: '50+', label: '50+ people' }
	];

	function validate(): boolean {
		let isValid = true;
		errors = { role: '', workspaceName: '', inviteCode: '', businessType: '', teamSize: '' };

		if (!role) {
			errors.role = 'Please select your role';
			isValid = false;
		}

		if (isOwner) {
			if (!workspaceName.trim()) {
				errors.workspaceName = 'Workspace name is required';
				isValid = false;
			} else if (workspaceName.trim().length < 2) {
				errors.workspaceName = 'Workspace name must be at least 2 characters';
				isValid = false;
			}
		} else if (role === 'employee') {
			if (!inviteCode.trim()) {
				errors.inviteCode = 'Invite code is required';
				isValid = false;
			}
			// Employees skip business type and team size
			return isValid;
		}

		if (!businessType) {
			errors.businessType = 'Please select your business type';
			isValid = false;
		}

		if (!teamSize) {
			errors.teamSize = 'Please select your team size';
			isValid = false;
		}

		return isValid;
	}

	async function handleContinue() {
		if (!validate()) return;

		// Employee flow: validate invite, accept, and skip to /window
		if (role === 'employee' && inviteCode) {
			// First ensure invite is valid
			if (!invitePreview?.valid) {
				await validateInviteCode(inviteCode);
				if (!invitePreview?.valid) {
					return; // Error already set
				}
			}

			// Accept the invite and join workspace
			isValidating = true;
			const success = await acceptInviteAndJoin();
			isValidating = false;

			if (success) {
				// Mark onboarding as complete and go to app
				await onboardingStore.complete();
				goto('/window');
				return;
			}
			return; // Error shown, stay on page
		}

		// Owner flow: continue through onboarding
		onboardingStore.setQuickInfo({
			workspaceName: isOwner ? workspaceName.trim() : '',
			role,
			businessType,
			teamSize
		});

		onboardingStore.nextStep();
		goto('/onboarding/username');
	}

	function handleBack() {
		onboardingStore.prevStep();
		goto('/onboarding');
	}
</script>

<svelte:head>
	<title>Quick Info - OSA Build</title>
</svelte:head>

<div class="onboarding-background">
	<div class="quick-info-screen">
		<div class="content">
			<!-- Main Message -->
			<div class="header">
				<h1 class="title">
					Tell us about<br />your work.
				</h1>
				<p class="subtitle">
					This helps us personalize your experience
				</p>
			</div>

			<!-- Form Fields -->
			<div class="form-section">
				<!-- Step 1: Role Selection (always first) -->
				<PillSelect
					label="Your Role"
					bind:value={role}
					options={roleOptions}
					error={errors.role}
					columns={3}
					required
				/>

				<!-- Step 2: Workspace Name OR Invite Code (conditional) -->
				{#if role}
					<div class="conditional-field">
						{#if isOwner}
							<RoundedInput
								label="Name your workspace"
								bind:value={workspaceName}
								placeholder="My Company"
								error={errors.workspaceName}
								required
							/>
						{:else}
							<RoundedInput
								label="Enter invite code"
								bind:value={inviteCode}
								placeholder="Paste your invite code from email"
								error={errors.inviteCode}
								required
							/>
							{#if isValidating}
								<p class="validating-text">Checking invite code...</p>
							{:else if invitePreview?.valid}
								<div class="invite-preview">
									<p class="preview-label">You're joining:</p>
									<p class="workspace-name">{invitePreview.workspaceName}</p>
									<p class="role-badge">as {invitePreview.role}</p>
								</div>
							{/if}
						{/if}
					</div>
				{/if}

				<!-- Step 3: Business Type (owners only) -->
				{#if isOwner || !role}
					<PillSelect
						label="Business Type"
						bind:value={businessType}
						options={businessTypeOptions}
						error={errors.businessType}
						columns={4}
						required
					/>
				{/if}

				<!-- Step 4: Team Size (owners only) -->
				{#if isOwner || !role}
					<PillSelect
						label="Team Size"
						bind:value={teamSize}
						options={teamSizeOptions}
						error={errors.teamSize}
						columns={3}
						required
					/>
				{/if}
			</div>

			<!-- CTA Buttons -->
			<div class="cta">
				<PillButton
					variant="primary"
					size="lg"
					onclick={handleContinue}
					disabled={isValidating || (role === 'employee' && !invitePreview?.valid)}
				>
					{#if isValidating}
						Joining...
					{:else if role === 'employee' && invitePreview?.valid}
						Join Workspace
					{:else}
						Continue
					{/if}
				</PillButton>

				<button class="btn-pill btn-pill-ghost back-button" onclick={handleBack}>
					Back
				</button>
			</div>
		</div>
	</div>
</div>

<style>
	.onboarding-background {
		min-height: 100vh;
		width: 100%;
		background-image: url('/logos/integrations/MIOSABRANDBackround.png');
		background-size: cover;
		background-position: center;
		background-repeat: no-repeat;
	}

	.quick-info-screen {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem;
	}

	.content {
		width: 100%;
		max-width: 560px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2.5rem;
		text-align: center;
	}

	.header {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		animation: fadeIn 0.8s ease-out 0.2s both;
	}

	.title {
		font-size: 2.5rem;
		font-weight: 700;
		color: #1A1A1A;
		line-height: 1.2;
		letter-spacing: -0.02em;
		margin: 0;
	}

	.subtitle {
		font-size: 1rem;
		color: #666666;
		margin: 0;
	}

	.form-section {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		animation: fadeIn 0.8s ease-out 0.3s both;
	}

	.cta {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
		animation: fadeIn 0.8s ease-out 0.4s both;
	}

	.back-button {
		background: transparent;
		border: none;
		color: #666666;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		padding: 0.5rem 1rem;
		font-family: inherit;
		transition: color 0.2s ease;
	}

	.back-button:hover {
		color: #1A1A1A;
	}

	.conditional-field {
		animation: slideIn 0.3s ease-out;
	}

	.validating-text {
		font-size: 0.875rem;
		color: #666666;
		margin: 0.5rem 0 0;
		text-align: left;
	}

	.invite-preview {
		margin-top: 1rem;
		padding: 1rem;
		background: rgba(16, 185, 129, 0.08);
		border: 1px solid rgba(16, 185, 129, 0.3);
		border-radius: 12px;
		text-align: left;
		animation: slideIn 0.3s ease-out;
	}

	.preview-label {
		font-size: 0.75rem;
		color: #666666;
		margin: 0 0 0.25rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.workspace-name {
		font-size: 1.125rem;
		font-weight: 600;
		color: #1A1A1A;
		margin: 0;
	}

	.role-badge {
		font-size: 0.875rem;
		color: #10B981;
		margin: 0.25rem 0 0;
		font-weight: 500;
	}

	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateY(-10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	@media (max-width: 768px) {
		.title {
			font-size: 2rem;
		}

		.content {
			gap: 2rem;
		}
	}

	@media (max-width: 480px) {
		.quick-info-screen {
			padding: 1.25rem 1rem;
			align-items: flex-start;
			padding-top: 2rem;
		}

		.content {
			gap: 1.5rem;
		}

		.title {
			font-size: 1.75rem;
		}

		.back-button {
			min-height: 2.75rem;
			padding: 0.5rem 1.5rem;
		}
	}

	@media (max-width: 320px) {
		.quick-info-screen {
			padding: 1rem 0.75rem;
		}

		.title {
			font-size: 1.5rem;
		}
	}
</style>
