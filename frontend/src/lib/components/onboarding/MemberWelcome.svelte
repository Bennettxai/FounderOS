<!--
  MemberWelcome.svelte
  Simplified welcome screen for invited workspace members
-->
<script lang="ts">
	import { Button } from '$lib/ui';
	import PurpleOrb from './PurpleOrb.svelte';

	interface Props {
		workspaceName?: string;
		inviterName?: string;
		userName?: string;
		onGetStarted?: () => void;
		class?: string;
	}

	let {
		workspaceName = 'Your Workspace',
		inviterName = 'your team',
		userName = '',
		onGetStarted,
		class: className = ''
	}: Props = $props();
</script>

<div class="member-welcome {className}">
	<div class="orb-container">
		<PurpleOrb size="lg" />
	</div>

	<div class="content">
		<h1 class="title">
			{#if userName}
				Welcome, {userName}!
			{:else}
				Welcome to BusinessOS!
			{/if}
		</h1>

		<p class="subtitle">
			{inviterName} has invited you to join
			<strong>{workspaceName}</strong>.
		</p>

		<div class="features">
			<div class="feature">
				<div class="feature-icon">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2" />
						<rect width="6" height="4" x="9" y="3" rx="1" />
						<path d="m9 12 2 2 4-4" />
					</svg>
				</div>
				<span>Collaborate on projects</span>
			</div>
			<div class="feature">
				<div class="feature-icon">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
					</svg>
				</div>
				<span>Communicate with your team</span>
			</div>
			<div class="feature">
				<div class="feature-icon">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M12 8V4H8" />
						<rect width="16" height="12" x="4" y="8" rx="2" />
						<path d="M2 14h2" />
						<path d="M20 14h2" />
						<path d="M15 13v2" />
						<path d="M9 13v2" />
					</svg>
				</div>
				<span>Use AI-powered tools</span>
			</div>
		</div>

		<Button variant="primary" size="large" onclick={onGetStarted}>
			Get Started
		</Button>
	</div>
</div>

<style>
	.member-welcome {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		padding: 48px 24px;
		max-width: 480px;
		margin: 0 auto;
		animation: fade-in 0.5s ease-out;
	}

	.orb-container {
		margin-bottom: 32px;
	}

	.content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}

	.title {
		font-size: 28px;
		font-weight: 600;
		color: var(--foreground, #1f2937);
		margin: 0;
	}

	.subtitle {
		font-size: 16px;
		color: var(--muted-foreground, #6b7280);
		margin: 0;
		line-height: 1.5;
	}

	.subtitle strong {
		color: var(--foreground, #1f2937);
	}

	.features {
		display: flex;
		flex-direction: column;
		gap: 12px;
		width: 100%;
		margin: 24px 0;
		text-align: left;
	}

	.feature {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		background-color: var(--secondary, #f9fafb);
		border-radius: 8px;
	}

	.feature-icon {
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		background-color: var(--background, #ffffff);
		color: var(--foreground, #1f2937);
		flex-shrink: 0;
	}

	.feature span {
		font-size: 14px;
		color: var(--foreground, #1f2937);
	}

	@keyframes fade-in {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* Dark mode */
	:global(.dark) .title {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .subtitle strong {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .feature {
		background-color: var(--secondary, #1a1a1a);
	}

	:global(.dark) .feature-icon {
		background-color: var(--background, #0a0a0a);
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .feature span {
		color: var(--foreground, #f9fafb);
	}
</style>
