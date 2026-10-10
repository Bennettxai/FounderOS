<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import StatusBadge from './StatusBadge.svelte';
	import CapacityBar from './CapacityBar.svelte';

	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface Project {
		id: string;
		name: string;
		taskCount: number;
		overdueCount: number;
	}

	interface Activity {
		id: string;
		description: string;
		createdAt: string;
	}

	interface TeamMember {
		id: string;
		name: string;
		role: string;
		email: string;
		avatar?: string;
		status: Status;
		capacity: number;
		joinedAt: string;
		skills: string[];
		projects: Project[];
		activity: Activity[];
		activeProjects?: number;
	}

	interface Props {
		open?: boolean;
		member?: TeamMember | null;
		onClose?: () => void;
		onEdit?: () => void;
		onDelete?: () => void;
		onAssignTask?: () => void;
	}

	let {
		open = $bindable(false),
		member = null,
		onClose,
		onEdit,
		onDelete,
		onAssignTask
	}: Props = $props();

	let confirmingDelete = $state(false);

	function handleClose() {
		open = false;
		confirmingDelete = false;
		onClose?.();
	}

	function getInitials(name: string) {
		return name
			.split(' ')
			.map(n => n.charAt(0))
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function formatRelativeTime(dateStr: string) {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return 'Just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString();
	}

	function formatJoinDate(dateStr: string) {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' });
	}
</script>

{#if open && member}
	<!-- Overlay -->
	<div
		class="td-slideover-backdrop"
		transition:fade={{ duration: 200 }}
		onclick={handleClose}
		role="button"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && handleClose()}
	></div>

	<!-- Slide-over Panel -->
	<div
		class="td-slideover"
		transition:fly={{ x: 400, duration: 300 }}
	>
		<!-- Header -->
		<div class="td-slideover__header">
			<span class="td-slideover__title">Team Member</span>
			<button
				onclick={handleClose}
				class="btn-pill btn-pill-ghost w-8 h-8 flex items-center justify-center"
				aria-label="Close panel"
			>
				<svg class="w-5 h-5" style="color: var(--dt3)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<!-- Identity -->
		<div class="td-slideover__identity">
			<div class="td-slideover__avatar-wrap">
				{#if member.avatar}
					<img src={member.avatar} alt={member.name} class="td-avatar td-avatar--xl" style="object-fit: cover" />
				{:else}
					<div class="td-avatar td-avatar--xl" style="background: var(--bos-avatar-default)">{getInitials(member.name)}</div>
				{/if}
				<span class="td-status-dot td-status-dot--lg td-status-dot--{member.status}"></span>
			</div>
			<div class="td-slideover__identity-info">
				<span class="td-slideover__name">{member.name}</span>
				<span class="td-slideover__role">{member.role}</span>
				<span class="td-slideover__contact-item">{member.email}</span>
			</div>
		</div>

		<!-- Content -->
		<div class="td-slideover__section">
			<span class="td-slideover__section-label">Status & Joined</span>
			<div style="display: flex; gap: 12px; align-items: center;">
				<StatusBadge status={member.status} />
				<span style="font-size: 12px; color: var(--dt3, #888);">Since {formatJoinDate(member.joinedAt)}</span>
			</div>
		</div>

		<div class="td-slideover__section">
			<span class="td-slideover__section-label">Workload</span>
			<CapacityBar capacity={member.capacity} size="lg" />
		</div>

		<div class="td-slideover__section">
			<span class="td-slideover__section-label">Active Projects ({member.projects.length || member.activeProjects || 0})</span>
			{#if member.projects.length > 0}
				<div class="td-slideover__contact-list">
					{#each member.projects as project}
						<div class="td-slideover__contact-item">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
							</svg>
							<span>{project.name}</span>
							<span style="margin-left: auto; font-size: 10px; color: var(--dt4, #bbb);">
								{project.taskCount} tasks
								{#if project.overdueCount > 0}
									· {project.overdueCount} overdue
								{/if}
							</span>
						</div>
					{/each}
				</div>
			{:else if member.activeProjects}
				<span style="font-size: 12px; color: var(--dt2, #555);">{member.activeProjects} active project{member.activeProjects === 1 ? '' : 's'}</span>
			{:else}
				<span style="font-size: 12px; color: var(--dt4, #bbb);">No active projects</span>
			{/if}
		</div>

		<div class="td-slideover__section">
			<span class="td-slideover__section-label">Skills ({member.skills.length})</span>
			{#if member.skills.length > 0}
				<div class="td-slideover__skills-grid">
					{#each member.skills as skill}
						<div class="td-skill-item">
							<span class="td-skill-item__dot"></span>
							<span class="td-skill-item__text">{skill}</span>
						</div>
					{/each}
				</div>
			{:else}
				<div class="td-skills-empty">
					<svg class="td-skills-empty__icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
					</svg>
					<span class="td-skills-empty__text">No skills listed</span>
				</div>
			{/if}
		</div>

		<div class="td-slideover__section td-slideover__section--grow">
			<span class="td-slideover__section-label">Recent Activity</span>
			{#if member.activity.length > 0}
				<div class="td-slideover__activity">
					{#each member.activity.slice(0, 5) as item}
						<div class="td-slideover__activity-item">
							<div class="td-slideover__activity-dot"></div>
							<div class="td-slideover__activity-body">
								<span class="td-slideover__activity-text">{item.description}</span>
								<span class="td-slideover__activity-time">{formatRelativeTime(item.createdAt)}</span>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<span style="font-size: 12px; color: var(--dt4, #bbb);">No recent activity</span>
			{/if}
		</div>

		<!-- Footer Actions -->
		<div class="td-slideover__footer">
			<div class="td-slideover__footer-row">
				<button
					onclick={onEdit}
					class="btn-pill btn-pill-soft btn-pill-sm flex-1"
				>
					Edit Member
				</button>
				<button
					onclick={onAssignTask}
					class="btn-pill btn-pill-primary btn-pill-sm flex-1"
				>
					Assign Task
				</button>
			</div>
			{#if confirmingDelete}
				<div class="td-slideover__delete-confirm">
					<span>Remove this member?</span>
					<button
						onclick={() => confirmingDelete = false}
						class="btn-pill btn-pill-ghost btn-pill-sm"
						style="font-size: 11px;"
					>
						Cancel
					</button>
					<button
						onclick={() => { confirmingDelete = false; onDelete?.(); }}
						class="btn-pill btn-pill-sm td-delete-confirm-btn"
						style="font-size: 11px;"
					>
						Confirm
					</button>
				</div>
			{:else}
				<button
					onclick={() => confirmingDelete = true}
					class="btn-pill btn-pill-ghost btn-pill-sm td-remove-btn"
					style="width: 100%; font-size: 12px;"
				>
					Remove Member
				</button>
			{/if}
		</div>
	</div>
{/if}

<style>
	.td-slideover-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0,0,0,0.3);
		z-index: 40;
		cursor: pointer;
	}
	:global(.dark) .td-slideover-backdrop { background: rgba(0,0,0,0.5); }

	.td-slideover {
		position: fixed;
		top: 0;
		right: 0;
		height: 100%;
		width: 340px;
		max-width: 95vw;
		z-index: 50;
		background: var(--dbg, #fff);
		border-left: 1px solid var(--dbd2, #f0f0f0);
		box-shadow: -8px 0 40px rgba(0,0,0,0.12);
		display: flex;
		flex-direction: column;
		overflow-y: auto;
	}
	:global(.dark) .td-slideover { background: var(--dbg); }

	.td-slideover__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px;
		border-bottom: 1px solid var(--dbd, #e0e0e0);
		position: sticky;
		top: 0;
		background: var(--dbg, #fff);
		z-index: 1;
	}
	:global(.dark) .td-slideover__header { background: var(--dbg); }

	.td-slideover__title {
		font-size: 13px;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.01em;
	}

	.td-slideover__identity {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 20px 16px 16px;
		border-bottom: 1px solid var(--dbd, #e0e0e0);
	}

	.td-slideover__avatar-wrap {
		position: relative;
		flex-shrink: 0;
	}
	.td-slideover__avatar-wrap .td-status-dot {
		position: absolute;
		bottom: 2px;
		right: 2px;
	}

	.td-slideover__identity-info {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}

	.td-slideover__name {
		font-size: 16px;
		font-weight: 800;
		color: var(--dt, #111);
		letter-spacing: -0.03em;
		line-height: 1.1;
	}

	.td-slideover__role {
		font-size: 12px;
		color: var(--dt3, #888);
		font-weight: 500;
	}

	.td-slideover__section {
		padding: 14px 16px;
		border-bottom: 1px solid var(--dbd, #e0e0e0);
	}
	.td-slideover__section--grow { flex: 1; border-bottom: none; }

	.td-slideover__section-label {
		font-size: 10px;
		font-weight: 800;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--dt3, #888);
		display: block;
		margin-bottom: 10px;
	}

	.td-slideover__contact-list {
		display: flex;
		flex-direction: column;
		gap: 7px;
	}

	.td-slideover__contact-item {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 12px;
		color: var(--dt2, #555);
	}
	.td-slideover__contact-item :global(svg) { color: var(--dt3, #888); flex-shrink: 0; }

	.td-slideover__skills-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 6px;
	}

	.td-skill-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		border-radius: 8px;
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		border-left: 2px solid var(--bos-border-color, #e3e2e4);
		transition: background 0.15s;
	}

	.td-skill-item:hover {
		background: var(--bos-hover-color, rgba(0, 0, 0, 0.04));
	}

	.td-skill-item__dot {
		width: 4px;
		height: 4px;
		border-radius: 9999px;
		background: var(--dt3, #888);
		flex-shrink: 0;
	}

	.td-skill-item__text {
		font-size: 12px;
		font-weight: 500;
		color: var(--dt, #111);
		line-height: 1.2;
	}

	.td-skills-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 16px 0;
	}

	.td-skills-empty__icon {
		width: 24px;
		height: 24px;
		color: var(--bos-v2-icon-tertiary, #c5c5c5);
	}

	.td-skills-empty__text {
		font-size: 12px;
		color: var(--bos-v2-text-tertiary, #a3a3a3);
	}

	.td-slideover__activity {
		display: flex;
		flex-direction: column;
	}

	.td-slideover__activity-item {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 8px 0;
		border-bottom: 1px solid var(--dbd, #e0e0e0);
	}
	.td-slideover__activity-item:last-child { border-bottom: none; }

	.td-slideover__activity-dot {
		width: 7px;
		height: 7px;
		border-radius: 9999px;
		background: var(--dbd2, #f0f0f0);
		flex-shrink: 0;
		margin-top: 4px;
	}

	.td-slideover__activity-body {
		display: flex;
		flex-direction: column;
		gap: 2px;
		flex: 1;
	}

	.td-slideover__activity-text {
		font-size: 12px;
		color: var(--dt, #111);
		font-weight: 500;
		line-height: 1.4;
	}

	.td-slideover__activity-time {
		font-size: 10px;
		color: var(--dt4, #bbb);
		font-weight: 500;
	}

	.td-slideover__footer {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 14px 16px;
		border-top: 1px solid var(--dbd, #e0e0e0);
	}

	.td-slideover__footer-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.td-slideover__delete-confirm {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		border-radius: 10px;
		background: var(--bos-background-error-color, #fef2f2);
		font-size: 12px;
		color: var(--bos-error-color);
	}
	.td-slideover__delete-confirm span:first-child { flex: 1; }
	.td-delete-confirm-btn { background: var(--bos-error-color); color: var(--bos-surface-on-color); }
	.td-remove-btn { color: var(--bos-error-color); }

	.td-avatar {
		border-radius: 9999px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		color: var(--bos-surface-on-color);
		flex-shrink: 0;
		letter-spacing: -0.02em;
	}
	.td-avatar--xl { width: 52px; height: 52px; font-size: 18px; }

	.td-status-dot {
		width: 9px;
		height: 9px;
		border-radius: 9999px;
		border: 2px solid var(--dbg, #fff);
		display: block;
		flex-shrink: 0;
	}
	.td-status-dot--lg { width: 12px; height: 12px; }
	.td-status-dot--available { background: #22c55e; }
	.td-status-dot--busy { background: #f59e0b; }
	.td-status-dot--overloaded { background: #ef4444; }
	.td-status-dot--ooo { background: #9ca3af; }
	:global(.dark) .td-status-dot { border-color: var(--dbg); }
</style>
