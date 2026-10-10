<!--
  FloatingChatScreen.svelte
  Layout container for conversational onboarding
  Features: PurpleOrb header, scrollable message area
-->
<script lang="ts">
	import { type Snippet } from 'svelte';
	import PurpleOrb from './PurpleOrb.svelte';

	interface Props {
		title?: string;
		header?: Snippet;
		children?: Snippet;
		footer?: Snippet;
		class?: string;
	}

	let {
		title = '',
		header,
		children,
		footer,
		class: className = ''
	}: Props = $props();
</script>

<div class="floating-chat-screen {className}">
	<!-- Header -->
	<header class="header">
		<div class="header-left">
			{#if title}
				<h1 class="title">{title}</h1>
			{/if}
		</div>

		<div class="header-center">
			{#if header}
				{@render header()}
			{:else}
				<PurpleOrb size="sm" />
			{/if}
		</div>

		<div class="header-right">
			<!-- Empty for balance -->
		</div>
	</header>

	<!-- Main Content -->
	<main class="content">
		{#if children}
			{@render children()}
		{/if}
	</main>

	<!-- Footer -->
	{#if footer}
		<footer class="footer">
			{@render footer()}
		</footer>
	{/if}
</div>

<style>
	.floating-chat-screen {
		display: flex;
		flex-direction: column;
		min-height: 100vh;
		background-color: var(--background, #ffffff);
		color: var(--foreground, #1f2937);
	}

	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 24px;
		border-bottom: 1px solid var(--border, #e5e7eb);
		position: sticky;
		top: 0;
		background-color: var(--background, #ffffff);
		z-index: 10;
	}

	.header-left,
	.header-right {
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 120px;
	}

	.header-right {
		justify-content: flex-end;
	}

	.header-center {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.title {
		font-size: 18px;
		font-weight: 600;
		margin: 0;
	}

	.content {
		flex: 1;
		display: flex;
		flex-direction: column;
		padding: 24px;
		overflow-y: auto;
	}

	.footer {
		padding: 16px 24px;
		border-top: 1px solid var(--border, #e5e7eb);
		background-color: var(--background, #ffffff);
		position: sticky;
		bottom: 0;
	}

	/* Dark mode */
	:global(.dark) .floating-chat-screen {
		background-color: var(--background, #0a0a0a);
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .header {
		background-color: var(--background, #0a0a0a);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .footer {
		background-color: var(--background, #0a0a0a);
		border-color: var(--border, #2a2a2a);
	}
</style>
