<script lang="ts">
	import { Dialog, DropdownMenu } from 'bits-ui';
	import { Users, Mail, ChevronDown, X, CheckCircle, AlertCircle, Loader2, Send } from 'lucide-svelte';
	import { createWorkspaceInvite } from '$lib/api/workspaces';

	interface MemberListItem {
		id: string;
		name: string;
		email: string;
		avatar?: string;
		role: string;
	}

	interface RoleOption {
		id: string;
		name: string;
		display_name: string;
		description?: string;
	}

	interface Props {
		open?: boolean;
		members?: MemberListItem[];
		onClose?: () => void;
		onCreate?: (member: { email: string; role: string }) => void;
		workspaceId?: string;
		roles?: RoleOption[];
		onInvite?: (data: { email: string; role: string }) => void;
	}

	let {
		open = $bindable(false),
		members = [],
		onClose,
		onCreate,
		workspaceId,
		roles = [],
		onInvite
	}: Props = $props();

	const ROLES = ['Assistant', 'Junior Level', 'Mid Level', 'Senior Level', 'Lead', 'Manager'];

	// Tab state
	let activeTab = $state<'members' | 'invite'>('members');

	// Members tab state
	let memberRoles = $state<Record<string, string>>(
		Object.fromEntries(members.map(m => [m.id, m.role]))
	);

	// Invite tab state
	let inviteEmail = $state('');
	let selectedRole = $state('member');
	let isSending = $state(false);
	let inviteError = $state<string | null>(null);
	let inviteSuccess = $state(false);

	// Keep memberRoles in sync if members prop changes
	$effect(() => {
		for (const m of members) {
			if (!(m.id in memberRoles)) {
				memberRoles = { ...memberRoles, [m.id]: m.role };
			}
		}
	});

	// Filter out owner role from assignable roles
	const assignableRoles = $derived(roles.filter(r => r.name !== 'owner'));

	const canInvite = $derived(inviteEmail.trim().length > 0 && inviteEmail.includes('@') && !isSending);

	async function handleInvite() {
		if (!canInvite) return;

		const emailVal = inviteEmail.trim();
		const roleVal = selectedRole;

		// If onInvite callback provided, use it
		if (onInvite) {
			onInvite({ email: emailVal, role: roleVal });
			inviteSuccess = true;
			return;
		}

		// If workspaceId provided, call API directly
		if (workspaceId) {
			try {
				isSending = true;
				inviteError = null;

				// Dev mock mode
				if (import.meta.env.DEV && workspaceId.startsWith('mock-')) {
					await new Promise(resolve => setTimeout(resolve, 800));
					inviteSuccess = true;
					return;
				}

				await createWorkspaceInvite(workspaceId, {
					email: emailVal,
					role: roleVal
				});
				inviteSuccess = true;
			} catch (err) {
				inviteError = err instanceof Error ? err.message : 'Failed to send invitation';
			} finally {
				isSending = false;
			}
			return;
		}

		// Fallback: use onCreate prop (legacy behaviour)
		onCreate?.({ email: emailVal, role: roleVal });
		inviteEmail = '';
	}

	function handleInviteKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			handleInvite();
		}
	}

	function handleClose() {
		inviteEmail = '';
		inviteError = null;
		inviteSuccess = false;
		isSending = false;
		activeTab = 'members';
		open = false;
		onClose?.();
	}

	function handleTabSwitch(tab: 'members' | 'invite') {
		activeTab = tab;
		// Reset invite state when switching tabs
		if (tab === 'members') {
			inviteEmail = '';
			inviteError = null;
			inviteSuccess = false;
		}
	}

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map(w => w[0])
			.join('')
			.slice(0, 2)
			.toUpperCase();
	}

	// Deterministic avatar color from name
	const AVATAR_COLORS = [
		'#6366f1', '#8b5cf6', '#ec4899', '#3b82f6',
		'#0ea5e9', '#22c55e', '#f59e0b', '#ef4444',
	];
	function getAvatarColor(name: string): string {
		let hash = 0;
		for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash);
		return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="td-invite-overlay" />
		<Dialog.Content class="td-invite-modal" aria-label="Invite Team Member">

			<!-- Header -->
			<div class="td-invite-modal__header">
				<div class="td-invite-modal__header-left">
					<div class="td-invite-modal__icon" aria-hidden="true">
						<Users size={18} />
					</div>
					<div class="td-invite-modal__titles">
						<Dialog.Title class="td-invite-modal__title">Invite Team Member</Dialog.Title>
						<p class="td-invite-modal__subtitle">Share tasks with other users</p>
					</div>
				</div>
				<Dialog.Close
					class="td-invite-modal__close"
					onclick={handleClose}
					aria-label="Close invite modal"
				>
					<X size={16} />
				</Dialog.Close>
			</div>

			<!-- Body -->
			<div class="td-invite-modal__body">

				<!-- Tabs -->
				<div class="td-invite-tabs" role="tablist" aria-label="Modal sections">
					<button
						role="tab"
						aria-selected={activeTab === 'members'}
						aria-controls="tab-panel-members"
						class="td-invite-tab {activeTab === 'members' ? 'td-invite-tab--active' : ''}"
						onclick={() => handleTabSwitch('members')}
					>
						Members
					</button>
					<button
						role="tab"
						aria-selected={activeTab === 'invite'}
						aria-controls="tab-panel-invite"
						class="td-invite-tab {activeTab === 'invite' ? 'td-invite-tab--active' : ''}"
						onclick={() => handleTabSwitch('invite')}
					>
						Invite
					</button>
				</div>

				<!-- Members Tab Panel -->
				{#if activeTab === 'members'}
					<div
						id="tab-panel-members"
						role="tabpanel"
						aria-label="Members"
						class="td-invite-tab-panel"
					>
						{#if members.length > 0}
							<div class="td-invite-section">
								<span class="td-invite-label" id="member-list-label">Member Lists</span>
								<ul class="td-invite-member-list" role="list" aria-labelledby="member-list-label">
									{#each members as member (member.id)}
										<li class="td-invite-member-row" role="listitem" aria-label="{member.name}, {member.role}">
											<!-- Avatar -->
											<div class="td-invite-member-avatar" aria-hidden="true">
												{#if member.avatar}
													<img
														src={member.avatar}
														alt="{member.name} avatar"
														class="td-invite-member-avatar__img"
													/>
												{:else}
													<div
														class="td-invite-member-avatar__initials"
														style="background: {getAvatarColor(member.name)};"
													>
														{getInitials(member.name)}
													</div>
												{/if}
											</div>

											<!-- Name + Email -->
											<div class="td-invite-member-info">
												<span class="td-invite-member-info__name">{member.name}</span>
												<span class="td-invite-member-info__email">{member.email}</span>
											</div>

											<!-- Role Dropdown -->
											<DropdownMenu.Root>
												<DropdownMenu.Trigger
													class="td-invite-role-pill"
													aria-label="Change role for {member.name}"
												>
													{memberRoles[member.id] ?? member.role}
													<ChevronDown size={12} />
												</DropdownMenu.Trigger>
												<DropdownMenu.Portal>
													<DropdownMenu.Content class="td-invite-dropdown" sideOffset={6}>
														{#each ROLES as roleOption}
															<DropdownMenu.Item
																class="td-invite-dropdown__item {(memberRoles[member.id] ?? member.role) === roleOption ? 'td-invite-dropdown__item--active' : ''}"
																onclick={() => { memberRoles = { ...memberRoles, [member.id]: roleOption }; }}
															>
																{roleOption}
															</DropdownMenu.Item>
														{/each}
													</DropdownMenu.Content>
												</DropdownMenu.Portal>
											</DropdownMenu.Root>
										</li>
									{/each}
								</ul>
							</div>
						{:else}
							<p class="td-invite-empty">No members yet. Switch to the Invite tab to add someone.</p>
						{/if}
					</div>
				{/if}

				<!-- Invite Tab Panel -->
				{#if activeTab === 'invite'}
					<div
						id="tab-panel-invite"
						role="tabpanel"
						aria-label="Invite by email"
						class="td-invite-tab-panel"
					>
						{#if inviteSuccess}
							<!-- Success state -->
							<div class="td-invite-success" aria-live="polite">
								<CheckCircle size={20} />
								<span>Invitation sent to <strong>{inviteEmail}</strong>!</span>
							</div>
						{:else}
							<!-- Error banner -->
							{#if inviteError}
								<div class="td-invite-error-banner" role="alert" aria-live="assertive">
									<AlertCircle size={14} />
									<span>{inviteError}</span>
								</div>
							{/if}

							<!-- Email field -->
							<div class="td-invite-section">
								<label for="invite-email" class="td-invite-label">Email Address</label>
								<div class="td-invite-input-wrap">
									<span class="td-invite-input-wrap__icon" aria-hidden="true">
										<Mail size={14} />
									</span>
									<input
										id="invite-email"
										type="email"
										bind:value={inviteEmail}
										onkeydown={handleInviteKeydown}
										placeholder="member@example.com"
										class="td-invite-input"
										aria-label="Email address to invite"
										disabled={isSending}
									/>
								</div>
							</div>

							<!-- Role field -->
							<div class="td-invite-section">
								<label for="invite-role" class="td-invite-label">Role</label>
								{#if assignableRoles.length > 0}
									<select
										id="invite-role"
										bind:value={selectedRole}
										class="td-invite-select bos-select"
										disabled={isSending}
										aria-label="Select role for invited member"
									>
										{#each assignableRoles as role (role.id)}
											<option value={role.name}>
												{role.display_name}{role.description ? ` — ${role.description}` : ''}
											</option>
										{/each}
									</select>
								{:else}
									<select
										id="invite-role"
										bind:value={selectedRole}
										class="td-invite-select bos-select"
										disabled={isSending}
										aria-label="Select role for invited member"
									>
										{#each ROLES as roleOption}
											<option value={roleOption.toLowerCase().replace(' ', '_')}>{roleOption}</option>
										{/each}
									</select>
								{/if}
								<p class="td-invite-hint">The role determines what permissions the member will have.</p>
							</div>

							<!-- Footer action -->
							<div class="td-invite-footer">
								<button
									onclick={handleInvite}
									disabled={!canInvite}
									class="btn-pill btn-pill-primary td-invite-send-btn"
									aria-label="Send invitation"
								>
									{#if isSending}
										<Loader2 size={14} class="td-invite-spinner" />
										Sending...
									{:else}
										<Send size={14} />
										Invite Member
									{/if}
								</button>
							</div>
						{/if}
					</div>
				{/if}

			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	/* ── Overlay ─────────────────────────────────────────────────── */
	:global(.td-invite-overlay) {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.45);
		z-index: 50;
		backdrop-filter: blur(2px);
	}

	/* ── Modal shell ─────────────────────────────────────────────── */
	:global(.td-invite-modal) {
		position: fixed;
		left: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		z-index: 51;
		width: 100%;
		max-width: 440px;
		background: var(--dbg, #fff);
		border-radius: 18px;
		border: 1px solid var(--dbd, #e0e0e0);
		box-shadow: 0 24px 64px rgba(0, 0, 0, 0.16), 0 4px 16px rgba(0, 0, 0, 0.08);
		overflow: hidden;
		outline: none;
	}

	/* ── Header ──────────────────────────────────────────────────── */
	.td-invite-modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 18px 20px 16px;
		border-bottom: 1px solid var(--dbd, #e0e0e0);
	}

	.td-invite-modal__header-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.td-invite-modal__icon {
		width: 36px;
		height: 36px;
		border-radius: 10px;
		background: color-mix(in srgb, var(--dt, #111) 8%, transparent);
		border: 1px solid var(--dbd, #e0e0e0);
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt, #111);
		flex-shrink: 0;
	}

	.td-invite-modal__titles {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	:global(.td-invite-modal__title) {
		font-size: 14px;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.02em;
		line-height: 1.2;
		margin: 0;
	}

	.td-invite-modal__subtitle {
		font-size: 12px;
		color: var(--dt3, #888);
		font-weight: 400;
		margin: 0;
		line-height: 1.3;
	}

	:global(.td-invite-modal__close) {
		width: 30px;
		height: 30px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--dt3, #888);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: background 0.12s, color 0.12s;
		flex-shrink: 0;
	}
	:global(.td-invite-modal__close:hover) {
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
	}

	/* ── Body ────────────────────────────────────────────────────── */
	.td-invite-modal__body {
		padding: 18px 20px 20px;
		display: flex;
		flex-direction: column;
		gap: 18px;
	}

	/* ── Tabs ────────────────────────────────────────────────────── */
	.td-invite-tabs {
		display: flex;
		flex-direction: row;
		gap: 8px;
	}

	.td-invite-tab {
		height: 28px;
		padding: 0 14px;
		border-radius: 9999px;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.12s, color 0.12s, border-color 0.12s;
		border: 1px solid var(--bos-border-color, var(--dbd, #e0e0e0));
		background: transparent;
		color: var(--bos-text-secondary-color, var(--dt3, #888));
		letter-spacing: -0.01em;
		outline: none;
	}

	.td-invite-tab--active {
		background: var(--bos-text-primary-color, var(--dt, #111));
		color: var(--bos-surface-on-color, #fff);
		border-color: var(--bos-text-primary-color, var(--dt, #111));
	}

	.td-invite-tab:not(.td-invite-tab--active):hover {
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
	}

	/* ── Tab panels ──────────────────────────────────────────────── */
	.td-invite-tab-panel {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}

	/* ── Section wrapper ─────────────────────────────────────────── */
	.td-invite-section {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.td-invite-label {
		font-size: 11px;
		font-weight: 700;
		color: var(--dt2, #555);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	/* ── Email input ─────────────────────────────────────────────── */
	.td-invite-input-wrap {
		position: relative;
		flex: 1;
		min-width: 0;
	}

	.td-invite-input-wrap__icon {
		position: absolute;
		left: 11px;
		top: 50%;
		transform: translateY(-50%);
		color: var(--dt4, #bbb);
		display: flex;
		align-items: center;
		pointer-events: none;
	}

	.td-invite-input {
		width: 100%;
		padding: 9px 12px 9px 32px;
		font-size: 13px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 10px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
		outline: none;
		transition: border-color 0.15s, box-shadow 0.15s, background 0.12s;
		box-sizing: border-box;
	}
	.td-invite-input::placeholder {
		color: var(--dt4, #bbb);
	}
	.td-invite-input:focus {
		border-color: var(--dt2, #555);
		background: var(--dbg, #fff);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--dt, #111) 6%, transparent);
	}
	.td-invite-input:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	/* ── Role select ─────────────────────────────────────────────── */
	.td-invite-select {
		width: 100%;
		padding: 9px 12px;
		font-size: 13px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 10px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
		outline: none;
		transition: border-color 0.15s, box-shadow 0.15s;
		box-sizing: border-box;
		cursor: pointer;
	}
	.td-invite-select:focus {
		border-color: var(--dt2, #555);
		background: var(--dbg, #fff);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--dt, #111) 6%, transparent);
	}
	.td-invite-select:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	/* ── Hint text ───────────────────────────────────────────────── */
	.td-invite-hint {
		font-size: 11px;
		color: var(--dt3, #888);
		margin: 0;
		line-height: 1.4;
	}

	/* ── Error banner ────────────────────────────────────────────── */
	.td-invite-error-banner {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 9px 12px;
		background: color-mix(in srgb, #ef4444 10%, var(--dbg, #fff));
		color: #ef4444;
		font-size: 12px;
		font-weight: 500;
		border-radius: 10px;
		border: 1px solid color-mix(in srgb, #ef4444 20%, var(--dbg, #fff));
	}

	/* ── Success state ───────────────────────────────────────────── */
	.td-invite-success {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 10px;
		padding: 28px 16px;
		background: color-mix(in srgb, #22c55e 10%, var(--dbg, #fff));
		color: #22c55e;
		font-size: 13px;
		font-weight: 500;
		border-radius: 12px;
		text-align: center;
	}
	.td-invite-success strong {
		color: #22c55e;
	}

	/* ── Footer ──────────────────────────────────────────────────── */
	.td-invite-footer {
		display: flex;
		justify-content: flex-end;
	}

	.td-invite-send-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		white-space: nowrap;
		flex-shrink: 0;
	}

	/* ── Empty state ─────────────────────────────────────────────── */
	.td-invite-empty {
		font-size: 13px;
		color: var(--dt3, #888);
		text-align: center;
		padding: 20px 0;
		margin: 0;
	}

	/* ── Spinner animation ───────────────────────────────────────── */
	:global(.td-invite-spinner) {
		animation: td-spin 0.8s linear infinite;
	}
	@keyframes td-spin {
		from { transform: rotate(0deg); }
		to   { transform: rotate(360deg); }
	}

	/* ── Member list ─────────────────────────────────────────────── */
	.td-invite-member-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
		max-height: 260px;
		overflow-y: auto;
	}

	.td-invite-member-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 10px;
		border-radius: 10px;
		transition: background 0.12s;
	}
	.td-invite-member-row:hover {
		background: var(--dbg2, #f5f5f5);
	}

	/* ── Avatar ──────────────────────────────────────────────────── */
	.td-invite-member-avatar {
		width: 34px;
		height: 34px;
		flex-shrink: 0;
	}

	.td-invite-member-avatar__img {
		width: 34px;
		height: 34px;
		border-radius: 9999px;
		object-fit: cover;
	}

	.td-invite-member-avatar__initials {
		width: 34px;
		height: 34px;
		border-radius: 9999px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 12px;
		font-weight: 800;
		color: var(--bos-surface-on-color);
		letter-spacing: -0.02em;
	}

	/* ── Member info ─────────────────────────────────────────────── */
	.td-invite-member-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.td-invite-member-info__name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt, #111);
		letter-spacing: -0.01em;
		line-height: 1.2;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.td-invite-member-info__email {
		font-size: 11px;
		color: var(--dt3, #888);
		font-weight: 400;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* ── Role dropdown pill ──────────────────────────────────────── */
	:global(.td-invite-role-pill) {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 26px;
		padding: 0 10px 0 11px;
		border-radius: 9999px;
		border: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg2, #f5f5f5);
		font-size: 11px;
		font-weight: 600;
		color: var(--dt2, #555);
		cursor: pointer;
		white-space: nowrap;
		flex-shrink: 0;
		transition: background 0.12s, border-color 0.12s;
	}
	:global(.td-invite-role-pill:hover) {
		background: var(--dbg3, #eee);
		border-color: var(--dbd2, #f0f0f0);
	}

	/* ── Role dropdown menu ──────────────────────────────────────── */
	:global(.td-invite-dropdown) {
		z-index: 60;
		min-width: 160px;
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 12px;
		box-shadow: 0 8px 28px rgba(0, 0, 0, 0.13);
		padding: 4px;
		outline: none;
	}

	:global(.td-invite-dropdown__item) {
		padding: 8px 12px;
		font-size: 12px;
		font-weight: 500;
		color: var(--dt, #111);
		border-radius: 8px;
		cursor: pointer;
		transition: background 0.1s;
		outline: none;
	}
	:global(.td-invite-dropdown__item:hover),
	:global(.td-invite-dropdown__item:focus) {
		background: var(--dbg2, #f5f5f5);
	}
	:global(.td-invite-dropdown__item--active) {
		color: var(--dt, #111);
		font-weight: 700;
		background: color-mix(in srgb, var(--dt, #111) 6%, transparent);
	}
</style>
