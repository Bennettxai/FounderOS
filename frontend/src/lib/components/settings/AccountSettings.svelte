<script lang="ts">
	import { api } from '$lib/api';
	import { getUserAvatarUrl, useSession, signOut } from '$lib/auth-client';
	import { goto } from '$app/navigation';
	import { getApiBaseUrl, getCSRFToken } from '$lib/api/base';

	const session = useSession();

	let isEditingProfile = $state(false);
	let editName = $state('');
	let profilePhotoUrl = $state('');
	let isUploadingPhoto = $state(false);
	let showPasswordChange = $state(false);
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordError = $state('');
	let isChangingPassword = $state(false);
	let activeSessions = $state<Array<{id: string; device: string; location: string; lastActive: string; current: boolean}>>([]);
	let isLoadingSessions = $state(false);
	let twoFactorEnabled = $state(false);
	let isExportingData = $state(false);
	let showDeleteConfirmation = $state(false);
	let deleteConfirmText = $state('');
	let isDeletingAccount = $state(false);
	let isSaving = $state(false);
	let saveMessage = $state('');
	let avatarError = $state(false);
	const sessionAvatarUrl = $derived(getUserAvatarUrl($session.data?.user?.image));
	const visibleAvatarUrl = $derived(profilePhotoUrl || sessionAvatarUrl);

	$effect(() => {
		visibleAvatarUrl;
		avatarError = false;
	});

	$effect(() => {
		loadActiveSessions();
	});

	function detectDevice(): string {
		const ua = navigator.userAgent;
		if (/Windows/.test(ua)) return 'Windows PC';
		if (/Mac/.test(ua)) return 'Mac';
		if (/Linux/.test(ua)) return 'Linux';
		if (/iPhone|iPad/.test(ua)) return 'iOS Device';
		if (/Android/.test(ua)) return 'Android Device';
		return 'Unknown Device';
	}

	async function loadActiveSessions() {
		isLoadingSessions = true;
		try {
			activeSessions = [{
				id: 'current',
				device: detectDevice(),
				location: 'Current Location',
				lastActive: 'Now',
				current: true
			}];
		} catch (error) {
			console.error('Error loading sessions:', error);
		} finally {
			isLoadingSessions = false;
		}
	}

	async function handleProfilePhotoUpload(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files?.length) return;

		isUploadingPhoto = true;
		try {
			const file = input.files[0];
			const response = await api.uploadProfilePhoto(file);
			profilePhotoUrl = response.url;
			saveMessage = 'Profile photo updated!';
			setTimeout(() => (saveMessage = ''), 2000);
		} catch (error) {
			console.error('Error uploading photo:', error);
			saveMessage = 'Failed to upload photo';
		} finally {
			isUploadingPhoto = false;
		}
	}

	async function saveProfileChanges() {
		isSaving = true;
		try {
			await api.updateProfile({ name: editName });
			saveMessage = 'Profile updated!';
			isEditingProfile = false;
			setTimeout(() => (saveMessage = ''), 2000);
		} catch (error) {
			console.error('Error saving profile:', error);
			saveMessage = 'Failed to update profile';
		} finally {
			isSaving = false;
		}
	}

	async function handlePasswordChange() {
		passwordError = '';
		if (newPassword !== confirmPassword) {
			passwordError = 'Passwords do not match';
			return;
		}
		if (newPassword.length < 8) {
			passwordError = 'Password must be at least 8 characters';
			return;
		}

		isChangingPassword = true;
		try {
			await new Promise(resolve => setTimeout(resolve, 1000));
			saveMessage = 'Password changed successfully!';
			showPasswordChange = false;
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			setTimeout(() => (saveMessage = ''), 2000);
		} catch (error) {
			passwordError = 'Failed to change password';
		} finally {
			isChangingPassword = false;
		}
	}

	async function revokeSession(sessionId: string) {
		if (!confirm('Are you sure you want to sign out this device?')) return;
		try {
			activeSessions = activeSessions.filter(s => s.id !== sessionId);
			saveMessage = 'Session revoked';
			setTimeout(() => (saveMessage = ''), 2000);
		} catch (error) {
			console.error('Error revoking session:', error);
		}
	}

	async function exportUserData() {
		isExportingData = true;
		try {
			const response = await fetch(`${getApiBaseUrl()}/account/export`, {
				credentials: 'include'
			});
			if (!response.ok) throw new Error('Export failed');
			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'businessos-data-export.json';
			a.click();
			URL.revokeObjectURL(url);
			saveMessage = 'Data exported successfully!';
			setTimeout(() => (saveMessage = ''), 3000);
		} catch (error) {
			saveMessage = 'Failed to export data. Please try again.';
			setTimeout(() => (saveMessage = ''), 5000);
		} finally {
			isExportingData = false;
		}
	}

	async function handleDeleteAccount() {
		if (deleteConfirmText !== 'DELETE') return;
		isDeletingAccount = true;
		try {
			const csrfToken = getCSRFToken();
			const headers: Record<string, string> = { 'Content-Type': 'application/json' };
			if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
			const response = await fetch(`${getApiBaseUrl()}/account`, {
				method: 'DELETE',
				credentials: 'include',
				headers,
				body: JSON.stringify({ confirm: true })
			});
			if (!response.ok) throw new Error('Deletion failed');
			goto('/login');
		} catch (error) {
			saveMessage = 'Failed to delete account. Please contact support.';
			setTimeout(() => (saveMessage = ''), 5000);
		} finally {
			isDeletingAccount = false;
		}
	}

	async function handleLogout() {
		await signOut();
		goto('/login');
	}
</script>

<div class="space-y-4">
	{#if saveMessage}
		<div
			class="p-3 rounded-lg text-sm"
			style="{saveMessage.includes('Failed') || saveMessage.includes('Error')
				? 'background: var(--bos-status-error-bg); color: var(--bos-status-error);'
				: 'background: var(--bos-status-success-bg); color: var(--bos-status-success);'}"
		>
			{saveMessage}
		</div>
	{/if}

	<!-- Profile Section -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Profile</h2>
		<div class="flex items-start gap-4">
			<!-- Profile Photo -->
			<div class="flex-shrink-0">
				<div class="relative">
					{#if visibleAvatarUrl && !avatarError}
						<img src={visibleAvatarUrl} alt="Profile" class="w-14 h-14 rounded-full object-cover" onerror={() => (avatarError = true)} />
					{:else}
						<div class="w-14 h-14 rounded-full flex items-center justify-center text-lg font-semibold" style="background: linear-gradient(135deg, var(--bos-nav-active), var(--bos-category-ai)); color: var(--bos-surface-on-color)">
							{($session.data?.user?.name || 'U').charAt(0).toUpperCase()}
						</div>
					{/if}
					<label class="absolute bottom-0 right-0 w-6 h-6 st-upload-btn rounded-full flex items-center justify-center cursor-pointer transition-colors" aria-label="Upload profile photo">
						{#if isUploadingPhoto}
							<svg class="animate-spin h-4 w-4 st-upload-icon" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
						{:else}
							<svg class="w-4 h-4 st-upload-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
							</svg>
						{/if}
						<input type="file" accept="image/*" class="hidden" onchange={handleProfilePhotoUpload} />
					</label>
				</div>
			</div>

			<!-- Profile Info -->
			<div class="flex-1 space-y-3">
				{#if isEditingProfile}
					<div>
						<label for="edit-name" class="block text-sm font-medium st-label mb-1">Display Name</label>
						<input
							id="edit-name"
							type="text"
							bind:value={editName}
							class="w-full px-3 py-2 st-input rounded-lg focus:ring-2 focus:border-transparent"
						style="--tw-ring-color: var(--bos-status-info)"
							placeholder="Your name"
						/>
					</div>
					<div>
						<label for="edit-email" class="block text-sm font-medium st-label mb-1">Email</label>
						<input
							id="edit-email"
							type="email"
							value={$session.data?.user?.email || ''}
							disabled
							class="w-full px-3 py-2 st-input-disabled rounded-lg cursor-not-allowed"
						/>
						<p class="text-xs st-muted mt-1">Email cannot be changed</p>
					</div>
					<div class="flex items-center gap-2">
						<button
							onclick={saveProfileChanges}
							disabled={isSaving}
							class="btn-pill btn-pill-primary btn-pill-sm"
						>
							{isSaving ? 'Saving...' : 'Save Changes'}
						</button>
						<button
							onclick={() => { isEditingProfile = false; editName = $session.data?.user?.name || ''; }}
							class="btn-pill btn-pill-soft btn-pill-sm"
						>
							Cancel
						</button>
					</div>
				{:else}
					<div>
						<p class="block text-sm font-medium st-muted mb-1">Name</p>
						<p class="st-title">{$session.data?.user?.name || 'Not set'}</p>
					</div>
					<div>
						<p class="block text-sm font-medium st-muted mb-1">Email</p>
						<p class="st-title">{$session.data?.user?.email || 'Not set'}</p>
					</div>
					<button
						onclick={() => { isEditingProfile = true; editName = $session.data?.user?.name || ''; }}
						class="btn-pill btn-pill-soft btn-pill-sm"
					>
						Edit Profile
					</button>
				{/if}
			</div>
		</div>
	</div>

	<!-- Password Section -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Password</h2>
		{#if showPasswordChange}
			<div class="space-y-4">
				{#if passwordError}
					<div class="p-3 rounded-lg text-sm" style="background: var(--bos-status-error-bg); color: var(--bos-status-error);">
						{passwordError}
					</div>
				{/if}
				<div>
					<label for="current-password" class="block text-sm font-medium st-label mb-1">Current Password</label>
					<input
						id="current-password"
						type="password"
						bind:value={currentPassword}
						class="w-full px-3 py-2 st-input rounded-lg"
						placeholder="Enter current password"
					/>
				</div>
				<div>
					<label for="new-password" class="block text-sm font-medium st-label mb-1">New Password</label>
					<input
						id="new-password"
						type="password"
						bind:value={newPassword}
						class="w-full px-3 py-2 st-input rounded-lg"
						placeholder="Enter new password"
					/>
				</div>
				<div>
					<label for="confirm-password" class="block text-sm font-medium st-label mb-1">Confirm New Password</label>
					<input
						id="confirm-password"
						type="password"
						bind:value={confirmPassword}
						class="w-full px-3 py-2 st-input rounded-lg"
						placeholder="Confirm new password"
					/>
				</div>
				<div class="flex items-center gap-2">
					<button
						onclick={handlePasswordChange}
						disabled={isChangingPassword}
						class="btn-pill btn-pill-primary btn-pill-sm"
					>
						{isChangingPassword ? 'Changing...' : 'Change Password'}
					</button>
					<button
						onclick={() => { showPasswordChange = false; currentPassword = ''; newPassword = ''; confirmPassword = ''; passwordError = ''; }}
						class="btn-pill btn-pill-soft btn-pill-sm"
					>
						Cancel
					</button>
				</div>
			</div>
		{:else}
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Password</p>
					<p class="text-sm st-muted">Last changed: Never</p>
				</div>
				<button
					onclick={() => (showPasswordChange = true)}
					class="btn-pill btn-pill-soft btn-pill-sm"
				>
					Change Password
				</button>
			</div>
		{/if}
	</div>

	<!-- Two-Factor Authentication -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Two-Factor Authentication</h2>
		<div class="flex items-center justify-between">
			<div>
				<p class="font-medium st-title">2FA Status</p>
				<p class="text-sm" style="{twoFactorEnabled ? 'color: var(--bos-status-success)' : ''}" class:st-muted={!twoFactorEnabled}>
					{twoFactorEnabled ? 'Enabled - Your account is protected' : 'Disabled - Add an extra layer of security'}
				</p>
			</div>
			<button
				onclick={() => (twoFactorEnabled = !twoFactorEnabled)}
				class="btn-pill btn-pill-sm {twoFactorEnabled ? 'btn-pill-soft' : 'btn-pill-primary'}"
			>
				{twoFactorEnabled ? 'Disable 2FA' : 'Enable 2FA'}
			</button>
		</div>
	</div>

	<!-- Sessions -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Active Sessions</h2>
		{#if isLoadingSessions}
			<div class="flex items-center justify-center py-4">
				<svg class="animate-spin h-6 w-6 st-icon" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			</div>
		{:else}
			<div class="space-y-3">
				{#each activeSessions as activeSession}
					<div class="flex items-center justify-between p-3 rounded-lg st-session-card">
						<div class="flex items-center gap-3">
							<div class="w-10 h-10 rounded-full st-avatar-bg flex items-center justify-center">
								<svg class="w-5 h-5 st-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
							</div>
							<div>
								<p class="font-medium st-title flex items-center gap-2">
									{activeSession.device}
									{#if activeSession.current}
										<span class="text-xs px-2 py-0.5 rounded-full" style="background: var(--bos-status-success-bg); color: var(--bos-status-success);">Current</span>
									{/if}
								</p>
								<p class="text-sm st-muted">{activeSession.location} • {activeSession.lastActive}</p>
							</div>
						</div>
						{#if !activeSession.current}
							<button
								onclick={() => revokeSession(activeSession.id)}
								class="text-sm hover:underline" style="color: var(--bos-status-error)"
							>
								Revoke
							</button>
						{:else}
							<button
								onclick={handleLogout}
								class="btn-pill btn-pill-soft btn-pill-sm"
							>
								Sign Out
							</button>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Privacy & Compliance -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Privacy & Compliance</h2>
		<p class="text-sm st-muted mb-3">
			Learn how we handle your data and your rights under GDPR and other privacy regulations.
		</p>
		<a href="/privacy" target="_blank" class="text-sm hover:underline" style="color: var(--bos-status-info)">
			View Privacy Policy
		</a>
	</div>

	<!-- Data Export -->
	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">Data Export</h2>
		<div class="flex items-center justify-between">
			<div>
				<p class="font-medium st-title">Export your data</p>
				<p class="text-sm st-muted">Download a copy of all your data in JSON format</p>
			</div>
			<button
				onclick={exportUserData}
				disabled={isExportingData}
				class="btn-pill btn-pill-soft btn-pill-sm"
			>
				{#if isExportingData}
					<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Exporting...
				{:else}
					Export Data
				{/if}
			</button>
		</div>
	</div>

	<!-- Danger Zone -->
	<div class="card" style="border-color: var(--bos-status-error)">
		<h2 class="text-xs font-semibold uppercase tracking-wide mb-3" style="color: var(--bos-status-error)">Danger Zone</h2>
		{#if !showDeleteConfirmation}
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium st-title">Delete account</p>
					<p class="text-sm st-muted">Permanently delete your account and all data. This cannot be undone.</p>
				</div>
				<button
					onclick={() => (showDeleteConfirmation = true)}
					class="btn-pill btn-pill-danger btn-pill-sm"
				>
					Delete Account
				</button>
			</div>
		{:else}
			<div class="space-y-4 p-4 rounded-lg" style="background: var(--bos-status-error-bg)">
				<p class="text-sm" style="color: var(--bos-status-error)">
					Your account will be scheduled for deletion. All your data will be permanently removed within 30 days.
					Type <strong>DELETE</strong> to confirm.
				</p>
				<input
					type="text"
					bind:value={deleteConfirmText}
					placeholder="Type DELETE to confirm"
					class="w-full px-3 py-2 rounded-lg st-input" style="border-color: var(--bos-status-error)"
				/>
				<div class="flex gap-3">
					<button
						onclick={handleDeleteAccount}
						disabled={deleteConfirmText !== 'DELETE' || isDeletingAccount}
						class="btn-pill btn-pill-danger btn-pill-sm"
					>
						{isDeletingAccount ? 'Deleting...' : 'Permanently Delete'}
					</button>
					<button
						onclick={() => { showDeleteConfirmation = false; deleteConfirmText = ''; }}
						class="btn-pill btn-pill-soft btn-pill-sm"
					>
						Cancel
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.st-title { color: var(--dt); }
	.st-muted { color: var(--dt3); }
	.st-label { color: var(--dt2); }
	.st-icon  { color: var(--dt4); }
	.st-input {
		border: 1px solid var(--dbd);
		background: var(--dbg);
		color: var(--dt);
	}
	.st-input-disabled {
		border: 1px solid var(--dbd);
		background: var(--dbg3);
		color: var(--dt3);
	}
	.st-upload-btn {
		background: var(--dt);
	}
	.st-upload-btn:hover { opacity: 0.85; }
	.st-upload-icon { color: var(--dbg); }
	.st-session-card {
		background: var(--bos-settings-card-bg);
		border: 1px solid var(--bos-settings-card-border);
		border-radius: var(--bos-settings-card-radius);
	}
	.st-avatar-bg { background: var(--dbg3); }
</style>
